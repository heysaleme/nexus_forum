package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"nexus-forum-backend/internal/resilience"
)

// ProbeCircuitBreaker exposes breaker state for admin verification.
// POST /api/admin/circuit-breaker/:name/probe?action=trip|call|status
func (h *Handlers) ProbeCircuitBreaker(c *gin.Context) {
	name := c.Param("name")
	action := c.DefaultQuery("action", "status")

	switch action {
	case "trip":
		resilience.ForceOpen(name)
	case "call":
		_, err := resilience.Execute(name, func() (interface{}, error) {
			return nil, errors.New("probe failure")
		})
		c.JSON(http.StatusOK, gin.H{
			"name":  name,
			"state": resilience.State(name).String(),
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":  name,
		"state": resilience.State(name).String(),
	})
}
