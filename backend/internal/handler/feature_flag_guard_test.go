package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"nexus-forum-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupFeatureFlagHandlers(t *testing.T, enabled bool) *Handlers {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.FeatureFlag{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = db.Create(&model.FeatureFlag{Key: "crosspost", Enabled: enabled, Description: "crosspost"}).Error
	flagSvc := service.NewFeatureFlagService(repository.NewFeatureFlagRepository(db))
	return &Handlers{FeatureFlags: flagSvc}
}

func TestCreateCrosspost_FeatureDisabled(t *testing.T) {
	h := setupFeatureFlagHandlers(t, false)
	body, _ := json.Marshal(map[string]interface{}{
		"original_post_id":    1,
		"target_community_id": 2,
		"title":               "x",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/posts/crosspost", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userID", uint(1))

	h.CreateCrosspost(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}
