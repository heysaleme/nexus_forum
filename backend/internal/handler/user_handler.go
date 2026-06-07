package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ================= User Handlers =================

func (h *Handlers) GetUserByID(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	user, err := h.UserService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handlers) Follow(c *gin.Context) {
	followerID, ok := getUserID(c)
	if !ok {
		return
	}

	followingID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.UserService.Follow(followerID, followingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) Unfollow(c *gin.Context) {
	followerID, ok := getUserID(c)
	if !ok {
		return
	}

	followingID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.UserService.Unfollow(followerID, followingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) ListUsers(c *gin.Context) {
	sortSpec := c.Query("sort")
	users, err := h.UserService.List(sortSpec, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handlers) UpdateUser(c *gin.Context) {
	roleVal, exists := c.Get("role")
	if !exists || roleVal.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	targetID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req struct {
		Role     string `json:"role"`
		IsBanned *bool  `json:"is_banned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.UserService.UpdateUser(targetID, req.Role, req.IsBanned)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// GetFollowers returns the list of followers for a user
func (h *Handlers) GetFollowers(c *gin.Context) {
	userID, ok := parseID(c, "id")
	if !ok {
		return
	}
	followers, err := h.UserService.GetFollowers(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, followers)
}

// GetFollowing returns the list of users a given user follows
func (h *Handlers) GetFollowing(c *gin.Context) {
	userID, ok := parseID(c, "id")
	if !ok {
		return
	}
	following, err := h.UserService.GetFollowing(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, following)
}

// CreateReport handles POST /reports — any logged-in user can submit a report
func (h *Handlers) CreateReport(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req struct {
		TargetID    uint   `json:"target_id"`
		TargetType  string `json:"target_type"`
		Reason      string `json:"reason"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Persist via moderation service log (reuse existing model)
	user, _ := h.UserService.GetByID(uid)
	username := "user"
	if user != nil {
		username = user.Username
	}

	err := h.ModService.CreateModerationLog(uid, username, "report", req.TargetType, req.TargetID, req.Reason+": "+req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Report submitted"})
}
