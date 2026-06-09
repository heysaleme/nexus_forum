package service

import (
	"github.com/golang-jwt/jwt/v5"
	"nexus-forum-backend/internal/model"
)

type Claims struct {
	UserID    uint   `json:"user_id"`
	SessionID uint   `json:"sid"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func stringsSplitEmail(email string) string {
	parts := stringsSplit(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

func stringsSplit(s, sep string) []string {
	var parts []string
	for {
		i := 0
		for i < len(s) && string(s[i]) != sep {
			i++
		}
		if i == len(s) {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:i])
		s = s[i+len(sep):]
	}
	return parts
}

func recalculateLevel(user *model.User) {
	user.Level = (user.XP / 100) + 1
	if user.Level < 1 {
		user.Level = 1
	}
}

func maxZero(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
