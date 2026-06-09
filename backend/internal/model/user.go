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
	FollowersCount int       `gorm:"default:0" json:"followers_count"`
	// Karma (computed from vote scores, not stored)
	PostKarma      int       `gorm:"-" json:"post_karma"`
	CommentKarma   int       `gorm:"-" json:"comment_karma"`
	TotalKarma     int       `gorm:"-" json:"total_karma"`
	FollowingCount int       `gorm:"default:0" json:"following_count"`
	Role           string    `gorm:"default:user" json:"role"` // "admin", "moderator", "user"
	AllowDMs       bool      `gorm:"default:true" json:"allow_dms"`
	IsPrivate      bool      `gorm:"default:false" json:"is_private"`
	IsBanned       bool      `gorm:"default:false" json:"is_banned"`
	IsShadowBanned bool      `gorm:"default:false" json:"-"`          // hidden from client — shadow ban is invisible to the banned user
	IsSuspicious   bool      `gorm:"default:false" json:"is_suspicious"` // exposed so frontend can show Turnstile widget
	OAuthProvider  string    `gorm:"default:''" json:"-"`             // "google" or ""
	OAuthSubject   string    `gorm:"default:''" json:"-"`             // provider-issued user ID
	LastSeenAt     time.Time `json:"last_seen_at"`
	IsOnline       bool      `gorm:"-" json:"is_online"`
	EmailVerified  bool      `gorm:"default:false" json:"email_verified"`
	EmailNotifyReply      bool `gorm:"default:true" json:"email_notify_reply"`
	EmailNotifyMention    bool `gorm:"default:true" json:"email_notify_mention"`
	EmailNotifyFollow     bool `gorm:"default:true" json:"email_notify_follow"`
	EmailNotifyModeration bool `gorm:"default:true" json:"email_notify_moderation"`
	EmailNotifyReport     bool `gorm:"default:true" json:"email_notify_report"`
	PushNotifyComments    bool `gorm:"default:true" json:"push_notify_comments"`
	PushNotifyReplies     bool `gorm:"default:true" json:"push_notify_replies"`
	PushNotifyMentions    bool `gorm:"default:true" json:"push_notify_mentions"`
	PushNotifyFollowers   bool `gorm:"default:true" json:"push_notify_followers"`
	PushNotifyMessages    bool `gorm:"default:true" json:"push_notify_messages"`
	PushNotifyModeration  bool `gorm:"default:true" json:"push_notify_moderation"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}


type UserFollow struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FollowerID  uint      `gorm:"not null;uniqueIndex:idx_follower_following" json:"follower_id"`
	FollowingID uint      `gorm:"not null;uniqueIndex:idx_follower_following" json:"following_id"`
	Status      string    `gorm:"default:'accepted';not null" json:"status"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`
}
