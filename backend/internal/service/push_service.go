package service

import (
	"fmt"
	"log/slog"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type PushService interface {
	Subscribe(userID uint, endpoint, p256dh, auth string) error
	Unsubscribe(userID uint, endpoint string) error
	SendToUser(user *model.User, notifType, title, body string) error
	SendToUserDetailed(user *model.User, notifType, title, body string) ([]PushDeliveryResult, error)
	SendTest(user *model.User) ([]PushDeliveryResult, error)
	HasSubscription(userID uint) (bool, error)
	PublicKey() string
}

type pushService struct {
	repo       repository.PushSubscriptionRepository
	publicKey  string
	privateKey string
	subject    string
}

func NewPushService(repo repository.PushSubscriptionRepository, publicKey, privateKey, subject string) PushService {
	return &pushService{repo: repo, publicKey: publicKey, privateKey: privateKey, subject: subject}
}

func (s *pushService) PublicKey() string { return s.publicKey }

func (s *pushService) Subscribe(userID uint, endpoint, p256dh, auth string) error {
	slog.Info("push: subscription saved", "user_id", userID, "endpoint_host", endpointHost(endpoint))
	return s.repo.Save(&model.PushSubscription{UserID: userID, Endpoint: endpoint, P256DH: p256dh, Auth: auth})
}

func (s *pushService) Unsubscribe(userID uint, endpoint string) error {
	return s.repo.DeleteByEndpoint(userID, endpoint)
}

func (s *pushService) shouldSend(user *model.User, notifType string) bool {
	if user == nil || s.publicKey == "" || s.privateKey == "" {
		return false
	}
	switch notifType {
	case "test":
		return true
	case "comment":
		return user.PushNotifyComments
	case "reply":
		return user.PushNotifyReplies
	case "mention":
		return user.PushNotifyMentions
	case "follow", "follow_request", "follow_accept":
		return user.PushNotifyFollowers
	case "message":
		return user.PushNotifyMessages
	default:
		return user.PushNotifyModeration
	}
}

func (s *pushService) HasSubscription(userID uint) (bool, error) {
	subs, err := s.repo.ListByUser(userID)
	if err != nil {
		return false, err
	}
	return len(subs) > 0, nil
}

func (s *pushService) SendTest(user *model.User) ([]PushDeliveryResult, error) {
	return s.sendToUserDetailed(user, "test", "Nexus Forum", "Тестовое push-уведомление")
}

func (s *pushService) SendToUserDetailed(user *model.User, notifType, title, body string) ([]PushDeliveryResult, error) {
	return s.sendToUserDetailed(user, notifType, title, body)
}

func (s *pushService) SendToUser(user *model.User, notifType, title, body string) error {
	results, err := s.sendToUserDetailed(user, notifType, title, body)
	if err != nil {
		return err
	}
	delivered := 0
	for _, r := range results {
		if r.Delivered {
			delivered++
		}
	}
	if delivered == 0 && len(results) > 0 {
		return fmt.Errorf("push not delivered to any subscription (see server logs)")
	}
	return nil
}

func AnyDelivered(results []PushDeliveryResult) bool {
	for _, r := range results {
		if r.Delivered {
			return true
		}
	}
	return false
}
