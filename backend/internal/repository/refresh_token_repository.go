package repository

import (
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(token *model.RefreshToken) error
	GetValidToken(token string) (*model.RefreshToken, error)
	Revoke(id uint) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *refreshTokenRepository) GetValidToken(token string) (*model.RefreshToken, error) {
	var row model.RefreshToken
	err := r.db.Where("token = ? AND revoked = ? AND expires_at > ?", token, false, time.Now()).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *refreshTokenRepository) Revoke(id uint) error {
	return r.db.Model(&model.RefreshToken{}).Where("id = ?", id).Update("revoked", true).Error
}
