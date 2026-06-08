package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRoles aborts with 403 unless the JWT role is in the allowed set.
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions: role required"})
			c.Abort()
			return
		}
		role, ok := roleVal.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions: invalid role"})
			c.Abort()
			return
		}
		if _, ok := allowed[role]; !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions: admin or moderator role required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAdmin allows only admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists || roleVal.(string) != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAdminOrMod allows admin and moderator roles.
func RequireAdminOrMod() gin.HandlerFunc {
	return RequireRoles("admin", "moderator")
}
