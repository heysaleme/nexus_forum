package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireAdminOrMod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{"regular user denied", "user", http.StatusForbidden},
		{"moderator allowed", "moderator", http.StatusOK},
		{"admin allowed", "admin", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				c.Set("role", tc.role)
				c.Next()
			}, RequireAdminOrMod(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{"regular user denied", "user", http.StatusForbidden},
		{"moderator denied", "moderator", http.StatusForbidden},
		{"admin allowed", "admin", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				c.Set("role", tc.role)
				c.Next()
			}, RequireAdmin(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}
