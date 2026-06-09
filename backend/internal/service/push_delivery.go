package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	pushcfg "nexus-forum-backend/internal/push"
	"nexus-forum-backend/internal/model"
)

// PushDeliveryResult captures the outcome of a single web-push attempt.
type PushDeliveryResult struct {
	Provider       string `json:"provider"`
	Endpoint       string `json:"endpoint"`
	HTTPStatusCode int    `json:"http_status_code"`
	ResponseBody   string `json:"response_body"`
	Error          string `json:"error,omitempty"`
	Delivered      bool   `json:"delivered"`
}

func pushProvider(endpoint string) string {
	host := endpointHost(endpoint)
	switch {
	case strings.Contains(host, "web.push.apple.com"):
		return "apple"
	case strings.Contains(host, "fcm.googleapis.com"):
		return "fcm"
	default:
		return host
	}
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
		Provider: pushProvider(sub.Endpoint),
		Endpoint: sub.Endpoint,
	}

	host := endpointHost(sub.Endpoint)
	aud, _ := pushcfg.JWTAudience(sub.Endpoint)
	exp := pushcfg.JWTExpiration()
	jwtSubject := ""
	subscriber := ""
	if s.vapid != nil {
		jwtSubject = s.vapid.JWTSubject
		subscriber = s.vapid.Subscriber
	}

	slog.Info("push: VAPID JWT before SendNotification",
		"endpoint_host", host,
		"provider", result.Provider,
		"jwt_audience", aud,
		"jwt_subject", jwtSubject,
		"jwt_expiration", exp.Format(time.RFC3339),
		"subscriber_option", subscriber,
		"vapid_public_key_prefix", truncate(publicKeyPrefix(s.vapid), 16),
		"payload_bytes", len(payload),
	)

	httpResp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256DH,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		Subscriber:      subscriber,
		VAPIDPublicKey:  s.vapid.PublicKey,
		VAPIDPrivateKey: s.vapid.PrivateKey,
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
	if s.vapid == nil || s.vapid.PublicKey == "" || s.vapid.PrivateKey == "" {
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

func publicKeyPrefix(vapid *pushcfg.Config) string {
	if vapid == nil {
		return ""
	}
	return vapid.PublicKey
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
