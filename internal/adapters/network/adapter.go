package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// NetworkAdapter implements the PMAAdapter interface for network devices
type NetworkAdapter struct {
	client            *Client
	devices           map[string]*NetworkAdapterDevice
	logger            *logrus.Logger
	config            NetworkAdapterConfig
	mutex             sync.RWMutex
	connected         bool
	lastSyncTime      time.Time
	startTime         time.Time
	actionsExecuted   int
	successfulActions int
	failedActions     int
	syncErrors        int
}

// NetworkAdapterConfig holds configuration for the Network adapter
type NetworkAdapterConfig struct {
	RouterURL    string        `json:"router_url"`
	AuthToken    string        `json:"auth_token"`
	PollInterval time.Duration `json:"poll_interval"`
}

// NetworkAdapterDevice represents a network device managed by the adapter
type NetworkAdapterDevice struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	IPAddress  string                 `json:"ip_address"`
	MACAddress string                 `json:"mac_address"`
	Status     string                 `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	LastSeen   time.Time              `json:"last_seen"`
}

// NewNetworkAdapter creates a new Network adapter
func NewNetworkAdapter(config NetworkAdapterConfig, logger *logrus.Logger) *NetworkAdapter {
	if config.PollInterval == 0 {
		config.PollInterval = 60 * time.Second
	}

	return &NetworkAdapter{
		client:    NewClient(config.RouterURL, config.AuthToken),
		devices:   make(map[string]*NetworkAdapterDevice),
		logger:    logger,
		config:    config,
		startTime: time.Now(),
	}
}

// ========================================
// PMAAdapter Interface Implementation
// ========================================

// GetID returns the unique identifier for this adapter instance
func (a *NetworkAdapter) GetID() string {
	return "network_adapter"
}

// GetSourceType returns the source type for Network
func (a *NetworkAdapter) GetSourceType() types.PMASourceType {
	return types.SourceNetwork
}

// GetName returns the adapter name
func (a *NetworkAdapter) GetName() string {
	return "Network Device Adapter"
}

// GetVersion returns the adapter version
func (a *NetworkAdapter) GetVersion() string {
	return "1.0.0"
}

// Connect establishes connection to network router
func (a *NetworkAdapter) Connect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Connecting to network router...")

	// Test connection
	_, err := a.client.GetSystemStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to network router: %w", err)
	}

	// Discover network devices/interfaces
	interfaces, err := a.client.GetNetworkInterfaces(ctx)
	if err != nil {
		a.logger.WithError(err).Warn("Failed to get network interfaces")
	} else {
		for _, iface := range interfaces {
			device := &NetworkAdapterDevice{
				ID:     fmt.Sprintf("network_interface_%s", iface.Name),
				Name:   fmt.Sprintf("Interface %s", iface.Name),
				Type:   "network_interface",
				Status: iface.State,
				Attributes: map[string]interface{}{
					"index": iface.Index,
					"mtu":   iface.MTU,
				},
				LastSeen: time.Now(),
			}
			a.devices[device.ID] = device
			a.logger.WithField("interface", iface.Name).Info("Discovered network interface")
		}
	}

	a.connected = true
	a.logger.Infof("Successfully connected to network router with %d interfaces", len(a.devices))
	return nil
}

// Disconnect closes connections to network router
func (a *NetworkAdapter) Disconnect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.connected = false
	a.logger.Info("Disconnected from network router")
	return nil
}

// IsConnected returns connection status
func (a *NetworkAdapter) IsConnected() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.connected
}

// GetStatus returns the adapter status
func (a *NetworkAdapter) GetStatus() string {
	if a.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

// ConvertEntity converts a network device to PMA entity
func (a *NetworkAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	device, ok := sourceEntity.(*NetworkAdapterDevice)
	if !ok {
		return nil, fmt.Errorf("unsupported network entity type: %T", sourceEntity)
	}

	return a.convertToDeviceEntity(device), nil
}

// ConvertEntities converts multiple network devices to PMA entities
func (a *NetworkAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
	pmaEntities := make([]types.PMAEntity, 0, len(sourceEntities))

	for _, sourceEntity := range sourceEntities {
		entity, err := a.ConvertEntity(sourceEntity)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to convert entity: %v", sourceEntity)
			continue
		}
		pmaEntities = append(pmaEntities, entity)
	}

	return pmaEntities, nil
}

// ConvertRoom converts a network room to PMA room (not supported)
func (a *NetworkAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	return nil, fmt.Errorf("room conversion not supported for network devices")
}

// ConvertArea converts a network area to PMA area (not supported)
func (a *NetworkAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	return nil, fmt.Errorf("area conversion not supported for network devices")
}

// ExecuteAction executes control actions on network devices (limited support)
func (a *NetworkAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	// Network devices typically have limited control capabilities
	return &types.PMAControlResult{
		Success:     false,
		EntityID:    action.EntityID,
		Action:      action.Action,
		ProcessedAt: time.Now(),
		Error: &types.PMAError{
			Code:     "NETWORK_CONTROL_LIMITED",
			Message:  "Network device control actions are limited",
			Source:   "network",
			EntityID: action.EntityID,
		},
	}, nil
}

// SyncEntities synchronizes entities from network devices
func (a *NetworkAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	if !a.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	// Update network interface status
	interfaces, err := a.client.GetNetworkInterfaces(ctx)
	if err != nil {
		a.logger.WithError(err).Warn("Failed to update network interfaces")
	} else {
		a.mutex.Lock()
		for _, iface := range interfaces {
			deviceID := fmt.Sprintf("network_interface_%s", iface.Name)
			if device, exists := a.devices[deviceID]; exists {
				device.Status = iface.State
				device.LastSeen = time.Now()
			}
		}
		a.mutex.Unlock()
	}

	a.mutex.RLock()
	devices := make([]*NetworkAdapterDevice, 0, len(a.devices))
	for _, device := range a.devices {
		devices = append(devices, device)
	}
	a.mutex.RUnlock()

	// Convert to interface slice
	sourceEntities := make([]interface{}, len(devices))
	for i, device := range devices {
		sourceEntities[i] = device
	}

	// Convert to PMA entities
	pmaEntities, err := a.ConvertEntities(sourceEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to convert network devices: %w", err)
	}

	a.lastSyncTime = time.Now()
	return pmaEntities, nil
}

// SyncRooms synchronizes rooms from network (not supported)
func (a *NetworkAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	return []*types.PMARoom{}, nil
}

// GetLastSyncTime returns the last synchronization time
func (a *NetworkAdapter) GetLastSyncTime() *time.Time {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	if a.lastSyncTime.IsZero() {
		return nil
	}
	return &a.lastSyncTime
}

// GetSupportedEntityTypes returns entity types supported by Network adapter
func (a *NetworkAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeDevice,
		types.EntityTypeSensor,
	}
}

// GetSupportedCapabilities returns capabilities supported by network devices
func (a *NetworkAdapter) GetSupportedCapabilities() []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityConnectivity,
	}
}

// SupportsRealtime returns whether network supports real-time updates
func (a *NetworkAdapter) SupportsRealtime() bool {
	return false // Network uses polling
}

// GetHealth returns adapter health information
func (a *NetworkAdapter) GetHealth() *types.AdapterHealth {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	issues := []string{}
	if !a.connected {
		issues = append(issues, "Not connected to network router")
	}

	// Check interface status
	downInterfaces := 0
	for _, device := range a.devices {
		if device.Status == "down" {
			downInterfaces++
		}
	}

	if downInterfaces > 0 {
		issues = append(issues, fmt.Sprintf("%d network interfaces are down", downInterfaces))
	}

	return &types.AdapterHealth{
		IsHealthy:       len(issues) == 0,
		LastHealthCheck: time.Now(),
		Issues:          issues,
		ResponseTime:    50 * time.Millisecond,
		ErrorRate:       a.calculateErrorRate(),
		Details: map[string]interface{}{
			"connected":       a.connected,
			"interface_count": len(a.devices),
			"down_interfaces": downInterfaces,
		},
	}
}

// GetMetrics returns adapter performance metrics
func (a *NetworkAdapter) GetMetrics() *types.AdapterMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var lastSync *time.Time
	if !a.lastSyncTime.IsZero() {
		lastSync = &a.lastSyncTime
	}

	return &types.AdapterMetrics{
		EntitiesManaged:     len(a.devices),
		RoomsManaged:        0,
		ActionsExecuted:     int64(a.actionsExecuted),
		SuccessfulActions:   int64(a.successfulActions),
		FailedActions:       int64(a.failedActions),
		AverageResponseTime: 50 * time.Millisecond,
		LastSync:            lastSync,
		SyncErrors:          a.syncErrors,
		Uptime:              time.Since(a.startTime),
	}
}

// Helper methods
func (a *NetworkAdapter) calculateErrorRate() float64 {
	if a.actionsExecuted == 0 {
		return 0.0
	}
	return float64(a.failedActions) / float64(a.actionsExecuted)
}

func (a *NetworkAdapter) convertToDeviceEntity(device *NetworkAdapterDevice) types.PMAEntity {
	var state types.PMAEntityState
	switch device.Status {
	case "up":
		state = types.StateActive
	case "down":
		state = types.StateUnavailable
	default:
		state = types.StateUnknown
	}

	entity := &types.PMABaseEntity{
		ID:           device.ID,
		Type:         types.EntityTypeDevice,
		FriendlyName: device.Name,
		Icon:         "mdi:ethernet",
		State:        state,
		Attributes: map[string]interface{}{
			"device_type": "network_device",
			"type":        device.Type,
			"ip_address":  device.IPAddress,
			"mac_address": device.MACAddress,
			"status":      device.Status,
		},
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{types.CapabilityConnectivity},
		Metadata: &types.PMAMetadata{
			Source:         types.SourceNetwork,
			SourceEntityID: device.ID,
			SourceData: map[string]interface{}{
				"device_id":   device.ID,
				"device_type": device.Type,
				"ip_address":  device.IPAddress,
				"mac_address": device.MACAddress,
			},
			LastSynced:   time.Now(),
			QualityScore: 0.85,
		},
		Available: time.Since(device.LastSeen) < 10*time.Minute,
	}

	// Add device-specific attributes
	for key, value := range device.Attributes {
		entity.Attributes[key] = value
	}

	return entity
}
