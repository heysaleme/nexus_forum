package model

import "time"

type FeatureFlag struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string    `gorm:"uniqueIndex;not null" json:"key"`
	Enabled     bool      `gorm:"default:false" json:"enabled"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
