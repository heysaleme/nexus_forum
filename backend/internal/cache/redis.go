package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
	ok  bool
}

func New(redisURL string) *Client {
	if redisURL == "" {
		return &Client{ok: false}
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Warn("redis url parse failed", "error", err)
		return &Client{ok: false}
	}
	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("redis unavailable, using in-memory fallbacks", "error", err)
		return &Client{ok: false}
	}
	slog.Info("redis connected")
	return &Client{rdb: rdb, ok: true}
}

func (c *Client) Enabled() bool { return c != nil && c.ok }

func (c *Client) Get(ctx context.Context, key string, dest interface{}) bool {
	if !c.Enabled() {
		return false
	}
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(val), dest) == nil
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if !c.Enabled() {
		return
	}
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, key, b, ttl).Err()
}

func (c *Client) Delete(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}
	_ = c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Incr(ctx context.Context, key string) int64 {
	if !c.Enabled() {
		return 0
	}
	n, _ := c.rdb.Incr(ctx, key).Result()
	return n
}

func (c *Client) AllowRate(ctx context.Context, key string, limit int, window time.Duration) bool {
	if !c.Enabled() {
		return true
	}
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		// Fail closed when Redis is configured but unavailable.
		slog.Warn("redis rate limit error", "error", err, "key", key)
		return false
	}
	if n == 1 {
		_ = c.rdb.Expire(ctx, key, window).Err()
	}
	return n <= int64(limit)
}
