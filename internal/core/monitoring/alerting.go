package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AlertingEngine manages sophisticated alerting rules and notifications
type AlertingEngine struct {
	config             *AlertingConfig
	logger             *logrus.Logger
	rules              map[string]*AlertRule
	activeAlerts       map[string]*ActiveAlert
	notificationMgr    *NotificationManager
	escalationMgr      *EscalationManager
	suppressionMgr     *SuppressionManager
	metricEvaluator    *MetricEvaluator
	ruleGroups         map[string]*AlertRuleGroup
	lastEvaluation     time.Time
	evaluationInterval time.Duration
	mu                 sync.RWMutex
	stopChan           chan bool
	runningEvaluations map[string]bool
}

// AlertingConfig contains alerting engine configuration
type AlertingConfig struct {
	Enabled              bool                   `json:"enabled"`
	EvaluationInterval   time.Duration          `json:"evaluation_interval"`
	AlertRetention       time.Duration          `json:"alert_retention"`
	MaxConcurrentEvals   int                    `json:"max_concurrent_evals"`
	DefaultSeverity      AlertSeverity          `json:"default_severity"`
	NotificationChannels []*NotificationChannel `json:"notification_channels"`
	EscalationPolicies   []*EscalationPolicy    `json:"escalation_policies"`
	SuppressionRules     []*SuppressionRule     `json:"suppression_rules"`
	MetricThresholds     map[string]float64     `json:"metric_thresholds"`
	AlertManagerEndpoint string                 `json:"alert_manager_endpoint"`
	EnableWebhooks       bool                   `json:"enable_webhooks"`
	WebhookTimeout       time.Duration          `json:"webhook_timeout"`
	RetryPolicy          *RetryPolicy           `json:"retry_policy"`
}

// AlertRule defines an alerting rule
type AlertRule struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Enabled           bool              `json:"enabled"`
	Expression        string            `json:"expression"`
	Severity          AlertSeverity     `json:"severity"`
	For               time.Duration     `json:"for"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	NotificationGroup string            `json:"notification_group"`
	EscalationPolicy  string            `json:"escalation_policy"`
	Conditions        []*AlertCondition `json:"conditions"`
	MetricQueries     []*MetricQuery    `json:"metric_queries"`
	LastEvaluated     time.Time         `json:"last_evaluated"`
	EvaluationCount   int64             `json:"evaluation_count"`
	TriggerCount      int64             `json:"trigger_count"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	CreatedBy         string            `json:"created_by"`
	GroupID           string            `json:"group_id"`
	Dependencies      []string          `json:"dependencies"`
	Inhibits          []string          `json:"inhibits"`
}

// AlertRuleGroup groups related alert rules
type AlertRuleGroup struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Rules            []*AlertRule      `json:"rules"`
	EvaluationTime   time.Duration     `json:"evaluation_time"`
	LastEvaluation   time.Time         `json:"last_evaluation"`
	Enabled          bool              `json:"enabled"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`
	ConcurrencyLimit int               `json:"concurrency_limit"`
}

// AlertCondition defines a condition for an alert rule
type AlertCondition struct {
	Metric      string            `json:"metric"`
	Operator    ConditionOperator `json:"operator"`
	Value       float64           `json:"value"`
	Aggregation string            `json:"aggregation"`
	TimeWindow  time.Duration     `json:"time_window"`
	Labels      map[string]string `json:"labels"`
}

// MetricQuery defines a metric query for alert evaluation
type MetricQuery struct {
	Name       string            `json:"name"`
	Query      string            `json:"query"`
	Datasource string            `json:"datasource"`
	RefID      string            `json:"ref_id"`
	Labels     map[string]string `json:"labels"`
}

// ActiveAlert represents an active alert
type ActiveAlert struct {
	ID                string            `json:"id"`
	RuleID            string            `json:"rule_id"`
	RuleName          string            `json:"rule_name"`
	Severity          AlertSeverity     `json:"severity"`
	State             AlertState        `json:"state"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	StartsAt          time.Time         `json:"starts_at"`
	EndsAt            *time.Time        `json:"ends_at,omitempty"`
	LastNotified      time.Time         `json:"last_notified"`
	NotificationCount int               `json:"notification_count"`
	Value             float64           `json:"value"`
	Fingerprint       string            `json:"fingerprint"`
	GeneratorURL      string            `json:"generator_url"`
	SilenceIDs        []string          `json:"silence_ids"`
	Inhibited         bool              `json:"inhibited"`
	InhibitedBy       []string          `json:"inhibited_by"`
	History           []*AlertEvent     `json:"history"`
}

// AlertEvent represents an event in an alert's lifecycle
type AlertEvent struct {
	Timestamp   time.Time   `json:"timestamp"`
	Type        EventType   `json:"type"`
	Description string      `json:"description"`
	Value       float64     `json:"value,omitempty"`
	User        string      `json:"user,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
}

// NotificationChannel defines a notification channel
type NotificationChannel struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       ChannelType              `json:"type"`
	Enabled    bool                     `json:"enabled"`
	Settings   map[string]interface{}   `json:"settings"`
	Conditions []*NotificationCondition `json:"conditions"`
	RateLimit  *RateLimit               `json:"rate_limit"`
	Template   *NotificationTemplate    `json:"template"`
	Retries    int                      `json:"retries"`
	Timeout    time.Duration            `json:"timeout"`
}

// NotificationCondition defines when notifications should be sent
type NotificationCondition struct {
	Severity    AlertSeverity     `json:"severity"`
	Tags        map[string]string `json:"tags"`
	TimeWindow  time.Duration     `json:"time_window"`
	MinInterval time.Duration     `json:"min_interval"`
}

// NotificationTemplate defines notification message templates
type NotificationTemplate struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Format  string `json:"format"` // "text", "html", "markdown"
}

// EscalationPolicy defines how alerts should be escalated
type EscalationPolicy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Steps       []*EscalationStep `json:"steps"`
	Enabled     bool              `json:"enabled"`
	Labels      map[string]string `json:"labels"`
}

// EscalationStep defines a step in an escalation policy
type EscalationStep struct {
	Order          int           `json:"order"`
	WaitTime       time.Duration `json:"wait_time"`
	Channels       []string      `json:"channels"`
	Conditions     []string      `json:"conditions"`
	AutoResolve    bool          `json:"auto_resolve"`
	RepeatInterval time.Duration `json:"repeat_interval"`
	MaxRepeats     int           `json:"max_repeats"`
}

// SuppressionRule defines alert suppression rules
type SuppressionRule struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Matchers  []*LabelMatcher `json:"matchers"`
	StartsAt  time.Time       `json:"starts_at"`
	EndsAt    time.Time       `json:"ends_at"`
	CreatedBy string          `json:"created_by"`
	Comment   string          `json:"comment"`
	Enabled   bool            `json:"enabled"`
}

// LabelMatcher defines label matching criteria
type LabelMatcher struct {
	Name     string         `json:"name"`
	Value    string         `json:"value"`
	Operator MatchType      `json:"operator"`
	Regex    *regexp.Regexp `json:"-"`
}

// RateLimit defines rate limiting for notifications
type RateLimit struct {
	Enabled   bool          `json:"enabled"`
	Interval  time.Duration `json:"interval"`
	MaxBurst  int           `json:"max_burst"`
	Threshold int           `json:"threshold"`
}

// RetryPolicy defines retry behavior for failed notifications
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// MetricEvaluator evaluates metrics for alert rules
type MetricEvaluator struct {
	prometheusClient PrometheusClient
	logger           *logrus.Logger
	queryTimeout     time.Duration
	maxQueryDuration time.Duration
}

// PrometheusClient interface for Prometheus queries
type PrometheusClient interface {
	Query(ctx context.Context, query string, ts time.Time) (interface{}, error)
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (interface{}, error)
}

// Enums
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
	SeverityInfo     AlertSeverity = "info"
)

type AlertState string

const (
	StateFiring     AlertState = "firing"
	StatePending    AlertState = "pending"
	StateResolved   AlertState = "resolved"
	StateSuppressed AlertState = "suppressed"
)

type ConditionOperator string

const (
	OpGreaterThan    ConditionOperator = "gt"
	OpLessThan       ConditionOperator = "lt"
	OpGreaterOrEqual ConditionOperator = "gte"
	OpLessOrEqual    ConditionOperator = "lte"
	OpEqual          ConditionOperator = "eq"
	OpNotEqual       ConditionOperator = "neq"
	OpContains       ConditionOperator = "contains"
	OpRegex          ConditionOperator = "regex"
)

type ChannelType string

const (
	ChannelEmail      ChannelType = "email"
	ChannelSlack      ChannelType = "slack"
	ChannelWebhook    ChannelType = "webhook"
	ChannelSMS        ChannelType = "sms"
	ChannelPagerDuty  ChannelType = "pagerduty"
	ChannelMattermost ChannelType = "mattermost"
	ChannelTeams      ChannelType = "teams"
	ChannelDiscord    ChannelType = "discord"
)

type EventType string

const (
	EventFiring       EventType = "firing"
	EventResolved     EventType = "resolved"
	EventSuppressed   EventType = "suppressed"
	EventAcknowledged EventType = "acknowledged"
	EventEscalated    EventType = "escalated"
)

type MatchType string

const (
	MatchEqual    MatchType = "equal"
	MatchNotEqual MatchType = "not_equal"
	MatchRegex    MatchType = "regex"
	MatchNotRegex MatchType = "not_regex"
)

// NewAlertingEngine creates a new alerting engine
func NewAlertingEngine(config *AlertingConfig, logger *logrus.Logger) *AlertingEngine {
	if config == nil {
		config = DefaultAlertingConfig()
	}

	engine := &AlertingEngine{
		config:             config,
		logger:             logger,
		rules:              make(map[string]*AlertRule),
		activeAlerts:       make(map[string]*ActiveAlert),
		ruleGroups:         make(map[string]*AlertRuleGroup),
		runningEvaluations: make(map[string]bool),
		evaluationInterval: config.EvaluationInterval,
		stopChan:           make(chan bool),
	}

	// Initialize components
	engine.notificationMgr = NewNotificationManager(config.NotificationChannels, logger)
	engine.escalationMgr = NewEscalationManager(config.EscalationPolicies, logger)
	engine.suppressionMgr = NewSuppressionManager(config.SuppressionRules, logger)
	engine.metricEvaluator = NewMetricEvaluator(logger)

	return engine
}

// DefaultAlertingConfig returns default alerting configuration
func DefaultAlertingConfig() *AlertingConfig {
	return &AlertingConfig{
		Enabled:            true,
		EvaluationInterval: time.Minute,
		AlertRetention:     time.Hour * 24 * 7, // 7 days
		MaxConcurrentEvals: 10,
		DefaultSeverity:    SeverityMedium,
		EnableWebhooks:     true,
		WebhookTimeout:     time.Second * 30,
		RetryPolicy: &RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  time.Second * 5,
			MaxDelay:      time.Minute * 5,
			BackoffFactor: 2.0,
		},
		MetricThresholds: map[string]float64{
			"cpu_usage":    80.0,
			"memory_usage": 85.0,
			"disk_usage":   90.0,
			"error_rate":   5.0,
		},
	}
}

// Start starts the alerting engine
func (ae *AlertingEngine) Start(ctx context.Context) error {
	if !ae.config.Enabled {
		ae.logger.Info("Alerting engine is disabled")
		return nil
	}

	ae.logger.Info("Starting alerting engine")

	// Start evaluation loop
	go ae.evaluationLoop(ctx)

	// Start cleanup routine
	go ae.cleanupRoutine(ctx)

	return nil
}

// Stop stops the alerting engine
func (ae *AlertingEngine) Stop() {
	ae.logger.Info("Stopping alerting engine")
	close(ae.stopChan)
}

// AddRule adds an alert rule
func (ae *AlertingEngine) AddRule(rule *AlertRule) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if rule.ID == "" {
		rule.ID = ae.generateRuleID(rule.Name)
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	ae.rules[rule.ID] = rule

	ae.logger.Infof("Added alert rule: %s", rule.Name)
	return nil
}

// RemoveRule removes an alert rule
func (ae *AlertingEngine) RemoveRule(ruleID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	delete(ae.rules, ruleID)

	// Remove any active alerts for this rule
	for alertID, alert := range ae.activeAlerts {
		if alert.RuleID == ruleID {
			delete(ae.activeAlerts, alertID)
		}
	}

	ae.logger.Infof("Removed alert rule: %s", ruleID)
	return nil
}

// GetRules returns all alert rules
func (ae *AlertingEngine) GetRules() map[string]*AlertRule {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	rules := make(map[string]*AlertRule)
	for id, rule := range ae.rules {
		rules[id] = rule
	}
	return rules
}

// GetActiveAlerts returns all active alerts
func (ae *AlertingEngine) GetActiveAlerts() map[string]*ActiveAlert {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	alerts := make(map[string]*ActiveAlert)
	for id, alert := range ae.activeAlerts {
		alerts[id] = alert
	}
	return alerts
}

// GetAlertsByState returns alerts filtered by state
func (ae *AlertingEngine) GetAlertsByState(state AlertState) []*ActiveAlert {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	var alerts []*ActiveAlert
	for _, alert := range ae.activeAlerts {
		if alert.State == state {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// GetAlertsBySeverity returns alerts filtered by severity
func (ae *AlertingEngine) GetAlertsBySeverity(severity AlertSeverity) []*ActiveAlert {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	var alerts []*ActiveAlert
	for _, alert := range ae.activeAlerts {
		if alert.Severity == severity {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// AcknowledgeAlert acknowledges an alert
func (ae *AlertingEngine) AcknowledgeAlert(alertID, userID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	alert, exists := ae.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	// Add acknowledgment event
	event := &AlertEvent{
		Timestamp:   time.Now(),
		Type:        EventAcknowledged,
		Description: fmt.Sprintf("Alert acknowledged by %s", userID),
		User:        userID,
	}
	alert.History = append(alert.History, event)

	ae.logger.Infof("Alert %s acknowledged by %s", alertID, userID)
	return nil
}

// ResolveAlert manually resolves an alert
func (ae *AlertingEngine) ResolveAlert(alertID, userID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	alert, exists := ae.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.State = StateResolved
	now := time.Now()
	alert.EndsAt = &now

	// Add resolution event
	event := &AlertEvent{
		Timestamp:   time.Now(),
		Type:        EventResolved,
		Description: fmt.Sprintf("Alert manually resolved by %s", userID),
		User:        userID,
	}
	alert.History = append(alert.History, event)

	ae.logger.Infof("Alert %s manually resolved by %s", alertID, userID)
	return nil
}

// evaluationLoop runs the main evaluation loop
func (ae *AlertingEngine) evaluationLoop(ctx context.Context) {
	ticker := time.NewTicker(ae.evaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ae.stopChan:
			return
		case <-ticker.C:
			ae.evaluateRules(ctx)
		}
	}
}

// evaluateRules evaluates all alert rules
func (ae *AlertingEngine) evaluateRules(ctx context.Context) {
	ae.mu.RLock()
	rules := make([]*AlertRule, 0, len(ae.rules))
	for _, rule := range ae.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	ae.mu.RUnlock()

	ae.lastEvaluation = time.Now()

	// Evaluate rules concurrently with limit
	semaphore := make(chan struct{}, ae.config.MaxConcurrentEvals)
	var wg sync.WaitGroup

	for _, rule := range rules {
		semaphore <- struct{}{}
		wg.Add(1)

		go func(r *AlertRule) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			ae.evaluateRule(ctx, r)
		}(rule)
	}

	wg.Wait()
}

// evaluateRule evaluates a single alert rule
func (ae *AlertingEngine) evaluateRule(ctx context.Context, rule *AlertRule) {
	ae.mu.Lock()
	if ae.runningEvaluations[rule.ID] {
		ae.mu.Unlock()
		return // Already evaluating this rule
	}
	ae.runningEvaluations[rule.ID] = true
	ae.mu.Unlock()

	defer func() {
		ae.mu.Lock()
		delete(ae.runningEvaluations, rule.ID)
		rule.LastEvaluated = time.Now()
		rule.EvaluationCount++
		ae.mu.Unlock()
	}()

	// Evaluate metric queries
	alertState, value, err := ae.metricEvaluator.EvaluateRule(ctx, rule)
	if err != nil {
		ae.logger.WithError(err).Errorf("Failed to evaluate rule %s", rule.Name)
		return
	}

	ae.handleAlertState(rule, alertState, value)
}

// handleAlertState handles the result of rule evaluation
func (ae *AlertingEngine) handleAlertState(rule *AlertRule, shouldFire bool, value float64) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	alertID := ae.generateAlertID(rule, value)
	existingAlert, exists := ae.activeAlerts[alertID]

	if shouldFire {
		if !exists {
			// Create new alert
			alert := &ActiveAlert{
				ID:          alertID,
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				State:       StatePending,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				StartsAt:    time.Now(),
				Value:       value,
				Fingerprint: ae.generateFingerprint(rule, value),
				History:     []*AlertEvent{},
			}

			// Copy labels and annotations from rule
			for k, v := range rule.Labels {
				alert.Labels[k] = v
			}
			for k, v := range rule.Annotations {
				alert.Annotations[k] = v
			}

			ae.activeAlerts[alertID] = alert
			rule.TriggerCount++

			// Add firing event
			event := &AlertEvent{
				Timestamp:   time.Now(),
				Type:        EventFiring,
				Description: "Alert started firing",
				Value:       value,
			}
			alert.History = append(alert.History, event)

			ae.logger.Warnf("Alert firing: %s (value: %.2f)", rule.Name, value)

			// Check if alert should move to firing state immediately or wait
			if rule.For == 0 {
				alert.State = StateFiring
				ae.sendNotification(alert)
			}

		} else {
			// Update existing alert
			existingAlert.Value = value
			existingAlert.LastNotified = time.Now()

			// Check if pending alert should move to firing
			if existingAlert.State == StatePending && rule.For > 0 {
				if time.Since(existingAlert.StartsAt) >= rule.For {
					existingAlert.State = StateFiring
					ae.sendNotification(existingAlert)

					// Add state change event
					event := &AlertEvent{
						Timestamp:   time.Now(),
						Type:        EventFiring,
						Description: "Alert moved from pending to firing",
						Value:       value,
					}
					existingAlert.History = append(existingAlert.History, event)
				}
			}
		}
	} else {
		if exists && existingAlert.State != StateResolved {
			// Resolve alert
			existingAlert.State = StateResolved
			now := time.Now()
			existingAlert.EndsAt = &now

			// Add resolution event
			event := &AlertEvent{
				Timestamp:   time.Now(),
				Type:        EventResolved,
				Description: "Alert condition no longer met",
				Value:       value,
			}
			existingAlert.History = append(existingAlert.History, event)

			ae.logger.Infof("Alert resolved: %s", rule.Name)
			ae.sendResolutionNotification(existingAlert)
		}
	}
}

// sendNotification sends notification for an alert
func (ae *AlertingEngine) sendNotification(alert *ActiveAlert) {
	if ae.notificationMgr != nil {
		go ae.notificationMgr.SendAlert(alert)
	}

	if ae.escalationMgr != nil {
		go ae.escalationMgr.StartEscalation(alert)
	}
}

// sendResolutionNotification sends resolution notification
func (ae *AlertingEngine) sendResolutionNotification(alert *ActiveAlert) {
	if ae.notificationMgr != nil {
		go ae.notificationMgr.SendResolution(alert)
	}
}

// cleanupRoutine performs periodic cleanup of old alerts
func (ae *AlertingEngine) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ae.stopChan:
			return
		case <-ticker.C:
			ae.cleanupOldAlerts()
		}
	}
}

// cleanupOldAlerts removes old resolved alerts
func (ae *AlertingEngine) cleanupOldAlerts() {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	cutoff := time.Now().Add(-ae.config.AlertRetention)
	toRemove := []string{}

	for alertID, alert := range ae.activeAlerts {
		if alert.State == StateResolved && alert.EndsAt != nil && alert.EndsAt.Before(cutoff) {
			toRemove = append(toRemove, alertID)
		}
	}

	for _, alertID := range toRemove {
		delete(ae.activeAlerts, alertID)
	}

	if len(toRemove) > 0 {
		ae.logger.Infof("Cleaned up %d old alerts", len(toRemove))
	}
}

// Helper functions
func (ae *AlertingEngine) generateRuleID(name string) string {
	return fmt.Sprintf("rule_%s_%d", name, time.Now().Unix())
}

func (ae *AlertingEngine) generateAlertID(rule *AlertRule, value float64) string {
	return fmt.Sprintf("alert_%s_%s", rule.ID, ae.generateFingerprint(rule, value))
}

func (ae *AlertingEngine) generateFingerprint(rule *AlertRule, value float64) string {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_id": rule.ID,
		"labels":  rule.Labels,
		"value":   int64(value), // Rounded to avoid float precision issues
	})
	return fmt.Sprintf("%x", data)[:16]
}

// Placeholder implementations for components
func NewNotificationManager(channels []*NotificationChannel, logger *logrus.Logger) *NotificationManager {
	return &NotificationManager{channels: channels, logger: logger}
}

func NewEscalationManager(policies []*EscalationPolicy, logger *logrus.Logger) *EscalationManager {
	return &EscalationManager{policies: policies, logger: logger}
}

func NewSuppressionManager(rules []*SuppressionRule, logger *logrus.Logger) *SuppressionManager {
	return &SuppressionManager{rules: rules, logger: logger}
}

func NewMetricEvaluator(logger *logrus.Logger) *MetricEvaluator {
	return &MetricEvaluator{
		logger:           logger,
		queryTimeout:     time.Second * 30,
		maxQueryDuration: time.Minute * 5,
	}
}

// Component placeholder structs
type NotificationManager struct {
	channels []*NotificationChannel
	logger   *logrus.Logger
}

func (nm *NotificationManager) SendAlert(alert *ActiveAlert) {
	nm.logger.Infof("Sending alert notification: %s", alert.RuleName)
}

func (nm *NotificationManager) SendResolution(alert *ActiveAlert) {
	nm.logger.Infof("Sending resolution notification: %s", alert.RuleName)
}

type EscalationManager struct {
	policies []*EscalationPolicy
	logger   *logrus.Logger
}

func (em *EscalationManager) StartEscalation(alert *ActiveAlert) {
	em.logger.Infof("Starting escalation for alert: %s", alert.RuleName)
}

type SuppressionManager struct {
	rules  []*SuppressionRule
	logger *logrus.Logger
}

func (me *MetricEvaluator) EvaluateRule(ctx context.Context, rule *AlertRule) (bool, float64, error) {
	// Mock evaluation - in real implementation would query Prometheus
	value := 50.0 + float64(time.Now().Unix()%50)
	shouldFire := value > 75.0

	me.logger.Debugf("Evaluated rule %s: value=%.2f, shouldFire=%t", rule.Name, value, shouldFire)
	return shouldFire, value, nil
}
