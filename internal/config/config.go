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
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
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
}
