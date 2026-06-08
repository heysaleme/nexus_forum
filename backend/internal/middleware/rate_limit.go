package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	limit   int
	window  time.Duration
}

// NewRateLimiter returns middleware that limits requests per IP per route.
func NewRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	rl := &rateLimiter{
		buckets: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
	return func(c *gin.Context) {
		key := c.ClientIP() + ":" + c.FullPath()
		if !rl.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func (rl *rateLimiter) allow(key string) bool {
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
