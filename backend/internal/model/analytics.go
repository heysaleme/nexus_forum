package model

import (
	"time"
)

// AnalyticsEvent tracks user actions for analytics dashboard
type AnalyticsEvent struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     *uint     `json:"user_id"`                    // nil = anonymous
	EventType  string    `gorm:"not null" json:"event_type"` // "page_view", "post_create", "comment_create", "vote", "register", "login"
	EntityType string    `json:"entity_type"`                // "post", "community", "user", ""
	EntityID   *uint     `json:"entity_id"`
	Metadata   string    `gorm:"type:text" json:"metadata"` // JSON extras
	CreatedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
