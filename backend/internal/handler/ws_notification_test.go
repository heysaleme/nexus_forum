package handler

import (
	"encoding/json"
	"testing"
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSendToUser_DeliversNotificationAndUnreadCount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	hub := NewWSHub(db)
	client := &wsClient{
		roomID: 0,
		userID: 42,
		send:   make(chan []byte, 8),
		hub:    hub,
	}
	hub.join <- client
	time.Sleep(20 * time.Millisecond)

	notif := &model.Notification{
		ID:     7,
		UserID: 42,
		Type:   "comment",
		Title:  "Новый комментарий",
		Body:   "test",
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"type": "notification",
		"data": notif,
	})
	if sent := hub.SendToUser(42, payload); sent != 1 {
		t.Fatalf("expected 1 client reached, got %d", sent)
	}

	select {
	case msg := <-client.send:
		var envelope struct {
			Type string               `json:"type"`
			Data *model.Notification  `json:"data"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if envelope.Type != "notification" || envelope.Data == nil || envelope.Data.Type != "comment" {
			t.Fatalf("unexpected payload: %s", string(msg))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification on client channel")
	}

	countPayload, _ := json.Marshal(map[string]interface{}{"type": "unread_count", "count": 3})
	if sent := hub.SendToUser(42, countPayload); sent != 1 {
		t.Fatalf("expected unread_count to reach 1 client, got %d", sent)
	}
	select {
	case msg := <-client.send:
		var envelope struct {
			Type  string `json:"type"`
			Count int64  `json:"count"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			t.Fatalf("unmarshal count: %v", err)
		}
		if envelope.Type != "unread_count" || envelope.Count != 3 {
			t.Fatalf("unexpected unread_count payload: %s", string(msg))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for unread_count")
	}

	hub.leave <- client
}
