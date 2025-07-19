package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type PMARepository struct {
	db  *sqlx.DB
	log *logrus.Logger
}

func NewPMARepository(db *sqlx.DB, log *logrus.Logger) *PMARepository {
	return &PMARepository{
		db:  db,
		log: log,
	}
}

// Entity operations
func (r *PMARepository) GetAllUnifiedEntities(ctx context.Context, filter *models.EntityFilter) ([]*models.UnifiedEntity, error) {
	query := `
		SELECT 
			e.entity_id, e.friendly_name, e.domain, e.state, e.attributes, e.last_updated, e.room_id,
			COALESCE(em.source, 'homeassistant') as source_type,
			COALESCE(em.source_entity_id, e.entity_id) as source_id,
			COALESCE(em.metadata, '{}') as metadata,
			e.last_updated as last_unified,
			CASE WHEN e.available THEN 'synced' ELSE 'error' END as sync_status,
			CASE WHEN e.available THEN 'available' ELSE 'unavailable' END as availability_status,
			NULL as response_time, 0 as error_count, NULL as last_error, NULL as last_error_time
		FROM entities e
		LEFT JOIN entity_metadata em ON e.entity_id = em.entity_id`

	args := []interface{}{}
	conditions := []string{}

	if filter != nil {
		if filter.Domain != nil {
			conditions = append(conditions, "e.domain = ?")
			args = append(args, *filter.Domain)
		}
		if filter.RoomID != nil {
			conditions = append(conditions, "e.room_id = ?")
			args = append(args, *filter.RoomID)
		}
		if filter.SourceType != nil {
			conditions = append(conditions, "COALESCE(em.source, 'homeassistant') = ?")
			args = append(args, string(*filter.SourceType))
		}
		if filter.SyncStatus != nil {
			conditions = append(conditions, "CASE WHEN e.available THEN 'synced' ELSE 'error' END = ?")
			args = append(args, *filter.SyncStatus)
		}
		if filter.Available != nil {
			if *filter.Available {
				conditions = append(conditions, "COALESCE(e.available, true) = true")
			} else {
				conditions = append(conditions, "COALESCE(e.available, true) = false")
			}
		}
		if filter.WithErrors != nil {
			if *filter.WithErrors {
				conditions = append(conditions, "COALESCE(e.available, true) = false")
			} else {
				conditions = append(conditions, "COALESCE(e.available, true) = true")
			}
		}
		if filter.LastUpdatedSince != nil {
			conditions = append(conditions, "e.last_updated >= ?")
			args = append(args, *filter.LastUpdatedSince)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY e.domain, e.entity_id"

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query unified entities: %w", err)
	}
	defer rows.Close()

	var entities []*models.UnifiedEntity
	for rows.Next() {
		entity := &models.UnifiedEntity{}
		err := rows.StructScan(entity)
		if err != nil {
			r.log.WithError(err).Error("Failed to scan unified entity")
			continue
		}

		// Load relationships if requested
		if filter != nil {
			if filter.IncludeRoom && entity.RoomID.Valid {
				room, err := r.GetUnifiedRoomByID(ctx, int(entity.RoomID.Int64))
				if err == nil {
					entity.Room = room
				}
			}
			if filter.IncludeDevice {
				device, err := r.GetDeviceByEntityID(ctx, entity.EntityID)
				if err == nil {
					entity.Device = device
				}
			}
		}

		entities = append(entities, entity)
	}

	return entities, nil
}

func (r *PMARepository) GetUnifiedEntityByID(ctx context.Context, entityID string) (*models.UnifiedEntity, error) {
	query := `
		SELECT 
			e.entity_id, e.friendly_name, e.domain, e.state, e.attributes, e.last_updated, e.room_id,
			COALESCE(em.source, 'homeassistant') as source_type,
			COALESCE(em.source_entity_id, e.entity_id) as source_id,
			COALESCE(em.metadata, '{}') as metadata,
			e.last_updated as last_unified,
			CASE WHEN e.available THEN 'synced' ELSE 'error' END as sync_status,
			CASE WHEN e.available THEN 'available' ELSE 'unavailable' END as availability_status,
			NULL as response_time, 0 as error_count, NULL as last_error, NULL as last_error_time
		FROM entities e
		LEFT JOIN entity_metadata em ON e.entity_id = em.entity_id
		WHERE e.entity_id = ?`

	entity := &models.UnifiedEntity{}
	err := r.db.GetContext(ctx, entity, query, entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get unified entity: %w", err)
	}

	return entity, nil
}

func (r *PMARepository) CreateOrUpdateUnifiedEntity(ctx context.Context, entity *models.UnifiedEntity) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert base entity
	entityQuery := `
		INSERT INTO entities (entity_id, friendly_name, domain, state, attributes, last_updated, room_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id) DO UPDATE SET
			friendly_name = excluded.friendly_name,
			state = excluded.state,
			attributes = excluded.attributes,
			last_updated = excluded.last_updated,
			room_id = excluded.room_id`

	_, err = tx.ExecContext(ctx, entityQuery,
		entity.EntityID, entity.FriendlyName, entity.Domain, entity.State,
		entity.Attributes, entity.LastUpdated, entity.RoomID)
	if err != nil {
		return fmt.Errorf("failed to upsert entity: %w", err)
	}

	// Upsert source mapping
	mappingQuery := `
		INSERT INTO entity_source_mappings (entity_id, source_type, source_id, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id, source_type) DO UPDATE SET
			source_id = excluded.source_id,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at`

	now := time.Now()
	_, err = tx.ExecContext(ctx, mappingQuery,
		entity.EntityID, string(entity.SourceType), entity.SourceID,
		entity.Metadata, now, now)
	if err != nil {
		return fmt.Errorf("failed to upsert entity source mapping: %w", err)
	}

	// Upsert unified entity data
	unifiedQuery := `
		INSERT INTO unified_entities (
			entity_id, last_unified, sync_status, availability_status,
			response_time, error_count, last_error, last_error_time
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id) DO UPDATE SET
			last_unified = excluded.last_unified,
			sync_status = excluded.sync_status,
			availability_status = excluded.availability_status,
			response_time = excluded.response_time,
			error_count = excluded.error_count,
			last_error = excluded.last_error,
			last_error_time = excluded.last_error_time`

	_, err = tx.ExecContext(ctx, unifiedQuery,
		entity.EntityID, entity.LastUnified, entity.SyncStatus, entity.AvailabilityStatus,
		entity.ResponseTime, entity.ErrorCount, entity.LastError, entity.LastErrorTime)
	if err != nil {
		return fmt.Errorf("failed to upsert unified entity: %w", err)
	}

	return tx.Commit()
}

func (r *PMARepository) UpdateEntityMetrics(ctx context.Context, entityID string, responseTime float64, errorCount int, lastError string) error {
	query := `
		INSERT INTO unified_entities (entity_id, response_time, error_count, last_error, last_error_time, last_unified)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id) DO UPDATE SET
			response_time = excluded.response_time,
			error_count = excluded.error_count,
			last_error = excluded.last_error,
			last_error_time = excluded.last_error_time,
			last_unified = excluded.last_unified`

	var lastErrorTime *time.Time
	if lastError != "" {
		now := time.Now()
		lastErrorTime = &now
	}

	_, err := r.db.ExecContext(ctx, query, entityID, responseTime, errorCount, lastError, lastErrorTime, time.Now())
	return err
}

// Room operations
func (r *PMARepository) GetAllUnifiedRooms(ctx context.Context, filter *models.RoomFilter) ([]*models.UnifiedRoom, error) {
	query := `
		SELECT 
			r.id, r.name, r.home_assistant_area_id, r.icon, r.description, r.created_at, r.updated_at,
			COALESCE(ur.entity_count, 0) as entity_count,
			COALESCE(ur.active_entity_count, 0) as active_entity_count,
			ur.last_activity,
			COALESCE(ur.sync_status, 'synced') as sync_status,
			COALESCE(ur.metadata, '{}') as metadata,
			ur.temperature_sensor, ur.humidity_sensor, ur.current_temp, ur.current_humidity,
			ur.power_consumption, ur.energy_today
		FROM rooms r
		LEFT JOIN unified_rooms ur ON r.id = ur.room_id`

	args := []interface{}{}
	conditions := []string{}

	if filter != nil {
		if filter.HasEntities != nil {
			if *filter.HasEntities {
				conditions = append(conditions, "COALESCE(ur.entity_count, 0) > 0")
			} else {
				conditions = append(conditions, "COALESCE(ur.entity_count, 0) = 0")
			}
		}
		if filter.SyncStatus != nil {
			conditions = append(conditions, "COALESCE(ur.sync_status, 'synced') = ?")
			args = append(args, *filter.SyncStatus)
		}
		if filter.LastActivitySince != nil {
			conditions = append(conditions, "ur.last_activity >= ?")
			args = append(args, *filter.LastActivitySince)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY r.name"

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query unified rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*models.UnifiedRoom
	for rows.Next() {
		room := &models.UnifiedRoom{}
		err := rows.StructScan(room)
		if err != nil {
			r.log.WithError(err).Error("Failed to scan unified room")
			continue
		}

		// Load entities if requested
		if filter != nil && filter.IncludeEntities {
			entities, err := r.GetUnifiedEntitiesByRoomID(ctx, room.ID)
			if err == nil {
				room.Entities = entities
			}
		}

		rooms = append(rooms, room)
	}

	return rooms, nil
}

func (r *PMARepository) GetUnifiedRoomByID(ctx context.Context, roomID int) (*models.UnifiedRoom, error) {
	query := `
		SELECT 
			r.id, r.name, r.home_assistant_area_id, r.icon, r.description, r.created_at, r.updated_at,
			COALESCE(ur.entity_count, 0) as entity_count,
			COALESCE(ur.active_entity_count, 0) as active_entity_count,
			ur.last_activity,
			COALESCE(ur.sync_status, 'synced') as sync_status,
			COALESCE(ur.metadata, '{}') as metadata,
			ur.temperature_sensor, ur.humidity_sensor, ur.current_temp, ur.current_humidity,
			ur.power_consumption, ur.energy_today
		FROM rooms r
		LEFT JOIN unified_rooms ur ON r.id = ur.room_id
		WHERE r.id = ?`

	room := &models.UnifiedRoom{}
	err := r.db.GetContext(ctx, room, query, roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get unified room: %w", err)
	}

	return room, nil
}

func (r *PMARepository) GetUnifiedEntitiesByRoomID(ctx context.Context, roomID int) ([]*models.UnifiedEntity, error) {
	filter := &models.EntityFilter{
		RoomID: &roomID,
	}
	return r.GetAllUnifiedEntities(ctx, filter)
}

func (r *PMARepository) CreateOrUpdateUnifiedRoom(ctx context.Context, room *models.UnifiedRoom) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert base room
	roomQuery := `
		INSERT INTO rooms (id, name, home_assistant_area_id, icon, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			home_assistant_area_id = excluded.home_assistant_area_id,
			icon = excluded.icon,
			description = excluded.description,
			updated_at = excluded.updated_at`

	_, err = tx.ExecContext(ctx, roomQuery,
		room.ID, room.Name, room.HomeAssistantAreaID, room.Icon, room.Description,
		room.CreatedAt, room.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert room: %w", err)
	}

	// Upsert unified room data
	unifiedQuery := `
		INSERT INTO unified_rooms (
			room_id, entity_count, active_entity_count, last_activity, sync_status, metadata,
			temperature_sensor, humidity_sensor, current_temp, current_humidity,
			power_consumption, energy_today
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(room_id) DO UPDATE SET
			entity_count = excluded.entity_count,
			active_entity_count = excluded.active_entity_count,
			last_activity = excluded.last_activity,
			sync_status = excluded.sync_status,
			metadata = excluded.metadata,
			temperature_sensor = excluded.temperature_sensor,
			humidity_sensor = excluded.humidity_sensor,
			current_temp = excluded.current_temp,
			current_humidity = excluded.current_humidity,
			power_consumption = excluded.power_consumption,
			energy_today = excluded.energy_today`

	_, err = tx.ExecContext(ctx, unifiedQuery,
		room.ID, room.EntityCount, room.ActiveEntityCount, room.LastActivity, room.SyncStatus,
		room.Metadata, room.TemperatureSensor, room.HumiditySensor, room.CurrentTemp,
		room.CurrentHumidity, room.PowerConsumption, room.EnergyToday)
	if err != nil {
		return fmt.Errorf("failed to upsert unified room: %w", err)
	}

	return tx.Commit()
}

func (r *PMARepository) UpdateRoomEntityCounts(ctx context.Context, roomID int) error {
	query := `
		INSERT INTO unified_rooms (room_id, entity_count, active_entity_count)
		VALUES (?, 
			(SELECT COUNT(*) FROM entities WHERE room_id = ?),
			(SELECT COUNT(*) FROM entities e 
			 LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id 
			 WHERE e.room_id = ? AND COALESCE(ue.availability_status, 'available') = 'available')
		)
		ON CONFLICT(room_id) DO UPDATE SET
			entity_count = excluded.entity_count,
			active_entity_count = excluded.active_entity_count`

	_, err := r.db.ExecContext(ctx, query, roomID, roomID, roomID)
	return err
}

// Device operations
func (r *PMARepository) GetDeviceByEntityID(ctx context.Context, entityID string) (*models.Device, error) {
	query := `
		SELECT d.* FROM devices d
		JOIN device_entities de ON d.id = de.device_id
		WHERE de.entity_id = ?`

	device := &models.Device{}
	err := r.db.GetContext(ctx, device, query, entityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get device by entity ID: %w", err)
	}

	return device, nil
}

// Statistics and metrics
func (r *PMARepository) GetEntityCounts(ctx context.Context) (*models.EntityCounts, error) {
	counts := &models.EntityCounts{
		ByDomain: make(map[string]int),
		BySource: make(map[models.EntitySourceType]int),
		ByStatus: make(map[string]int),
	}

	// Total count
	err := r.db.GetContext(ctx, &counts.Total, "SELECT COUNT(*) FROM entities")
	if err != nil {
		return nil, fmt.Errorf("failed to get total entity count: %w", err)
	}

	// By domain
	domainRows, err := r.db.QueryxContext(ctx, "SELECT domain, COUNT(*) as count FROM entities GROUP BY domain")
	if err != nil {
		return nil, fmt.Errorf("failed to get domain counts: %w", err)
	}
	defer domainRows.Close()

	for domainRows.Next() {
		var domain string
		var count int
		if err := domainRows.Scan(&domain, &count); err == nil {
			counts.ByDomain[domain] = count
		}
	}

	// By source
	sourceRows, err := r.db.QueryxContext(ctx, `
		SELECT COALESCE(esm.source_type, 'home_assistant') as source_type, COUNT(*) as count 
		FROM entities e
		LEFT JOIN entity_source_mappings esm ON e.entity_id = esm.entity_id
		GROUP BY source_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to get source counts: %w", err)
	}
	defer sourceRows.Close()

	for sourceRows.Next() {
		var source string
		var count int
		if err := sourceRows.Scan(&source, &count); err == nil {
			counts.BySource[models.EntitySourceType(source)] = count
		}
	}

	// Availability counts
	err = r.db.GetContext(ctx, &counts.Available, `
		SELECT COUNT(*) FROM entities e
		LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id
		WHERE COALESCE(ue.availability_status, 'available') = 'available'`)
	if err != nil {
		return nil, fmt.Errorf("failed to get available count: %w", err)
	}

	counts.Unavailable = counts.Total - counts.Available

	// Error counts
	err = r.db.GetContext(ctx, &counts.WithErrors, `
		SELECT COUNT(*) FROM unified_entities WHERE error_count > 0`)
	if err != nil {
		return nil, fmt.Errorf("failed to get error count: %w", err)
	}

	return counts, nil
}

func (r *PMARepository) GetRoomCounts(ctx context.Context) (*models.RoomCounts, error) {
	counts := &models.RoomCounts{}

	// Total count
	err := r.db.GetContext(ctx, &counts.Total, "SELECT COUNT(*) FROM rooms")
	if err != nil {
		return nil, fmt.Errorf("failed to get total room count: %w", err)
	}

	// With entities
	err = r.db.GetContext(ctx, &counts.WithEntities, `
		SELECT COUNT(DISTINCT room_id) FROM entities WHERE room_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms with entities count: %w", err)
	}

	counts.WithoutEntities = counts.Total - counts.WithEntities

	// Synced count
	err = r.db.GetContext(ctx, &counts.Synced, `
		SELECT COUNT(*) FROM unified_rooms WHERE sync_status = 'synced'`)
	if err != nil {
		return nil, fmt.Errorf("failed to get synced room count: %w", err)
	}

	counts.NotSynced = counts.Total - counts.Synced

	return counts, nil
}

// Settings operations
func (r *PMARepository) GetPMASettings(ctx context.Context) (*models.PMASettings, error) {
	settings := &models.PMASettings{}
	query := `SELECT * FROM pma_settings ORDER BY id DESC LIMIT 1`

	err := r.db.GetContext(ctx, settings, query)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings
			return &models.PMASettings{
				AutoSyncEnabled:     true,
				SyncIntervalMinutes: 15,
				CacheEnabled:        true,
				CacheTTLSeconds:     300,
				MaxRetryAttempts:    3,
				RetryDelayMS:        2000,
				HealthCheckInterval: 60,
				MetricsEnabled:      true,
				UpdatedAt:           time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get PMA settings: %w", err)
	}

	return settings, nil
}

func (r *PMARepository) UpdatePMASettings(ctx context.Context, settings *models.PMASettings) error {
	query := `
		INSERT INTO pma_settings (
			auto_sync_enabled, sync_interval_minutes, cache_enabled, cache_ttl_seconds,
			max_retry_attempts, retry_delay_ms, health_check_interval, metrics_enabled, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			auto_sync_enabled = excluded.auto_sync_enabled,
			sync_interval_minutes = excluded.sync_interval_minutes,
			cache_enabled = excluded.cache_enabled,
			cache_ttl_seconds = excluded.cache_ttl_seconds,
			max_retry_attempts = excluded.max_retry_attempts,
			retry_delay_ms = excluded.retry_delay_ms,
			health_check_interval = excluded.health_check_interval,
			metrics_enabled = excluded.metrics_enabled,
			updated_at = excluded.updated_at`

	settings.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		settings.AutoSyncEnabled, settings.SyncIntervalMinutes, settings.CacheEnabled,
		settings.CacheTTLSeconds, settings.MaxRetryAttempts, settings.RetryDelayMS,
		settings.HealthCheckInterval, settings.MetricsEnabled, settings.UpdatedAt)

	return err
}

// Cleanup operations
func (r *PMARepository) CleanupOldSyncData(ctx context.Context, olderThan time.Time) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clean up old source mappings for deleted entities
	_, err = tx.ExecContext(ctx, `
		DELETE FROM entity_source_mappings 
		WHERE entity_id NOT IN (SELECT entity_id FROM entities)`)
	if err != nil {
		return fmt.Errorf("failed to cleanup source mappings: %w", err)
	}

	// Clean up old unified entity data for deleted entities
	_, err = tx.ExecContext(ctx, `
		DELETE FROM unified_entities 
		WHERE entity_id NOT IN (SELECT entity_id FROM entities)`)
	if err != nil {
		return fmt.Errorf("failed to cleanup unified entities: %w", err)
	}

	// Clean up old unified room data for deleted rooms
	_, err = tx.ExecContext(ctx, `
		DELETE FROM unified_rooms 
		WHERE room_id NOT IN (SELECT id FROM rooms)`)
	if err != nil {
		return fmt.Errorf("failed to cleanup unified rooms: %w", err)
	}

	return tx.Commit()
}
