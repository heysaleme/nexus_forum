package model

import "time"

type PushSubscription struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Endpoint  string    `gorm:"type:text;not null" json:"endpoint"`
	P256DH    string    `gorm:"not null" json:"p256dh"`
	Auth      string    `gorm:"not null" json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}
