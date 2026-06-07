package service

import (
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type NotificationService interface {
	GetByUser(userID uint) ([]*model.Notification, error)
	MarkAllRead(userID uint) error
	MarkRead(id, userID uint) error
}

type notificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) GetByUser(userID uint) ([]*model.Notification, error) {
	return s.repo.GetByUser(userID)
}

func (s *notificationService) MarkAllRead(userID uint) error {
	return s.repo.MarkAllRead(userID)
}

func (s *notificationService) MarkRead(id, userID uint) error {
	return s.repo.MarkRead(id, userID)
}
