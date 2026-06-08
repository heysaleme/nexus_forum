package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Health returns service and database readiness for probes and monitoring.
func Health(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		if sqlDB, err := db.DB(); err != nil {
			dbStatus = "error"
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = "error"
		}

		status := http.StatusOK
		if dbStatus != "ok" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"status":   "ok",
			"db":       dbStatus,
			"time":     time.Now().Format(time.RFC3339),
			"version":  "1.0.0",
			"services": gin.H{"api": "ok", "database": dbStatus},
		})
	}
}
