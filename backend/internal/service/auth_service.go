package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type AuthService interface {
	Register(email, password string) error
	VerifyOTP(email, otpCode string) (string, *model.User, error)
	Login(email, password string) (string, *model.User, error)
	ValidateToken(tokenStr string) (*Claims, error)
}

// In-memory pending registration store for OTP code verification (simulating Redis/DB)
var pendingRegistrations = make(map[string]string) // email -> hashed_password

type authService struct {
	repo      repository.UserRepository
	jwtSecret string
}

func NewAuthService(repo repository.UserRepository, jwtSecret string) AuthService {
	return &authService{repo: repo, jwtSecret: jwtSecret}
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

func (s *authService) VerifyOTP(email, otpCode string) (string, *model.User, error) {
	if otpCode != "123456" {
		return "", nil, errors.New("invalid OTP code, use demo code 123456")
	}

	hashedPassword, ok := pendingRegistrations[email]
	if !ok {
		return "", nil, errors.New("no pending registration for this email")
	}

	// Create User
	username := stringsSplitEmail(email)
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
		return "", nil, err
	}

	delete(pendingRegistrations, email)

	token, err := s.generateJWT(user)
	return token, user, err
}

func (s *authService) Login(email, password string) (string, *model.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	if user.IsBanned {
		return "", nil, errors.New("this account is banned")
	}

	token, err := s.generateJWT(user)
	return token, user, err
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
