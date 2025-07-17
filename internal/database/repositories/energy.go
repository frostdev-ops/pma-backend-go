package repositories

import (
	"context"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/energy"
)

// EnergyRepository defines energy data access methods
type EnergyRepository interface {
	// Settings management
	GetSettings(ctx context.Context) (*energy.EnergySettings, error)
	UpdateSettings(ctx context.Context, settings *energy.EnergySettings) error

	// Energy history management
	CreateEnergyHistory(ctx context.Context, history *energy.EnergyHistory) error
	GetEnergyHistory(ctx context.Context, filter *energy.EnergyHistoryFilter) ([]*energy.EnergyHistory, error)
	GetEnergyHistoryCount(ctx context.Context, filter *energy.EnergyHistoryFilter) (int, error)
	CleanupOldHistory(ctx context.Context, days int) error

	// Device energy management
	CreateDeviceEnergy(ctx context.Context, deviceEnergy *energy.DeviceEnergy) error
	CreateDeviceEnergyBatch(ctx context.Context, deviceEnergies []*energy.DeviceEnergy) error
	GetDeviceEnergy(ctx context.Context, filter *energy.DeviceEnergyFilter) ([]*energy.DeviceEnergy, error)
	GetDeviceEnergyCount(ctx context.Context, filter *energy.DeviceEnergyFilter) (int, error)
	GetDeviceEnergyByEntity(ctx context.Context, entityID string, startDate, endDate time.Time) ([]*energy.DeviceEnergy, error)
	GetTopEnergyConsumers(ctx context.Context, limit int, startDate, endDate time.Time) ([]*energy.DeviceEnergy, error)
	CleanupOldDeviceEnergy(ctx context.Context, days int) error

	// Statistics and aggregations
	GetEnergyStats(ctx context.Context, startDate, endDate time.Time) (*energy.EnergyStats, error)
	GetTotalEnergyConsumption(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetTotalEnergyCost(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetDeviceEnergyMetrics(ctx context.Context) (*energy.EnergyMetrics, error)
}
