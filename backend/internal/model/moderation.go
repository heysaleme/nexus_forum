package model

import (
	"time"
)

// ModerationLog tracks all moderation actions (bans, removals, mutes, warns)
type ModerationLog struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ModeratorID uint       `gorm:"not null" json:"moderator_id"` // user who performed the action
	TargetID    uint       `gorm:"not null" json:"target_id"`    // user or content affected
	TargetType  string     `gorm:"not null" json:"target_type"`  // "user", "post", "comment"
	Action      string     `gorm:"not null" json:"action"`       // "ban", "unban", "remove_post", "remove_comment", "mute", "warn"
	Reason      string     `json:"reason"`
	CommunityID *uint      `json:"community_id"` // nil = global action
	ExpiresAt   *time.Time `json:"expires_at"`   // nil = permanent
	CreatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`

	// Helper fields
	ModeratorUsername string `gorm:"-" json:"moderator_username"`
}
