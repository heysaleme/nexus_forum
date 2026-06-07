package model

import (
	"time"
)

type User struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username       string    `gorm:"unique;not null" json:"username"`
	Email          string    `gorm:"unique;not null" json:"email"`
	PasswordHash   string    `gorm:"not null" json:"-"`
	AvatarURL      string    `json:"avatar_url"`
	BannerURL      string    `json:"banner_url"`
	Bio            string    `json:"bio"`
	Title          string    `json:"title"`
	ProfileTheme   string    `gorm:"default:default" json:"profile_theme"`
	Level          int       `gorm:"default:1" json:"level"`
	XP             int       `gorm:"default:0" json:"xp"`
	FollowersCount int       `gorm:"default:0" json:"followers_count"`
	FollowingCount int       `gorm:"default:0" json:"following_count"`
	Role           string    `gorm:"default:user" json:"role"` // "admin", "moderator", "user"
	AllowDMs       bool      `gorm:"default:true" json:"allow_dms"`
	IsPrivate      bool      `gorm:"default:false" json:"is_private"`
	IsBanned       bool      `gorm:"default:false" json:"is_banned"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}

type UserFollow struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FollowerID  uint      `gorm:"not null;uniqueIndex:idx_follower_following" json:"follower_id"`
	FollowingID uint      `gorm:"not null;uniqueIndex:idx_follower_following" json:"following_id"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
