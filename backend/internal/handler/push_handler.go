package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) GetPushPublicKey(c *gin.Context) {
	key := h.PushService.PublicKey()
	if key == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "push notifications not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_key": key})
}

func (h *Handlers) SubscribePush(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var body struct {
		Endpoint string `json:"endpoint"`
		P256DH   string `json:"p256dh"`
		Auth     string `json:"auth"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.PushService.Subscribe(uid, body.Endpoint, body.P256DH, body.Auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) UnsubscribePush(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.PushService.Unsubscribe(uid, body.Endpoint)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
