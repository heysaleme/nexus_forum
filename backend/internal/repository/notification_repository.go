package repository

import (
	"nexus-forum-backend/internal/model"

	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(notification *model.Notification) error
	GetByUser(userID uint) ([]*model.Notification, error)
	MarkAllRead(userID uint) error
	MarkRead(id, userID uint) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

var NotificationDispatcher func(userID uint, notif *model.Notification)

func (r *notificationRepository) Create(notification *model.Notification) error {
	err := r.db.Create(notification).Error
	if err == nil && NotificationDispatcher != nil {
		NotificationDispatcher(notification.UserID, notification)
	}
	return err
}

func (r *notificationRepository) GetByUser(userID uint) ([]*model.Notification, error) {
	var notifications []*model.Notification
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) MarkAllRead(userID uint) error {
	return r.db.Model(&model.Notification{}).Where("user_id = ?", userID).Update("is_read", true).Error
}

func (r *notificationRepository) MarkRead(id, userID uint) error {
	return r.db.Model(&model.Notification{}).Where("id = ? AND user_id = ?", id, userID).Update("is_read", true).Error
}
