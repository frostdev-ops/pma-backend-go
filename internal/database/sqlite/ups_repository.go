package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// UPSRepository implements repositories.UPSRepository
type UPSRepository struct {
	db *sql.DB
}

// NewUPSRepository creates a new UPSRepository
func NewUPSRepository(db *sql.DB) repositories.UPSRepository {
	return &UPSRepository{db: db}
}

// CreateStatus creates a new UPS status record
func (r *UPSRepository) CreateStatus(ctx context.Context, status *models.UPSStatus) error {
	query := `
		INSERT INTO ups_status (
			battery_charge, battery_runtime, input_voltage, output_voltage,
			load, status, temperature, last_updated
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		status.BatteryCharge,
		status.BatteryRuntime,
		status.InputVoltage,
		status.OutputVoltage,
		status.Load,
		status.Status,
		status.Temperature,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create UPS status: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	status.ID = int(id)
	status.LastUpdated = now

	return nil
}

// GetLatestStatus retrieves the most recent UPS status
func (r *UPSRepository) GetLatestStatus(ctx context.Context) (*models.UPSStatus, error) {
	query := `
		SELECT id, battery_charge, battery_runtime, input_voltage, output_voltage,
		       load, status, temperature, last_updated
		FROM ups_status
		ORDER BY last_updated DESC
		LIMIT 1
	`

	status := &models.UPSStatus{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&status.ID,
		&status.BatteryCharge,
		&status.BatteryRuntime,
		&status.InputVoltage,
		&status.OutputVoltage,
		&status.Load,
		&status.Status,
		&status.Temperature,
		&status.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no UPS status records found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest UPS status: %w", err)
	}

	return status, nil
}

// GetStatusHistory retrieves UPS status history with a limit
func (r *UPSRepository) GetStatusHistory(ctx context.Context, limit int) ([]*models.UPSStatus, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT id, battery_charge, battery_runtime, input_voltage, output_voltage,
		       load, status, temperature, last_updated
		FROM ups_status
		ORDER BY last_updated DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query UPS status history: %w", err)
	}
	defer rows.Close()

	var statusHistory []*models.UPSStatus
	for rows.Next() {
		status := &models.UPSStatus{}
		err := rows.Scan(
			&status.ID,
			&status.BatteryCharge,
			&status.BatteryRuntime,
			&status.InputVoltage,
			&status.OutputVoltage,
			&status.Load,
			&status.Status,
			&status.Temperature,
			&status.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan UPS status: %w", err)
		}
		statusHistory = append(statusHistory, status)
	}

	return statusHistory, nil
}

// CleanupOldStatus removes UPS status records older than the specified number of days
func (r *UPSRepository) CleanupOldStatus(ctx context.Context, keepDays int) error {
	if keepDays <= 0 {
		keepDays = 30 // Default to keep 30 days
	}

	query := `
		DELETE FROM ups_status
		WHERE last_updated < datetime('now', '-' || ? || ' days')
	`

	result, err := r.db.ExecContext(ctx, query, keepDays)
	if err != nil {
		return fmt.Errorf("failed to cleanup old UPS status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the cleanup operation (this would typically be done by the service layer)
	if rowsAffected > 0 {
		// Successfully cleaned up records
	}

	return nil
}

// GetStatusByTimeRange retrieves UPS status records within a time range
func (r *UPSRepository) GetStatusByTimeRange(ctx context.Context, start, end time.Time) ([]*models.UPSStatus, error) {
	query := `
		SELECT id, battery_charge, battery_runtime, input_voltage, output_voltage,
		       load, status, temperature, last_updated
		FROM ups_status
		WHERE last_updated BETWEEN ? AND ?
		ORDER BY last_updated DESC
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query UPS status by time range: %w", err)
	}
	defer rows.Close()

	var statusHistory []*models.UPSStatus
	for rows.Next() {
		status := &models.UPSStatus{}
		err := rows.Scan(
			&status.ID,
			&status.BatteryCharge,
			&status.BatteryRuntime,
			&status.InputVoltage,
			&status.OutputVoltage,
			&status.Load,
			&status.Status,
			&status.Temperature,
			&status.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan UPS status: %w", err)
		}
		statusHistory = append(statusHistory, status)
	}

	return statusHistory, nil
}

// GetBatteryTrends retrieves battery charge trends over time for analytics
func (r *UPSRepository) GetBatteryTrends(ctx context.Context, hours int) ([]*models.UPSStatus, error) {
	if hours <= 0 {
		hours = 24 // Default to 24 hours
	}

	query := `
		SELECT id, battery_charge, battery_runtime, input_voltage, output_voltage,
		       load, status, temperature, last_updated
		FROM ups_status
		WHERE last_updated >= datetime('now', '-' || ? || ' hours')
		ORDER BY last_updated ASC
	`

	rows, err := r.db.QueryContext(ctx, query, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to query battery trends: %w", err)
	}
	defer rows.Close()

	var trends []*models.UPSStatus
	for rows.Next() {
		status := &models.UPSStatus{}
		err := rows.Scan(
			&status.ID,
			&status.BatteryCharge,
			&status.BatteryRuntime,
			&status.InputVoltage,
			&status.OutputVoltage,
			&status.Load,
			&status.Status,
			&status.Temperature,
			&status.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan UPS status: %w", err)
		}
		trends = append(trends, status)
	}

	return trends, nil
}
