package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pushcfg "nexus-forum-backend/internal/push"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
	"nexus-forum-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPushTest(t *testing.T) (*Handlers, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.PushSubscription{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := &model.User{
		Username:     "pushuser",
		Email:        "push@example.com",
		PasswordHash: "x",
		ProfileTheme: "default",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	pushRepo := repository.NewPushSubscriptionRepository(db)
	pushSvc := service.NewPushService(pushRepo, &pushcfg.Config{
		PublicKey:        "test-public",
		PrivateKey:       "test-private",
		Subscriber:       "test@example.com",
		JWTSubject:       "mailto:test@example.com",
		ConfiguredPublic: "test-public",
		DerivedPublicKey: "test-public",
		KeysMatch:        true,
	})
	h := &Handlers{
		PushService: pushSvc,
		UserService: service.NewUserService(
			repository.NewUserRepository(db),
			repository.NewFollowRepository(db),
			repository.NewNotificationRepository(db),
			repository.NewModerationRepository(db),
			repository.NewKarmaRepository(db),
		),
	}
	return h, db
}

func TestSubscribePush_CreatesRow(t *testing.T) {
	h, db := setupPushTest(t)
	body, _ := json.Marshal(map[string]string{
		"endpoint": "https://push.example.com/sub/1",
		"p256dh":   "key",
		"auth":     "secret",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/push/subscribe", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userID", uint(1))

	h.SubscribePush(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}

	var count int64
	if err := db.Model(&model.PushSubscription{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 subscription, got %d", count)
	}
}

func TestGetPushStatus(t *testing.T) {
	h, db := setupPushTest(t)
	_ = db.Create(&model.PushSubscription{
		UserID:   1,
		Endpoint: "https://push.example.com/sub/1",
		P256DH:   "k",
		Auth:     "a",
	}).Error

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/push/status", nil)
	c.Set("userID", uint(1))

	h.GetPushStatus(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("json: %v", err)
	}
	if res["subscribed"] != true {
		t.Fatalf("expected subscribed true, got %v", res["subscribed"])
	}
}
