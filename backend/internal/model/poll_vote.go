package model

import "time"

type PollVote struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	PostID    uint      `gorm:"not null"`
	OptionIdx int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
