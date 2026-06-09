package queue

import (
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const notificationQueue = "nexus.notifications"

type NotificationEvent struct {
	UserID uint                   `json:"user_id"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
}

type Publisher struct {
	ch  *amqp.Channel
	ok  bool
}

func NewPublisher(amqpURL string) *Publisher {
	if amqpURL == "" {
		return &Publisher{ok: false}
	}
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		slog.Warn("rabbitmq unavailable", "error", err)
		return &Publisher{ok: false}
	}
	ch, err := conn.Channel()
	if err != nil {
		slog.Warn("rabbitmq channel failed", "error", err)
		return &Publisher{ok: false}
	}
	_, err = ch.QueueDeclare(notificationQueue, true, false, false, false, nil)
	if err != nil {
		slog.Warn("rabbitmq queue declare failed", "error", err)
		return &Publisher{ok: false}
	}
	slog.Info("rabbitmq connected", "queue", notificationQueue)
	return &Publisher{ch: ch, ok: true}
}

func (p *Publisher) Enabled() bool { return p != nil && p.ok }

func (p *Publisher) PublishNotification(evt NotificationEvent) {
	if !p.Enabled() {
		return
	}
	body, err := json.Marshal(evt)
	if err != nil {
		return
	}
	_ = p.ch.Publish("", notificationQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
