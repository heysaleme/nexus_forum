package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/dto"
	"nexus-forum-backend/internal/model"
)

// ================= Community Handlers =================

func (h *Handlers) CreateCommunity(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.CreateCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if isBase64DataURL(req.AvatarURL) || isBase64DataURL(req.BannerURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload community images via /api/upload; base64 data URLs are not allowed"})
		return
	}

	comm := &model.Community{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Visibility:  req.Visibility,
		AvatarURL:   req.AvatarURL,
		BannerURL:   req.BannerURL,
		Rules:       req.Rules,
		OwnerID:     uid,
	}

	err := h.CommService.Create(comm)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comm)
}

func (h *Handlers) GetCommunityByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err == nil {
		comm, err := h.CommService.GetByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "community not found"})
			return
		}
		c.JSON(http.StatusOK, comm)
		return
	}

	// Try Slug
	comm, err := h.CommService.GetBySlug(idStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "community not found"})
		return
	}
	c.JSON(http.StatusOK, comm)
}

func (h *Handlers) ListCommunities(c *gin.Context) {
	sortSpec := c.Query("sort")
	communities, err := h.CommService.List(sortSpec, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, communities)
}

func (h *Handlers) JoinCommunity(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	commID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.CommService.Join(uid, commID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) LeaveCommunity(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	commID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.CommService.Leave(uid, commID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetCommunityMembers(c *gin.Context) {
	commID, ok := parseID(c, "id")
	if !ok {
		return
	}

	members, err := h.CommService.GetMembers(commID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type memberView struct {
		ID          uint   `json:"id"`
		UserID      uint   `json:"user_id"`
		CommunityID uint   `json:"community_id"`
		Role        string `json:"role"`
		Username    string `json:"username"`
		AvatarURL   string `json:"avatar_url"`
		Title       string `json:"title"`
	}

	views := make([]memberView, 0, len(members))
	for _, m := range members {
		view := memberView{
			ID:          m.ID,
			UserID:      m.UserID,
			CommunityID: m.CommunityID,
			Role:        m.Role,
		}
		if u, err := h.UserService.GetByID(m.UserID); err == nil {
			view.Username = u.Username
			view.AvatarURL = u.AvatarURL
			view.Title = u.Title
		}
		views = append(views, view)
	}

	c.JSON(http.StatusOK, views)
}

func (h *Handlers) GetCommunityMemberships(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query param required"})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	memberships, err := h.CommService.GetMemberships(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, memberships)
}

func (h *Handlers) DeleteCommunity(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	commID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.CommService.Delete(uid, commID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
