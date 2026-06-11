package queue

const notificationQueue = "nexus.notifications"

// NotificationEvent is published when an in-app notification is created.
type NotificationEvent struct {
	UserID uint                   `json:"user_id"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
}

// NewPublisher connects to RabbitMQ and starts the notification consumer.
// Kept for call-site compatibility; returns *Client.
func NewPublisher(amqpURL string) *Client {
	return NewClient(amqpURL)
}
