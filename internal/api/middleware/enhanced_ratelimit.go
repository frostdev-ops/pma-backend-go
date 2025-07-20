package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// EnhancedRateLimitConfig contains advanced rate limiting configuration
type EnhancedRateLimitConfig struct {
	// Global settings
	EnableRateLimit bool          `json:"enable_rate_limit"`
	GlobalRequests  int           `json:"global_requests"`
	GlobalWindow    time.Duration `json:"global_window"`
	GlobalBurst     int           `json:"global_burst"`

	// Per-endpoint settings
	EnablePerEndpoint    bool                      `json:"enable_per_endpoint"`
	EndpointLimits       map[string]*EndpointLimit `json:"endpoint_limits"`
	DefaultEndpointLimit *EndpointLimit            `json:"default_endpoint_limit"`

	// Per-user settings
	EnablePerUser    bool                  `json:"enable_per_user"`
	UserLimits       map[string]*UserLimit `json:"user_limits"`
	DefaultUserLimit *UserLimit            `json:"default_user_limit"`

	// Adaptive settings
	EnableAdaptive     bool          `json:"enable_adaptive"`
	AdaptiveThreshold  float64       `json:"adaptive_threshold"`
	AdaptiveMultiplier float64       `json:"adaptive_multiplier"`
	AdaptiveWindow     time.Duration `json:"adaptive_window"`

	// Advanced features
	EnableDistributed     bool     `json:"enable_distributed"`
	EnableWhitelisting    bool     `json:"enable_whitelisting"`
	WhitelistedIPs        []string `json:"whitelisted_ips"`
	WhitelistedUserAgents []string `json:"whitelisted_user_agents"`

	// Punishment settings
	EnablePunishment     bool          `json:"enable_punishment"`
	PunishmentMultiplier float64       `json:"punishment_multiplier"`
	PunishmentDuration   time.Duration `json:"punishment_duration"`
	MaxViolations        int           `json:"max_violations"`

	// Sliding window settings
	EnableSlidingWindow bool `json:"enable_sliding_window"`
	SlidingWindowSize   int  `json:"sliding_window_size"`

	// Headers and responses
	IncludeHeaders     bool   `json:"include_headers"`
	CustomErrorMessage string `json:"custom_error_message"`

	// Cleanup
	CleanupInterval time.Duration `json:"cleanup_interval"`
	VisitorTimeout  time.Duration `json:"visitor_timeout"`
}

// EndpointLimit defines rate limits for specific endpoints
type EndpointLimit struct {
	Path     string        `json:"path"`
	Method   string        `json:"method"`
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst    int           `json:"burst"`
	Enabled  bool          `json:"enabled"`
}

// UserLimit defines rate limits for specific users
type UserLimit struct {
	UserID   string        `json:"user_id"`
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst    int           `json:"burst"`
	Enabled  bool          `json:"enabled"`
}

// EnhancedRateLimiter implements advanced rate limiting with multiple strategies
type EnhancedRateLimiter struct {
	config           *EnhancedRateLimitConfig
	logger           *logrus.Logger
	visitors         map[string]*EnhancedVisitor
	endpointVisitors map[string]*EnhancedVisitor
	userVisitors     map[string]*EnhancedVisitor
	violations       map[string]*ViolationRecord
	adaptiveMetrics  *AdaptiveMetrics
	whitelistCache   map[string]bool
	mu               sync.RWMutex
	cleanupTicker    *time.Ticker
	stopCleanup      chan bool
	requestMetrics   *RequestMetrics
}

// EnhancedVisitor tracks visitor information with advanced features
type EnhancedVisitor struct {
	Key              string               `json:"key"`
	TokenBucket      *EnhancedTokenBucket `json:"token_bucket"`
	SlidingWindow    *SlidingWindow       `json:"sliding_window"`
	LastSeen         time.Time            `json:"last_seen"`
	RequestCount     int64                `json:"request_count"`
	ViolationCount   int                  `json:"violation_count"`
	IsPunished       bool                 `json:"is_punished"`
	PunishmentExpiry time.Time            `json:"punishment_expiry"`
	UserAgent        string               `json:"user_agent"`
	Fingerprint      string               `json:"fingerprint"`
}

// EnhancedTokenBucket implements an enhanced token bucket algorithm
type EnhancedTokenBucket struct {
	Tokens         float64   `json:"tokens"`
	Capacity       float64   `json:"capacity"`
	RefillRate     float64   `json:"refill_rate"`
	LastRefill     time.Time `json:"last_refill"`
	BurstAllowance float64   `json:"burst_allowance"`
	AdaptiveRate   float64   `json:"adaptive_rate"`
	mu             sync.Mutex
}

// SlidingWindow implements a sliding window rate limiter
type SlidingWindow struct {
	Requests    []time.Time   `json:"requests"`
	WindowSize  time.Duration `json:"window_size"`
	MaxRequests int           `json:"max_requests"`
	mu          sync.Mutex
}

// ViolationRecord tracks rate limit violations
type ViolationRecord struct {
	IP              string    `json:"ip"`
	Count           int       `json:"count"`
	FirstViolation  time.Time `json:"first_violation"`
	LastViolation   time.Time `json:"last_violation"`
	PunishmentLevel int       `json:"punishment_level"`
	Endpoint        string    `json:"endpoint"`
}

// AdaptiveMetrics tracks metrics for adaptive rate limiting
type AdaptiveMetrics struct {
	TotalRequests  int64     `json:"total_requests"`
	ErrorRate      float64   `json:"error_rate"`
	AverageLatency float64   `json:"average_latency"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	LastUpdate     time.Time `json:"last_update"`
	mu             sync.RWMutex
}

// RequestMetrics tracks request statistics
type RequestMetrics struct {
	RequestsAllowed     int64 `json:"requests_allowed"`
	RequestsBlocked     int64 `json:"requests_blocked"`
	TotalRequests       int64 `json:"total_requests"`
	UniqueVisitors      int64 `json:"unique_visitors"`
	ViolationsDetected  int64 `json:"violations_detected"`
	AdaptiveAdjustments int64 `json:"adaptive_adjustments"`
	mu                  sync.RWMutex
}

// NewEnhancedRateLimiter creates a new enhanced rate limiter
func NewEnhancedRateLimiter(config *EnhancedRateLimitConfig, logger *logrus.Logger) *EnhancedRateLimiter {
	if config == nil {
		config = DefaultEnhancedRateLimitConfig()
	}

	limiter := &EnhancedRateLimiter{
		config:           config,
		logger:           logger,
		visitors:         make(map[string]*EnhancedVisitor),
		endpointVisitors: make(map[string]*EnhancedVisitor),
		userVisitors:     make(map[string]*EnhancedVisitor),
		violations:       make(map[string]*ViolationRecord),
		whitelistCache:   make(map[string]bool),
		stopCleanup:      make(chan bool),
		adaptiveMetrics:  &AdaptiveMetrics{},
		requestMetrics:   &RequestMetrics{},
	}

	// Initialize whitelist cache
	for _, ip := range config.WhitelistedIPs {
		limiter.whitelistCache[ip] = true
	}

	// Start cleanup routine
	if config.CleanupInterval > 0 {
		limiter.startCleanup()
	}

	return limiter
}

// DefaultEnhancedRateLimitConfig returns default enhanced rate limit configuration
func DefaultEnhancedRateLimitConfig() *EnhancedRateLimitConfig {
	return &EnhancedRateLimitConfig{
		EnableRateLimit:      true,
		GlobalRequests:       100,
		GlobalWindow:         time.Minute,
		GlobalBurst:          20,
		EnablePerEndpoint:    true,
		EnablePerUser:        false,
		EnableAdaptive:       true,
		AdaptiveThreshold:    0.8,
		AdaptiveMultiplier:   0.5,
		AdaptiveWindow:       time.Minute * 5,
		EnableDistributed:    false,
		EnableWhitelisting:   true,
		WhitelistedIPs:       []string{"127.0.0.1", "::1"},
		EnablePunishment:     true,
		PunishmentMultiplier: 2.0,
		PunishmentDuration:   time.Minute * 15,
		MaxViolations:        5,
		EnableSlidingWindow:  true,
		SlidingWindowSize:    100,
		IncludeHeaders:       true,
		CustomErrorMessage:   "Rate limit exceeded. Please try again later.",
		CleanupInterval:      time.Minute * 5,
		VisitorTimeout:       time.Hour,
		DefaultEndpointLimit: &EndpointLimit{
			Requests: 50,
			Window:   time.Minute,
			Burst:    10,
			Enabled:  true,
		},
		EndpointLimits: map[string]*EndpointLimit{
			"POST:/api/auth/*": {
				Path:     "/api/auth/*",
				Method:   "POST",
				Requests: 5,
				Window:   time.Minute,
				Burst:    2,
				Enabled:  true,
			},
			"GET:/api/entities": {
				Path:     "/api/entities",
				Method:   "GET",
				Requests: 200,
				Window:   time.Minute,
				Burst:    50,
				Enabled:  true,
			},
		},
	}
}

// RateLimitMiddleware returns the enhanced rate limiting middleware
func (erl *EnhancedRateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !erl.config.EnableRateLimit {
			c.Next()
			return
		}

		startTime := time.Now()
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		endpoint := fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())

		// Check whitelist
		if erl.isWhitelisted(clientIP, userAgent) {
			c.Next()
			return
		}

		// Generate visitor key
		visitorKey := erl.generateVisitorKey(clientIP, userAgent)

		// Check if client is currently punished
		if erl.isPunished(visitorKey) {
			erl.recordBlocked()
			erl.addRateLimitHeaders(c, 0, 0, time.Now().Add(erl.config.PunishmentDuration))
			utils.SendError(c, http.StatusTooManyRequests, erl.config.CustomErrorMessage)
			c.Abort()
			return
		}

		// Apply adaptive adjustments
		if erl.config.EnableAdaptive {
			erl.applyAdaptiveAdjustments()
		}

		// Check global rate limit
		if !erl.checkGlobalLimit(visitorKey, clientIP, userAgent) {
			erl.recordViolation(clientIP, endpoint)
			erl.recordBlocked()
			erl.addRateLimitHeaders(c, 0, 0, time.Now().Add(erl.config.GlobalWindow))
			utils.SendError(c, http.StatusTooManyRequests, erl.config.CustomErrorMessage)
			c.Abort()
			return
		}

		// Check endpoint-specific limit
		if erl.config.EnablePerEndpoint {
			if !erl.checkEndpointLimit(endpoint, visitorKey, clientIP, userAgent) {
				erl.recordViolation(clientIP, endpoint)
				erl.recordBlocked()
				erl.addRateLimitHeaders(c, 0, 0, time.Now().Add(erl.config.DefaultEndpointLimit.Window))
				utils.SendError(c, http.StatusTooManyRequests, erl.config.CustomErrorMessage)
				c.Abort()
				return
			}
		}

		// Check user-specific limit (if authentication is available)
		if erl.config.EnablePerUser {
			if userID, exists := c.Get("user_id"); exists {
				if userIDStr, ok := userID.(string); ok {
					if !erl.checkUserLimit(userIDStr, visitorKey) {
						erl.recordViolation(clientIP, endpoint)
						erl.recordBlocked()
						erl.addRateLimitHeaders(c, 0, 0, time.Now().Add(erl.config.DefaultUserLimit.Window))
						utils.SendError(c, http.StatusTooManyRequests, erl.config.CustomErrorMessage)
						c.Abort()
						return
					}
				}
			}
		}

		// Request allowed
		erl.recordAllowed()
		erl.updateAdaptiveMetrics(time.Since(startTime))

		// Add rate limit headers
		visitor := erl.getOrCreateVisitor(visitorKey, clientIP, userAgent)
		remaining := int(visitor.TokenBucket.Tokens)
		resetTime := visitor.TokenBucket.LastRefill.Add(erl.config.GlobalWindow)
		erl.addRateLimitHeaders(c, remaining, erl.config.GlobalRequests, resetTime)

		c.Next()
	}
}

// checkGlobalLimit checks the global rate limit
func (erl *EnhancedRateLimiter) checkGlobalLimit(visitorKey, clientIP, userAgent string) bool {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	visitor := erl.getOrCreateVisitor(visitorKey, clientIP, userAgent)

	if erl.config.EnableSlidingWindow {
		return visitor.SlidingWindow.Allow()
	}

	return visitor.TokenBucket.Consume(1.0)
}

// checkEndpointLimit checks endpoint-specific rate limits
func (erl *EnhancedRateLimiter) checkEndpointLimit(endpoint, visitorKey, clientIP, userAgent string) bool {
	limit := erl.getEndpointLimit(endpoint)
	if limit == nil || !limit.Enabled {
		return true
	}

	erl.mu.Lock()
	defer erl.mu.Unlock()

	endpointKey := fmt.Sprintf("%s:%s", visitorKey, endpoint)
	visitor, exists := erl.endpointVisitors[endpointKey]
	if !exists {
		visitor = erl.createVisitorForLimit(endpointKey, clientIP, userAgent, limit.Requests, limit.Window, limit.Burst)
		erl.endpointVisitors[endpointKey] = visitor
	}

	visitor.LastSeen = time.Now()

	if erl.config.EnableSlidingWindow {
		return visitor.SlidingWindow.Allow()
	}

	return visitor.TokenBucket.Consume(1.0)
}

// checkUserLimit checks user-specific rate limits
func (erl *EnhancedRateLimiter) checkUserLimit(userID, visitorKey string) bool {
	limit := erl.getUserLimit(userID)
	if limit == nil || !limit.Enabled {
		return true
	}

	erl.mu.Lock()
	defer erl.mu.Unlock()

	userKey := fmt.Sprintf("user:%s", userID)
	visitor, exists := erl.userVisitors[userKey]
	if !exists {
		visitor = erl.createVisitorForLimit(userKey, "", "", limit.Requests, limit.Window, limit.Burst)
		erl.userVisitors[userKey] = visitor
	}

	visitor.LastSeen = time.Now()

	if erl.config.EnableSlidingWindow {
		return visitor.SlidingWindow.Allow()
	}

	return visitor.TokenBucket.Consume(1.0)
}

// getOrCreateVisitor gets or creates a visitor
func (erl *EnhancedRateLimiter) getOrCreateVisitor(visitorKey, clientIP, userAgent string) *EnhancedVisitor {
	visitor, exists := erl.visitors[visitorKey]
	if !exists {
		visitor = erl.createVisitorForLimit(visitorKey, clientIP, userAgent,
			erl.config.GlobalRequests, erl.config.GlobalWindow, erl.config.GlobalBurst)
		erl.visitors[visitorKey] = visitor
	}

	visitor.LastSeen = time.Now()
	visitor.RequestCount++

	return visitor
}

// createVisitorForLimit creates a new visitor with specific limits
func (erl *EnhancedRateLimiter) createVisitorForLimit(key, clientIP, userAgent string, requests int, window time.Duration, burst int) *EnhancedVisitor {
	now := time.Now()

	visitor := &EnhancedVisitor{
		Key:          key,
		LastSeen:     now,
		RequestCount: 0,
		UserAgent:    userAgent,
		Fingerprint:  erl.generateFingerprint(clientIP, userAgent),
		TokenBucket: &EnhancedTokenBucket{
			Tokens:         float64(burst),
			Capacity:       float64(burst),
			RefillRate:     float64(requests) / window.Seconds(),
			LastRefill:     now,
			BurstAllowance: float64(burst),
			AdaptiveRate:   1.0,
		},
	}

	if erl.config.EnableSlidingWindow {
		visitor.SlidingWindow = &SlidingWindow{
			Requests:    make([]time.Time, 0, erl.config.SlidingWindowSize),
			WindowSize:  window,
			MaxRequests: requests,
		}
	}

	return visitor
}

// Consume tokens from the enhanced token bucket
func (etb *EnhancedTokenBucket) Consume(tokens float64) bool {
	etb.mu.Lock()
	defer etb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(etb.LastRefill).Seconds()

	// Refill tokens based on adaptive rate
	tokensToAdd := elapsed * etb.RefillRate * etb.AdaptiveRate
	etb.Tokens = minFloat64(etb.Capacity, etb.Tokens+tokensToAdd)
	etb.LastRefill = now

	if etb.Tokens >= tokens {
		etb.Tokens -= tokens
		return true
	}

	return false
}

// Allow checks if a request is allowed in the sliding window
func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.WindowSize)

	// Remove old requests
	validRequests := make([]time.Time, 0, len(sw.Requests))
	for _, reqTime := range sw.Requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	sw.Requests = validRequests

	// Check if we can allow another request
	if len(sw.Requests) < sw.MaxRequests {
		sw.Requests = append(sw.Requests, now)
		return true
	}

	return false
}

// Helper functions
func (erl *EnhancedRateLimiter) generateVisitorKey(clientIP, userAgent string) string {
	return fmt.Sprintf("%s:%s", clientIP, erl.generateFingerprint(clientIP, userAgent))
}

func (erl *EnhancedRateLimiter) generateFingerprint(clientIP, userAgent string) string {
	data := fmt.Sprintf("%s|%s", clientIP, userAgent)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // First 8 bytes for shorter fingerprint
}

func (erl *EnhancedRateLimiter) isWhitelisted(clientIP, userAgent string) bool {
	erl.mu.RLock()
	defer erl.mu.RUnlock()

	if !erl.config.EnableWhitelisting {
		return false
	}

	// Check IP whitelist
	if erl.whitelistCache[clientIP] {
		return true
	}

	// Check user agent whitelist
	for _, whitelistedUA := range erl.config.WhitelistedUserAgents {
		if strings.Contains(userAgent, whitelistedUA) {
			return true
		}
	}

	return false
}

func (erl *EnhancedRateLimiter) isPunished(visitorKey string) bool {
	erl.mu.RLock()
	defer erl.mu.RUnlock()

	if !erl.config.EnablePunishment {
		return false
	}

	visitor, exists := erl.visitors[visitorKey]
	if !exists {
		return false
	}

	return visitor.IsPunished && time.Now().Before(visitor.PunishmentExpiry)
}

func (erl *EnhancedRateLimiter) recordViolation(clientIP, endpoint string) {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	violation, exists := erl.violations[clientIP]
	if !exists {
		violation = &ViolationRecord{
			IP:             clientIP,
			Count:          0,
			FirstViolation: time.Now(),
			Endpoint:       endpoint,
		}
		erl.violations[clientIP] = violation
	}

	violation.Count++
	violation.LastViolation = time.Now()
	violation.Endpoint = endpoint

	// Apply punishment if threshold exceeded
	if erl.config.EnablePunishment && violation.Count >= erl.config.MaxViolations {
		erl.applyPunishment(clientIP, violation)
	}

	erl.requestMetrics.mu.Lock()
	erl.requestMetrics.ViolationsDetected++
	erl.requestMetrics.mu.Unlock()
}

func (erl *EnhancedRateLimiter) applyPunishment(clientIP string, violation *ViolationRecord) {
	// Find visitor and apply punishment
	for key, visitor := range erl.visitors {
		if strings.Contains(key, clientIP) {
			visitor.IsPunished = true
			visitor.ViolationCount = violation.Count

			// Increase punishment duration based on violation level
			punishmentDuration := time.Duration(float64(erl.config.PunishmentDuration) *
				(erl.config.PunishmentMultiplier * float64(violation.PunishmentLevel+1)))
			visitor.PunishmentExpiry = time.Now().Add(punishmentDuration)

			violation.PunishmentLevel++

			erl.logger.Warnf("Applied punishment to IP %s for %v (level %d)",
				clientIP, punishmentDuration, violation.PunishmentLevel)
			break
		}
	}
}

func (erl *EnhancedRateLimiter) getEndpointLimit(endpoint string) *EndpointLimit {
	// Direct match
	if limit, exists := erl.config.EndpointLimits[endpoint]; exists {
		return limit
	}

	// Pattern matching
	for pattern, limit := range erl.config.EndpointLimits {
		if matchEndpointPattern(pattern, endpoint) {
			return limit
		}
	}

	return erl.config.DefaultEndpointLimit
}

func (erl *EnhancedRateLimiter) getUserLimit(userID string) *UserLimit {
	if limit, exists := erl.config.UserLimits[userID]; exists {
		return limit
	}
	return erl.config.DefaultUserLimit
}

func matchEndpointPattern(pattern, endpoint string) bool {
	// Simple wildcard matching
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(endpoint, parts[0]) && strings.HasSuffix(endpoint, parts[1])
		}
	}
	return pattern == endpoint
}

func (erl *EnhancedRateLimiter) applyAdaptiveAdjustments() {
	erl.adaptiveMetrics.mu.RLock()
	errorRate := erl.adaptiveMetrics.ErrorRate
	avgLatency := erl.adaptiveMetrics.AverageLatency
	erl.adaptiveMetrics.mu.RUnlock()

	// Apply adaptive rate adjustments based on system health
	adaptiveMultiplier := 1.0

	if errorRate > erl.config.AdaptiveThreshold {
		// High error rate - reduce limits
		adaptiveMultiplier = erl.config.AdaptiveMultiplier
	} else if avgLatency > 1000 { // 1 second
		// High latency - reduce limits
		adaptiveMultiplier = 0.8
	}

	// Apply adjustments to all active visitors
	erl.mu.Lock()
	for _, visitor := range erl.visitors {
		visitor.TokenBucket.AdaptiveRate = adaptiveMultiplier
	}
	erl.mu.Unlock()

	if adaptiveMultiplier != 1.0 {
		erl.requestMetrics.mu.Lock()
		erl.requestMetrics.AdaptiveAdjustments++
		erl.requestMetrics.mu.Unlock()

		erl.logger.Infof("Applied adaptive rate limit adjustment: %.2f", adaptiveMultiplier)
	}
}

func (erl *EnhancedRateLimiter) updateAdaptiveMetrics(latency time.Duration) {
	erl.adaptiveMetrics.mu.Lock()
	defer erl.adaptiveMetrics.mu.Unlock()

	erl.adaptiveMetrics.TotalRequests++
	erl.adaptiveMetrics.AverageLatency = (erl.adaptiveMetrics.AverageLatency + latency.Seconds()) / 2
	erl.adaptiveMetrics.LastUpdate = time.Now()
}

func (erl *EnhancedRateLimiter) addRateLimitHeaders(c *gin.Context, remaining, limit int, resetTime time.Time) {
	if !erl.config.IncludeHeaders {
		return
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	c.Header("X-RateLimit-Reset-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
}

func (erl *EnhancedRateLimiter) recordAllowed() {
	erl.requestMetrics.mu.Lock()
	defer erl.requestMetrics.mu.Unlock()

	erl.requestMetrics.RequestsAllowed++
	erl.requestMetrics.TotalRequests++
}

func (erl *EnhancedRateLimiter) recordBlocked() {
	erl.requestMetrics.mu.Lock()
	defer erl.requestMetrics.mu.Unlock()

	erl.requestMetrics.RequestsBlocked++
	erl.requestMetrics.TotalRequests++
}

// startCleanup starts the cleanup routine
func (erl *EnhancedRateLimiter) startCleanup() {
	erl.cleanupTicker = time.NewTicker(erl.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-erl.cleanupTicker.C:
				erl.cleanup()
			case <-erl.stopCleanup:
				erl.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanup removes old visitors and violations
func (erl *EnhancedRateLimiter) cleanup() {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-erl.config.VisitorTimeout)

	// Clean up visitors
	for key, visitor := range erl.visitors {
		if visitor.LastSeen.Before(cutoff) && !visitor.IsPunished {
			delete(erl.visitors, key)
		}
	}

	// Clean up endpoint visitors
	for key, visitor := range erl.endpointVisitors {
		if visitor.LastSeen.Before(cutoff) {
			delete(erl.endpointVisitors, key)
		}
	}

	// Clean up user visitors
	for key, visitor := range erl.userVisitors {
		if visitor.LastSeen.Before(cutoff) {
			delete(erl.userVisitors, key)
		}
	}

	// Clean up old violations
	violationCutoff := now.Add(-erl.config.PunishmentDuration * 2)
	for ip, violation := range erl.violations {
		if violation.LastViolation.Before(violationCutoff) {
			delete(erl.violations, ip)
		}
	}
}

// GetMetrics returns current rate limiting metrics
func (erl *EnhancedRateLimiter) GetMetrics() *RequestMetrics {
	erl.requestMetrics.mu.RLock()
	defer erl.requestMetrics.mu.RUnlock()

	erl.mu.RLock()
	uniqueVisitors := int64(len(erl.visitors))
	erl.mu.RUnlock()

	return &RequestMetrics{
		RequestsAllowed:     erl.requestMetrics.RequestsAllowed,
		RequestsBlocked:     erl.requestMetrics.RequestsBlocked,
		TotalRequests:       erl.requestMetrics.TotalRequests,
		UniqueVisitors:      uniqueVisitors,
		ViolationsDetected:  erl.requestMetrics.ViolationsDetected,
		AdaptiveAdjustments: erl.requestMetrics.AdaptiveAdjustments,
	}
}

// GetTopViolators returns the top IP addresses with most violations
func (erl *EnhancedRateLimiter) GetTopViolators(limit int) []*ViolationRecord {
	erl.mu.RLock()
	defer erl.mu.RUnlock()

	violations := make([]*ViolationRecord, 0, len(erl.violations))
	for _, violation := range erl.violations {
		violations = append(violations, violation)
	}

	// Sort by violation count
	sort.Slice(violations, func(i, j int) bool {
		return violations[i].Count > violations[j].Count
	})

	if limit > 0 && limit < len(violations) {
		violations = violations[:limit]
	}

	return violations
}

// Stop stops the rate limiter cleanup routine
func (erl *EnhancedRateLimiter) Stop() {
	if erl.stopCleanup != nil {
		close(erl.stopCleanup)
	}
}

// BlockIP manually blocks an IP address
func (erl *EnhancedRateLimiter) BlockIP(ip string) {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	// Find and punish all visitors from this IP
	for key, visitor := range erl.visitors {
		if strings.Contains(key, ip) {
			visitor.IsPunished = true
			visitor.PunishmentExpiry = time.Now().Add(time.Hour * 24) // 24 hour block
		}
	}

	erl.logger.Infof("Manually blocked IP: %s", ip)
}

// UnblockIP manually unblocks an IP address
func (erl *EnhancedRateLimiter) UnblockIP(ip string) {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	// Unblock all visitors from this IP
	for key, visitor := range erl.visitors {
		if strings.Contains(key, ip) {
			visitor.IsPunished = false
			visitor.PunishmentExpiry = time.Time{}
			visitor.ViolationCount = 0
		}
	}

	// Remove violations
	delete(erl.violations, ip)

	erl.logger.Infof("Manually unblocked IP: %s", ip)
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
