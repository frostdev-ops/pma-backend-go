package repositories

import (
	"context"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
)

// UserRepository defines user data access methods
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetAll(ctx context.Context) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
}

// ConfigRepository defines system config data access methods
type ConfigRepository interface {
	Get(ctx context.Context, key string) (*models.SystemConfig, error)
	Set(ctx context.Context, config *models.SystemConfig) error
	GetAll(ctx context.Context) ([]*models.SystemConfig, error)
	Delete(ctx context.Context, key string) error
}

// EntityRepository defines entity data access methods
type EntityRepository interface {
	Create(ctx context.Context, entity *models.Entity) error
	GetByID(ctx context.Context, entityID string) (*models.Entity, error)
	GetAll(ctx context.Context) ([]*models.Entity, error)
	GetByRoom(ctx context.Context, roomID int) ([]*models.Entity, error)
	Update(ctx context.Context, entity *models.Entity) error
	Delete(ctx context.Context, entityID string) error
}

// RoomRepository defines room data access methods
type RoomRepository interface {
	Create(ctx context.Context, room *models.Room) error
	GetByID(ctx context.Context, id int) (*models.Room, error)
	GetByName(ctx context.Context, name string) (*models.Room, error)
	GetAll(ctx context.Context) ([]*models.Room, error)
	Update(ctx context.Context, room *models.Room) error
	Delete(ctx context.Context, id int) error
}

// AuthRepository defines authentication data access methods
type AuthRepository interface {
	GetSettings(ctx context.Context) (*models.AuthSetting, error)
	SetSettings(ctx context.Context, settings *models.AuthSetting) error
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, token string) (*models.Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteExpiredSessions(ctx context.Context) error
	RecordFailedAttempt(ctx context.Context, attempt *models.FailedAuthAttempt) error
	GetFailedAttempts(ctx context.Context, clientID string, since int64) ([]*models.FailedAuthAttempt, error)
	CleanupFailedAttempts(ctx context.Context, before int64) error
}

// KioskRepository defines kiosk device data access methods
type KioskRepository interface {
	CreateToken(ctx context.Context, token *models.KioskToken) error
	GetToken(ctx context.Context, token string) (*models.KioskToken, error)
	UpdateTokenLastUsed(ctx context.Context, token string) error
	DeleteToken(ctx context.Context, token string) error
	GetAllTokens(ctx context.Context) ([]*models.KioskToken, error)
	CreatePairingSession(ctx context.Context, session *models.KioskPairingSession) error
	GetPairingSession(ctx context.Context, pin string) (*models.KioskPairingSession, error)
	UpdatePairingSession(ctx context.Context, session *models.KioskPairingSession) error
	DeletePairingSession(ctx context.Context, id string) error
	CleanupExpiredSessions(ctx context.Context) error
}

// NetworkRepository defines network device data access methods
type NetworkRepository interface {
	CreateDevice(ctx context.Context, device *models.NetworkDevice) error
	GetDevice(ctx context.Context, ipAddress string) (*models.NetworkDevice, error)
	GetAllDevices(ctx context.Context) ([]*models.NetworkDevice, error)
	GetOnlineDevices(ctx context.Context) ([]*models.NetworkDevice, error)
	UpdateDevice(ctx context.Context, device *models.NetworkDevice) error
	DeleteDevice(ctx context.Context, ipAddress string) error
	UpdateDeviceStatus(ctx context.Context, ipAddress string, isOnline bool) error
}

// UPSRepository defines UPS monitoring data access methods
type UPSRepository interface {
	CreateStatus(ctx context.Context, status *models.UPSStatus) error
	GetLatestStatus(ctx context.Context) (*models.UPSStatus, error)
	GetStatusHistory(ctx context.Context, limit int) ([]*models.UPSStatus, error)
	CleanupOldStatus(ctx context.Context, keepDays int) error
	GetStatusByTimeRange(ctx context.Context, start, end time.Time) ([]*models.UPSStatus, error)
	GetBatteryTrends(ctx context.Context, hours int) ([]*models.UPSStatus, error)
}

// CameraRepository defines camera device data access methods
type CameraRepository interface {
	Create(ctx context.Context, camera *models.Camera) error
	GetByID(ctx context.Context, id int) (*models.Camera, error)
	GetByEntityID(ctx context.Context, entityID string) (*models.Camera, error)
	GetAll(ctx context.Context) ([]*models.Camera, error)
	GetEnabled(ctx context.Context) ([]*models.Camera, error)
	Update(ctx context.Context, camera *models.Camera) error
	Delete(ctx context.Context, id int) error
}

// DisplayRepository defines display settings data access methods
type DisplayRepository interface {
	GetSettings(ctx context.Context) (*models.DisplaySettings, error)
	UpdateSettings(ctx context.Context, settings *models.DisplaySettings) error
}

// BluetoothRepository defines Bluetooth device data access methods
type BluetoothRepository interface {
	CreateDevice(ctx context.Context, device *models.BluetoothDevice) error
	GetDevice(ctx context.Context, address string) (*models.BluetoothDevice, error)
	GetAllDevices(ctx context.Context) ([]*models.BluetoothDevice, error)
	GetPairedDevices(ctx context.Context) ([]*models.BluetoothDevice, error)
	GetConnectedDevices(ctx context.Context) ([]*models.BluetoothDevice, error)
	UpdateDevice(ctx context.Context, device *models.BluetoothDevice) error
	DeleteDevice(ctx context.Context, address string) error
}
