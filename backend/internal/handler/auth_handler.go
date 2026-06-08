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

	accessToken, refreshToken, user, err := h.AuthService.VerifyOTP(req.Email, req.OTPCode)
	if err != nil {
		slog.Error("otp verification failed", "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user registration completed via otp", "email", req.Email, "user_id", user.ID)
	uid := user.ID
	_ = h.Analytics.Track(&uid, "register", "user", &uid, "")
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

func (h *Handlers) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, user, err := h.AuthService.Login(req.Email, req.Password)
	if err != nil {
		slog.Warn("login attempt failed", "email", req.Email, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("user logged in successfully", "email", req.Email, "user_id", user.ID, "role", user.Role)
	uid := user.ID
	_ = h.Analytics.Track(&uid, "login", "user", &uid, "")
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
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

	user.IsOnline = h.WSHub.IsUserOnline(user.ID)
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
	if isBase64DataURL(req.AvatarURL) || isBase64DataURL(req.BannerURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload avatars/banners via /api/upload; base64 data URLs are not allowed"})
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

func (h *Handlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.AuthService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *Handlers) ForgotPassword(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.AuthService.RequestPasswordReset(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{"success": true, "message": "If the email exists, a reset link has been sent"}
	if token != "" {
		resp["reset_token"] = token
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handlers) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetSubmit
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.AuthService.ResetPassword(req.ResetToken, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
