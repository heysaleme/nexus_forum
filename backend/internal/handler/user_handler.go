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
