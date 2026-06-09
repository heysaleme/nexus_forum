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

func TestAnyDelivered(t *testing.T) {
	if AnyDelivered([]PushDeliveryResult{{Delivered: false}, {HTTPStatusCode: 410}}) {
		t.Fatal("expected false")
	}
	if !AnyDelivered([]PushDeliveryResult{{Delivered: true, HTTPStatusCode: 201}}) {
		t.Fatal("expected true")
	}
}

func TestIsPushDelivered(t *testing.T) {
	if !isPushDelivered(201, nil) {
		t.Fatal("201 should be delivered")
	}
	if isPushDelivered(410, nil) {
		t.Fatal("410 should not be delivered")
	}
}
