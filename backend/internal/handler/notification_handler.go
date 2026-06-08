package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/model"
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

	h.pushUnreadCount(uid)

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

	h.pushUnreadCount(uid)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetUnreadNotificationCount(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var count int64
	if h.WSHub.db != nil {
		err := h.WSHub.db.Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", uid, false).Count(&count).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *Handlers) pushUnreadCount(userID uint) {
	if h.WSHub.db == nil {
		return
	}
	var count int64
	h.WSHub.db.Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)
	payload, err := json.Marshal(struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}{
		Type:  "unread_count",
		Count: count,
	})
	if err == nil {
		h.WSHub.SendToUser(userID, payload)
	}
}
