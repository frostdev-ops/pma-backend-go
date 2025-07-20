package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// EnhancedClient extends the basic client with optimization features
type EnhancedClient struct {
	*Client // Embed original client

	// Optimization settings
	compressionEnabled bool
	compressionType    CompressionAlgorithm
	batchingEnabled    bool
	pooled             bool

	// Performance tracking
	metrics           *EnhancedClientMetrics
	throughputTracker *ClientThroughputTracker
	latencyTracker    *LatencyTracker

	// Health monitoring
	health        *EnhancedClientHealth
	healthChecker *ClientHealthChecker

	// Message processing
	messageProcessor  *MessageProcessor
	compressionEngine *CompressionEngine

	// Flow control
	sendLimiter         *RateLimiter
	receiveLimiter      *RateLimiter
	backpressureHandler *BackpressureHandler

	// Connection management
	connectionManager *ConnectionManager
	reconnectStrategy *ReconnectStrategy

	// Security
	securityManager *ClientSecurityManager

	// Configuration
	config *EnhancedClientConfig

	// State management
	state        ClientState
	stateMu      sync.RWMutex
	lastActivity time.Time

	// Channels and control
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	// Optimization features
	messageBuffer     *CircularBuffer
	compressionBuffer []byte
	batchBuffer       *MessageBatchBuffer

	// Monitoring
	logger *logrus.Logger
}

// EnhancedClientConfig contains configuration for enhanced clients
type EnhancedClientConfig struct {
	// Compression settings
	EnableCompression    bool                 `json:"enable_compression"`
	CompressionAlgorithm CompressionAlgorithm `json:"compression_algorithm"`
	CompressionThreshold int                  `json:"compression_threshold"`
	CompressionLevel     int                  `json:"compression_level"`

	// Batching settings
	EnableBatching bool          `json:"enable_batching"`
	BatchSize      int           `json:"batch_size"`
	BatchTimeout   time.Duration `json:"batch_timeout"`
	MaxBatchSize   int           `json:"max_batch_size"`

	// Rate limiting
	SendRateLimit    int `json:"send_rate_limit"`    // messages per second
	ReceiveRateLimit int `json:"receive_rate_limit"` // messages per second
	BurstLimit       int `json:"burst_limit"`        // burst allowance

	// Health monitoring
	EnableHealthCheck    bool          `json:"enable_health_check"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	PingInterval         time.Duration `json:"ping_interval"`
	PongTimeout          time.Duration `json:"pong_timeout"`
	MaxConsecutiveErrors int           `json:"max_consecutive_errors"`

	// Performance
	ReadBufferSize  int           `json:"read_buffer_size"`
	WriteBufferSize int           `json:"write_buffer_size"`
	MaxMessageSize  int64         `json:"max_message_size"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ReadTimeout     time.Duration `json:"read_timeout"`

	// Reconnection
	EnableAutoReconnect  bool          `json:"enable_auto_reconnect"`
	ReconnectInterval    time.Duration `json:"reconnect_interval"`
	MaxReconnectAttempts int           `json:"max_reconnect_attempts"`
	BackoffMultiplier    float64       `json:"backoff_multiplier"`
	MaxBackoffTime       time.Duration `json:"max_backoff_time"`

	// Security
	EnableTLS        bool          `json:"enable_tls"`
	VerifyTLS        bool          `json:"verify_tls"`
	MaxConnectionAge time.Duration `json:"max_connection_age"`

	// Debugging
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsInterval time.Duration `json:"metrics_interval"`
	EnableProfiling bool          `json:"enable_profiling"`
	LogLevel        string        `json:"log_level"`
}

// EnhancedClientMetrics tracks detailed client performance metrics
type EnhancedClientMetrics struct {
	// Message statistics
	MessagesSent       int64 `json:"messages_sent"`
	MessagesReceived   int64 `json:"messages_received"`
	BytesSent          int64 `json:"bytes_sent"`
	BytesReceived      int64 `json:"bytes_received"`
	CompressedMessages int64 `json:"compressed_messages"`
	BatchedMessages    int64 `json:"batched_messages"`

	// Performance metrics
	AverageMessageSize float64       `json:"average_message_size"`
	MessageRate        float64       `json:"message_rate"` // messages per second
	ThroughputMbps     float64       `json:"throughput_mbps"`
	AverageLatency     time.Duration `json:"average_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`

	// Connection metrics
	ConnectionCount   int64         `json:"connection_count"`
	ReconnectionCount int64         `json:"reconnection_count"`
	ConnectionUptime  time.Duration `json:"connection_uptime"`
	TotalDowntime     time.Duration `json:"total_downtime"`

	// Error metrics
	ErrorCount          int64 `json:"error_count"`
	TimeoutCount        int64 `json:"timeout_count"`
	CompressionErrors   int64 `json:"compression_errors"`
	DecompressionErrors int64 `json:"decompression_errors"`

	// Health metrics
	HealthScore       float64   `json:"health_score"`
	LastHealthCheck   time.Time `json:"last_health_check"`
	ConsecutiveErrors int       `json:"consecutive_errors"`

	// Resource usage
	MemoryUsage       int64   `json:"memory_usage"`
	GoroutineCount    int     `json:"goroutine_count"`
	BufferUtilization float64 `json:"buffer_utilization"`

	// Timestamps
	StartTime       time.Time `json:"start_time"`
	LastMessageTime time.Time `json:"last_message_time"`
	LastErrorTime   time.Time `json:"last_error_time"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

// EnhancedClientHealth provides detailed health information
type EnhancedClientHealth struct {
	Status             HealthStatus   `json:"status"`
	Score              float64        `json:"score"` // 0-100
	LastPing           time.Time      `json:"last_ping"`
	LastPong           time.Time      `json:"last_pong"`
	PingLatency        time.Duration  `json:"ping_latency"`
	AveragePingLatency time.Duration  `json:"average_ping_latency"`
	ConsecutiveErrors  int            `json:"consecutive_errors"`
	LastError          string         `json:"last_error"`
	LastErrorTime      time.Time      `json:"last_error_time"`
	Issues             []HealthIssue  `json:"issues"`
	Recommendations    []string       `json:"recommendations"`
	NetworkQuality     NetworkQuality `json:"network_quality"`
	ResourceHealth     ResourceHealth `json:"resource_health"`
}

// ClientThroughputTracker tracks throughput over time
type ClientThroughputTracker struct {
	samples        []ThroughputSample
	maxSamples     int
	windowDuration time.Duration
	mu             sync.RWMutex
}

// LatencyTracker tracks message latency
type LatencyTracker struct {
	samples    []LatencySample
	maxSamples int
	mu         sync.RWMutex
}

// MessageProcessor handles message processing with optimization
type MessageProcessor struct {
	compressionEngine *CompressionEngine
	batchProcessor    *BatchProcessor
	filterChain       []MessageFilter
	transformChain    []MessageTransform
	validator         MessageValidator
	stats             *ProcessorStats
	logger            *logrus.Logger
}

// RateLimiter controls message rate
type RateLimiter struct {
	rate       int       // messages per second
	burst      int       // burst allowance
	tokens     int       // current tokens
	lastRefill time.Time // last refill time
	mu         sync.Mutex
}

// BackpressureHandler manages flow control
type BackpressureHandler struct {
	config          *BackpressureConfig
	currentPressure float64
	thresholds      *PressureThresholds
	strategies      []BackpressureStrategy
	logger          *logrus.Logger
	mu              sync.RWMutex
}

// ConnectionManager handles connection lifecycle
type ConnectionManager struct {
	config            *ConnectionConfig
	currentConnection *websocket.Conn
	connectionHistory []ConnectionEvent
	connectionPool    *ConnectionPool
	healthMonitor     *ConnectionHealthMonitor
	logger            *logrus.Logger
	mu                sync.RWMutex
}

// ReconnectStrategy manages reconnection logic
type ReconnectStrategy struct {
	config          *ReconnectConfig
	attemptCount    int
	lastAttempt     time.Time
	backoffDuration time.Duration
	strategy        ReconnectStrategyType
	logger          *logrus.Logger
}

// ClientSecurityManager handles client-side security
type ClientSecurityManager struct {
	config        *SecurityConfig
	tokenManager  *TokenManager
	encryptionKey []byte
	validator     SecurityValidator
	audit         *SecurityAudit
	logger        *logrus.Logger
}

// CircularBuffer implements a circular buffer for messages
type CircularBuffer struct {
	buffer   [][]byte
	head     int
	tail     int
	size     int
	capacity int
	mu       sync.RWMutex
}

// MessageBatchBuffer buffers messages for batching
type MessageBatchBuffer struct {
	messages           []EnhancedMessage
	maxSize            int
	timeout            time.Duration
	lastFlush          time.Time
	compressionEnabled bool
	mu                 sync.Mutex
}

// ClientHealthChecker performs health checks on the client
type ClientHealthChecker struct {
	config      *HealthCheckConfig
	checks      []HealthCheck
	lastResults map[string]HealthCheckResult
	logger      *logrus.Logger
	mu          sync.RWMutex
}

// Supporting data structures
type ClientState string

const (
	StateDisconnected ClientState = "disconnected"
	StateConnecting   ClientState = "connecting"
	StateConnected    ClientState = "connected"
	StateReconnecting ClientState = "reconnecting"
	StateError        ClientState = "error"
	StateClosed       ClientState = "closed"
)

type NetworkQuality string

const (
	NetworkExcellent NetworkQuality = "excellent"
	NetworkGood      NetworkQuality = "good"
	NetworkFair      NetworkQuality = "fair"
	NetworkPoor      NetworkQuality = "poor"
)

type ResourceHealth struct {
	MemoryUsage    float64 `json:"memory_usage"`
	CPUUsage       float64 `json:"cpu_usage"`
	BufferUsage    float64 `json:"buffer_usage"`
	GoroutineCount int     `json:"goroutine_count"`
	Status         string  `json:"status"`
}

type HealthIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

type LatencySample struct {
	Timestamp time.Time
	Latency   time.Duration
	MessageID string
}

type EnhancedMessage struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	Priority   Priority               `json:"priority"`
	Compressed bool                   `json:"compressed"`
	Encrypted  bool                   `json:"encrypted"`
	Retry      bool                   `json:"retry"`
	TTL        time.Duration          `json:"ttl"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type MessageFilter interface {
	Filter(message *EnhancedMessage) bool
	GetName() string
}

type MessageTransform interface {
	Transform(message *EnhancedMessage) (*EnhancedMessage, error)
	GetName() string
}

type MessageValidator interface {
	Validate(message *EnhancedMessage) error
	GetRules() []ValidationRule
}

type ValidationRule struct {
	Name        string
	Description string
	Validator   func(*EnhancedMessage) error
}

type BatchProcessor struct {
	config    *BatchConfig
	buffer    []*EnhancedMessage
	timer     *time.Timer
	processor func([]*EnhancedMessage) error
	stats     *BatchProcessorStats
	mu        sync.Mutex
}

type ProcessorStats struct {
	MessagesProcessed   int64         `json:"messages_processed"`
	MessagesFiltered    int64         `json:"messages_filtered"`
	MessagesTransformed int64         `json:"messages_transformed"`
	ValidationErrors    int64         `json:"validation_errors"`
	ProcessingTime      time.Duration `json:"processing_time"`
	AverageTime         time.Duration `json:"average_time"`
}

type BackpressureConfig struct {
	Enabled            bool          `json:"enabled"`
	LowThreshold       float64       `json:"low_threshold"`
	MediumThreshold    float64       `json:"medium_threshold"`
	HighThreshold      float64       `json:"high_threshold"`
	CriticalThreshold  float64       `json:"critical_threshold"`
	ResponseStrategies []string      `json:"response_strategies"`
	MonitoringInterval time.Duration `json:"monitoring_interval"`
}

type PressureThresholds struct {
	Low      float64
	Medium   float64
	High     float64
	Critical float64
}

type BackpressureStrategy interface {
	Apply(pressure float64) error
	GetName() string
	IsApplicable(pressure float64) bool
}

type ConnectionConfig struct {
	MaxConnectionAge  time.Duration `json:"max_connection_age"`
	IdleTimeout       time.Duration `json:"idle_timeout"`
	KeepAliveInterval time.Duration `json:"keep_alive_interval"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
}

type ConnectionEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type ConnectionHealthMonitor struct {
	config  *HealthMonitorConfig
	metrics *ConnectionHealthMetrics
	alerts  []HealthAlert
	logger  *logrus.Logger
}

type ConnectionHealthMetrics struct {
	Latency              time.Duration `json:"latency"`
	PacketLoss           float64       `json:"packet_loss"`
	Jitter               time.Duration `json:"jitter"`
	BandwidthUtilization float64       `json:"bandwidth_utilization"`
	ErrorRate            float64       `json:"error_rate"`
}

type HealthAlert struct {
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

type ReconnectConfig struct {
	MaxAttempts       int           `json:"max_attempts"`
	InitialDelay      time.Duration `json:"initial_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	Jitter            bool          `json:"jitter"`
}

type ReconnectStrategyType string

const (
	ReconnectImmediate   ReconnectStrategyType = "immediate"
	ReconnectLinear      ReconnectStrategyType = "linear"
	ReconnectExponential ReconnectStrategyType = "exponential"
	ReconnectCustom      ReconnectStrategyType = "custom"
)

type SecurityConfig struct {
	EnableEncryption     bool          `json:"enable_encryption"`
	EnableAuthentication bool          `json:"enable_authentication"`
	TokenRefreshInterval time.Duration `json:"token_refresh_interval"`
	MaxTokenAge          time.Duration `json:"max_token_age"`
	EncryptionAlgorithm  string        `json:"encryption_algorithm"`
	EnableAudit          bool          `json:"enable_audit"`
}

type TokenManager struct {
	currentToken string
	refreshToken string
	expiresAt    time.Time
	refreshFunc  func() (string, string, time.Time, error)
	mu           sync.RWMutex
}

type SecurityValidator interface {
	ValidateMessage(message *EnhancedMessage) error
	ValidateConnection() error
}

type SecurityAudit struct {
	events []SecurityEvent
	logger *logrus.Logger
	mu     sync.RWMutex
}

type SecurityEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	ClientID  string                 `json:"client_id"`
	Event     string                 `json:"event"`
	Details   map[string]interface{} `json:"details"`
	Severity  string                 `json:"severity"`
}

type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
}

type HealthCheck interface {
	Check() HealthCheckResult
	GetName() string
	IsEnabled() bool
}

type HealthCheckResult struct {
	Name      string                 `json:"name"`
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type HealthMonitorConfig struct {
	Enabled            bool               `json:"enabled"`
	MonitoringInterval time.Duration      `json:"monitoring_interval"`
	AlertThresholds    map[string]float64 `json:"alert_thresholds"`
	HistorySize        int                `json:"history_size"`
}

type BatchProcessorStats struct {
	BatchesProcessed int64         `json:"batches_processed"`
	MessagesPerBatch float64       `json:"messages_per_batch"`
	AverageBatchSize int           `json:"average_batch_size"`
	ProcessingTime   time.Duration `json:"processing_time"`
	CompressionRatio float64       `json:"compression_ratio"`
}

// NewEnhancedClient creates a new enhanced WebSocket client
func NewEnhancedClient(originalClient *Client, config *EnhancedClientConfig, logger *logrus.Logger) *EnhancedClient {
	if config == nil {
		config = DefaultEnhancedClientConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &EnhancedClient{
		Client:       originalClient,
		config:       config,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		stopChan:     make(chan struct{}),
		state:        StateDisconnected,
		lastActivity: time.Now(),
		metrics:      &EnhancedClientMetrics{StartTime: time.Now()},
		health:       &EnhancedClientHealth{Status: HealthHealthy, Score: 100.0},
	}

	// Initialize components based on configuration
	if config.EnableCompression {
		client.compressionEnabled = true
		client.compressionType = config.CompressionAlgorithm
		client.compressionEngine = NewCompressionEngine(&CompressionConfig{
			Algorithm: config.CompressionAlgorithm,
			Level:     config.CompressionLevel,
			Threshold: config.CompressionThreshold,
		}, logger)
	}

	if config.EnableBatching {
		client.batchingEnabled = true
		client.batchBuffer = NewMessageBatchBuffer(config.BatchSize, config.BatchTimeout)
	}

	client.throughputTracker = NewClientThroughputTracker(100, time.Minute*5)
	client.latencyTracker = NewLatencyTracker(1000)
	client.messageProcessor = NewMessageProcessor(client.compressionEngine, logger)

	if config.SendRateLimit > 0 {
		client.sendLimiter = NewRateLimiter(config.SendRateLimit, config.BurstLimit)
	}

	if config.ReceiveRateLimit > 0 {
		client.receiveLimiter = NewRateLimiter(config.ReceiveRateLimit, config.BurstLimit)
	}

	client.backpressureHandler = NewBackpressureHandler(&BackpressureConfig{
		Enabled:           true,
		LowThreshold:      0.3,
		MediumThreshold:   0.6,
		HighThreshold:     0.8,
		CriticalThreshold: 0.95,
	}, logger)

	client.connectionManager = NewConnectionManager(&ConnectionConfig{
		MaxConnectionAge:  config.MaxConnectionAge,
		IdleTimeout:       time.Minute * 5,
		KeepAliveInterval: config.PingInterval,
	}, logger)

	if config.EnableAutoReconnect {
		client.reconnectStrategy = NewReconnectStrategy(&ReconnectConfig{
			MaxAttempts:       config.MaxReconnectAttempts,
			InitialDelay:      config.ReconnectInterval,
			MaxDelay:          config.MaxBackoffTime,
			BackoffMultiplier: config.BackoffMultiplier,
		}, logger)
	}

	client.securityManager = NewClientSecurityManager(&SecurityConfig{
		EnableEncryption:     config.EnableTLS,
		EnableAuthentication: true,
		TokenRefreshInterval: time.Hour,
	}, logger)

	if config.EnableHealthCheck {
		client.healthChecker = NewClientHealthChecker(&HealthCheckConfig{
			Enabled:  true,
			Interval: config.HealthCheckInterval,
			Timeout:  time.Second * 10,
		}, logger)
	}

	client.messageBuffer = NewCircularBuffer(1000)

	return client
}

// DefaultEnhancedClientConfig returns default configuration
func DefaultEnhancedClientConfig() *EnhancedClientConfig {
	return &EnhancedClientConfig{
		EnableCompression:    true,
		CompressionAlgorithm: CompressionGzip,
		CompressionThreshold: 1024,
		CompressionLevel:     6,

		EnableBatching: true,
		BatchSize:      10,
		BatchTimeout:   time.Millisecond * 100,
		MaxBatchSize:   50,

		SendRateLimit:    100,
		ReceiveRateLimit: 200,
		BurstLimit:       20,

		EnableHealthCheck:    true,
		HealthCheckInterval:  time.Second * 30,
		PingInterval:         time.Second * 30,
		PongTimeout:          time.Second * 60,
		MaxConsecutiveErrors: 5,

		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		MaxMessageSize:  1024 * 1024,
		WriteTimeout:    time.Second * 10,
		ReadTimeout:     time.Second * 60,

		EnableAutoReconnect:  true,
		ReconnectInterval:    time.Second * 5,
		MaxReconnectAttempts: 10,
		BackoffMultiplier:    2.0,
		MaxBackoffTime:       time.Minute * 5,

		EnableTLS:        false,
		VerifyTLS:        true,
		MaxConnectionAge: time.Hour * 24,

		EnableMetrics:   true,
		MetricsInterval: time.Second * 30,
		EnableProfiling: false,
		LogLevel:        "info",
	}
}

// Start starts the enhanced client with all optimizations
func (ec *EnhancedClient) Start() error {
	ec.stateMu.Lock()
	ec.state = StateConnecting
	ec.stateMu.Unlock()

	ec.logger.Info("Starting enhanced WebSocket client...")

	// Start health checker
	if ec.healthChecker != nil {
		go ec.healthChecker.Start(ec.ctx)
	}

	// Start metrics collection
	if ec.config.EnableMetrics {
		go ec.collectMetrics()
	}

	// Start connection monitoring
	go ec.monitorConnection()

	// Start message processing
	go ec.processMessages()

	// Start the original client pumps
	go ec.Client.writePump()

	ec.stateMu.Lock()
	ec.state = StateConnected
	ec.stateMu.Unlock()

	ec.logger.Info("Enhanced WebSocket client started successfully")
	return nil
}

// Stop gracefully stops the enhanced client
func (ec *EnhancedClient) Stop() error {
	ec.logger.Info("Stopping enhanced WebSocket client...")

	ec.stateMu.Lock()
	ec.state = StateClosed
	ec.stateMu.Unlock()

	// Cancel context to signal shutdown
	ec.cancel()

	// Close stop channel
	close(ec.stopChan)

	ec.logger.Info("Enhanced WebSocket client stopped successfully")
	return nil
}

// SendEnhancedMessage sends a message with optimization features
func (ec *EnhancedClient) SendEnhancedMessage(message *EnhancedMessage) error {
	// Check rate limit
	if ec.sendLimiter != nil {
		if !ec.sendLimiter.Allow() {
			return fmt.Errorf("rate limit exceeded")
		}
	}

	// Check backpressure
	if pressure := ec.backpressureHandler.GetCurrentPressure(); pressure > 0.8 {
		return fmt.Errorf("backpressure too high: %.2f", pressure)
	}

	// Process message
	processedMsg, err := ec.messageProcessor.Process(message)
	if err != nil {
		atomic.AddInt64(&ec.metrics.ErrorCount, 1)
		return fmt.Errorf("message processing failed: %w", err)
	}

	// Handle batching
	if ec.batchingEnabled {
		return ec.addToBatch(processedMsg)
	}

	// Send immediately
	return ec.sendMessage(processedMsg)
}

// GetMetrics returns current client metrics
func (ec *EnhancedClient) GetMetrics() *EnhancedClientMetrics {
	ec.updateMetrics()
	return ec.metrics
}

// GetHealth returns current client health status
func (ec *EnhancedClient) GetHealth() *EnhancedClientHealth {
	ec.updateHealth()
	return ec.health
}

// GetState returns current client state
func (ec *EnhancedClient) GetState() ClientState {
	ec.stateMu.RLock()
	defer ec.stateMu.RUnlock()
	return ec.state
}

// Private methods for internal operations
func (ec *EnhancedClient) collectMetrics() {
	ticker := time.NewTicker(ec.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ticker.C:
			ec.updateMetrics()
		}
	}
}

func (ec *EnhancedClient) updateMetrics() {
	now := time.Now()

	// Update connection uptime
	ec.metrics.ConnectionUptime = now.Sub(ec.metrics.StartTime)

	// Update throughput
	if ec.throughputTracker != nil {
		ec.metrics.ThroughputMbps = ec.throughputTracker.GetThroughputMbps()
		ec.metrics.MessageRate = ec.throughputTracker.GetMessageRate()
	}

	// Update latency
	if ec.latencyTracker != nil {
		ec.metrics.AverageLatency = ec.latencyTracker.GetAverageLatency()
		ec.metrics.P95Latency = ec.latencyTracker.GetPercentileLatency(0.95)
		ec.metrics.P99Latency = ec.latencyTracker.GetPercentileLatency(0.99)
	}

	// Update message size
	if ec.metrics.MessagesSent > 0 {
		ec.metrics.AverageMessageSize = float64(ec.metrics.BytesSent) / float64(ec.metrics.MessagesSent)
	}

	// Update buffer utilization
	if ec.messageBuffer != nil {
		ec.metrics.BufferUtilization = ec.messageBuffer.GetUtilization()
	}

	ec.metrics.LastUpdateTime = now
}

func (ec *EnhancedClient) updateHealth() {
	now := time.Now()

	// Calculate health score based on various factors
	score := 100.0

	// Deduct for errors
	if ec.health.ConsecutiveErrors > 0 {
		score -= float64(ec.health.ConsecutiveErrors) * 10
	}

	// Deduct for high latency
	if ec.health.PingLatency > time.Millisecond*100 {
		score -= 20
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}

	ec.health.Score = score

	// Determine status based on score
	if score >= 80 {
		ec.health.Status = HealthHealthy
	} else if score >= 60 {
		ec.health.Status = HealthWarning
	} else {
		ec.health.Status = HealthCritical
	}
}

func (ec *EnhancedClient) monitorConnection() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ticker.C:
			ec.performHealthCheck()
		}
	}
}

func (ec *EnhancedClient) performHealthCheck() {
	// Send ping and measure response time
	start := time.Now()

	pingMsg := &EnhancedMessage{
		ID:        fmt.Sprintf("ping_%d", time.Now().UnixNano()),
		Type:      "ping",
		Data:      map[string]interface{}{"timestamp": start.Unix()},
		Timestamp: start,
		Priority:  PriorityHigh,
	}

	if err := ec.sendMessage(pingMsg); err != nil {
		ec.health.ConsecutiveErrors++
		ec.health.LastError = err.Error()
		ec.health.LastErrorTime = time.Now()
	} else {
		ec.health.LastPing = start
		if ec.health.ConsecutiveErrors > 0 {
			ec.health.ConsecutiveErrors = 0
		}
	}
}

func (ec *EnhancedClient) processMessages() {
	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ec.stopChan:
			return
		default:
			// Process any pending messages
			ec.processPendingMessages()
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func (ec *EnhancedClient) processPendingMessages() {
	// Implementation would process messages from buffer
}

func (ec *EnhancedClient) addToBatch(message *EnhancedMessage) error {
	if ec.batchBuffer == nil {
		return ec.sendMessage(message)
	}

	return ec.batchBuffer.Add(message)
}

func (ec *EnhancedClient) sendMessage(message *EnhancedMessage) error {
	// Convert to JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Apply compression if enabled
	if ec.compressionEnabled && len(data) > ec.config.CompressionThreshold {
		if compressedData, err := ec.compressionEngine.CompressMessage(data); err == nil {
			data = compressedData
			atomic.AddInt64(&ec.metrics.CompressedMessages, 1)
		}
	}

	// Send via original client
	select {
	case ec.Client.send <- data:
		atomic.AddInt64(&ec.metrics.MessagesSent, 1)
		atomic.AddInt64(&ec.metrics.BytesSent, int64(len(data)))
		ec.lastActivity = time.Now()
		return nil
	default:
		return fmt.Errorf("send channel is full")
	}
}

// Component factory functions
func NewClientThroughputTracker(maxSamples int, windowDuration time.Duration) *ClientThroughputTracker {
	return &ClientThroughputTracker{
		samples:        make([]ThroughputSample, 0, maxSamples),
		maxSamples:     maxSamples,
		windowDuration: windowDuration,
	}
}

func (ctt *ClientThroughputTracker) AddSample(bytes, messages int64) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()

	sample := ThroughputSample{
		Timestamp: time.Now(),
		Bytes:     bytes,
		Messages:  messages,
	}

	ctt.samples = append(ctt.samples, sample)

	// Remove old samples
	cutoff := time.Now().Add(-ctt.windowDuration)
	for i, s := range ctt.samples {
		if s.Timestamp.After(cutoff) {
			ctt.samples = ctt.samples[i:]
			break
		}
	}

	// Limit samples
	if len(ctt.samples) > ctt.maxSamples {
		ctt.samples = ctt.samples[len(ctt.samples)-ctt.maxSamples:]
	}
}

func (ctt *ClientThroughputTracker) GetThroughputMbps() float64 {
	ctt.mu.RLock()
	defer ctt.mu.RUnlock()

	if len(ctt.samples) < 2 {
		return 0
	}

	totalBytes := int64(0)
	start := ctt.samples[0].Timestamp
	end := ctt.samples[len(ctt.samples)-1].Timestamp

	for _, sample := range ctt.samples {
		totalBytes += sample.Bytes
	}

	duration := end.Sub(start).Seconds()
	if duration <= 0 {
		return 0
	}

	bitsPerSecond := float64(totalBytes*8) / duration
	return bitsPerSecond / 1024 / 1024 // Convert to Mbps
}

func (ctt *ClientThroughputTracker) GetMessageRate() float64 {
	ctt.mu.RLock()
	defer ctt.mu.RUnlock()

	if len(ctt.samples) < 2 {
		return 0
	}

	totalMessages := int64(0)
	start := ctt.samples[0].Timestamp
	end := ctt.samples[len(ctt.samples)-1].Timestamp

	for _, sample := range ctt.samples {
		totalMessages += sample.Messages
	}

	duration := end.Sub(start).Seconds()
	if duration <= 0 {
		return 0
	}

	return float64(totalMessages) / duration
}

func NewLatencyTracker(maxSamples int) *LatencyTracker {
	return &LatencyTracker{
		samples:    make([]LatencySample, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

func (lt *LatencyTracker) AddSample(messageID string, latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	sample := LatencySample{
		Timestamp: time.Now(),
		Latency:   latency,
		MessageID: messageID,
	}

	lt.samples = append(lt.samples, sample)

	if len(lt.samples) > lt.maxSamples {
		lt.samples = lt.samples[1:]
	}
}

func (lt *LatencyTracker) GetAverageLatency() time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.samples) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, sample := range lt.samples {
		total += sample.Latency
	}

	return total / time.Duration(len(lt.samples))
}

func (lt *LatencyTracker) GetPercentileLatency(percentile float64) time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.samples) == 0 {
		return 0
	}

	// Sort by latency
	latencies := make([]time.Duration, len(lt.samples))
	for i, sample := range lt.samples {
		latencies[i] = sample.Latency
	}

	// Simple percentile calculation
	index := int(float64(len(latencies)) * percentile)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

func NewMessageProcessor(compressionEngine *CompressionEngine, logger *logrus.Logger) *MessageProcessor {
	return &MessageProcessor{
		compressionEngine: compressionEngine,
		stats:             &ProcessorStats{},
		logger:            logger,
	}
}

func (mp *MessageProcessor) Process(message *EnhancedMessage) (*EnhancedMessage, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		mp.stats.ProcessingTime += duration
		atomic.AddInt64(&mp.stats.MessagesProcessed, 1)

		if mp.stats.MessagesProcessed > 0 {
			mp.stats.AverageTime = mp.stats.ProcessingTime / time.Duration(mp.stats.MessagesProcessed)
		}
	}()

	// Apply filters
	for _, filter := range mp.filterChain {
		if !filter.Filter(message) {
			atomic.AddInt64(&mp.stats.MessagesFiltered, 1)
			return nil, fmt.Errorf("message filtered by %s", filter.GetName())
		}
	}

	// Apply transformations
	processedMessage := message
	for _, transform := range mp.transformChain {
		var err error
		processedMessage, err = transform.Transform(processedMessage)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
		atomic.AddInt64(&mp.stats.MessagesTransformed, 1)
	}

	// Validate
	if mp.validator != nil {
		if err := mp.validator.Validate(processedMessage); err != nil {
			atomic.AddInt64(&mp.stats.ValidationErrors, 1)
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return processedMessage, nil
}

func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     burst,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	// Add tokens based on elapsed time
	tokensToAdd := int(elapsed * float64(rl.rate))
	rl.tokens += tokensToAdd

	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}

	rl.lastRefill = now

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func NewBackpressureHandler(config *BackpressureConfig, logger *logrus.Logger) *BackpressureHandler {
	return &BackpressureHandler{
		config: config,
		thresholds: &PressureThresholds{
			Low:      config.LowThreshold,
			Medium:   config.MediumThreshold,
			High:     config.HighThreshold,
			Critical: config.CriticalThreshold,
		},
		logger: logger,
	}
}

func (bh *BackpressureHandler) GetCurrentPressure() float64 {
	bh.mu.RLock()
	defer bh.mu.RUnlock()
	return bh.currentPressure
}

func (bh *BackpressureHandler) UpdatePressure(pressure float64) {
	bh.mu.Lock()
	defer bh.mu.Unlock()

	oldPressure := bh.currentPressure
	bh.currentPressure = pressure

	// Apply strategies if pressure changed significantly
	if pressure > oldPressure+0.1 || pressure < oldPressure-0.1 {
		for _, strategy := range bh.strategies {
			if strategy.IsApplicable(pressure) {
				strategy.Apply(pressure)
			}
		}
	}
}

func NewConnectionManager(config *ConnectionConfig, logger *logrus.Logger) *ConnectionManager {
	return &ConnectionManager{
		config:            config,
		connectionHistory: make([]ConnectionEvent, 0),
		logger:            logger,
	}
}

func NewReconnectStrategy(config *ReconnectConfig, logger *logrus.Logger) *ReconnectStrategy {
	return &ReconnectStrategy{
		config:          config,
		strategy:        ReconnectExponential,
		backoffDuration: config.InitialDelay,
		logger:          logger,
	}
}

func NewClientSecurityManager(config *SecurityConfig, logger *logrus.Logger) *ClientSecurityManager {
	return &ClientSecurityManager{
		config: config,
		audit:  &SecurityAudit{events: make([]SecurityEvent, 0)},
		logger: logger,
	}
}

func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		buffer:   make([][]byte, capacity),
		capacity: capacity,
	}
}

func (cb *CircularBuffer) GetUtilization() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.capacity == 0 {
		return 0
	}

	return float64(cb.size) / float64(cb.capacity)
}

func NewMessageBatchBuffer(maxSize int, timeout time.Duration) *MessageBatchBuffer {
	return &MessageBatchBuffer{
		messages:  make([]EnhancedMessage, 0, maxSize),
		maxSize:   maxSize,
		timeout:   timeout,
		lastFlush: time.Now(),
	}
}

func (mbb *MessageBatchBuffer) Add(message *EnhancedMessage) error {
	mbb.mu.Lock()
	defer mbb.mu.Unlock()

	mbb.messages = append(mbb.messages, *message)

	// Check if batch should be flushed
	if len(mbb.messages) >= mbb.maxSize || time.Since(mbb.lastFlush) >= mbb.timeout {
		return mbb.flush()
	}

	return nil
}

func (mbb *MessageBatchBuffer) flush() error {
	// Implementation would flush the batch
	mbb.messages = mbb.messages[:0]
	mbb.lastFlush = time.Now()
	return nil
}

func NewClientHealthChecker(config *HealthCheckConfig, logger *logrus.Logger) *ClientHealthChecker {
	return &ClientHealthChecker{
		config:      config,
		checks:      make([]HealthCheck, 0),
		lastResults: make(map[string]HealthCheckResult),
		logger:      logger,
	}
}

func (chc *ClientHealthChecker) Start(ctx context.Context) {
	if !chc.config.Enabled {
		return
	}

	ticker := time.NewTicker(chc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			chc.performHealthChecks()
		}
	}
}

func (chc *ClientHealthChecker) performHealthChecks() {
	chc.mu.Lock()
	defer chc.mu.Unlock()

	for _, check := range chc.checks {
		if check.IsEnabled() {
			result := check.Check()
			chc.lastResults[check.GetName()] = result
		}
	}
}
