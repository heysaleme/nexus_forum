package repository

import (
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type PasswordResetRepository interface {
	Create(token *model.PasswordResetToken) error
	GetValidToken(token string) (*model.PasswordResetToken, error)
	MarkUsed(id uint) error
	DeleteByUserID(userID uint) error
}

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(token *model.PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *passwordResetRepository) GetValidToken(token string) (*model.PasswordResetToken, error) {
	var row model.PasswordResetToken
	err := r.db.Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *passwordResetRepository) MarkUsed(id uint) error {
	return r.db.Model(&model.PasswordResetToken{}).Where("id = ?", id).Update("used", true).Error
}

func (r *passwordResetRepository) DeleteByUserID(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.PasswordResetToken{}).Error
}
