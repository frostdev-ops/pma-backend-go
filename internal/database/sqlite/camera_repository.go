package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// CameraRepository implements repositories.CameraRepository
type CameraRepository struct {
	db *sql.DB
}

// NewCameraRepository creates a new CameraRepository
func NewCameraRepository(db *sql.DB) repositories.CameraRepository {
	return &CameraRepository{db: db}
}

// Create creates a new camera
func (r *CameraRepository) Create(ctx context.Context, camera *models.Camera) error {
	query := `
		INSERT INTO cameras (entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	// Convert capabilities and settings to JSON
	capabilitiesJSON, err := json.Marshal(camera.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	settingsJSON, err := json.Marshal(camera.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		camera.EntityID,
		camera.Name,
		camera.Type,
		nullStringFromSQLNullString(camera.StreamURL),
		nullStringFromSQLNullString(camera.SnapshotURL),
		capabilitiesJSON,
		settingsJSON,
		camera.IsEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to create camera: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	camera.ID = int(id)
	camera.CreatedAt = time.Now()
	camera.UpdatedAt = time.Now()

	return nil
}

// GetByID retrieves a camera by ID
func (r *CameraRepository) GetByID(ctx context.Context, id int) (*models.Camera, error) {
	query := `
		SELECT id, entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at
		FROM cameras
		WHERE id = ?
	`

	var camera models.Camera
	var capabilitiesJSON, settingsJSON []byte
	var streamURL, snapshotURL sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&camera.ID,
		&camera.EntityID,
		&camera.Name,
		&camera.Type,
		&streamURL,
		&snapshotURL,
		&capabilitiesJSON,
		&settingsJSON,
		&camera.IsEnabled,
		&camera.CreatedAt,
		&camera.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("camera with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get camera by ID: %w", err)
	}

	// Convert NULL strings
	camera.StreamURL = sqlNullStringFromNullString(streamURL)
	camera.SnapshotURL = sqlNullStringFromNullString(snapshotURL)

	// Unmarshal JSON fields
	if len(capabilitiesJSON) > 0 {
		if err := json.Unmarshal(capabilitiesJSON, &camera.Capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &camera.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &camera, nil
}

// GetByEntityID retrieves a camera by entity ID
func (r *CameraRepository) GetByEntityID(ctx context.Context, entityID string) (*models.Camera, error) {
	query := `
		SELECT id, entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at
		FROM cameras
		WHERE entity_id = ?
	`

	var camera models.Camera
	var capabilitiesJSON, settingsJSON []byte
	var streamURL, snapshotURL sql.NullString

	err := r.db.QueryRowContext(ctx, query, entityID).Scan(
		&camera.ID,
		&camera.EntityID,
		&camera.Name,
		&camera.Type,
		&streamURL,
		&snapshotURL,
		&capabilitiesJSON,
		&settingsJSON,
		&camera.IsEnabled,
		&camera.CreatedAt,
		&camera.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("camera with entity ID %s not found", entityID)
		}
		return nil, fmt.Errorf("failed to get camera by entity ID: %w", err)
	}

	// Convert NULL strings
	camera.StreamURL = sqlNullStringFromNullString(streamURL)
	camera.SnapshotURL = sqlNullStringFromNullString(snapshotURL)

	// Unmarshal JSON fields
	if len(capabilitiesJSON) > 0 {
		if err := json.Unmarshal(capabilitiesJSON, &camera.Capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &camera.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &camera, nil
}

// GetAll retrieves all cameras
func (r *CameraRepository) GetAll(ctx context.Context) ([]*models.Camera, error) {
	query := `
		SELECT id, entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at
		FROM cameras
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cameras: %w", err)
	}
	defer rows.Close()

	var cameras []*models.Camera

	for rows.Next() {
		var camera models.Camera
		var capabilitiesJSON, settingsJSON []byte
		var streamURL, snapshotURL sql.NullString

		err := rows.Scan(
			&camera.ID,
			&camera.EntityID,
			&camera.Name,
			&camera.Type,
			&streamURL,
			&snapshotURL,
			&capabilitiesJSON,
			&settingsJSON,
			&camera.IsEnabled,
			&camera.CreatedAt,
			&camera.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan camera row: %w", err)
		}

		// Convert NULL strings
		camera.StreamURL = sqlNullStringFromNullString(streamURL)
		camera.SnapshotURL = sqlNullStringFromNullString(snapshotURL)

		// Unmarshal JSON fields
		if len(capabilitiesJSON) > 0 {
			if err := json.Unmarshal(capabilitiesJSON, &camera.Capabilities); err != nil {
				return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
			}
		}

		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &camera.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}

		cameras = append(cameras, &camera)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate camera rows: %w", err)
	}

	return cameras, nil
}

// GetEnabled retrieves all enabled cameras
func (r *CameraRepository) GetEnabled(ctx context.Context) ([]*models.Camera, error) {
	query := `
		SELECT id, entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at
		FROM cameras
		WHERE is_enabled = TRUE
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled cameras: %w", err)
	}
	defer rows.Close()

	var cameras []*models.Camera

	for rows.Next() {
		var camera models.Camera
		var capabilitiesJSON, settingsJSON []byte
		var streamURL, snapshotURL sql.NullString

		err := rows.Scan(
			&camera.ID,
			&camera.EntityID,
			&camera.Name,
			&camera.Type,
			&streamURL,
			&snapshotURL,
			&capabilitiesJSON,
			&settingsJSON,
			&camera.IsEnabled,
			&camera.CreatedAt,
			&camera.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan camera row: %w", err)
		}

		// Convert NULL strings
		camera.StreamURL = sqlNullStringFromNullString(streamURL)
		camera.SnapshotURL = sqlNullStringFromNullString(snapshotURL)

		// Unmarshal JSON fields
		if len(capabilitiesJSON) > 0 {
			if err := json.Unmarshal(capabilitiesJSON, &camera.Capabilities); err != nil {
				return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
			}
		}

		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &camera.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}

		cameras = append(cameras, &camera)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate camera rows: %w", err)
	}

	return cameras, nil
}

// Update updates an existing camera
func (r *CameraRepository) Update(ctx context.Context, camera *models.Camera) error {
	query := `
		UPDATE cameras
		SET name = ?, type = ?, stream_url = ?, snapshot_url = ?, capabilities = ?, settings = ?, is_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	// Convert capabilities and settings to JSON
	capabilitiesJSON, err := json.Marshal(camera.Capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	settingsJSON, err := json.Marshal(camera.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		camera.Name,
		camera.Type,
		nullStringFromSQLNullString(camera.StreamURL),
		nullStringFromSQLNullString(camera.SnapshotURL),
		capabilitiesJSON,
		settingsJSON,
		camera.IsEnabled,
		camera.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update camera: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("camera with ID %d not found", camera.ID)
	}

	camera.UpdatedAt = time.Now()

	return nil
}

// Delete deletes a camera by ID
func (r *CameraRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM cameras WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete camera: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("camera with ID %d not found", id)
	}

	return nil
}

// GetByType retrieves cameras by type (e.g., "ring", "generic")
func (r *CameraRepository) GetByType(ctx context.Context, cameraType string) ([]*models.Camera, error) {
	query := `
		SELECT id, entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at
		FROM cameras
		WHERE type = ?
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, cameraType)
	if err != nil {
		return nil, fmt.Errorf("failed to query cameras by type: %w", err)
	}
	defer rows.Close()

	var cameras []*models.Camera

	for rows.Next() {
		var camera models.Camera
		var capabilitiesJSON, settingsJSON []byte
		var streamURL, snapshotURL sql.NullString

		err := rows.Scan(
			&camera.ID,
			&camera.EntityID,
			&camera.Name,
			&camera.Type,
			&streamURL,
			&snapshotURL,
			&capabilitiesJSON,
			&settingsJSON,
			&camera.IsEnabled,
			&camera.CreatedAt,
			&camera.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan camera row: %w", err)
		}

		// Convert NULL strings
		camera.StreamURL = sqlNullStringFromNullString(streamURL)
		camera.SnapshotURL = sqlNullStringFromNullString(snapshotURL)

		// Unmarshal JSON fields
		if len(capabilitiesJSON) > 0 {
			if err := json.Unmarshal(capabilitiesJSON, &camera.Capabilities); err != nil {
				return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
			}
		}

		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &camera.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}

		cameras = append(cameras, &camera)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate camera rows: %w", err)
	}

	return cameras, nil
}

// UpdateStatus updates camera enabled status
func (r *CameraRepository) UpdateStatus(ctx context.Context, id int, enabled bool) error {
	query := `
		UPDATE cameras
		SET is_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, enabled, id)
	if err != nil {
		return fmt.Errorf("failed to update camera status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("camera with ID %d not found", id)
	}

	return nil
}

// UpdateStreamURL updates camera stream URL
func (r *CameraRepository) UpdateStreamURL(ctx context.Context, id int, streamURL string) error {
	query := `
		UPDATE cameras
		SET stream_url = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, streamURL, id)
	if err != nil {
		return fmt.Errorf("failed to update camera stream URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("camera with ID %d not found", id)
	}

	return nil
}

// UpdateSnapshotURL updates camera snapshot URL
func (r *CameraRepository) UpdateSnapshotURL(ctx context.Context, id int, snapshotURL string) error {
	query := `
		UPDATE cameras
		SET snapshot_url = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, snapshotURL, id)
	if err != nil {
		return fmt.Errorf("failed to update camera snapshot URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("camera with ID %d not found", id)
	}

	return nil
}

// SearchCameras searches cameras by name or entity ID
func (r *CameraRepository) SearchCameras(ctx context.Context, query string) ([]*models.Camera, error) {
	searchQuery := `
		SELECT id, entity_id, name, type, stream_url, snapshot_url, capabilities, settings, is_enabled, created_at, updated_at
		FROM cameras
		WHERE name LIKE ? OR entity_id LIKE ?
		ORDER BY name ASC
	`

	searchPattern := "%" + strings.ToLower(query) + "%"

	rows, err := r.db.QueryContext(ctx, searchQuery, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search cameras: %w", err)
	}
	defer rows.Close()

	var cameras []*models.Camera

	for rows.Next() {
		var camera models.Camera
		var capabilitiesJSON, settingsJSON []byte
		var streamURL, snapshotURL sql.NullString

		err := rows.Scan(
			&camera.ID,
			&camera.EntityID,
			&camera.Name,
			&camera.Type,
			&streamURL,
			&snapshotURL,
			&capabilitiesJSON,
			&settingsJSON,
			&camera.IsEnabled,
			&camera.CreatedAt,
			&camera.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan camera row: %w", err)
		}

		// Convert NULL strings
		camera.StreamURL = sqlNullStringFromNullString(streamURL)
		camera.SnapshotURL = sqlNullStringFromNullString(snapshotURL)

		// Unmarshal JSON fields
		if len(capabilitiesJSON) > 0 {
			if err := json.Unmarshal(capabilitiesJSON, &camera.Capabilities); err != nil {
				return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
			}
		}

		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &camera.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}

		cameras = append(cameras, &camera)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate camera rows: %w", err)
	}

	return cameras, nil
}

// CountCameras returns the total number of cameras
func (r *CameraRepository) CountCameras(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM cameras`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count cameras: %w", err)
	}

	return count, nil
}

// CountEnabledCameras returns the number of enabled cameras
func (r *CameraRepository) CountEnabledCameras(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM cameras WHERE is_enabled = TRUE`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count enabled cameras: %w", err)
	}

	return count, nil
}

// Helper functions for handling sql.NullString
func nullStringFromSQLNullString(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func sqlNullStringFromNullString(ns sql.NullString) sql.NullString {
	return ns
}
