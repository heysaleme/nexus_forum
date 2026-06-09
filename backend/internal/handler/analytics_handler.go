package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ================= Analytics Handlers =================

func (h *Handlers) GetAnalyticsDashboard(c *gin.Context) {
	if !requireAdminRole(c) {
		return
	}

	dashboard, err := h.Analytics.GetDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

func (h *Handlers) GetAnalyticsActivity(c *gin.Context) {
	if !requireAdminRole(c) {
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	activity, err := h.Analytics.GetActivity(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activity": activity, "days": days})
}

func (h *Handlers) GetAnalyticsReports(c *gin.Context) {
	if !requireAdminRole(c) {
		return
	}
	reasons, err := h.Analytics.GetReportReasons()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report_reasons": reasons})
}

func requireAdminRole(c *gin.Context) bool {
	roleVal, exists := c.Get("role")
	if !exists || roleVal.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return false
	}
	return true
}

func (h *Handlers) GetAnalyticsRetention(c *gin.Context) {
	if !requireAdminRole(c) {
		return
	}
	retention, err := h.Analytics.GetRetention()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"retention": retention})
}

func (h *Handlers) GetAnalyticsEngagement(c *gin.Context) {
	if !requireAdminRole(c) {
		return
	}
	engagement, err := h.Analytics.GetEngagement()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, engagement)
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
