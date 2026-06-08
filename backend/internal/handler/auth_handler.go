package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/model"
)

// ================= Auth Handlers =================

func (h *Handlers) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.AuthService.Register(req.Email, req.Password)
	if err != nil {
		slog.Error("registration failed", "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user registration initiated", "email", req.Email)
	c.JSON(http.StatusOK, gin.H{"success": true, "otp_required": true})
}

func (h *Handlers) VerifyOTP(c *gin.Context) {
	var req dto.OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.AuthService.VerifyOTP(req.Email, req.OTPCode)
	if err != nil {
		slog.Error("otp verification failed", "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user registration completed via otp", "email", req.Email, "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"user":         user,
	})
}

func (h *Handlers) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.AuthService.Login(req.Email, req.Password)
	if err != nil {
		slog.Warn("login attempt failed", "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user logged in successfully", "email", req.Email, "user_id", user.ID, "role", user.Role)
	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"user":         user,
	})
}

func (h *Handlers) GetMe(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	user, err := h.UserService.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handlers) UpdateMe(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userReq := model.User{
		Username:     req.Username,
		Bio:          req.Bio,
		Title:        req.Title,
		AvatarURL:    req.AvatarURL,
		BannerURL:    req.BannerURL,
		ProfileTheme: req.ProfileTheme,
	}
	if req.AllowDMs != nil {
		userReq.AllowDMs = *req.AllowDMs
	}
	if req.IsPrivate != nil {
		userReq.IsPrivate = *req.IsPrivate
	}

	user, err := h.UserService.UpdateProfile(uid, userReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handlers) ChangePassword(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.AuthService.ChangePassword(uid, req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
