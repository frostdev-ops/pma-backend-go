package types

import (
	"context"
	"time"
)

// PMAAdapter defines the interface that all source adapters must implement
type PMAAdapter interface {
	// Identification
	GetID() string
	GetSourceType() PMASourceType
	GetName() string
	GetVersion() string

	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	GetStatus() string

	// Entity conversion
	ConvertEntity(sourceEntity interface{}) (PMAEntity, error)
	ConvertEntities(sourceEntities []interface{}) ([]PMAEntity, error)

	// Room/Area conversion
	ConvertRoom(sourceRoom interface{}) (*PMARoom, error)
	ConvertArea(sourceArea interface{}) (*PMAArea, error)

	// Control routing
	ExecuteAction(ctx context.Context, action PMAControlAction) (*PMAControlResult, error)

	// Synchronization
	SyncEntities(ctx context.Context) ([]PMAEntity, error)
	SyncRooms(ctx context.Context) ([]*PMARoom, error)
	GetLastSyncTime() *time.Time

	// Capabilities
	GetSupportedEntityTypes() []PMAEntityType
	GetSupportedCapabilities() []PMACapability
	SupportsRealtime() bool

	// Health and monitoring
	GetHealth() *AdapterHealth
	GetMetrics() *AdapterMetrics
}

// AdapterHealth represents the health status of an adapter
type AdapterHealth struct {
	IsHealthy       bool                   `json:"is_healthy"`
	LastHealthCheck time.Time              `json:"last_health_check"`
	Issues          []string               `json:"issues,omitempty"`
	ResponseTime    time.Duration          `json:"response_time"`
	ErrorRate       float64                `json:"error_rate"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

// AdapterMetrics represents performance metrics for an adapter
type AdapterMetrics struct {
	EntitiesManaged     int           `json:"entities_managed"`
	RoomsManaged        int           `json:"rooms_managed"`
	ActionsExecuted     int64         `json:"actions_executed"`
	SuccessfulActions   int64         `json:"successful_actions"`
	FailedActions       int64         `json:"failed_actions"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastSync            *time.Time    `json:"last_sync,omitempty"`
	SyncErrors          int           `json:"sync_errors"`
	Uptime              time.Duration `json:"uptime"`
}

// TypeConverter defines methods for converting specific entity types
type TypeConverter interface {
	ConvertLight(source interface{}) (*PMALightEntity, error)
	ConvertSwitch(source interface{}) (*PMASwitchEntity, error)
	ConvertSensor(source interface{}) (*PMASensorEntity, error)
	ConvertClimate(source interface{}) (PMAClimate, error)
	ConvertCover(source interface{}) (PMACover, error)
	ConvertCamera(source interface{}) (PMACamera, error)
	ConvertDevice(source interface{}) (PMADevice, error)
}

// CapabilityDetector helps identify capabilities of source entities
type CapabilityDetector interface {
	DetectCapabilities(sourceEntity interface{}) ([]PMACapability, error)
	CanDim(sourceEntity interface{}) bool
	CanChangeColor(sourceEntity interface{}) bool
	HasBattery(sourceEntity interface{}) bool
	SupportsPosition(sourceEntity interface{}) bool
}

// StateMapper maps source states to PMA states
type StateMapper interface {
	MapState(sourceState interface{}) PMAEntityState
	MapAttributes(sourceAttributes interface{}) map[string]interface{}
	NormalizeValue(key string, value interface{}) interface{}
}

// ControlRouter routes control commands to the appropriate source
type ControlRouter interface {
	RouteAction(ctx context.Context, action PMAControlAction) (*PMAControlResult, error)
	ValidateAction(action PMAControlAction) error
	GetActionMapping(pmaAction string) (sourceAction string, parameters map[string]interface{}, err error)
}

// ValidationRules defines validation rules for different entity types
type ValidationRules interface {
	ValidateEntity(entity PMAEntity) []ValidationError
	ValidateAction(action PMAControlAction) []ValidationError
	ValidateConversion(source interface{}, converted PMAEntity) []ValidationError
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// EntityIDGenerator generates consistent IDs across the PMA system
type EntityIDGenerator interface {
	GenerateEntityID(source PMASourceType, sourceID string) string
	GenerateRoomID(source PMASourceType, sourceID string) string
	GenerateAreaID(source PMASourceType, sourceID string) string
	ParseSourceID(pmaID string) (source PMASourceType, sourceID string, err error)
}

// QualityAssessment evaluates the quality of converted entities
type QualityAssessment interface {
	AssessEntity(entity PMAEntity) float64
	AssessConversion(source interface{}, converted PMAEntity) float64
	GetQualityFactors() []string
}

// ConversionContext provides context for entity conversion
type ConversionContext struct {
	SourceType    PMASourceType          `json:"source_type"`
	SourceVersion string                 `json:"source_version"`
	ConvertedAt   time.Time              `json:"converted_at"`
	ConvertedBy   string                 `json:"converted_by"`
	Options       map[string]interface{} `json:"options,omitempty"`
	Mappings      map[string]string      `json:"mappings,omitempty"`
}

// ConversionResult represents the result of a conversion operation
type ConversionResult struct {
	Success      bool               `json:"success"`
	Entity       PMAEntity          `json:"entity,omitempty"`
	Errors       []ValidationError  `json:"errors,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
	QualityScore float64            `json:"quality_score"`
	Context      *ConversionContext `json:"context"`
	Duration     time.Duration      `json:"duration"`
}

// BatchConversionResult represents the result of converting multiple entities
type BatchConversionResult struct {
	TotalCount     int               `json:"total_count"`
	SuccessCount   int               `json:"success_count"`
	FailureCount   int               `json:"failure_count"`
	Entities       []PMAEntity       `json:"entities"`
	Errors         []ValidationError `json:"errors,omitempty"`
	Warnings       []string          `json:"warnings,omitempty"`
	AverageQuality float64           `json:"average_quality"`
	ProcessingTime time.Duration     `json:"processing_time"`
}

// AdapterRegistry manages all registered adapters
type AdapterRegistry interface {
	RegisterAdapter(adapter PMAAdapter) error
	UnregisterAdapter(adapterID string) error
	GetAdapter(adapterID string) (PMAAdapter, error)
	GetAdapterBySource(sourceType PMASourceType) (PMAAdapter, error)
	GetAllAdapters() []PMAAdapter
	GetConnectedAdapters() []PMAAdapter
	GetAdapterMetrics(adapterID string) (*AdapterMetrics, error)
}

// EntityRegistry manages all PMA entities across sources
type EntityRegistry interface {
	RegisterEntity(entity PMAEntity) error
	UnregisterEntity(entityID string) error
	GetEntity(entityID string) (PMAEntity, error)
	GetEntitiesByType(entityType PMAEntityType) ([]PMAEntity, error)
	GetEntitiesBySource(source PMASourceType) ([]PMAEntity, error)
	GetEntitiesByRoom(roomID string) ([]PMAEntity, error)
	GetAllEntities() ([]PMAEntity, error)
	UpdateEntity(entity PMAEntity) error
	SearchEntities(query string) ([]PMAEntity, error)
}

// ConflictResolver handles conflicts when multiple sources provide the same entity
type ConflictResolver interface {
	ResolveEntityConflict(entities []PMAEntity) (PMAEntity, error)
	ResolvePrioritySource(sources []PMASourceType, entityType PMAEntityType) PMASourceType
	MergeEntityAttributes(entities []PMAEntity) map[string]interface{}
	CreateVirtualEntity(entities []PMAEntity) (PMAEntity, error)
}

// SourcePriorityManager manages priority between different sources
type SourcePriorityManager interface {
	SetSourcePriority(source PMASourceType, priority int) error
	GetSourcePriority(source PMASourceType) int
	GetPriorityOrder() []PMASourceType
	ShouldOverride(currentSource, newSource PMASourceType) bool
}
