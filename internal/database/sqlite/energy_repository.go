package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/energy"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// EnergyRepository implements repositories.EnergyRepository
type EnergyRepository struct {
	db *sql.DB
}

// NewEnergyRepository creates a new EnergyRepository
func NewEnergyRepository(db *sql.DB) repositories.EnergyRepository {
	return &EnergyRepository{db: db}
}

// GetSettings retrieves energy settings
func (r *EnergyRepository) GetSettings(ctx context.Context) (*energy.EnergySettings, error) {
	query := `
		SELECT id, energy_rate, currency, tracking_enabled, update_interval, historical_period, updated_at
		FROM energy_settings
		WHERE id = 1
	`

	settings := &energy.EnergySettings{}
	var trackingEnabled int

	err := r.db.QueryRowContext(ctx, query).Scan(
		&settings.ID,
		&settings.EnergyRate,
		&settings.Currency,
		&trackingEnabled,
		&settings.UpdateInterval,
		&settings.HistoricalPeriod,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default settings if none exist
		return &energy.EnergySettings{
			ID:               1,
			EnergyRate:       energy.DefaultEnergyRate,
			Currency:         energy.DefaultCurrency,
			TrackingEnabled:  energy.DefaultTrackingEnabled,
			UpdateInterval:   energy.DefaultUpdateInterval,
			HistoricalPeriod: energy.DefaultHistoricalPeriod,
			UpdatedAt:        time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get energy settings: %w", err)
	}

	settings.TrackingEnabled = trackingEnabled == 1
	return settings, nil
}

// UpdateSettings updates energy settings
func (r *EnergyRepository) UpdateSettings(ctx context.Context, settings *energy.EnergySettings) error {
	query := `
		INSERT OR REPLACE INTO energy_settings (id, energy_rate, currency, tracking_enabled, update_interval, historical_period, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`

	trackingEnabled := 0
	if settings.TrackingEnabled {
		trackingEnabled = 1
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		settings.EnergyRate,
		settings.Currency,
		trackingEnabled,
		settings.UpdateInterval,
		settings.HistoricalPeriod,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update energy settings: %w", err)
	}

	return nil
}

// CreateEnergyHistory creates a new energy history entry
func (r *EnergyRepository) CreateEnergyHistory(ctx context.Context, history *energy.EnergyHistory) error {
	query := `
		INSERT INTO energy_history (timestamp, power_consumption, energy_usage, cost, device_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		history.Timestamp.Format(time.RFC3339),
		history.PowerConsumption,
		history.EnergyUsage,
		history.Cost,
		history.DeviceCount,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create energy history: %w", err)
	}

	return nil
}

// GetEnergyHistory retrieves energy history with filtering
func (r *EnergyRepository) GetEnergyHistory(ctx context.Context, filter *energy.EnergyHistoryFilter) ([]*energy.EnergyHistory, error) {
	query := `
		SELECT id, timestamp, power_consumption, energy_usage, cost, device_count, created_at
		FROM energy_history
		WHERE 1=1
	`
	args := []interface{}{}

	// Add filters
	if filter != nil {
		if filter.StartDate != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartDate.Format(time.RFC3339))
		}
		if filter.EndDate != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndDate.Format(time.RFC3339))
		}
	}

	query += " ORDER BY timestamp DESC"

	// Add limit and offset
	if filter != nil {
		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query energy history: %w", err)
	}
	defer rows.Close()

	var histories []*energy.EnergyHistory
	for rows.Next() {
		history := &energy.EnergyHistory{}
		var timestampStr string

		err := rows.Scan(
			&history.ID,
			&timestampStr,
			&history.PowerConsumption,
			&history.EnergyUsage,
			&history.Cost,
			&history.DeviceCount,
			&history.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan energy history: %w", err)
		}

		history.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		histories = append(histories, history)
	}

	return histories, nil
}

// GetEnergyHistoryCount returns the count of energy history entries matching the filter
func (r *EnergyRepository) GetEnergyHistoryCount(ctx context.Context, filter *energy.EnergyHistoryFilter) (int, error) {
	query := "SELECT COUNT(*) FROM energy_history WHERE 1=1"
	args := []interface{}{}

	if filter != nil {
		if filter.StartDate != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartDate.Format(time.RFC3339))
		}
		if filter.EndDate != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndDate.Format(time.RFC3339))
		}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count energy history: %w", err)
	}

	return count, nil
}

// CleanupOldHistory removes energy history older than specified days
func (r *EnergyRepository) CleanupOldHistory(ctx context.Context, days int) error {
	query := `DELETE FROM energy_history WHERE created_at < datetime('now', '-' || ? || ' days')`

	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return fmt.Errorf("failed to cleanup old energy history: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Log cleanup if needed
	}

	return nil
}

// CreateDeviceEnergy creates a new device energy entry
func (r *EnergyRepository) CreateDeviceEnergy(ctx context.Context, deviceEnergy *energy.DeviceEnergy) error {
	query := `
		INSERT INTO device_energy (entity_id, device_name, room, power_consumption, energy_usage, 
		                          cost, state, is_on, percentage, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	isOn := 0
	if deviceEnergy.IsOn {
		isOn = 1
	}

	now := time.Now()
	_, err := r.db.ExecContext(
		ctx,
		query,
		deviceEnergy.EntityID,
		deviceEnergy.DeviceName,
		deviceEnergy.Room,
		deviceEnergy.PowerConsumption,
		deviceEnergy.EnergyUsage,
		deviceEnergy.Cost,
		deviceEnergy.State,
		isOn,
		deviceEnergy.Percentage,
		deviceEnergy.Timestamp.Format(time.RFC3339),
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create device energy: %w", err)
	}

	return nil
}

// CreateDeviceEnergyBatch creates multiple device energy entries in a transaction
func (r *EnergyRepository) CreateDeviceEnergyBatch(ctx context.Context, deviceEnergies []*energy.DeviceEnergy) error {
	if len(deviceEnergies) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO device_energy (entity_id, device_name, room, power_consumption, energy_usage, 
		                          cost, state, is_on, percentage, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, deviceEnergy := range deviceEnergies {
		isOn := 0
		if deviceEnergy.IsOn {
			isOn = 1
		}

		_, err = stmt.ExecContext(
			ctx,
			deviceEnergy.EntityID,
			deviceEnergy.DeviceName,
			deviceEnergy.Room,
			deviceEnergy.PowerConsumption,
			deviceEnergy.EnergyUsage,
			deviceEnergy.Cost,
			deviceEnergy.State,
			isOn,
			deviceEnergy.Percentage,
			deviceEnergy.Timestamp.Format(time.RFC3339),
			now,
		)

		if err != nil {
			return fmt.Errorf("failed to insert device energy: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDeviceEnergy retrieves device energy with filtering
func (r *EnergyRepository) GetDeviceEnergy(ctx context.Context, filter *energy.DeviceEnergyFilter) ([]*energy.DeviceEnergy, error) {
	query := `
		SELECT id, entity_id, device_name, room, power_consumption, energy_usage, cost, 
		       state, is_on, percentage, timestamp, created_at
		FROM device_energy
		WHERE 1=1
	`
	args := []interface{}{}

	// Add filters
	if filter != nil {
		if filter.EntityID != nil {
			query += " AND entity_id = ?"
			args = append(args, *filter.EntityID)
		}
		if filter.Room != nil {
			query += " AND room = ?"
			args = append(args, *filter.Room)
		}
		if filter.StartDate != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartDate.Format(time.RFC3339))
		}
		if filter.EndDate != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndDate.Format(time.RFC3339))
		}
	}

	query += " ORDER BY timestamp DESC"

	// Add limit and offset
	if filter != nil {
		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query device energy: %w", err)
	}
	defer rows.Close()

	var deviceEnergies []*energy.DeviceEnergy
	for rows.Next() {
		deviceEnergy := &energy.DeviceEnergy{}
		var timestampStr string
		var isOn int

		err := rows.Scan(
			&deviceEnergy.ID,
			&deviceEnergy.EntityID,
			&deviceEnergy.DeviceName,
			&deviceEnergy.Room,
			&deviceEnergy.PowerConsumption,
			&deviceEnergy.EnergyUsage,
			&deviceEnergy.Cost,
			&deviceEnergy.State,
			&isOn,
			&deviceEnergy.Percentage,
			&timestampStr,
			&deviceEnergy.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device energy: %w", err)
		}

		deviceEnergy.IsOn = isOn == 1
		deviceEnergy.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		deviceEnergies = append(deviceEnergies, deviceEnergy)
	}

	return deviceEnergies, nil
}

// GetDeviceEnergyCount returns the count of device energy entries matching the filter
func (r *EnergyRepository) GetDeviceEnergyCount(ctx context.Context, filter *energy.DeviceEnergyFilter) (int, error) {
	query := "SELECT COUNT(*) FROM device_energy WHERE 1=1"
	args := []interface{}{}

	if filter != nil {
		if filter.EntityID != nil {
			query += " AND entity_id = ?"
			args = append(args, *filter.EntityID)
		}
		if filter.Room != nil {
			query += " AND room = ?"
			args = append(args, *filter.Room)
		}
		if filter.StartDate != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartDate.Format(time.RFC3339))
		}
		if filter.EndDate != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndDate.Format(time.RFC3339))
		}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count device energy: %w", err)
	}

	return count, nil
}

// GetDeviceEnergyByEntity retrieves device energy for a specific entity
func (r *EnergyRepository) GetDeviceEnergyByEntity(ctx context.Context, entityID string, startDate, endDate time.Time) ([]*energy.DeviceEnergy, error) {
	filter := &energy.DeviceEnergyFilter{
		EntityID:  &entityID,
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     1000, // Reasonable limit
	}

	return r.GetDeviceEnergy(ctx, filter)
}

// GetTopEnergyConsumers retrieves the top energy consuming devices
func (r *EnergyRepository) GetTopEnergyConsumers(ctx context.Context, limit int, startDate, endDate time.Time) ([]*energy.DeviceEnergy, error) {
	query := `
		SELECT entity_id, device_name, room, 
		       AVG(power_consumption) as avg_power,
		       SUM(energy_usage) as total_energy,
		       SUM(cost) as total_cost,
		       state, is_on, 0 as percentage,
		       MAX(timestamp) as latest_timestamp,
		       MAX(created_at) as latest_created_at
		FROM device_energy
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY entity_id
		ORDER BY total_energy DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top energy consumers: %w", err)
	}
	defer rows.Close()

	var topConsumers []*energy.DeviceEnergy
	for rows.Next() {
		deviceEnergy := &energy.DeviceEnergy{}
		var timestampStr string
		var isOn int

		err := rows.Scan(
			&deviceEnergy.EntityID,
			&deviceEnergy.DeviceName,
			&deviceEnergy.Room,
			&deviceEnergy.PowerConsumption,
			&deviceEnergy.EnergyUsage,
			&deviceEnergy.Cost,
			&deviceEnergy.State,
			&isOn,
			&deviceEnergy.Percentage,
			&timestampStr,
			&deviceEnergy.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top energy consumer: %w", err)
		}

		deviceEnergy.IsOn = isOn == 1
		deviceEnergy.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		topConsumers = append(topConsumers, deviceEnergy)
	}

	return topConsumers, nil
}

// CleanupOldDeviceEnergy removes device energy data older than specified days
func (r *EnergyRepository) CleanupOldDeviceEnergy(ctx context.Context, days int) error {
	query := `DELETE FROM device_energy WHERE created_at < datetime('now', '-' || ? || ' days')`

	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return fmt.Errorf("failed to cleanup old device energy: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Log cleanup if needed
	}

	return nil
}

// GetEnergyStats retrieves energy statistics for the given period
func (r *EnergyRepository) GetEnergyStats(ctx context.Context, startDate, endDate time.Time) (*energy.EnergyStats, error) {
	// Get basic stats
	query := `
		SELECT 
			COALESCE(AVG(power_consumption), 0) as avg_power,
			COALESCE(MAX(power_consumption), 0) as peak_power,
			COALESCE(SUM(energy_usage), 0) as total_energy,
			COALESCE(SUM(cost), 0) as total_cost
		FROM energy_history
		WHERE timestamp >= ? AND timestamp <= ?
	`

	stats := &energy.EnergyStats{}
	err := r.db.QueryRowContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(
		&stats.AveragePower,
		&stats.PeakPower,
		&stats.TotalEnergy,
		&stats.TotalCost,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get energy stats: %w", err)
	}

	// Get current power (latest entry)
	currentQuery := `
		SELECT COALESCE(power_consumption, 0)
		FROM energy_history
		ORDER BY timestamp DESC
		LIMIT 1
	`
	r.db.QueryRowContext(ctx, currentQuery).Scan(&stats.CurrentPower)

	// Get top consumers
	topConsumers, err := r.GetTopEnergyConsumers(ctx, 10, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get top consumers: %w", err)
	}

	// Convert to DeviceEnergyConsumption format
	for _, consumer := range topConsumers {
		stats.TopConsumers = append(stats.TopConsumers, energy.DeviceEnergyConsumption{
			EntityID:         consumer.EntityID,
			DeviceName:       consumer.DeviceName,
			Room:             consumer.Room,
			PowerConsumption: consumer.PowerConsumption,
			EnergyUsage:      consumer.EnergyUsage,
			Cost:             consumer.Cost,
			State:            consumer.State,
			IsOn:             consumer.IsOn,
			Percentage:       consumer.Percentage,
		})
	}

	// Get recent history
	historyFilter := &energy.EnergyHistoryFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     24, // Last 24 entries
	}
	histories, err := r.GetEnergyHistory(ctx, historyFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get energy history: %w", err)
	}

	for _, history := range histories {
		stats.History = append(stats.History, energy.EnergyHistoryEntry{
			Timestamp:        history.Timestamp,
			PowerConsumption: history.PowerConsumption,
			EnergyUsage:      history.EnergyUsage,
			Cost:             history.Cost,
		})
	}

	// Calculate energy savings based on historical data and automation efficiency
	savings, err := r.calculateEnergySavings(ctx, startDate, endDate, stats.History)
	if err != nil {
		// Log error but don't fail the entire operation
		// Set default savings values
		savings = energy.EnergySavings{
			TotalSavings:        0,
			AutomationSavings:   0,
			OptimizationSavings: 0,
			SchedulingSavings:   0,
			PeriodDays:          int(endDate.Sub(startDate).Hours() / 24),
		}
	}
	stats.Savings = savings

	return stats, nil
}

// GetTotalEnergyConsumption returns total energy consumption for the given period
func (r *EnergyRepository) GetTotalEnergyConsumption(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(energy_usage), 0)
		FROM energy_history
		WHERE timestamp >= ? AND timestamp <= ?
	`

	var total float64
	err := r.db.QueryRowContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total energy consumption: %w", err)
	}

	return total, nil
}

// GetTotalEnergyCost returns total energy cost for the given period
func (r *EnergyRepository) GetTotalEnergyCost(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(cost), 0)
		FROM energy_history
		WHERE timestamp >= ? AND timestamp <= ?
	`

	var total float64
	err := r.db.QueryRowContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total energy cost: %w", err)
	}

	return total, nil
}

// GetDeviceEnergyMetrics returns metrics about device energy tracking
func (r *EnergyRepository) GetDeviceEnergyMetrics(ctx context.Context) (*energy.EnergyMetrics, error) {
	// Get basic device counts
	deviceCountQuery := `
		SELECT 
			COUNT(DISTINCT entity_id) as total_devices,
			COUNT(DISTINCT CASE WHEN power_consumption > 0 THEN entity_id END) as active_devices,
			COUNT(DISTINCT CASE WHEN power_consumption > 0 THEN entity_id END) as power_devices
		FROM device_energy
		WHERE created_at >= datetime('now', '-1 day')
	`

	metrics := &energy.EnergyMetrics{}
	err := r.db.QueryRowContext(ctx, deviceCountQuery).Scan(
		&metrics.TotalDevicesTracked,
		&metrics.ActiveDevices,
		&metrics.PowerDevices,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get device metrics: %w", err)
	}

	// Get last update time
	lastUpdateQuery := `
		SELECT MAX(timestamp)
		FROM energy_history
	`
	var lastUpdateStr sql.NullString
	r.db.QueryRowContext(ctx, lastUpdateQuery).Scan(&lastUpdateStr)
	if lastUpdateStr.Valid {
		metrics.LastUpdateTime, _ = time.Parse(time.RFC3339, lastUpdateStr.String)
	}

	// Get settings
	settings, err := r.GetSettings(ctx)
	if err == nil {
		metrics.UpdateInterval = settings.UpdateInterval
		metrics.TrackingEnabled = settings.TrackingEnabled
		metrics.HistoryRetentionDays = settings.HistoricalPeriod
	}

	// Count Shelly devices (devices with entity_id containing "shelly")
	shellyCountQuery := `
		SELECT COUNT(DISTINCT entity_id)
		FROM device_energy
		WHERE entity_id LIKE '%shelly%'
		AND created_at >= datetime('now', '-1 day')
	`
	r.db.QueryRowContext(ctx, shellyCountQuery).Scan(&metrics.ShellyDevices)

	// Note: UPS detection would need to be implemented based on UPS service integration
	metrics.UPSDetected = false

	return metrics, nil
}

// calculateEnergySavings calculates energy savings based on historical data and automation patterns
func (r *EnergyRepository) calculateEnergySavings(ctx context.Context, startDate, endDate time.Time, history []energy.EnergyHistoryEntry) (energy.EnergySavings, error) {
	periodDays := int(endDate.Sub(startDate).Hours() / 24)
	if periodDays <= 0 {
		periodDays = 1
	}

	// Calculate baseline energy consumption (what usage would be without automation)
	totalUsage := 0.0
	for _, entry := range history {
		totalUsage += entry.EnergyUsage
	}

	// Query automation activity to estimate savings
	automationSavings, err := r.calculateAutomationSavings(ctx, startDate, endDate)
	if err != nil {
		automationSavings = 0
	}

	// Query optimization patterns (scheduled operations, load balancing)
	optimizationSavings, err := r.calculateOptimizationSavings(ctx, startDate, endDate)
	if err != nil {
		optimizationSavings = 0
	}

	// Calculate scheduling savings (off-peak usage, timer-based controls)
	schedulingSavings, err := r.calculateSchedulingSavings(ctx, startDate, endDate)
	if err != nil {
		schedulingSavings = 0
	}

	totalSavings := automationSavings + optimizationSavings + schedulingSavings

	return energy.EnergySavings{
		TotalSavings:        totalSavings,
		AutomationSavings:   automationSavings,
		OptimizationSavings: optimizationSavings,
		SchedulingSavings:   schedulingSavings,
		PeriodDays:          periodDays,
	}, nil
}

// calculateAutomationSavings estimates savings from automation rules
func (r *EnergyRepository) calculateAutomationSavings(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	// Query for automation executions that resulted in energy savings
	query := `
		SELECT COUNT(*) as executions
		FROM automation_executions ae
		JOIN automations a ON ae.automation_id = a.id
		WHERE ae.executed_at >= ? AND ae.executed_at <= ?
		AND ae.status = 'completed'
		AND (a.name LIKE '%energy%' OR a.name LIKE '%power%' OR a.name LIKE '%turn off%')
	`

	var executions int
	err := r.db.QueryRowContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(&executions)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate automation savings: %w", err)
	}

	// Estimate average savings per automation execution (configurable value)
	avgSavingsPerExecution := 0.1 // kWh per execution
	return float64(executions) * avgSavingsPerExecution, nil
}

// calculateOptimizationSavings estimates savings from optimization strategies
func (r *EnergyRepository) calculateOptimizationSavings(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	// Look for patterns in energy usage that indicate optimization
	query := `
		SELECT AVG(energy_usage) as avg_usage
		FROM energy_history
		WHERE timestamp >= ? AND timestamp <= ?
		AND strftime('%H', timestamp) BETWEEN '02' AND '06'  -- Low usage hours
	`

	var lowHourUsage float64
	err := r.db.QueryRowContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(&lowHourUsage)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate optimization savings: %w", err)
	}

	// Compare with peak hour usage to estimate optimization effectiveness
	peakQuery := `
		SELECT AVG(energy_usage) as avg_usage
		FROM energy_history
		WHERE timestamp >= ? AND timestamp <= ?
		AND strftime('%H', timestamp) BETWEEN '18' AND '22'  -- Peak usage hours
	`

	var peakHourUsage float64
	err = r.db.QueryRowContext(ctx, peakQuery, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(&peakHourUsage)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate optimization savings: %w", err)
	}

	// Calculate savings based on load shifting effectiveness
	if peakHourUsage > lowHourUsage {
		shiftEfficiency := (peakHourUsage - lowHourUsage) / peakHourUsage
		if shiftEfficiency > 0.1 { // If there's significant load shifting
			return shiftEfficiency * lowHourUsage * 0.15, nil // 15% of shifted load as savings
		}
	}

	return 0, nil
}

// calculateSchedulingSavings estimates savings from scheduled operations
func (r *EnergyRepository) calculateSchedulingSavings(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	// Query for scheduled automation executions
	query := `
		SELECT COUNT(*) as scheduled_executions
		FROM automation_executions ae
		JOIN automations a ON ae.automation_id = a.id
		WHERE ae.executed_at >= ? AND ae.executed_at <= ?
		AND ae.status = 'completed'
		AND (a.trigger_config LIKE '%time%' OR a.trigger_config LIKE '%schedule%')
	`

	var scheduledExecutions int
	err := r.db.QueryRowContext(ctx, query, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)).Scan(&scheduledExecutions)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate scheduling savings: %w", err)
	}

	// Estimate savings from scheduled operations (typically timer-based device control)
	avgSavingsPerSchedule := 0.05 // kWh per scheduled operation
	return float64(scheduledExecutions) * avgSavingsPerSchedule, nil
}
