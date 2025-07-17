package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// NetworkRepository implements repositories.NetworkRepository
type NetworkRepository struct {
	db *sql.DB
}

// NewNetworkRepository creates a new NetworkRepository
func NewNetworkRepository(db *sql.DB) repositories.NetworkRepository {
	return &NetworkRepository{db: db}
}

// CreateDevice creates a new network device
func (r *NetworkRepository) CreateDevice(ctx context.Context, device *models.NetworkDevice) error {
	query := `
		INSERT INTO network_devices (ip_address, mac_address, hostname, manufacturer, device_type, last_seen, first_seen, is_online, services, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	device.LastSeen = now
	device.FirstSeen = now

	_, err := r.db.ExecContext(
		ctx,
		query,
		device.IPAddress,
		device.MACAddress,
		device.Hostname,
		device.Manufacturer,
		device.DeviceType,
		device.LastSeen,
		device.FirstSeen,
		device.IsOnline,
		device.Services,
		device.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create network device: %w", err)
	}

	return nil
}

// GetDevice retrieves a network device by IP address
func (r *NetworkRepository) GetDevice(ctx context.Context, ipAddress string) (*models.NetworkDevice, error) {
	query := `
		SELECT id, ip_address, mac_address, hostname, manufacturer, device_type, last_seen, first_seen, is_online, services, metadata
		FROM network_devices
		WHERE ip_address = ?
	`

	device := &models.NetworkDevice{}
	err := r.db.QueryRowContext(ctx, query, ipAddress).Scan(
		&device.ID,
		&device.IPAddress,
		&device.MACAddress,
		&device.Hostname,
		&device.Manufacturer,
		&device.DeviceType,
		&device.LastSeen,
		&device.FirstSeen,
		&device.IsOnline,
		&device.Services,
		&device.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("network device not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network device: %w", err)
	}

	return device, nil
}

// GetAllDevices retrieves all network devices
func (r *NetworkRepository) GetAllDevices(ctx context.Context) ([]*models.NetworkDevice, error) {
	query := `
		SELECT id, ip_address, mac_address, hostname, manufacturer, device_type, last_seen, first_seen, is_online, services, metadata
		FROM network_devices
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query network devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.NetworkDevice
	for rows.Next() {
		device := &models.NetworkDevice{}
		err := rows.Scan(
			&device.ID,
			&device.IPAddress,
			&device.MACAddress,
			&device.Hostname,
			&device.Manufacturer,
			&device.DeviceType,
			&device.LastSeen,
			&device.FirstSeen,
			&device.IsOnline,
			&device.Services,
			&device.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan network device: %w", err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// GetOnlineDevices retrieves all online network devices
func (r *NetworkRepository) GetOnlineDevices(ctx context.Context) ([]*models.NetworkDevice, error) {
	query := `
		SELECT id, ip_address, mac_address, hostname, manufacturer, device_type, last_seen, first_seen, is_online, services, metadata
		FROM network_devices
		WHERE is_online = TRUE
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query online devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.NetworkDevice
	for rows.Next() {
		device := &models.NetworkDevice{}
		err := rows.Scan(
			&device.ID,
			&device.IPAddress,
			&device.MACAddress,
			&device.Hostname,
			&device.Manufacturer,
			&device.DeviceType,
			&device.LastSeen,
			&device.FirstSeen,
			&device.IsOnline,
			&device.Services,
			&device.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan online device: %w", err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// UpdateDevice updates a network device
func (r *NetworkRepository) UpdateDevice(ctx context.Context, device *models.NetworkDevice) error {
	query := `
		UPDATE network_devices 
		SET mac_address = ?, hostname = ?, manufacturer = ?, device_type = ?, last_seen = ?, is_online = ?, services = ?, metadata = ?
		WHERE ip_address = ?
	`

	device.LastSeen = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		device.MACAddress,
		device.Hostname,
		device.Manufacturer,
		device.DeviceType,
		device.LastSeen,
		device.IsOnline,
		device.Services,
		device.Metadata,
		device.IPAddress,
	)

	if err != nil {
		return fmt.Errorf("failed to update network device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("network device not found")
	}

	return nil
}

// DeleteDevice removes a network device
func (r *NetworkRepository) DeleteDevice(ctx context.Context, ipAddress string) error {
	query := `DELETE FROM network_devices WHERE ip_address = ?`

	result, err := r.db.ExecContext(ctx, query, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to delete network device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("network device not found")
	}

	return nil
}

// UpdateDeviceStatus updates the online status of a device
func (r *NetworkRepository) UpdateDeviceStatus(ctx context.Context, ipAddress string, isOnline bool) error {
	query := `UPDATE network_devices SET is_online = ?, last_seen = ? WHERE ip_address = ?`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, isOnline, now, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("network device not found")
	}

	return nil
}
