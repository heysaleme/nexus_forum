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

	c.JSON(http.StatusOK, members)
}
