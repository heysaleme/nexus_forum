package model

type PostSearchToken struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	PostID uint   `gorm:"not null;index:idx_post_search_post"`
	Token  string `gorm:"not null;index:idx_post_search_token"`
	Stem   string `gorm:"not null;index:idx_post_search_stem"`
	Kind   string `gorm:"not null;default:token"`
}
