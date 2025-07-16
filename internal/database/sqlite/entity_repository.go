package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
