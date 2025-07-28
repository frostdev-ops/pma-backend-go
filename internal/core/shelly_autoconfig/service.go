package shelly_autoconfig

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/shelly"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/sirupsen/logrus"
)

func logMemStatsShelly(logger *logrus.Logger, context string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.WithFields(logrus.Fields{
		"context":        context,
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
	}).Info("[MEMSTATS][SHELLY] Memory usage snapshot")
}

// DiscoveryMethod represents how a device was discovered
type DiscoveryMethod string

const (
	DiscoveryMethodAP        DiscoveryMethod = "ap_broadcasting"
	DiscoveryMethodNetwork   DiscoveryMethod = "network_scan"
	DiscoveryMethodBluetooth DiscoveryMethod = "bluetooth"
	DiscoveryMethodMDNS      DiscoveryMethod = "mdns"
)

// ConfigurationFlow represents the configuration path chosen
type ConfigurationFlow string

const (
	ConfigurationFlowManual ConfigurationFlow = "manual"
	ConfigurationFlowAI     ConfigurationFlow = "ai_assisted"
)

// AutoConfigState represents the current state of auto-configuration
type AutoConfigState string

const (
	StateIdle        AutoConfigState = "idle"
	StateDiscovering AutoConfigState = "discovering"
	StateConnecting  AutoConfigState = "connecting"
	StateGathering   AutoConfigState = "gathering_info"
	StateConfirming  AutoConfigState = "confirming"
	StateConfiguring AutoConfigState = "configuring"
	StateCompleted   AutoConfigState = "completed"
	StateFailed      AutoConfigState = "failed"
)

// DiscoveredDevice represents a newly discovered Shelly device pending configuration
type DiscoveredDevice struct {
	ID              string                 `json:"id"`
	MAC             string                 `json:"mac"`
	Name            string                 `json:"name"`
	Model           string                 `json:"model"`
	Type            string                 `json:"type"`
	IP              string                 `json:"ip"`
	Port            int                    `json:"port"`
	Generation      int                    `json:"generation"`
	FirmwareVersion string                 `json:"firmware_version"`
	WiFiMode        string                 `json:"wifi_mode"` // "ap" or "sta"
	WiFiSSID        string                 `json:"wifi_ssid"`
	AuthEnabled     bool                   `json:"auth_enabled"`
	DiscoveryMethod DiscoveryMethod        `json:"discovery_method"`
	FirstSeen       time.Time              `json:"first_seen"`
	LastSeen        time.Time              `json:"last_seen"`
	IsConfigured    bool                   `json:"is_configured"`
	DeviceInfo      map[string]interface{} `json:"device_info"`
	Capabilities    []string               `json:"capabilities"`
}

// ConfigurationSession represents an active configuration session
type ConfigurationSession struct {
	ID             string            `json:"id"`
	DeviceID       string            `json:"device_id"`
	DeviceMAC      string            `json:"device_mac"`
	State          AutoConfigState   `json:"state"`
	Flow           ConfigurationFlow `json:"flow"`
	StartedAt      time.Time         `json:"started_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	Error          string            `json:"error,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	Steps          []ConfigStep      `json:"steps"`
	NetworkBackup  *NetworkBackup    `json:"network_backup,omitempty"`
}

// ConfigStep represents a step in the configuration process
type ConfigStep struct {
	Step        string                 `json:"step"`
	Status      string                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// NetworkBackup stores network configuration before connecting to Shelly AP
type NetworkBackup struct {
	OriginalInterface string    `json:"original_interface"`
	OriginalSSID      string    `json:"original_ssid"`
	BackupTime        time.Time `json:"backup_time"`
	RestoreRequired   bool      `json:"restore_required"`
}

// Service provides auto-configuration capabilities for Shelly devices
type Service struct {
	logger           *logrus.Logger
	shellyAdapter    *shelly.ShellyAdapter
	wsHub            *websocket.Hub
	aiService        AIService
	networkManager   NetworkManager
	mutex            sync.RWMutex
	activeSessions   map[string]*ConfigurationSession
	discoveredCache  map[string]*DiscoveredDevice
	discoveryRunning bool
	config           ServiceConfig
}

// ServiceConfig holds configuration for the auto-config service
type ServiceConfig struct {
	Enabled                  bool          `json:"enabled"`
	DiscoveryInterval        time.Duration `json:"discovery_interval"`
	SessionTimeout           time.Duration `json:"session_timeout"`
	MaxConcurrentSessions    int           `json:"max_concurrent_sessions"`
	WiFiConnectionTimeout    time.Duration `json:"wifi_connection_timeout"`
	RequireNetworkSafety     bool          `json:"require_network_safety"`
	AllowedDiscoveryMethods  []string      `json:"allowed_discovery_methods"`
	DefaultConfigurationFlow string        `json:"default_configuration_flow"`
	NotificationChannels     []string      `json:"notification_channels"`
}

// AIService interface for AI integration
type AIService interface {
	SendNotification(ctx context.Context, notification *types.AINotification) error
	StartConfigurationChat(ctx context.Context, deviceInfo *DiscoveredDevice) (string, error)
	ExecuteMCPTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error)
}

// NetworkManager interface for network operations
type NetworkManager interface {
	GetActiveInterfaces() ([]NetworkInterface, error)
	ConnectToWiFi(ctx context.Context, ssid, password string) error
	DisconnectFromWiFi(ctx context.Context, interface_ string) error
	BackupNetworkConfig() (*NetworkBackup, error)
	RestoreNetworkConfig(backup *NetworkBackup) error
	IsNetworkSafe() (bool, error)
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsActive  bool   `json:"is_active"`
	IPAddress string `json:"ip_address"`
	SSID      string `json:"ssid,omitempty"`
}

// NewService creates a new auto-configuration service
func NewService(
	logger *logrus.Logger,
	shellyAdapter *shelly.ShellyAdapter,
	wsHub *websocket.Hub,
	aiService AIService,
	networkManager NetworkManager,
	config ServiceConfig,
) *Service {
	// Set defaults
	if config.DiscoveryInterval == 0 {
		config.DiscoveryInterval = 30 * time.Second
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 30 * time.Minute
	}
	if config.MaxConcurrentSessions == 0 {
		config.MaxConcurrentSessions = 5
	}
	if config.WiFiConnectionTimeout == 0 {
		config.WiFiConnectionTimeout = 60 * time.Second
	}
	if len(config.AllowedDiscoveryMethods) == 0 {
		config.AllowedDiscoveryMethods = []string{"ap_broadcasting", "network_scan", "mdns"}
	}
	if config.DefaultConfigurationFlow == "" {
		config.DefaultConfigurationFlow = "ai_assisted"
	}

	return &Service{
		logger:          logger,
		shellyAdapter:   shellyAdapter,
		wsHub:           wsHub,
		aiService:       aiService,
		networkManager:  networkManager,
		activeSessions:  make(map[string]*ConfigurationSession),
		discoveredCache: make(map[string]*DiscoveredDevice),
		config:          config,
	}
}

// Start begins the auto-configuration service
func (s *Service) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Shelly auto-configuration service is disabled")
		return nil
	}

	s.logger.Info("Starting Shelly auto-configuration service...")

	// Start discovery monitoring
	go s.runDiscoveryMonitor(ctx)

	// Start session cleanup
	go s.runSessionCleanup(ctx)

	s.logger.Info("Shelly auto-configuration service started successfully")
	return nil
}

// Stop stops the auto-configuration service
func (s *Service) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("Stopping Shelly auto-configuration service...")

	// Cancel all active sessions
	for sessionID, session := range s.activeSessions {
		session.State = StateFailed
		session.Error = "Service shutdown"
		if session.CompletedAt == nil {
			now := time.Now()
			session.CompletedAt = &now
		}
		s.logger.Infof("Cancelled configuration session %s due to service shutdown", sessionID)
	}

	s.discoveryRunning = false
	s.logger.Info("Shelly auto-configuration service stopped")
	return nil
}

// runDiscoveryMonitor continuously monitors for new Shelly devices
func (s *Service) runDiscoveryMonitor(ctx context.Context) {
	s.discoveryRunning = true
	ticker := time.NewTicker(s.config.DiscoveryInterval)
	defer ticker.Stop()

	logMemStatsShelly(s.logger, "before_discovery_monitor_loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.discoveryRunning {
				logMemStatsShelly(s.logger, "before_performDiscoveryScan")
				s.logger.WithField("discoveredCache_size", len(s.discoveredCache)).Info("Shelly discoveredCache size before scan")
				s.performDiscoveryScan(ctx)
				logMemStatsShelly(s.logger, "after_performDiscoveryScan")
				s.logger.WithField("discoveredCache_size", len(s.discoveredCache)).Info("Shelly discoveredCache size after scan")
				if len(s.discoveredCache) > 0 {
					for _, dev := range s.discoveredCache {
						s.logger.WithField("sample_discovered_device", fmt.Sprintf("%#v", dev)).Info("Sample discovered device")
						break
					}
				}
			}
		}
	}
}

// performDiscoveryScan performs a discovery scan for new devices
func (s *Service) performDiscoveryScan(ctx context.Context) {
	s.logger.Debug("Performing Shelly device discovery scan...")

	// Get discovered devices from the Shelly adapter
	devices := s.shellyAdapter.GetClient().GetDiscoveredDevices()

	for _, device := range devices {
		// Check if this is a new unconfigured device
		if s.isNewUnconfiguredDevice(device) {
			discoveredDevice := s.convertToDiscoveredDevice(device)
			s.handleNewDevice(ctx, discoveredDevice)
		}
	}
}

// isNewUnconfiguredDevice checks if a device is new and unconfigured
func (s *Service) isNewUnconfiguredDevice(device *shelly.EnhancedShellyDevice) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if already in cache
	if _, exists := s.discoveredCache[device.MAC]; exists {
		return false
	}

	// Check if already configured (connected to our network and not in AP mode)
	if device.WiFiMode == "sta" && device.IsConfigured {
		return false
	}

	// Check if device is in AP mode (indicating it needs configuration)
	return device.WiFiMode == "ap" || !device.IsConfigured
}

// convertToDiscoveredDevice converts a Shelly device to our format
func (s *Service) convertToDiscoveredDevice(device *shelly.EnhancedShellyDevice) *DiscoveredDevice {
	var discoveryMethod DiscoveryMethod
	switch device.DiscoveryMethod {
	case 0: // MDNS
		discoveryMethod = DiscoveryMethodMDNS
	case 1: // Network scan
		discoveryMethod = DiscoveryMethodNetwork
	default:
		discoveryMethod = DiscoveryMethodAP
	}

	return &DiscoveredDevice{
		ID:              device.ID,
		MAC:             device.MAC,
		Name:            device.Name,
		Model:           device.Model,
		Type:            device.Type,
		IP:              device.IP,
		Port:            device.Port,
		Generation:      int(device.Generation),
		FirmwareVersion: device.FirmwareVersion,
		WiFiMode:        device.WiFiMode,
		WiFiSSID:        device.WiFiSSID,
		AuthEnabled:     device.AuthEnabled,
		DiscoveryMethod: discoveryMethod,
		FirstSeen:       device.FirstSeen,
		LastSeen:        device.LastSeen,
		IsConfigured:    device.IsConfigured,
		Capabilities:    device.Capabilities,
		DeviceInfo: map[string]interface{}{
			"info":     device.Info,
			"status":   device.Status,
			"settings": device.Settings,
		},
	}
}

// handleNewDevice handles a newly discovered device
func (s *Service) handleNewDevice(ctx context.Context, device *DiscoveredDevice) {
	s.logger.WithFields(logrus.Fields{
		"device_id":    device.ID,
		"device_mac":   device.MAC,
		"device_model": device.Model,
		"wifi_mode":    device.WiFiMode,
	}).Info("New unconfigured Shelly device discovered")

	// Add to cache
	s.mutex.Lock()
	s.discoveredCache[device.MAC] = device
	s.mutex.Unlock()

	// Send AI notification about new device discovery
	go s.sendNewDeviceNotification(ctx, device)

	// If auto-configuration is enabled, start configuration process
	if s.shouldAutoStartConfiguration(device) {
		go s.startAutoConfiguration(ctx, device)
	}
}

// shouldAutoStartConfiguration determines if auto-configuration should start automatically
func (s *Service) shouldAutoStartConfiguration(device *DiscoveredDevice) bool {
	// Check if device is in AP mode (needs configuration)
	if device.WiFiMode != "ap" {
		return false
	}

	// Check if we have capacity for more sessions
	s.mutex.RLock()
	sessionCount := len(s.activeSessions)
	s.mutex.RUnlock()

	return sessionCount < s.config.MaxConcurrentSessions
}

// sendNewDeviceNotification sends an AI notification about a new device
func (s *Service) sendNewDeviceNotification(ctx context.Context, device *DiscoveredDevice) {
	if s.aiService == nil {
		return
	}

	notification := &types.AINotification{
		Type:      types.NotificationTypeDeviceDiscovery,
		Title:     "New Shelly Device Discovered",
		Message:   fmt.Sprintf("Found new %s device (%s) that needs configuration", device.Model, device.Name),
		Priority:  types.NotificationPriorityMedium,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"device":          device,
			"requires_config": device.WiFiMode == "ap",
			"auto_config":     s.shouldAutoStartConfiguration(device),
		},
		Actions: []types.NotificationAction{
			{
				ID:    "configure_manual",
				Label: "Configure Manually",
				Type:  types.ActionTypeButton,
				Data:  map[string]interface{}{"device_mac": device.MAC, "flow": "manual"},
			},
			{
				ID:    "configure_ai",
				Label: "Configure with AI",
				Type:  types.ActionTypeButton,
				Data:  map[string]interface{}{"device_mac": device.MAC, "flow": "ai_assisted"},
			},
			{
				ID:    "ignore",
				Label: "Ignore",
				Type:  types.ActionTypeButton,
				Data:  map[string]interface{}{"device_mac": device.MAC},
			},
		},
	}

	if err := s.aiService.SendNotification(ctx, notification); err != nil {
		s.logger.WithError(err).Error("Failed to send new device notification")
	}
}

// startAutoConfiguration starts automatic configuration for a device
func (s *Service) startAutoConfiguration(ctx context.Context, device *DiscoveredDevice) {
	sessionID := fmt.Sprintf("auto_%s_%d", device.MAC, time.Now().Unix())

	session := &ConfigurationSession{
		ID:        sessionID,
		DeviceID:  device.ID,
		DeviceMAC: device.MAC,
		State:     StateDiscovering,
		Flow:      ConfigurationFlow(s.config.DefaultConfigurationFlow),
		StartedAt: time.Now(),
		Steps:     []ConfigStep{},
	}

	s.mutex.Lock()
	s.activeSessions[sessionID] = session
	s.mutex.Unlock()

	s.logger.WithField("session_id", sessionID).Info("Starting auto-configuration session")

	// Execute configuration flow
	if err := s.executeConfigurationFlow(ctx, session, device); err != nil {
		s.logger.WithError(err).WithField("session_id", sessionID).Error("Auto-configuration failed")
		session.State = StateFailed
		session.Error = err.Error()
		now := time.Now()
		session.CompletedAt = &now
	}
}

// executeConfigurationFlow executes the complete configuration flow
func (s *Service) executeConfigurationFlow(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	steps := []string{"connect", "gather_info", "confirm", "configure"}

	for _, stepName := range steps {
		step := ConfigStep{
			Step:      stepName,
			Status:    "started",
			StartedAt: time.Now(),
		}
		session.Steps = append(session.Steps, step)

		switch stepName {
		case "connect":
			session.State = StateConnecting
			err := s.connectToDevice(ctx, session, device)
			s.completeStep(session, len(session.Steps)-1, err)
			if err != nil {
				return fmt.Errorf("connection failed: %w", err)
			}

		case "gather_info":
			session.State = StateGathering
			err := s.gatherDeviceInfo(ctx, session, device)
			s.completeStep(session, len(session.Steps)-1, err)
			if err != nil {
				return fmt.Errorf("info gathering failed: %w", err)
			}

		case "confirm":
			session.State = StateConfirming
			err := s.confirmDevice(ctx, session, device)
			s.completeStep(session, len(session.Steps)-1, err)
			if err != nil {
				return fmt.Errorf("device confirmation failed: %w", err)
			}

		case "configure":
			session.State = StateConfiguring
			err := s.configureDevice(ctx, session, device)
			s.completeStep(session, len(session.Steps)-1, err)
			if err != nil {
				return fmt.Errorf("device configuration failed: %w", err)
			}
		}
	}

	session.State = StateCompleted
	now := time.Now()
	session.CompletedAt = &now

	s.logger.WithField("session_id", session.ID).Info("Auto-configuration completed successfully")
	return nil
}

// completeStep marks a configuration step as completed
func (s *Service) completeStep(session *ConfigurationSession, stepIndex int, err error) {
	if stepIndex >= len(session.Steps) {
		return
	}

	step := &session.Steps[stepIndex]
	now := time.Now()
	step.CompletedAt = &now

	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
	} else {
		step.Status = "completed"
	}
}

// connectToDevice establishes connection to the Shelly device
func (s *Service) connectToDevice(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	// If device is in AP mode, we need to connect to its WiFi
	if device.WiFiMode == "ap" {
		return s.connectToShellyAP(ctx, session, device)
	}

	// If device is already on network, verify connectivity
	return s.verifyNetworkConnectivity(ctx, device)
}

// connectToShellyAP connects to a Shelly device's access point
func (s *Service) connectToShellyAP(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	// Safety check: ensure we have other network connections
	if s.config.RequireNetworkSafety {
		safe, err := s.networkManager.IsNetworkSafe()
		if err != nil {
			return fmt.Errorf("network safety check failed: %w", err)
		}
		if !safe {
			return fmt.Errorf("network safety requirement not met - cannot connect to Shelly AP")
		}
	}

	// Backup current network configuration
	backup, err := s.networkManager.BackupNetworkConfig()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to backup network config, proceeding anyway")
	} else {
		session.NetworkBackup = backup
	}

	// Connect to Shelly AP
	shellySSID := device.WiFiSSID
	if shellySSID == "" {
		// Generate expected Shelly AP SSID
		shellySSID = fmt.Sprintf("shelly%s-%s", device.Model, device.MAC[len(device.MAC)-6:])
	}

	s.logger.WithField("ssid", shellySSID).Info("Connecting to Shelly AP...")

	ctx, cancel := context.WithTimeout(ctx, s.config.WiFiConnectionTimeout)
	defer cancel()

	if err := s.networkManager.ConnectToWiFi(ctx, shellySSID, ""); err != nil {
		return fmt.Errorf("failed to connect to Shelly AP: %w", err)
	}

	s.logger.Info("Successfully connected to Shelly AP")
	return nil
}

// verifyNetworkConnectivity verifies we can communicate with the device
func (s *Service) verifyNetworkConnectivity(ctx context.Context, device *DiscoveredDevice) error {
	// Try to connect to device IP
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", device.IP, device.Port), 5*time.Second)
	if err != nil {
		return fmt.Errorf("cannot connect to device at %s:%d: %w", device.IP, device.Port, err)
	}
	conn.Close()

	return nil
}

// gatherDeviceInfo gathers comprehensive information about the device
func (s *Service) gatherDeviceInfo(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	s.logger.WithField("device_id", device.ID).Info("Gathering device information...")

	// Get device info from Shelly client
	client := s.shellyAdapter.GetClient()

	// Get device info
	info, err := client.GetDeviceInfo(ctx, device.IP)
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}

	// Get device status
	status, err := client.GetDeviceStatus(ctx, device.IP)
	if err != nil {
		return fmt.Errorf("failed to get device status: %w", err)
	}

	// Get device settings
	settings, err := client.GetDeviceSettings(ctx, device.IP)
	if err != nil {
		return fmt.Errorf("failed to get device settings: %w", err)
	}

	// Update device info
	device.DeviceInfo = map[string]interface{}{
		"info":     info,
		"status":   status,
		"settings": settings,
	}

	s.logger.WithField("device_id", device.ID).Info("Device information gathered successfully")
	return nil
}

// confirmDevice sends device confirmation to user/AI
func (s *Service) confirmDevice(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	s.logger.WithField("device_id", device.ID).Info("Confirming device configuration...")

	// Send device confirmation notification
	if s.aiService != nil {
		notification := &types.AINotification{
			Type:      types.NotificationTypeDeviceConfirmation,
			Title:     "Shelly Device Ready for Configuration",
			Message:   fmt.Sprintf("Device %s (%s) is ready to be configured", device.Name, device.Model),
			Priority:  types.NotificationPriorityHigh,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"session":     session,
				"device":      device,
				"device_info": device.DeviceInfo,
			},
		}

		if err := s.aiService.SendNotification(ctx, notification); err != nil {
			s.logger.WithError(err).Error("Failed to send device confirmation notification")
		}
	}

	// For AI-assisted flow, start conversation
	if session.Flow == ConfigurationFlowAI && s.aiService != nil {
		conversationID, err := s.aiService.StartConfigurationChat(ctx, device)
		if err != nil {
			s.logger.WithError(err).Error("Failed to start AI configuration chat")
		} else {
			session.ConversationID = conversationID
		}
	}

	return nil
}

// configureDevice performs the actual device configuration
func (s *Service) configureDevice(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	s.logger.WithField("device_id", device.ID).Info("Configuring device...")

	switch session.Flow {
	case ConfigurationFlowManual:
		return s.configureDeviceManual(ctx, session, device)
	case ConfigurationFlowAI:
		return s.configureDeviceAI(ctx, session, device)
	default:
		return fmt.Errorf("unknown configuration flow: %s", session.Flow)
	}
}

// configureDeviceManual handles manual configuration flow
func (s *Service) configureDeviceManual(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	// For manual flow, we wait for user input via API endpoints
	// This is a placeholder - actual implementation would wait for user configuration
	s.logger.WithField("session_id", session.ID).Info("Manual configuration flow initiated - waiting for user input")

	// Send notification that manual configuration is ready
	if s.aiService != nil {
		notification := &types.AINotification{
			Type:      types.NotificationTypeConfigurationReady,
			Title:     "Manual Configuration Ready",
			Message:   fmt.Sprintf("Device %s is ready for manual configuration", device.Name),
			Priority:  types.NotificationPriorityHigh,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"session_id": session.ID,
				"device":     device,
				"config_url": fmt.Sprintf("/api/v1/devices/shelly/configure/%s", session.ID),
			},
		}
		s.aiService.SendNotification(ctx, notification)
	}

	return nil
}

// configureDeviceAI handles AI-assisted configuration flow
func (s *Service) configureDeviceAI(ctx context.Context, session *ConfigurationSession, device *DiscoveredDevice) error {
	if s.aiService == nil {
		return fmt.Errorf("AI service not available for AI-assisted configuration")
	}

	s.logger.WithField("session_id", session.ID).Info("Starting AI-assisted configuration...")

	// Use MCP tools to configure the device
	configParams := map[string]interface{}{
		"device_id":    device.ID,
		"device_mac":   device.MAC,
		"device_ip":    device.IP,
		"device_model": device.Model,
		"device_info":  device.DeviceInfo,
		"session_id":   session.ID,
		"wifi_settings": map[string]interface{}{
			"ssid":     "PMA-Network",     // This should come from configuration
			"password": "secure-password", // This should come from secure storage
		},
	}

	result, err := s.aiService.ExecuteMCPTool(ctx, "shelly_configure_device", configParams)
	if err != nil {
		return fmt.Errorf("AI configuration failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"result":     result,
	}).Info("AI-assisted configuration completed")

	return nil
}

// runSessionCleanup periodically cleans up expired sessions
func (s *Service) runSessionCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes expired configuration sessions
func (s *Service) cleanupExpiredSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for sessionID, session := range s.activeSessions {
		if now.Sub(session.StartedAt) > s.config.SessionTimeout {
			// Restore network if needed
			if session.NetworkBackup != nil && session.NetworkBackup.RestoreRequired {
				if err := s.networkManager.RestoreNetworkConfig(session.NetworkBackup); err != nil {
					s.logger.WithError(err).Warn("Failed to restore network config during cleanup")
				}
			}

			delete(s.activeSessions, sessionID)
			s.logger.WithField("session_id", sessionID).Info("Cleaned up expired configuration session")
		}
	}
}

// GetActiveSessions returns all active configuration sessions
func (s *Service) GetActiveSessions() map[string]*ConfigurationSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy to avoid race conditions
	sessions := make(map[string]*ConfigurationSession)
	for id, session := range s.activeSessions {
		sessionCopy := *session
		sessions[id] = &sessionCopy
	}

	return sessions
}

// GetDiscoveredDevices returns all discovered devices
func (s *Service) GetDiscoveredDevices() map[string]*DiscoveredDevice {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy to avoid race conditions
	devices := make(map[string]*DiscoveredDevice)
	for mac, device := range s.discoveredCache {
		deviceCopy := *device
		devices[mac] = &deviceCopy
	}

	return devices
}

// StartManualConfiguration starts a manual configuration session
func (s *Service) StartManualConfiguration(ctx context.Context, deviceMAC string, userID string) (*ConfigurationSession, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Find the device
	device, exists := s.discoveredCache[deviceMAC]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceMAC)
	}

	// Check session limits
	if len(s.activeSessions) >= s.config.MaxConcurrentSessions {
		return nil, fmt.Errorf("maximum concurrent sessions reached")
	}

	sessionID := fmt.Sprintf("manual_%s_%d", deviceMAC, time.Now().Unix())
	session := &ConfigurationSession{
		ID:        sessionID,
		DeviceID:  device.ID,
		DeviceMAC: deviceMAC,
		State:     StateIdle,
		Flow:      ConfigurationFlowManual,
		StartedAt: time.Now(),
		UserID:    userID,
		Steps:     []ConfigStep{},
	}

	s.activeSessions[sessionID] = session

	s.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"device_mac": deviceMAC,
		"user_id":    userID,
	}).Info("Started manual configuration session")

	return session, nil
}

// StartAIConfiguration starts an AI-assisted configuration session
func (s *Service) StartAIConfiguration(ctx context.Context, deviceMAC string, userID string) (*ConfigurationSession, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Find the device
	device, exists := s.discoveredCache[deviceMAC]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceMAC)
	}

	// Check session limits
	if len(s.activeSessions) >= s.config.MaxConcurrentSessions {
		return nil, fmt.Errorf("maximum concurrent sessions reached")
	}

	sessionID := fmt.Sprintf("ai_%s_%d", deviceMAC, time.Now().Unix())
	session := &ConfigurationSession{
		ID:        sessionID,
		DeviceID:  device.ID,
		DeviceMAC: deviceMAC,
		State:     StateIdle,
		Flow:      ConfigurationFlowAI,
		StartedAt: time.Now(),
		UserID:    userID,
		Steps:     []ConfigStep{},
	}

	s.activeSessions[sessionID] = session

	// Start AI-assisted configuration
	go func() {
		if err := s.executeConfigurationFlow(ctx, session, device); err != nil {
			s.logger.WithError(err).WithField("session_id", sessionID).Error("AI configuration failed")
			session.State = StateFailed
			session.Error = err.Error()
			now := time.Now()
			session.CompletedAt = &now
		}
	}()

	s.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"device_mac": deviceMAC,
		"user_id":    userID,
	}).Info("Started AI-assisted configuration session")

	return session, nil
}
