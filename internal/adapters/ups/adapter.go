package ups

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// UPSAdapter implements the PMAAdapter interface for UPS devices
type UPSAdapter struct {
	client            *NUTClient
	devices           map[string]*UPSDevice
	logger            *logrus.Logger
	config            UPSAdapterConfig
	mutex             sync.RWMutex
	connected         bool
	lastSyncTime      time.Time
	startTime         time.Time
	actionsExecuted   int
	successfulActions int
	failedActions     int
	syncErrors        int
}

// UPSAdapterConfig holds configuration for the UPS adapter
type UPSAdapterConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	UPSNames     []string      `json:"ups_names"`
	PollInterval time.Duration `json:"poll_interval"`
}

// UPSDevice represents a UPS device
type UPSDevice struct {
	Name     string    `json:"name"`
	Data     *UPSData  `json:"data"`
	LastSeen time.Time `json:"last_seen"`
}

// NewUPSAdapter creates a new UPS adapter
func NewUPSAdapter(config UPSAdapterConfig, logger *logrus.Logger) *UPSAdapter {
	if config.PollInterval == 0 {
		config.PollInterval = 30 * time.Second
	}

	return &UPSAdapter{
		client:    NewNUTClient(config.Host, config.Port, logger),
		devices:   make(map[string]*UPSDevice),
		logger:    logger,
		config:    config,
		startTime: time.Now(),
	}
}

// ========================================
// PMAAdapter Interface Implementation
// ========================================

// GetID returns the unique identifier for this adapter instance
func (a *UPSAdapter) GetID() string {
	return fmt.Sprintf("ups_%s", a.config.Host)
}

// GetSourceType returns the source type for UPS
func (a *UPSAdapter) GetSourceType() types.PMASourceType {
	return types.SourceUPS
}

// GetName returns the adapter name
func (a *UPSAdapter) GetName() string {
	return "UPS Monitoring Adapter"
}

// GetVersion returns the adapter version
func (a *UPSAdapter) GetVersion() string {
	return "1.0.0"
}

// Connect establishes connection to UPS server
func (a *UPSAdapter) Connect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Connecting to UPS server...")

	if err := a.client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to UPS server: %w", err)
	}

	// Discover UPS devices
	for _, upsName := range a.config.UPSNames {
		upsData, err := a.client.GetUPSData(ctx, upsName)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to get data for UPS %s", upsName)
			continue
		}

		device := &UPSDevice{
			Name:     upsName,
			Data:     upsData,
			LastSeen: time.Now(),
		}
		a.devices[upsName] = device
		a.logger.WithField("ups_name", upsName).Info("Discovered UPS device")
	}

	a.connected = true
	a.logger.Infof("Successfully connected to UPS server with %d devices", len(a.devices))
	return nil
}

// Disconnect closes connections to UPS server
func (a *UPSAdapter) Disconnect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.connected = false
	if a.client != nil {
		a.client.Close()
	}
	a.logger.Info("Disconnected from UPS server")
	return nil
}

// IsConnected returns connection status
func (a *UPSAdapter) IsConnected() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.connected
}

// GetStatus returns the adapter status
func (a *UPSAdapter) GetStatus() string {
	if a.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

// ConvertEntity converts a UPS device to PMA sensor entities
func (a *UPSAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	device, ok := sourceEntity.(*UPSDevice)
	if !ok {
		return nil, fmt.Errorf("unsupported UPS entity type: %T", sourceEntity)
	}

	// Create battery level sensor
	return a.convertToBatterySensor(device), nil
}

// ConvertEntities converts multiple UPS devices to PMA entities
func (a *UPSAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
	pmaEntities := make([]types.PMAEntity, 0)

	for _, sourceEntity := range sourceEntities {
		device, ok := sourceEntity.(*UPSDevice)
		if !ok {
			a.logger.Warnf("Skipping non-UPS entity: %T", sourceEntity)
			continue
		}

		// Create multiple sensor entities for different UPS metrics
		entities := a.convertToSensorEntities(device)
		pmaEntities = append(pmaEntities, entities...)
	}

	return pmaEntities, nil
}

// ConvertRoom converts a UPS room to PMA room (not supported)
func (a *UPSAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	return nil, fmt.Errorf("room conversion not supported for UPS devices")
}

// ConvertArea converts a UPS area to PMA area (not supported)
func (a *UPSAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	return nil, fmt.Errorf("area conversion not supported for UPS devices")
}

// ExecuteAction executes control actions on UPS devices (limited support)
func (a *UPSAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	// UPS devices typically have limited control capabilities
	return &types.PMAControlResult{
		Success:     false,
		EntityID:    action.EntityID,
		Action:      action.Action,
		ProcessedAt: time.Now(),
		Error: &types.PMAError{
			Code:     "UPS_CONTROL_NOT_SUPPORTED",
			Message:  "UPS control actions are not supported",
			Source:   "ups",
			EntityID: action.EntityID,
		},
	}, nil
}

// SyncEntities synchronizes entities from UPS devices
func (a *UPSAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	if !a.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	a.mutex.RLock()
	devices := make([]*UPSDevice, 0, len(a.devices))
	for _, device := range a.devices {
		devices = append(devices, device)
	}
	a.mutex.RUnlock()

	// Update UPS data
	for _, device := range devices {
		upsData, err := a.client.GetUPSData(ctx, device.Name)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to update data for UPS %s", device.Name)
			continue
		}

		a.mutex.Lock()
		device.Data = upsData
		device.LastSeen = time.Now()
		a.mutex.Unlock()
	}

	// Convert to interface slice
	sourceEntities := make([]interface{}, len(devices))
	for i, device := range devices {
		sourceEntities[i] = device
	}

	// Convert to PMA entities
	pmaEntities, err := a.ConvertEntities(sourceEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to convert UPS devices: %w", err)
	}

	a.lastSyncTime = time.Now()
	return pmaEntities, nil
}

// SyncRooms synchronizes rooms from UPS (not supported)
func (a *UPSAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	return []*types.PMARoom{}, nil
}

// GetLastSyncTime returns the last synchronization time
func (a *UPSAdapter) GetLastSyncTime() *time.Time {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	if a.lastSyncTime.IsZero() {
		return nil
	}
	return &a.lastSyncTime
}

// GetSupportedEntityTypes returns entity types supported by UPS adapter
func (a *UPSAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeSensor,
		types.EntityTypeDevice,
	}
}

// GetSupportedCapabilities returns capabilities supported by UPS devices
func (a *UPSAdapter) GetSupportedCapabilities() []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityBattery,
	}
}

// SupportsRealtime returns whether UPS supports real-time updates
func (a *UPSAdapter) SupportsRealtime() bool {
	return false // UPS uses polling
}

// GetHealth returns adapter health information
func (a *UPSAdapter) GetHealth() *types.AdapterHealth {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	issues := []string{}
	if !a.connected {
		issues = append(issues, "Not connected to UPS server")
	}

	// Check UPS device status
	criticalUPS := 0
	for _, device := range a.devices {
		if device.Data != nil && device.Data.BatteryCharge != nil && *device.Data.BatteryCharge < 20 {
			criticalUPS++
		}
	}

	if criticalUPS > 0 {
		issues = append(issues, fmt.Sprintf("%d UPS devices have critical battery levels", criticalUPS))
	}

	return &types.AdapterHealth{
		IsHealthy:       len(issues) == 0,
		LastHealthCheck: time.Now(),
		Issues:          issues,
		ResponseTime:    100 * time.Millisecond,
		ErrorRate:       a.calculateErrorRate(),
		Details: map[string]interface{}{
			"connected":    a.connected,
			"device_count": len(a.devices),
			"critical_ups": criticalUPS,
		},
	}
}

// GetMetrics returns adapter performance metrics
func (a *UPSAdapter) GetMetrics() *types.AdapterMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var lastSync *time.Time
	if !a.lastSyncTime.IsZero() {
		lastSync = &a.lastSyncTime
	}

	// Each UPS device creates multiple sensor entities
	entityCount := len(a.devices) * 4 // battery, load, runtime, voltage sensors

	return &types.AdapterMetrics{
		EntitiesManaged:     entityCount,
		RoomsManaged:        0,
		ActionsExecuted:     int64(a.actionsExecuted),
		SuccessfulActions:   int64(a.successfulActions),
		FailedActions:       int64(a.failedActions),
		AverageResponseTime: 100 * time.Millisecond,
		LastSync:            lastSync,
		SyncErrors:          a.syncErrors,
		Uptime:              time.Since(a.startTime),
	}
}

// Helper methods
func (a *UPSAdapter) calculateErrorRate() float64 {
	if a.actionsExecuted == 0 {
		return 0.0
	}
	return float64(a.failedActions) / float64(a.actionsExecuted)
}

func (a *UPSAdapter) convertToBatterySensor(device *UPSDevice) types.PMAEntity {
	var batteryLevel float64
	if device.Data.BatteryCharge != nil {
		batteryLevel = *device.Data.BatteryCharge
	}

	entity := &types.PMABaseEntity{
		ID:           fmt.Sprintf("ups_%s_battery", device.Name),
		Type:         types.EntityTypeSensor,
		FriendlyName: fmt.Sprintf("%s Battery Level", device.Name),
		Icon:         "mdi:battery",
		State:        types.StateActive,
		Attributes: map[string]interface{}{
			"device_type":  "ups_sensor",
			"sensor_type":  "battery",
			"value":        batteryLevel,
			"unit":         "%",
			"ups_name":     device.Name,
			"ups_status":   device.Data.Status,
			"manufacturer": device.Data.Manufacturer,
			"model":        device.Data.Model,
		},
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{types.CapabilityBattery},
		Metadata: &types.PMAMetadata{
			Source:         types.SourceUPS,
			SourceEntityID: device.Name,
			SourceData: map[string]interface{}{
				"ups_name": device.Name,
				"sensor":   "battery",
			},
			LastSynced:   time.Now(),
			QualityScore: 0.95,
		},
		Available: time.Since(device.LastSeen) < 5*time.Minute,
	}

	return entity
}

func (a *UPSAdapter) convertToSensorEntities(device *UPSDevice) []types.PMAEntity {
	entities := make([]types.PMAEntity, 0, 4)

	// Battery level sensor
	if device.Data.BatteryCharge != nil {
		entity := &types.PMABaseEntity{
			ID:           fmt.Sprintf("ups_%s_battery", device.Name),
			Type:         types.EntityTypeSensor,
			FriendlyName: fmt.Sprintf("%s Battery Level", device.Name),
			Icon:         "mdi:battery",
			State:        types.StateActive,
			Attributes: map[string]interface{}{
				"value":      *device.Data.BatteryCharge,
				"unit":       "%",
				"ups_name":   device.Name,
				"ups_status": device.Data.Status,
			},
			LastUpdated:  time.Now(),
			Capabilities: []types.PMACapability{types.CapabilityBattery},
			Metadata: &types.PMAMetadata{
				Source:         types.SourceUPS,
				SourceEntityID: fmt.Sprintf("%s_battery", device.Name),
				SourceData:     map[string]interface{}{"ups_name": device.Name, "sensor": "battery"},
				LastSynced:     time.Now(),
				QualityScore:   0.95,
			},
			Available: time.Since(device.LastSeen) < 5*time.Minute,
		}
		entities = append(entities, entity)
	}

	// Load sensor
	if device.Data.LoadPercent != nil {
		entity := &types.PMABaseEntity{
			ID:           fmt.Sprintf("ups_%s_load", device.Name),
			Type:         types.EntityTypeSensor,
			FriendlyName: fmt.Sprintf("%s Load", device.Name),
			Icon:         "mdi:flash",
			State:        types.StateActive,
			Attributes: map[string]interface{}{
				"value":      *device.Data.LoadPercent,
				"unit":       "%",
				"ups_name":   device.Name,
				"ups_status": device.Data.Status,
			},
			LastUpdated:  time.Now(),
			Capabilities: []types.PMACapability{},
			Metadata: &types.PMAMetadata{
				Source:         types.SourceUPS,
				SourceEntityID: fmt.Sprintf("%s_load", device.Name),
				SourceData:     map[string]interface{}{"ups_name": device.Name, "sensor": "load"},
				LastSynced:     time.Now(),
				QualityScore:   0.95,
			},
			Available: time.Since(device.LastSeen) < 5*time.Minute,
		}
		entities = append(entities, entity)
	}

	// Runtime sensor
	if device.Data.BatteryRuntime != nil {
		entity := &types.PMABaseEntity{
			ID:           fmt.Sprintf("ups_%s_runtime", device.Name),
			Type:         types.EntityTypeSensor,
			FriendlyName: fmt.Sprintf("%s Runtime", device.Name),
			Icon:         "mdi:timer",
			State:        types.StateActive,
			Attributes: map[string]interface{}{
				"value":      *device.Data.BatteryRuntime,
				"unit":       "seconds",
				"ups_name":   device.Name,
				"ups_status": device.Data.Status,
			},
			LastUpdated:  time.Now(),
			Capabilities: []types.PMACapability{},
			Metadata: &types.PMAMetadata{
				Source:         types.SourceUPS,
				SourceEntityID: fmt.Sprintf("%s_runtime", device.Name),
				SourceData:     map[string]interface{}{"ups_name": device.Name, "sensor": "runtime"},
				LastSynced:     time.Now(),
				QualityScore:   0.95,
			},
			Available: time.Since(device.LastSeen) < 5*time.Minute,
		}
		entities = append(entities, entity)
	}

	// Output voltage sensor
	if device.Data.OutputVoltage != nil {
		entity := &types.PMABaseEntity{
			ID:           fmt.Sprintf("ups_%s_voltage", device.Name),
			Type:         types.EntityTypeSensor,
			FriendlyName: fmt.Sprintf("%s Output Voltage", device.Name),
			Icon:         "mdi:flash-triangle",
			State:        types.StateActive,
			Attributes: map[string]interface{}{
				"value":      *device.Data.OutputVoltage,
				"unit":       "V",
				"ups_name":   device.Name,
				"ups_status": device.Data.Status,
			},
			LastUpdated:  time.Now(),
			Capabilities: []types.PMACapability{},
			Metadata: &types.PMAMetadata{
				Source:         types.SourceUPS,
				SourceEntityID: fmt.Sprintf("%s_voltage", device.Name),
				SourceData:     map[string]interface{}{"ups_name": device.Name, "sensor": "voltage"},
				LastSynced:     time.Now(),
				QualityScore:   0.95,
			},
			Available: time.Since(device.LastSeen) < 5*time.Minute,
		}
		entities = append(entities, entity)
	}

	return entities
}
