package model

import "time"

type EmailVerification struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string    `gorm:"index;not null" json:"email"`
	Code         string    `gorm:"not null" json:"-"`
	Token        string    `gorm:"uniqueIndex" json:"-"`
	PasswordHash string    `gorm:"not null" json:"-"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
