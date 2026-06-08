package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/service"
)

// oauthStateStore holds short-lived nonces that prevent CSRF on the OAuth callback.
// Each state expires after 10 minutes.
var (
	oauthStateMu    sync.Mutex
	oauthStateStore = make(map[string]time.Time)
)

// generateState creates a cryptographically random state nonce and registers it.
func generateState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)
	oauthStateMu.Lock()
	oauthStateStore[state] = time.Now().Add(10 * time.Minute)
	oauthStateMu.Unlock()
	return state, nil
}

// consumeState validates and removes a state nonce. Returns false if invalid/expired.
func consumeState(state string) bool {
	oauthStateMu.Lock()
	defer oauthStateMu.Unlock()
	exp, ok := oauthStateStore[state]
	if !ok {
		return false
	}
	delete(oauthStateStore, state)
	return time.Now().Before(exp)
}

// init starts a background goroutine that prunes expired states every 5 minutes.
func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			oauthStateMu.Lock()
			now := time.Now()
			for k, exp := range oauthStateStore {
				if now.After(exp) {
					delete(oauthStateStore, k)
				}
			}
			oauthStateMu.Unlock()
		}
	}()
}

// OAuthConfig carries runtime OAuth configuration values.
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GithubClientID     string
	GithubClientSecret string
	FrontendURL        string
}

// GetOAuthProviderConfig returns which OAuth providers are available.
// Route: GET /api/auth/oauth/config
// Public — no auth required.
func GetOAuthProviderConfig(cfg OAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"google_enabled": cfg.GoogleClientID != "",
			"github_enabled": cfg.GithubClientID != "",
			"apple_enabled":  false,
		})
	}
}

// GoogleOAuthInitiate redirects the user to Google's OAuth 2.0 consent screen.
// Route: GET /api/auth/oauth/google
func GoogleOAuthInitiate(cfg OAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.GoogleClientID == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth is not configured"})
			return
		}

		state, err := generateState()
		if err != nil {
			slog.Error("failed to generate OAuth state", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		redirectURI := cfg.FrontendURL + "/auth/callback/google"
		params := url.Values{
			"client_id":     {cfg.GoogleClientID},
			"redirect_uri":  {redirectURI},
			"response_type": {"code"},
			"scope":         {"openid email profile"},
			"state":         {state},
			"access_type":   {"online"},
		}
		authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
		c.JSON(http.StatusOK, gin.H{"url": authURL, "state": state})
	}
}

// googleTokenResponse is the JSON payload from Google's token endpoint.
type googleTokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// googleUserInfo holds the fields we extract from Google's ID token payload.
type googleUserInfo struct {
	Sub       string `json:"sub"`   // stable Google user ID
	Email     string `json:"email"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
}

// GoogleOAuthCallback exchanges the authorization code for user info and issues a Nexus JWT.
// Route: POST /api/auth/oauth/google/callback
// Body: { "code": string, "state": string }
func GoogleOAuthCallback(cfg OAuthConfig, authSvc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.GoogleClientID == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth is not configured"})
			return
		}

		var body struct {
			Code  string `json:"code" binding:"required"`
			State string `json:"state" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code and state are required"})
			return
		}

		// CSRF check — verify state nonce
		if !consumeState(body.State) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired state"})
			return
		}

		// Exchange code → tokens
		redirectURI := cfg.FrontendURL + "/auth/callback/google"
		tokenResp, err := exchangeGoogleCode(body.Code, redirectURI, cfg)
		if err != nil {
			slog.Error("google token exchange failed", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to exchange code with Google"})
			return
		}

		// Decode ID token payload (JWT — we trust Google signed it, no need to verify sig here
		// since we just obtained it directly from Google's token endpoint over HTTPS)
		userInfo, err := decodeGoogleIDToken(tokenResp.IDToken)
		if err != nil {
			slog.Error("failed to decode google id_token", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user info"})
			return
		}

		user, accessToken, refreshToken, err := authSvc.FindOrCreateOAuthUser("google", userInfo.Sub, userInfo.Email, userInfo.Name, userInfo.Picture)
		if err != nil {
			slog.Error("FindOrCreateOAuthUser failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate user"})
			return
		}

		slog.Info("google oauth login", "user_id", user.ID, "email", user.Email)
		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user":          user,
		})
	}
}

// exchangeGoogleCode calls Google's token endpoint and returns the token response.
func exchangeGoogleCode(code, redirectURI string, cfg OAuthConfig) (*googleTokenResponse, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {cfg.GoogleClientID},
		"client_secret": {cfg.GoogleClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", form)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var tokenResp googleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("google error: %s", tokenResp.Error)
	}
	return &tokenResp, nil
}

// decodeGoogleIDToken extracts the payload from a Google ID token (JWT) without signature verification.
// This is safe because we received the token directly from Google's token endpoint over HTTPS.
func decodeGoogleIDToken(idToken string) (*googleUserInfo, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// JWT uses base64url encoding, may need padding
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	var info googleUserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	if info.Sub == "" {
		return nil, errors.New("missing 'sub' in id_token")
	}
	return &info, nil
}

type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

type githubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubOAuthInitiate returns the GitHub authorization URL.
// Route: GET /api/auth/oauth/github
func GitHubOAuthInitiate(cfg OAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.GithubClientID == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub OAuth is not configured"})
			return
		}

		state, err := generateState()
		if err != nil {
			slog.Error("failed to generate OAuth state", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		redirectURI := cfg.FrontendURL + "/auth/callback/github"
		params := url.Values{
			"client_id":    {cfg.GithubClientID},
			"redirect_uri": {redirectURI},
			"scope":        {"read:user user:email"},
			"state":        {state},
		}
		authURL := "https://github.com/login/oauth/authorize?" + params.Encode()
		c.JSON(http.StatusOK, gin.H{"url": authURL, "state": state})
	}
}

// GitHubOAuthCallback exchanges the GitHub code and issues Nexus tokens.
// Route: POST /api/auth/oauth/github/callback
func GitHubOAuthCallback(cfg OAuthConfig, authSvc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.GithubClientID == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub OAuth is not configured"})
			return
		}

		var body struct {
			Code  string `json:"code" binding:"required"`
			State string `json:"state" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code and state are required"})
			return
		}

		if !consumeState(body.State) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired state"})
			return
		}

		redirectURI := cfg.FrontendURL + "/auth/callback/github"
		accessToken, err := exchangeGitHubCode(body.Code, redirectURI, cfg)
		if err != nil {
			slog.Error("github token exchange failed", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to exchange code with GitHub"})
			return
		}

		userInfo, err := fetchGitHubUser(accessToken)
		if err != nil {
			slog.Error("failed to fetch github user", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user info"})
			return
		}

		email := userInfo.Email
		if email == "" {
			email, _ = fetchGitHubPrimaryEmail(accessToken)
		}
		if email == "" {
			email = fmt.Sprintf("%s@users.noreply.github.com", userInfo.Login)
		}

		displayName := userInfo.Name
		if displayName == "" {
			displayName = userInfo.Login
		}

		subject := fmt.Sprintf("%d", userInfo.ID)
		user, jwtAccess, refreshToken, err := authSvc.FindOrCreateOAuthUser("github", subject, email, displayName, userInfo.AvatarURL)
		if err != nil {
			slog.Error("FindOrCreateOAuthUser failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate user"})
			return
		}

		slog.Info("github oauth login", "user_id", user.ID, "email", user.Email)
		c.JSON(http.StatusOK, gin.H{
			"access_token":  jwtAccess,
			"refresh_token": refreshToken,
			"user":          user,
		})
	}
}

func exchangeGitHubCode(code, redirectURI string, cfg OAuthConfig) (string, error) {
	form := url.Values{
		"client_id":     {cfg.GithubClientID},
		"client_secret": {cfg.GithubClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp githubTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("github error: %s", tokenResp.Error)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("missing access_token")
	}
	return tokenResp.AccessToken, nil
}

func fetchGitHubUser(accessToken string) (*githubUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info githubUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	if info.ID == 0 {
		return nil, errors.New("missing github user id")
	}
	return &info, nil
}

func fetchGitHubPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}
