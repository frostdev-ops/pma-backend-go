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
}
