package repositories

import (
	"context"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
)

// UserRepository defines user data access methods
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetAll(ctx context.Context) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
}

// ConfigRepository defines system config data access methods
type ConfigRepository interface {
	Get(ctx context.Context, key string) (*models.SystemConfig, error)
	Set(ctx context.Context, config *models.SystemConfig) error
	GetAll(ctx context.Context) ([]*models.SystemConfig, error)
	Delete(ctx context.Context, key string) error
}

// EntityRepository defines entity data access methods
type EntityRepository interface {
	Create(ctx context.Context, entity *models.Entity) error
	GetByID(ctx context.Context, entityID string) (*models.Entity, error)
	GetAll(ctx context.Context) ([]*models.Entity, error)
	GetByRoom(ctx context.Context, roomID int) ([]*models.Entity, error)
	Update(ctx context.Context, entity *models.Entity) error
	Delete(ctx context.Context, entityID string) error

	// PMA-specific methods
	CreateOrUpdatePMAEntity(entity types.PMAEntity) error
	GetPMAEntity(entityID string) (types.PMAEntity, error)
	GetPMAEntitiesBySource(source types.PMASourceType) ([]types.PMAEntity, error)
	DeletePMAEntity(entityID string) error
	UpdatePMAEntityMetadata(entityID string, metadata *types.PMAMetadata) error
}

// RoomRepository defines room data access methods
type RoomRepository interface {
	Create(ctx context.Context, room *models.Room) error
	GetByID(ctx context.Context, id int) (*models.Room, error)
	GetByName(ctx context.Context, name string) (*models.Room, error)
	GetAll(ctx context.Context) ([]*models.Room, error)
	GetByAreaID(ctx context.Context, areaID int) ([]*models.Room, error)
	Update(ctx context.Context, room *models.Room) error
	Delete(ctx context.Context, id int) error

	// Hierarchical methods for simplified Area → Room → Entity structure
	GetRoomsWithEntities(ctx context.Context, areaID *int) ([]models.RoomWithEntities, error)
	AssignToArea(ctx context.Context, roomID int, areaID *int) error
	GetUnassignedRooms(ctx context.Context) ([]*models.Room, error)
}

// AuthRepository defines authentication data access methods
type AuthRepository interface {
	GetSettings(ctx context.Context) (*models.AuthSetting, error)
	SetSettings(ctx context.Context, settings *models.AuthSetting) error
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, token string) (*models.Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteExpiredSessions(ctx context.Context) error
	RecordFailedAttempt(ctx context.Context, attempt *models.FailedAuthAttempt) error
	GetFailedAttempts(ctx context.Context, clientID string, since int64) ([]*models.FailedAuthAttempt, error)
	CleanupFailedAttempts(ctx context.Context, before int64) error
}

// KioskRepository defines kiosk device data access methods
type KioskRepository interface {
	// Token management
	CreateToken(ctx context.Context, token *models.KioskToken) error
	GetToken(ctx context.Context, token string) (*models.KioskToken, error)
	UpdateTokenLastUsed(ctx context.Context, token string) error
	DeleteToken(ctx context.Context, token string) error
	GetAllTokens(ctx context.Context) ([]*models.KioskToken, error)
	GetTokensByRoom(ctx context.Context, roomID string) ([]*models.KioskToken, error)
	UpdateTokenStatus(ctx context.Context, tokenID string, active bool) error

	// Pairing session management
	CreatePairingSession(ctx context.Context, session *models.KioskPairingSession) error
	GetPairingSession(ctx context.Context, pin string) (*models.KioskPairingSession, error)
	UpdatePairingSession(ctx context.Context, session *models.KioskPairingSession) error
	DeletePairingSession(ctx context.Context, id string) error
	CleanupExpiredSessions(ctx context.Context) error

	// Configuration management
	CreateConfig(ctx context.Context, config *models.KioskConfig) error
	GetConfig(ctx context.Context, roomID string) (*models.KioskConfig, error)
	UpdateConfig(ctx context.Context, config *models.KioskConfig) error
	DeleteConfig(ctx context.Context, roomID string) error

	// Device group management
	CreateDeviceGroup(ctx context.Context, group *models.KioskDeviceGroup) error
	GetDeviceGroup(ctx context.Context, groupID string) (*models.KioskDeviceGroup, error)
	GetAllDeviceGroups(ctx context.Context) ([]*models.KioskDeviceGroup, error)
	UpdateDeviceGroup(ctx context.Context, group *models.KioskDeviceGroup) error
	DeleteDeviceGroup(ctx context.Context, groupID string) error
	AddTokenToGroup(ctx context.Context, tokenID, groupID string) error
	RemoveTokenFromGroup(ctx context.Context, tokenID, groupID string) error
	GetTokenGroups(ctx context.Context, tokenID string) ([]*models.KioskDeviceGroup, error)
	GetGroupTokens(ctx context.Context, groupID string) ([]*models.KioskToken, error)

	// Logging
	CreateLog(ctx context.Context, log *models.KioskLog) error
	GetLogs(ctx context.Context, tokenID string, query *models.KioskLogQuery) ([]*models.KioskLog, error)
	DeleteOldLogs(ctx context.Context, olderThanDays int) error

	// Device status management
	CreateOrUpdateDeviceStatus(ctx context.Context, status *models.KioskDeviceStatus) error
	GetDeviceStatus(ctx context.Context, tokenID string) (*models.KioskDeviceStatus, error)
	GetAllDeviceStatuses(ctx context.Context) ([]*models.KioskDeviceStatus, error)
	UpdateHeartbeat(ctx context.Context, tokenID string) error

	// Command management
	CreateCommand(ctx context.Context, command *models.KioskCommand) error
	GetCommand(ctx context.Context, commandID string) (*models.KioskCommand, error)
	GetPendingCommands(ctx context.Context, tokenID string) ([]*models.KioskCommand, error)
	UpdateCommandStatus(ctx context.Context, commandID, status string) error
	CompleteCommand(ctx context.Context, commandID string, resultData []byte, errorMsg string) error
	CleanupExpiredCommands(ctx context.Context) error

	// Statistics
	GetKioskStats(ctx context.Context) (*models.KioskStatsResponse, error)
}

// NetworkRepository defines network device data access methods
type NetworkRepository interface {
	CreateDevice(ctx context.Context, device *models.NetworkDevice) error
	GetDevice(ctx context.Context, ipAddress string) (*models.NetworkDevice, error)
	GetAllDevices(ctx context.Context) ([]*models.NetworkDevice, error)
	GetOnlineDevices(ctx context.Context) ([]*models.NetworkDevice, error)
	UpdateDevice(ctx context.Context, device *models.NetworkDevice) error
	DeleteDevice(ctx context.Context, ipAddress string) error
	UpdateDeviceStatus(ctx context.Context, ipAddress string, isOnline bool) error
}

// UPSRepository defines UPS monitoring data access methods
type UPSRepository interface {
	CreateStatus(ctx context.Context, status *models.UPSStatus) error
	GetLatestStatus(ctx context.Context) (*models.UPSStatus, error)
	GetStatusHistory(ctx context.Context, limit int) ([]*models.UPSStatus, error)
	CleanupOldStatus(ctx context.Context, keepDays int) error
	GetStatusByTimeRange(ctx context.Context, start, end time.Time) ([]*models.UPSStatus, error)
	GetBatteryTrends(ctx context.Context, hours int) ([]*models.UPSStatus, error)
}

// CameraRepository defines camera device data access methods
type CameraRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, camera *models.Camera) error
	GetByID(ctx context.Context, id int) (*models.Camera, error)
	GetByEntityID(ctx context.Context, entityID string) (*models.Camera, error)
	GetAll(ctx context.Context) ([]*models.Camera, error)
	GetEnabled(ctx context.Context) ([]*models.Camera, error)
	Update(ctx context.Context, camera *models.Camera) error
	Delete(ctx context.Context, id int) error

	// Advanced queries
	GetByType(ctx context.Context, cameraType string) ([]*models.Camera, error)
	SearchCameras(ctx context.Context, query string) ([]*models.Camera, error)
	CountCameras(ctx context.Context) (int, error)
	CountEnabledCameras(ctx context.Context) (int, error)

	// Status and URL management
	UpdateStatus(ctx context.Context, id int, enabled bool) error
	UpdateStreamURL(ctx context.Context, id int, streamURL string) error
	UpdateSnapshotURL(ctx context.Context, id int, snapshotURL string) error
}

// DisplayRepository defines display settings data access methods
type DisplayRepository interface {
	GetSettings(ctx context.Context) (*models.DisplaySettings, error)
	UpdateSettings(ctx context.Context, settings *models.DisplaySettings) error
}

// BluetoothRepository defines Bluetooth device data access methods
type BluetoothRepository interface {
	CreateDevice(ctx context.Context, device *models.BluetoothDevice) error
	GetDevice(ctx context.Context, address string) (*models.BluetoothDevice, error)
	GetAllDevices(ctx context.Context) ([]*models.BluetoothDevice, error)
	GetPairedDevices(ctx context.Context) ([]*models.BluetoothDevice, error)
	GetConnectedDevices(ctx context.Context) ([]*models.BluetoothDevice, error)
	UpdateDevice(ctx context.Context, device *models.BluetoothDevice) error
	DeleteDevice(ctx context.Context, address string) error
}

// AreaRepository defines area management data access methods
type AreaRepository interface {
	// Area CRUD operations
	CreateArea(ctx context.Context, area *models.Area) error
	GetAreaByID(ctx context.Context, id int) (*models.Area, error)
	GetAreaByAreaID(ctx context.Context, areaID string) (*models.Area, error)
	GetAllAreas(ctx context.Context, includeInactive bool) ([]*models.Area, error)
	GetAreasByType(ctx context.Context, areaType string) ([]*models.Area, error)
	GetAreasByParent(ctx context.Context, parentID int) ([]*models.Area, error)
	GetAreaHierarchy(ctx context.Context) (*models.AreaHierarchy, error)
	UpdateArea(ctx context.Context, area *models.Area) error
	DeleteArea(ctx context.Context, id int) error

	// Area mapping operations
	CreateAreaMapping(ctx context.Context, mapping *models.AreaMapping) error
	GetAreaMapping(ctx context.Context, id int) (*models.AreaMapping, error)
	GetAreaMappingByExternal(ctx context.Context, externalAreaID, externalSystem string) (*models.AreaMapping, error)
	GetAllAreaMappings(ctx context.Context) ([]*models.AreaMappingWithDetails, error)
	GetAreaMappingsBySystem(ctx context.Context, externalSystem string) ([]*models.AreaMapping, error)
	GetAreaMappingsByArea(ctx context.Context, areaID int) ([]*models.AreaMapping, error)
	UpdateAreaMapping(ctx context.Context, mapping *models.AreaMapping) error
	DeleteAreaMapping(ctx context.Context, id int) error

	// Area settings operations
	GetAreaSetting(ctx context.Context, settingKey string, areaID *int) (*models.AreaSetting, error)
	GetAreaSettings(ctx context.Context, areaID *int) ([]*models.AreaSetting, error)
	SetAreaSetting(ctx context.Context, setting *models.AreaSetting) error
	DeleteAreaSetting(ctx context.Context, settingKey string, areaID *int) error
	GetGlobalSettings(ctx context.Context) (*models.AreaSettings, error)
	SetGlobalSettings(ctx context.Context, settings *models.AreaSettings) error

	// Area analytics operations
	CreateAreaAnalytic(ctx context.Context, analytic *models.AreaAnalytic) error
	GetAreaAnalytics(ctx context.Context, areaID int, startDate, endDate *time.Time) ([]*models.AreaAnalytic, error)
	GetAreaAnalyticsByMetric(ctx context.Context, metricName string, startDate, endDate *time.Time) ([]*models.AreaAnalytic, error)
	GetAreaAnalyticsSummary(ctx context.Context, areaIDs []int) ([]*models.AreaAnalyticsSummary, error)
	DeleteOldAnalytics(ctx context.Context, olderThanDays int) error

	// Area sync log operations
	CreateSyncLog(ctx context.Context, syncLog *models.AreaSyncLog) error
	GetSyncLog(ctx context.Context, id int) (*models.AreaSyncLog, error)
	GetSyncLogsBySystem(ctx context.Context, externalSystem string, limit int) ([]*models.AreaSyncLog, error)
	GetLastSyncTime(ctx context.Context, externalSystem string) (*time.Time, error)
	UpdateSyncLog(ctx context.Context, syncLog *models.AreaSyncLog) error
	DeleteOldSyncLogs(ctx context.Context, olderThanDays int) error

	// Room-area assignment operations
	CreateRoomAreaAssignment(ctx context.Context, assignment *models.RoomAreaAssignment) error
	GetRoomAreaAssignments(ctx context.Context, roomID int) ([]*models.RoomAreaAssignment, error)
	GetAreaRoomAssignments(ctx context.Context, areaID int) ([]*models.RoomAreaAssignment, error)
	UpdateRoomAreaAssignment(ctx context.Context, assignment *models.RoomAreaAssignment) error
	DeleteRoomAreaAssignment(ctx context.Context, id int) error
	DeleteRoomAreaAssignmentsByRoom(ctx context.Context, roomID int) error
	DeleteRoomAreaAssignmentsByArea(ctx context.Context, areaID int) error

	// Status and statistics
	GetAreaStatus(ctx context.Context) (*models.AreaStatus, error)
	GetEntityCountsByArea(ctx context.Context) (map[int]int, error)
	GetRoomCountsByArea(ctx context.Context) (map[int]int, error)

	// Bulk operations for simplified Area → Room → Entity hierarchy
	GetAreaWithRoomsAndEntities(ctx context.Context, areaID int) (*models.AreaWithRoomsAndEntities, error)
	GetAreaSummaries(ctx context.Context) ([]models.AreaSummary, error)
	GetAreaEntitiesForBulkAction(ctx context.Context, areaID int, filters models.BulkActionFilters) ([]models.SimpleEntity, error)
}

// ControllerRepository defines controller dashboard data access methods
type ControllerRepository interface {
	// Dashboard CRUD operations
	CreateDashboard(ctx context.Context, dashboard *models.ControllerDashboard) error
	GetDashboardByID(ctx context.Context, id int) (*models.ControllerDashboard, error)
	GetDashboardsByUserID(ctx context.Context, userID *int, includeShared bool) ([]*models.ControllerDashboard, error)
	GetAllDashboards(ctx context.Context, userID *int) ([]*models.ControllerDashboard, error)
	UpdateDashboard(ctx context.Context, dashboard *models.ControllerDashboard) error
	DeleteDashboard(ctx context.Context, id int) error
	DuplicateDashboard(ctx context.Context, id int, userID *int, newName string) (*models.ControllerDashboard, error)

	// Dashboard searching and filtering
	SearchDashboards(ctx context.Context, userID *int, query string, category string, tags []string) ([]*models.ControllerDashboard, error)
	GetDashboardsByCategory(ctx context.Context, userID *int, category string) ([]*models.ControllerDashboard, error)
	GetFavoriteDashboards(ctx context.Context, userID int) ([]*models.ControllerDashboard, error)
	ToggleFavorite(ctx context.Context, dashboardID int, userID int) error
	UpdateLastAccessed(ctx context.Context, dashboardID int) error

	// Template operations
	CreateTemplate(ctx context.Context, template *models.ControllerTemplate) error
	GetTemplateByID(ctx context.Context, id int) (*models.ControllerTemplate, error)
	GetTemplatesByUserID(ctx context.Context, userID *int, includePublic bool) ([]*models.ControllerTemplate, error)
	GetPublicTemplates(ctx context.Context) ([]*models.ControllerTemplate, error)
	UpdateTemplate(ctx context.Context, template *models.ControllerTemplate) error
	DeleteTemplate(ctx context.Context, id int) error
	IncrementTemplateUsage(ctx context.Context, id int) error

	// Sharing operations
	CreateShare(ctx context.Context, share *models.ControllerShare) error
	GetSharesByDashboardID(ctx context.Context, dashboardID int) ([]*models.ControllerShare, error)
	GetSharesByUserID(ctx context.Context, userID int) ([]*models.ControllerShare, error)
	UpdateSharePermissions(ctx context.Context, id int, permissions string) error
	DeleteShare(ctx context.Context, id int) error
	CheckUserAccess(ctx context.Context, dashboardID int, userID int) (string, error) // returns permission level

	// Usage analytics
	LogUsage(ctx context.Context, log *models.ControllerUsageLog) error
	GetUsageStats(ctx context.Context, dashboardID int, timeRange string) (map[string]interface{}, error)
	GetDashboardAnalytics(ctx context.Context, userID *int) (map[string]interface{}, error)
	CleanupOldLogs(ctx context.Context, retentionDays int) error

	// Import/Export
	ExportDashboard(ctx context.Context, id int) (map[string]interface{}, error)
	ImportDashboard(ctx context.Context, data map[string]interface{}, userID *int) (*models.ControllerDashboard, error)
}
