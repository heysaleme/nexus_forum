package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/dto"
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
	resp := gin.H{
		"success":         true,
		"otp_required":    true,
		"smtp_configured": h.SMTPConfigured,
	}
	if pending, err := h.AuthService.GetPendingVerification(req.Email); err == nil && pending != nil {
		if !h.SMTPConfigured || pending.Code == "123456" {
			resp["confirm_token"] = pending.Token
			resp["confirm_url"] = strings.TrimRight(h.FrontendURL, "/") + "/confirm-email?token=" + pending.Token
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handlers) ResendOTP(c *gin.Context) {
	var req dto.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.AuthService.ResendVerification(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "smtp_configured": h.SMTPConfigured})
}

func (h *Handlers) ConfirmEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		var body struct {
			Token string `json:"token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "confirmation token required"})
			return
		}
		token = body.Token
	}

	accessToken, refreshToken, user, err := h.AuthService.ConfirmEmailByToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid := user.ID
	_ = h.Analytics.Track(&uid, "register", "user", &uid, "email_link")
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
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

	accessToken, refreshToken, user, err := h.AuthService.LoginWithContext(
		req.Email, req.Password, c.GetHeader("User-Agent"), c.ClientIP(),
	)
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

	user, err := h.UserService.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}
	if req.Title != "" {
		user.Title = req.Title
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	if req.BannerURL != "" {
		user.BannerURL = req.BannerURL
	}
	if req.ProfileTheme != "" {
		user.ProfileTheme = req.ProfileTheme
	}
	if req.AllowDMs != nil {
		user.AllowDMs = *req.AllowDMs
	}
	if req.IsPrivate != nil {
		user.IsPrivate = *req.IsPrivate
	}
	if req.EmailNotifyReply != nil {
		user.EmailNotifyReply = *req.EmailNotifyReply
	}
	if req.EmailNotifyMention != nil {
		user.EmailNotifyMention = *req.EmailNotifyMention
	}
	if req.EmailNotifyFollow != nil {
		user.EmailNotifyFollow = *req.EmailNotifyFollow
	}
	if req.EmailNotifyModeration != nil {
		user.EmailNotifyModeration = *req.EmailNotifyModeration
	}
	if req.EmailNotifyReport != nil {
		user.EmailNotifyReport = *req.EmailNotifyReport
	}

	if err := h.UserService.Save(user); err != nil {
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

func (h *Handlers) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req)
	_ = h.AuthService.Logout(req.RefreshToken)
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

	accessToken, refreshToken, err := h.AuthService.RefreshAccessTokenWithContext(
		req.RefreshToken, c.GetHeader("User-Agent"), c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *Handlers) ListSessions(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	sessions, err := h.AuthService.ListSessions(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (h *Handlers) RevokeSession(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	sessionID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.AuthService.RevokeSession(uid, sessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) RevokeOtherSessions(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var body struct {
		KeepSessionID uint `json:"keep_session_id"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.AuthService.RevokeOtherSessions(uid, body.KeepSessionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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
