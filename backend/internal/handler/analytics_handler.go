package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ================= Analytics Handlers =================

func (h *Handlers) GetAnalyticsDashboard(c *gin.Context) {
	_, ok := getUserID(c)
	if !ok {
		return
	}

	dashboard, err := h.Analytics.GetDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

func (h *Handlers) TrackEvent(c *gin.Context) {
	var body struct {
		EventType  string `json:"event_type" binding:"required"`
		EntityType string `json:"entity_type"`
		EntityID   *uint  `json:"entity_id"`
		Meta       string `json:"meta"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get optional userID from context (may not be authenticated)
	var userIDPtr *uint
	if uid, exists := c.Get("userID"); exists {
		id := uid.(uint)
		userIDPtr = &id
	}

	_ = h.Analytics.Track(userIDPtr, body.EventType, body.EntityType, body.EntityID, body.Meta)
	c.JSON(http.StatusOK, gin.H{"tracked": true})
}
