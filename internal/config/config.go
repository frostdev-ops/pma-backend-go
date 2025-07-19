package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server           ServerConfig           `mapstructure:"server"`
	Database         DatabaseConfig         `mapstructure:"database"`
	Auth             AuthConfig             `mapstructure:"auth"`
	HomeAssistant    HomeAssistantConfig    `mapstructure:"home_assistant"`
	Logging          LoggingConfig          `mapstructure:"logging"`
	WebSocket        WebSocketConfig        `mapstructure:"websocket"`
	AI               AIConfig               `mapstructure:"ai"`
	Router           RouterConfig           `mapstructure:"router"`
	Devices          DevicesConfig          `mapstructure:"devices"`
	System           SystemConfig           `mapstructure:"system"`
	ExternalServices ExternalServicesConfig `mapstructure:"external_services"`
	Storage          StorageConfig          `mapstructure:"storage"`
	Security         SecurityConfig         `mapstructure:"security"`
	Monitoring       MonitoringConfig       `mapstructure:"monitoring"`
	FileManager      FileManagerConfig      `mapstructure:"file_manager"`
	Performance      PerformanceConfig      `mapstructure:"performance"`

	// Unified Adapter Configuration
	Ring struct {
		Enabled  bool   `mapstructure:"enabled"`
		Email    string `mapstructure:"email"`
		Password string `mapstructure:"password"`
	} `mapstructure:"ring"`

	Shelly struct {
		Enabled bool `mapstructure:"enabled"`
	} `mapstructure:"shelly"`

	UPS struct {
		Enabled bool   `mapstructure:"enabled"`
		Host    string `mapstructure:"host"`
		Port    int    `mapstructure:"port"`
	} `mapstructure:"ups"`

	Network struct {
		Enabled bool `mapstructure:"enabled"`
	} `mapstructure:"network"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Path           string          `mapstructure:"path"`
	MigrationsPath string          `mapstructure:"migrations_path"`
	BackupEnabled  bool            `mapstructure:"backup_enabled"`
	BackupPath     string          `mapstructure:"backup_path"`
	MaxConnections int             `mapstructure:"max_connections"`
	Migration      MigrationConfig `mapstructure:"migration"`
}

type MigrationConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	AutoMigrate bool `mapstructure:"auto_migrate"`
}

type AuthConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
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
	Enabled           bool             `mapstructure:"enabled"`
	DiscoveryInterval string           `mapstructure:"discovery_interval"`
	Username          string           `mapstructure:"username"`
	Password          string           `mapstructure:"password"`
	DiscoveryTimeout  string           `mapstructure:"discovery_timeout"`
	MockDevices       ShellyMockConfig `mapstructure:"mock_devices"`
}

// UPSConfig contains UPS integration configuration
type UPSConfig struct {
	Enabled              bool     `mapstructure:"enabled"`
	NUTHost              string   `mapstructure:"nut_host"`
	NUTPort              int      `mapstructure:"nut_port"`
	UPSName              string   `mapstructure:"ups_name"`
	NUTServers           []string `mapstructure:"nut_servers"`
	PollInterval         string   `mapstructure:"poll_interval"`
	MonitoringInterval   string   `mapstructure:"monitoring_interval"`
	HistoryRetentionDays int      `mapstructure:"history_retention_days"`
}

// RouterConfig contains router/network configuration
type RouterConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	BaseURL           string `mapstructure:"base_url"`
	AuthToken         string `mapstructure:"auth_token"`
	Timeout           string `mapstructure:"timeout"`
	RetryAttempts     int    `mapstructure:"retry_attempts"`
	MonitoringEnabled bool   `mapstructure:"monitoring_enabled"`
	AutoDiscovery     bool   `mapstructure:"auto_discovery"`
	TrafficLogging    bool   `mapstructure:"traffic_logging"`
}

// SystemConfig contains system-wide configuration
type SystemConfig struct {
	Environment           string                         `mapstructure:"environment"`
	DeviceIDFile          string                         `mapstructure:"device_id_file"`
	MaxLogEntries         int                            `mapstructure:"max_log_entries"`
	PerformanceMonitoring bool                           `mapstructure:"performance_monitoring"`
	Services              map[string]SystemServiceConfig `mapstructure:"services"`
}

// SystemServiceConfig contains configuration for system services
type SystemServiceConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Host        string `mapstructure:"host,omitempty"`
	Port        int    `mapstructure:"port,omitempty"`
	Path        string `mapstructure:"path,omitempty"`
	ServiceName string `mapstructure:"service_name,omitempty"`
	DisplayName string `mapstructure:"display_name"`
	Type        string `mapstructure:"type"`
}

// ExternalServicesConfig contains external service endpoint configuration
type ExternalServicesConfig struct {
	IPCheckServices IPCheckServicesConfig `mapstructure:"ip_check_services"`
	Ring            ExternalRingConfig    `mapstructure:"ring"`
	MockData        MockDataConfig        `mapstructure:"mock_data"`
}

// IPCheckServicesConfig contains IP check service endpoints
type IPCheckServicesConfig struct {
	Primary  string `mapstructure:"primary"`
	Fallback string `mapstructure:"fallback"`
	Timeout  string `mapstructure:"timeout"`
}

// ExternalRingConfig contains external Ring API configuration
type ExternalRingConfig struct {
	APIBaseURL string `mapstructure:"api_base_url"`
	OAuthURL   string `mapstructure:"oauth_url"`
}

// MockDataConfig contains mock/testing data configuration
type MockDataConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	RingSnapshotsBase string `mapstructure:"ring_snapshots_base"`
	RingStreamsBase   string `mapstructure:"ring_streams_base"`
}

// StorageConfig contains file storage and path configuration
type StorageConfig struct {
	BasePath     string `mapstructure:"base_path"`
	TempPath     string `mapstructure:"temp_path"`
	BackupPath   string `mapstructure:"backup_path"`
	CachePath    string `mapstructure:"cache_path"`
	LogsPath     string `mapstructure:"logs_path"`
	DeviceIDFile string `mapstructure:"device_id_file"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EnableCORS     bool                    `mapstructure:"enable_cors"`
	AllowedOrigins []string                `mapstructure:"allowed_origins"`
	RateLimiting   SecurityRateLimitConfig `mapstructure:"rate_limiting"`
}

// SecurityRateLimitConfig contains rate limiting configuration
type SecurityRateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
	BurstSize         int  `mapstructure:"burst_size"`
}

// ShellyMockDevice contains mock Shelly device configuration
type ShellyMockDevice struct {
	IP    string `mapstructure:"ip"`
	MAC   string `mapstructure:"mac"`
	Model string `mapstructure:"model"`
	Name  string `mapstructure:"name"`
	Type  string `mapstructure:"type"`
}

// ShellyMockConfig contains Shelly mock device configuration
type ShellyMockConfig struct {
	Enabled bool               `mapstructure:"enabled"`
	Devices []ShellyMockDevice `mapstructure:"devices"`
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

	// Router configuration bindings
	viper.BindEnv("router.base_url", "PMA_ROUTER_BASE_URL")
	viper.BindEnv("router.auth_token", "PMA_ROUTER_AUTH_TOKEN")
	viper.BindEnv("router.enabled", "PMA_ROUTER_ENABLED")

	// Device configuration bindings
	viper.BindEnv("devices.ring.email", "RING_EMAIL")
	viper.BindEnv("devices.ring.password", "RING_PASSWORD")
	viper.BindEnv("devices.ring.enabled", "RING_ENABLED")
	viper.BindEnv("devices.shelly.password", "SHELLY_PASSWORD")
	viper.BindEnv("devices.shelly.enabled", "SHELLY_ENABLED")
	viper.BindEnv("devices.shelly.mock_devices.enabled", "SHELLY_MOCK_ENABLED")
	viper.BindEnv("devices.ups.enabled", "UPS_ENABLED")
	viper.BindEnv("devices.ups.nut_host", "UPS_NUT_HOST")
	viper.BindEnv("devices.ups.nut_port", "UPS_NUT_PORT")
	viper.BindEnv("devices.ups.ups_name", "UPS_NAME")

	// System configuration bindings
	viper.BindEnv("system.environment", "PMA_ENVIRONMENT")
	viper.BindEnv("system.services.home_assistant.host", "HA_HOST")
	viper.BindEnv("system.services.home_assistant.port", "HA_PORT")
	viper.BindEnv("system.services.home_assistant.enabled", "HA_SERVICE_ENABLED")

	// Storage configuration bindings
	viper.BindEnv("storage.base_path", "PMA_STORAGE_BASE_PATH")
	viper.BindEnv("storage.temp_path", "PMA_TEMP_PATH")
	viper.BindEnv("storage.backup_path", "PMA_BACKUP_PATH")
	viper.BindEnv("storage.cache_path", "PMA_CACHE_PATH")
	viper.BindEnv("storage.logs_path", "PMA_LOGS_PATH")

	// Security configuration bindings
	viper.BindEnv("security.allowed_origins", "PMA_ALLOWED_ORIGINS")
	viper.BindEnv("security.enable_cors", "PMA_ENABLE_CORS")
	viper.BindEnv("security.rate_limiting.enabled", "PMA_RATE_LIMITING_ENABLED")

	// External services bindings
	viper.BindEnv("external_services.mock_data.enabled", "PMA_MOCK_DATA_ENABLED")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration for completeness and correctness
func (c *Config) Validate() error {
	var errors []string

	// Validate server configuration
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errors = append(errors, "server.port must be between 1 and 65535")
	}
	if c.Server.Host == "" {
		errors = append(errors, "server.host is required")
	}

	// Validate database configuration
	if c.Database.Path == "" {
		errors = append(errors, "database.path is required")
	}

	// Validate authentication configuration
	if c.Auth.Enabled && (c.Auth.JWTSecret == "" || c.Auth.JWTSecret == "your-secret-key-here") {
		errors = append(errors, "auth.jwt_secret must be set to a secure value when enabled")
	}
	if c.Auth.Enabled && c.Auth.TokenExpiry <= 0 {
		errors = append(errors, "auth.token_expiry must be greater than 0 when enabled")
	}

	// Validate Home Assistant configuration if sync is enabled
	if c.HomeAssistant.Sync.Enabled {
		if c.HomeAssistant.URL == "" {
			errors = append(errors, "home_assistant.url is required when sync is enabled")
		}
		if c.HomeAssistant.Token == "" {
			errors = append(errors, "home_assistant.token is required when sync is enabled")
		}
	}

	// Validate router configuration if enabled
	if c.Router.Enabled {
		if c.Router.BaseURL == "" {
			errors = append(errors, "router.base_url is required when router is enabled")
		}
		if c.Router.RetryAttempts < 0 {
			errors = append(errors, "router.retry_attempts must be non-negative")
		}
	}

	// Validate UPS configuration if enabled
	if c.Devices.UPS.Enabled {
		if c.Devices.UPS.NUTHost == "" {
			errors = append(errors, "devices.ups.nut_host is required when UPS is enabled")
		}
		if c.Devices.UPS.NUTPort <= 0 || c.Devices.UPS.NUTPort > 65535 {
			errors = append(errors, "devices.ups.nut_port must be between 1 and 65535")
		}
		if c.Devices.UPS.UPSName == "" {
			errors = append(errors, "devices.ups.ups_name is required when UPS is enabled")
		}
	}

	// Validate Ring configuration if enabled
	if c.Devices.Ring.Enabled {
		if c.Devices.Ring.Email == "" {
			errors = append(errors, "devices.ring.email is required when Ring is enabled")
		}
		if c.Devices.Ring.Password == "" {
			errors = append(errors, "devices.ring.password is required when Ring is enabled")
		}
	}

	// Validate Shelly configuration if enabled
	if c.Devices.Shelly.Enabled {
		if c.Devices.Shelly.Username == "" {
			errors = append(errors, "devices.shelly.username is required when Shelly is enabled")
		}
	}

	// Validate AI providers
	// hasEnabledProvider := false  // Temporarily disabled for deployment
	for i, provider := range c.AI.Providers {
		if provider.Enabled {
			// hasEnabledProvider = true  // Temporarily disabled for deployment
			if provider.Type == "" {
				errors = append(errors, fmt.Sprintf("ai.providers[%d].type is required", i))
			}
			if provider.Type != "ollama" && provider.APIKey == "" {
				errors = append(errors, fmt.Sprintf("ai.providers[%d].api_key is required for %s", i, provider.Type))
			}
			if provider.Type == "ollama" && provider.URL == "" {
				errors = append(errors, fmt.Sprintf("ai.providers[%d].url is required for Ollama", i))
			}
		}
	}
	// Temporarily disabled for deployment - if !hasEnabledProvider {
	//	errors = append(errors, "at least one AI provider must be enabled")
	// }

	// Validate external services
	if c.ExternalServices.IPCheckServices.Primary == "" {
		errors = append(errors, "external_services.ip_check_services.primary is required")
	}
	if c.ExternalServices.IPCheckServices.Fallback == "" {
		errors = append(errors, "external_services.ip_check_services.fallback is required")
	}

	// Validate storage paths
	if c.Storage.BasePath == "" {
		errors = append(errors, "storage.base_path is required")
	}
	if c.Storage.TempPath == "" {
		errors = append(errors, "storage.temp_path is required")
	}

	// Validate system services configuration
	if len(c.System.Services) == 0 {
		errors = append(errors, "at least one system service should be configured")
	}

	// If there are validation errors, return them
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
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
	viper.SetDefault("database.migration.enabled", true)
	viper.SetDefault("database.migration.auto_migrate", true)

	// Auth defaults
	viper.SetDefault("auth.enabled", true)
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

	// Enhanced Shelly defaults with mock devices
	viper.SetDefault("devices.shelly.mock_devices.enabled", false)
	viper.SetDefault("devices.shelly.mock_devices.devices", []map[string]interface{}{
		{
			"ip":    "192.168.100.50",
			"mac":   "AA:BB:CC:DD:EE:01",
			"model": "SHSW-1PM",
			"name":  "shelly1pm-DDEE01",
			"type":  "switch_pm",
		},
		{
			"ip":    "192.168.100.51",
			"mac":   "AA:BB:CC:DD:EE:02",
			"model": "SHSW-25",
			"name":  "shelly25-DDEE02",
			"type":  "roller",
		},
	})

	// Enhanced UPS defaults (replacing previous simple ones)
	viper.SetDefault("devices.ups.enabled", false)
	viper.SetDefault("devices.ups.nut_host", "localhost")
	viper.SetDefault("devices.ups.nut_port", 3493)
	viper.SetDefault("devices.ups.ups_name", "ups")
	viper.SetDefault("devices.ups.nut_servers", []string{"localhost:3493"})
	viper.SetDefault("devices.ups.poll_interval", "30s")
	viper.SetDefault("devices.ups.monitoring_interval", "30s")
	viper.SetDefault("devices.ups.history_retention_days", 30)

	// Router defaults
	viper.SetDefault("router.enabled", true)
	viper.SetDefault("router.base_url", "http://192.168.100.1:8080")
	viper.SetDefault("router.auth_token", "")
	viper.SetDefault("router.timeout", "30s")
	viper.SetDefault("router.retry_attempts", 3)
	viper.SetDefault("router.monitoring_enabled", true)
	viper.SetDefault("router.auto_discovery", true)
	viper.SetDefault("router.traffic_logging", true)

	// System defaults
	viper.SetDefault("system.environment", "development")
	viper.SetDefault("system.device_id_file", "./data/device_id")
	viper.SetDefault("system.max_log_entries", 1000)
	viper.SetDefault("system.performance_monitoring", true)

	// System services defaults
	viper.SetDefault("system.services.home_assistant.enabled", true)
	viper.SetDefault("system.services.home_assistant.host", "192.168.100.2")
	viper.SetDefault("system.services.home_assistant.port", 8123)
	viper.SetDefault("system.services.home_assistant.display_name", "Home Assistant")
	viper.SetDefault("system.services.home_assistant.type", "network_service")

	viper.SetDefault("system.services.database.enabled", true)
	viper.SetDefault("system.services.database.path", "./data/pma.db")
	viper.SetDefault("system.services.database.display_name", "SQLite Database")
	viper.SetDefault("system.services.database.type", "file_service")

	viper.SetDefault("system.services.websocket.enabled", true)
	viper.SetDefault("system.services.websocket.display_name", "WebSocket Service")
	viper.SetDefault("system.services.websocket.type", "internal_service")

	viper.SetDefault("system.services.backend.enabled", true)
	viper.SetDefault("system.services.backend.service_name", "pma-backend")
	viper.SetDefault("system.services.backend.display_name", "PMA Backend")
	viper.SetDefault("system.services.backend.type", "systemd_service")

	viper.SetDefault("system.services.nginx.enabled", true)
	viper.SetDefault("system.services.nginx.service_name", "nginx")
	viper.SetDefault("system.services.nginx.display_name", "Nginx Web Server")
	viper.SetDefault("system.services.nginx.type", "systemd_service")

	viper.SetDefault("system.services.router.enabled", true)
	viper.SetDefault("system.services.router.service_name", "pma-router")
	viper.SetDefault("system.services.router.display_name", "PMA Router")
	viper.SetDefault("system.services.router.type", "systemd_service")

	// External services defaults
	viper.SetDefault("external_services.ip_check_services.primary", "http://httpbin.org/ip")
	viper.SetDefault("external_services.ip_check_services.fallback", "http://ipv4.icanhazip.com")
	viper.SetDefault("external_services.ip_check_services.timeout", "10s")

	viper.SetDefault("external_services.ring.api_base_url", "https://api.ring.com")
	viper.SetDefault("external_services.ring.oauth_url", "https://oauth.ring.com/oauth/token")

	viper.SetDefault("external_services.mock_data.enabled", false)
	viper.SetDefault("external_services.mock_data.ring_snapshots_base", "https://ring-snapshots.s3.amazonaws.com/mock")
	viper.SetDefault("external_services.mock_data.ring_streams_base", "https://ring-streams.example.com")

	// Storage defaults
	viper.SetDefault("storage.base_path", "./data")
	viper.SetDefault("storage.temp_path", "./data/temp")
	viper.SetDefault("storage.backup_path", "./data/backups")
	viper.SetDefault("storage.cache_path", "./data/cache")
	viper.SetDefault("storage.logs_path", "./logs")
	viper.SetDefault("storage.device_id_file", "./data/device_id")

	// Security defaults
	viper.SetDefault("security.enable_cors", true)
	viper.SetDefault("security.allowed_origins", []string{"*"})
	viper.SetDefault("security.rate_limiting.enabled", true)
	viper.SetDefault("security.rate_limiting.requests_per_minute", 60)
	viper.SetDefault("security.rate_limiting.burst_size", 10)

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
