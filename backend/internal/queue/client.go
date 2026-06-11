package queue

import (
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client publishes notification events and runs a background consumer.
type Client struct {
	conn *amqp.Connection
	pub  *amqp.Channel
	ok   bool
}

// NewClient connects to RabbitMQ, declares the notification queue, and starts a consumer.
func NewClient(amqpURL string) *Client {
	if amqpURL == "" {
		return &Client{ok: false}
	}
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		slog.Warn("rabbitmq unavailable", "error", err)
		return &Client{ok: false}
	}
	pub, err := conn.Channel()
	if err != nil {
		slog.Warn("rabbitmq publish channel failed", "error", err)
		_ = conn.Close()
		return &Client{ok: false}
	}
	if err := declareQueue(pub); err != nil {
		slog.Warn("rabbitmq queue declare failed", "error", err)
		_ = pub.Close()
		_ = conn.Close()
		return &Client{ok: false}
	}

	c := &Client{conn: conn, pub: pub, ok: true}
	slog.Info("rabbitmq connected", "queue", notificationQueue)
	go c.runConsumer()
	return c
}

func declareQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(notificationQueue, true, false, false, false, nil)
	return err
}

func (c *Client) Enabled() bool { return c != nil && c.ok }

func (c *Client) PublishNotification(evt NotificationEvent) {
	if !c.Enabled() {
		return
	}
	body, err := json.Marshal(evt)
	if err != nil {
		return
	}
	if err := c.pub.Publish("", notificationQueue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); err != nil {
		slog.Warn("rabbitmq publish failed", "error", err, "user_id", evt.UserID, "type", evt.Type)
	}
}

func (c *Client) runConsumer() {
	if c.conn == nil {
		return
	}
	ch, err := c.conn.Channel()
	if err != nil {
		slog.Warn("rabbitmq consumer channel failed", "error", err)
		return
	}
	defer ch.Close()

	if err := declareQueue(ch); err != nil {
		slog.Warn("rabbitmq consumer queue declare failed", "error", err)
		return
	}

	if err := ch.Qos(10, 0, false); err != nil {
		slog.Warn("rabbitmq consumer qos failed", "error", err)
		return
	}

	deliveries, err := ch.Consume(notificationQueue, "nexus-api-consumer", false, false, false, false, nil)
	if err != nil {
		slog.Warn("rabbitmq consumer subscribe failed", "error", err)
		return
	}

	slog.Info("rabbitmq consumer started", "queue", notificationQueue)
	for msg := range deliveries {
		var evt NotificationEvent
		if err := json.Unmarshal(msg.Body, &evt); err != nil {
			slog.Warn("rabbitmq consumer invalid payload", "error", err)
			_ = msg.Nack(false, false)
			continue
		}
		slog.Info("rabbitmq notification consumed",
			"user_id", evt.UserID,
			"type", evt.Type,
			"title", evt.Data["title"],
		)
		_ = msg.Ack(false)
	}
}
