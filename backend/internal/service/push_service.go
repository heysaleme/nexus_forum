package service

import (
	"encoding/json"
	"log/slog"

	webpush "github.com/SherClockHolmes/webpush-go"
	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type PushService interface {
	Subscribe(userID uint, endpoint, p256dh, auth string) error
	Unsubscribe(userID uint, endpoint string) error
	SendToUser(user *model.User, notifType, title, body string) error
	SendTest(user *model.User) error
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

func (s *pushService) SendTest(user *model.User) error {
	return s.SendToUser(user, "test", "Nexus Forum", "Тестовое push-уведомление")
}

func (s *pushService) SendToUser(user *model.User, notifType, title, body string) error {
	if !s.shouldSend(user, notifType) {
		return nil
	}
	subs, err := s.repo.ListByUser(user.ID)
	if err != nil || len(subs) == 0 {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body, "type": notifType})
	for _, sub := range subs {
		httpResp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256DH,
				Auth:   sub.Auth,
			},
		}, &webpush.Options{
			Subscriber:      s.subject,
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
			TTL:             60,
		})
		if err != nil {
			slog.Warn("web push failed", "user_id", user.ID, "error", err)
			continue
		}
		if httpResp != nil {
			_ = httpResp.Body.Close()
		}
	}
	return nil
}
