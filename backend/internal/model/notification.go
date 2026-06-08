package model

import (
	"time"
)

type Notification struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Type        string    `gorm:"not null" json:"type"` // "reply", "follow"
	Title       string    `gorm:"not null" json:"title"`
	Body        string    `gorm:"not null" json:"body"`
	ActorAvatar string    `json:"actor_avatar"`
	IsRead      bool      `gorm:"default:false" json:"is_read"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
