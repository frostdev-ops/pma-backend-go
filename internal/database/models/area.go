package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Area represents an enhanced area in the system
type Area struct {
	ID           int            `json:"id" db:"id"`
	Name         string         `json:"name" db:"name"`
	AreaID       sql.NullString `json:"area_id" db:"area_id"`
	Description  sql.NullString `json:"description" db:"description"`
	Icon         sql.NullString `json:"icon" db:"icon"`
	FloorLevel   int            `json:"floor_level" db:"floor_level"`
	ParentAreaID sql.NullInt64  `json:"parent_area_id" db:"parent_area_id"`
	Color        sql.NullString `json:"color" db:"color"`
	IsActive     bool           `json:"is_active" db:"is_active"`
	AreaType     string         `json:"area_type" db:"area_type"`
	Metadata     sql.NullString `json:"metadata" db:"metadata"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// AreaWithChildren represents an area with its child areas
type AreaWithChildren struct {
	Area
	Children    []*AreaWithChildren `json:"children,omitempty"`
	EntityCount int                 `json:"entity_count"`
	RoomCount   int                 `json:"room_count"`
}

// AreaMapping represents a mapping between PMA areas and external systems
type AreaMapping struct {
	ID             int          `json:"id" db:"id"`
	PMAAreaID      int          `json:"pma_area_id" db:"pma_area_id"`
	ExternalAreaID string       `json:"external_area_id" db:"external_area_id"`
	ExternalSystem string       `json:"external_system" db:"external_system"`
	MappingType    string       `json:"mapping_type" db:"mapping_type"`
	AutoSync       bool         `json:"auto_sync" db:"auto_sync"`
	SyncPriority   int          `json:"sync_priority" db:"sync_priority"`
	LastSynced     sql.NullTime `json:"last_synced" db:"last_synced"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at" db:"updated_at"`
}

// AreaMappingWithDetails includes area and external system details
type AreaMappingWithDetails struct {
	AreaMapping
	AreaName         string `json:"area_name"`
	ExternalAreaName string `json:"external_area_name,omitempty"`
}

// AreaSetting represents configuration settings for areas
type AreaSetting struct {
	ID           int            `json:"id" db:"id"`
	SettingKey   string         `json:"setting_key" db:"setting_key"`
	SettingValue sql.NullString `json:"setting_value" db:"setting_value"`
	AreaID       sql.NullInt64  `json:"area_id" db:"area_id"`
	IsGlobal     bool           `json:"is_global" db:"is_global"`
	DataType     string         `json:"data_type" db:"data_type"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// AreaSettings represents a collection of settings for an area or global
type AreaSettings struct {
	AreaID   *int                   `json:"area_id,omitempty"`
	Settings map[string]interface{} `json:"settings"`
}

// AreaAnalytic represents analytics data for areas
type AreaAnalytic struct {
	ID              int            `json:"id" db:"id"`
	AreaID          int            `json:"area_id" db:"area_id"`
	MetricName      string         `json:"metric_name" db:"metric_name"`
	MetricValue     float64        `json:"metric_value" db:"metric_value"`
	MetricUnit      sql.NullString `json:"metric_unit" db:"metric_unit"`
	AggregationType string         `json:"aggregation_type" db:"aggregation_type"`
	TimePeriod      sql.NullString `json:"time_period" db:"time_period"`
	RecordedAt      time.Time      `json:"recorded_at" db:"recorded_at"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
}

// AreaAnalyticsSummary represents aggregated analytics for an area
type AreaAnalyticsSummary struct {
	AreaID        int                    `json:"area_id"`
	AreaName      string                 `json:"area_name"`
	EntityCount   int                    `json:"entity_count"`
	DeviceCount   int                    `json:"device_count"`
	RoomCount     int                    `json:"room_count"`
	ActiveDevices int                    `json:"active_devices"`
	EnergyUsage   float64                `json:"energy_usage,omitempty"`
	LastActivity  *time.Time             `json:"last_activity,omitempty"`
	HealthScore   float64                `json:"health_score"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// AreaSyncLog represents synchronization history and status
type AreaSyncLog struct {
	ID             int            `json:"id" db:"id"`
	SyncType       string         `json:"sync_type" db:"sync_type"`
	ExternalSystem string         `json:"external_system" db:"external_system"`
	Status         string         `json:"status" db:"status"`
	AreasProcessed int            `json:"areas_processed" db:"areas_processed"`
	AreasUpdated   int            `json:"areas_updated" db:"areas_updated"`
	AreasCreated   int            `json:"areas_created" db:"areas_created"`
	AreasDeleted   int            `json:"areas_deleted" db:"areas_deleted"`
	ErrorMessage   sql.NullString `json:"error_message" db:"error_message"`
	SyncDetails    sql.NullString `json:"sync_details" db:"sync_details"`
	StartedAt      time.Time      `json:"started_at" db:"started_at"`
	CompletedAt    sql.NullTime   `json:"completed_at" db:"completed_at"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// RoomAreaAssignment represents the relationship between rooms and areas
type RoomAreaAssignment struct {
	ID              int       `json:"id" db:"id"`
	RoomID          int       `json:"room_id" db:"room_id"`
	AreaID          int       `json:"area_id" db:"area_id"`
	AssignmentType  string    `json:"assignment_type" db:"assignment_type"`
	ConfidenceScore float64   `json:"confidence_score" db:"confidence_score"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// AreaStatus represents the overall status of the area management system
type AreaStatus struct {
	TotalAreas           int        `json:"total_areas"`
	ActiveAreas          int        `json:"active_areas"`
	MappedAreas          int        `json:"mapped_areas"`
	UnmappedAreas        int        `json:"unmapped_areas"`
	TotalRooms           int        `json:"total_rooms"`
	AssignedRooms        int        `json:"assigned_rooms"`
	UnassignedRooms      int        `json:"unassigned_rooms"`
	TotalEntities        int        `json:"total_entities"`
	EntitiesWithAreas    int        `json:"entities_with_areas"`
	EntitiesWithoutAreas int        `json:"entities_without_areas"`
	LastSyncTime         *time.Time `json:"last_sync_time"`
	SyncStatus           string     `json:"sync_status"`
	IsConnected          bool       `json:"is_connected"`
	SyncEnabled          bool       `json:"sync_enabled"`
	ExternalSystems      []string   `json:"external_systems"`
	HealthScore          float64    `json:"health_score"`
	Issues               []string   `json:"issues,omitempty"`
	Recommendations      []string   `json:"recommendations,omitempty"`
}

// CreateAreaRequest represents a request to create a new area
type CreateAreaRequest struct {
	Name         string                 `json:"name" binding:"required"`
	AreaID       *string                `json:"area_id,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Icon         *string                `json:"icon,omitempty"`
	FloorLevel   *int                   `json:"floor_level,omitempty"`
	ParentAreaID *int                   `json:"parent_area_id,omitempty"`
	Color        *string                `json:"color,omitempty"`
	AreaType     *string                `json:"area_type,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAreaRequest represents a request to update an area
type UpdateAreaRequest struct {
	Name         *string                `json:"name,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Icon         *string                `json:"icon,omitempty"`
	FloorLevel   *int                   `json:"floor_level,omitempty"`
	ParentAreaID *int                   `json:"parent_area_id,omitempty"`
	Color        *string                `json:"color,omitempty"`
	IsActive     *bool                  `json:"is_active,omitempty"`
	AreaType     *string                `json:"area_type,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// CreateAreaMappingRequest represents a request to create an area mapping
type CreateAreaMappingRequest struct {
	PMAAreaID      int    `json:"pma_area_id" binding:"required"`
	ExternalAreaID string `json:"external_area_id" binding:"required"`
	ExternalSystem string `json:"external_system"`
	MappingType    string `json:"mapping_type"`
	AutoSync       *bool  `json:"auto_sync,omitempty"`
	SyncPriority   *int   `json:"sync_priority,omitempty"`
}

// UpdateAreaMappingRequest represents a request to update an area mapping
type UpdateAreaMappingRequest struct {
	MappingType  *string `json:"mapping_type,omitempty"`
	AutoSync     *bool   `json:"auto_sync,omitempty"`
	SyncPriority *int    `json:"sync_priority,omitempty"`
}

// SyncRequest represents a request to trigger synchronization
type SyncRequest struct {
	SyncType       string `json:"sync_type"` // full, incremental, manual
	ExternalSystem string `json:"external_system"`
	ForceSync      bool   `json:"force_sync"`
	AreaIDs        []int  `json:"area_ids,omitempty"` // For selective sync
}

// AreaAnalyticsRequest represents a request for area analytics
type AreaAnalyticsRequest struct {
	AreaIDs    []int      `json:"area_ids,omitempty"`
	Metrics    []string   `json:"metrics,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	TimePeriod string     `json:"time_period,omitempty"` // hourly, daily, weekly, monthly
	Grouping   string     `json:"grouping,omitempty"`    // area, type, floor
}

// AreaHierarchy represents the hierarchical structure of areas
type AreaHierarchy struct {
	Root       []*AreaWithChildren `json:"root"`
	MaxDepth   int                 `json:"max_depth"`
	TotalAreas int                 `json:"total_areas"`
}

// HomeAssistantArea represents a Home Assistant area structure
type HomeAssistantArea struct {
	AreaID  string   `json:"area_id"`
	Name    string   `json:"name"`
	Picture *string  `json:"picture,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
	Labels  []string `json:"labels,omitempty"`
}

// External system integration types
const (
	ExternalSystemHomeAssistant = "homeassistant"
	ExternalSystemOpenHAB       = "openhab"
	ExternalSystemDomoticz      = "domoticz"
)

// Area types
const (
	AreaTypeRoom     = "room"
	AreaTypeZone     = "zone"
	AreaTypeBuilding = "building"
	AreaTypeFloor    = "floor"
	AreaTypeOutdoor  = "outdoor"
	AreaTypeUtility  = "utility"
)

// Mapping types
const (
	MappingTypeDirect  = "direct"
	MappingTypeDerived = "derived"
	MappingTypeManual  = "manual"
)

// Assignment types
const (
	AssignmentTypePrimary   = "primary"
	AssignmentTypeSecondary = "secondary"
	AssignmentTypeInherited = "inherited"
)

// Sync statuses
const (
	SyncStatusPending = "pending"
	SyncStatusRunning = "running"
	SyncStatusSuccess = "success"
	SyncStatusFailed  = "failed"
	SyncStatusPartial = "partial"
)

// Sync types
const (
	SyncTypeFull        = "full"
	SyncTypeIncremental = "incremental"
	SyncTypeManual      = "manual"
)

// Aggregation types
const (
	AggregationSnapshot = "snapshot"
	AggregationSum      = "sum"
	AggregationAvg      = "avg"
	AggregationMax      = "max"
	AggregationMin      = "min"
)

// Helper methods

// GetMetadataAsMap returns the metadata as a map
func (a *Area) GetMetadataAsMap() (map[string]interface{}, error) {
	if !a.Metadata.Valid {
		return nil, nil
	}

	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(a.Metadata.String), &metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// SetMetadataFromMap sets the metadata from a map
func (a *Area) SetMetadataFromMap(metadata map[string]interface{}) error {
	if metadata == nil {
		a.Metadata = sql.NullString{Valid: false}
		return nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	a.Metadata = sql.NullString{
		String: string(data),
		Valid:  true,
	}

	return nil
}

// GetSyncDetailsAsMap returns the sync details as a map
func (asl *AreaSyncLog) GetSyncDetailsAsMap() (map[string]interface{}, error) {
	if !asl.SyncDetails.Valid {
		return nil, nil
	}

	var details map[string]interface{}
	err := json.Unmarshal([]byte(asl.SyncDetails.String), &details)
	if err != nil {
		return nil, err
	}

	return details, nil
}

// SetSyncDetailsFromMap sets the sync details from a map
func (asl *AreaSyncLog) SetSyncDetailsFromMap(details map[string]interface{}) error {
	if details == nil {
		asl.SyncDetails = sql.NullString{Valid: false}
		return nil
	}

	data, err := json.Marshal(details)
	if err != nil {
		return err
	}

	asl.SyncDetails = sql.NullString{
		String: string(data),
		Valid:  true,
	}

	return nil
}

// Duration returns the sync duration
func (asl *AreaSyncLog) Duration() *time.Duration {
	if !asl.CompletedAt.Valid {
		return nil
	}

	duration := asl.CompletedAt.Time.Sub(asl.StartedAt)
	return &duration
}
