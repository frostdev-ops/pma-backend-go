package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityConfig contains advanced security configuration
type SecurityConfig struct {
	// Rate limiting
	EnableRateLimit      bool          `json:"enable_rate_limit"`
	RateLimitRequests    int           `json:"rate_limit_requests"`
	RateLimitWindow      time.Duration `json:"rate_limit_window"`
	RateLimitBurst       int           `json:"rate_limit_burst"`
	RateLimitPerEndpoint bool          `json:"rate_limit_per_endpoint"`

	// IP filtering
	EnableIPFilter       bool     `json:"enable_ip_filter"`
	AllowedIPs           []string `json:"allowed_ips"`
	BlockedIPs           []string `json:"blocked_ips"`
	AllowPrivateNetworks bool     `json:"allow_private_networks"`

	// Request validation
	EnableRequestValidation bool  `json:"enable_request_validation"`
	MaxRequestSize          int64 `json:"max_request_size"`
	MaxHeaderSize           int   `json:"max_header_size"`
	MaxQueryParams          int   `json:"max_query_params"`

	// Security headers
	EnableSecurityHeaders bool   `json:"enable_security_headers"`
	CSPDirectives         string `json:"csp_directives"`

	// Attack prevention
	EnableSQLInjectionFilter     bool `json:"enable_sql_injection_filter"`
	EnableXSSFilter              bool `json:"enable_xss_filter"`
	EnableCommandInjectionFilter bool `json:"enable_command_injection_filter"`

	// Advanced features
	EnableFingerprinting     bool `json:"enable_fingerprinting"`
	EnableHoneypot           bool `json:"enable_honeypot"`
	EnableThreatIntelligence bool `json:"enable_threat_intelligence"`

	// Logging and alerting
	LogSecurityEvents bool `json:"log_security_events"`
	AlertOnThreats    bool `json:"alert_on_threats"`
}

// AdvancedSecurityMiddleware provides comprehensive API security
type AdvancedSecurityMiddleware struct {
	config             *SecurityConfig
	logger             *logrus.Logger
	ipFilter           *IPFilter
	requestValidator   *RequestValidator
	attackDetector     *AttackDetector
	threatIntelligence *ThreatIntelligence
	securityMetrics    *SecurityMetrics
	mu                 sync.RWMutex
}

// IPFilter handles IP-based access control
type IPFilter struct {
	allowedNetworks []*net.IPNet
	blockedIPs      map[string]bool
	allowPrivate    bool
	mu              sync.RWMutex
}

// RequestValidator validates request structure and content
type RequestValidator struct {
	maxRequestSize        int64
	maxHeaderSize         int
	maxQueryParams        int
	sqlInjectionRegex     []*regexp.Regexp
	xssRegex              []*regexp.Regexp
	commandInjectionRegex []*regexp.Regexp
}

// AttackDetector detects and prevents common attacks
type AttackDetector struct {
	suspiciousPatterns map[string]*regexp.Regexp
	attackCounts       map[string]*AttackCount
	mu                 sync.RWMutex
}

// AttackCount tracks attack attempts per IP
type AttackCount struct {
	SQLInjection     int       `json:"sql_injection"`
	XSS              int       `json:"xss"`
	CommandInjection int       `json:"command_injection"`
	PathTraversal    int       `json:"path_traversal"`
	LastAttempt      time.Time `json:"last_attempt"`
}

// ThreatIntelligence provides threat intelligence and geolocation
type ThreatIntelligence struct {
	knownBadIPs   map[string]*ThreatInfo
	suspiciousUAs []*regexp.Regexp
	geoBlocks     map[string]bool // country codes
	mu            sync.RWMutex
}

// ThreatInfo contains threat intelligence data
type ThreatInfo struct {
	ThreatType  string    `json:"threat_type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	ReportCount int       `json:"report_count"`
}

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	RequestsBlocked     int64 `json:"requests_blocked"`
	AttacksDetected     int64 `json:"attacks_detected"`
	SuspiciousRequests  int64 `json:"suspicious_requests"`
	RateLimitViolations int64 `json:"rate_limit_violations"`
	IPFilterViolations  int64 `json:"ip_filter_violations"`
	mu                  sync.RWMutex
}

// NewAdvancedSecurityMiddleware creates a new advanced security middleware
func NewAdvancedSecurityMiddleware(config *SecurityConfig, logger *logrus.Logger) *AdvancedSecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	middleware := &AdvancedSecurityMiddleware{
		config:          config,
		logger:          logger,
		securityMetrics: &SecurityMetrics{},
	}

	// Initialize components
	middleware.initializeIPFilter()
	middleware.initializeRequestValidator()
	middleware.initializeAttackDetector()
	middleware.initializeThreatIntelligence()

	return middleware
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableRateLimit:              true,
		RateLimitRequests:            100,
		RateLimitWindow:              time.Minute,
		RateLimitBurst:               20,
		RateLimitPerEndpoint:         true,
		EnableIPFilter:               true,
		AllowedIPs:                   []string{},
		BlockedIPs:                   []string{},
		AllowPrivateNetworks:         true,
		EnableRequestValidation:      true,
		MaxRequestSize:               10 * 1024 * 1024, // 10MB
		MaxHeaderSize:                8192,             // 8KB
		MaxQueryParams:               50,
		EnableSecurityHeaders:        true,
		CSPDirectives:                "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';",
		EnableSQLInjectionFilter:     true,
		EnableXSSFilter:              true,
		EnableCommandInjectionFilter: true,
		EnableFingerprinting:         false,
		EnableHoneypot:               false,
		EnableThreatIntelligence:     false,
		LogSecurityEvents:            true,
		AlertOnThreats:               true,
	}
}

// SecurityMiddleware returns the main security middleware function
func (asm *AdvancedSecurityMiddleware) SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// 1. IP Filtering
		if asm.config.EnableIPFilter {
			if blocked, reason := asm.ipFilter.IsBlocked(c.ClientIP()); blocked {
				asm.logSecurityEvent("ip_blocked", c, map[string]interface{}{
					"reason": reason,
					"ip":     c.ClientIP(),
				})
				asm.incrementMetric("ip_filter_violations")
				utils.SendError(c, http.StatusForbidden, "Access denied")
				c.Abort()
				return
			}
		}

		// 2. Request Validation
		if asm.config.EnableRequestValidation {
			if err := asm.requestValidator.ValidateRequest(c); err != nil {
				asm.logSecurityEvent("request_validation_failed", c, map[string]interface{}{
					"error": err.Error(),
				})
				asm.incrementMetric("requests_blocked")
				utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
				c.Abort()
				return
			}
		}

		// 3. Attack Detection
		if asm.detectAttacks(c) {
			asm.logSecurityEvent("attack_detected", c, nil)
			asm.incrementMetric("attacks_detected")
			utils.SendError(c, http.StatusForbidden, "Potential attack detected")
			c.Abort()
			return
		}

		// 4. Threat Intelligence
		if asm.config.EnableThreatIntelligence {
			if threat := asm.threatIntelligence.CheckThreat(c); threat != nil {
				asm.logSecurityEvent("threat_detected", c, map[string]interface{}{
					"threat_type": threat.ThreatType,
					"severity":    threat.Severity,
				})
				asm.incrementMetric("suspicious_requests")

				if threat.Severity == "high" || threat.Severity == "critical" {
					utils.SendError(c, http.StatusForbidden, "Access denied due to threat intelligence")
					c.Abort()
					return
				}
			}
		}

		// 5. Security Headers
		if asm.config.EnableSecurityHeaders {
			asm.addSecurityHeaders(c)
		}

		// Continue to next middleware
		c.Next()

		// Log successful request
		duration := time.Since(startTime)
		asm.logSecurityEvent("request_processed", c, map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"status":      c.Writer.Status(),
		})
	}
}

// initializeIPFilter sets up IP filtering
func (asm *AdvancedSecurityMiddleware) initializeIPFilter() {
	asm.ipFilter = &IPFilter{
		allowedNetworks: make([]*net.IPNet, 0),
		blockedIPs:      make(map[string]bool),
		allowPrivate:    asm.config.AllowPrivateNetworks,
	}

	// Parse allowed IP networks
	for _, ipStr := range asm.config.AllowedIPs {
		if strings.Contains(ipStr, "/") {
			_, network, err := net.ParseCIDR(ipStr)
			if err != nil {
				asm.logger.WithError(err).Warnf("Invalid CIDR network: %s", ipStr)
				continue
			}
			asm.ipFilter.allowedNetworks = append(asm.ipFilter.allowedNetworks, network)
		} else {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				asm.logger.Warnf("Invalid IP address: %s", ipStr)
				continue
			}
			// Convert single IP to /32 network
			network := &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(32, 32),
			}
			if ip.To4() == nil {
				network.Mask = net.CIDRMask(128, 128)
			}
			asm.ipFilter.allowedNetworks = append(asm.ipFilter.allowedNetworks, network)
		}
	}

	// Parse blocked IPs
	for _, ipStr := range asm.config.BlockedIPs {
		asm.ipFilter.blockedIPs[ipStr] = true
	}
}

// IsBlocked checks if an IP is blocked
func (ipf *IPFilter) IsBlocked(ipStr string) (bool, string) {
	ipf.mu.RLock()
	defer ipf.mu.RUnlock()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true, "invalid_ip"
	}

	// Check if IP is explicitly blocked
	if ipf.blockedIPs[ipStr] {
		return true, "blocked_ip"
	}

	// Check if IP is in allowed networks (if any are configured)
	if len(ipf.allowedNetworks) > 0 {
		allowed := false
		for _, network := range ipf.allowedNetworks {
			if network.Contains(ip) {
				allowed = true
				break
			}
		}
		if !allowed {
			return true, "not_in_allowlist"
		}
	}

	// Check private networks if not allowed
	if !ipf.allowPrivate && isPrivateIP(ip) {
		return true, "private_network_blocked"
	}

	return false, ""
}

// isPrivateIP checks if an IP is in private ranges
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	}

	for _, rangeStr := range privateRanges {
		_, network, _ := net.ParseCIDR(rangeStr)
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// initializeRequestValidator sets up request validation
func (asm *AdvancedSecurityMiddleware) initializeRequestValidator() {
	asm.requestValidator = &RequestValidator{
		maxRequestSize: asm.config.MaxRequestSize,
		maxHeaderSize:  asm.config.MaxHeaderSize,
		maxQueryParams: asm.config.MaxQueryParams,
	}

	// SQL Injection patterns
	if asm.config.EnableSQLInjectionFilter {
		sqlPatterns := []string{
			`(?i)(union\s+select)`,
			`(?i)(select\s+.*\s+from)`,
			`(?i)(insert\s+into)`,
			`(?i)(delete\s+from)`,
			`(?i)(update\s+.*\s+set)`,
			`(?i)(drop\s+(table|database))`,
			`(?i)(exec\s*\()`,
			`(?i)(script\s*:)`,
			`('.*'.*=.*'.*')`,
			`(;\s*(drop|delete|insert|update))`,
			`(--|\#|\/\*)`,
		}

		for _, pattern := range sqlPatterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				asm.logger.WithError(err).Warnf("Invalid SQL injection regex: %s", pattern)
				continue
			}
			asm.requestValidator.sqlInjectionRegex = append(asm.requestValidator.sqlInjectionRegex, regex)
		}
	}

	// XSS patterns
	if asm.config.EnableXSSFilter {
		xssPatterns := []string{
			`(?i)(<script[^>]*>)`,
			`(?i)(</script>)`,
			`(?i)(javascript:)`,
			`(?i)(on\w+\s*=)`,
			`(?i)(<iframe[^>]*>)`,
			`(?i)(<object[^>]*>)`,
			`(?i)(<embed[^>]*>)`,
			`(?i)(expression\s*\()`,
			`(?i)(vbscript:)`,
			`(?i)(<link[^>]*>)`,
		}

		for _, pattern := range xssPatterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				asm.logger.WithError(err).Warnf("Invalid XSS regex: %s", pattern)
				continue
			}
			asm.requestValidator.xssRegex = append(asm.requestValidator.xssRegex, regex)
		}
	}

	// Command injection patterns
	if asm.config.EnableCommandInjectionFilter {
		cmdPatterns := []string{
			`(?i)(;|\||&|\$\(|` + "`" + `|\$\{)`,
			`(?i)(nc\s+.*\s+.*\s+\d+)`,
			`(?i)(rm\s+-rf)`,
			`(?i)(wget\s+http)`,
			`(?i)(curl\s+http)`,
			`(?i)(chmod\s+\+x)`,
			`(?i)(\/bin\/(sh|bash|csh|zsh))`,
			`(?i)(cmd\.exe)`,
			`(?i)(powershell)`,
		}

		for _, pattern := range cmdPatterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				asm.logger.WithError(err).Warnf("Invalid command injection regex: %s", pattern)
				continue
			}
			asm.requestValidator.commandInjectionRegex = append(asm.requestValidator.commandInjectionRegex, regex)
		}
	}
}

// ValidateRequest validates the incoming request
func (rv *RequestValidator) ValidateRequest(c *gin.Context) error {
	// Check request size
	if c.Request.ContentLength > rv.maxRequestSize {
		return fmt.Errorf("request size exceeds maximum allowed (%d bytes)", rv.maxRequestSize)
	}

	// Check header size
	headerSize := 0
	for name, values := range c.Request.Header {
		headerSize += len(name)
		for _, value := range values {
			headerSize += len(value)
		}
	}
	if headerSize > rv.maxHeaderSize {
		return fmt.Errorf("header size exceeds maximum allowed (%d bytes)", rv.maxHeaderSize)
	}

	// Check query parameter count
	if len(c.Request.URL.Query()) > rv.maxQueryParams {
		return fmt.Errorf("too many query parameters (max: %d)", rv.maxQueryParams)
	}

	// Check for malicious patterns in URL, headers, and query parameters
	fullURL := c.Request.URL.String()

	// SQL Injection check
	for _, regex := range rv.sqlInjectionRegex {
		if regex.MatchString(fullURL) {
			return fmt.Errorf("potential SQL injection detected in URL")
		}
	}

	// XSS check
	for _, regex := range rv.xssRegex {
		if regex.MatchString(fullURL) {
			return fmt.Errorf("potential XSS attack detected in URL")
		}
	}

	// Command injection check
	for _, regex := range rv.commandInjectionRegex {
		if regex.MatchString(fullURL) {
			return fmt.Errorf("potential command injection detected in URL")
		}
	}

	// Check headers for malicious content
	for _, values := range c.Request.Header {
		for _, value := range values {
			for _, regex := range rv.sqlInjectionRegex {
				if regex.MatchString(value) {
					return fmt.Errorf("potential SQL injection detected in headers")
				}
			}
			for _, regex := range rv.xssRegex {
				if regex.MatchString(value) {
					return fmt.Errorf("potential XSS attack detected in headers")
				}
			}
		}
	}

	return nil
}

// initializeAttackDetector sets up attack detection
func (asm *AdvancedSecurityMiddleware) initializeAttackDetector() {
	asm.attackDetector = &AttackDetector{
		suspiciousPatterns: make(map[string]*regexp.Regexp),
		attackCounts:       make(map[string]*AttackCount),
	}

	// Add suspicious patterns
	patterns := map[string]string{
		"path_traversal":     `(?i)(\.\.\/|\.\.\\|%2e%2e%2f|%2e%2e%5c)`,
		"directory_listing":  `(?i)(index\s+of\s+\/|directory\s+listing)`,
		"password_files":     `(?i)(passwd|shadow|htpasswd|web\.config)`,
		"backup_files":       `(?i)\.(bak|backup|old|tmp|swp)$`,
		"config_files":       `(?i)\.(conf|config|ini|env)$`,
		"log_files":          `(?i)\.(log|logs)$`,
		"vulnerability_scan": `(?i)(nmap|nessus|openvas|nikto|dirb|gobuster)`,
		"automated_tools":    `(?i)(bot|crawler|scanner|spider)`,
	}

	for name, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			asm.logger.WithError(err).Warnf("Invalid suspicious pattern regex for %s: %s", name, pattern)
			continue
		}
		asm.attackDetector.suspiciousPatterns[name] = regex
	}
}

// detectAttacks checks for various attack patterns
func (asm *AdvancedSecurityMiddleware) detectAttacks(c *gin.Context) bool {
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	fullURL := c.Request.URL.String()

	asm.attackDetector.mu.Lock()
	defer asm.attackDetector.mu.Unlock()

	// Get or create attack count for this IP
	attackCount, exists := asm.attackDetector.attackCounts[clientIP]
	if !exists {
		attackCount = &AttackCount{}
		asm.attackDetector.attackCounts[clientIP] = attackCount
	}

	detected := false

	// Check for path traversal
	if asm.attackDetector.suspiciousPatterns["path_traversal"].MatchString(fullURL) {
		attackCount.PathTraversal++
		attackCount.LastAttempt = time.Now()
		detected = true
		asm.logger.Warnf("Path traversal attempt from %s: %s", clientIP, fullURL)
	}

	// Check for vulnerability scanning
	if asm.attackDetector.suspiciousPatterns["vulnerability_scan"].MatchString(userAgent) {
		detected = true
		asm.logger.Warnf("Vulnerability scan detected from %s: %s", clientIP, userAgent)
	}

	// Check for automated tools
	if asm.attackDetector.suspiciousPatterns["automated_tools"].MatchString(userAgent) {
		detected = true
		asm.logger.Warnf("Automated tool detected from %s: %s", clientIP, userAgent)
	}

	// Check for sensitive file access attempts
	for pattern, regex := range asm.attackDetector.suspiciousPatterns {
		if strings.Contains(pattern, "_files") && regex.MatchString(fullURL) {
			detected = true
			asm.logger.Warnf("Sensitive file access attempt (%s) from %s: %s", pattern, clientIP, fullURL)
		}
	}

	return detected
}

// initializeThreatIntelligence sets up threat intelligence
func (asm *AdvancedSecurityMiddleware) initializeThreatIntelligence() {
	asm.threatIntelligence = &ThreatIntelligence{
		knownBadIPs: make(map[string]*ThreatInfo),
		geoBlocks:   make(map[string]bool),
	}

	// Initialize with some known suspicious user agents
	suspiciousUAPatterns := []string{
		`(?i)(sqlmap|havij|pangolin)`,
		`(?i)(acunetix|netsparker|appscan)`,
		`(?i)(nikto|skipfish|w3af)`,
		`(?i)(masscan|zmap|zgrab)`,
		`(?i)(python-requests|curl|wget)`,
	}

	for _, pattern := range suspiciousUAPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			asm.logger.WithError(err).Warnf("Invalid suspicious UA regex: %s", pattern)
			continue
		}
		asm.threatIntelligence.suspiciousUAs = append(asm.threatIntelligence.suspiciousUAs, regex)
	}
}

// CheckThreat checks for threat intelligence indicators
func (ti *ThreatIntelligence) CheckThreat(c *gin.Context) *ThreatInfo {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Check known bad IPs
	if threat, exists := ti.knownBadIPs[clientIP]; exists {
		threat.LastSeen = time.Now()
		return threat
	}

	// Check suspicious user agents
	for _, regex := range ti.suspiciousUAs {
		if regex.MatchString(userAgent) {
			return &ThreatInfo{
				ThreatType:  "suspicious_user_agent",
				Severity:    "medium",
				Description: "Request from suspicious user agent",
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
				ReportCount: 1,
			}
		}
	}

	return nil
}

// addSecurityHeaders adds security headers to the response
func (asm *AdvancedSecurityMiddleware) addSecurityHeaders(c *gin.Context) {
	headers := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Cache-Control":             "no-cache, no-store, must-revalidate",
		"Pragma":                    "no-cache",
		"Expires":                   "0",
	}

	if asm.config.CSPDirectives != "" {
		headers["Content-Security-Policy"] = asm.config.CSPDirectives
	}

	for header, value := range headers {
		c.Header(header, value)
	}
}

// Security event logging
func (asm *AdvancedSecurityMiddleware) logSecurityEvent(eventType string, c *gin.Context, metadata map[string]interface{}) {
	if !asm.config.LogSecurityEvents {
		return
	}

	fields := logrus.Fields{
		"event_type": eventType,
		"ip":         c.ClientIP(),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"user_agent": c.GetHeader("User-Agent"),
		"timestamp":  time.Now(),
	}

	if metadata != nil {
		for k, v := range metadata {
			fields[k] = v
		}
	}

	asm.logger.WithFields(fields).Info("Security event")
}

// Metrics helpers
func (asm *AdvancedSecurityMiddleware) incrementMetric(metric string) {
	asm.securityMetrics.mu.Lock()
	defer asm.securityMetrics.mu.Unlock()

	switch metric {
	case "requests_blocked":
		asm.securityMetrics.RequestsBlocked++
	case "attacks_detected":
		asm.securityMetrics.AttacksDetected++
	case "suspicious_requests":
		asm.securityMetrics.SuspiciousRequests++
	case "rate_limit_violations":
		asm.securityMetrics.RateLimitViolations++
	case "ip_filter_violations":
		asm.securityMetrics.IPFilterViolations++
	}
}

// GetSecurityMetrics returns current security metrics
func (asm *AdvancedSecurityMiddleware) GetSecurityMetrics() *SecurityMetrics {
	asm.securityMetrics.mu.RLock()
	defer asm.securityMetrics.mu.RUnlock()

	return &SecurityMetrics{
		RequestsBlocked:     asm.securityMetrics.RequestsBlocked,
		AttacksDetected:     asm.securityMetrics.AttacksDetected,
		SuspiciousRequests:  asm.securityMetrics.SuspiciousRequests,
		RateLimitViolations: asm.securityMetrics.RateLimitViolations,
		IPFilterViolations:  asm.securityMetrics.IPFilterViolations,
	}
}

// UpdateThreatIntelligence updates threat intelligence data
func (asm *AdvancedSecurityMiddleware) UpdateThreatIntelligence(ip string, threat *ThreatInfo) {
	asm.threatIntelligence.mu.Lock()
	defer asm.threatIntelligence.mu.Unlock()

	asm.threatIntelligence.knownBadIPs[ip] = threat
}

// BlockIP adds an IP to the blocked list
func (asm *AdvancedSecurityMiddleware) BlockIP(ip string) {
	asm.ipFilter.mu.Lock()
	defer asm.ipFilter.mu.Unlock()

	asm.ipFilter.blockedIPs[ip] = true
	asm.logger.Infof("IP %s has been blocked", ip)
}

// UnblockIP removes an IP from the blocked list
func (asm *AdvancedSecurityMiddleware) UnblockIP(ip string) {
	asm.ipFilter.mu.Lock()
	defer asm.ipFilter.mu.Unlock()

	delete(asm.ipFilter.blockedIPs, ip)
	asm.logger.Infof("IP %s has been unblocked", ip)
}

// GenerateFingerprint generates a request fingerprint for tracking
func (asm *AdvancedSecurityMiddleware) GenerateFingerprint(c *gin.Context) string {
	if !asm.config.EnableFingerprinting {
		return ""
	}

	// Create fingerprint from various request attributes
	fingerprint := fmt.Sprintf("%s|%s|%s|%s",
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		c.GetHeader("Accept-Language"),
		c.GetHeader("Accept-Encoding"),
	)

	// Hash the fingerprint
	hash := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(hash[:])
}
