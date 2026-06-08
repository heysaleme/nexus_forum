package model

import (
	"time"
)

type Report struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReporterID        uint      `gorm:"not null" json:"reporter_id"`
	ReporterUsername  string    `json:"reporter_username"`
	TargetID          uint      `gorm:"not null" json:"target_id"`
	TargetType        string    `gorm:"not null" json:"target_type"` // "post", "comment", "user"
	Reason            string    `gorm:"not null" json:"reason"`
	Description       string    `json:"description"`
	Status            string    `gorm:"default:'pending';not null" json:"status"` // "pending", "resolved", "rejected"
	ModeratorResponse string    `json:"moderator_response"`
	CreatedAt         time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
