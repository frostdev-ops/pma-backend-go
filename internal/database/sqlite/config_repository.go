package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// ConfigRepository implements repositories.ConfigRepository
type ConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository creates a new ConfigRepository
func NewConfigRepository(db *sql.DB) repositories.ConfigRepository {
	return &ConfigRepository{db: db}
}

// Get retrieves a configuration value by key
func (r *ConfigRepository) Get(ctx context.Context, key string) (*models.SystemConfig, error) {
	query := `
		SELECT key, value, encrypted, description, updated_at
		FROM system_config
		WHERE key = ?
	`

	config := &models.SystemConfig{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&config.Key,
		&config.Value,
		&config.Encrypted,
		&config.Description,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("config key not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return config, nil
}

// Set creates or updates a configuration value
func (r *ConfigRepository) Set(ctx context.Context, config *models.SystemConfig) error {
	query := `
		INSERT INTO system_config (key, value, encrypted, description, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			encrypted = excluded.encrypted,
			description = excluded.description,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		config.Key,
		config.Value,
		config.Encrypted,
		config.Description,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// GetAll retrieves all configuration values
func (r *ConfigRepository) GetAll(ctx context.Context) ([]*models.SystemConfig, error) {
	query := `
		SELECT key, value, encrypted, description, updated_at
		FROM system_config
		ORDER BY key
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	var configs []*models.SystemConfig
	for rows.Next() {
		config := &models.SystemConfig{}
		err := rows.Scan(
			&config.Key,
			&config.Value,
			&config.Encrypted,
			&config.Description,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// Delete removes a configuration value
func (r *ConfigRepository) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM system_config WHERE key = ?`

	result, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("config key not found: %s", key)
	}

	return nil
}
