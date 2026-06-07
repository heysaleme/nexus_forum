package model

import (
	"time"
)

type Community struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"unique;not null" json:"name"`
	Slug        string    `gorm:"unique;not null" json:"slug"`
	Description string    `json:"description"`
	OwnerID     uint      `gorm:"not null" json:"owner_id"`
	MemberCount int       `gorm:"default:0" json:"member_count"`
	PostCount   int       `gorm:"default:0" json:"post_count"`
	Visibility  string    `gorm:"default:public" json:"visibility"` // "public", "private", "nsfw"
	AvatarURL   string    `json:"avatar_url"`
	BannerURL   string    `json:"banner_url"`
	Rules       string    `gorm:"type:text" json:"rules"` // JSON representation of rule objects
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}

type CommunityMember struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint      `gorm:"not null;uniqueIndex:idx_user_comm" json:"user_id"`
	CommunityID uint      `gorm:"not null;uniqueIndex:idx_user_comm" json:"community_id"`
	Role        string    `gorm:"default:member" json:"role"` // "owner", "moderator", "member"
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
