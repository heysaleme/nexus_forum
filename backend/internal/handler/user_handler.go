package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/service"
)

func viewerCanSeeModerationFields(c *gin.Context, targetUserID uint) bool {
	roleVal, exists := c.Get("role")
	if !exists {
		return false
	}
	role := roleVal.(string)
	if role != "admin" && role != "moderator" {
		return false
	}
	if uid, ok := c.Get("userID"); ok {
		if uid.(uint) == targetUserID {
			return false
		}
	}
	return true
}

func userResponse(u *model.User, includeModeration bool) map[string]interface{} {
	data, _ := json.Marshal(u)
	var out map[string]interface{}
	_ = json.Unmarshal(data, &out)
	if includeModeration {
		out["is_shadow_banned"] = u.IsShadowBanned
	}
	return out
}

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

	// Verify privacy for stats
	reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
	isAuthorized := false
	if user.IsPrivate {
		if isAuthenticated {
			if reqUserID == user.ID {
				isAuthorized = true
			} else {
				following, _ := h.UserService.IsFollowing(reqUserID, user.ID)
				if following {
					isAuthorized = true
				}
			}
		}
		if !isAuthorized {
			user.FollowersCount = 0
			user.FollowingCount = 0
			user.XP = 0
			user.Level = 1
			user.Bio = ""
		}
	}

	user.IsOnline = h.WSHub.IsUserOnline(user.ID)
	includeMod := viewerCanSeeModerationFields(c, user.ID)
	if includeMod {
		c.JSON(http.StatusOK, userResponse(user, true))
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handlers) GetUserProfileStats(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	stats, err := h.UserService.GetProfileStats(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handlers) GetUserAchievements(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	items, err := h.UserService.GetAchievements(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if items == nil {
		items = []service.Achievement{}
	}
	c.JSON(http.StatusOK, items)
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

	reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
	roleVal, _ := c.Get("role")
	viewerRole, _ := roleVal.(string)
	isModeratorViewer := viewerRole == "admin" || viewerRole == "moderator"

	var visibleUsers []map[string]interface{}
	for _, u := range users {
		u.IsOnline = h.WSHub.IsUserOnline(u.ID)
		if u.IsPrivate && !isModeratorViewer {
			isAuthorized := false
			if isAuthenticated {
				if reqUserID == u.ID {
					isAuthorized = true
				} else {
					following, _ := h.UserService.IsFollowing(reqUserID, u.ID)
					if following {
						isAuthorized = true
					}
				}
			}
			if !isAuthorized {
				continue
			}
		}
		visibleUsers = append(visibleUsers, userResponse(u, isModeratorViewer))
	}

	c.JSON(http.StatusOK, visibleUsers)
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

	actorID, ok := getUserID(c)
	if !ok {
		return
	}

	updated, err := h.UserService.UpdateUser(actorID, targetID, req.Role, req.IsBanned)
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

	// Verify privacy
	targetUser, err := h.UserService.GetByID(userID)
	if err == nil && targetUser.IsPrivate {
		reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
		isAuthorized := false
		if isAuthenticated {
			if reqUserID == userID {
				isAuthorized = true
			} else {
				following, _ := h.UserService.IsFollowing(reqUserID, userID)
				if following {
					isAuthorized = true
				}
			}
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "This account is private. Follow to view followers."})
			return
		}
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

	// Verify privacy
	targetUser, err := h.UserService.GetByID(userID)
	if err == nil && targetUser.IsPrivate {
		reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
		isAuthorized := false
		if isAuthenticated {
			if reqUserID == userID {
				isAuthorized = true
			} else {
				following, _ := h.UserService.IsFollowing(reqUserID, userID)
				if following {
					isAuthorized = true
				}
			}
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "This account is private. Follow to view following."})
			return
		}
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

	err := h.ModService.CreateReport(uid, req.TargetType, req.TargetID, req.Reason, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Report submitted"})
}

func (h *Handlers) GetFollowRequests(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	requests, err := h.UserService.GetPendingFollowRequests(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *Handlers) AcceptFollowRequest(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	followerID, ok := parseID(c, "follower_id")
	if !ok {
		return
	}

	err := h.UserService.AcceptFollowRequest(followerID, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) RejectFollowRequest(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	followerID, ok := parseID(c, "follower_id")
	if !ok {
		return
	}

	err := h.UserService.RejectFollowRequest(followerID, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetFollowStatus(c *gin.Context) {
	reqUserID, isAuthenticated := getOptionalUserID(c, h.AuthService)
	if !isAuthenticated {
		c.JSON(http.StatusOK, gin.H{"status": "none"})
		return
	}

	targetID, ok := parseID(c, "id")
	if !ok {
		return
	}

	follow, err := h.UserService.GetFollowRecord(reqUserID, targetID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "none"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": follow.Status})
}
