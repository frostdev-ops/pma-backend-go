package devices

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager handles all device adapters and provides a unified interface
type Manager struct {
	adapters      map[string]DeviceAdapter
	devices       map[string]Device
	repository    Repository
	eventHandlers []func(DeviceEvent)
	logger        *logrus.Logger
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new device manager
func NewManager(repository Repository, logger *logrus.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		adapters:      make(map[string]DeviceAdapter),
		devices:       make(map[string]Device),
		repository:    repository,
		eventHandlers: make([]func(DeviceEvent), 0),
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// RegisterAdapter registers a new device adapter
func (m *Manager) RegisterAdapter(name string, adapter DeviceAdapter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.adapters[name]; exists {
		return fmt.Errorf("adapter %s already registered", name)
	}

	m.adapters[name] = adapter
	m.logger.WithField("adapter", name).Info("Registered device adapter")

	// Subscribe to adapter events
	adapter.Subscribe(m.handleAdapterEvent)

	return nil
}

// ConnectAdapter connects a specific adapter
func (m *Manager) ConnectAdapter(name string) error {
	m.mu.RLock()
	adapter, exists := m.adapters[name]
	m.mu.RUnlock()

	if !exists {
		return NewAdapterError(name, "connect", ErrAdapterNotFound)
	}

	if err := adapter.Connect(); err != nil {
		return NewAdapterError(name, "connect", err)
	}

	// Discover devices after connection
	devices, err := adapter.Discover()
	if err != nil {
		m.logger.WithError(err).WithField("adapter", name).Warn("Failed to discover devices")
		return nil // Don't fail connection if discovery fails
	}

	// Register discovered devices
	m.mu.Lock()
	for _, device := range devices {
		m.devices[device.GetID()] = device
		// Save to repository
		if err := m.repository.CreateDevice(device); err != nil {
			m.logger.WithError(err).WithField("device_id", device.GetID()).Warn("Failed to save device")
		}
	}
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"adapter": name,
		"devices": len(devices),
	}).Info("Adapter connected and devices discovered")

	return nil
}

// DisconnectAdapter disconnects a specific adapter
func (m *Manager) DisconnectAdapter(name string) error {
	m.mu.RLock()
	adapter, exists := m.adapters[name]
	m.mu.RUnlock()

	if !exists {
		return NewAdapterError(name, "disconnect", ErrAdapterNotFound)
	}

	return adapter.Disconnect()
}

// ConnectAll connects all registered adapters
func (m *Manager) ConnectAll() error {
	m.mu.RLock()
	adapterNames := make([]string, 0, len(m.adapters))
	for name := range m.adapters {
		adapterNames = append(adapterNames, name)
	}
	m.mu.RUnlock()

	var connectErrors []error
	for _, name := range adapterNames {
		if err := m.ConnectAdapter(name); err != nil {
			connectErrors = append(connectErrors, err)
		}
	}

	if len(connectErrors) > 0 {
		return fmt.Errorf("failed to connect %d adapters", len(connectErrors))
	}

	return nil
}

// DisconnectAll disconnects all adapters
func (m *Manager) DisconnectAll() error {
	m.mu.RLock()
	adapterNames := make([]string, 0, len(m.adapters))
	for name := range m.adapters {
		adapterNames = append(adapterNames, name)
	}
	m.mu.RUnlock()

	var disconnectErrors []error
	for _, name := range adapterNames {
		if err := m.DisconnectAdapter(name); err != nil {
			disconnectErrors = append(disconnectErrors, err)
		}
	}

	if len(disconnectErrors) > 0 {
		return fmt.Errorf("failed to disconnect %d adapters", len(disconnectErrors))
	}

	return nil
}

// DiscoverDevices discovers devices from all connected adapters
func (m *Manager) DiscoverDevices() ([]Device, error) {
	m.mu.RLock()
	adaptersCopy := make(map[string]DeviceAdapter)
	for name, adapter := range m.adapters {
		adaptersCopy[name] = adapter
	}
	m.mu.RUnlock()

	allDevices := make([]Device, 0)
	for name, adapter := range adaptersCopy {
		devices, err := adapter.Discover()
		if err != nil {
			m.logger.WithError(err).WithField("adapter", name).Warn("Discovery failed")
			continue
		}
		allDevices = append(allDevices, devices...)
	}

	// Update internal device map
	m.mu.Lock()
	for _, device := range allDevices {
		m.devices[device.GetID()] = device
		// Save to repository
		if err := m.repository.CreateOrUpdateDevice(device); err != nil {
			m.logger.WithError(err).WithField("device_id", device.GetID()).Warn("Failed to save device")
		}
	}
	m.mu.Unlock()

	return allDevices, nil
}

// GetDevice retrieves a device by ID
func (m *Manager) GetDevice(id string) (Device, error) {
	m.mu.RLock()
	device, exists := m.devices[id]
	m.mu.RUnlock()

	if !exists {
		// Try to load from repository
		device, err := m.repository.GetDevice(id)
		if err != nil {
			return nil, ErrDeviceNotFound
		}

		// Cache the device
		m.mu.Lock()
		m.devices[id] = device
		m.mu.Unlock()

		return device, nil
	}

	return device, nil
}

// GetAllDevices returns all devices
func (m *Manager) GetAllDevices() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]Device, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, device)
	}

	return devices
}

// GetDevicesByType returns devices of a specific type
func (m *Manager) GetDevicesByType(deviceType DeviceType) []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]Device, 0)
	for _, device := range m.devices {
		if device.GetType() == deviceType {
			devices = append(devices, device)
		}
	}

	return devices
}

// UpdateDeviceState updates a device state
func (m *Manager) UpdateDeviceState(id string, key string, value interface{}) error {
	device, err := m.GetDevice(id)
	if err != nil {
		return err
	}

	if err := device.SetState(key, value); err != nil {
		return NewDeviceError(id, device.GetType(), "set_state", err)
	}

	// Save state to repository
	if err := m.repository.SaveDeviceState(id, device.GetState()); err != nil {
		m.logger.WithError(err).WithField("device_id", id).Warn("Failed to save device state")
	}

	// Emit state change event
	m.emitEvent(DeviceEvent{
		DeviceID:  id,
		EventType: EventTypeStateChange,
		Data: map[string]interface{}{
			"key":   key,
			"value": value,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// ExecuteCommand executes a command on a device
func (m *Manager) ExecuteCommand(id string, command string, params map[string]interface{}) (interface{}, error) {
	device, err := m.GetDevice(id)
	if err != nil {
		return nil, err
	}

	// Check if device is online
	if device.GetStatus() != DeviceStatusOnline {
		return nil, NewDeviceError(id, device.GetType(), "execute", ErrDeviceOffline)
	}

	result, err := device.Execute(command, params)
	if err != nil {
		return nil, NewDeviceError(id, device.GetType(), "execute", err)
	}

	return result, nil
}

// RemoveDevice removes a device
func (m *Manager) RemoveDevice(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.devices, id)

	if err := m.repository.DeleteDevice(id); err != nil {
		return err
	}

	m.logger.WithField("device_id", id).Info("Device removed")

	return nil
}

// GetAdapterStatus returns the status of all adapters
func (m *Manager) GetAdapterStatus() map[string]AdapterStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]AdapterStatus)
	for name, adapter := range m.adapters {
		status[name] = adapter.GetStatus()
	}

	return status
}

// Subscribe adds an event handler
func (m *Manager) Subscribe(handler func(DeviceEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventHandlers = append(m.eventHandlers, handler)
}

// Start starts the device manager
func (m *Manager) Start() error {
	// Load devices from repository
	devices, err := m.repository.GetAllDevices()
	if err != nil {
		return err
	}

	m.mu.Lock()
	for _, device := range devices {
		m.devices[device.GetID()] = device
	}
	m.mu.Unlock()

	m.logger.WithField("devices", len(devices)).Info("Device manager started")

	// Start health check routine
	go m.healthCheckLoop()

	return nil
}

// Stop stops the device manager
func (m *Manager) Stop() error {
	m.cancel()
	return m.DisconnectAll()
}

// handleAdapterEvent handles events from adapters
func (m *Manager) handleAdapterEvent(event DeviceEvent) {
	// Update device status if needed
	if event.EventType == EventTypeConnected || event.EventType == EventTypeDisconnected {
		if device, err := m.GetDevice(event.DeviceID); err == nil {
			if baseDevice, ok := device.(*BaseDevice); ok {
				if event.EventType == EventTypeConnected {
					baseDevice.SetStatus(DeviceStatusOnline)
				} else {
					baseDevice.SetStatus(DeviceStatusOffline)
				}
			}
		}
	}

	// Save event to repository
	if err := m.repository.SaveDeviceEvent(event); err != nil {
		m.logger.WithError(err).WithField("device_id", event.DeviceID).Warn("Failed to save device event")
	}

	// Forward event to handlers
	m.emitEvent(event)
}

// emitEvent sends an event to all handlers
func (m *Manager) emitEvent(event DeviceEvent) {
	m.mu.RLock()
	handlers := make([]func(DeviceEvent), len(m.eventHandlers))
	copy(handlers, m.eventHandlers)
	m.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// healthCheckLoop periodically checks device health
func (m *Manager) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all devices
func (m *Manager) performHealthCheck() {
	m.mu.RLock()
	adaptersCopy := make(map[string]DeviceAdapter)
	for name, adapter := range m.adapters {
		adaptersCopy[name] = adapter
	}
	m.mu.RUnlock()

	for name, adapter := range adaptersCopy {
		status := adapter.GetStatus()
		if !status.Connected {
			m.logger.WithField("adapter", name).Warn("Adapter disconnected, attempting reconnection")
			if err := m.ConnectAdapter(name); err != nil {
				m.logger.WithError(err).WithField("adapter", name).Error("Failed to reconnect adapter")
			}
		}
	}
}
