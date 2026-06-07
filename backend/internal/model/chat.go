package model

import (
	"time"
)

type ChatRoom struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `json:"name"`
	Type          string    `gorm:"default:direct" json:"type"`   // "direct", "group"
	Participants  string    `gorm:"type:text" json:"participants"` // JSON array of user IDs
	AvatarURL     string    `json:"avatar_url"`
	LastMessage   string    `json:"last_message"`
	LastMessageAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_message_at"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}

type Message struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID     uint      `gorm:"not null" json:"chat_room_id"`
	SenderID       uint      `gorm:"not null" json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	SenderAvatar   string    `json:"sender_avatar"`
	Content        string    `gorm:"not null" json:"content"`
	MessageType    string    `gorm:"default:text" json:"message_type"` // "text", "image"
	IsRead         bool      `gorm:"default:false" json:"is_read"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
