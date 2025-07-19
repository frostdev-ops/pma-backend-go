package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// EntitySourceType represents the source of an entity
type EntitySourceType string

const (
	EntitySourceHomeAssistant EntitySourceType = "home_assistant"
	EntitySourceShelly        EntitySourceType = "shelly"
	EntitySourceRing          EntitySourceType = "ring"
	EntitySourceDatabase      EntitySourceType = "database"
	EntitySourceCustom        EntitySourceType = "custom"
)

// UnifiedEntity represents an enhanced entity with PMA features
type UnifiedEntity struct {
	// Base entity data
	EntityID     string          `json:"entity_id" db:"entity_id"`
	FriendlyName sql.NullString  `json:"friendly_name" db:"friendly_name"`
	Domain       string          `json:"domain" db:"domain"`
	State        sql.NullString  `json:"state" db:"state"`
	Attributes   json.RawMessage `json:"attributes" db:"attributes"`
	LastUpdated  time.Time       `json:"last_updated" db:"last_updated"`
	RoomID       sql.NullInt64   `json:"room_id" db:"room_id"`

	// PMA enhancements
	SourceType         EntitySourceType `json:"source_type" db:"source_type"`
	SourceID           string           `json:"source_id" db:"source_id"`
	Metadata           json.RawMessage  `json:"metadata" db:"metadata"`
	LastUnified        time.Time        `json:"last_unified" db:"last_unified"`
	SyncStatus         string           `json:"sync_status" db:"sync_status"`
	AvailabilityStatus string           `json:"availability_status" db:"availability_status"`

	// Performance tracking
	ResponseTime  sql.NullFloat64 `json:"response_time" db:"response_time"`
	ErrorCount    int             `json:"error_count" db:"error_count"`
	LastError     sql.NullString  `json:"last_error" db:"last_error"`
	LastErrorTime sql.NullTime    `json:"last_error_time" db:"last_error_time"`

	// Relationships (populated on request)
	Room     *UnifiedRoom `json:"room,omitempty"`
	Device   *Device      `json:"device,omitempty"`
	Category *Category    `json:"category,omitempty"`
}

// UnifiedRoom represents an enhanced room with PMA features
type UnifiedRoom struct {
	// Base room data
	ID                  int            `json:"id" db:"id"`
	Name                string         `json:"name" db:"name"`
	HomeAssistantAreaID sql.NullString `json:"home_assistant_area_id" db:"home_assistant_area_id"`
	Icon                sql.NullString `json:"icon" db:"icon"`
	Description         sql.NullString `json:"description" db:"description"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`

	// PMA enhancements
	EntityCount       int             `json:"entity_count" db:"entity_count"`
	ActiveEntityCount int             `json:"active_entity_count" db:"active_entity_count"`
	LastActivity      sql.NullTime    `json:"last_activity" db:"last_activity"`
	SyncStatus        string          `json:"sync_status" db:"sync_status"`
	Metadata          json.RawMessage `json:"metadata" db:"metadata"`

	// Room statistics
	TemperatureSensor sql.NullString  `json:"temperature_sensor" db:"temperature_sensor"`
	HumiditySensor    sql.NullString  `json:"humidity_sensor" db:"humidity_sensor"`
	CurrentTemp       sql.NullFloat64 `json:"current_temp" db:"current_temp"`
	CurrentHumidity   sql.NullFloat64 `json:"current_humidity" db:"current_humidity"`

	// Energy monitoring
	PowerConsumption sql.NullFloat64 `json:"power_consumption" db:"power_consumption"`
	EnergyToday      sql.NullFloat64 `json:"energy_today" db:"energy_today"`

	// Relationships (populated on request)
	Entities []*UnifiedEntity `json:"entities,omitempty"`
}

// Device represents a physical device that may contain multiple entities
type Device struct {
	ID               string          `json:"id" db:"id"`
	Name             string          `json:"name" db:"name"`
	Manufacturer     sql.NullString  `json:"manufacturer" db:"manufacturer"`
	Model            sql.NullString  `json:"model" db:"model"`
	SWVersion        sql.NullString  `json:"sw_version" db:"sw_version"`
	HWVersion        sql.NullString  `json:"hw_version" db:"hw_version"`
	AreaID           sql.NullString  `json:"area_id" db:"area_id"`
	ConfigEntries    json.RawMessage `json:"config_entries" db:"config_entries"`
	Connections      json.RawMessage `json:"connections" db:"connections"`
	Identifiers      json.RawMessage `json:"identifiers" db:"identifiers"`
	ViaDeviceID      sql.NullString  `json:"via_device_id" db:"via_device_id"`
	DisabledBy       sql.NullString  `json:"disabled_by" db:"disabled_by"`
	ConfigurationURL sql.NullString  `json:"configuration_url" db:"configuration_url"`
	EntryType        sql.NullString  `json:"entry_type" db:"entry_type"`
	LastUpdated      time.Time       `json:"last_updated" db:"last_updated"`
}

// Category represents an entity category for organization
type Category struct {
	ID          int            `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Icon        sql.NullString `json:"icon" db:"icon"`
	Color       sql.NullString `json:"color" db:"color"`
	Description sql.NullString `json:"description" db:"description"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

// EntitySourceMapping tracks entity source relationships
type EntitySourceMapping struct {
	ID         int              `json:"id" db:"id"`
	EntityID   string           `json:"entity_id" db:"entity_id"`
	SourceType EntitySourceType `json:"source_type" db:"source_type"`
	SourceID   string           `json:"source_id" db:"source_id"`
	Metadata   json.RawMessage  `json:"metadata" db:"metadata"`
	CreatedAt  time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at" db:"updated_at"`
}

// PMAStatus represents the overall status of the PMA system
type PMAStatus struct {
	Initialized     bool            `json:"initialized"`
	Healthy         bool            `json:"healthy"`
	LastSyncTime    *time.Time      `json:"last_sync_time"`
	NextSyncTime    *time.Time      `json:"next_sync_time"`
	SyncInterval    int             `json:"sync_interval_minutes"`
	AutoSyncEnabled bool            `json:"auto_sync_enabled"`
	ComponentHealth map[string]bool `json:"component_health"`
	SyncStats       SyncStats       `json:"sync_stats"`
	EntityCounts    EntityCounts    `json:"entity_counts"`
	RoomCounts      RoomCounts      `json:"room_counts"`
	SystemMetrics   SystemMetrics   `json:"system_metrics"`
	LastUpdate      time.Time       `json:"last_update"`
}

// SyncStats represents synchronization statistics
type SyncStats struct {
	TotalSyncs          int     `json:"total_syncs"`
	SuccessfulSyncs     int     `json:"successful_syncs"`
	FailedSyncs         int     `json:"failed_syncs"`
	LastSyncDuration    float64 `json:"last_sync_duration_ms"`
	AverageSyncDuration float64 `json:"average_sync_duration_ms"`
	CurrentlySyncing    bool    `json:"currently_syncing"`
}

// EntityCounts represents entity count statistics
type EntityCounts struct {
	Total       int                      `json:"total"`
	ByDomain    map[string]int           `json:"by_domain"`
	BySource    map[EntitySourceType]int `json:"by_source"`
	ByStatus    map[string]int           `json:"by_status"`
	Available   int                      `json:"available"`
	Unavailable int                      `json:"unavailable"`
	WithErrors  int                      `json:"with_errors"`
}

// RoomCounts represents room count statistics
type RoomCounts struct {
	Total           int `json:"total"`
	WithEntities    int `json:"with_entities"`
	WithoutEntities int `json:"without_entities"`
	Synced          int `json:"synced"`
	NotSynced       int `json:"not_synced"`
}

// SystemMetrics represents system performance metrics
type SystemMetrics struct {
	CacheHitRate    float64 `json:"cache_hit_rate"`
	CacheSize       int     `json:"cache_size"`
	AverageResponse float64 `json:"average_response_time_ms"`
	ErrorRate       float64 `json:"error_rate"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
	MemoryUsageMB   float64 `json:"memory_usage_mb"`
	DatabaseSize    int64   `json:"database_size_bytes"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Entities    int       `json:"entities"`
	Rooms       int       `json:"rooms"`
	Devices     int       `json:"devices"`
	Duration    float64   `json:"duration_ms"`
	DataSource  string    `json:"data_source"`
	Errors      []string  `json:"errors,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Forced      bool      `json:"forced"`
	PartialSync bool      `json:"partial_sync"`
}

// PMASettings represents PMA configuration settings
type PMASettings struct {
	ID                  int       `json:"id" db:"id"`
	AutoSyncEnabled     bool      `json:"auto_sync_enabled" db:"auto_sync_enabled"`
	SyncIntervalMinutes int       `json:"sync_interval_minutes" db:"sync_interval_minutes"`
	CacheEnabled        bool      `json:"cache_enabled" db:"cache_enabled"`
	CacheTTLSeconds     int       `json:"cache_ttl_seconds" db:"cache_ttl_seconds"`
	MaxRetryAttempts    int       `json:"max_retry_attempts" db:"max_retry_attempts"`
	RetryDelayMS        int       `json:"retry_delay_ms" db:"retry_delay_ms"`
	HealthCheckInterval int       `json:"health_check_interval" db:"health_check_interval"`
	MetricsEnabled      bool      `json:"metrics_enabled" db:"metrics_enabled"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// Cache entry for entities and rooms
type CacheEntry struct {
	Key       string      `json:"key"`
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
	HitCount  int         `json:"hit_count"`
	CreatedAt time.Time   `json:"created_at"`
}

// EntityFilter represents filters for entity queries
type EntityFilter struct {
	Domain           *string           `json:"domain,omitempty"`
	RoomID           *int              `json:"room_id,omitempty"`
	SourceType       *EntitySourceType `json:"source_type,omitempty"`
	SyncStatus       *string           `json:"sync_status,omitempty"`
	Available        *bool             `json:"available,omitempty"`
	WithErrors       *bool             `json:"with_errors,omitempty"`
	LastUpdatedSince *time.Time        `json:"last_updated_since,omitempty"`
	IncludeRoom      bool              `json:"include_room,omitempty"`
	IncludeDevice    bool              `json:"include_device,omitempty"`
	IncludeCategory  bool              `json:"include_category,omitempty"`
}

// RoomFilter represents filters for room queries
type RoomFilter struct {
	HasEntities       *bool      `json:"has_entities,omitempty"`
	SyncStatus        *string    `json:"sync_status,omitempty"`
	LastActivitySince *time.Time `json:"last_activity_since,omitempty"`
	IncludeEntities   bool       `json:"include_entities,omitempty"`
	IncludeStats      bool       `json:"include_stats,omitempty"`
}
