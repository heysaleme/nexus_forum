package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type PushSubscriptionRepository interface {
	Save(sub *model.PushSubscription) error
	DeleteByEndpoint(userID uint, endpoint string) error
	ListByUser(userID uint) ([]*model.PushSubscription, error)
}

type pushSubscriptionRepository struct {
	db *gorm.DB
}

func NewPushSubscriptionRepository(db *gorm.DB) PushSubscriptionRepository {
	return &pushSubscriptionRepository{db: db}
}

func (r *pushSubscriptionRepository) Save(sub *model.PushSubscription) error {
	var existing model.PushSubscription
	if err := r.db.Where("user_id = ? AND endpoint = ?", sub.UserID, sub.Endpoint).First(&existing).Error; err == nil {
		existing.P256DH = sub.P256DH
		existing.Auth = sub.Auth
		return r.db.Save(&existing).Error
	}
	return r.db.Create(sub).Error
}

func (r *pushSubscriptionRepository) DeleteByEndpoint(userID uint, endpoint string) error {
	return r.db.Where("user_id = ? AND endpoint = ?", userID, endpoint).Delete(&model.PushSubscription{}).Error
}

func (r *pushSubscriptionRepository) ListByUser(userID uint) ([]*model.PushSubscription, error) {
	var subs []*model.PushSubscription
	err := r.db.Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}
