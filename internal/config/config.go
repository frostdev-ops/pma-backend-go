package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Auth          AuthConfig          `mapstructure:"auth"`
	HomeAssistant HomeAssistantConfig `mapstructure:"home_assistant"`
	Logging       LoggingConfig       `mapstructure:"logging"`
	WebSocket     WebSocketConfig     `mapstructure:"websocket"`
	AI            AIConfig            `mapstructure:"ai"`
	Devices       DevicesConfig       `mapstructure:"devices"`
	Monitoring    MonitoringConfig    `mapstructure:"monitoring"`
	FileManager   FileManagerConfig   `mapstructure:"file_manager"`
	Performance   PerformanceConfig   `mapstructure:"performance"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Path           string `mapstructure:"path"`
	MigrationsPath string `mapstructure:"migrations_path"`
	BackupEnabled  bool   `mapstructure:"backup_enabled"`
	BackupPath     string `mapstructure:"backup_path"`
	MaxConnections int    `mapstructure:"max_connections"`
}

type AuthConfig struct {
	JWTSecret   string `mapstructure:"jwt_secret"`
	TokenExpiry int    `mapstructure:"token_expiry"`
}

type HomeAssistantConfig struct {
	URL   string            `mapstructure:"url"`
	Token string            `mapstructure:"token"`
	Sync  HomeAssistantSync `mapstructure:"sync"`
}

// HomeAssistantSync contains sync service configuration
type HomeAssistantSync struct {
	Enabled              bool     `mapstructure:"enabled"`
	FullSyncInterval     string   `mapstructure:"full_sync_interval"`
	SupportedDomains     []string `mapstructure:"supported_domains"`
	ConflictResolution   string   `mapstructure:"conflict_resolution"`
	BatchSize            int      `mapstructure:"batch_size"`
	RetryAttempts        int      `mapstructure:"retry_attempts"`
	RetryDelay           string   `mapstructure:"retry_delay"`
	EventBufferSize      int      `mapstructure:"event_buffer_size"`
	EventProcessingDelay string   `mapstructure:"event_processing_delay"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type WebSocketConfig struct {
	PingInterval int `mapstructure:"ping_interval"`
	PongTimeout  int `mapstructure:"pong_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`

	// Home Assistant event forwarding configuration
	HomeAssistant WebSocketHAConfig `mapstructure:"homeassistant"`
}

// WebSocketHAConfig contains Home Assistant specific WebSocket configuration
type WebSocketHAConfig struct {
	Enabled              bool     `mapstructure:"enabled"`
	MaxEventsPerSecond   int      `mapstructure:"max_events_per_second"`
	BatchEvents          bool     `mapstructure:"batch_events"`
	BatchWindow          string   `mapstructure:"batch_window"` // Parse to time.Duration
	DefaultSubscriptions []string `mapstructure:"default_subscriptions"`
	ForwardAllEntities   bool     `mapstructure:"forward_all_entities"`
	MaxErrorsRetained    int      `mapstructure:"max_errors_retained"`
}

// AIConfig contains AI/LLM provider configuration
type AIConfig struct {
	Providers       []AIProviderConfig `mapstructure:"providers"`
	FallbackEnabled bool               `mapstructure:"fallback_enabled"`
	FallbackDelay   string             `mapstructure:"fallback_delay"`
	DefaultProvider string             `mapstructure:"default_provider"`
	MaxRetries      int                `mapstructure:"max_retries"`
	Timeout         string             `mapstructure:"timeout"`
}

// AIProviderConfig contains configuration for a specific AI provider
type AIProviderConfig struct {
	Type           string                 `mapstructure:"type"`
	Enabled        bool                   `mapstructure:"enabled"`
	URL            string                 `mapstructure:"url,omitempty"`
	APIKey         string                 `mapstructure:"api_key,omitempty"`
	DefaultModel   string                 `mapstructure:"default_model"`
	MaxTokens      int                    `mapstructure:"max_tokens,omitempty"`
	AutoStart      bool                   `mapstructure:"auto_start,omitempty"`
	ResourceLimits AIResourceLimits       `mapstructure:"resource_limits,omitempty"`
	Models         []string               `mapstructure:"models,omitempty"`
	Priority       int                    `mapstructure:"priority"`
	Extra          map[string]interface{} `mapstructure:"extra,omitempty"`
}

// AIResourceLimits contains resource limits for local providers like Ollama
type AIResourceLimits struct {
	MaxMemory string `mapstructure:"max_memory"`
	MaxCPU    int    `mapstructure:"max_cpu"`
	MaxGPU    int    `mapstructure:"max_gpu"`
}

// DevicesConfig contains device integration configuration
type DevicesConfig struct {
	HealthCheckInterval string       `mapstructure:"health_check_interval"`
	Ring                RingConfig   `mapstructure:"ring"`
	Shelly              ShellyConfig `mapstructure:"shelly"`
	UPS                 UPSConfig    `mapstructure:"ups"`
}

// RingConfig contains Ring integration configuration
type RingConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Email         string `mapstructure:"email"`
	Password      string `mapstructure:"password"`
	PollInterval  string `mapstructure:"poll_interval"`
	EventInterval string `mapstructure:"event_interval"`
	EventLimit    int    `mapstructure:"event_limit"`
	AutoReconnect bool   `mapstructure:"auto_reconnect"`
}

// ShellyConfig contains Shelly integration configuration
type ShellyConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	DiscoveryInterval string `mapstructure:"discovery_interval"`
	Username          string `mapstructure:"username"`
	Password          string `mapstructure:"password"`
	DiscoveryTimeout  string `mapstructure:"discovery_timeout"`
}

// UPSConfig contains UPS integration configuration
type UPSConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	NUTServers   []string `mapstructure:"nut_servers"`
	PollInterval string   `mapstructure:"poll_interval"`
}

// MonitoringConfig contains monitoring and metrics configuration
type MonitoringConfig struct {
	Enabled          bool                        `mapstructure:"enabled"`
	MetricsRetention string                      `mapstructure:"metrics_retention"`
	SnapshotInterval string                      `mapstructure:"snapshot_interval"`
	Alerts           MonitoringAlertsConfig      `mapstructure:"alerts"`
	Prometheus       MonitoringPrometheusConfig  `mapstructure:"prometheus"`
	Performance      MonitoringPerformanceConfig `mapstructure:"performance"`
}

// MonitoringAlertsConfig contains alert configuration
type MonitoringAlertsConfig struct {
	Enabled    bool                             `mapstructure:"enabled"`
	Thresholds MonitoringAlertsThresholdsConfig `mapstructure:"thresholds"`
}

// MonitoringAlertsThresholdsConfig contains alert threshold configuration
type MonitoringAlertsThresholdsConfig struct {
	CPUPercent    float64 `mapstructure:"cpu_percent"`
	MemoryPercent float64 `mapstructure:"memory_percent"`
	DiskPercent   float64 `mapstructure:"disk_percent"`
	ErrorRate     float64 `mapstructure:"error_rate"`
}

// MonitoringPrometheusConfig contains Prometheus configuration
type MonitoringPrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// MonitoringPerformanceConfig contains performance monitoring configuration
type MonitoringPerformanceConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	CalculateP99    bool `mapstructure:"calculate_p99"`
	CalculateP95    bool `mapstructure:"calculate_p95"`
	CalculateP50    bool `mapstructure:"calculate_p50"`
	TrackUserAgents bool `mapstructure:"track_user_agents"`
}

// FileManagerConfig contains file management configuration
type FileManagerConfig struct {
	Storage FileStorageConfig `mapstructure:"storage"`
	Media   FileMediaConfig   `mapstructure:"media"`
	Backup  FileBackupConfig  `mapstructure:"backup"`
	Logs    FileLogsConfig    `mapstructure:"logs"`
}

// FileStorageConfig contains storage configuration
type FileStorageConfig struct {
	BasePath    string `mapstructure:"base_path"`
	MaxFileSize int64  `mapstructure:"max_file_size"`
	TotalQuota  int64  `mapstructure:"total_quota"`
	TempPath    string `mapstructure:"temp_path"`
}

// FileMediaConfig contains media processing configuration
type FileMediaConfig struct {
	EnableStreaming   bool     `mapstructure:"enable_streaming"`
	ThumbnailSizes    []int    `mapstructure:"thumbnail_sizes"`
	TranscodeProfiles []string `mapstructure:"transcode_profiles"`
	CachePath         string   `mapstructure:"cache_path"`
}

// FileBackupConfig contains backup configuration
type FileBackupConfig struct {
	AutoBackup      bool   `mapstructure:"auto_backup"`
	RetentionDays   int    `mapstructure:"retention_days"`
	MaxBackups      int    `mapstructure:"max_backups"`
	CompressBackups bool   `mapstructure:"compress_backups"`
	BackupPath      string `mapstructure:"backup_path"`
}

// FileLogsConfig contains log management configuration
type FileLogsConfig struct {
	RetentionDays    int   `mapstructure:"retention_days"`
	MaxLogSize       int64 `mapstructure:"max_log_size"`
	RotationEnabled  bool  `mapstructure:"rotation_enabled"`
	CompressionLevel int   `mapstructure:"compression_level"`
}

// PerformanceConfig contains performance optimization configuration
type PerformanceConfig struct {
	Database  DatabasePerformanceConfig `mapstructure:"database"`
	Memory    MemoryPerformanceConfig   `mapstructure:"memory"`
	API       APIPerformanceConfig      `mapstructure:"api"`
	WebSocket WSPerformanceConfig       `mapstructure:"websocket"`
}

// DatabasePerformanceConfig contains database performance settings
type DatabasePerformanceConfig struct {
	MaxConnections   int    `mapstructure:"max_connections"`
	MaxIdleConns     int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime  string `mapstructure:"conn_max_lifetime"`
	QueryTimeout     string `mapstructure:"query_timeout"`
	EnableQueryCache bool   `mapstructure:"enable_query_cache"`
	CacheTTL         string `mapstructure:"cache_ttl"`
}

// MemoryPerformanceConfig contains memory management settings
type MemoryPerformanceConfig struct {
	GCTarget      int            `mapstructure:"gc_target"`
	HeapLimit     int64          `mapstructure:"heap_limit"`
	EnablePooling bool           `mapstructure:"enable_pooling"`
	PoolSizes     map[string]int `mapstructure:"pool_sizes"`
}

// APIPerformanceConfig contains API performance settings
type APIPerformanceConfig struct {
	EnableResponseCache bool   `mapstructure:"enable_response_cache"`
	CacheTTL            string `mapstructure:"cache_ttl"`
	EnableCompression   bool   `mapstructure:"enable_compression"`
	MaxRequestSize      int64  `mapstructure:"max_request_size"`
}

// WSPerformanceConfig contains WebSocket performance settings
type WSPerformanceConfig struct {
	WriteBufferSize   int  `mapstructure:"write_buffer_size"`
	ReadBufferSize    int  `mapstructure:"read_buffer_size"`
	EnableCompression bool `mapstructure:"enable_compression"`
	MessageQueueSize  int  `mapstructure:"message_queue_size"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()

	// Override specific values from env
	viper.BindEnv("auth.jwt_secret", "JWT_SECRET")
	viper.BindEnv("home_assistant.token", "HOME_ASSISTANT_TOKEN")
	viper.BindEnv("server.port", "PORT")
	viper.BindEnv("database.path", "DATABASE_PATH")
	viper.BindEnv("logging.level", "LOG_LEVEL")

	// AI environment bindings
	viper.BindEnv("ai.providers.0.api_key", "OPENAI_API_KEY")
	viper.BindEnv("ai.providers.1.api_key", "CLAUDE_API_KEY")
	viper.BindEnv("ai.providers.2.api_key", "GEMINI_API_KEY")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 3001)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "development")

	// Database defaults
	viper.SetDefault("database.path", "./data/pma.db")
	viper.SetDefault("database.migrations_path", "./migrations")
	viper.SetDefault("database.backup_enabled", true)
	viper.SetDefault("database.backup_path", "./data/backups")
	viper.SetDefault("database.max_connections", 25)

	// Auth defaults
	viper.SetDefault("auth.token_expiry", 3600)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// WebSocket defaults
	viper.SetDefault("websocket.ping_interval", 30)
	viper.SetDefault("websocket.pong_timeout", 60)
	viper.SetDefault("websocket.write_timeout", 10)

	// WebSocket Home Assistant defaults
	viper.SetDefault("websocket.homeassistant.enabled", true)
	viper.SetDefault("websocket.homeassistant.max_events_per_second", 50)
	viper.SetDefault("websocket.homeassistant.batch_events", true)
	viper.SetDefault("websocket.homeassistant.batch_window", "100ms")
	viper.SetDefault("websocket.homeassistant.default_subscriptions", []string{
		"ha_state_changed",
		"ha_sync_status",
	})
	viper.SetDefault("websocket.homeassistant.forward_all_entities", false)
	viper.SetDefault("websocket.homeassistant.max_errors_retained", 100)

	// AI defaults
	viper.SetDefault("ai.fallback_enabled", true)
	viper.SetDefault("ai.fallback_delay", "2s")
	viper.SetDefault("ai.default_provider", "ollama")
	viper.SetDefault("ai.max_retries", 3)
	viper.SetDefault("ai.timeout", "30s")

	// Default AI providers
	viper.SetDefault("ai.providers", []map[string]interface{}{
		{
			"type":          "ollama",
			"enabled":       true,
			"url":           "http://localhost:11434",
			"default_model": "llama2",
			"auto_start":    true,
			"priority":      1,
			"resource_limits": map[string]interface{}{
				"max_memory": "4GB",
				"max_cpu":    80,
			},
		},
		{
			"type":          "openai",
			"enabled":       false,
			"default_model": "gpt-3.5-turbo",
			"max_tokens":    4096,
			"priority":      2,
		},
		{
			"type":          "claude",
			"enabled":       false,
			"default_model": "claude-3-haiku-20240307",
			"max_tokens":    4096,
			"priority":      3,
		},
		{
			"type":          "gemini",
			"enabled":       false,
			"default_model": "gemini-pro",
			"max_tokens":    4096,
			"priority":      4,
		},
	})

	// Device defaults
	viper.SetDefault("devices.health_check_interval", "30s")

	// Ring defaults
	viper.SetDefault("devices.ring.enabled", false)
	viper.SetDefault("devices.ring.poll_interval", "5m")
	viper.SetDefault("devices.ring.event_interval", "30s")
	viper.SetDefault("devices.ring.event_limit", 20)
	viper.SetDefault("devices.ring.auto_reconnect", true)

	// Shelly defaults
	viper.SetDefault("devices.shelly.enabled", false)
	viper.SetDefault("devices.shelly.discovery_interval", "5m")
	viper.SetDefault("devices.shelly.username", "admin")
	viper.SetDefault("devices.shelly.discovery_timeout", "30s")

	// UPS defaults
	viper.SetDefault("devices.ups.enabled", false)
	viper.SetDefault("devices.ups.nut_servers", []string{"localhost:3493"})
	viper.SetDefault("devices.ups.poll_interval", "30s")

	// Monitoring defaults
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.metrics_retention", "24h")
	viper.SetDefault("monitoring.snapshot_interval", "30s")

	// Monitoring alerts defaults
	viper.SetDefault("monitoring.alerts.enabled", true)
	viper.SetDefault("monitoring.alerts.thresholds.cpu_percent", 80.0)
	viper.SetDefault("monitoring.alerts.thresholds.memory_percent", 85.0)
	viper.SetDefault("monitoring.alerts.thresholds.disk_percent", 90.0)
	viper.SetDefault("monitoring.alerts.thresholds.error_rate", 0.05)

	// Monitoring Prometheus defaults
	viper.SetDefault("monitoring.prometheus.enabled", true)
	viper.SetDefault("monitoring.prometheus.path", "/metrics")

	// Monitoring performance defaults
	viper.SetDefault("monitoring.performance.enabled", true)
	viper.SetDefault("monitoring.performance.calculate_p99", true)
	viper.SetDefault("monitoring.performance.calculate_p95", true)
	viper.SetDefault("monitoring.performance.calculate_p50", true)
	viper.SetDefault("monitoring.performance.track_user_agents", false)

	// File Manager defaults
	viper.SetDefault("file_manager.storage.base_path", "./data/files")
	viper.SetDefault("file_manager.storage.max_file_size", 1073741824) // 1GB
	viper.SetDefault("file_manager.storage.total_quota", 10737418240)  // 10GB
	viper.SetDefault("file_manager.storage.temp_path", "./data/temp")

	// File Manager Media defaults
	viper.SetDefault("file_manager.media.enable_streaming", true)
	viper.SetDefault("file_manager.media.thumbnail_sizes", []int{150, 300, 600})
	viper.SetDefault("file_manager.media.transcode_profiles", []string{"720p", "480p"})
	viper.SetDefault("file_manager.media.cache_path", "./data/cache")

	// File Manager Backup defaults
	viper.SetDefault("file_manager.backup.auto_backup", true)
	viper.SetDefault("file_manager.backup.retention_days", 30)
	viper.SetDefault("file_manager.backup.max_backups", 10)
	viper.SetDefault("file_manager.backup.compress_backups", true)
	viper.SetDefault("file_manager.backup.backup_path", "./data/backups")

	// File Manager Logs defaults
	viper.SetDefault("file_manager.logs.retention_days", 7)
	viper.SetDefault("file_manager.logs.max_log_size", 104857600) // 100MB
	viper.SetDefault("file_manager.logs.rotation_enabled", true)
	viper.SetDefault("file_manager.logs.compression_level", 6)

	// Performance defaults
	// Database performance
	viper.SetDefault("performance.database.max_connections", 25)
	viper.SetDefault("performance.database.max_idle_conns", 10)
	viper.SetDefault("performance.database.conn_max_lifetime", "1h")
	viper.SetDefault("performance.database.query_timeout", "30s")
	viper.SetDefault("performance.database.enable_query_cache", true)
	viper.SetDefault("performance.database.cache_ttl", "5m")

	// Memory performance
	viper.SetDefault("performance.memory.gc_target", 100)
	viper.SetDefault("performance.memory.heap_limit", 536870912) // 512MB
	viper.SetDefault("performance.memory.enable_pooling", true)
	viper.SetDefault("performance.memory.pool_sizes", map[string]int{
		"buffer":            100,
		"json_response":     200,
		"string_builder":    150,
		"byte_slice_small":  300,
		"byte_slice_medium": 200,
		"byte_slice_large":  100,
	})

	// API performance
	viper.SetDefault("performance.api.enable_response_cache", true)
	viper.SetDefault("performance.api.cache_ttl", "5m")
	viper.SetDefault("performance.api.enable_compression", true)
	viper.SetDefault("performance.api.max_request_size", 10485760) // 10MB

	// WebSocket performance
	viper.SetDefault("performance.websocket.write_buffer_size", 1024)
	viper.SetDefault("performance.websocket.read_buffer_size", 1024)
	viper.SetDefault("performance.websocket.enable_compression", true)
	viper.SetDefault("performance.websocket.message_queue_size", 256)
}
