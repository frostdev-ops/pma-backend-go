package system

import (
	"time"
)

// DeviceInfo represents device information
type DeviceInfo struct {
	DeviceID    string                 `json:"device_id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	OS          string                 `json:"os"`
	Arch        string                 `json:"arch"`
	Platform    string                 `json:"platform"`
	Hostname    string                 `json:"hostname"`
	LastSeen    time.Time              `json:"last_seen"`
	Uptime      int64                  `json:"uptime"`
	Environment string                 `json:"environment"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	DeviceID  string         `json:"device_id"`
	Timestamp time.Time      `json:"timestamp"`
	CPU       CPUInfo        `json:"cpu"`
	Memory    MemoryInfo     `json:"memory"`
	Disk      DiskInfo       `json:"disk"`
	Network   NetworkInfo    `json:"network"`
	Services  ServicesHealth `json:"services"`
}

// CPUInfo represents CPU information and usage
type CPUInfo struct {
	Usage       float64   `json:"usage"`
	LoadAverage []float64 `json:"load_average"`
	Temperature *float64  `json:"temperature,omitempty"`
	Cores       int       `json:"cores"`
	Model       string    `json:"model"`
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Free        uint64  `json:"free"`
	Cached      uint64  `json:"cached"`
	Buffers     uint64  `json:"buffers"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Path        string  `json:"path"`
	Filesystem  string  `json:"filesystem"`
}

// NetworkInfo represents network information
type NetworkInfo struct {
	Interfaces   []NetworkInterface  `json:"interfaces"`
	Connectivity NetworkConnectivity `json:"connectivity"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	HardwareAddr string        `json:"hardware_addr"`
	Addresses    []string      `json:"addresses"`
	IsUp         bool          `json:"is_up"`
	IsLoopback   bool          `json:"is_loopback"`
	Speed        int64         `json:"speed,omitempty"`
	MTU          int           `json:"mtu"`
	Stats        *NetworkStats `json:"stats,omitempty"`
}

// NetworkStats represents network interface statistics
type NetworkStats struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Errors      uint64 `json:"errors"`
	Drops       uint64 `json:"drops"`
}

// NetworkConnectivity represents network connectivity status
type NetworkConnectivity struct {
	HasInternet bool     `json:"has_internet"`
	ExternalIP  string   `json:"external_ip,omitempty"`
	DNS         []string `json:"dns"`
	Gateway     string   `json:"gateway,omitempty"`
	Latency     *int64   `json:"latency,omitempty"`
}

// ServicesHealth represents health status of core services
type ServicesHealth struct {
	HomeAssistant ServiceHealth `json:"home_assistant"`
	Database      ServiceHealth `json:"database"`
	WebSocket     ServiceHealth `json:"web_socket"`
}

// ServiceHealth represents health status of a service
type ServiceHealth struct {
	Status       string                 `json:"status"`
	Uptime       int64                  `json:"uptime"`
	LastCheck    time.Time              `json:"last_check"`
	ResponseTime int64                  `json:"response_time"`
	ErrorCount   int                    `json:"error_count"`
	Details      map[string]interface{} `json:"details"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// SystemLogs represents system logs response
type SystemLogs struct {
	Logs       []LogEntry `json:"logs"`
	TotalCount int        `json:"total_count"`
	HasMore    bool       `json:"has_more"`
	LastID     *string    `json:"last_id,omitempty"`
	TimeRange  TimeRange  `json:"time_range"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// LogsRequest represents a request for logs
type LogsRequest struct {
	Limit     int       `json:"limit,omitempty"`
	Level     string    `json:"level,omitempty"`
	Service   string    `json:"service,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	LastID    string    `json:"last_id,omitempty"`
}

// SystemConfig represents system configuration
type SystemConfig struct {
	// Core system configuration
	System SystemSectionConfig `json:"system"`

	// Server configuration
	Server ServerSectionConfig `json:"server"`

	// Database configuration
	Database DatabaseSectionConfig `json:"database"`

	// Authentication configuration
	Auth AuthSectionConfig `json:"auth"`

	// WebSocket configuration
	WebSocket WebSocketSectionConfig `json:"websocket"`

	// External service configurations
	Services ServicesSectionConfig `json:"services"`

	// Monitoring and alerting
	Monitoring MonitoringSectionConfig `json:"monitoring"`

	// Security settings
	Security SecuritySectionConfig `json:"security"`

	// Metadata
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Version   int    `json:"version"`
	Checksum  string `json:"checksum,omitempty"`
}

// SystemSectionConfig represents core system settings
type SystemSectionConfig struct {
	Name                string  `json:"name"`
	Description         string  `json:"description,omitempty"`
	Timezone            string  `json:"timezone"`
	Locale              string  `json:"locale"`
	LogLevel            string  `json:"log_level"`
	DebugMode           bool    `json:"debug_mode"`
	MaintenanceMode     bool    `json:"maintenance_mode"`
	AutoBackup          bool    `json:"auto_backup"`
	BackupRetentionDays int     `json:"backup_retention_days"`
	HubUser             HubUser `json:"hub_user"`
}

// HubUser represents the local hub user configuration
type HubUser struct {
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	Avatar  string `json:"avatar,omitempty"`
	Enabled bool   `json:"enabled"`
}

// ServerSectionConfig represents server configuration
type ServerSectionConfig struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	HTTPSEnabled   bool     `json:"https_enabled"`
	CORSEnabled    bool     `json:"cors_enabled"`
	CORSOrigins    []string `json:"cors_origins"`
	RateLimiting   bool     `json:"rate_limiting"`
	MaxRequestSize int64    `json:"max_request_size"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

// DatabaseSectionConfig represents database configuration
type DatabaseSectionConfig struct {
	Type               string `json:"type"`
	Host               string `json:"host,omitempty"`
	Port               int    `json:"port,omitempty"`
	Name               string `json:"name"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	SSLEnabled         bool   `json:"ssl_enabled,omitempty"`
	ConnectionPoolSize int    `json:"connection_pool_size"`
	MaxIdleConnections int    `json:"max_idle_connections"`
	ConnectionTimeout  int    `json:"connection_timeout"`
}

// AuthSectionConfig represents authentication configuration
type AuthSectionConfig struct {
	Enabled             bool   `json:"enabled"`
	Method              string `json:"method"`
	PinLength           int    `json:"pin_length,omitempty"`
	PasswordMinLength   int    `json:"password_min_length,omitempty"`
	SessionTimeout      int    `json:"session_timeout"`
	JWTSecret           string `json:"jwt_secret"`
	RefreshTokenEnabled bool   `json:"refresh_token_enabled"`
	MaxSessionsPerUser  int    `json:"max_sessions_per_user"`
	LockoutThreshold    int    `json:"lockout_threshold"`
	LockoutDuration     int    `json:"lockout_duration"`
}

// WebSocketSectionConfig represents WebSocket configuration
type WebSocketSectionConfig struct {
	Enabled            bool `json:"enabled"`
	Port               int  `json:"port,omitempty"`
	MaxConnections     int  `json:"max_connections"`
	HeartbeatInterval  int  `json:"heartbeat_interval"`
	MessageBufferSize  int  `json:"message_buffer_size"`
	CompressionEnabled bool `json:"compression_enabled"`
}

// ServicesSectionConfig represents external service configurations
type ServicesSectionConfig struct {
	HomeAssistant *HomeAssistantConfig `json:"homeassistant,omitempty"`
	AI            *AIConfig            `json:"ai,omitempty"`
	Energy        *EnergyConfig        `json:"energy,omitempty"`
	Network       *NetworkConfig       `json:"network,omitempty"`
}

// HomeAssistantConfig represents Home Assistant configuration
type HomeAssistantConfig struct {
	Enabled           bool     `json:"enabled"`
	URL               string   `json:"url"`
	Token             string   `json:"token"`
	VerifySSL         bool     `json:"verify_ssl"`
	Timeout           int      `json:"timeout"`
	ReconnectInterval int      `json:"reconnect_interval"`
	SyncInterval      int      `json:"sync_interval"`
	IncludedDomains   []string `json:"included_domains"`
	ExcludedDomains   []string `json:"excluded_domains"`
}

// AIConfig represents AI service configuration
type AIConfig struct {
	Enabled         string            `json:"enabled"`
	DefaultProvider string            `json:"default_provider"`
	Providers       AIProvidersConfig `json:"providers"`
}

// AIProvidersConfig represents AI provider configurations
type AIProvidersConfig struct {
	Ollama *OllamaConfig `json:"ollama,omitempty"`
	OpenAI *OpenAIConfig `json:"openai,omitempty"`
	Claude *ClaudeConfig `json:"claude,omitempty"`
	Gemini *GeminiConfig `json:"gemini,omitempty"`
}

// OllamaConfig represents Ollama configuration
type OllamaConfig struct {
	Enabled bool     `json:"enabled"`
	URL     string   `json:"url"`
	Models  []string `json:"models"`
	Timeout int      `json:"timeout"`
}

// OpenAIConfig represents OpenAI configuration
type OpenAIConfig struct {
	Enabled     bool    `json:"enabled"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// ClaudeConfig represents Claude configuration
type ClaudeConfig struct {
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

// GeminiConfig represents Gemini configuration
type GeminiConfig struct {
	Enabled        bool              `json:"enabled"`
	APIKey         string            `json:"api_key"`
	Model          string            `json:"model"`
	SafetySettings map[string]string `json:"safety_settings"`
}

// EnergyConfig represents energy monitoring configuration
type EnergyConfig struct {
	Enabled                bool    `json:"enabled"`
	UpdateInterval         int     `json:"update_interval"`
	CostPerKWH             float64 `json:"cost_per_kwh"`
	Currency               string  `json:"currency"`
	RetentionDays          int     `json:"retention_days"`
	TrackIndividualDevices bool    `json:"track_individual_devices"`
}

// NetworkConfig represents network monitoring configuration
type NetworkConfig struct {
	Enabled            bool     `json:"enabled"`
	MonitorInterfaces  []string `json:"monitor_interfaces"`
	ScanInterval       int      `json:"scan_interval"`
	PortScanEnabled    bool     `json:"port_scan_enabled"`
	IntrusionDetection bool     `json:"intrusion_detection"`
}

// MonitoringSectionConfig represents monitoring and alerting configuration
type MonitoringSectionConfig struct {
	Enabled              bool               `json:"enabled"`
	MetricsRetentionDays int                `json:"metrics_retention_days"`
	HealthCheckInterval  int                `json:"health_check_interval"`
	AlertThresholds      AlertThresholds    `json:"alert_thresholds"`
	Notifications        NotificationConfig `json:"notifications"`
}

// AlertThresholds represents alert threshold configuration
type AlertThresholds struct {
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	ResponseTime int     `json:"response_time"`
	ErrorRate    float64 `json:"error_rate"`
}

// NotificationConfig represents notification configuration
type NotificationConfig struct {
	EmailEnabled    bool     `json:"email_enabled"`
	EmailRecipients []string `json:"email_recipients"`
	WebhookEnabled  bool     `json:"webhook_enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	SlackEnabled    bool     `json:"slack_enabled"`
	SlackWebhook    string   `json:"slack_webhook,omitempty"`
}

// SecuritySectionConfig represents security configuration
type SecuritySectionConfig struct {
	EncryptionEnabled     bool     `json:"encryption_enabled"`
	AuditLogging          bool     `json:"audit_logging"`
	FailedLoginTracking   bool     `json:"failed_login_tracking"`
	IPWhitelist           []string `json:"ip_whitelist"`
	IPBlacklist           []string `json:"ip_blacklist"`
	APIKeyRequired        bool     `json:"api_key_required"`
	CSRFProtection        bool     `json:"csrf_protection"`
	ContentSecurityPolicy bool     `json:"content_security_policy"`
}

// PowerAction represents a power management action
type PowerAction struct {
	Action    string `json:"action"` // "reboot" or "shutdown"
	Delay     int    `json:"delay,omitempty"`
	Force     bool   `json:"force,omitempty"`
	Reason    string `json:"reason,omitempty"`
	RequestBy string `json:"request_by,omitempty"`
}

// UpdateInfo represents update information
type UpdateInfo struct {
	Available         bool      `json:"available"`
	CurrentVersion    string    `json:"current_version"`
	LatestVersion     string    `json:"latest_version"`
	ReleaseNotes      string    `json:"release_notes,omitempty"`
	Size              int64     `json:"size,omitempty"`
	LastChecked       time.Time `json:"last_checked"`
	UpdateChannel     string    `json:"update_channel"`
	AutoUpdateEnabled bool      `json:"auto_update_enabled"`
}

// UpdateStatus represents update status
type UpdateStatus struct {
	InProgress          bool       `json:"in_progress"`
	Status              string     `json:"status"`
	Progress            int        `json:"progress"`
	Step                string     `json:"step"`
	Error               string     `json:"error,omitempty"`
	StartedAt           time.Time  `json:"started_at,omitempty"`
	CompletedAt         time.Time  `json:"completed_at,omitempty"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
}

// ServiceConfig represents configuration for a tracked service
type ServiceConfig struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "network_service", "file_service", "systemd_service", "internal_service"
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	// Network service specific
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
	// File service specific
	Path string `json:"path,omitempty"`
	// Systemd service specific
	ServiceName string `json:"service_name,omitempty"`
}
