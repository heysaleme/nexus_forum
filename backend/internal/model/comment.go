package model

import (
	"time"
)

type Comment struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID          uint      `gorm:"not null;index" json:"post_id"`
	ParentID        *uint     `json:"parent_id"`
	ReplyToUserID   *uint     `gorm:"index" json:"reply_to_user_id"`
	AuthorID        uint      `gorm:"not null" json:"author_id"`
	Content         string    `gorm:"not null" json:"content"`
	Score           int       `gorm:"default:0" json:"score"`
	IsDeleted       bool      `gorm:"default:false" json:"is_deleted"`
	IsShadowContent bool      `gorm:"default:false" json:"-"`
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
	UserVote        int       `gorm:"-" json:"user_vote"`

	// Helper fields to join for responses
	AuthorUsername   string `gorm:"-" json:"author_username"`
	AuthorAvatar     string `gorm:"-" json:"author_avatar"`
	ReplyToUsername  string `gorm:"-" json:"reply_to_username"`
}
