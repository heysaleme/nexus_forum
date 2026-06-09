package model

import (
	"time"
)

type Post struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	CommunityID  uint   `gorm:"not null;index" json:"community_id"`
	AuthorID     uint   `gorm:"not null;index:idx_posts_author_status,priority:1" json:"author_id"`
	Title        string `gorm:"not null" json:"title"`
	Content      string `json:"content"`
	Type         string `gorm:"default:text" json:"type"` // "text", "image", "link", "poll"
	Score        int    `gorm:"default:0" json:"score"`
	Upvotes      int    `gorm:"default:0" json:"upvotes"`
	Downvotes    int    `gorm:"default:0" json:"downvotes"`
	Views        int    `gorm:"default:0" json:"views"`
	CommentCount int    `gorm:"default:0" json:"comment_count"`
	Status       string `gorm:"default:published;index:idx_posts_author_status,priority:2" json:"status"` // "draft", "published", "scheduled", "removed"
	PublishAt    *time.Time `gorm:"index" json:"publish_at,omitempty"`
	MediaUrls    string `gorm:"type:text" json:"media_urls"`     // JSON array of strings
	LinkUrl      string `json:"link_url"`
	Tags         string `gorm:"type:text" json:"tags"` // JSON array of strings
	SearchBlob   string `gorm:"type:text" json:"-"`    // normalized lowercase index for Unicode search
	PollOptions  string `gorm:"type:text" json:"poll_options"`
	PollVotes    string `gorm:"type:text" json:"poll_votes"`

	OriginalPostID  *uint     `gorm:"index" json:"original_post_id,omitempty"`
	IsCrosspost     bool      `gorm:"default:false" json:"is_crosspost"`
	IsPinned        bool      `gorm:"default:false" json:"is_pinned"`
	IsNSFW          bool      `gorm:"default:false" json:"is_nsfw"`
	IsSpoiler       bool      `gorm:"default:false" json:"is_spoiler"`
	IsShadowContent bool      `gorm:"default:false" json:"-"` // hidden from non-authors when true
	CreatedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`

	// Crosspost helper fields
	OriginalPostTitle string `gorm:"-" json:"original_post_title,omitempty"`
	OriginalCommunity string `gorm:"-" json:"original_community_name,omitempty"`

	// Helper fields to join for responses
	AuthorUsername  string `gorm:"-" json:"author_username"`
	AuthorAvatar    string `gorm:"-" json:"author_avatar"`
	CommunityName   string `gorm:"-" json:"community_name"`
	CommunityAvatar string `gorm:"-" json:"community_avatar"`

	// Current user vote
	UserVote     int `gorm:"-" json:"user_vote"`
	UserPollVote int `gorm:"-" json:"user_poll_vote"`
}

type SavedPost struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_save" json:"user_id"`
	PostID    uint      `gorm:"not null;uniqueIndex:idx_user_save" json:"post_id"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_date"`

	// Helper fields for details
	PostTitle     string `gorm:"-" json:"post_title"`
	PostCommunity string `gorm:"-" json:"post_community"`
}
