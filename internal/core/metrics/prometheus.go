package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusCollector implements MetricsCollector using Prometheus metrics
type PrometheusCollector struct {
	config *MetricsConfig

	// HTTP Metrics
	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	httpActiveConnections prometheus.Gauge

	// WebSocket Metrics
	websocketConnections *prometheus.GaugeVec
	websocketMessages    *prometheus.CounterVec

	// Database Metrics
	databaseQueryDuration *prometheus.HistogramVec
	databaseConnections   prometheus.Gauge

	// Device Metrics
	deviceOperations *prometheus.CounterVec
	deviceStatus     *prometheus.GaugeVec

	// Automation Metrics
	automationExecutions *prometheus.CounterVec

	// LLM Metrics
	llmRequests     *prometheus.CounterVec
	llmTokensUsed   *prometheus.CounterVec
	llmResponseTime *prometheus.HistogramVec

	// System Metrics
	systemCPU    prometheus.Gauge
	systemMemory prometheus.Gauge
	systemDisk   prometheus.Gauge

	// Alert Metrics
	alertsTotal  *prometheus.CounterVec
	alertsActive *prometheus.GaugeVec

	// Generic metrics
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	gauges     map[string]*prometheus.GaugeVec
}

// NewPrometheusCollector creates a new Prometheus metrics collector
func NewPrometheusCollector(config *MetricsConfig) *PrometheusCollector {
	if config == nil {
		config = &MetricsConfig{
			Enabled: true,
			Prefix:  "pma",
		}
	}

	prefix := config.Prefix

	collector := &PrometheusCollector{
		config:     config,
		counters:   make(map[string]*prometheus.CounterVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
	}

	// Initialize HTTP metrics
	collector.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	collector.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	collector.httpActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Initialize WebSocket metrics
	collector.websocketConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_websocket_connections",
			Help: "Number of active WebSocket connections",
		},
		[]string{"type"},
	)

	collector.websocketMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"type", "direction"},
	)

	// Initialize Database metrics
	collector.databaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"operation"},
	)

	collector.databaseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_database_connections",
			Help: "Number of active database connections",
		},
	)

	// Initialize Device metrics
	collector.deviceOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_device_operations_total",
			Help: "Total number of device operations",
		},
		[]string{"device_type", "operation", "success"},
	)

	collector.deviceStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_device_status",
			Help: "Device status (1 = online, 0 = offline)",
		},
		[]string{"device_type", "device_id"},
	)

	// Initialize Automation metrics
	collector.automationExecutions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_automation_executions_total",
			Help: "Total number of automation rule executions",
		},
		[]string{"rule_id", "success"},
	)

	// Initialize LLM metrics
	collector.llmRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_llm_requests_total",
			Help: "Total number of LLM requests",
		},
		[]string{"provider", "success"},
	)

	collector.llmTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_llm_tokens_used_total",
			Help: "Total number of LLM tokens used",
		},
		[]string{"provider"},
	)

	collector.llmResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_llm_response_time_seconds",
			Help:    "LLM response time in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		},
		[]string{"provider"},
	)

	// Initialize System metrics
	collector.systemCPU = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_system_cpu_usage_percent",
			Help: "System CPU usage percentage",
		},
	)

	collector.systemMemory = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_system_memory_usage_percent",
			Help: "System memory usage percentage",
		},
	)

	collector.systemDisk = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_system_disk_usage_percent",
			Help: "System disk usage percentage",
		},
	)

	// Initialize Alert metrics
	collector.alertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_alerts_total",
			Help: "Total number of alerts generated",
		},
		[]string{"severity", "source"},
	)

	collector.alertsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_alerts_active",
			Help: "Number of active alerts",
		},
		[]string{"severity", "source"},
	)

	return collector
}

// RecordHTTPRequest records HTTP request metrics
func (p *PrometheusCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	if !p.config.Enabled {
		return
	}

	p.httpRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	p.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordWebSocketConnection records WebSocket connection metrics
func (p *PrometheusCollector) RecordWebSocketConnection(action string) {
	if !p.config.Enabled {
		return
	}

	switch action {
	case "connect":
		p.websocketConnections.WithLabelValues("client").Inc()
	case "disconnect":
		p.websocketConnections.WithLabelValues("client").Dec()
	case "message_sent":
		p.websocketMessages.WithLabelValues("client", "outbound").Inc()
	case "message_received":
		p.websocketMessages.WithLabelValues("client", "inbound").Inc()
	}
}

// RecordDatabaseQuery records database query metrics
func (p *PrometheusCollector) RecordDatabaseQuery(operation string, duration time.Duration) {
	if !p.config.Enabled {
		return
	}

	p.databaseQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordDeviceOperation records device operation metrics
func (p *PrometheusCollector) RecordDeviceOperation(deviceType, operation string, success bool, duration time.Duration) {
	if !p.config.Enabled {
		return
	}

	successStr := "false"
	if success {
		successStr = "true"
	}

	p.deviceOperations.WithLabelValues(deviceType, operation, successStr).Inc()
}

// RecordAutomationExecution records automation execution metrics
func (p *PrometheusCollector) RecordAutomationExecution(ruleID string, success bool, duration time.Duration) {
	if !p.config.Enabled {
		return
	}

	successStr := "false"
	if success {
		successStr = "true"
	}

	p.automationExecutions.WithLabelValues(ruleID, successStr).Inc()
}

// RecordLLMRequest records LLM request metrics
func (p *PrometheusCollector) RecordLLMRequest(provider string, success bool, duration time.Duration, tokens int) {
	if !p.config.Enabled {
		return
	}

	successStr := "false"
	if success {
		successStr = "true"
	}

	p.llmRequests.WithLabelValues(provider, successStr).Inc()
	p.llmResponseTime.WithLabelValues(provider).Observe(duration.Seconds())
	if tokens > 0 {
		p.llmTokensUsed.WithLabelValues(provider).Add(float64(tokens))
	}
}

// RecordSystemResource records system resource metrics
func (p *PrometheusCollector) RecordSystemResource(cpu, memory, disk float64) {
	if !p.config.Enabled {
		return
	}

	p.systemCPU.Set(cpu)
	p.systemMemory.Set(memory)
	p.systemDisk.Set(disk)
}

// RecordAlert records alert metrics
func (p *PrometheusCollector) RecordAlert(severity, source, message string) {
	if !p.config.Enabled {
		return
	}

	p.alertsTotal.WithLabelValues(severity, source).Inc()
	p.alertsActive.WithLabelValues(severity, source).Inc()
}

// IncrementCounter increments a generic counter metric
func (p *PrometheusCollector) IncrementCounter(name string, labels map[string]string) {
	if !p.config.Enabled {
		return
	}

	counter, exists := p.counters[name]
	if !exists {
		labelNames := make([]string, 0, len(labels))
		for key := range labels {
			labelNames = append(labelNames, key)
		}

		counter = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: p.config.Prefix + "_" + name,
				Help: "Generic counter metric: " + name,
			},
			labelNames,
		)
		p.counters[name] = counter
	}

	labelValues := make([]string, 0, len(labels))
	for _, value := range labels {
		labelValues = append(labelValues, value)
	}

	counter.WithLabelValues(labelValues...).Inc()
}

// RecordHistogram records a value in a histogram metric
func (p *PrometheusCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	if !p.config.Enabled {
		return
	}

	histogram, exists := p.histograms[name]
	if !exists {
		labelNames := make([]string, 0, len(labels))
		for key := range labels {
			labelNames = append(labelNames, key)
		}

		histogram = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    p.config.Prefix + "_" + name,
				Help:    "Generic histogram metric: " + name,
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)
		p.histograms[name] = histogram
	}

	labelValues := make([]string, 0, len(labels))
	for _, value := range labels {
		labelValues = append(labelValues, value)
	}

	histogram.WithLabelValues(labelValues...).Observe(value)
}

// SetGauge sets a gauge metric value
func (p *PrometheusCollector) SetGauge(name string, value float64, labels map[string]string) {
	if !p.config.Enabled {
		return
	}

	gauge, exists := p.gauges[name]
	if !exists {
		labelNames := make([]string, 0, len(labels))
		for key := range labels {
			labelNames = append(labelNames, key)
		}

		gauge = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: p.config.Prefix + "_" + name,
				Help: "Generic gauge metric: " + name,
			},
			labelNames,
		)
		p.gauges[name] = gauge
	}

	labelValues := make([]string, 0, len(labels))
	for _, value := range labels {
		labelValues = append(labelValues, value)
	}

	gauge.WithLabelValues(labelValues...).Set(value)
}
