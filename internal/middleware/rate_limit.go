package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Rate Limiter (Token Bucket) ──────────────────────────────────

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// RateLimiter implements a token-bucket rate limiter per client IP.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64       // tokens per second
	burst    int           // max tokens (burst capacity)
	cleanTTL time.Duration // cleanup stale entries after
}

// NewRateLimiter creates a rate limiter.
//   - rps: requests per second allowed
//   - burst: max burst capacity (e.g. 10 means up to 10 requests in a burst)
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rps,
		burst:    burst,
		cleanTTL: 10 * time.Minute,
	}
	go rl.cleanup()
	return rl
}

// Allow checks whether a request from the given key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &bucket{
			tokens:    float64(rl.burst) - 1,
			lastCheck: now,
		}
		return true
	}

	// Add tokens based on elapsed time
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastCheck = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// cleanup removes stale buckets every 5 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.Sub(b.lastCheck) > rl.cleanTTL {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// ── Gin Middleware ────────────────────────────────────────────────

// RateLimitMiddleware creates a global rate limit middleware.
// For production: ~20 rps with burst of 40 per IP.
func RateLimitMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

// StrictRateLimitMiddleware creates a stricter rate limit (e.g. for auth endpoints).
// For login/register: ~5 rps with burst of 10 per IP.
func StrictRateLimitMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please wait before trying again.",
			})
			return
		}
		c.Next()
	}
}
