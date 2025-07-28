package interfaces

import (
	"context"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
)

// EntityServiceInterface defines the interface for entity operations needed by MCP
type EntityServiceInterface interface {
	GetByID(ctx context.Context, entityID string, options EntityGetOptions) (*EntityWithRoom, error)
	GetByRoom(ctx context.Context, roomID string, options EntityGetAllOptions) ([]*EntityWithRoom, error)
	ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error)
}

// RoomServiceInterface defines the interface for room operations needed by MCP
type RoomServiceInterface interface {
	GetRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error)
	GetAllRooms(ctx context.Context) ([]*types.PMARoom, error)
}

// SystemServiceInterface defines the interface for system operations needed by MCP
type SystemServiceInterface interface {
	GetSystemStatus(ctx context.Context) (*SystemStatus, error)
	GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)
}

// EnergyServiceInterface defines the interface for energy operations needed by MCP
type EnergyServiceInterface interface {
	GetCurrentEnergyData(ctx context.Context, deviceID string) (*EnergyData, error)
	GetEnergySettings(ctx context.Context) (*EnergySettings, error)
}

// AutomationServiceInterface defines the interface for automation operations needed by MCP
type AutomationServiceInterface interface {
	AddAutomationRule(ctx context.Context, rule *AutomationRule) (*AutomationResult, error)
	ExecuteScene(ctx context.Context, sceneID string) (*SceneResult, error)
}

// Data structures used by interfaces

type EntityGetOptions struct {
	IncludeRoom bool
	IncludeArea bool
}

type EntityGetAllOptions struct {
	IncludeRoom bool
	IncludeArea bool
	Types       []types.PMAEntityType
	Sources     []types.PMASourceType
}

type EntityWithRoom struct {
	Entity types.PMAEntity `json:"entity"`
	Room   *types.PMARoom  `json:"room,omitempty"`
	Area   *types.PMAArea  `json:"area,omitempty"`
}

type SystemStatus struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	DeviceID    string            `json:"device_id"`
	CPU         *CPUInfo          `json:"cpu,omitempty"`
	Memory      *MemoryInfo       `json:"memory,omitempty"`
	Disk        *DiskInfo         `json:"disk,omitempty"`
	Services    map[string]string `json:"services,omitempty"`
	Uptime      time.Duration     `json:"uptime"`
	SystemLoad  []float64         `json:"system_load,omitempty"`
	NetworkInfo *NetworkInfo      `json:"network,omitempty"`
	Temperature *TemperatureInfo  `json:"temperature,omitempty"`
}

type DeviceInfo struct {
	DeviceID     string    `json:"device_id"`
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	Architecture string    `json:"architecture"`
	KernelInfo   string    `json:"kernel_info"`
	CPUModel     string    `json:"cpu_model"`
	CPUCores     int       `json:"cpu_cores"`
	TotalMemory  uint64    `json:"total_memory"`
	BootTime     time.Time `json:"boot_time"`
	Timezone     string    `json:"timezone"`
}

type CPUInfo struct {
	Usage       float64   `json:"usage"`
	LoadAverage []float64 `json:"load_average"`
	Cores       int       `json:"cores"`
	Model       string    `json:"model"`
	Frequency   float64   `json:"frequency,omitempty"`
}

type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Buffers     uint64  `json:"buffers,omitempty"`
	Cached      uint64  `json:"cached,omitempty"`
}

type DiskInfo struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Filesystem  string  `json:"filesystem"`
	MountPoint  string  `json:"mount_point"`
}

type NetworkInfo struct {
	Interfaces []NetworkInterface `json:"interfaces"`
	PublicIP   string             `json:"public_ip,omitempty"`
}

type NetworkInterface struct {
	Name      string   `json:"name"`
	IsUp      bool     `json:"is_up"`
	Addresses []string `json:"addresses"`
	MTU       int      `json:"mtu"`
	BytesSent uint64   `json:"bytes_sent,omitempty"`
	BytesRecv uint64   `json:"bytes_recv,omitempty"`
}

type TemperatureInfo struct {
	CPUTemp    float64 `json:"cpu_temp,omitempty"`
	GPUTemp    float64 `json:"gpu_temp,omitempty"`
	SystemTemp float64 `json:"system_temp,omitempty"`
	Unit       string  `json:"unit"` // Celsius or Fahrenheit
}

type EnergyData struct {
	Timestamp             time.Time                `json:"timestamp"`
	TotalPowerConsumption float64                  `json:"total_power_consumption"`
	TotalEnergyUsage      float64                  `json:"total_energy_usage"`
	TotalCost             float64                  `json:"total_cost"`
	UPSPowerConsumption   float64                  `json:"ups_power_consumption"`
	DeviceBreakdown       map[string]*DeviceEnergy `json:"device_breakdown,omitempty"`
	EntityID              string                   `json:"entity_id,omitempty"`
	DeviceName            string                   `json:"device_name,omitempty"`
	PowerConsumption      float64                  `json:"power_consumption,omitempty"`
	EnergyUsage           float64                  `json:"energy_usage,omitempty"`
	Cost                  float64                  `json:"cost,omitempty"`
	State                 string                   `json:"state,omitempty"`
	IsOn                  bool                     `json:"is_on,omitempty"`
	Current               float64                  `json:"current,omitempty"`
	Voltage               float64                  `json:"voltage,omitempty"`
	Frequency             float64                  `json:"frequency,omitempty"`
	HasSensors            bool                     `json:"has_sensors,omitempty"`
	SensorsFound          []string                 `json:"sensors_found,omitempty"`
	Percentage            float64                  `json:"percentage,omitempty"`
}

type DeviceEnergy struct {
	DeviceName       string  `json:"device_name"`
	PowerConsumption float64 `json:"power_consumption"`
	EnergyUsage      float64 `json:"energy_usage"`
	Cost             float64 `json:"cost"`
	State            string  `json:"state"`
	IsOn             bool    `json:"is_on"`
	Percentage       float64 `json:"percentage"`
}

type EnergySettings struct {
	EnergyRate       float64 `json:"energy_rate"`
	Currency         string  `json:"currency"`
	TrackingEnabled  bool    `json:"tracking_enabled"`
	UpdateInterval   int     `json:"update_interval"`
	HistoricalPeriod int     `json:"historical_period"`
}

type AutomationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Triggers    []interface{}          `json:"triggers"`
	Actions     []interface{}          `json:"actions"`
	Conditions  []interface{}          `json:"conditions,omitempty"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type AutomationResult struct {
	Success      bool      `json:"success"`
	AutomationID string    `json:"automation_id"`
	Name         string    `json:"name"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
	Note         string    `json:"note"`
}

type SceneResult struct {
	Success    bool      `json:"success"`
	SceneID    string    `json:"scene_id"`
	Message    string    `json:"message"`
	ExecutedAt time.Time `json:"executed_at"`
	Note       string    `json:"note"`
}
