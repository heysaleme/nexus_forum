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
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type AuthService interface {
	Register(email, password string) error
	VerifyOTP(email, otpCode string) (accessToken, refreshToken string, user *model.User, err error)
	Login(email, password string) (accessToken, refreshToken string, user *model.User, err error)
	RefreshAccessToken(refreshToken string) (accessToken, newRefreshToken string, err error)
	Logout(refreshToken string) error
	ValidateToken(tokenStr string) (*Claims, error)
	ChangePassword(userID uint, oldPassword, newPassword string) error
	RequestPasswordReset(email string) (string, error)
	ResetPassword(resetToken, newPassword string) error
	FindOrCreateOAuthUser(provider, sub, email, name, avatarURL string) (*model.User, string, string, error)
}

// In-memory pending registration store for OTP code verification (simulating Redis/DB)
var pendingRegistrations = make(map[string]string) // email -> hashed_password

type authService struct {
	repo       repository.UserRepository
	modRepo    repository.ModerationRepository
	resetRepo   repository.PasswordResetRepository
	refreshRepo repository.RefreshTokenRepository
	jwtSecret   string
}

func NewAuthService(repo repository.UserRepository, modRepo repository.ModerationRepository, resetRepo repository.PasswordResetRepository, refreshRepo repository.RefreshTokenRepository, jwtSecret string) AuthService {
	return &authService{repo: repo, modRepo: modRepo, resetRepo: resetRepo, refreshRepo: refreshRepo, jwtSecret: jwtSecret}
}

func (s *authService) Register(email, password string) error {
	_, err := s.repo.GetByEmail(email)
	if err == nil {
		return errors.New("email already registered")
	}
	if _, exists := pendingRegistrations[email]; exists {
		return errors.New("registration already pending for this email")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	pendingRegistrations[email] = string(hashed)
	return nil
}

func (s *authService) VerifyOTP(email, otpCode string) (string, string, *model.User, error) {
	if otpCode != "123456" {
		return "", "", nil, errors.New("invalid OTP code, use demo code 123456")
	}

	hashedPassword, ok := pendingRegistrations[email]
	if !ok {
		return "", "", nil, errors.New("no pending registration for this email")
	}

	// Create User
	parts := strings.Split(email, "@")
	username := parts[0]
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         "user",
		ProfileTheme: "default",
		Level:        1,
		XP:           0,
		AllowDMs:     true,
	}

	err := s.repo.Create(user)
	if err != nil {
		return "", "", nil, err
	}

	delete(pendingRegistrations, email)

	access, refresh, err := s.issueTokenPair(user)
	return access, refresh, user, err
}

func (s *authService) Login(email, password string) (string, string, *model.User, error) {
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

	access, refresh, err := s.issueTokenPair(user)
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

	if err := s.refreshRepo.Revoke(row.ID); err != nil {
		return "", "", err
	}

	return s.issueTokenPair(user)
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
		return claims, nil
	}
	return nil, errors.New("invalid token claims")
}

func (s *authService) issueTokenPair(user *model.User) (string, string, error) {
	access, err := s.generateJWT(user)
	if err != nil {
		return "", "", err
	}
	refresh, err := s.createRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *authService) createRefreshToken(userID uint) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	row := &model.RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.refreshRepo.Create(row); err != nil {
		return "", err
	}
	return token, nil
}

func (s *authService) generateJWT(user *model.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
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
