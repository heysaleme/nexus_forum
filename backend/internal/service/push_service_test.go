package service

import (
	"testing"

	"nexus-forum-backend/internal/model"
)

func TestPushService_shouldSend_RespectsPreferences(t *testing.T) {
	svc := NewPushService(nil, "pub", "priv", "mailto:t@e.com").(*pushService)
	user := &model.User{
		PushNotifyComments: false,
		PushNotifyReplies:  true,
	}
	if svc.shouldSend(user, "comment") {
		t.Fatal("comment push should be disabled")
	}
	if !svc.shouldSend(user, "reply") {
		t.Fatal("reply push should be enabled")
	}
	if !svc.shouldSend(user, "test") {
		t.Fatal("test push should always send when configured")
	}
}
