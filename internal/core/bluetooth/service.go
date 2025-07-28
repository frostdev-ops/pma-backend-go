package bluetooth

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service represents the Bluetooth service
type Service struct {
	repo              repositories.BluetoothRepository
	log               *logrus.Logger
	mutex             sync.RWMutex
	scanSession       *ScanSession
	pairingSessions   map[string]*PairingSession
	discoveredDevices map[string]*Device
	connectionRetries map[string]int
	eventListeners    []EventListener

	// Configuration
	maxConnectionRetries int
	retryDelayMS         int
	pairingTimeoutMS     int
	maxScanDuration      int
}

// EventListener defines the interface for event listeners
type EventListener interface {
	OnDeviceDiscovered(device *Device)
	OnDeviceConnected(device *Device)
	OnDeviceDisconnected(device *Device)
	OnPairingStarted(session *PairingSession)
	OnPairingCompleted(device *Device)
	OnPairingFailed(session *PairingSession, err error)
	OnScanStarted()
	OnScanStopped()
}

// NewService creates a new Bluetooth service
func NewService(repo repositories.BluetoothRepository, logger *logrus.Logger) *Service {
	return &Service{
		repo:                 repo,
		log:                  logger,
		pairingSessions:      make(map[string]*PairingSession),
		discoveredDevices:    make(map[string]*Device),
		connectionRetries:    make(map[string]int),
		eventListeners:       []EventListener{},
		maxConnectionRetries: 3,
		retryDelayMS:         2000,
		pairingTimeoutMS:     60000,
		maxScanDuration:      60,
	}
}

// AddEventListener adds an event listener
func (s *Service) AddEventListener(listener EventListener) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.eventListeners = append(s.eventListeners, listener)
}

// CheckAvailability checks if Bluetooth is available on the system
func (s *Service) CheckAvailability(ctx context.Context) (*BluetoothAvailability, error) {
	availability := &BluetoothAvailability{
		Available: false,
	}

	// Check platform
	if runtime.GOOS != "linux" {
		availability.Error = "Bluetooth is only supported on Linux systems"
		return availability, nil
	}

	// Check if bluetoothctl is available
	if _, err := exec.LookPath("bluetoothctl"); err != nil {
		availability.Error = "bluetoothctl not found. Install bluez: sudo apt-get install bluez"
		return availability, nil
	}

	// Check if bluetooth service is running with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "systemctl", "is-active", "bluetooth")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		availability.ServiceActive = true
	}

	// Check for Bluetooth adapter with timeout
	cmdCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()

	cmd = exec.CommandContext(cmdCtx2, "bluetoothctl", "list")
	output, err = cmd.Output()
	if err != nil {
		availability.Error = fmt.Sprintf("Failed to list Bluetooth adapters: %v", err)
		return availability, nil
	}

	// Parse adapter info
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Controller") {
			availability.AdapterFound = true
			break
		}
	}

	if availability.AdapterFound {
		availability.Available = true
	} else {
		availability.Error = "No Bluetooth adapter found"
	}

	// Get BlueZ version
	cmd = exec.CommandContext(ctx, "bluetoothctl", "--version")
	if output, err := cmd.Output(); err == nil {
		availability.BluezVersion = strings.TrimSpace(string(output))
	}

	return availability, nil
}

// GetAdapterInfo retrieves information about the Bluetooth adapter
func (s *Service) GetAdapterInfo(ctx context.Context) (*BluetoothAdapter, error) {
	// Check availability first
	availability, err := s.CheckAvailability(ctx)
	if err != nil {
		return nil, err
	}

	if !availability.Available {
		return nil, fmt.Errorf("bluetooth not available: %s", availability.Error)
	}

	// Get adapter info with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bluetoothctl", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get adapter info: %v", err)
	}

	// Parse adapter information
	adapter := &BluetoothAdapter{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name:") {
			adapter.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "Alias:") {
			adapter.Alias = strings.TrimSpace(strings.TrimPrefix(line, "Alias:"))
		} else if strings.HasPrefix(line, "Class:") {
			adapter.Class = strings.TrimSpace(strings.TrimPrefix(line, "Class:"))
		} else if strings.HasPrefix(line, "Powered:") {
			powered := strings.TrimSpace(strings.TrimPrefix(line, "Powered:"))
			adapter.Powered = powered == "yes"
		} else if strings.HasPrefix(line, "Discoverable:") {
			discoverable := strings.TrimSpace(strings.TrimPrefix(line, "Discoverable:"))
			adapter.Discoverable = discoverable == "yes"
		} else if strings.HasPrefix(line, "Pairable:") {
			pairable := strings.TrimSpace(strings.TrimPrefix(line, "Pairable:"))
			adapter.Pairable = pairable == "yes"
		}
	}

	return adapter, nil
}

// SetPower turns the Bluetooth adapter on or off
func (s *Service) SetPower(ctx context.Context, powered bool) error {
	var cmd *exec.Cmd

	// Use timeout context
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if powered {
		cmd = exec.CommandContext(cmdCtx, "bluetoothctl", "power", "on")
	} else {
		cmd = exec.CommandContext(cmdCtx, "bluetoothctl", "power", "off")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bluetooth power: %w", err)
	}

	return nil
}

// SetDiscoverable makes the adapter discoverable or non-discoverable
func (s *Service) SetDiscoverable(ctx context.Context, discoverable bool, timeout int) error {
	var cmd *exec.Cmd
	if discoverable {
		if timeout > 0 {
			cmd = exec.CommandContext(ctx, "bluetoothctl", "discoverable", "on", strconv.Itoa(timeout))
		} else {
			cmd = exec.CommandContext(ctx, "bluetoothctl", "discoverable", "on")
		}
	} else {
		cmd = exec.CommandContext(ctx, "bluetoothctl", "discoverable", "off")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set discoverable state: %w", err)
	}

	s.log.Infof("Bluetooth adapter discoverable set to: %t", discoverable)
	return nil
}

// ScanForDevices scans for nearby Bluetooth devices
func (s *Service) ScanForDevices(ctx context.Context, duration int) ([]*Device, error) {
	s.mutex.Lock()
	if s.scanSession != nil && s.scanSession.Active {
		s.mutex.Unlock()
		return nil, fmt.Errorf("device scan already in progress")
	}

	// Validate duration
	if duration < 1 || duration > s.maxScanDuration {
		s.mutex.Unlock()
		return nil, fmt.Errorf("duration must be between 1 and %d seconds", s.maxScanDuration)
	}

	// Initialize scan session
	s.scanSession = &ScanSession{
		Active:            true,
		StartTime:         time.Now(),
		Duration:          duration,
		Status:            ScanStatusActive,
		DiscoveredDevices: make(map[string]*Device),
	}
	s.discoveredDevices = make(map[string]*Device)
	s.mutex.Unlock()

	// Notify listeners
	for _, listener := range s.eventListeners {
		listener.OnScanStarted()
	}

	s.log.Infof("Starting Bluetooth device scan for %d seconds", duration)

	// Ensure adapter is powered on
	if err := s.SetPower(ctx, true); err != nil {
		s.stopScan(ctx)
		return nil, fmt.Errorf("failed to power on adapter: %w", err)
	}

	// Start scanning
	cmd := exec.CommandContext(ctx, "bluetoothctl", "scan", "on")
	if err := cmd.Run(); err != nil {
		s.stopScan(ctx)
		return nil, fmt.Errorf("failed to start scan: %w", err)
	}

	// Create a context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, time.Duration(duration)*time.Second)
	defer cancel()

	// Collect devices periodically during scan
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-scanCtx.Done():
				return
			case <-ticker.C:
				if err := s.collectDiscoveredDevices(ctx); err != nil {
					s.log.Warnf("Failed to collect devices during scan: %v", err)
				}
			}
		}
	}()

	// Wait for scan to complete
	<-scanCtx.Done()

	// Final collection
	if err := s.collectDiscoveredDevices(ctx); err != nil {
		s.log.Warnf("Failed to collect devices after scan: %v", err)
	}

	// Stop scanning
	s.stopScan(ctx)

	// Convert discovered devices to slice
	devices := make([]*Device, 0, len(s.discoveredDevices))
	s.mutex.RLock()
	for _, device := range s.discoveredDevices {
		devices = append(devices, device)
	}
	s.mutex.RUnlock()

	s.log.Infof("Bluetooth scan completed. Found %d devices", len(devices))
	return devices, nil
}

// stopScan stops the current scan
func (s *Service) stopScan(ctx context.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.scanSession != nil {
		s.scanSession.Active = false
		s.scanSession.Status = ScanStatusStopped
	}

	// Stop bluetoothctl scan
	cmd := exec.CommandContext(ctx, "bluetoothctl", "scan", "off")
	cmd.Run() // Ignore error

	// Notify listeners
	for _, listener := range s.eventListeners {
		listener.OnScanStopped()
	}
}

// collectDiscoveredDevices collects devices found during scan
func (s *Service) collectDiscoveredDevices(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "bluetoothctl", "devices")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	deviceRegex := regexp.MustCompile(`Device ([0-9A-F:]{17}) (.+)`)

	for _, line := range lines {
		matches := deviceRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			address := matches[1]
			name := matches[2]

			// Get detailed device info
			if device, err := s.GetDeviceInfo(ctx, address); err == nil {
				if device.Name == "" {
					device.Name = name
				}

				s.mutex.Lock()
				s.discoveredDevices[address] = device
				s.mutex.Unlock()

				// Notify listeners
				for _, listener := range s.eventListeners {
					listener.OnDeviceDiscovered(device)
				}
			}
		}
	}

	return nil
}

// GetDeviceInfo retrieves detailed information about a specific device
func (s *Service) GetDeviceInfo(ctx context.Context, address string) (*Device, error) {
	cmd := exec.CommandContext(ctx, "bluetoothctl", "info", address)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	device := &Device{
		Address:   address,
		Name:      "",
		Connected: false,
		Paired:    false,
		Trusted:   false,
		Blocked:   false,
		Services:  []string{},
		LastSeen:  time.Now(),
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Name: ") {
			device.Name = strings.TrimPrefix(line, "Name: ")
		} else if strings.HasPrefix(line, "Alias: ") {
			device.Alias = strings.TrimPrefix(line, "Alias: ")
		} else if strings.HasPrefix(line, "Connected: ") {
			device.Connected = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "Paired: ") {
			device.Paired = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "Trusted: ") {
			device.Trusted = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "Blocked: ") {
			device.Blocked = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "RSSI: ") {
			if rssiStr := strings.TrimPrefix(line, "RSSI: "); rssiStr != "" {
				if rssi, err := strconv.Atoi(rssiStr); err == nil {
					device.RSSI = &rssi
				}
			}
		} else if strings.Contains(line, "UUID: ") {
			// Extract service UUID and name
			if idx := strings.Index(line, "UUID: "); idx != -1 {
				uuidPart := line[idx+6:]
				if parenIdx := strings.Index(uuidPart, "("); parenIdx != -1 {
					serviceName := strings.TrimSpace(uuidPart[parenIdx+1:])
					serviceName = strings.TrimSuffix(serviceName, ")")
					device.Services = append(device.Services, serviceName)
				}
			}
		}
	}

	// Determine device type based on services
	device.DeviceType = determineDeviceType(device.Services)

	return device, nil
}

// PairDevice pairs with a Bluetooth device
func (s *Service) PairDevice(ctx context.Context, request *PairRequest) (*PairResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if already pairing
	for _, session := range s.pairingSessions {
		if session.Status == PairingStatusPending {
			return &PairResponse{
				Success: false,
				Message: "Another pairing operation is already in progress",
			}, nil
		}
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Get device info first
	device, err := s.GetDeviceInfo(ctx, request.Address)
	if err != nil {
		return &PairResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get device info: %v", err),
		}, nil
	}

	// Check if already paired
	if device.Paired {
		return &PairResponse{
			Success: false,
			Message: "Device is already paired",
		}, nil
	}

	// Create pairing session
	session := &PairingSession{
		SessionID:       sessionID,
		DeviceAddress:   request.Address,
		DeviceName:      device.Name,
		Method:          request.Method,
		Status:          PairingStatusPending,
		PIN:             request.PIN,
		StartTime:       time.Now(),
		AgentRegistered: true,
	}

	if request.Method == "" {
		session.Method = PairingMethodSSP
	}

	s.pairingSessions[sessionID] = session

	// Notify listeners
	for _, listener := range s.eventListeners {
		listener.OnPairingStarted(session)
	}

	s.log.Infof("Starting pairing with device %s (%s)", device.Name, request.Address)

	// Ensure adapter is powered on
	if err := s.SetPower(ctx, true); err != nil {
		session.Status = PairingStatusFailed
		session.ErrorMessage = fmt.Sprintf("Failed to power on adapter: %v", err)
		return &PairResponse{
			Success: false,
			Message: session.ErrorMessage,
		}, nil
	}

	// Remove device if already exists (for clean pairing)
	removeCmd := exec.CommandContext(ctx, "bluetoothctl", "remove", request.Address)
	removeCmd.Run() // Ignore error

	// Start pairing
	pairCmd := exec.CommandContext(ctx, "bluetoothctl", "pair", request.Address)
	if err := pairCmd.Run(); err != nil {
		session.Status = PairingStatusFailed
		session.ErrorMessage = fmt.Sprintf("Pairing failed: %v", err)

		for _, listener := range s.eventListeners {
			listener.OnPairingFailed(session, err)
		}

		return &PairResponse{
			Success: false,
			Message: session.ErrorMessage,
		}, nil
	}

	// Wait a moment for pairing to complete
	time.Sleep(2 * time.Second)

	// Check if pairing was successful
	updatedDevice, err := s.GetDeviceInfo(ctx, request.Address)
	if err != nil || !updatedDevice.Paired {
		session.Status = PairingStatusFailed
		session.ErrorMessage = "Pairing verification failed"
		return &PairResponse{
			Success: false,
			Message: session.ErrorMessage,
		}, nil
	}

	// Trust the device for auto-connection
	trustCmd := exec.CommandContext(ctx, "bluetoothctl", "trust", request.Address)
	trustCmd.Run() // Ignore error

	// Update session
	now := time.Now()
	session.Status = PairingStatusSuccess
	session.EndTime = &now
	updatedDevice.Trusted = true
	updatedDevice.PairedAt = &now

	// Save device to database
	if err := s.repo.CreateDevice(ctx, updatedDevice.ToModel()); err != nil {
		s.log.Warnf("Failed to save paired device to database: %v", err)
	}

	// Notify listeners
	for _, listener := range s.eventListeners {
		listener.OnPairingCompleted(updatedDevice)
	}

	s.log.Infof("Successfully paired with device %s (%s)", updatedDevice.Name, request.Address)

	return &PairResponse{
		Success:   true,
		SessionID: sessionID,
		Message:   fmt.Sprintf("Successfully paired with device %s", updatedDevice.Name),
	}, nil
}

// ConnectDevice connects to a paired device
func (s *Service) ConnectDevice(ctx context.Context, request *ConnectRequest) error {
	s.log.Infof("Attempting to connect to device: %s", request.Address)

	// Check if device is paired
	device, err := s.GetDeviceInfo(ctx, request.Address)
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	if !device.Paired {
		return fmt.Errorf("device must be paired before connecting")
	}

	// Trust device if requested
	if request.Trust {
		trustCmd := exec.CommandContext(ctx, "bluetoothctl", "trust", request.Address)
		trustCmd.Run() // Ignore error
	}

	// Connect to device
	connectCmd := exec.CommandContext(ctx, "bluetoothctl", "connect", request.Address)
	if err := connectCmd.Run(); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	// Wait for connection to establish
	time.Sleep(2 * time.Second)

	// Verify connection
	updatedDevice, err := s.GetDeviceInfo(ctx, request.Address)
	if err != nil {
		return fmt.Errorf("failed to verify connection: %w", err)
	}

	if !updatedDevice.Connected {
		return fmt.Errorf("connection verification failed")
	}

	// Update device in database
	if err := s.repo.UpdateDevice(ctx, updatedDevice.ToModel()); err != nil {
		s.log.Warnf("Failed to update connected device in database: %v", err)
	}

	// Notify listeners
	for _, listener := range s.eventListeners {
		listener.OnDeviceConnected(updatedDevice)
	}

	s.log.Infof("Successfully connected to device %s (%s)", updatedDevice.Name, request.Address)
	return nil
}

// DisconnectDevice disconnects from a connected device
func (s *Service) DisconnectDevice(ctx context.Context, address string) error {
	s.log.Infof("Disconnecting from device: %s", address)

	cmd := exec.CommandContext(ctx, "bluetoothctl", "disconnect", address)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disconnect from device: %w", err)
	}

	// Wait for disconnection
	time.Sleep(1 * time.Second)

	// Update device status
	if device, err := s.GetDeviceInfo(ctx, address); err == nil {
		// Update device in database
		if err := s.repo.UpdateDevice(ctx, device.ToModel()); err != nil {
			s.log.Warnf("Failed to update disconnected device in database: %v", err)
		}

		// Notify listeners
		for _, listener := range s.eventListeners {
			listener.OnDeviceDisconnected(device)
		}
	}

	s.log.Infof("Successfully disconnected from device: %s", address)
	return nil
}

// RemoveDevice removes/unpairs a device
func (s *Service) RemoveDevice(ctx context.Context, address string) error {
	s.log.Infof("Removing device: %s", address)

	cmd := exec.CommandContext(ctx, "bluetoothctl", "remove", address)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove device: %w", err)
	}

	// Remove from database
	if err := s.repo.DeleteDevice(ctx, address); err != nil {
		s.log.Warnf("Failed to remove device from database: %v", err)
	}

	s.log.Infof("Successfully removed device: %s", address)
	return nil
}

// GetPairedDevices retrieves all paired devices
func (s *Service) GetPairedDevices(ctx context.Context) ([]*Device, error) {
	cmd := exec.CommandContext(ctx, "bluetoothctl", "paired-devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get paired devices: %w", err)
	}

	devices := []*Device{}
	lines := strings.Split(string(output), "\n")
	deviceRegex := regexp.MustCompile(`Device ([0-9A-F:]{17}) (.+)`)

	for _, line := range lines {
		matches := deviceRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			address := matches[1]
			if device, err := s.GetDeviceInfo(ctx, address); err == nil {
				devices = append(devices, device)
			}
		}
	}

	// Sort by connection status and name
	// Connected devices first, then by name
	for i := 0; i < len(devices); i++ {
		for j := i + 1; j < len(devices); j++ {
			if (!devices[i].Connected && devices[j].Connected) ||
				(devices[i].Connected == devices[j].Connected && devices[i].Name > devices[j].Name) {
				devices[i], devices[j] = devices[j], devices[i]
			}
		}
	}

	return devices, nil
}

// GetAllDevices retrieves all devices from database
func (s *Service) GetAllDevices(ctx context.Context) ([]*Device, error) {
	models, err := s.repo.GetAllDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices from database: %w", err)
	}

	devices := make([]*Device, len(models))
	for i, model := range models {
		devices[i] = DeviceFromModel(model)
	}

	return devices, nil
}

// GetConnectedDevices retrieves all connected devices
func (s *Service) GetConnectedDevices(ctx context.Context) ([]*Device, error) {
	models, err := s.repo.GetConnectedDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected devices: %w", err)
	}

	devices := make([]*Device, len(models))
	for i, model := range models {
		devices[i] = DeviceFromModel(model)
	}

	return devices, nil
}

// GetDeviceStats returns statistics about Bluetooth devices
func (s *Service) GetDeviceStats(ctx context.Context) (*DeviceStats, error) {
	allDevices, err := s.repo.GetAllDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device stats: %w", err)
	}

	pairedDevices, err := s.repo.GetPairedDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get paired devices: %w", err)
	}

	connectedDevices, err := s.repo.GetConnectedDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected devices: %w", err)
	}

	// Count recent devices (seen in last 24h)
	recentCount := 0
	yesterday := time.Now().Add(-24 * time.Hour)
	for _, device := range allDevices {
		if device.LastSeen.After(yesterday) {
			recentCount++
		}
	}

	return &DeviceStats{
		TotalDevices:     len(allDevices),
		PairedDevices:    len(pairedDevices),
		ConnectedDevices: len(connectedDevices),
		RecentDevices:    recentCount,
	}, nil
}

// GetCapabilities returns the service capabilities
func (s *Service) GetCapabilities(ctx context.Context) (*ServiceCapabilities, error) {
	availability, err := s.CheckAvailability(ctx)
	if err != nil {
		return nil, err
	}

	capabilities := &ServiceCapabilities{
		Available:        availability.Available,
		ScanSupported:    availability.Available,
		PairSupported:    availability.Available,
		ConnectSupported: availability.Available,
		SupportedMethods: []string{"pin", "ssp", "passkey", "confirm"},
		MaxScanDuration:  s.maxScanDuration,
		Platform:         runtime.GOOS,
	}

	return capabilities, nil
}
