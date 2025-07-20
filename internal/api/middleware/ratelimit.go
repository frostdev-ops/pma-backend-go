package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int
	burst    int
	cleanup  time.Duration
}

type Visitor struct {
	limiter  *TokenBucket
	lastSeen time.Time
}

type TokenBucket struct {
	tokens     int
	capacity   int
	refillRate int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  time.Minute * 5,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// RateLimitMiddleware returns a rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.allow(ip) {
			utils.SendError(c, http.StatusTooManyRequests, "Rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	visitor, exists := rl.visitors[key]
	if !exists {
		visitor = &Visitor{
			limiter: &TokenBucket{
				tokens:     rl.burst,
				capacity:   rl.burst,
				refillRate: rl.rate,
				lastRefill: time.Now(),
			},
			lastSeen: time.Now(),
		}
		rl.visitors[key] = visitor
	}

	visitor.lastSeen = time.Now()
	return visitor.limiter.consume()
}

func (tb *TokenBucket) consume() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Refill tokens
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
	tb.tokens = minInt(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, visitor := range rl.visitors {
			if time.Since(visitor.lastSeen) > rl.cleanup {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
