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
	Environment     string                 `json:"environment"`
	Debug           bool                   `json:"debug"`
	LogLevel        string                 `json:"log_level"`
	MaxLogEntries   int                    `json:"max_log_entries"`
	Services        map[string]interface{} `json:"services"`
	Features        map[string]bool        `json:"features"`
	UpdateChannel   string                 `json:"update_channel"`
	AutoUpdate      bool                   `json:"auto_update"`
	MaintenanceMode bool                   `json:"maintenance_mode"`
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
