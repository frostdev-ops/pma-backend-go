package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// AreaRepository implements repositories.AreaRepository
type AreaRepository struct {
	db *sql.DB
}

// NewAreaRepository creates a new AreaRepository
func NewAreaRepository(db *sql.DB) repositories.AreaRepository {
	return &AreaRepository{db: db}
}

// Area CRUD operations

// CreateArea creates a new area
func (r *AreaRepository) CreateArea(ctx context.Context, area *models.Area) error {
	query := `
		INSERT INTO areas (name, area_id, description, icon, floor_level, parent_area_id, color, is_active, area_type, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		area.Name,
		area.AreaID,
		area.Description,
		area.Icon,
		area.FloorLevel,
		area.ParentAreaID,
		area.Color,
		area.IsActive,
		area.AreaType,
		area.Metadata,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create area: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted area ID: %w", err)
	}

	area.ID = int(id)
	area.CreatedAt = now
	area.UpdatedAt = now
	return nil
}

// GetAreaByID retrieves an area by its internal ID
func (r *AreaRepository) GetAreaByID(ctx context.Context, id int) (*models.Area, error) {
	query := `
		SELECT id, name, area_id, description, icon, floor_level, parent_area_id, color, is_active, area_type, metadata, created_at, updated_at
		FROM areas WHERE id = ?
	`

	var area models.Area
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&area.ID,
		&area.Name,
		&area.AreaID,
		&area.Description,
		&area.Icon,
		&area.FloorLevel,
		&area.ParentAreaID,
		&area.Color,
		&area.IsActive,
		&area.AreaType,
		&area.Metadata,
		&area.CreatedAt,
		&area.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get area by ID: %w", err)
	}

	return &area, nil
}

// GetAreaByAreaID retrieves an area by its external area ID
func (r *AreaRepository) GetAreaByAreaID(ctx context.Context, areaID string) (*models.Area, error) {
	query := `
		SELECT id, name, area_id, description, icon, floor_level, parent_area_id, color, is_active, area_type, metadata, created_at, updated_at
		FROM areas WHERE area_id = ?
	`

	var area models.Area
	err := r.db.QueryRowContext(ctx, query, areaID).Scan(
		&area.ID,
		&area.Name,
		&area.AreaID,
		&area.Description,
		&area.Icon,
		&area.FloorLevel,
		&area.ParentAreaID,
		&area.Color,
		&area.IsActive,
		&area.AreaType,
		&area.Metadata,
		&area.CreatedAt,
		&area.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get area by area ID: %w", err)
	}

	return &area, nil
}

// GetAllAreas retrieves all areas
func (r *AreaRepository) GetAllAreas(ctx context.Context, includeInactive bool) ([]*models.Area, error) {
	query := `
		SELECT id, name, area_id, description, icon, floor_level, parent_area_id, color, is_active, area_type, metadata, created_at, updated_at
		FROM areas
	`

	if !includeInactive {
		query += " WHERE is_active = TRUE"
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all areas: %w", err)
	}
	defer rows.Close()

	var areas []*models.Area
	for rows.Next() {
		var area models.Area
		err := rows.Scan(
			&area.ID,
			&area.Name,
			&area.AreaID,
			&area.Description,
			&area.Icon,
			&area.FloorLevel,
			&area.ParentAreaID,
			&area.Color,
			&area.IsActive,
			&area.AreaType,
			&area.Metadata,
			&area.CreatedAt,
			&area.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area: %w", err)
		}
		areas = append(areas, &area)
	}

	return areas, nil
}

// GetAreasByType retrieves areas by type
func (r *AreaRepository) GetAreasByType(ctx context.Context, areaType string) ([]*models.Area, error) {
	query := `
		SELECT id, name, area_id, description, icon, floor_level, parent_area_id, color, is_active, area_type, metadata, created_at, updated_at
		FROM areas WHERE area_type = ? AND is_active = TRUE
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, areaType)
	if err != nil {
		return nil, fmt.Errorf("failed to get areas by type: %w", err)
	}
	defer rows.Close()

	var areas []*models.Area
	for rows.Next() {
		var area models.Area
		err := rows.Scan(
			&area.ID,
			&area.Name,
			&area.AreaID,
			&area.Description,
			&area.Icon,
			&area.FloorLevel,
			&area.ParentAreaID,
			&area.Color,
			&area.IsActive,
			&area.AreaType,
			&area.Metadata,
			&area.CreatedAt,
			&area.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area: %w", err)
		}
		areas = append(areas, &area)
	}

	return areas, nil
}

// GetAreasByParent retrieves child areas of a parent area
func (r *AreaRepository) GetAreasByParent(ctx context.Context, parentID int) ([]*models.Area, error) {
	query := `
		SELECT id, name, area_id, description, icon, floor_level, parent_area_id, color, is_active, area_type, metadata, created_at, updated_at
		FROM areas WHERE parent_area_id = ? AND is_active = TRUE
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get areas by parent: %w", err)
	}
	defer rows.Close()

	var areas []*models.Area
	for rows.Next() {
		var area models.Area
		err := rows.Scan(
			&area.ID,
			&area.Name,
			&area.AreaID,
			&area.Description,
			&area.Icon,
			&area.FloorLevel,
			&area.ParentAreaID,
			&area.Color,
			&area.IsActive,
			&area.AreaType,
			&area.Metadata,
			&area.CreatedAt,
			&area.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area: %w", err)
		}
		areas = append(areas, &area)
	}

	return areas, nil
}

// GetAreaHierarchy builds the complete area hierarchy
func (r *AreaRepository) GetAreaHierarchy(ctx context.Context) (*models.AreaHierarchy, error) {
	// Get all areas
	areas, err := r.GetAllAreas(ctx, false)
	if err != nil {
		return nil, err
	}

	// Get entity counts by area
	entityCounts, err := r.GetEntityCountsByArea(ctx)
	if err != nil {
		return nil, err
	}

	// Get room counts by area
	roomCounts, err := r.GetRoomCountsByArea(ctx)
	if err != nil {
		return nil, err
	}

	// Build hierarchy map
	areaMap := make(map[int]*models.AreaWithChildren)
	var rootAreas []*models.AreaWithChildren
	maxDepth := 0

	// Create AreaWithChildren for each area
	for _, area := range areas {
		areaWithChildren := &models.AreaWithChildren{
			Area:        *area,
			Children:    []*models.AreaWithChildren{},
			EntityCount: entityCounts[area.ID],
			RoomCount:   roomCounts[area.ID],
		}
		areaMap[area.ID] = areaWithChildren
	}

	// Build parent-child relationships
	for _, area := range areas {
		areaWithChildren := areaMap[area.ID]
		if area.ParentAreaID.Valid {
			parentID := int(area.ParentAreaID.Int64)
			if parent, exists := areaMap[parentID]; exists {
				parent.Children = append(parent.Children, areaWithChildren)
			} else {
				// Parent not found, treat as root
				rootAreas = append(rootAreas, areaWithChildren)
			}
		} else {
			rootAreas = append(rootAreas, areaWithChildren)
		}
	}

	// Calculate max depth
	var calculateDepth func(*models.AreaWithChildren, int) int
	calculateDepth = func(area *models.AreaWithChildren, depth int) int {
		max := depth
		for _, child := range area.Children {
			childMax := calculateDepth(child, depth+1)
			if childMax > max {
				max = childMax
			}
		}
		return max
	}

	for _, rootArea := range rootAreas {
		depth := calculateDepth(rootArea, 1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return &models.AreaHierarchy{
		Root:       rootAreas,
		MaxDepth:   maxDepth,
		TotalAreas: len(areas),
	}, nil
}

// UpdateArea updates an existing area
func (r *AreaRepository) UpdateArea(ctx context.Context, area *models.Area) error {
	query := `
		UPDATE areas 
		SET name = ?, area_id = ?, description = ?, icon = ?, floor_level = ?, parent_area_id = ?, 
		    color = ?, is_active = ?, area_type = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	area.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		area.Name,
		area.AreaID,
		area.Description,
		area.Icon,
		area.FloorLevel,
		area.ParentAreaID,
		area.Color,
		area.IsActive,
		area.AreaType,
		area.Metadata,
		area.UpdatedAt,
		area.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update area: %w", err)
	}

	return nil
}

// DeleteArea deletes an area and its relationships
func (r *AreaRepository) DeleteArea(ctx context.Context, id int) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete room area assignments
	_, err = tx.ExecContext(ctx, "DELETE FROM room_area_assignments WHERE area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete room area assignments: %w", err)
	}

	// Delete area mappings
	_, err = tx.ExecContext(ctx, "DELETE FROM area_mappings WHERE pma_area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete area mappings: %w", err)
	}

	// Delete area settings
	_, err = tx.ExecContext(ctx, "DELETE FROM area_settings WHERE area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete area settings: %w", err)
	}

	// Delete area analytics
	_, err = tx.ExecContext(ctx, "DELETE FROM area_analytics WHERE area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete area analytics: %w", err)
	}

	// Update child areas to remove parent reference
	_, err = tx.ExecContext(ctx, "UPDATE areas SET parent_area_id = NULL WHERE parent_area_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to update child areas: %w", err)
	}

	// Delete the area
	_, err = tx.ExecContext(ctx, "DELETE FROM areas WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete area: %w", err)
	}

	return tx.Commit()
}

// Area mapping operations

// CreateAreaMapping creates a new area mapping
func (r *AreaRepository) CreateAreaMapping(ctx context.Context, mapping *models.AreaMapping) error {
	query := `
		INSERT INTO area_mappings (pma_area_id, external_area_id, external_system, mapping_type, auto_sync, sync_priority, last_synced, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		mapping.PMAAreaID,
		mapping.ExternalAreaID,
		mapping.ExternalSystem,
		mapping.MappingType,
		mapping.AutoSync,
		mapping.SyncPriority,
		mapping.LastSynced,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create area mapping: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted mapping ID: %w", err)
	}

	mapping.ID = int(id)
	mapping.CreatedAt = now
	mapping.UpdatedAt = now
	return nil
}

// GetAreaMapping retrieves an area mapping by ID
func (r *AreaRepository) GetAreaMapping(ctx context.Context, id int) (*models.AreaMapping, error) {
	query := `
		SELECT id, pma_area_id, external_area_id, external_system, mapping_type, auto_sync, sync_priority, last_synced, created_at, updated_at
		FROM area_mappings WHERE id = ?
	`

	var mapping models.AreaMapping
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&mapping.ID,
		&mapping.PMAAreaID,
		&mapping.ExternalAreaID,
		&mapping.ExternalSystem,
		&mapping.MappingType,
		&mapping.AutoSync,
		&mapping.SyncPriority,
		&mapping.LastSynced,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get area mapping: %w", err)
	}

	return &mapping, nil
}

// GetAreaMappingByExternal retrieves an area mapping by external ID and system
func (r *AreaRepository) GetAreaMappingByExternal(ctx context.Context, externalAreaID, externalSystem string) (*models.AreaMapping, error) {
	query := `
		SELECT id, pma_area_id, external_area_id, external_system, mapping_type, auto_sync, sync_priority, last_synced, created_at, updated_at
		FROM area_mappings WHERE external_area_id = ? AND external_system = ?
	`

	var mapping models.AreaMapping
	err := r.db.QueryRowContext(ctx, query, externalAreaID, externalSystem).Scan(
		&mapping.ID,
		&mapping.PMAAreaID,
		&mapping.ExternalAreaID,
		&mapping.ExternalSystem,
		&mapping.MappingType,
		&mapping.AutoSync,
		&mapping.SyncPriority,
		&mapping.LastSynced,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get area mapping by external: %w", err)
	}

	return &mapping, nil
}

// GetAllAreaMappings retrieves all area mappings with details
func (r *AreaRepository) GetAllAreaMappings(ctx context.Context) ([]*models.AreaMappingWithDetails, error) {
	query := `
		SELECT am.id, am.pma_area_id, am.external_area_id, am.external_system, am.mapping_type, 
		       am.auto_sync, am.sync_priority, am.last_synced, am.created_at, am.updated_at,
		       a.name as area_name
		FROM area_mappings am
		JOIN areas a ON am.pma_area_id = a.id
		ORDER BY a.name, am.external_system
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all area mappings: %w", err)
	}
	defer rows.Close()

	var mappings []*models.AreaMappingWithDetails
	for rows.Next() {
		var mapping models.AreaMappingWithDetails
		err := rows.Scan(
			&mapping.ID,
			&mapping.PMAAreaID,
			&mapping.ExternalAreaID,
			&mapping.ExternalSystem,
			&mapping.MappingType,
			&mapping.AutoSync,
			&mapping.SyncPriority,
			&mapping.LastSynced,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
			&mapping.AreaName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area mapping: %w", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, nil
}

// GetAreaMappingsBySystem retrieves area mappings by external system
func (r *AreaRepository) GetAreaMappingsBySystem(ctx context.Context, externalSystem string) ([]*models.AreaMapping, error) {
	query := `
		SELECT id, pma_area_id, external_area_id, external_system, mapping_type, auto_sync, sync_priority, last_synced, created_at, updated_at
		FROM area_mappings WHERE external_system = ?
		ORDER BY external_area_id
	`

	rows, err := r.db.QueryContext(ctx, query, externalSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get area mappings by system: %w", err)
	}
	defer rows.Close()

	var mappings []*models.AreaMapping
	for rows.Next() {
		var mapping models.AreaMapping
		err := rows.Scan(
			&mapping.ID,
			&mapping.PMAAreaID,
			&mapping.ExternalAreaID,
			&mapping.ExternalSystem,
			&mapping.MappingType,
			&mapping.AutoSync,
			&mapping.SyncPriority,
			&mapping.LastSynced,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area mapping: %w", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, nil
}

// GetAreaMappingsByArea retrieves area mappings by area ID
func (r *AreaRepository) GetAreaMappingsByArea(ctx context.Context, areaID int) ([]*models.AreaMapping, error) {
	query := `
		SELECT id, pma_area_id, external_area_id, external_system, mapping_type, auto_sync, sync_priority, last_synced, created_at, updated_at
		FROM area_mappings WHERE pma_area_id = ?
		ORDER BY external_system, external_area_id
	`

	rows, err := r.db.QueryContext(ctx, query, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get area mappings by area: %w", err)
	}
	defer rows.Close()

	var mappings []*models.AreaMapping
	for rows.Next() {
		var mapping models.AreaMapping
		err := rows.Scan(
			&mapping.ID,
			&mapping.PMAAreaID,
			&mapping.ExternalAreaID,
			&mapping.ExternalSystem,
			&mapping.MappingType,
			&mapping.AutoSync,
			&mapping.SyncPriority,
			&mapping.LastSynced,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area mapping: %w", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, nil
}

// UpdateAreaMapping updates an existing area mapping
func (r *AreaRepository) UpdateAreaMapping(ctx context.Context, mapping *models.AreaMapping) error {
	query := `
		UPDATE area_mappings 
		SET mapping_type = ?, auto_sync = ?, sync_priority = ?, last_synced = ?, updated_at = ?
		WHERE id = ?
	`

	mapping.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		mapping.MappingType,
		mapping.AutoSync,
		mapping.SyncPriority,
		mapping.LastSynced,
		mapping.UpdatedAt,
		mapping.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update area mapping: %w", err)
	}

	return nil
}

// DeleteAreaMapping deletes an area mapping
func (r *AreaRepository) DeleteAreaMapping(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM area_mappings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete area mapping: %w", err)
	}
	return nil
}

// Area settings operations

// GetAreaSetting retrieves a specific area setting
func (r *AreaRepository) GetAreaSetting(ctx context.Context, settingKey string, areaID *int) (*models.AreaSetting, error) {
	query := `
		SELECT id, setting_key, setting_value, area_id, is_global, data_type, created_at, updated_at
		FROM area_settings 
		WHERE setting_key = ? AND ((area_id = ? AND is_global = FALSE) OR (area_id IS NULL AND is_global = TRUE AND ? IS NULL))
	`

	var setting models.AreaSetting
	err := r.db.QueryRowContext(ctx, query, settingKey, areaID, areaID).Scan(
		&setting.ID,
		&setting.SettingKey,
		&setting.SettingValue,
		&setting.AreaID,
		&setting.IsGlobal,
		&setting.DataType,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get area setting: %w", err)
	}

	return &setting, nil
}

// GetAreaSettings retrieves all settings for an area or global settings
func (r *AreaRepository) GetAreaSettings(ctx context.Context, areaID *int) ([]*models.AreaSetting, error) {
	var query string
	var args []interface{}

	if areaID == nil {
		query = `
			SELECT id, setting_key, setting_value, area_id, is_global, data_type, created_at, updated_at
			FROM area_settings 
			WHERE is_global = TRUE
			ORDER BY setting_key
		`
	} else {
		query = `
			SELECT id, setting_key, setting_value, area_id, is_global, data_type, created_at, updated_at
			FROM area_settings 
			WHERE area_id = ? AND is_global = FALSE
			ORDER BY setting_key
		`
		args = append(args, *areaID)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get area settings: %w", err)
	}
	defer rows.Close()

	var settings []*models.AreaSetting
	for rows.Next() {
		var setting models.AreaSetting
		err := rows.Scan(
			&setting.ID,
			&setting.SettingKey,
			&setting.SettingValue,
			&setting.AreaID,
			&setting.IsGlobal,
			&setting.DataType,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area setting: %w", err)
		}
		settings = append(settings, &setting)
	}

	return settings, nil
}

// SetAreaSetting creates or updates an area setting
func (r *AreaRepository) SetAreaSetting(ctx context.Context, setting *models.AreaSetting) error {
	query := `
		INSERT INTO area_settings (setting_key, setting_value, area_id, is_global, data_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(setting_key, area_id) DO UPDATE SET
		setting_value = excluded.setting_value,
		data_type = excluded.data_type,
		updated_at = excluded.updated_at
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		setting.SettingKey,
		setting.SettingValue,
		setting.AreaID,
		setting.IsGlobal,
		setting.DataType,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to set area setting: %w", err)
	}

	if setting.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			setting.ID = int(id)
		}
	}

	setting.CreatedAt = now
	setting.UpdatedAt = now
	return nil
}

// DeleteAreaSetting deletes an area setting
func (r *AreaRepository) DeleteAreaSetting(ctx context.Context, settingKey string, areaID *int) error {
	var query string
	var args []interface{}

	if areaID == nil {
		query = "DELETE FROM area_settings WHERE setting_key = ? AND is_global = TRUE"
		args = []interface{}{settingKey}
	} else {
		query = "DELETE FROM area_settings WHERE setting_key = ? AND area_id = ?"
		args = []interface{}{settingKey, *areaID}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete area setting: %w", err)
	}
	return nil
}

// GetGlobalSettings retrieves all global settings as a structured object
func (r *AreaRepository) GetGlobalSettings(ctx context.Context) (*models.AreaSettings, error) {
	settings, err := r.GetAreaSettings(ctx, nil)
	if err != nil {
		return nil, err
	}

	settingsMap := make(map[string]interface{})
	for _, setting := range settings {
		if setting.SettingValue.Valid {
			value := setting.SettingValue.String
			switch setting.DataType {
			case "boolean":
				settingsMap[setting.SettingKey] = value == "true"
			case "integer":
				// Parse integer value
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
		AreaID:   nil,
		Settings: settingsMap,
	}, nil
}

// SetGlobalSettings updates global settings
func (r *AreaRepository) SetGlobalSettings(ctx context.Context, settings *models.AreaSettings) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
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

		query := `
			INSERT INTO area_settings (setting_key, setting_value, area_id, is_global, data_type, created_at, updated_at)
			VALUES (?, ?, NULL, TRUE, ?, ?, ?)
			ON CONFLICT(setting_key, area_id) DO UPDATE SET
			setting_value = excluded.setting_value,
			data_type = excluded.data_type,
			updated_at = excluded.updated_at
		`

		_, err = tx.ExecContext(ctx, query, key, valueStr, dataType, now, now)
		if err != nil {
			return fmt.Errorf("failed to set global setting %s: %w", key, err)
		}
	}

	return tx.Commit()
}

// Area analytics operations

// CreateAreaAnalytic creates a new area analytic record
func (r *AreaRepository) CreateAreaAnalytic(ctx context.Context, analytic *models.AreaAnalytic) error {
	query := `
		INSERT INTO area_analytics (area_id, metric_name, metric_value, metric_unit, aggregation_type, time_period, recorded_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		analytic.AreaID,
		analytic.MetricName,
		analytic.MetricValue,
		analytic.MetricUnit,
		analytic.AggregationType,
		analytic.TimePeriod,
		analytic.RecordedAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create area analytic: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted analytic ID: %w", err)
	}

	analytic.ID = int(id)
	analytic.CreatedAt = now
	return nil
}

// GetAreaAnalytics retrieves analytics for an area within a time range
func (r *AreaRepository) GetAreaAnalytics(ctx context.Context, areaID int, startDate, endDate *time.Time) ([]*models.AreaAnalytic, error) {
	query := `
		SELECT id, area_id, metric_name, metric_value, metric_unit, aggregation_type, time_period, recorded_at, created_at
		FROM area_analytics 
		WHERE area_id = ?
	`
	args := []interface{}{areaID}

	if startDate != nil {
		query += " AND recorded_at >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += " AND recorded_at <= ?"
		args = append(args, *endDate)
	}

	query += " ORDER BY recorded_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get area analytics: %w", err)
	}
	defer rows.Close()

	var analytics []*models.AreaAnalytic
	for rows.Next() {
		var analytic models.AreaAnalytic
		err := rows.Scan(
			&analytic.ID,
			&analytic.AreaID,
			&analytic.MetricName,
			&analytic.MetricValue,
			&analytic.MetricUnit,
			&analytic.AggregationType,
			&analytic.TimePeriod,
			&analytic.RecordedAt,
			&analytic.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area analytic: %w", err)
		}
		analytics = append(analytics, &analytic)
	}

	return analytics, nil
}

// GetAreaAnalyticsByMetric retrieves analytics by metric name within a time range
func (r *AreaRepository) GetAreaAnalyticsByMetric(ctx context.Context, metricName string, startDate, endDate *time.Time) ([]*models.AreaAnalytic, error) {
	query := `
		SELECT id, area_id, metric_name, metric_value, metric_unit, aggregation_type, time_period, recorded_at, created_at
		FROM area_analytics 
		WHERE metric_name = ?
	`
	args := []interface{}{metricName}

	if startDate != nil {
		query += " AND recorded_at >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += " AND recorded_at <= ?"
		args = append(args, *endDate)
	}

	query += " ORDER BY area_id, recorded_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get area analytics by metric: %w", err)
	}
	defer rows.Close()

	var analytics []*models.AreaAnalytic
	for rows.Next() {
		var analytic models.AreaAnalytic
		err := rows.Scan(
			&analytic.ID,
			&analytic.AreaID,
			&analytic.MetricName,
			&analytic.MetricValue,
			&analytic.MetricUnit,
			&analytic.AggregationType,
			&analytic.TimePeriod,
			&analytic.RecordedAt,
			&analytic.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area analytic: %w", err)
		}
		analytics = append(analytics, &analytic)
	}

	return analytics, nil
}

// GetAreaAnalyticsSummary generates analytics summary for areas
func (r *AreaRepository) GetAreaAnalyticsSummary(ctx context.Context, areaIDs []int) ([]*models.AreaAnalyticsSummary, error) {
	var whereClause string
	var args []interface{}

	if len(areaIDs) > 0 {
		placeholders := make([]string, len(areaIDs))
		for i, id := range areaIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		whereClause = fmt.Sprintf("WHERE a.id IN (%s)", strings.Join(placeholders, ","))
	}

	query := fmt.Sprintf(`
		SELECT 
			a.id as area_id,
			a.name as area_name,
			COALESCE(ec.entity_count, 0) as entity_count,
			COALESCE(rc.room_count, 0) as room_count,
			0 as device_count,
			0 as active_devices,
			0.0 as energy_usage,
			0.8 as health_score
		FROM areas a
		LEFT JOIN (
			SELECT 
				raa.area_id,
				COUNT(DISTINCT e.entity_id) as entity_count
			FROM room_area_assignments raa
			JOIN entities e ON e.room_id = raa.room_id
			WHERE raa.assignment_type = 'primary'
			GROUP BY raa.area_id
		) ec ON a.id = ec.area_id
		LEFT JOIN (
			SELECT 
				area_id,
				COUNT(*) as room_count
			FROM room_area_assignments
			WHERE assignment_type = 'primary'
			GROUP BY area_id
		) rc ON a.id = rc.area_id
		%s
		ORDER BY a.name
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get area analytics summary: %w", err)
	}
	defer rows.Close()

	var summaries []*models.AreaAnalyticsSummary
	for rows.Next() {
		var summary models.AreaAnalyticsSummary
		err := rows.Scan(
			&summary.AreaID,
			&summary.AreaName,
			&summary.EntityCount,
			&summary.RoomCount,
			&summary.DeviceCount,
			&summary.ActiveDevices,
			&summary.EnergyUsage,
			&summary.HealthScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area analytics summary: %w", err)
		}

		summary.Metrics = make(map[string]interface{})
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// DeleteOldAnalytics deletes analytics older than specified days
func (r *AreaRepository) DeleteOldAnalytics(ctx context.Context, olderThanDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)
	_, err := r.db.ExecContext(ctx, "DELETE FROM area_analytics WHERE recorded_at < ?", cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old analytics: %w", err)
	}
	return nil
}

// Area sync log operations

// CreateSyncLog creates a new sync log entry
func (r *AreaRepository) CreateSyncLog(ctx context.Context, syncLog *models.AreaSyncLog) error {
	query := `
		INSERT INTO area_sync_log (sync_type, external_system, status, areas_processed, areas_updated, areas_created, areas_deleted, error_message, sync_details, started_at, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		syncLog.SyncType,
		syncLog.ExternalSystem,
		syncLog.Status,
		syncLog.AreasProcessed,
		syncLog.AreasUpdated,
		syncLog.AreasCreated,
		syncLog.AreasDeleted,
		syncLog.ErrorMessage,
		syncLog.SyncDetails,
		syncLog.StartedAt,
		syncLog.CompletedAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted sync log ID: %w", err)
	}

	syncLog.ID = int(id)
	syncLog.CreatedAt = now
	return nil
}

// GetSyncLog retrieves a sync log by ID
func (r *AreaRepository) GetSyncLog(ctx context.Context, id int) (*models.AreaSyncLog, error) {
	query := `
		SELECT id, sync_type, external_system, status, areas_processed, areas_updated, areas_created, areas_deleted, error_message, sync_details, started_at, completed_at, created_at
		FROM area_sync_log WHERE id = ?
	`

	var syncLog models.AreaSyncLog
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&syncLog.ID,
		&syncLog.SyncType,
		&syncLog.ExternalSystem,
		&syncLog.Status,
		&syncLog.AreasProcessed,
		&syncLog.AreasUpdated,
		&syncLog.AreasCreated,
		&syncLog.AreasDeleted,
		&syncLog.ErrorMessage,
		&syncLog.SyncDetails,
		&syncLog.StartedAt,
		&syncLog.CompletedAt,
		&syncLog.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get sync log: %w", err)
	}

	return &syncLog, nil
}

// GetSyncLogsBySystem retrieves sync logs by external system
func (r *AreaRepository) GetSyncLogsBySystem(ctx context.Context, externalSystem string, limit int) ([]*models.AreaSyncLog, error) {
	query := `
		SELECT id, sync_type, external_system, status, areas_processed, areas_updated, areas_created, areas_deleted, error_message, sync_details, started_at, completed_at, created_at
		FROM area_sync_log 
		WHERE external_system = ?
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, externalSystem, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync logs by system: %w", err)
	}
	defer rows.Close()

	var syncLogs []*models.AreaSyncLog
	for rows.Next() {
		var syncLog models.AreaSyncLog
		err := rows.Scan(
			&syncLog.ID,
			&syncLog.SyncType,
			&syncLog.ExternalSystem,
			&syncLog.Status,
			&syncLog.AreasProcessed,
			&syncLog.AreasUpdated,
			&syncLog.AreasCreated,
			&syncLog.AreasDeleted,
			&syncLog.ErrorMessage,
			&syncLog.SyncDetails,
			&syncLog.StartedAt,
			&syncLog.CompletedAt,
			&syncLog.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}
		syncLogs = append(syncLogs, &syncLog)
	}

	return syncLogs, nil
}

// GetLastSyncTime retrieves the last successful sync time for a system
func (r *AreaRepository) GetLastSyncTime(ctx context.Context, externalSystem string) (*time.Time, error) {
	query := `
		SELECT MAX(completed_at) 
		FROM area_sync_log 
		WHERE external_system = ? AND status = 'success' AND completed_at IS NOT NULL
	`

	var lastSync sql.NullTime
	err := r.db.QueryRowContext(ctx, query, externalSystem).Scan(&lastSync)
	if err != nil {
		return nil, fmt.Errorf("failed to get last sync time: %w", err)
	}

	if lastSync.Valid {
		return &lastSync.Time, nil
	}
	return nil, nil
}

// UpdateSyncLog updates an existing sync log
func (r *AreaRepository) UpdateSyncLog(ctx context.Context, syncLog *models.AreaSyncLog) error {
	query := `
		UPDATE area_sync_log 
		SET status = ?, areas_processed = ?, areas_updated = ?, areas_created = ?, areas_deleted = ?, error_message = ?, sync_details = ?, completed_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		syncLog.Status,
		syncLog.AreasProcessed,
		syncLog.AreasUpdated,
		syncLog.AreasCreated,
		syncLog.AreasDeleted,
		syncLog.ErrorMessage,
		syncLog.SyncDetails,
		syncLog.CompletedAt,
		syncLog.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	return nil
}

// DeleteOldSyncLogs deletes sync logs older than specified days
func (r *AreaRepository) DeleteOldSyncLogs(ctx context.Context, olderThanDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)
	_, err := r.db.ExecContext(ctx, "DELETE FROM area_sync_log WHERE created_at < ?", cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old sync logs: %w", err)
	}
	return nil
}

// Room-area assignment operations

// CreateRoomAreaAssignment creates a new room-area assignment
func (r *AreaRepository) CreateRoomAreaAssignment(ctx context.Context, assignment *models.RoomAreaAssignment) error {
	query := `
		INSERT INTO room_area_assignments (room_id, area_id, assignment_type, confidence_score, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		assignment.RoomID,
		assignment.AreaID,
		assignment.AssignmentType,
		assignment.ConfidenceScore,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create room area assignment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted assignment ID: %w", err)
	}

	assignment.ID = int(id)
	assignment.CreatedAt = now
	assignment.UpdatedAt = now
	return nil
}

// GetRoomAreaAssignments retrieves area assignments for a room
func (r *AreaRepository) GetRoomAreaAssignments(ctx context.Context, roomID int) ([]*models.RoomAreaAssignment, error) {
	query := `
		SELECT id, room_id, area_id, assignment_type, confidence_score, created_at, updated_at
		FROM room_area_assignments 
		WHERE room_id = ?
		ORDER BY assignment_type, confidence_score DESC
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room area assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*models.RoomAreaAssignment
	for rows.Next() {
		var assignment models.RoomAreaAssignment
		err := rows.Scan(
			&assignment.ID,
			&assignment.RoomID,
			&assignment.AreaID,
			&assignment.AssignmentType,
			&assignment.ConfidenceScore,
			&assignment.CreatedAt,
			&assignment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room area assignment: %w", err)
		}
		assignments = append(assignments, &assignment)
	}

	return assignments, nil
}

// GetAreaRoomAssignments retrieves room assignments for an area
func (r *AreaRepository) GetAreaRoomAssignments(ctx context.Context, areaID int) ([]*models.RoomAreaAssignment, error) {
	query := `
		SELECT id, room_id, area_id, assignment_type, confidence_score, created_at, updated_at
		FROM room_area_assignments 
		WHERE area_id = ?
		ORDER BY assignment_type, confidence_score DESC
	`

	rows, err := r.db.QueryContext(ctx, query, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get area room assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*models.RoomAreaAssignment
	for rows.Next() {
		var assignment models.RoomAreaAssignment
		err := rows.Scan(
			&assignment.ID,
			&assignment.RoomID,
			&assignment.AreaID,
			&assignment.AssignmentType,
			&assignment.ConfidenceScore,
			&assignment.CreatedAt,
			&assignment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan area room assignment: %w", err)
		}
		assignments = append(assignments, &assignment)
	}

	return assignments, nil
}

// UpdateRoomAreaAssignment updates an existing room-area assignment
func (r *AreaRepository) UpdateRoomAreaAssignment(ctx context.Context, assignment *models.RoomAreaAssignment) error {
	query := `
		UPDATE room_area_assignments 
		SET assignment_type = ?, confidence_score = ?, updated_at = ?
		WHERE id = ?
	`

	assignment.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		assignment.AssignmentType,
		assignment.ConfidenceScore,
		assignment.UpdatedAt,
		assignment.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update room area assignment: %w", err)
	}

	return nil
}

// DeleteRoomAreaAssignment deletes a room-area assignment
func (r *AreaRepository) DeleteRoomAreaAssignment(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM room_area_assignments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete room area assignment: %w", err)
	}
	return nil
}

// DeleteRoomAreaAssignmentsByRoom deletes all area assignments for a room
func (r *AreaRepository) DeleteRoomAreaAssignmentsByRoom(ctx context.Context, roomID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM room_area_assignments WHERE room_id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room area assignments by room: %w", err)
	}
	return nil
}

// DeleteRoomAreaAssignmentsByArea deletes all room assignments for an area
func (r *AreaRepository) DeleteRoomAreaAssignmentsByArea(ctx context.Context, areaID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM room_area_assignments WHERE area_id = ?", areaID)
	if err != nil {
		return fmt.Errorf("failed to delete room area assignments by area: %w", err)
	}
	return nil
}

// Status and statistics

// GetAreaStatus generates overall area system status
func (r *AreaRepository) GetAreaStatus(ctx context.Context) (*models.AreaStatus, error) {
	// Get area counts
	var totalAreas, activeAreas int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM areas").Scan(&totalAreas)
	if err != nil {
		return nil, fmt.Errorf("failed to get total areas: %w", err)
	}

	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM areas WHERE is_active = TRUE").Scan(&activeAreas)
	if err != nil {
		return nil, fmt.Errorf("failed to get active areas: %w", err)
	}

	// Get mapping counts
	var mappedAreas int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT pma_area_id) FROM area_mappings").Scan(&mappedAreas)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped areas: %w", err)
	}

	// Get room counts
	var totalRooms, assignedRooms int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM rooms").Scan(&totalRooms)
	if err != nil {
		return nil, fmt.Errorf("failed to get total rooms: %w", err)
	}

	err = r.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT room_id) FROM room_area_assignments").Scan(&assignedRooms)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned rooms: %w", err)
	}

	// Get entity counts
	var totalEntities, entitiesWithAreas int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM entities").Scan(&totalEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to get total entities: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.entity_id) 
		FROM entities e 
		JOIN room_area_assignments raa ON e.room_id = raa.room_id
	`).Scan(&entitiesWithAreas)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities with areas: %w", err)
	}

	// Get last sync time
	lastSyncTime, err := r.GetLastSyncTime(ctx, models.ExternalSystemHomeAssistant)
	if err != nil {
		lastSyncTime = nil // Not critical, continue
	}

	// Get sync status
	var syncStatus string = "unknown"
	var lastStatus sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT status 
		FROM area_sync_log 
		WHERE external_system = ? 
		ORDER BY started_at DESC 
		LIMIT 1
	`, models.ExternalSystemHomeAssistant).Scan(&lastStatus)
	if err == nil && lastStatus.Valid {
		syncStatus = lastStatus.String
	}

	// Get global settings
	globalSettings, err := r.GetGlobalSettings(ctx)
	var syncEnabled bool = true
	if err == nil && globalSettings != nil {
		if enabled, ok := globalSettings.Settings["sync_enabled"].(bool); ok {
			syncEnabled = enabled
		}
	}

	// Calculate health score (simplified)
	healthScore := 0.5 // Base score
	if totalAreas > 0 {
		healthScore += float64(activeAreas) / float64(totalAreas) * 0.2
		healthScore += float64(mappedAreas) / float64(totalAreas) * 0.3
	}
	if totalRooms > 0 {
		healthScore += float64(assignedRooms) / float64(totalRooms) * 0.3
	}
	if totalEntities > 0 {
		healthScore += float64(entitiesWithAreas) / float64(totalEntities) * 0.2
	}

	status := &models.AreaStatus{
		TotalAreas:           totalAreas,
		ActiveAreas:          activeAreas,
		MappedAreas:          mappedAreas,
		UnmappedAreas:        totalAreas - mappedAreas,
		TotalRooms:           totalRooms,
		AssignedRooms:        assignedRooms,
		UnassignedRooms:      totalRooms - assignedRooms,
		TotalEntities:        totalEntities,
		EntitiesWithAreas:    entitiesWithAreas,
		EntitiesWithoutAreas: totalEntities - entitiesWithAreas,
		LastSyncTime:         lastSyncTime,
		SyncStatus:           syncStatus,
		IsConnected:          true, // TODO: Check actual HA connection
		SyncEnabled:          syncEnabled,
		ExternalSystems:      []string{models.ExternalSystemHomeAssistant},
		HealthScore:          healthScore,
		Issues:               []string{},
		Recommendations:      []string{},
	}

	// Add recommendations based on status
	if status.UnmappedAreas > 0 {
		status.Recommendations = append(status.Recommendations, fmt.Sprintf("%d areas need mapping to external systems", status.UnmappedAreas))
	}
	if status.UnassignedRooms > 0 {
		status.Recommendations = append(status.Recommendations, fmt.Sprintf("%d rooms need area assignment", status.UnassignedRooms))
	}
	if status.EntitiesWithoutAreas > 0 {
		status.Recommendations = append(status.Recommendations, fmt.Sprintf("%d entities are not assigned to areas", status.EntitiesWithoutAreas))
	}

	return status, nil
}

// GetEntityCountsByArea returns entity counts grouped by area
func (r *AreaRepository) GetEntityCountsByArea(ctx context.Context) (map[int]int, error) {
	query := `
		SELECT raa.area_id, COUNT(DISTINCT e.entity_id) as entity_count
		FROM room_area_assignments raa
		LEFT JOIN entities e ON e.room_id = raa.room_id
		WHERE raa.assignment_type = 'primary'
		GROUP BY raa.area_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity counts by area: %w", err)
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var areaID, entityCount int
		err := rows.Scan(&areaID, &entityCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity count: %w", err)
		}
		counts[areaID] = entityCount
	}

	return counts, nil
}

// GetRoomCountsByArea returns room counts grouped by area
func (r *AreaRepository) GetRoomCountsByArea(ctx context.Context) (map[int]int, error) {
	query := `
		SELECT area_id, COUNT(*) as room_count
		FROM room_area_assignments
		WHERE assignment_type = 'primary'
		GROUP BY area_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get room counts by area: %w", err)
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var areaID, roomCount int
		err := rows.Scan(&areaID, &roomCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room count: %w", err)
		}
		counts[areaID] = roomCount
	}

	return counts, nil
}
