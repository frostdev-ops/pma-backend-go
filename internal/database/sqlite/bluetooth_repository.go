package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// BluetoothRepository implements repositories.BluetoothRepository
type BluetoothRepository struct {
	db *sql.DB
}

// NewBluetoothRepository creates a new BluetoothRepository
func NewBluetoothRepository(db *sql.DB) repositories.BluetoothRepository {
	return &BluetoothRepository{db: db}
}

// CreateDevice creates a new Bluetooth device
func (r *BluetoothRepository) CreateDevice(ctx context.Context, device *models.BluetoothDevice) error {
	query := `
		INSERT INTO bluetooth_devices (address, name, device_class, is_paired, is_connected, services, last_seen, paired_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	var pairedAt *time.Time
	if device.IsPaired {
		pairedAt = &now
	}

	result, err := r.db.ExecContext(
		ctx,
		query,
		device.Address,
		device.Name,
		device.DeviceClass,
		device.IsPaired,
		device.IsConnected,
		device.Services,
		now,
		pairedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create Bluetooth device: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted device ID: %w", err)
	}

	device.ID = int(id)
	device.LastSeen = now
	if pairedAt != nil {
		device.PairedAt = sql.NullTime{Time: *pairedAt, Valid: true}
	}

	return nil
}

// GetDevice retrieves a Bluetooth device by address
func (r *BluetoothRepository) GetDevice(ctx context.Context, address string) (*models.BluetoothDevice, error) {
	query := `
		SELECT id, address, name, device_class, is_paired, is_connected, services, last_seen, paired_at
		FROM bluetooth_devices
		WHERE address = ?
	`

	device := &models.BluetoothDevice{}
	err := r.db.QueryRowContext(ctx, query, address).Scan(
		&device.ID,
		&device.Address,
		&device.Name,
		&device.DeviceClass,
		&device.IsPaired,
		&device.IsConnected,
		&device.Services,
		&device.LastSeen,
		&device.PairedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Bluetooth device not found with address: %s", address)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Bluetooth device: %w", err)
	}

	return device, nil
}

// GetAllDevices retrieves all Bluetooth devices
func (r *BluetoothRepository) GetAllDevices(ctx context.Context) ([]*models.BluetoothDevice, error) {
	query := `
		SELECT id, address, name, device_class, is_paired, is_connected, services, last_seen, paired_at
		FROM bluetooth_devices
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query Bluetooth devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.BluetoothDevice
	for rows.Next() {
		device := &models.BluetoothDevice{}
		err := rows.Scan(
			&device.ID,
			&device.Address,
			&device.Name,
			&device.DeviceClass,
			&device.IsPaired,
			&device.IsConnected,
			&device.Services,
			&device.LastSeen,
			&device.PairedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan Bluetooth device: %w", err)
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate Bluetooth devices: %w", err)
	}

	return devices, nil
}

// GetPairedDevices retrieves all paired Bluetooth devices
func (r *BluetoothRepository) GetPairedDevices(ctx context.Context) ([]*models.BluetoothDevice, error) {
	query := `
		SELECT id, address, name, device_class, is_paired, is_connected, services, last_seen, paired_at
		FROM bluetooth_devices
		WHERE is_paired = 1
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query paired Bluetooth devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.BluetoothDevice
	for rows.Next() {
		device := &models.BluetoothDevice{}
		err := rows.Scan(
			&device.ID,
			&device.Address,
			&device.Name,
			&device.DeviceClass,
			&device.IsPaired,
			&device.IsConnected,
			&device.Services,
			&device.LastSeen,
			&device.PairedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan paired Bluetooth device: %w", err)
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate paired Bluetooth devices: %w", err)
	}

	return devices, nil
}

// GetConnectedDevices retrieves all connected Bluetooth devices
func (r *BluetoothRepository) GetConnectedDevices(ctx context.Context) ([]*models.BluetoothDevice, error) {
	query := `
		SELECT id, address, name, device_class, is_paired, is_connected, services, last_seen, paired_at
		FROM bluetooth_devices
		WHERE is_connected = 1
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connected Bluetooth devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.BluetoothDevice
	for rows.Next() {
		device := &models.BluetoothDevice{}
		err := rows.Scan(
			&device.ID,
			&device.Address,
			&device.Name,
			&device.DeviceClass,
			&device.IsPaired,
			&device.IsConnected,
			&device.Services,
			&device.LastSeen,
			&device.PairedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connected Bluetooth device: %w", err)
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate connected Bluetooth devices: %w", err)
	}

	return devices, nil
}

// UpdateDevice updates an existing Bluetooth device
func (r *BluetoothRepository) UpdateDevice(ctx context.Context, device *models.BluetoothDevice) error {
	query := `
		UPDATE bluetooth_devices
		SET name = ?, device_class = ?, is_paired = ?, is_connected = ?, services = ?, last_seen = ?, paired_at = ?
		WHERE address = ?
	`

	var pairedAt interface{}
	if device.PairedAt.Valid {
		pairedAt = device.PairedAt.Time
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		device.Name,
		device.DeviceClass,
		device.IsPaired,
		device.IsConnected,
		device.Services,
		time.Now(),
		pairedAt,
		device.Address,
	)
	if err != nil {
		return fmt.Errorf("failed to update Bluetooth device: %w", err)
	}

	return nil
}

// DeleteDevice deletes a Bluetooth device by address
func (r *BluetoothRepository) DeleteDevice(ctx context.Context, address string) error {
	query := `DELETE FROM bluetooth_devices WHERE address = ?`

	result, err := r.db.ExecContext(ctx, query, address)
	if err != nil {
		return fmt.Errorf("failed to delete Bluetooth device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Bluetooth device not found with address: %s", address)
	}

	return nil
}
