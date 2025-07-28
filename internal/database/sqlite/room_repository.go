package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// RoomRepository implements repositories.RoomRepository
type RoomRepository struct {
	db *sql.DB
}

// NewRoomRepository creates a new RoomRepository
func NewRoomRepository(db *sql.DB) repositories.RoomRepository {
	return &RoomRepository{db: db}
}

// Create creates a new room
func (r *RoomRepository) Create(ctx context.Context, room *models.Room) error {
	query := `
		INSERT INTO rooms (name, area_id, home_assistant_area_id, icon, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		room.Name,
		room.AreaID,
		room.HomeAssistantAreaID,
		room.Icon,
		room.Description,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted room ID: %w", err)
	}

	room.ID = int(id)
	room.CreatedAt = now
	room.UpdatedAt = now

	return nil
}

// GetByID retrieves a room by ID
func (r *RoomRepository) GetByID(ctx context.Context, id int) (*models.Room, error) {
	query := `
		SELECT id, name, area_id, home_assistant_area_id, icon, description, created_at, updated_at
		FROM rooms
		WHERE id = ?
	`

	room := &models.Room{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&room.ID,
		&room.Name,
		&room.AreaID,
		&room.HomeAssistantAreaID,
		&room.Icon,
		&room.Description,
		&room.CreatedAt,
		&room.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("room not found with ID: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	return room, nil
}

// GetByName retrieves a room by name
func (r *RoomRepository) GetByName(ctx context.Context, name string) (*models.Room, error) {
	query := `
		SELECT id, name, area_id, home_assistant_area_id, icon, description, created_at, updated_at
		FROM rooms
		WHERE name = ?
	`

	room := &models.Room{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&room.ID,
		&room.Name,
		&room.AreaID,
		&room.HomeAssistantAreaID,
		&room.Icon,
		&room.Description,
		&room.CreatedAt,
		&room.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("room not found with name: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	return room, nil
}

// GetAll retrieves all rooms
func (r *RoomRepository) GetAll(ctx context.Context) ([]*models.Room, error) {
	query := `
		SELECT id, name, area_id, home_assistant_area_id, icon, description, created_at, updated_at
		FROM rooms
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room := &models.Room{}
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.AreaID,
			&room.HomeAssistantAreaID,
			&room.Icon,
			&room.Description,
			&room.CreatedAt,
			&room.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// Update updates an existing room
func (r *RoomRepository) Update(ctx context.Context, room *models.Room) error {
	query := `
		UPDATE rooms 
		SET name = ?, area_id = ?, home_assistant_area_id = ?, icon = ?, description = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		room.Name,
		room.AreaID,
		room.HomeAssistantAreaID,
		room.Icon,
		room.Description,
		now,
		room.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("room not found with ID: %d", room.ID)
	}

	room.UpdatedAt = now

	return nil
}

// Delete removes a room
func (r *RoomRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM rooms WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("room not found with ID: %d", id)
	}

	return nil
}

// GetByAreaID retrieves all rooms in a specific area
func (r *RoomRepository) GetByAreaID(ctx context.Context, areaID int) ([]*models.Room, error) {
	query := `
		SELECT id, name, area_id, home_assistant_area_id, icon, description, created_at, updated_at
		FROM rooms
		WHERE area_id = ?
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, areaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms by area: %w", err)
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room := &models.Room{}
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.AreaID,
			&room.HomeAssistantAreaID,
			&room.Icon,
			&room.Description,
			&room.CreatedAt,
			&room.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// GetRoomsWithEntities retrieves rooms with their entities for hierarchical view
func (r *RoomRepository) GetRoomsWithEntities(ctx context.Context, areaID *int) ([]models.RoomWithEntities, error) {
	var query string
	var args []interface{}

	if areaID != nil {
		query = `
			SELECT 
				r.id, r.name, r.area_id, r.home_assistant_area_id, r.icon, r.description,
				COUNT(e.entity_id) as entity_count
			FROM rooms r
			LEFT JOIN entities e ON e.room_id = r.id
			WHERE r.area_id = ?
			GROUP BY r.id, r.name, r.area_id, r.home_assistant_area_id, r.icon, r.description
			ORDER BY r.name
		`
		args = append(args, *areaID)
	} else {
		query = `
			SELECT 
				r.id, r.name, r.area_id, r.home_assistant_area_id, r.icon, r.description,
				COUNT(e.entity_id) as entity_count
			FROM rooms r
			LEFT JOIN entities e ON e.room_id = r.id
			GROUP BY r.id, r.name, r.area_id, r.home_assistant_area_id, r.icon, r.description
			ORDER BY r.name
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms with entities: %w", err)
	}
	defer rows.Close()

	var rooms []models.RoomWithEntities
	for rows.Next() {
		var room models.RoomWithEntities
		var areaID sql.NullInt64
		var haAreaID, icon, description sql.NullString

		err := rows.Scan(
			&room.ID,
			&room.Name,
			&areaID,
			&haAreaID,
			&icon,
			&description,
			&room.EntityCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room with entities: %w", err)
		}

		if areaID.Valid {
			intVal := int(areaID.Int64)
			room.AreaID = &intVal
		}
		if haAreaID.Valid {
			room.HomeAssistantAreaID = &haAreaID.String
		}
		if icon.Valid {
			room.Icon = &icon.String
		}
		if description.Valid {
			room.Description = &description.String
		}

		rooms = append(rooms, room)
	}

	return rooms, nil
}

// AssignToArea assigns a room to an area
func (r *RoomRepository) AssignToArea(ctx context.Context, roomID int, areaID *int) error {
	query := `UPDATE rooms SET area_id = ?, updated_at = ? WHERE id = ?`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, areaID, now, roomID)
	if err != nil {
		return fmt.Errorf("failed to assign room to area: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("room not found with ID: %d", roomID)
	}

	return nil
}

// GetUnassignedRooms retrieves all rooms that are not assigned to any area
func (r *RoomRepository) GetUnassignedRooms(ctx context.Context) ([]*models.Room, error) {
	query := `
		SELECT id, name, area_id, home_assistant_area_id, icon, description, created_at, updated_at
		FROM rooms
		WHERE area_id IS NULL
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unassigned rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room := &models.Room{}
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.AreaID,
			&room.HomeAssistantAreaID,
			&room.Icon,
			&room.Description,
			&room.CreatedAt,
			&room.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}
