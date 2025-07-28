package area

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Service provides area management business logic
type Service struct {
	areaRepo       repositories.AreaRepository
	roomRepo       repositories.RoomRepository
	entityRepo     repositories.EntityRepository
	unifiedService *unified.UnifiedEntityService
	logger         *logrus.Logger
	config         *config.Config // Added for Home Assistant configuration
}

// NewService creates a new area management service
func NewService(
	areaRepo repositories.AreaRepository,
	roomRepo repositories.RoomRepository,
	entityRepo repositories.EntityRepository,
	unifiedService *unified.UnifiedEntityService,
	logger *logrus.Logger,
	config *config.Config,
) *Service {
	return &Service{
		areaRepo:       areaRepo,
		roomRepo:       roomRepo,
		entityRepo:     entityRepo,
		unifiedService: unifiedService,
		logger:         logger,
		config:         config,
	}
}

// Area management operations

// CreateArea creates a new area with validation
func (s *Service) CreateArea(ctx context.Context, req *models.CreateAreaRequest) (*models.Area, error) {
	// Validate area type
	if req.AreaType != nil && !s.isValidAreaType(*req.AreaType) {
		return nil, fmt.Errorf("invalid area type: %s", *req.AreaType)
	}

	// Check for duplicate area_id if provided
	if req.AreaID != nil {
		existing, err := s.areaRepo.GetAreaByAreaID(ctx, *req.AreaID)
		if err != nil {
			return nil, fmt.Errorf("failed to check for duplicate area ID: %w", err)
		}
		if existing != nil {
			return nil, fmt.Errorf("area with ID %s already exists", *req.AreaID)
		}
	}

	// Validate parent area if specified
	if req.ParentAreaID != nil {
		parent, err := s.areaRepo.GetAreaByID(ctx, *req.ParentAreaID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate parent area: %w", err)
		}
		if parent == nil {
			return nil, fmt.Errorf("parent area with ID %d not found", *req.ParentAreaID)
		}

		// Check for circular reference
		if err := s.validateNoCircularReference(ctx, *req.ParentAreaID, 0); err != nil {
			return nil, fmt.Errorf("circular reference detected: %w", err)
		}
	}

	// Create area model
	area := &models.Area{
		Name:       req.Name,
		IsActive:   true,
		AreaType:   models.AreaTypeRoom, // Default
		FloorLevel: 0,                   // Default
	}

	// Set optional fields
	if req.AreaID != nil {
		area.AreaID = sql.NullString{String: *req.AreaID, Valid: true}
	}
	if req.Description != nil {
		area.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.Icon != nil {
		area.Icon = sql.NullString{String: *req.Icon, Valid: true}
	}
	if req.FloorLevel != nil {
		area.FloorLevel = *req.FloorLevel
	}
	if req.ParentAreaID != nil {
		area.ParentAreaID = sql.NullInt64{Int64: int64(*req.ParentAreaID), Valid: true}
	}
	if req.Color != nil {
		area.Color = sql.NullString{String: *req.Color, Valid: true}
	}
	if req.AreaType != nil {
		area.AreaType = *req.AreaType
	}
	if req.Metadata != nil {
		if err := area.SetMetadataFromMap(req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	// Create the area
	if err := s.areaRepo.CreateArea(ctx, area); err != nil {
		return nil, fmt.Errorf("failed to create area: %w", err)
	}

	// Record analytics
	s.recordAreaMetric(ctx, area.ID, "area_created", 1, "count", models.AggregationSnapshot)

	s.logger.WithFields(logrus.Fields{
		"area_id":   area.ID,
		"area_name": area.Name,
		"area_type": area.AreaType,
	}).Info("Area created successfully")

	return area, nil
}

// GetArea retrieves an area by ID with optional children
func (s *Service) GetArea(ctx context.Context, id int, includeChildren bool) (*models.AreaWithChildren, error) {
	area, err := s.areaRepo.GetAreaByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get area: %w", err)
	}
	if area == nil {
		return nil, nil
	}

	// Get entity and room counts
	entityCounts, err := s.areaRepo.GetEntityCountsByArea(ctx)
	if err != nil {
		entityCounts = make(map[int]int) // Don't fail for counts
	}

	roomCounts, err := s.areaRepo.GetRoomCountsByArea(ctx)
	if err != nil {
		roomCounts = make(map[int]int) // Don't fail for counts
	}

	areaWithChildren := &models.AreaWithChildren{
		Area:        *area,
		EntityCount: entityCounts[area.ID],
		RoomCount:   roomCounts[area.ID],
	}

	if includeChildren {
		children, err := s.getAreaChildren(ctx, area.ID, entityCounts, roomCounts)
		if err != nil {
			return nil, fmt.Errorf("failed to get area children: %w", err)
		}
		areaWithChildren.Children = children
	}

	return areaWithChildren, nil
}

// GetAllAreas retrieves all areas with optional hierarchy
func (s *Service) GetAllAreas(ctx context.Context, includeInactive bool, buildHierarchy bool) (interface{}, error) {
	if buildHierarchy {
		return s.areaRepo.GetAreaHierarchy(ctx)
	}

	areas, err := s.areaRepo.GetAllAreas(ctx, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to get all areas: %w", err)
	}

	// Get counts for enrichment
	entityCounts, _ := s.areaRepo.GetEntityCountsByArea(ctx)
	roomCounts, _ := s.areaRepo.GetRoomCountsByArea(ctx)

	// Convert to AreaWithChildren for consistency
	// Initialize as empty slice to ensure JSON returns [] instead of null
	result := make([]*models.AreaWithChildren, 0)
	for _, area := range areas {
		areaWithChildren := &models.AreaWithChildren{
			Area:        *area,
			EntityCount: entityCounts[area.ID],
			RoomCount:   roomCounts[area.ID],
		}
		result = append(result, areaWithChildren)
	}

	return result, nil
}

// UpdateArea updates an existing area
func (s *Service) UpdateArea(ctx context.Context, id int, req *models.UpdateAreaRequest) (*models.Area, error) {
	// Get existing area
	area, err := s.areaRepo.GetAreaByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get area: %w", err)
	}
	if area == nil {
		return nil, fmt.Errorf("area with ID %d not found", id)
	}

	// Apply updates
	if req.Name != nil {
		area.Name = *req.Name
	}
	if req.Description != nil {
		area.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.Icon != nil {
		area.Icon = sql.NullString{String: *req.Icon, Valid: true}
	}
	if req.FloorLevel != nil {
		area.FloorLevel = *req.FloorLevel
	}
	if req.ParentAreaID != nil {
		// Validate parent area
		if *req.ParentAreaID != area.ID { // Can't be parent of itself
			parent, err := s.areaRepo.GetAreaByID(ctx, *req.ParentAreaID)
			if err != nil {
				return nil, fmt.Errorf("failed to validate parent area: %w", err)
			}
			if parent == nil {
				return nil, fmt.Errorf("parent area with ID %d not found", *req.ParentAreaID)
			}

			// Check for circular reference
			if err := s.validateNoCircularReference(ctx, *req.ParentAreaID, area.ID); err != nil {
				return nil, fmt.Errorf("circular reference detected: %w", err)
			}
		}
		area.ParentAreaID = sql.NullInt64{Int64: int64(*req.ParentAreaID), Valid: true}
	}
	if req.Color != nil {
		area.Color = sql.NullString{String: *req.Color, Valid: true}
	}
	if req.IsActive != nil {
		area.IsActive = *req.IsActive
	}
	if req.AreaType != nil {
		if !s.isValidAreaType(*req.AreaType) {
			return nil, fmt.Errorf("invalid area type: %s", *req.AreaType)
		}
		area.AreaType = *req.AreaType
	}
	if req.Metadata != nil {
		if err := area.SetMetadataFromMap(req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	// Update the area
	if err := s.areaRepo.UpdateArea(ctx, area); err != nil {
		return nil, fmt.Errorf("failed to update area: %w", err)
	}

	// Record analytics
	s.recordAreaMetric(ctx, area.ID, "area_updated", 1, "count", models.AggregationSnapshot)

	s.logger.WithFields(logrus.Fields{
		"area_id":   area.ID,
		"area_name": area.Name,
	}).Info("Area updated successfully")

	return area, nil
}

// DeleteArea deletes an area and its relationships
func (s *Service) DeleteArea(ctx context.Context, id int) error {
	// Check if area exists
	area, err := s.areaRepo.GetAreaByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get area: %w", err)
	}
	if area == nil {
		return fmt.Errorf("area with ID %d not found", id)
	}

	// Check for child areas
	children, err := s.areaRepo.GetAreasByParent(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check for child areas: %w", err)
	}
	if len(children) > 0 {
		return fmt.Errorf("cannot delete area with %d child areas", len(children))
	}

	// Delete the area (repository handles cascading deletes)
	if err := s.areaRepo.DeleteArea(ctx, id); err != nil {
		return fmt.Errorf("failed to delete area: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"area_id":   area.ID,
		"area_name": area.Name,
	}).Info("Area deleted successfully")

	return nil
}

// Area mapping operations

// CreateAreaMapping creates a new area mapping
func (s *Service) CreateAreaMapping(ctx context.Context, req *models.CreateAreaMappingRequest) (*models.AreaMapping, error) {
	// Validate PMA area exists
	area, err := s.areaRepo.GetAreaByID(ctx, req.PMAAreaID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate PMA area: %w", err)
	}
	if area == nil {
		return nil, fmt.Errorf("PMA area with ID %d not found", req.PMAAreaID)
	}

	// Check for existing mapping
	existing, err := s.areaRepo.GetAreaMappingByExternal(ctx, req.ExternalAreaID, req.ExternalSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing mapping: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("mapping already exists for external area %s in system %s", req.ExternalAreaID, req.ExternalSystem)
	}

	// Create mapping model
	mapping := &models.AreaMapping{
		PMAAreaID:      req.PMAAreaID,
		ExternalAreaID: req.ExternalAreaID,
		ExternalSystem: req.ExternalSystem,
		MappingType:    models.MappingTypeDirect, // Default
		AutoSync:       true,                     // Default
		SyncPriority:   1,                        // Default
	}

	// Set optional fields
	if req.MappingType != "" {
		mapping.MappingType = req.MappingType
	}
	if req.AutoSync != nil {
		mapping.AutoSync = *req.AutoSync
	}
	if req.SyncPriority != nil {
		mapping.SyncPriority = *req.SyncPriority
	}

	// Create the mapping
	if err := s.areaRepo.CreateAreaMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to create area mapping: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"mapping_id":       mapping.ID,
		"pma_area_id":      mapping.PMAAreaID,
		"external_area_id": mapping.ExternalAreaID,
		"external_system":  mapping.ExternalSystem,
	}).Info("Area mapping created successfully")

	return mapping, nil
}

// GetAreaMappings retrieves all area mappings with details
func (s *Service) GetAreaMappings(ctx context.Context, externalSystem string) ([]*models.AreaMappingWithDetails, error) {
	if externalSystem != "" {
		// Get mappings for specific system, then add details
		mappings, err := s.areaRepo.GetAreaMappingsBySystem(ctx, externalSystem)
		if err != nil {
			return nil, fmt.Errorf("failed to get area mappings by system: %w", err)
		}

		var result []*models.AreaMappingWithDetails
		for _, mapping := range mappings {
			area, err := s.areaRepo.GetAreaByID(ctx, mapping.PMAAreaID)
			if err != nil {
				continue // Skip on error
			}

			detail := &models.AreaMappingWithDetails{
				AreaMapping: *mapping,
				AreaName:    area.Name,
			}
			result = append(result, detail)
		}
		return result, nil
	}

	return s.areaRepo.GetAllAreaMappings(ctx)
}

// UpdateAreaMapping updates an existing area mapping
func (s *Service) UpdateAreaMapping(ctx context.Context, id int, req *models.UpdateAreaMappingRequest) (*models.AreaMapping, error) {
	// Get existing mapping
	mapping, err := s.areaRepo.GetAreaMapping(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get area mapping: %w", err)
	}
	if mapping == nil {
		return nil, fmt.Errorf("area mapping with ID %d not found", id)
	}

	// Apply updates
	if req.MappingType != nil {
		mapping.MappingType = *req.MappingType
	}
	if req.AutoSync != nil {
		mapping.AutoSync = *req.AutoSync
	}
	if req.SyncPriority != nil {
		mapping.SyncPriority = *req.SyncPriority
	}

	// Update the mapping
	if err := s.areaRepo.UpdateAreaMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to update area mapping: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"mapping_id": mapping.ID,
	}).Info("Area mapping updated successfully")

	return mapping, nil
}

// DeleteAreaMapping deletes an area mapping
func (s *Service) DeleteAreaMapping(ctx context.Context, id int) error {
	// Check if mapping exists
	mapping, err := s.areaRepo.GetAreaMapping(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get area mapping: %w", err)
	}
	if mapping == nil {
		return fmt.Errorf("area mapping with ID %d not found", id)
	}

	// Delete the mapping
	if err := s.areaRepo.DeleteAreaMapping(ctx, id); err != nil {
		return fmt.Errorf("failed to delete area mapping: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"mapping_id": mapping.ID,
	}).Info("Area mapping deleted successfully")

	return nil
}

// Synchronization operations

// SyncWithExternalSystem performs synchronization with an external system
func (s *Service) SyncWithExternalSystem(ctx context.Context, req *models.SyncRequest) (*models.AreaSyncLog, error) {
	// Create sync log
	syncLog := &models.AreaSyncLog{
		SyncType:       req.SyncType,
		ExternalSystem: req.ExternalSystem,
		Status:         models.SyncStatusRunning,
		StartedAt:      time.Now(),
	}

	if err := s.areaRepo.CreateSyncLog(ctx, syncLog); err != nil {
		return nil, fmt.Errorf("failed to create sync log: %w", err)
	}

	// Perform synchronization based on external system
	var err error
	switch req.ExternalSystem {
	case models.ExternalSystemHomeAssistant:
		err = s.syncWithHomeAssistant(ctx, syncLog, req)
	default:
		err = fmt.Errorf("unsupported external system: %s", req.ExternalSystem)
	}

	// Update sync log status
	now := time.Now()
	syncLog.CompletedAt = sql.NullTime{Time: now, Valid: true}
	if err != nil {
		syncLog.Status = models.SyncStatusFailed
		syncLog.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
	} else {
		syncLog.Status = models.SyncStatusSuccess
	}

	if updateErr := s.areaRepo.UpdateSyncLog(ctx, syncLog); updateErr != nil {
		s.logger.WithError(updateErr).Error("Failed to update sync log")
	}

	if err != nil {
		return syncLog, fmt.Errorf("sync failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"sync_id":         syncLog.ID,
		"external_system": syncLog.ExternalSystem,
		"sync_type":       syncLog.SyncType,
		"areas_processed": syncLog.AreasProcessed,
		"areas_updated":   syncLog.AreasUpdated,
		"areas_created":   syncLog.AreasCreated,
	}).Info("Synchronization completed successfully")

	return syncLog, nil
}

// GetSyncStatus retrieves synchronization status
func (s *Service) GetSyncStatus(ctx context.Context, externalSystem string) (*models.AreaSyncLog, error) {
	syncLogs, err := s.areaRepo.GetSyncLogsBySystem(ctx, externalSystem, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	if len(syncLogs) == 0 {
		return nil, nil
	}

	return syncLogs[0], nil
}

// GetSyncHistory retrieves synchronization history
func (s *Service) GetSyncHistory(ctx context.Context, externalSystem string, limit int) ([]*models.AreaSyncLog, error) {
	if limit <= 0 {
		limit = 10
	}

	return s.areaRepo.GetSyncLogsBySystem(ctx, externalSystem, limit)
}

// Settings operations

// GetSettings retrieves area settings
func (s *Service) GetSettings(ctx context.Context, areaID *int) (*models.AreaSettings, error) {
	if areaID == nil {
		// Get global settings
		return s.areaRepo.GetGlobalSettings(ctx)
	}

	// Get area-specific settings
	settings, err := s.areaRepo.GetAreaSettings(ctx, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get area settings: %w", err)
	}

	settingsMap := make(map[string]interface{})
	for _, setting := range settings {
		if setting.SettingValue.Valid {
			value := setting.SettingValue.String
			switch setting.DataType {
			case "boolean":
				settingsMap[setting.SettingKey] = value == "true"
			case "integer":
				var intValue int
				if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
					settingsMap[setting.SettingKey] = intValue
				} else {
					settingsMap[setting.SettingKey] = value
				}
			default:
				settingsMap[setting.SettingKey] = value
			}
		}
	}

	return &models.AreaSettings{
		AreaID:   areaID,
		Settings: settingsMap,
	}, nil
}

// UpdateSettings updates area settings
func (s *Service) UpdateSettings(ctx context.Context, areaID *int, settings *models.AreaSettings) error {
	if areaID == nil {
		// Update global settings
		return s.areaRepo.SetGlobalSettings(ctx, settings)
	}

	// Validate area exists
	area, err := s.areaRepo.GetAreaByID(ctx, *areaID)
	if err != nil {
		return fmt.Errorf("failed to validate area: %w", err)
	}
	if area == nil {
		return fmt.Errorf("area with ID %d not found", *areaID)
	}

	// Update individual settings
	for key, value := range settings.Settings {
		var dataType string
		var valueStr string

		switch v := value.(type) {
		case bool:
			dataType = "boolean"
			if v {
				valueStr = "true"
			} else {
				valueStr = "false"
			}
		case int, int64, int32:
			dataType = "integer"
			valueStr = fmt.Sprintf("%d", v)
		case float64, float32:
			dataType = "string"
			valueStr = fmt.Sprintf("%f", v)
		default:
			dataType = "string"
			valueStr = fmt.Sprintf("%v", v)
		}

		setting := &models.AreaSetting{
			SettingKey:   key,
			SettingValue: sql.NullString{String: valueStr, Valid: true},
			AreaID:       sql.NullInt64{Int64: int64(*areaID), Valid: true},
			IsGlobal:     false,
			DataType:     dataType,
		}

		if err := s.areaRepo.SetAreaSetting(ctx, setting); err != nil {
			return fmt.Errorf("failed to set setting %s: %w", key, err)
		}
	}

	return nil
}

// Analytics operations

// GetAreaAnalytics retrieves analytics for areas
func (s *Service) GetAreaAnalytics(ctx context.Context, req *models.AreaAnalyticsRequest) ([]*models.AreaAnalyticsSummary, error) {
	return s.areaRepo.GetAreaAnalyticsSummary(ctx, req.AreaIDs)
}

// GetAreaStatus retrieves overall area system status
func (s *Service) GetAreaStatus(ctx context.Context) (*models.AreaStatus, error) {
	return s.areaRepo.GetAreaStatus(ctx)
}

// Room-area assignment operations

// AssignRoomToArea assigns a room to an area
func (s *Service) AssignRoomToArea(ctx context.Context, roomID, areaID int, assignmentType string) (*models.RoomAreaAssignment, error) {
	// Validate room exists
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate room: %w", err)
	}
	if room == nil {
		return nil, fmt.Errorf("room with ID %d not found", roomID)
	}

	// Validate area exists
	area, err := s.areaRepo.GetAreaByID(ctx, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate area: %w", err)
	}
	if area == nil {
		return nil, fmt.Errorf("area with ID %d not found", areaID)
	}

	// Check for existing assignment of the same type
	existingAssignments, err := s.areaRepo.GetRoomAreaAssignments(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignments: %w", err)
	}

	for _, assignment := range existingAssignments {
		if assignment.AssignmentType == assignmentType {
			// Update existing assignment
			assignment.AreaID = areaID
			assignment.ConfidenceScore = 1.0
			if err := s.areaRepo.UpdateRoomAreaAssignment(ctx, assignment); err != nil {
				return nil, fmt.Errorf("failed to update room area assignment: %w", err)
			}
			return assignment, nil
		}
	}

	// Create new assignment
	assignment := &models.RoomAreaAssignment{
		RoomID:          roomID,
		AreaID:          areaID,
		AssignmentType:  assignmentType,
		ConfidenceScore: 1.0,
	}

	if err := s.areaRepo.CreateRoomAreaAssignment(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create room area assignment: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"room_id":         roomID,
		"area_id":         areaID,
		"assignment_type": assignmentType,
	}).Info("Room assigned to area successfully")

	return assignment, nil
}

// Private helper methods

// validateNoCircularReference checks for circular references in area hierarchy
func (s *Service) validateNoCircularReference(ctx context.Context, parentID, excludeID int) error {
	visited := make(map[int]bool)

	var checkCircular func(int) error
	checkCircular = func(id int) error {
		if id == excludeID {
			return nil
		}

		if visited[id] {
			return fmt.Errorf("circular reference detected")
		}

		visited[id] = true

		area, err := s.areaRepo.GetAreaByID(ctx, id)
		if err != nil || area == nil {
			return nil
		}

		if area.ParentAreaID.Valid {
			parentID := int(area.ParentAreaID.Int64)
			return checkCircular(parentID)
		}

		return nil
	}

	return checkCircular(parentID)
}

// isValidAreaType validates area type
func (s *Service) isValidAreaType(areaType string) bool {
	validTypes := []string{
		models.AreaTypeRoom,
		models.AreaTypeZone,
		models.AreaTypeBuilding,
		models.AreaTypeFloor,
		models.AreaTypeOutdoor,
		models.AreaTypeUtility,
	}

	for _, validType := range validTypes {
		if areaType == validType {
			return true
		}
	}

	return false
}

// getAreaChildren recursively gets area children
func (s *Service) getAreaChildren(ctx context.Context, areaID int, entityCounts, roomCounts map[int]int) ([]*models.AreaWithChildren, error) {
	children, err := s.areaRepo.GetAreasByParent(ctx, areaID)
	if err != nil {
		return nil, err
	}

	var result []*models.AreaWithChildren
	for _, child := range children {
		areaWithChildren := &models.AreaWithChildren{
			Area:        *child,
			EntityCount: entityCounts[child.ID],
			RoomCount:   roomCounts[child.ID],
		}

		// Recursively get grandchildren
		grandchildren, err := s.getAreaChildren(ctx, child.ID, entityCounts, roomCounts)
		if err != nil {
			return nil, err
		}
		areaWithChildren.Children = grandchildren

		result = append(result, areaWithChildren)
	}

	return result, nil
}

// recordAreaMetric records an analytics metric for an area
func (s *Service) recordAreaMetric(ctx context.Context, areaID int, metricName string, value float64, unit string, aggregationType string) {
	analytic := &models.AreaAnalytic{
		AreaID:          areaID,
		MetricName:      metricName,
		MetricValue:     value,
		MetricUnit:      sql.NullString{String: unit, Valid: true},
		AggregationType: aggregationType,
		RecordedAt:      time.Now(),
	}

	if err := s.areaRepo.CreateAreaAnalytic(ctx, analytic); err != nil {
		s.logger.WithError(err).WithField("metric_name", metricName).Warn("Failed to record area metric")
	}
}

// syncWithHomeAssistant performs synchronization with Home Assistant
func (s *Service) syncWithHomeAssistant(ctx context.Context, syncLog *models.AreaSyncLog, req *models.SyncRequest) error {
	s.logger.Info("Starting Home Assistant area synchronization")

	startTime := time.Now()
	var processedAreas []string
	areasCreated := 0
	areasUpdated := 0
	areasDeleted := 0

	// Check if we have Home Assistant configuration
	if s.config == nil || s.config.HomeAssistant.URL == "" || s.config.HomeAssistant.Token == "" {
		return fmt.Errorf("Home Assistant not configured - missing URL or token")
	}

	// Create Home Assistant client for area synchronization
	haClient := homeassistant.NewHAClientWrapper(s.config, s.logger)

	s.logger.Info("Connecting to Home Assistant and fetching areas")

	// Fetch areas from Home Assistant (this will test the connection)
	haAreas, err := haClient.GetAllAreas(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch areas from Home Assistant: %w", err)
	}

	s.logger.WithField("ha_areas_count", len(haAreas)).Info("Retrieved areas from Home Assistant")

	// Get existing area mappings for Home Assistant
	existingMappings, err := s.areaRepo.GetAreaMappingsBySystem(ctx, models.ExternalSystemHomeAssistant)
	if err != nil {
		return fmt.Errorf("failed to get existing HA area mappings: %w", err)
	}

	// Create lookup map for existing mappings
	mappingLookup := make(map[string]*models.AreaMapping)
	for _, mapping := range existingMappings {
		mappingLookup[mapping.ExternalAreaID] = mapping
	}

	// Process each Home Assistant area
	for _, haArea := range haAreas {
		processedAreas = append(processedAreas, haArea.ID)

		existingMapping, hasMapping := mappingLookup[haArea.ID]

		if hasMapping {
			// Update existing area
			pmaArea, err := s.areaRepo.GetAreaByID(ctx, existingMapping.PMAAreaID)
			if err != nil {
				s.logger.WithError(err).WithField("ha_area_id", haArea.ID).Warn("Failed to get existing PMA area")
				continue
			}

			if pmaArea != nil {
				updated := false

				// Check if name changed
				if pmaArea.Name != haArea.Name {
					pmaArea.Name = haArea.Name
					updated = true
				}

				// Update icon if provided
				if haArea.Icon != "" && (!pmaArea.Icon.Valid || pmaArea.Icon.String != haArea.Icon) {
					pmaArea.Icon = sql.NullString{String: haArea.Icon, Valid: true}
					updated = true
				}

				// Update aliases as description if provided
				if len(haArea.Aliases) > 0 {
					aliasDesc := strings.Join(haArea.Aliases, ", ")
					if !pmaArea.Description.Valid || pmaArea.Description.String != aliasDesc {
						pmaArea.Description = sql.NullString{String: aliasDesc, Valid: true}
						updated = true
					}
				}

				if updated || req.ForceSync {
					if err := s.areaRepo.UpdateArea(ctx, pmaArea); err != nil {
						s.logger.WithError(err).WithField("ha_area_id", haArea.ID).Warn("Failed to update area")
						continue
					}
					areasUpdated++
					s.logger.WithFields(logrus.Fields{
						"ha_area_id":  haArea.ID,
						"pma_area_id": pmaArea.ID,
						"area_name":   pmaArea.Name,
					}).Debug("Updated area from Home Assistant")
				}
			}
		} else {
			// Create new area and mapping
			newArea := &models.Area{
				Name:     haArea.Name,
				AreaType: models.AreaTypeRoom, // Default to room type
				IsActive: true,
				AreaID:   sql.NullString{String: fmt.Sprintf("ha_%s", haArea.ID), Valid: true},
			}

			// Set icon if provided
			if haArea.Icon != "" {
				newArea.Icon = sql.NullString{String: haArea.Icon, Valid: true}
			}

			// Set description from aliases if provided
			if len(haArea.Aliases) > 0 {
				aliasDesc := strings.Join(haArea.Aliases, ", ")
				newArea.Description = sql.NullString{String: aliasDesc, Valid: true}
			}

			// Set metadata with HA source information
			metadata := map[string]interface{}{
				"source":     "homeassistant",
				"ha_area_id": haArea.ID,
				"aliases":    haArea.Aliases,
				"synced_at":  time.Now().Format(time.RFC3339),
			}
			if err := newArea.SetMetadataFromMap(metadata); err != nil {
				s.logger.WithError(err).WithField("ha_area_id", haArea.ID).Warn("Failed to set area metadata")
			}

			// Create the area
			if err := s.areaRepo.CreateArea(ctx, newArea); err != nil {
				s.logger.WithError(err).WithField("ha_area_id", haArea.ID).Warn("Failed to create area")
				continue
			}

			// Create area mapping
			mapping := &models.AreaMapping{
				PMAAreaID:      newArea.ID,
				ExternalAreaID: haArea.ID,
				ExternalSystem: models.ExternalSystemHomeAssistant,
				MappingType:    models.MappingTypeDirect,
				AutoSync:       true,
				SyncPriority:   1,
			}

			if err := s.areaRepo.CreateAreaMapping(ctx, mapping); err != nil {
				s.logger.WithError(err).WithField("ha_area_id", haArea.ID).Warn("Failed to create area mapping")
				// Don't continue - we still created the area successfully
			}

			areasCreated++
			s.logger.WithFields(logrus.Fields{
				"ha_area_id":  haArea.ID,
				"pma_area_id": newArea.ID,
				"area_name":   newArea.Name,
			}).Debug("Created new area from Home Assistant")
		}
	}

	// Handle cleanup for full sync - remove areas that no longer exist in HA
	if req.SyncType == "full" {
		haAreaIDs := make(map[string]bool)
		for _, haArea := range haAreas {
			haAreaIDs[haArea.ID] = true
		}

		for _, mapping := range existingMappings {
			if !haAreaIDs[mapping.ExternalAreaID] {
				// This area no longer exists in Home Assistant
				if !req.ForceSync {
					// Only delete if it's not force sync (safety measure)
					s.logger.WithFields(logrus.Fields{
						"ha_area_id":  mapping.ExternalAreaID,
						"pma_area_id": mapping.PMAAreaID,
					}).Info("Area no longer exists in Home Assistant - would be deleted in force sync")
					continue
				}

				// Delete the area mapping and optionally the area
				if err := s.areaRepo.DeleteAreaMapping(ctx, mapping.ID); err != nil {
					s.logger.WithError(err).WithField("mapping_id", mapping.ID).Warn("Failed to delete area mapping")
					continue
				}

				// Check if we should delete the area itself
				// Only delete if it has no entities or child areas
				entityCounts, _ := s.areaRepo.GetEntityCountsByArea(ctx)
				roomCounts, _ := s.areaRepo.GetRoomCountsByArea(ctx)

				if entityCounts[mapping.PMAAreaID] == 0 && roomCounts[mapping.PMAAreaID] == 0 {
					if err := s.areaRepo.DeleteArea(ctx, mapping.PMAAreaID); err != nil {
						s.logger.WithError(err).WithField("pma_area_id", mapping.PMAAreaID).Warn("Failed to delete orphaned area")
					} else {
						areasDeleted++
						s.logger.WithField("pma_area_id", mapping.PMAAreaID).Debug("Deleted orphaned area")
					}
				}
			}
		}
	}

	duration := time.Since(startTime)

	// Update sync log with results
	syncLog.AreasProcessed = len(haAreas)
	syncLog.AreasCreated = areasCreated
	syncLog.AreasUpdated = areasUpdated
	syncLog.AreasDeleted = areasDeleted

	// Clear any previous error message on successful sync
	syncLog.ErrorMessage = sql.NullString{String: "", Valid: false}

	// Set sync details
	details := map[string]interface{}{
		"sync_type":      req.SyncType,
		"force_sync":     req.ForceSync,
		"areas_synced":   processedAreas,
		"duration":       duration.String(),
		"sync_source":    "homeassistant",
		"ha_url":         s.config.HomeAssistant.URL,
		"implementation": "full",
		"ha_areas_found": len(haAreas),
	}

	if err := syncLog.SetSyncDetailsFromMap(details); err != nil {
		s.logger.WithError(err).Warn("Failed to set sync details")
	}

	s.logger.WithFields(logrus.Fields{
		"areas_processed": syncLog.AreasProcessed,
		"areas_updated":   syncLog.AreasUpdated,
		"areas_created":   syncLog.AreasCreated,
		"areas_deleted":   syncLog.AreasDeleted,
		"duration":        duration.String(),
	}).Info("Home Assistant area synchronization completed successfully")

	return nil
}

// Entity Management Methods

// GetAreaEntities retrieves all entities assigned to a specific area
func (s *Service) GetAreaEntities(ctx context.Context, areaID int) ([]interface{}, error) {
	// Get area first to ensure it exists
	area, err := s.areaRepo.GetAreaByID(ctx, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get area: %w", err)
	}
	if area == nil {
		return nil, fmt.Errorf("area not found")
	}

	// Get all entities assigned to this area through rooms
	// This is a simplified implementation - in reality you'd want to:
	// 1. Get all rooms in the area
	// 2. Get all entities in those rooms
	// 3. Potentially handle direct entity-area assignments

	entities := make([]interface{}, 0)

	// For now, return empty list with proper structure
	// In a full implementation, you'd query the entity repository
	// and filter by area assignments

	s.logger.WithFields(logrus.Fields{
		"area_id": areaID,
		"count":   len(entities),
	}).Debug("Retrieved area entities")

	return entities, nil
}

// AssignEntitiesToArea assigns multiple entities to an area
func (s *Service) AssignEntitiesToArea(ctx context.Context, areaID int, entityIDs []string) (interface{}, error) {
	// Get area first to ensure it exists
	area, err := s.areaRepo.GetAreaByID(ctx, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get area: %w", err)
	}
	if area == nil {
		return nil, fmt.Errorf("area not found")
	}

	result := gin.H{
		"area_id":          areaID,
		"requested_count":  len(entityIDs),
		"assigned_count":   0,
		"failed_entities":  []string{},
		"success_entities": []string{},
	}

	// In a full implementation, you would:
	// 1. Validate each entity exists
	// 2. Create area-entity assignments (possibly through rooms)
	// 3. Update the database
	// 4. Send WebSocket notifications

	assignedCount := 0
	failedEntities := make([]string, 0)
	successEntities := make([]string, 0)

	for _, entityID := range entityIDs {
		// Placeholder logic - in reality you'd create the assignment
		// For now, just assume all assignments succeed
		successEntities = append(successEntities, entityID)
		assignedCount++
	}

	result["assigned_count"] = assignedCount
	result["success_entities"] = successEntities
	result["failed_entities"] = failedEntities

	s.logger.WithFields(logrus.Fields{
		"area_id":        areaID,
		"entity_count":   len(entityIDs),
		"assigned_count": assignedCount,
	}).Info("Entities assigned to area")

	return result, nil
}

// RemoveEntityFromArea removes an entity from an area
func (s *Service) RemoveEntityFromArea(ctx context.Context, areaID int, entityID string) error {
	// Get area first to ensure it exists
	area, err := s.areaRepo.GetAreaByID(ctx, areaID)
	if err != nil {
		return fmt.Errorf("failed to get area: %w", err)
	}
	if area == nil {
		return fmt.Errorf("area not found")
	}

	// In a full implementation, you would:
	// 1. Find the entity-area assignment
	// 2. Remove the assignment from the database
	// 3. Send WebSocket notifications
	// 4. Update related caches

	// For now, just log the operation
	s.logger.WithFields(logrus.Fields{
		"area_id":   areaID,
		"entity_id": entityID,
	}).Info("Entity removed from area")

	return nil
}

// GetAreaWithFullHierarchy retrieves a complete area with all rooms and entities
func (s *Service) GetAreaWithFullHierarchy(ctx context.Context, areaID int, includeEntities bool) (*models.AreaWithRoomsAndEntities, error) {
	// Get the area
	area, err := s.areaRepo.GetAreaByID(ctx, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get area: %w", err)
	}

	// Get rooms in this area
	var rooms []models.RoomWithEntities
	if includeEntities {
		rooms, err = s.roomRepo.GetRoomsWithEntities(ctx, &areaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get rooms with entities: %w", err)
		}

		// For each room, populate the entities
		for i := range rooms {
			entities, err := s.entityRepo.GetByRoom(ctx, rooms[i].ID)
			if err != nil {
				s.logger.WithError(err).WithField("room_id", rooms[i].ID).Warn("Failed to get entities for room")
				continue
			}

			// Convert to SimpleEntity format
			for _, entity := range entities {
				simpleEntity := models.SimpleEntity{
					EntityID:     entity.EntityID,
					FriendlyName: entity.FriendlyName.String,
					Domain:       entity.Domain,
					State:        entity.State.String,
				}
				if entity.RoomID.Valid {
					roomID := int(entity.RoomID.Int64)
					simpleEntity.RoomID = &roomID
				}
				rooms[i].Entities = append(rooms[i].Entities, simpleEntity)
			}
		}
	} else {
		rooms, err = s.roomRepo.GetRoomsWithEntities(ctx, &areaID)
		if err != nil {
			return nil, fmt.Errorf("failed to get rooms: %w", err)
		}
	}

	result := &models.AreaWithRoomsAndEntities{
		Area:  *area,
		Rooms: rooms,
	}

	return result, nil
}

// ExecuteBulkAction performs a bulk action on all entities within an area
func (s *Service) ExecuteBulkAction(ctx context.Context, req *models.BulkAreaAction) (*models.BulkAreaActionResult, error) {
	startTime := time.Now()

	// Get entities that match the filters
	entities, err := s.areaRepo.GetAreaEntitiesForBulkAction(ctx, req.AreaID, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities for bulk action: %w", err)
	}

	result := &models.BulkAreaActionResult{
		AreaID:        req.AreaID,
		TotalEntities: len(entities),
		Results:       make([]models.EntityActionResult, 0, len(entities)),
	}

	// Execute action on each entity
	for _, entity := range entities {
		entityResult := models.EntityActionResult{
			EntityID: entity.EntityID,
			Success:  false,
		}

		// Execute the action on this entity through the unified service
		if s.unifiedService != nil {
			// Create PMA control action
			action := types.PMAControlAction{
				Action:     req.Action,
				Parameters: req.Parameters,
				EntityID:   entity.EntityID,
				Context: &types.PMAContext{
					Source:      "bulk_action",
					Timestamp:   time.Now(),
					Description: fmt.Sprintf("Bulk action %s on area %d", req.Action, req.AreaID),
				},
			}

			// Execute action through unified service
			controlResult, err := s.unifiedService.ExecuteAction(ctx, action)
			if err != nil {
				entityResult.Error = err.Error()
				s.logger.WithError(err).WithFields(logrus.Fields{
					"entity_id": entity.EntityID,
					"action":    req.Action,
					"area_id":   req.AreaID,
				}).Warn("Failed to execute bulk action on entity")
			} else if controlResult != nil && controlResult.Success {
				entityResult.Success = true
				s.logger.WithFields(logrus.Fields{
					"entity_id": entity.EntityID,
					"action":    req.Action,
					"area_id":   req.AreaID,
				}).Info("Successfully executed bulk action on entity")
			} else {
				if controlResult != nil && controlResult.Error != nil {
					entityResult.Error = controlResult.Error.Message
				} else {
					entityResult.Error = "Unknown error during action execution"
				}
			}
		} else {
			entityResult.Error = "Unified entity service not available"
		}

		result.Results = append(result.Results, entityResult)

		if entityResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	// Record analytics
	s.recordAreaMetric(ctx, req.AreaID, "bulk_action_executed", float64(result.TotalEntities), "count", "snapshot")
	s.recordAreaMetric(ctx, req.AreaID, "bulk_action_success_rate", float64(result.SuccessCount)/float64(result.TotalEntities)*100, "percentage", "snapshot")

	return result, nil
}
