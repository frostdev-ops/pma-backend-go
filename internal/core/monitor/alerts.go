package monitor

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a system alert
type Alert struct {
	ID         string                 `json:"id"`
	Severity   AlertSeverity          `json:"severity"`
	Source     string                 `json:"source"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	ResolvedBy string                 `json:"resolved_by,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
}

// AlertRule represents a rule for generating alerts
type AlertRule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Metric     string        `json:"metric"`
	Operator   string        `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold  float64       `json:"threshold"`
	Duration   time.Duration `json:"duration"`
	Severity   AlertSeverity `json:"severity"`
	Message    string        `json:"message"`
	Enabled    bool          `json:"enabled"`
	Conditions []string      `json:"conditions,omitempty"`
}

// AlertThreshold represents a threshold configuration
type AlertThreshold struct {
	Metric   string        `json:"metric"`
	Warning  float64       `json:"warning"`
	Critical float64       `json:"critical"`
	Operator string        `json:"operator"`
	Duration time.Duration `json:"duration"`
}

// AlertManagerConfig contains configuration for the alert manager
type AlertManagerConfig struct {
	Enabled             bool             `json:"enabled"`
	MaxAlerts           int              `json:"max_alerts"`
	RetentionPeriod     time.Duration    `json:"retention_period"`
	DefaultThresholds   []AlertThreshold `json:"default_thresholds"`
	NotificationEnabled bool             `json:"notification_enabled"`
}

// AlertManager manages system alerts
type AlertManager struct {
	config     *AlertManagerConfig
	logger     *logrus.Logger
	alerts     map[string]*Alert
	rules      map[string]*AlertRule
	thresholds map[string]*AlertThreshold
	mu         sync.RWMutex

	// Callbacks for alert events
	onAlertCreated  []func(*Alert)
	onAlertResolved []func(*Alert)
	onAlertUpdated  []func(*Alert)
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config *AlertManagerConfig, logger *logrus.Logger) *AlertManager {
	if config == nil {
		config = &AlertManagerConfig{
			Enabled:             true,
			MaxAlerts:           1000,
			RetentionPeriod:     24 * time.Hour,
			NotificationEnabled: true,
		}
	}

	am := &AlertManager{
		config:     config,
		logger:     logger,
		alerts:     make(map[string]*Alert),
		rules:      make(map[string]*AlertRule),
		thresholds: make(map[string]*AlertThreshold),
	}

	// Set up default thresholds
	am.setupDefaultThresholds()

	// Start cleanup routine
	go am.cleanupRoutine()

	return am
}

// CreateAlert creates a new alert
func (am *AlertManager) CreateAlert(alert Alert) error {
	if !am.config.Enabled {
		return nil
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate ID if not provided
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Initialize details if nil
	if alert.Details == nil {
		alert.Details = make(map[string]interface{})
	}

	// Check if we're at max alerts limit
	if len(am.alerts) >= am.config.MaxAlerts {
		am.removeOldestAlert()
	}

	// Store the alert
	am.alerts[alert.ID] = &alert

	am.logger.WithFields(logrus.Fields{
		"alert_id": alert.ID,
		"severity": alert.Severity,
		"source":   alert.Source,
		"message":  alert.Message,
	}).Info("Alert created")

	// Trigger callbacks
	for _, callback := range am.onAlertCreated {
		go callback(&alert)
	}

	return nil
}

// ResolveAlert resolves an existing alert
func (am *AlertManager) ResolveAlert(id string) error {
	return am.ResolveAlertBy(id, "system")
}

// ResolveAlertBy resolves an alert with a specific resolver
func (am *AlertManager) ResolveAlertBy(id, resolvedBy string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[id]
	if !exists {
		return fmt.Errorf("alert with ID %s not found", id)
	}

	if alert.Resolved {
		return fmt.Errorf("alert with ID %s is already resolved", id)
	}

	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now
	alert.ResolvedBy = resolvedBy
	alert.Duration = now.Sub(alert.Timestamp)

	am.logger.WithFields(logrus.Fields{
		"alert_id":    alert.ID,
		"resolved_by": resolvedBy,
		"duration":    alert.Duration,
	}).Info("Alert resolved")

	// Trigger callbacks
	for _, callback := range am.onAlertResolved {
		go callback(alert)
	}

	return nil
}

// GetActiveAlerts returns all active (unresolved) alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// GetAllAlerts returns all alerts
func (am *AlertManager) GetAllAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []Alert
	for _, alert := range am.alerts {
		alerts = append(alerts, *alert)
	}

	return alerts
}

// GetAlertsBySource returns alerts from a specific source
func (am *AlertManager) GetAlertsBySource(source string) []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []Alert
	for _, alert := range am.alerts {
		if alert.Source == source {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// GetAlertsBySeverity returns alerts of a specific severity
func (am *AlertManager) GetAlertsBySeverity(severity AlertSeverity) []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []Alert
	for _, alert := range am.alerts {
		if alert.Severity == severity {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// SetThreshold sets an alert threshold for a metric
func (am *AlertManager) SetThreshold(metric string, threshold float64) error {
	return am.SetThresholdWithOperator(metric, threshold, ">=")
}

// SetThresholdWithOperator sets an alert threshold with a specific operator
func (am *AlertManager) SetThresholdWithOperator(metric string, threshold float64, operator string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.thresholds[metric] = &AlertThreshold{
		Metric:   metric,
		Critical: threshold,
		Operator: operator,
		Duration: 5 * time.Minute,
	}

	am.logger.WithFields(logrus.Fields{
		"metric":    metric,
		"threshold": threshold,
		"operator":  operator,
	}).Info("Alert threshold updated")

	return nil
}

// CheckThresholds checks if any thresholds are exceeded
func (am *AlertManager) CheckThresholds(metrics map[string]float64) {
	if !am.config.Enabled {
		return
	}

	am.mu.RLock()
	thresholds := make(map[string]*AlertThreshold)
	for k, v := range am.thresholds {
		thresholds[k] = v
	}
	am.mu.RUnlock()

	for metric, value := range metrics {
		threshold, exists := thresholds[metric]
		if !exists {
			continue
		}

		if am.checkThresholdExceeded(value, threshold.Critical, threshold.Operator) {
			alertID := fmt.Sprintf("threshold_%s_%v", metric, value)

			// Check if alert already exists
			if am.alertExists(alertID) {
				continue
			}

			alert := Alert{
				ID:       alertID,
				Severity: AlertSeverityCritical,
				Source:   "threshold_monitor",
				Message:  fmt.Sprintf("%s threshold exceeded: %.2f %s %.2f", metric, value, threshold.Operator, threshold.Critical),
				Details: map[string]interface{}{
					"metric":    metric,
					"value":     value,
					"threshold": threshold.Critical,
					"operator":  threshold.Operator,
				},
			}

			am.CreateAlert(alert)
		}
	}
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	am.rules[rule.ID] = &rule

	am.logger.WithFields(logrus.Fields{
		"rule_id": rule.ID,
		"name":    rule.Name,
		"metric":  rule.Metric,
	}).Info("Alert rule added")
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(id string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.rules, id)

	am.logger.WithField("rule_id", id).Info("Alert rule removed")
}

// GetRules returns all alert rules
func (am *AlertManager) GetRules() []AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var rules []AlertRule
	for _, rule := range am.rules {
		rules = append(rules, *rule)
	}

	return rules
}

// OnAlertCreated registers a callback for when alerts are created
func (am *AlertManager) OnAlertCreated(callback func(*Alert)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.onAlertCreated = append(am.onAlertCreated, callback)
}

// OnAlertResolved registers a callback for when alerts are resolved
func (am *AlertManager) OnAlertResolved(callback func(*Alert)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.onAlertResolved = append(am.onAlertResolved, callback)
}

// OnAlertUpdated registers a callback for when alerts are updated
func (am *AlertManager) OnAlertUpdated(callback func(*Alert)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.onAlertUpdated = append(am.onAlertUpdated, callback)
}

// GetAlertStats returns statistics about alerts
func (am *AlertManager) GetAlertStats() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	totalAlerts := len(am.alerts)
	activeAlerts := 0
	criticalAlerts := 0
	warningAlerts := 0
	infoAlerts := 0

	for _, alert := range am.alerts {
		if !alert.Resolved {
			activeAlerts++
		}

		switch alert.Severity {
		case AlertSeverityCritical:
			criticalAlerts++
		case AlertSeverityWarning:
			warningAlerts++
		case AlertSeverityInfo:
			infoAlerts++
		}
	}

	return map[string]interface{}{
		"total_alerts":    totalAlerts,
		"active_alerts":   activeAlerts,
		"critical_alerts": criticalAlerts,
		"warning_alerts":  warningAlerts,
		"info_alerts":     infoAlerts,
		"rules_count":     len(am.rules),
	}
}

// checkThresholdExceeded checks if a value exceeds a threshold
func (am *AlertManager) checkThresholdExceeded(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// alertExists checks if an alert with the given ID already exists
func (am *AlertManager) alertExists(id string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	_, exists := am.alerts[id]
	return exists
}

// removeOldestAlert removes the oldest resolved alert
func (am *AlertManager) removeOldestAlert() {
	var oldest *Alert
	var oldestID string

	for id, alert := range am.alerts {
		if alert.Resolved && (oldest == nil || alert.Timestamp.Before(oldest.Timestamp)) {
			oldest = alert
			oldestID = id
		}
	}

	if oldest != nil {
		delete(am.alerts, oldestID)
		am.logger.WithField("alert_id", oldestID).Debug("Removed oldest alert")
	}
}

// setupDefaultThresholds sets up default alert thresholds
func (am *AlertManager) setupDefaultThresholds() {
	defaultThresholds := []AlertThreshold{
		{Metric: "cpu_usage", Critical: 90.0, Warning: 80.0, Operator: ">="},
		{Metric: "memory_usage", Critical: 95.0, Warning: 85.0, Operator: ">="},
		{Metric: "disk_usage", Critical: 95.0, Warning: 90.0, Operator: ">="},
		{Metric: "error_rate", Critical: 0.1, Warning: 0.05, Operator: ">="},
	}

	for _, threshold := range defaultThresholds {
		am.thresholds[threshold.Metric] = &threshold
	}
}

// cleanupRoutine periodically removes old resolved alerts
func (am *AlertManager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		am.cleanupOldAlerts()
	}
}

// cleanupOldAlerts removes alerts older than the retention period
func (am *AlertManager) cleanupOldAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-am.config.RetentionPeriod)
	removed := 0

	for id, alert := range am.alerts {
		if alert.Resolved && alert.Timestamp.Before(cutoff) {
			delete(am.alerts, id)
			removed++
		}
	}

	if removed > 0 {
		am.logger.WithField("removed_count", removed).Info("Cleaned up old alerts")
	}
}

// NewAlert creates a new alert with the given parameters
func NewAlert(severity AlertSeverity, source, message string) Alert {
	return Alert{
		ID:        uuid.New().String(),
		Severity:  severity,
		Source:    source,
		Message:   message,
		Details:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// WithDetails adds details to an alert
func (a Alert) WithDetails(details map[string]interface{}) Alert {
	if a.Details == nil {
		a.Details = make(map[string]interface{})
	}

	for k, v := range details {
		a.Details[k] = v
	}

	return a
}

// WithDetail adds a single detail to an alert
func (a Alert) WithDetail(key string, value interface{}) Alert {
	if a.Details == nil {
		a.Details = make(map[string]interface{})
	}

	a.Details[key] = value
	return a
}

// ToJSON converts an alert to JSON
func (a Alert) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// FromJSON creates an alert from JSON
func AlertFromJSON(data []byte) (*Alert, error) {
	var alert Alert
	err := json.Unmarshal(data, &alert)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}
