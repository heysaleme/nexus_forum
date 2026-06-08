package model

import "time"

// KeywordFilter stores a banned word/pattern that the auto-moderation system checks content against.
type KeywordFilter struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Pattern   string    `gorm:"not null;uniqueIndex" json:"pattern"` // plain text or regex
	IsRegex   bool      `gorm:"default:false" json:"is_regex"`
	// Action controls what happens when content matches:
	//   "block"  — reject the content with an error (author sees the rejection)
	//   "shadow" — silently hide the content from other users (only the author sees it)
	Action    string    `gorm:"default:'block'" json:"action"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
}
