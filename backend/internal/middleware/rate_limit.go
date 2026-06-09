package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"nexus-forum-backend/internal/cache"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	limit   int
	window  time.Duration
	redis   *cache.Client
}

// NewRateLimiter returns middleware that limits requests per IP per route.
// Uses Redis when available, otherwise in-memory sliding window.
func NewRateLimiter(limit int, window time.Duration, redis *cache.Client) gin.HandlerFunc {
	rl := &rateLimiter{
		buckets: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
		redis:   redis,
	}
	return func(c *gin.Context) {
		key := "rl:" + c.ClientIP() + ":" + c.FullPath()
		if rl.redis != nil && rl.redis.Enabled() {
			if !rl.redis.AllowRate(context.Background(), key, limit, window) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
				return
			}
			c.Next()
			return
		}
		if !rl.allowMemory(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func (rl *rateLimiter) allowMemory(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)
	times := rl.buckets[key]

	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.buckets[key] = valid
		return false
	}

	rl.buckets[key] = append(valid, now)
	return true
}

// FeedCacheKey builds a redis cache key for feed responses.
func FeedCacheKey(sort string, limit int, viewerID uint) string {
	return fmt.Sprintf("feed:%s:%d:%d", sort, limit, viewerID)
}
