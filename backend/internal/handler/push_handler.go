package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/featureflags"
	"nexus-forum-backend/internal/service"
)

func (h *Handlers) GetPushPublicKey(c *gin.Context) {
	if !h.requireFeature(c, featureflags.WebPush) {
		return
	}
	key := h.PushService.PublicKey()
	if key == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "push notifications not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_key": key})
}

func (h *Handlers) SubscribePush(c *gin.Context) {
	if !h.requireFeature(c, featureflags.WebPush) {
		return
	}
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

func (h *Handlers) GetPushDebug(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, h.PushService.VapidDebug(uid))
}

func (h *Handlers) GetPushStatus(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	hasSub, err := h.PushService.HasSubscription(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"subscribed":     hasSub,
		"vapid_configured": h.PushService.PublicKey() != "",
	})
}

func (h *Handlers) TestPush(c *gin.Context) {
	if !h.requireFeature(c, featureflags.WebPush) {
		return
	}
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	user, err := h.UserService.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	results, err := h.PushService.SendTest(user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "results": results})
		return
	}
	delivered := service.AnyDelivered(results)
	resp := gin.H{
		"delivered": delivered,
		"results":   results,
	}
	if !delivered {
		c.JSON(http.StatusBadGateway, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handlers) UnsubscribePush(c *gin.Context) {
	if !h.requireFeature(c, featureflags.WebPush) {
		return
	}
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
