package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ================= Notification Handlers =================

func (h *Handlers) GetNotifications(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	notifications, err := h.NotifService.GetByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (h *Handlers) MarkAllNotificationsRead(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.NotifService.MarkAllRead(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) MarkNotificationRead(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	notifID, ok := parseID(c, "id")
	if !ok {
		return
	}

	err := h.NotifService.MarkRead(notifID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
