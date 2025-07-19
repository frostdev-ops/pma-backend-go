package kiosk

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service defines the kiosk service interface
type Service interface {
	// Pairing and token management
	GeneratePairingPIN(ctx context.Context, roomID string, allowedDevices []string) (string, error)
	ValidatePairingPIN(ctx context.Context, request *models.KioskPairingRequest) (*models.KioskPairingResponse, error)
	ValidateToken(ctx context.Context, token string) (*models.KioskToken, error)
	RevokeToken(ctx context.Context, tokenID string) error
	GetAllTokens(ctx context.Context) ([]*models.KioskToken, error)
	GetTokensByRoom(ctx context.Context, roomID string) ([]*models.KioskToken, error)

	// Configuration management
	GetKioskConfig(ctx context.Context, roomID string) (*models.KioskConfig, error)
	UpdateKioskConfig(ctx context.Context, roomID string, updates *models.KioskConfigUpdateRequest) error
	GetKioskDevices(ctx context.Context, token string) ([]*models.KioskDeviceInfo, error)

	// Device group management
	CreateDeviceGroup(ctx context.Context, request *models.KioskDeviceGroupCreateRequest) (*models.KioskDeviceGroup, error)
	GetAllDeviceGroups(ctx context.Context) ([]*models.KioskDeviceGroup, error)
	GetDeviceGroup(ctx context.Context, groupID string) (*models.KioskDeviceGroup, error)
	UpdateDeviceGroup(ctx context.Context, groupID string, updates *models.KioskDeviceGroupCreateRequest) error
	DeleteDeviceGroup(ctx context.Context, groupID string) error
	AddTokenToGroup(ctx context.Context, tokenID, groupID string) error
	RemoveTokenFromGroup(ctx context.Context, tokenID, groupID string) error

	// Device command execution
	ExecuteDeviceCommand(ctx context.Context, token string, command *models.KioskCommandRequest) (*models.KioskCommandResponse, error)

	// Logging and monitoring
	LogActivity(ctx context.Context, tokenID, level, category, message string, details map[string]interface{}) error
	GetLogs(ctx context.Context, tokenID string, query *models.KioskLogQuery) ([]*models.KioskLog, error)
	UpdateDeviceStatus(ctx context.Context, tokenID string, status *models.KioskDeviceStatus) error
	GetDeviceStatus(ctx context.Context, tokenID string) (*models.KioskDeviceStatus, error)
	RecordHeartbeat(ctx context.Context, tokenID string) error

	// Remote command management
	SendCommand(ctx context.Context, tokenID, commandType string, commandData map[string]interface{}) (string, error)
	GetPendingCommands(ctx context.Context, tokenID string) ([]*models.KioskCommand, error)
	AcknowledgeCommand(ctx context.Context, commandID string) error
	CompleteCommand(ctx context.Context, commandID string, resultData map[string]interface{}, errorMsg string) error

	// Statistics and health
	GetKioskStats(ctx context.Context) (*models.KioskStatsResponse, error)
	CleanupExpiredData(ctx context.Context) error
}

// ServiceImpl implements the kiosk service
type ServiceImpl struct {
	repo             repositories.KioskRepository
	entityRepo       repositories.EntityRepository
	roomRepo         repositories.RoomRepository
	logger           *logrus.Logger
	pinExpiryMinutes int
	tokenExpiryHours int
}

// NewService creates a new kiosk service
func NewService(
	repo repositories.KioskRepository,
	entityRepo repositories.EntityRepository,
	roomRepo repositories.RoomRepository,
	logger *logrus.Logger,
) Service {
	return &ServiceImpl{
		repo:             repo,
		entityRepo:       entityRepo,
		roomRepo:         roomRepo,
		logger:           logger,
		pinExpiryMinutes: 5,    // 5 minutes for PIN expiry
		tokenExpiryHours: 2160, // 90 days for token expiry
	}
}

// ======== PAIRING AND TOKEN MANAGEMENT ========

// GeneratePairingPIN generates a new 6-digit PIN for kiosk pairing
func (s *ServiceImpl) GeneratePairingPIN(ctx context.Context, roomID string, allowedDevices []string) (string, error) {
	// Generate a 6-digit PIN
	max := big.NewInt(999999)
	min := big.NewInt(100000)
	n, err := rand.Int(rand.Reader, max.Sub(max, min).Add(max, big.NewInt(1)))
	if err != nil {
		return "", fmt.Errorf("failed to generate random PIN: %w", err)
	}
	pin := fmt.Sprintf("%06d", n.Add(n, min).Int64())

	// Create device info for the session
	deviceInfo := map[string]interface{}{
		"allowed_devices": allowedDevices,
		"generated_at":    time.Now().UTC(),
	}
	deviceInfoBytes, _ := json.Marshal(deviceInfo)

	// Create pairing session
	session := &models.KioskPairingSession{
		ID:         uuid.New().String(),
		Pin:        pin,
		RoomID:     roomID,
		DeviceInfo: deviceInfoBytes,
		ExpiresAt:  time.Now().Add(time.Duration(s.pinExpiryMinutes) * time.Minute),
		Status:     "pending",
	}

	err = s.repo.CreatePairingSession(ctx, session)
	if err != nil {
		return "", fmt.Errorf("failed to create pairing session: %w", err)
	}

	s.logger.Infof("Generated kiosk pairing PIN: %s for room: %s", pin, roomID)
	return pin, nil
}

// ValidatePairingPIN validates a PIN and returns a long-lived token
func (s *ServiceImpl) ValidatePairingPIN(ctx context.Context, request *models.KioskPairingRequest) (*models.KioskPairingResponse, error) {
	// Get pairing session
	session, err := s.repo.GetPairingSession(ctx, request.Pin)
	if err != nil {
		return &models.KioskPairingResponse{
			Success: false,
			Error:   "Invalid or expired PIN",
		}, nil
	}

	// Validate room ID if provided
	if request.RoomID != "" && session.RoomID != request.RoomID {
		return &models.KioskPairingResponse{
			Success: false,
			Error:   "PIN does not match the specified room",
		}, nil
	}

	// Parse device info
	var deviceInfo map[string]interface{}
	if err := json.Unmarshal(session.DeviceInfo, &deviceInfo); err != nil {
		s.logger.Warnf("Failed to parse device info from session: %v", err)
	}

	// Get allowed devices from request or session
	allowedDevices := request.AllowedDevices
	if len(allowedDevices) == 0 {
		if devices, ok := deviceInfo["allowed_devices"].([]string); ok {
			allowedDevices = devices
		}
	}

	// Generate long-lived token
	tokenID := uuid.New().String()
	token := fmt.Sprintf("kiosk_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	allowedDevicesBytes, _ := json.Marshal(allowedDevices)

	// Create kiosk token
	kioskToken := &models.KioskToken{
		ID:             tokenID,
		Token:          token,
		Name:           request.Name,
		RoomID:         session.RoomID,
		AllowedDevices: allowedDevicesBytes,
		Active:         true,
		ExpiresAt:      sql.NullTime{Time: time.Now().Add(time.Duration(s.tokenExpiryHours) * time.Hour), Valid: true},
	}

	err = s.repo.CreateToken(ctx, kioskToken)
	if err != nil {
		return &models.KioskPairingResponse{
			Success: false,
			Error:   "Failed to create kiosk token",
		}, nil
	}

	// Delete the used pairing session
	_ = s.repo.DeletePairingSession(ctx, session.ID)

	// Get kiosk configuration
	config, err := s.GetKioskConfig(ctx, session.RoomID)
	if err != nil {
		s.logger.Warnf("Failed to get kiosk config for room %s: %v", session.RoomID, err)
	}

	// Log successful pairing
	_ = s.LogActivity(ctx, tokenID, "info", "security",
		fmt.Sprintf("Kiosk device '%s' successfully paired", request.Name),
		map[string]interface{}{
			"room_id": session.RoomID,
			"pin":     request.Pin,
		})

	s.logger.Infof("Kiosk paired successfully: %s for room: %s", request.Name, session.RoomID)

	return &models.KioskPairingResponse{
		Success:   true,
		Token:     token,
		Config:    config,
		ExpiresAt: kioskToken.ExpiresAt.Time.Format(time.RFC3339),
	}, nil
}

// ValidateToken validates a kiosk token and updates last used timestamp
func (s *ServiceImpl) ValidateToken(ctx context.Context, token string) (*models.KioskToken, error) {
	kioskToken, err := s.repo.GetToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if token has expired
	if kioskToken.ExpiresAt.Valid && kioskToken.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	// Update last used timestamp
	err = s.repo.UpdateTokenLastUsed(ctx, token)
	if err != nil {
		s.logger.Warnf("Failed to update token last used: %v", err)
	}

	return kioskToken, nil
}

// RevokeToken deactivates a kiosk token
func (s *ServiceImpl) RevokeToken(ctx context.Context, tokenID string) error {
	err := s.repo.UpdateTokenStatus(ctx, tokenID, false)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Log token revocation
	_ = s.LogActivity(ctx, tokenID, "warn", "security", "Kiosk token revoked", nil)

	return nil
}

// GetAllTokens retrieves all active kiosk tokens
func (s *ServiceImpl) GetAllTokens(ctx context.Context) ([]*models.KioskToken, error) {
	return s.repo.GetAllTokens(ctx)
}

// GetTokensByRoom retrieves kiosk tokens for a specific room
func (s *ServiceImpl) GetTokensByRoom(ctx context.Context, roomID string) ([]*models.KioskToken, error) {
	return s.repo.GetTokensByRoom(ctx, roomID)
}

// ======== CONFIGURATION MANAGEMENT ========

// GetKioskConfig retrieves kiosk configuration for a room, creating default if not exists
func (s *ServiceImpl) GetKioskConfig(ctx context.Context, roomID string) (*models.KioskConfig, error) {
	config, err := s.repo.GetConfig(ctx, roomID)
	if err != nil {
		// Create default configuration if not found
		if strings.Contains(err.Error(), "not found") {
			defaultConfig := &models.KioskConfig{
				RoomID:                roomID,
				Theme:                 "auto",
				Layout:                "grid",
				QuickActions:          json.RawMessage("[]"),
				UpdateInterval:        1000,
				DisplayTimeout:        300,
				Brightness:            80,
				ScreensaverEnabled:    true,
				ScreensaverType:       "clock",
				ScreensaverTimeout:    900,
				AutoHideNavigation:    false,
				FullscreenMode:        true,
				VoiceControlEnabled:   false,
				GestureControlEnabled: false,
			}

			err = s.repo.CreateConfig(ctx, defaultConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}

			return defaultConfig, nil
		}
		return nil, err
	}

	return config, nil
}

// UpdateKioskConfig updates kiosk configuration for a room
func (s *ServiceImpl) UpdateKioskConfig(ctx context.Context, roomID string, updates *models.KioskConfigUpdateRequest) error {
	// Get current config
	config, err := s.GetKioskConfig(ctx, roomID)
	if err != nil {
		return err
	}

	// Apply updates
	if updates.Theme != nil {
		config.Theme = *updates.Theme
	}
	if updates.Layout != nil {
		config.Layout = *updates.Layout
	}
	if updates.QuickActions != nil {
		quickActionsBytes, _ := json.Marshal(updates.QuickActions)
		config.QuickActions = quickActionsBytes
	}
	if updates.UpdateInterval != nil {
		config.UpdateInterval = *updates.UpdateInterval
	}
	if updates.DisplayTimeout != nil {
		config.DisplayTimeout = *updates.DisplayTimeout
	}
	if updates.Brightness != nil {
		config.Brightness = *updates.Brightness
	}
	if updates.ScreensaverEnabled != nil {
		config.ScreensaverEnabled = *updates.ScreensaverEnabled
	}
	if updates.ScreensaverType != nil {
		config.ScreensaverType = *updates.ScreensaverType
	}
	if updates.ScreensaverTimeout != nil {
		config.ScreensaverTimeout = *updates.ScreensaverTimeout
	}
	if updates.AutoHideNavigation != nil {
		config.AutoHideNavigation = *updates.AutoHideNavigation
	}
	if updates.FullscreenMode != nil {
		config.FullscreenMode = *updates.FullscreenMode
	}
	if updates.VoiceControlEnabled != nil {
		config.VoiceControlEnabled = *updates.VoiceControlEnabled
	}
	if updates.GestureControlEnabled != nil {
		config.GestureControlEnabled = *updates.GestureControlEnabled
	}

	// Save updated config
	err = s.repo.UpdateConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Log configuration update
	tokens, _ := s.repo.GetTokensByRoom(ctx, roomID)
	for _, token := range tokens {
		_ = s.LogActivity(ctx, token.ID, "info", "system", "Kiosk configuration updated",
			map[string]interface{}{
				"updates": updates,
			})
	}

	return nil
}

// GetKioskDevices retrieves devices allowed for a kiosk token
func (s *ServiceImpl) GetKioskDevices(ctx context.Context, token string) ([]*models.KioskDeviceInfo, error) {
	kioskToken, err := s.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Parse allowed devices
	var allowedDeviceIds []string
	if err := json.Unmarshal(kioskToken.AllowedDevices, &allowedDeviceIds); err != nil {
		s.logger.Warnf("Failed to parse allowed devices for token %s: %v", token, err)
	}

	// Convert room ID to integer for entity repository
	roomIDInt, err := strconv.Atoi(kioskToken.RoomID)
	if err != nil {
		return nil, fmt.Errorf("invalid room ID format: %w", err)
	}

	// Get entities from the room
	entities, err := s.entityRepo.GetByRoom(ctx, roomIDInt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities for room: %w", err)
	}

	var devices []*models.KioskDeviceInfo
	for _, entity := range entities {
		// Filter by allowed devices if specified
		if len(allowedDeviceIds) > 0 {
			allowed := false
			for _, allowedID := range allowedDeviceIds {
				if entity.EntityID == allowedID {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		// Parse attributes
		var attributes map[string]interface{}
		if entity.Attributes != nil {
			_ = json.Unmarshal(entity.Attributes, &attributes)
		}

		device := &models.KioskDeviceInfo{
			ID:         entity.EntityID,
			Name:       entity.FriendlyName.String,
			Type:       entity.Domain,
			State:      entity.State.String,
			Attributes: attributes,
		}

		// Set icon based on device type
		device.Icon = getDeviceIcon(entity.Domain, attributes)

		devices = append(devices, device)
	}

	return devices, nil
}

// ======== DEVICE GROUP MANAGEMENT ========

// CreateDeviceGroup creates a new device group
func (s *ServiceImpl) CreateDeviceGroup(ctx context.Context, request *models.KioskDeviceGroupCreateRequest) (*models.KioskDeviceGroup, error) {
	group := &models.KioskDeviceGroup{
		ID:          generateGroupID(request.Name),
		Name:        request.Name,
		Description: request.Description,
		Color:       request.Color,
		Icon:        request.Icon,
	}

	// Set defaults
	if group.Color == "" {
		group.Color = "#3b82f6"
	}
	if group.Icon == "" {
		group.Icon = "devices"
	}

	err := s.repo.CreateDeviceGroup(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to create device group: %w", err)
	}

	return group, nil
}

// GetAllDeviceGroups retrieves all device groups
func (s *ServiceImpl) GetAllDeviceGroups(ctx context.Context) ([]*models.KioskDeviceGroup, error) {
	return s.repo.GetAllDeviceGroups(ctx)
}

// GetDeviceGroup retrieves a device group by ID
func (s *ServiceImpl) GetDeviceGroup(ctx context.Context, groupID string) (*models.KioskDeviceGroup, error) {
	return s.repo.GetDeviceGroup(ctx, groupID)
}

// UpdateDeviceGroup updates a device group
func (s *ServiceImpl) UpdateDeviceGroup(ctx context.Context, groupID string, updates *models.KioskDeviceGroupCreateRequest) error {
	group, err := s.repo.GetDeviceGroup(ctx, groupID)
	if err != nil {
		return err
	}

	// Apply updates
	group.Name = updates.Name
	group.Description = updates.Description
	if updates.Color != "" {
		group.Color = updates.Color
	}
	if updates.Icon != "" {
		group.Icon = updates.Icon
	}

	return s.repo.UpdateDeviceGroup(ctx, group)
}

// DeleteDeviceGroup removes a device group
func (s *ServiceImpl) DeleteDeviceGroup(ctx context.Context, groupID string) error {
	return s.repo.DeleteDeviceGroup(ctx, groupID)
}

// AddTokenToGroup adds a kiosk token to a device group
func (s *ServiceImpl) AddTokenToGroup(ctx context.Context, tokenID, groupID string) error {
	return s.repo.AddTokenToGroup(ctx, tokenID, groupID)
}

// RemoveTokenFromGroup removes a kiosk token from a device group
func (s *ServiceImpl) RemoveTokenFromGroup(ctx context.Context, tokenID, groupID string) error {
	return s.repo.RemoveTokenFromGroup(ctx, tokenID, groupID)
}

// ======== DEVICE COMMAND EXECUTION ========

// ExecuteDeviceCommand executes a command on a device
func (s *ServiceImpl) ExecuteDeviceCommand(ctx context.Context, token string, command *models.KioskCommandRequest) (*models.KioskCommandResponse, error) {
	// Validate token
	kioskToken, err := s.ValidateToken(ctx, token)
	if err != nil {
		return &models.KioskCommandResponse{
			Success:   false,
			DeviceID:  command.DeviceID,
			Error:     "Invalid or expired token",
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// Check if device is allowed
	var allowedDeviceIds []string
	if err := json.Unmarshal(kioskToken.AllowedDevices, &allowedDeviceIds); err == nil && len(allowedDeviceIds) > 0 {
		allowed := false
		for _, allowedID := range allowedDeviceIds {
			if command.DeviceID == allowedID {
				allowed = true
				break
			}
		}
		if !allowed {
			return &models.KioskCommandResponse{
				Success:   false,
				DeviceID:  command.DeviceID,
				Error:     "Device not allowed for this kiosk",
				Timestamp: time.Now().Format(time.RFC3339),
			}, nil
		}
	}

	// Get device entity
	entity, err := s.entityRepo.GetByID(ctx, command.DeviceID)
	if err != nil {
		return &models.KioskCommandResponse{
			Success:   false,
			DeviceID:  command.DeviceID,
			Error:     "Device not found",
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// Execute command (this would integrate with Home Assistant or device controllers)
	// For now, we'll simulate the command execution
	newState := simulateDeviceCommand(entity, command)

	// Log the command execution
	_ = s.LogActivity(ctx, kioskToken.ID, "info", "user_action",
		fmt.Sprintf("Device command executed: %s on %s", command.Action, command.DeviceID),
		map[string]interface{}{
			"device_id": command.DeviceID,
			"action":    command.Action,
			"payload":   command.Payload,
			"old_state": entity.State.String,
			"new_state": newState,
		})

	return &models.KioskCommandResponse{
		Success:   true,
		DeviceID:  command.DeviceID,
		NewState:  newState,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// ======== LOGGING AND MONITORING ========

// LogActivity creates a log entry for kiosk activity
func (s *ServiceImpl) LogActivity(ctx context.Context, tokenID, level, category, message string, details map[string]interface{}) error {
	detailsBytes, _ := json.Marshal(details)

	log := &models.KioskLog{
		KioskTokenID: tokenID,
		Level:        level,
		Category:     category,
		Message:      message,
		Details:      detailsBytes,
	}

	return s.repo.CreateLog(ctx, log)
}

// GetLogs retrieves logs for a kiosk token
func (s *ServiceImpl) GetLogs(ctx context.Context, tokenID string, query *models.KioskLogQuery) ([]*models.KioskLog, error) {
	return s.repo.GetLogs(ctx, tokenID, query)
}

// UpdateDeviceStatus updates the status of a kiosk device
func (s *ServiceImpl) UpdateDeviceStatus(ctx context.Context, tokenID string, status *models.KioskDeviceStatus) error {
	status.KioskTokenID = tokenID
	return s.repo.CreateOrUpdateDeviceStatus(ctx, status)
}

// GetDeviceStatus retrieves the status of a kiosk device
func (s *ServiceImpl) GetDeviceStatus(ctx context.Context, tokenID string) (*models.KioskDeviceStatus, error) {
	return s.repo.GetDeviceStatus(ctx, tokenID)
}

// RecordHeartbeat records a heartbeat for a kiosk device
func (s *ServiceImpl) RecordHeartbeat(ctx context.Context, tokenID string) error {
	return s.repo.UpdateHeartbeat(ctx, tokenID)
}

// ======== REMOTE COMMAND MANAGEMENT ========

// SendCommand sends a remote command to a kiosk device
func (s *ServiceImpl) SendCommand(ctx context.Context, tokenID, commandType string, commandData map[string]interface{}) (string, error) {
	commandDataBytes, _ := json.Marshal(commandData)

	command := &models.KioskCommand{
		ID:           uuid.New().String(),
		KioskTokenID: tokenID,
		CommandType:  commandType,
		CommandData:  commandDataBytes,
		Status:       "pending",
		ExpiresAt:    time.Now().Add(24 * time.Hour), // Commands expire after 24 hours
	}

	err := s.repo.CreateCommand(ctx, command)
	if err != nil {
		return "", fmt.Errorf("failed to create command: %w", err)
	}

	return command.ID, nil
}

// GetPendingCommands retrieves pending commands for a kiosk token
func (s *ServiceImpl) GetPendingCommands(ctx context.Context, tokenID string) ([]*models.KioskCommand, error) {
	return s.repo.GetPendingCommands(ctx, tokenID)
}

// AcknowledgeCommand marks a command as acknowledged
func (s *ServiceImpl) AcknowledgeCommand(ctx context.Context, commandID string) error {
	return s.repo.UpdateCommandStatus(ctx, commandID, "acknowledged")
}

// CompleteCommand marks a command as completed with result data
func (s *ServiceImpl) CompleteCommand(ctx context.Context, commandID string, resultData map[string]interface{}, errorMsg string) error {
	resultDataBytes, _ := json.Marshal(resultData)
	return s.repo.CompleteCommand(ctx, commandID, resultDataBytes, errorMsg)
}

// ======== STATISTICS AND HEALTH ========

// GetKioskStats retrieves kiosk system statistics
func (s *ServiceImpl) GetKioskStats(ctx context.Context) (*models.KioskStatsResponse, error) {
	return s.repo.GetKioskStats(ctx)
}

// CleanupExpiredData removes expired sessions, commands, and old logs
func (s *ServiceImpl) CleanupExpiredData(ctx context.Context) error {
	// Cleanup expired pairing sessions
	if err := s.repo.CleanupExpiredSessions(ctx); err != nil {
		s.logger.Warnf("Failed to cleanup expired sessions: %v", err)
	}

	// Cleanup expired commands
	if err := s.repo.CleanupExpiredCommands(ctx); err != nil {
		s.logger.Warnf("Failed to cleanup expired commands: %v", err)
	}

	// Cleanup old logs (older than 30 days)
	if err := s.repo.DeleteOldLogs(ctx, 30); err != nil {
		s.logger.Warnf("Failed to cleanup old logs: %v", err)
	}

	return nil
}

// ======== HELPER FUNCTIONS ========

// getDeviceIcon returns an appropriate icon for a device based on its domain and attributes
func getDeviceIcon(domain string, attributes map[string]interface{}) string {
	switch domain {
	case "light":
		return "lightbulb"
	case "switch":
		return "power"
	case "cover":
		return "window"
	case "climate":
		return "thermometer"
	case "fan":
		return "fan"
	case "media_player":
		return "music"
	case "camera":
		return "camera"
	case "lock":
		return "lock"
	case "alarm_control_panel":
		return "shield"
	case "sensor":
		if deviceClass, ok := attributes["device_class"].(string); ok {
			switch deviceClass {
			case "temperature":
				return "thermometer"
			case "humidity":
				return "droplet"
			case "motion":
				return "motion"
			case "door", "window":
				return "door"
			}
		}
		return "sensor"
	default:
		return "device"
	}
}

// generateGroupID generates a URL-safe ID from a group name
func generateGroupID(name string) string {
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	// Add timestamp to ensure uniqueness
	id = fmt.Sprintf("%s-%d", id, time.Now().Unix())
	return id
}

// simulateDeviceCommand simulates device command execution
func simulateDeviceCommand(entity *models.Entity, command *models.KioskCommandRequest) string {
	switch command.Action {
	case "toggle":
		if entity.State.String == "on" {
			return "off"
		}
		return "on"
	case "turn_on":
		return "on"
	case "turn_off":
		return "off"
	default:
		return entity.State.String
	}
}
