package dto

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type OTPVerifyRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"otpCode" binding:"required"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetSubmit struct {
	ResetToken  string `json:"resetToken" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type UpdateProfileRequest struct {
	Username     string `json:"username"`
	Title        string `json:"title"`
	Bio          string `json:"bio"`
	AvatarURL    string `json:"avatar_url"`
	BannerURL    string `json:"banner_url"`
	ProfileTheme string `json:"profile_theme"`
	AllowDMs     *bool  `json:"allow_dms"`
	IsPrivate    *bool  `json:"is_private"`
}

type CreateCommunityRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"` // "public", "private", "nsfw"
	AvatarURL   string `json:"avatar_url"`
	BannerURL   string `json:"banner_url"`
	Rules       string `json:"rules"` // JSON array string
}

type CreatePostRequest struct {
	CommunityID uint     `json:"community_id" binding:"required"`
	Title       string   `json:"title" binding:"required"`
	Content     string   `json:"content"`
	Type        string   `json:"type" binding:"required"` // "text", "image", "link", "poll"
	MediaUrls   []string `json:"media_urls"`
	LinkUrl     string   `json:"link_url"`
	Tags        []string `json:"tags"`
	PollOptions interface{} `json:"poll_options"` // JSON representation or options struct
	IsNSFW      *bool    `json:"is_nsfw"`
	IsSpoiler   *bool    `json:"is_spoiler"`
}

type CreateCommentRequest struct {
	PostID   uint   `json:"post_id" binding:"required"`
	ParentID *uint  `json:"parent_id"`
	Content  string `json:"content" binding:"required"`
}

type VoteRequest struct {
	Value int `json:"value" binding:"required"` // 1 or -1
}

type CreateChatRoomRequest struct {
	Participants []uint `json:"participants" binding:"required"`
	Name         string `json:"name"`
}

type CreateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}
