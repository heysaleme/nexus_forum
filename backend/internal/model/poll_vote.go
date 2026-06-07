package model

import "time"

type PollVote struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"uniqueIndex:idx_poll_vote"`
	PostID    uint `gorm:"uniqueIndex:idx_poll_vote"`
	OptionIdx int  `gorm:"not null"`
	CreatedAt time.Time
}
