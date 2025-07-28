package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// EntityRepository implements repositories.EntityRepository
type EntityRepository struct {
	db *sql.DB
}

// NewEntityRepository creates a new EntityRepository
func NewEntityRepository(db *sql.DB) repositories.EntityRepository {
	return &EntityRepository{db: db}
}

// Create creates a new entity
func (r *EntityRepository) Create(ctx context.Context, entity *models.Entity) error {
	query := `
		INSERT INTO entities (entity_id, friendly_name, domain, state, attributes, last_updated, room_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		entity.EntityID,
		entity.FriendlyName,
		entity.Domain,
		entity.State,
		entity.Attributes,
		now,
		entity.RoomID,
	)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	entity.LastUpdated = now

	return nil
}

// GetByID retrieves an entity by entity ID
func (r *EntityRepository) GetByID(ctx context.Context, entityID string) (*models.Entity, error) {
	query := `
		SELECT entity_id, friendly_name, domain, state, attributes, last_updated, room_id
		FROM entities
		WHERE entity_id = ?
	`

	entity := &models.Entity{}
	err := r.db.QueryRowContext(ctx, query, entityID).Scan(
		&entity.EntityID,
		&entity.FriendlyName,
		&entity.Domain,
		&entity.State,
		&entity.Attributes,
		&entity.LastUpdated,
		&entity.RoomID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entity not found with ID: %s", entityID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	return entity, nil
}

// GetAll retrieves all entities
func (r *EntityRepository) GetAll(ctx context.Context) ([]*models.Entity, error) {
	query := `
		SELECT entity_id, friendly_name, domain, state, attributes, last_updated, room_id
		FROM entities
		ORDER BY entity_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer rows.Close()

	var entities []*models.Entity
	for rows.Next() {
		entity := &models.Entity{}
		err := rows.Scan(
			&entity.EntityID,
			&entity.FriendlyName,
			&entity.Domain,
			&entity.State,
			&entity.Attributes,
			&entity.LastUpdated,
			&entity.RoomID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetByRoom retrieves all entities in a specific room
func (r *EntityRepository) GetByRoom(ctx context.Context, roomID int) ([]*models.Entity, error) {
	query := `
		SELECT entity_id, friendly_name, domain, state, attributes, last_updated, room_id
		FROM entities
		WHERE room_id = ?
		ORDER BY entity_id
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities by room: %w", err)
	}
	defer rows.Close()

	var entities []*models.Entity
	for rows.Next() {
		entity := &models.Entity{}
		err := rows.Scan(
			&entity.EntityID,
			&entity.FriendlyName,
			&entity.Domain,
			&entity.State,
			&entity.Attributes,
			&entity.LastUpdated,
			&entity.RoomID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// Update updates an existing entity
func (r *EntityRepository) Update(ctx context.Context, entity *models.Entity) error {
	query := `
		UPDATE entities 
		SET friendly_name = ?, domain = ?, state = ?, attributes = ?, last_updated = ?, room_id = ?
		WHERE entity_id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		entity.FriendlyName,
		entity.Domain,
		entity.State,
		entity.Attributes,
		now,
		entity.RoomID,
		entity.EntityID,
	)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity not found with ID: %s", entity.EntityID)
	}

	entity.LastUpdated = now

	return nil
}

// Delete removes an entity
func (r *EntityRepository) Delete(ctx context.Context, entityID string) error {
	query := `DELETE FROM entities WHERE entity_id = ?`

	result, err := r.db.ExecContext(ctx, query, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity not found with ID: %s", entityID)
	}

	return nil
}

// PMA-specific methods

// CreateOrUpdatePMAEntity stores a PMA entity with metadata
func (r *EntityRepository) CreateOrUpdatePMAEntity(entity types.PMAEntity) error {
	// Convert PMA entity to database model
	metadata, err := json.Marshal(entity.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	attributes, err := json.Marshal(entity.GetAttributes())
	if err != nil {
		return fmt.Errorf("failed to marshal attributes: %w", err)
	}

	capabilities, err := json.Marshal(entity.GetCapabilities())
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	// Extract room ID as integer if possible
	var roomID sql.NullInt64
	if entity.GetRoomID() != nil {
		// Try to parse as integer
		if id, err := strconv.ParseInt(*entity.GetRoomID(), 10, 64); err == nil {
			roomID = sql.NullInt64{Int64: id, Valid: true}
		}
	}

	// Insert/update main entity record
	query := `
		INSERT INTO entities (entity_id, friendly_name, domain, state, attributes, last_updated, room_id, pma_capabilities, available)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id) DO UPDATE SET
			friendly_name = excluded.friendly_name,
			domain = excluded.domain,
			state = excluded.state,
			attributes = excluded.attributes,
			last_updated = excluded.last_updated,
			room_id = excluded.room_id,
			pma_capabilities = excluded.pma_capabilities,
			available = excluded.available
	`

	_, err = r.db.Exec(
		query,
		entity.GetID(),
		entity.GetFriendlyName(),
		string(entity.GetType()),
		string(entity.GetState()),
		attributes,
		entity.GetLastUpdated(),
		roomID,
		capabilities,
		entity.IsAvailable(),
	)

	if err != nil {
		return fmt.Errorf("failed to create/update PMA entity: %w", err)
	}

	// Store metadata separately in metadata table
	metaQuery := `
		INSERT INTO entity_metadata (entity_id, source, source_entity_id, metadata, quality_score, last_synced, is_virtual, virtual_sources)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id) DO UPDATE SET
			source = excluded.source,
			source_entity_id = excluded.source_entity_id,
			metadata = excluded.metadata,
			quality_score = excluded.quality_score,
			last_synced = excluded.last_synced,
			is_virtual = excluded.is_virtual,
			virtual_sources = excluded.virtual_sources
	`

	meta := entity.GetMetadata()
	if meta != nil {
		var virtualSources string
		if meta.VirtualSources != nil {
			virtualSourcesBytes, _ := json.Marshal(meta.VirtualSources)
			virtualSources = string(virtualSourcesBytes)
		}

		_, err = r.db.Exec(
			metaQuery,
			entity.GetID(),
			string(meta.Source),
			meta.SourceEntityID,
			metadata,
			meta.QualityScore,
			meta.LastSynced,
			meta.IsVirtual,
			virtualSources,
		)
		if err != nil {
			return fmt.Errorf("failed to store entity metadata: %w", err)
		}
	}

	return nil
}

// GetPMAEntity retrieves a PMA entity with metadata
func (r *EntityRepository) GetPMAEntity(entityID string) (types.PMAEntity, error) {
	// Query entity with metadata
	query := `
		SELECT 
			e.entity_id, e.friendly_name, e.domain, e.state, e.attributes, e.last_updated, e.room_id,
			COALESCE(e.pma_capabilities, '[]') as pma_capabilities, COALESCE(e.available, true) as available,
			em.source, em.source_entity_id, em.metadata, em.quality_score, em.last_synced,
			em.is_virtual, em.virtual_sources
		FROM entities e
		LEFT JOIN entity_metadata em ON e.entity_id = em.entity_id
		WHERE e.entity_id = ?
	`

	var entity models.Entity
	var capabilitiesJSON string
	var available bool
	var metadata sql.NullString
	var source sql.NullString
	var sourceEntityID sql.NullString
	var qualityScore sql.NullFloat64
	var lastSynced sql.NullTime
	var isVirtual sql.NullBool
	var virtualSources sql.NullString

	err := r.db.QueryRow(query, entityID).Scan(
		&entity.EntityID,
		&entity.FriendlyName,
		&entity.Domain,
		&entity.State,
		&entity.Attributes,
		&entity.LastUpdated,
		&entity.RoomID,
		&capabilitiesJSON,
		&available,
		&source,
		&sourceEntityID,
		&metadata,
		&qualityScore,
		&lastSynced,
		&isVirtual,
		&virtualSources,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get PMA entity: %w", err)
	}

	// Convert to PMA entity
	return r.convertToPMAEntity(entity, capabilitiesJSON, available, metadata, source, sourceEntityID, qualityScore, lastSynced, isVirtual, virtualSources)
}

// GetPMAEntitiesBySource retrieves all PMA entities from a specific source
func (r *EntityRepository) GetPMAEntitiesBySource(source types.PMASourceType) ([]types.PMAEntity, error) {
	query := `
		SELECT 
			e.entity_id, e.friendly_name, e.domain, e.state, e.attributes, e.last_updated, e.room_id,
			COALESCE(e.pma_capabilities, '[]') as pma_capabilities, COALESCE(e.available, true) as available,
			em.source, em.source_entity_id, em.metadata, em.quality_score, em.last_synced,
			em.is_virtual, em.virtual_sources
		FROM entities e
		LEFT JOIN entity_metadata em ON e.entity_id = em.entity_id
		WHERE em.source = ?
		ORDER BY e.entity_id
	`

	rows, err := r.db.Query(query, string(source))
	if err != nil {
		return nil, fmt.Errorf("failed to query PMA entities by source: %w", err)
	}
	defer rows.Close()

	var entities []types.PMAEntity
	for rows.Next() {
		var entity models.Entity
		var capabilitiesJSON string
		var available bool
		var metadata sql.NullString
		var entitySource sql.NullString
		var sourceEntityID sql.NullString
		var qualityScore sql.NullFloat64
		var lastSynced sql.NullTime
		var isVirtual sql.NullBool
		var virtualSources sql.NullString

		err := rows.Scan(
			&entity.EntityID,
			&entity.FriendlyName,
			&entity.Domain,
			&entity.State,
			&entity.Attributes,
			&entity.LastUpdated,
			&entity.RoomID,
			&capabilitiesJSON,
			&available,
			&entitySource,
			&sourceEntityID,
			&metadata,
			&qualityScore,
			&lastSynced,
			&isVirtual,
			&virtualSources,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PMA entity: %w", err)
		}

		pmaEntity, err := r.convertToPMAEntity(entity, capabilitiesJSON, available, metadata, entitySource, sourceEntityID, qualityScore, lastSynced, isVirtual, virtualSources)
		if err != nil {
			return nil, err
		}

		entities = append(entities, pmaEntity)
	}

	return entities, rows.Err()
}

// DeletePMAEntity removes a PMA entity and its metadata
func (r *EntityRepository) DeletePMAEntity(entityID string) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete metadata first (due to foreign key constraint)
	_, err = tx.Exec("DELETE FROM entity_metadata WHERE entity_id = ?", entityID)
	if err != nil {
		return fmt.Errorf("failed to delete entity metadata: %w", err)
	}

	// Delete entity
	result, err := tx.Exec("DELETE FROM entities WHERE entity_id = ?", entityID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity not found with ID: %s", entityID)
	}

	return tx.Commit()
}

// UpdatePMAEntityMetadata updates only the metadata for a PMA entity
func (r *EntityRepository) UpdatePMAEntityMetadata(entityID string, metadata *types.PMAMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var virtualSources string
	if metadata.VirtualSources != nil {
		virtualSourcesBytes, _ := json.Marshal(metadata.VirtualSources)
		virtualSources = string(virtualSourcesBytes)
	}

	query := `
		UPDATE entity_metadata 
		SET source = ?, source_entity_id = ?, metadata = ?, quality_score = ?, 
		    last_synced = ?, is_virtual = ?, virtual_sources = ?
		WHERE entity_id = ?
	`

	result, err := r.db.Exec(
		query,
		string(metadata.Source),
		metadata.SourceEntityID,
		metadataJSON,
		metadata.QualityScore,
		metadata.LastSynced,
		metadata.IsVirtual,
		virtualSources,
		entityID,
	)
	if err != nil {
		return fmt.Errorf("failed to update entity metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("metadata not found for entity ID: %s", entityID)
	}

	return nil
}

// convertToPMAEntity converts database entity to PMA entity
func (r *EntityRepository) convertToPMAEntity(
	entity models.Entity,
	capabilitiesJSON string,
	available bool,
	metadata sql.NullString,
	source sql.NullString,
	sourceEntityID sql.NullString,
	qualityScore sql.NullFloat64,
	lastSynced sql.NullTime,
	isVirtual sql.NullBool,
	virtualSources sql.NullString,
) (types.PMAEntity, error) {
	// Parse capabilities
	var capabilities []types.PMACapability
	if capabilitiesJSON != "" {
		if err := json.Unmarshal([]byte(capabilitiesJSON), &capabilities); err != nil {
			capabilities = []types.PMACapability{} // Default to empty if parsing fails
		}
	}

	// Parse attributes
	var attributes map[string]interface{}
	if len(entity.Attributes) > 0 {
		if err := json.Unmarshal(entity.Attributes, &attributes); err != nil {
			attributes = make(map[string]interface{})
		}
	} else {
		attributes = make(map[string]interface{})
	}

	// Build metadata
	var pmaMetadata *types.PMAMetadata
	if source.Valid && sourceEntityID.Valid {
		pmaMetadata = &types.PMAMetadata{
			Source:         types.PMASourceType(source.String),
			SourceEntityID: sourceEntityID.String,
			LastSynced:     time.Now(),
			QualityScore:   1.0,
		}

		if qualityScore.Valid {
			pmaMetadata.QualityScore = qualityScore.Float64
		}
		if lastSynced.Valid {
			pmaMetadata.LastSynced = lastSynced.Time
		}
		if isVirtual.Valid {
			pmaMetadata.IsVirtual = isVirtual.Bool
		}
		if virtualSources.Valid && virtualSources.String != "" {
			var sources []types.PMASourceType
			if err := json.Unmarshal([]byte(virtualSources.String), &sources); err == nil {
				pmaMetadata.VirtualSources = sources
			}
		}

		// Parse source data from metadata field
		if metadata.Valid && metadata.String != "" {
			var sourceData map[string]interface{}
			if err := json.Unmarshal([]byte(metadata.String), &sourceData); err == nil {
				pmaMetadata.SourceData = sourceData
			}
		}
	}

	// Convert room ID
	var roomID *string
	if entity.RoomID.Valid {
		roomIDStr := strconv.FormatInt(entity.RoomID.Int64, 10)
		roomID = &roomIDStr
	}

	// Create base PMA entity
	baseEntity := &types.PMABaseEntity{
		ID:           entity.EntityID,
		Type:         types.PMAEntityType(entity.Domain),
		FriendlyName: entity.FriendlyName.String,
		State:        types.PMAEntityState(entity.State.String),
		Attributes:   attributes,
		LastUpdated:  entity.LastUpdated,
		Capabilities: capabilities,
		RoomID:       roomID,
		Metadata:     pmaMetadata,
		Available:    available,
	}

	return baseEntity, nil
}
