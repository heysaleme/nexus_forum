package handler

import (
	"net/http"

	"nexus-forum-backend/internal/featureflags"

	"github.com/gin-gonic/gin"
)

// requireFeature aborts with 403 when the flag is disabled or missing.
func (h *Handlers) requireFeature(c *gin.Context, key string) bool {
	if h.FeatureFlags != nil && h.FeatureFlags.IsEnabled(key) {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error":   "feature disabled",
		"feature": key,
	})
	return false
}

// liveWSAllowed is used by broadcast helpers (no gin context).
func (h *Handlers) liveWSAllowed() bool {
	return h.FeatureFlags != nil && h.FeatureFlags.IsEnabled(featureflags.LiveWS)
}
