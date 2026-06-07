package model

import (
	"time"
)

type Vote struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_user_vote" json:"user_id"`
	EntityType string    `gorm:"not null;uniqueIndex:idx_user_vote" json:"entity_type"` // "post", "comment"
	EntityID   uint      `gorm:"not null;uniqueIndex:idx_user_vote" json:"entity_id"`
	Value      int       `gorm:"not null" json:"value"` // -1 or 1
	CreatedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
