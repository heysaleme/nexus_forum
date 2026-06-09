package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"nexus-forum-backend/internal/model"
)

// PushDeliveryResult captures the outcome of a single web-push attempt.
type PushDeliveryResult struct {
	Endpoint       string `json:"endpoint"`
	HTTPStatusCode int    `json:"http_status_code"`
	ResponseBody   string `json:"response_body"`
	Error          string `json:"error,omitempty"`
	Delivered      bool   `json:"delivered"`
}

func endpointHost(endpoint string) string {
	if i := strings.Index(endpoint, "://"); i >= 0 {
		rest := endpoint[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return endpoint
}

func isPushDelivered(statusCode int, err error) bool {
	if err != nil {
		return false
	}
	// Apple Push (web.push.apple.com) and FCM return 201 Created on success.
	return statusCode >= 200 && statusCode < 300
}

func (s *pushService) sendPayload(sub *model.PushSubscription, payload []byte) PushDeliveryResult {
	result := PushDeliveryResult{
		Endpoint: sub.Endpoint,
	}

	slog.Info("push: before SendNotification",
		"endpoint_host", endpointHost(sub.Endpoint),
		"endpoint_prefix", truncate(sub.Endpoint, 64),
		"payload_bytes", len(payload),
	)

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
		result.Error = err.Error()
		slog.Error("push: webpush.SendNotification error",
			"endpoint_host", endpointHost(sub.Endpoint),
			"error", err,
		)
		return result
	}

	if httpResp != nil {
		result.HTTPStatusCode = httpResp.StatusCode
		body, readErr := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if readErr != nil {
			result.ResponseBody = fmt.Sprintf("(read error: %v)", readErr)
		} else {
			result.ResponseBody = strings.TrimSpace(string(body))
		}
	}

	result.Delivered = isPushDelivered(result.HTTPStatusCode, err)

	slog.Info("push: SendNotification result",
		"endpoint_host", endpointHost(sub.Endpoint),
		"http_status", result.HTTPStatusCode,
		"response_body", truncate(result.ResponseBody, 256),
		"delivered", result.Delivered,
	)

	return result
}

func (s *pushService) sendToUserDetailed(user *model.User, notifType, title, body string) ([]PushDeliveryResult, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	if !s.shouldSend(user, notifType) {
		slog.Info("push: skipped by user preference", "user_id", user.ID, "type", notifType)
		return nil, fmt.Errorf("push disabled for notification type %q", notifType)
	}
	if s.publicKey == "" || s.privateKey == "" {
		return nil, fmt.Errorf("VAPID keys not configured")
	}

	subs, err := s.repo.ListByUser(user.ID)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("no push subscriptions for user %d", user.ID)
	}

	payload, _ := json.Marshal(map[string]string{
		"title": title,
		"body":  body,
		"type":  notifType,
		"url":   "/notifications",
	})

	results := make([]PushDeliveryResult, 0, len(subs))
	for _, sub := range subs {
		results = append(results, s.sendPayload(sub, payload))
	}
	return results, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
