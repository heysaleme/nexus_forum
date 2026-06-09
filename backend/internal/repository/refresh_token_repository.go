package repository

import (
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(token *model.RefreshToken) error
	GetValidToken(token string) (*model.RefreshToken, error)
	ListActiveByUser(userID uint) ([]*model.RefreshToken, error)
	Revoke(id uint) error
	RevokeByToken(token string) error
	RevokeByIDForUser(id, userID uint) error
	RevokeAllForUserExcept(userID uint, exceptID uint) error
	TouchLastUsed(id uint) error
	IsSessionActive(id uint) (bool, error)
	GetByToken(token string) (*model.RefreshToken, error)
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

func (r *refreshTokenRepository) RevokeByToken(token string) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("token = ? AND revoked = ?", token, false).
		Update("revoked", true).Error
}

func (r *refreshTokenRepository) ListActiveByUser(userID uint) ([]*model.RefreshToken, error) {
	var rows []*model.RefreshToken
	err := r.db.Where("user_id = ? AND revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Order("last_used_at DESC, created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *refreshTokenRepository) RevokeByIDForUser(id, userID uint) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("id = ? AND user_id = ? AND revoked = ?", id, userID, false).
		Update("revoked", true).Error
}

func (r *refreshTokenRepository) RevokeAllForUserExcept(userID uint, exceptID uint) error {
	q := r.db.Model(&model.RefreshToken{}).Where("user_id = ? AND revoked = ?", userID, false)
	if exceptID > 0 {
		q = q.Where("id != ?", exceptID)
	}
	return q.Update("revoked", true).Error
}

func (r *refreshTokenRepository) TouchLastUsed(id uint) error {
	return r.db.Model(&model.RefreshToken{}).Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *refreshTokenRepository) IsSessionActive(id uint) (bool, error) {
	if id == 0 {
		return true, nil
	}
	var count int64
	err := r.db.Model(&model.RefreshToken{}).
		Where("id = ? AND revoked = ? AND expires_at > ?", id, false, time.Now()).
		Count(&count).Error
	return count > 0, err
}

func (r *refreshTokenRepository) GetByToken(token string) (*model.RefreshToken, error) {
	var row model.RefreshToken
	err := r.db.Where("token = ?", token).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
