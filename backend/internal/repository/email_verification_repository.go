package repository

import (
	"time"

	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type EmailVerificationRepository interface {
	Upsert(row *model.EmailVerification) error
	GetValid(email, code string) (*model.EmailVerification, error)
	GetPendingByEmail(email string) (*model.EmailVerification, error)
	DeleteByEmail(email string) error
}

type emailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(db *gorm.DB) EmailVerificationRepository {
	return &emailVerificationRepository{db: db}
}

func (r *emailVerificationRepository) Upsert(row *model.EmailVerification) error {
	_ = r.DeleteByEmail(row.Email)
	return r.db.Create(row).Error
}

func (r *emailVerificationRepository) GetValid(email, code string) (*model.EmailVerification, error) {
	var row model.EmailVerification
	err := r.db.Where("email = ? AND code = ? AND expires_at > ?", email, code, time.Now()).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *emailVerificationRepository) GetPendingByEmail(email string) (*model.EmailVerification, error) {
	var row model.EmailVerification
	err := r.db.Where("email = ? AND expires_at > ?", email, time.Now()).
		Order("created_at DESC").
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *emailVerificationRepository) DeleteByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&model.EmailVerification{}).Error
}
