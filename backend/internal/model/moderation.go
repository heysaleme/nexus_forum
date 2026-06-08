package model

import (
	"time"
)

// ModerationLog tracks all moderation actions (bans, removals, mutes, warns)
type ModerationLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ActorID    uint      `gorm:"not null;column:actor_id" json:"actor_id"`
	TargetID   uint      `gorm:"not null;column:target_id" json:"target_id"`
	TargetType string    `gorm:"not null;column:target_type" json:"target_type"`
	Action      string    `gorm:"not null;column:action" json:"action"`
	Details     string    `gorm:"column:details" json:"details"`
	CommunityID *uint     `gorm:"index;column:community_id" json:"community_id,omitempty"`
	CreatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP;column:created_at" json:"created_date"`

	// Helper fields
	ModeratorUsername string `gorm:"-" json:"moderator_username"`
}
