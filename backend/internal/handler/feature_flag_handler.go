package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListFeatureFlags(c *gin.Context) {
	flags, err := h.FeatureFlags.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flags)
}

func (h *Handlers) UpdateFeatureFlag(c *gin.Context) {
	key := c.Param("key")
	var body struct {
		Enabled     bool   `json:"enabled"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	flag, err := h.FeatureFlags.Set(key, body.Enabled, body.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flag)
}

func (h *Handlers) GetPublicFeatureFlags(c *gin.Context) {
	flags, err := h.FeatureFlags.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make(map[string]bool, len(flags))
	for _, f := range flags {
		out[f.Key] = f.Enabled
	}
	c.JSON(http.StatusOK, out)
}
