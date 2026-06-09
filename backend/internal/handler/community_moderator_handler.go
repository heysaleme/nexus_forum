package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) PromoteCommunityModerator(c *gin.Context) {
	actorID, ok := getUserID(c)
	if !ok {
		return
	}
	commID, ok := parseID(c, "id")
	if !ok {
		return
	}
	targetID, ok := parseID(c, "userId")
	if !ok {
		return
	}
	if err := h.CommService.PromoteModerator(actorID, commID, targetID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) DemoteCommunityModerator(c *gin.Context) {
	actorID, ok := getUserID(c)
	if !ok {
		return
	}
	commID, ok := parseID(c, "id")
	if !ok {
		return
	}
	targetID, ok := parseID(c, "userId")
	if !ok {
		return
	}
	if err := h.CommService.DemoteModerator(actorID, commID, targetID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) ListCommunityModerators(c *gin.Context) {
	commID, ok := parseID(c, "id")
	if !ok {
		return
	}
	mods, err := h.CommService.ListModerators(commID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mods)
}
