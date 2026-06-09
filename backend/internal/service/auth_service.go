package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"nexus-forum-backend/internal/email"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type AuthService interface {
	Register(email, password string) error
	VerifyOTP(email, otpCode string) (accessToken, refreshToken string, user *model.User, err error)
	Login(email, password string) (accessToken, refreshToken string, user *model.User, err error)
	LoginWithContext(email, password, userAgent, ipAddress string) (accessToken, refreshToken string, user *model.User, err error)
	RefreshAccessToken(refreshToken string) (accessToken, newRefreshToken string, err error)
	RefreshAccessTokenWithContext(refreshToken string, userAgent, ipAddress string) (accessToken, newRefreshToken string, err error)
	Logout(refreshToken string) error
	ListSessions(userID uint) ([]*model.RefreshToken, error)
	RevokeSession(userID, sessionID uint) error
	RevokeOtherSessions(userID, keepSessionID uint) error
	ValidateToken(tokenStr string) (*Claims, error)
	ResendVerification(email string) error
	ChangePassword(userID uint, oldPassword, newPassword string) error
	RequestPasswordReset(email string) (string, error)
	ResetPassword(resetToken, newPassword string) error
	FindOrCreateOAuthUser(provider, sub, email, name, avatarURL string) (*model.User, string, string, error)
}

type authService struct {
	repo       repository.UserRepository
	modRepo    repository.ModerationRepository
	resetRepo   repository.PasswordResetRepository
	refreshRepo repository.RefreshTokenRepository
	verifyRepo  repository.EmailVerificationRepository
	mailer      *email.Mailer
	jwtSecret   string
}

func NewAuthService(
	repo repository.UserRepository,
	modRepo repository.ModerationRepository,
	resetRepo repository.PasswordResetRepository,
	refreshRepo repository.RefreshTokenRepository,
	verifyRepo repository.EmailVerificationRepository,
	mailer *email.Mailer,
	jwtSecret string,
) AuthService {
	return &authService{
		repo: repo, modRepo: modRepo, resetRepo: resetRepo, refreshRepo: refreshRepo,
		verifyRepo: verifyRepo, mailer: mailer, jwtSecret: jwtSecret,
	}
}

func (s *authService) Register(email, password string) error {
	if _, err := s.repo.GetByEmail(email); err == nil {
		return errors.New("email already registered")
	}
	if pending, err := s.verifyRepo.GetPendingByEmail(email); err == nil && pending != nil {
		return errors.New("registration already pending for this email")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	code, err := s.verificationCode()
	if err != nil {
		return err
	}

	row := &model.EmailVerification{
		Email:        email,
		Code:         code,
		PasswordHash: string(hashed),
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}
	if err := s.verifyRepo.Upsert(row); err != nil {
		return err
	}

	if s.mailer != nil && s.mailer.Enabled() {
		if err := s.mailer.SendVerification(email, code); err != nil {
			return fmt.Errorf("failed to send verification email: %w", err)
		}
	}
	return nil
}

func (s *authService) ResendVerification(email string) error {
	if _, err := s.repo.GetByEmail(email); err == nil {
		return errors.New("email already registered")
	}

	pending, err := s.verifyRepo.GetPendingByEmail(email)
	if err != nil {
		return errors.New("no pending registration for this email")
	}

	code, err := s.verificationCode()
	if err != nil {
		return err
	}
	pending.Code = code
	pending.ExpiresAt = time.Now().Add(15 * time.Minute)
	if err := s.verifyRepo.Upsert(pending); err != nil {
		return err
	}
	if s.mailer != nil && s.mailer.Enabled() {
		return s.mailer.SendVerification(email, code)
	}
	return nil
}

func (s *authService) verificationCode() (string, error) {
	if s.mailer != nil && s.mailer.Enabled() {
		return generateOTPCode()
	}
	return "123456", nil
}

func (s *authService) VerifyOTP(email, otpCode string) (string, string, *model.User, error) {
	row, err := s.verifyRepo.GetValid(email, otpCode)
	if err != nil {
		return "", "", nil, errors.New("invalid or expired verification code")
	}

	parts := strings.Split(email, "@")
	username := parts[0]
	user := &model.User{
		Username:      username,
		Email:         email,
		PasswordHash:  row.PasswordHash,
		Role:          "user",
		ProfileTheme:  "default",
		Level:         1,
		XP:            0,
		AllowDMs:      true,
		EmailVerified: true,
	}

	if err := s.repo.Create(user); err != nil {
		return "", "", nil, err
	}

	_ = s.verifyRepo.DeleteByEmail(email)

	access, refresh, err := s.issueTokenPair(user)
	return access, refresh, user, err
}

func generateOTPCode() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", n), nil
}

func (s *authService) Login(email, password string) (string, string, *model.User, error) {
	return s.LoginWithContext(email, password, "", "")
}

func (s *authService) LoginWithContext(email, password, userAgent, ipAddress string) (string, string, *model.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    0,
			TargetID:   0,
			TargetType: "user",
			Action:     "login_failed",
			Details:    "Failed login attempt (email not found) for: " + email,
		})
		return "", "", nil, errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    0,
			TargetID:   user.ID,
			TargetType: "user",
			Action:     "login_failed",
			Details:    "Failed login attempt (invalid password) for user ID: " + strconvFormatUint(user.ID),
		})
		return "", "", nil, errors.New("invalid email or password")
	}

	if user.IsBanned {
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    user.ID,
			TargetID:   user.ID,
			TargetType: "user",
			Action:     "login_failed",
			Details:    "Failed login attempt (user is banned) for user ID: " + strconvFormatUint(user.ID),
		})
		return "", "", nil, errors.New("this account is banned")
	}

	access, refresh, err := s.issueTokenPairWithMeta(user, userAgent, ipAddress)
	if err == nil {
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    user.ID,
			TargetID:   user.ID,
			TargetType: "user",
			Action:     "login_success",
			Details:    "User logged in successfully",
		})
	}
	return access, refresh, user, err
}

func (s *authService) Logout(refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.refreshRepo.RevokeByToken(refreshToken)
}

func (s *authService) RefreshAccessToken(refreshToken string) (string, string, error) {
	return s.RefreshAccessTokenWithContext(refreshToken, "", "")
}

func (s *authService) RefreshAccessTokenWithContext(refreshToken, userAgent, ipAddress string) (string, string, error) {
	row, err := s.refreshRepo.GetValidToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid or expired refresh token")
	}

	user, err := s.repo.GetByID(row.UserID)
	if err != nil {
		return "", "", errors.New("user not found")
	}
	if user.IsBanned {
		return "", "", errors.New("this account is banned")
	}

	_ = s.refreshRepo.TouchLastUsed(row.ID)

	if err := s.refreshRepo.Revoke(row.ID); err != nil {
		return "", "", err
	}

	return s.issueTokenPairWithMeta(user, userAgent, ipAddress)
}

func (s *authService) ListSessions(userID uint) ([]*model.RefreshToken, error) {
	return s.refreshRepo.ListActiveByUser(userID)
}

func (s *authService) RevokeSession(userID, sessionID uint) error {
	return s.refreshRepo.RevokeByIDForUser(sessionID, userID)
}

func (s *authService) RevokeOtherSessions(userID, keepSessionID uint) error {
	return s.refreshRepo.RevokeAllForUserExcept(userID, keepSessionID)
}

func (s *authService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.SessionID == 0 {
			return nil, errors.New("session expired: please log in again")
		}
		active, err := s.refreshRepo.IsSessionActive(claims.SessionID)
		if err != nil {
			return nil, errors.New("session validation failed")
		}
		if !active {
			return nil, errors.New("session revoked")
		}
		return claims, nil
	}
	return nil, errors.New("invalid token claims")
}

func (s *authService) issueTokenPair(user *model.User) (string, string, error) {
	return s.issueTokenPairWithMeta(user, "", "")
}

func (s *authService) issueTokenPairWithMeta(user *model.User, userAgent, ipAddress string) (string, string, error) {
	row, refresh, err := s.createRefreshToken(user.ID, userAgent, ipAddress)
	if err != nil {
		return "", "", err
	}
	access, err := s.generateJWT(user, row.ID)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *authService) createRefreshToken(userID uint, userAgent, ipAddress string) (*model.RefreshToken, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}
	token := hex.EncodeToString(tokenBytes)
	now := time.Now()
	row := &model.RefreshToken{
		UserID:     userID,
		Token:      token,
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		LastUsedAt: now,
	}
	if err := s.refreshRepo.Create(row); err != nil {
		return nil, "", err
	}
	return row, token, nil
}

func (s *authService) generateJWT(user *model.User, sessionID uint) (string, error) {
	claims := Claims{
		UserID:    user.ID,
		SessionID: sessionID,
		Email:     user.Email,
		Username:  user.Username,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// FindOrCreateOAuthUser finds an existing user by OAuth identity or creates one.
// OAuth users never need a password — a random secure hash is stored.
func (s *authService) FindOrCreateOAuthUser(provider, sub, email, name, avatarURL string) (*model.User, string, string, error) {
	// 1. Try by (provider, subject) — most precise lookup
	user, err := s.repo.GetByOAuth(provider, sub)
	if err == nil {
		access, refresh, err := s.issueTokenPair(user)
		return user, access, refresh, err
	}

	// 2. Try by email — link existing account to OAuth identity
	if email != "" {
		user, err = s.repo.GetByEmail(email)
		if err == nil {
			user.OAuthProvider = provider
			user.OAuthSubject = sub
			if avatarURL != "" && user.AvatarURL == "" {
				user.AvatarURL = avatarURL
			}
			_ = s.repo.Update(user)
			access, refresh, err := s.issueTokenPair(user)
			return user, access, refresh, err
		}
	}

	// 3. Create new user
	// Derive username from name or email prefix, ensure uniqueness
	username := deriveUsername(name, email)
	if _, err := s.repo.GetByUsername(username); err == nil {
		// Already taken — append part of the sub to make unique
		suffix := sub
		if len(suffix) > 6 {
			suffix = suffix[len(suffix)-6:]
		}
		username = username + "_" + suffix
	}

	// Random password hash — OAuth users never use password login
	randomBytes := fmt.Sprintf("%d", time.Now().UnixNano())
	hashed, _ := bcrypt.GenerateFromPassword([]byte(randomBytes), bcrypt.DefaultCost)

	newUser := &model.User{
		Username:      username,
		Email:         email,
		PasswordHash:  string(hashed),
		AvatarURL:     avatarURL,
		Role:          "user",
		ProfileTheme:  "default",
		Level:         1,
		XP:            0,
		AllowDMs:      true,
		OAuthProvider: provider,
		OAuthSubject:  sub,
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, "", "", err
	}

	access, refresh, err := s.issueTokenPair(newUser)
	return newUser, access, refresh, err
}

// deriveUsername produces a clean username from display name or email.
func deriveUsername(name, email string) string {
	if name != "" {
		// Replace spaces and special chars
		clean := strings.ToLower(name)
		var b strings.Builder
		for _, r := range clean {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
				b.WriteRune(r)
			} else if r == ' ' {
				b.WriteRune('_')
			}
		}
		result := b.String()
		if result != "" && len(result) >= 2 {
			return result
		}
	}
	if email != "" {
		parts := strings.Split(email, "@")
		if len(parts) > 0 && parts[0] != "" {
			return strings.ToLower(parts[0])
		}
	}
	return "user"
}

func (s *authService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
	if err != nil {
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    userID,
			TargetID:   userID,
			TargetType: "user",
			Action:     "password_reset_request",
			Details:    "Failed password change attempt: invalid old password",
		})
		return errors.New("invalid old password")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashed)
	err = s.repo.Update(user)
	if err == nil {
		_ = s.modRepo.CreateLog(&model.ModerationLog{
			ActorID:    userID,
			TargetID:   userID,
			TargetType: "user",
			Action:     "password_reset_completed",
			Details:    "Password changed successfully via Settings",
		})
	}
	return err
}

func (s *authService) RequestPasswordReset(email string) (string, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		// Do not reveal whether the email exists
		return "", nil
	}

	_ = s.resetRepo.DeleteByUserID(user.ID)

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	resetRow := &model.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := s.resetRepo.Create(resetRow); err != nil {
		return "", err
	}

	_ = s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:    user.ID,
		TargetID:   user.ID,
		TargetType: "user",
		Action:     "password_reset_request",
		Details:    "Password reset requested",
	})

	return token, nil
}

func (s *authService) ResetPassword(resetToken, newPassword string) error {
	resetRow, err := s.resetRepo.GetValidToken(resetToken)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	user, err := s.repo.GetByID(resetRow.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashed)
	if err := s.repo.Update(user); err != nil {
		return err
	}

	if err := s.resetRepo.MarkUsed(resetRow.ID); err != nil {
		return err
	}

	_ = s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:    user.ID,
		TargetID:   user.ID,
		TargetType: "user",
		Action:     "password_reset_completed",
		Details:    "Password reset via email token",
	})

	return nil
}

func strconvFormatUint(n uint) string {
	var res []byte
	if n == 0 {
		return "0"
	}
	for n > 0 {
		res = append([]byte{byte('0' + n%10)}, res...)
		n /= 10
	}
	return string(res)
}
