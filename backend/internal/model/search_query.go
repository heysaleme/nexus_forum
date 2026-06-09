package model

import "time"

type SearchQuery struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Query     string    `gorm:"not null;index" json:"query"`
	Count     int       `gorm:"default:1" json:"count"`
	UpdatedAt time.Time `gorm:"index" json:"updated_at"`
}
