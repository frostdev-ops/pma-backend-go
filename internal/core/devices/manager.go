package devices

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DeviceManager manages all device adapters and provides a unified interface
type DeviceManager struct {
	adapters       map[string]DeviceAdapter
	devices        map[string]Device
	eventCallbacks []func(DeviceEvent)
	repository     DeviceRepository
	logger         *logrus.Logger
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	healthTicker   *time.Ticker
}

// DeviceManagerConfig holds configuration for the device manager
type DeviceManagerConfig struct {
	HealthCheckInterval time.Duration
}

// NewDeviceManager creates a new device manager instance
func NewDeviceManager(repo DeviceRepository, logger *logrus.Logger, config DeviceManagerConfig) *DeviceManager {
	ctx, cancel := context.WithCancel(context.Background())

	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	return &DeviceManager{
		adapters:     make(map[string]DeviceAdapter),
		devices:      make(map[string]Device),
		repository:   repo,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		healthTicker: time.NewTicker(config.HealthCheckInterval),
	}
}

// RegisterAdapter registers a new device adapter
func (dm *DeviceManager) RegisterAdapter(adapter DeviceAdapter) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	adapterType := adapter.GetType()
	if _, exists := dm.adapters[adapterType]; exists {
		return fmt.Errorf("%w: %s", ErrAdapterAlreadyExists, adapterType)
	}

	dm.adapters[adapterType] = adapter
	dm.logger.WithField("adapter_type", adapterType).Info("Device adapter registered")

	// Subscribe to adapter events
	if err := adapter.Subscribe(dm.handleDeviceEvent); err != nil {
		dm.logger.WithError(err).WithField("adapter_type", adapterType).
			Error("Failed to subscribe to adapter events")
	}

	return nil
}

// UnregisterAdapter removes a device adapter
func (dm *DeviceManager) UnregisterAdapter(adapterType string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	adapter, exists := dm.adapters[adapterType]
	if !exists {
		return fmt.Errorf("%w: %s", ErrAdapterNotFound, adapterType)
	}

	// Disconnect and cleanup
	adapter.Unsubscribe()
	adapter.Disconnect()

	// Remove devices from this adapter
	for deviceID, device := range dm.devices {
		if device.GetAdapterType() == adapterType {
			delete(dm.devices, deviceID)
		}
	}

	delete(dm.adapters, adapterType)
	dm.logger.WithField("adapter_type", adapterType).Info("Device adapter unregistered")

	return nil
}

// ConnectAdapter connects a specific adapter
func (dm *DeviceManager) ConnectAdapter(adapterType string) error {
	dm.mutex.RLock()
	adapter, exists := dm.adapters[adapterType]
	dm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("%w: %s", ErrAdapterNotFound, adapterType)
	}

	return adapter.Connect(dm.ctx)
}

// ConnectAll connects all registered adapters
func (dm *DeviceManager) ConnectAll() error {
	dm.mutex.RLock()
	adapters := make([]DeviceAdapter, 0, len(dm.adapters))
	for _, adapter := range dm.adapters {
		adapters = append(adapters, adapter)
	}
	dm.mutex.RUnlock()

	var errors []error
	for _, adapter := range adapters {
		if err := adapter.Connect(dm.ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to connect %s: %w", adapter.GetType(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("connection errors: %v", errors)
	}

	return nil
}

// DiscoverDevices discovers devices from all or specific adapters
func (dm *DeviceManager) DiscoverDevices(adapterTypes ...string) (*DeviceDiscoveryResult, error) {
	result := &DeviceDiscoveryResult{
		Devices: make([]Device, 0),
		Errors:  make([]error, 0),
	}

	dm.mutex.RLock()
	adaptersToDiscover := make([]DeviceAdapter, 0)

	if len(adapterTypes) == 0 {
		// Discover from all adapters
		for _, adapter := range dm.adapters {
			adaptersToDiscover = append(adaptersToDiscover, adapter)
		}
	} else {
		// Discover from specific adapters
		for _, adapterType := range adapterTypes {
			if adapter, exists := dm.adapters[adapterType]; exists {
				adaptersToDiscover = append(adaptersToDiscover, adapter)
			} else {
				result.Errors = append(result.Errors, fmt.Errorf("%w: %s", ErrAdapterNotFound, adapterType))
			}
		}
	}
	dm.mutex.RUnlock()

	// Perform discovery
	for _, adapter := range adaptersToDiscover {
		devices, err := adapter.Discover(dm.ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("discovery failed for %s: %w", adapter.GetType(), err))
			continue
		}

		result.Devices = append(result.Devices, devices...)

		// Update local device registry
		dm.mutex.Lock()
		for _, device := range devices {
			dm.devices[device.GetID()] = device
		}
		dm.mutex.Unlock()
	}

	return result, nil
}

// GetDevice retrieves a specific device by ID
func (dm *DeviceManager) GetDevice(deviceID string) (Device, error) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	device, exists := dm.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrDeviceNotFound, deviceID)
	}

	return device, nil
}

// GetDevices returns all registered devices
func (dm *DeviceManager) GetDevices() []Device {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	devices := make([]Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		devices = append(devices, device)
	}

	return devices
}

// GetDevicesByType returns devices of a specific type
func (dm *DeviceManager) GetDevicesByType(deviceType string) []Device {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	devices := make([]Device, 0)
	for _, device := range dm.devices {
		if device.GetType() == deviceType {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetDevicesByAdapter returns devices from a specific adapter
func (dm *DeviceManager) GetDevicesByAdapter(adapterType string) []Device {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	devices := make([]Device, 0)
	for _, device := range dm.devices {
		if device.GetAdapterType() == adapterType {
			devices = append(devices, device)
		}
	}

	return devices
}

// ExecuteCommand executes a command on a specific device
func (dm *DeviceManager) ExecuteCommand(deviceID, command string, params map[string]interface{}) (interface{}, error) {
	device, err := dm.GetDevice(deviceID)
	if err != nil {
		return nil, err
	}

	result, err := device.Execute(command, params)
	if err != nil {
		dm.logger.WithError(err).
			WithField("device_id", deviceID).
			WithField("command", command).
			Error("Command execution failed")
		return nil, err
	}

	// Emit event for command execution
	dm.emitEvent(DeviceEvent{
		DeviceID:    deviceID,
		AdapterType: device.GetAdapterType(),
		EventType:   "command_executed",
		Data: map[string]interface{}{
			"command": command,
			"params":  params,
			"result":  result,
		},
		Timestamp: time.Now(),
		Source:    "device_manager",
	})

	return result, nil
}

// SetDeviceState updates a device's state
func (dm *DeviceManager) SetDeviceState(deviceID, key string, value interface{}) error {
	device, err := dm.GetDevice(deviceID)
	if err != nil {
		return err
	}

	oldValue := device.GetState()[key]
	if err := device.SetState(key, value); err != nil {
		return err
	}

	// Emit state change event
	dm.emitEvent(DeviceEvent{
		DeviceID:    deviceID,
		AdapterType: device.GetAdapterType(),
		EventType:   EventTypeStateChanged,
		Data: map[string]interface{}{
			"key":       key,
			"old_value": oldValue,
			"new_value": value,
			"state":     device.GetState(),
		},
		Timestamp: time.Now(),
		Source:    "device_manager",
	})

	// Save state to repository
	if dm.repository != nil {
		if err := dm.repository.SaveDeviceState(deviceID, device.GetState()); err != nil {
			dm.logger.WithError(err).WithField("device_id", deviceID).
				Error("Failed to save device state")
		}
	}

	return nil
}

// RegisterDevice manually registers a device
func (dm *DeviceManager) RegisterDevice(device Device) error {
	if err := device.Validate(); err != nil {
		return err
	}

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.devices[device.GetID()] = device

	// Save to repository
	if dm.repository != nil {
		if err := dm.repository.SaveDevice(device); err != nil {
			dm.logger.WithError(err).WithField("device_id", device.GetID()).
				Error("Failed to save device to repository")
		}
	}

	dm.logger.WithField("device_id", device.GetID()).
		WithField("device_type", device.GetType()).
		Info("Device registered")

	return nil
}

// UnregisterDevice removes a device from management
func (dm *DeviceManager) UnregisterDevice(deviceID string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if _, exists := dm.devices[deviceID]; !exists {
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, deviceID)
	}

	delete(dm.devices, deviceID)

	// Remove from repository
	if dm.repository != nil {
		if err := dm.repository.DeleteDevice(deviceID); err != nil {
			dm.logger.WithError(err).WithField("device_id", deviceID).
				Error("Failed to delete device from repository")
		}
	}

	dm.logger.WithField("device_id", deviceID).Info("Device unregistered")
	return nil
}

// Subscribe adds an event callback
func (dm *DeviceManager) Subscribe(callback func(DeviceEvent)) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.eventCallbacks = append(dm.eventCallbacks, callback)
}

// Start begins the device manager's background operations
func (dm *DeviceManager) Start() {
	go dm.healthCheckLoop()
	dm.logger.Info("Device manager started")
}

// Stop stops the device manager and disconnects all adapters
func (dm *DeviceManager) Stop() {
	dm.cancel()
	dm.healthTicker.Stop()

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for _, adapter := range dm.adapters {
		adapter.Unsubscribe()
		adapter.Disconnect()
	}

	dm.logger.Info("Device manager stopped")
}

// GetAdapterStatus returns the status of all adapters
func (dm *DeviceManager) GetAdapterStatus() map[string]bool {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	status := make(map[string]bool)
	for adapterType, adapter := range dm.adapters {
		status[adapterType] = adapter.IsConnected()
	}

	return status
}

// handleDeviceEvent processes events from device adapters
func (dm *DeviceManager) handleDeviceEvent(event DeviceEvent) {
	dm.logger.WithFields(logrus.Fields{
		"device_id":    event.DeviceID,
		"adapter_type": event.AdapterType,
		"event_type":   event.EventType,
	}).Debug("Received device event")

	// Update device state if it's a state change event
	if event.EventType == EventTypeStateChanged {
		if device, err := dm.GetDevice(event.DeviceID); err == nil {
			if state, ok := event.Data["state"].(map[string]interface{}); ok {
				for key, value := range state {
					device.SetState(key, value)
				}
			}
		}
	}

	// Save event to repository
	if dm.repository != nil {
		if err := dm.repository.SaveDeviceEvent(event); err != nil {
			dm.logger.WithError(err).Error("Failed to save device event")
		}
	}

	// Forward to subscribers
	dm.emitEvent(event)
}

// emitEvent sends an event to all subscribers
func (dm *DeviceManager) emitEvent(event DeviceEvent) {
	dm.mutex.RLock()
	callbacks := make([]func(DeviceEvent), len(dm.eventCallbacks))
	copy(callbacks, dm.eventCallbacks)
	dm.mutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb func(DeviceEvent)) {
			defer func() {
				if r := recover(); r != nil {
					dm.logger.WithField("panic", r).Error("Event callback panicked")
				}
			}()
			cb(event)
		}(callback)
	}
}

// healthCheckLoop performs periodic health checks on adapters
func (dm *DeviceManager) healthCheckLoop() {
	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-dm.healthTicker.C:
			dm.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all adapters
func (dm *DeviceManager) performHealthCheck() {
	dm.mutex.RLock()
	adapters := make([]DeviceAdapter, 0, len(dm.adapters))
	for _, adapter := range dm.adapters {
		adapters = append(adapters, adapter)
	}
	dm.mutex.RUnlock()

	for _, adapter := range adapters {
		if err := adapter.HealthCheck(); err != nil {
			dm.logger.WithError(err).
				WithField("adapter_type", adapter.GetType()).
				Error("Adapter health check failed")

			// Emit health check failure event
			dm.emitEvent(DeviceEvent{
				AdapterType: adapter.GetType(),
				EventType:   "health_check_failed",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
				Timestamp: time.Now(),
				Source:    "health_check",
			})
		}
	}
}
