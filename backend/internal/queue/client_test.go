package queue

import (
	"encoding/json"
	"testing"
)

func TestNotificationEventJSON(t *testing.T) {
	evt := NotificationEvent{
		UserID: 42,
		Type:   "reply",
		Data:   map[string]interface{}{"title": "t", "body": "b"},
	}
	b, err := json.Marshal(evt)
	if err != nil {
		t.Fatal(err)
	}
	var decoded NotificationEvent
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.UserID != 42 || decoded.Type != "reply" {
		t.Fatalf("decoded mismatch: %+v", decoded)
	}
}

func TestNewClientEmptyURL(t *testing.T) {
	c := NewClient("")
	if c.Enabled() {
		t.Fatal("expected disabled client")
	}
}
