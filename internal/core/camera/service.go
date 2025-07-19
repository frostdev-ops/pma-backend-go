package camera

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/sirupsen/logrus"
)

// CameraService implements the Service interface
type CameraService struct {
	repo            CameraRepository
	ringIntegration RingIntegration
	logger          *logrus.Logger
}

// NewService creates a new camera service
func NewService(repo CameraRepository, ringIntegration RingIntegration, logger *logrus.Logger) Service {
	return &CameraService{
		repo:            repo,
		ringIntegration: ringIntegration,
		logger:          logger,
	}
}

// RegisterRingCamera registers a Ring camera in the database
func (s *CameraService) RegisterRingCamera(ctx context.Context, ringDevice *RingDeviceInfo) (*Camera, error) {
	entityID := fmt.Sprintf("ring_camera_%d", ringDevice.ID)

	// Check if camera already exists
	if existing := s.getCameraByEntityID(ctx, entityID); existing != nil {
		s.logger.WithField("entity_id", entityID).Info("Ring camera already registered, updating")
		return s.updateRingCamera(ctx, existing, ringDevice)
	}

	// Create capabilities based on Ring device features
	capabilities := CameraCapabilities{
		HasVideo:         true,
		HasAudio:         true,
		HasMotion:        true,
		HasSnapshot:      true,
		HasLiveStream:    true,
		HasRecording:     ringDevice.HasSubscription,
		HasBattery:       ringDevice.BatteryLife != nil,
		SupportedFormats: []string{"hls", "rtsp"},
	}

	// Check for light capability (most Ring devices have lights)
	if ringDevice.Kind == "doorbell" || ringDevice.Kind == "stickup_cam" {
		capabilities.HasLight = true
	}

	// Check for siren capability
	if ringDevice.Kind == "stickup_cam" || ringDevice.Kind == "doorbell" {
		capabilities.HasSiren = true
	}

	// Create settings
	settings := CameraSettings{
		MotionDetection:  ringDevice.MotionDetection,
		RecordingEnabled: ringDevice.StreamingEnabled && ringDevice.HasSubscription,
		Quality:          "high",
		StreamFormat:     "hls",
		NotificationSettings: map[string]interface{}{
			"motion_alerts":   true,
			"doorbell_alerts": ringDevice.Kind == "doorbell",
		},
	}

	camera := &Camera{
		EntityID:     entityID,
		Name:         ringDevice.Description,
		Type:         "ring",
		Capabilities: s.capabilitiesToMap(capabilities),
		Settings:     s.settingsToMap(settings),
		IsEnabled:    true,
	}

	// Set URLs for Ring integration
	streamURL := fmt.Sprintf("/api/ring/cameras/%d/stream", ringDevice.ID)
	snapshotURL := fmt.Sprintf("/api/ring/cameras/%d/snapshot", ringDevice.ID)
	camera.StreamURL = &streamURL
	camera.SnapshotURL = &snapshotURL

	// Create camera using repository
	err := s.repo.Create(ctx, camera)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ring camera: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"camera_id": camera.ID,
		"entity_id": entityID,
		"ring_id":   ringDevice.ID,
		"name":      ringDevice.Description,
	}).Info("Ring camera registered successfully")

	return camera, nil
}

// updateRingCamera updates an existing Ring camera with fresh data
func (s *CameraService) updateRingCamera(ctx context.Context, camera *Camera, ringDevice *RingDeviceInfo) (*Camera, error) {
	// Update basic info
	camera.Name = ringDevice.Description

	// Update capabilities and settings
	capabilities := s.mapToCapabilities(camera.Capabilities)
	capabilities.HasRecording = ringDevice.HasSubscription
	capabilities.HasBattery = ringDevice.BatteryLife != nil
	camera.Capabilities = s.capabilitiesToMap(capabilities)

	settings := s.mapToSettings(camera.Settings)
	settings.MotionDetection = ringDevice.MotionDetection
	settings.RecordingEnabled = ringDevice.StreamingEnabled && ringDevice.HasSubscription
	camera.Settings = s.settingsToMap(settings)

	// Update camera using repository
	err := s.repo.Update(ctx, camera)
	if err != nil {
		return nil, fmt.Errorf("failed to update Ring camera: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"camera_id": camera.ID,
		"ring_id":   ringDevice.ID,
		"name":      ringDevice.Description,
	}).Info("Ring camera updated successfully")

	return camera, nil
}

// RegisterGenericCamera registers a generic IP camera
func (s *CameraService) RegisterGenericCamera(ctx context.Context, name, streamURL, snapshotURL string, capabilities CameraCapabilities, settings CameraSettings) (*Camera, error) {
	entityID := fmt.Sprintf("camera_%d", time.Now().UnixNano())

	camera := &Camera{
		EntityID:     entityID,
		Name:         name,
		Type:         "generic",
		Capabilities: s.capabilitiesToMap(capabilities),
		Settings:     s.settingsToMap(settings),
		IsEnabled:    true,
	}

	if streamURL != "" {
		camera.StreamURL = &streamURL
	}

	if snapshotURL != "" {
		camera.SnapshotURL = &snapshotURL
	}

	// Create camera using repository
	err := s.repo.Create(ctx, camera)
	if err != nil {
		return nil, fmt.Errorf("failed to create generic camera: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"camera_id": camera.ID,
		"entity_id": entityID,
		"name":      name,
		"type":      "generic",
	}).Info("Generic camera registered successfully")

	return camera, nil
}

// SyncRingCameras synchronizes all Ring cameras with the database
func (s *CameraService) SyncRingCameras(ctx context.Context) error {
	if s.ringIntegration == nil || !s.ringIntegration.IsConnected() {
		return fmt.Errorf("Ring integration not available or not connected")
	}

	ringDevices, err := s.ringIntegration.GetCameras(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Ring cameras: %w", err)
	}

	s.logger.WithField("device_count", len(ringDevices)).Info("Starting Ring camera synchronization")

	for _, device := range ringDevices {
		// Only process camera-type devices
		if device.Kind != "doorbell" && device.Kind != "stickup_cam" && device.Kind != "cam" {
			continue
		}

		_, err := s.RegisterRingCamera(ctx, device)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"ring_id": device.ID,
				"name":    device.Description,
			}).Error("Failed to register Ring camera")
			// Continue with other cameras even if one fails
			continue
		}
	}

	s.logger.Info("Ring camera synchronization completed")
	return nil
}

// DiscoverRingCameras discovers and registers all Ring cameras
func (s *CameraService) DiscoverRingCameras(ctx context.Context) error {
	if s.ringIntegration == nil {
		return fmt.Errorf("Ring integration not available")
	}

	s.logger.Info("Starting Ring camera discovery")
	return s.SyncRingCameras(ctx)
}

// GetCameraCapabilities returns capabilities for a specific camera
func (s *CameraService) GetCameraCapabilities(ctx context.Context, cameraID int) (*CameraCapabilities, error) {
	camera := s.getCameraByID(ctx, cameraID)
	if camera == nil {
		return nil, fmt.Errorf("camera with ID %d not found", cameraID)
	}

	capabilities := s.mapToCapabilities(camera.Capabilities)
	return &capabilities, nil
}

// GetCameraSettings returns settings for a specific camera
func (s *CameraService) GetCameraSettings(ctx context.Context, cameraID int) (*CameraSettings, error) {
	camera := s.getCameraByID(ctx, cameraID)
	if camera == nil {
		return nil, fmt.Errorf("camera with ID %d not found", cameraID)
	}

	settings := s.mapToSettings(camera.Settings)
	return &settings, nil
}

// UpdateCameraSettings updates settings for a specific camera
func (s *CameraService) UpdateCameraSettings(ctx context.Context, cameraID int, settings CameraSettings) error {
	camera := s.getCameraByID(ctx, cameraID)
	if camera == nil {
		return fmt.Errorf("camera with ID %d not found", cameraID)
	}

	camera.Settings = s.settingsToMap(settings)

	// Update camera using repository adapter
	err := s.repo.Update(ctx, camera)
	if err != nil {
		return fmt.Errorf("failed to update camera settings: %w", err)
	}

	s.logger.WithField("camera_id", cameraID).Info("Camera settings updated successfully")
	return nil
}

// GetCameraHealth returns health status for a camera
func (s *CameraService) GetCameraHealth(ctx context.Context, cameraID int) (map[string]interface{}, error) {
	camera := s.getCameraByID(ctx, cameraID)
	if camera == nil {
		return nil, fmt.Errorf("camera with ID %d not found", cameraID)
	}

	health := map[string]interface{}{
		"camera_id":          camera.ID,
		"entity_id":          camera.EntityID,
		"name":               camera.Name,
		"type":               camera.Type,
		"enabled":            camera.IsEnabled,
		"last_updated":       camera.UpdatedAt,
		"stream_available":   camera.StreamURL != nil,
		"snapshot_available": camera.SnapshotURL != nil,
	}

	// Add type-specific health checks
	if camera.Type == "ring" && s.ringIntegration != nil {
		health["ring_connected"] = s.ringIntegration.IsConnected()

		// Get device health from Ring if available
		if ringHealth, err := s.ringIntegration.GetDeviceHealth(ctx, fmt.Sprintf("%d", camera.ID)); err == nil {
			health["ring_device_health"] = ringHealth
		}
	}

	return health, nil
}

// GetCameraStats returns camera statistics
func (s *CameraService) GetCameraStats(ctx context.Context) (map[string]interface{}, error) {
	totalCameras, err := s.repo.CountCameras(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total cameras: %w", err)
	}

	enabledCameras, err := s.repo.CountEnabledCameras(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count enabled cameras: %w", err)
	}

	// Get cameras by type
	ringCameras, err := s.repo.GetByType(ctx, "ring")
	if err != nil {
		return nil, fmt.Errorf("failed to get Ring cameras: %w", err)
	}

	genericCameras, err := s.repo.GetByType(ctx, "generic")
	if err != nil {
		return nil, fmt.Errorf("failed to get generic cameras: %w", err)
	}

	stats := map[string]interface{}{
		"total_cameras":   totalCameras,
		"enabled_cameras": enabledCameras,
		"ring_cameras":    len(ringCameras),
		"generic_cameras": len(genericCameras),
		"online_cameras":  enabledCameras, // For now, assume enabled = online
		"ring_connected":  s.ringIntegration != nil && s.ringIntegration.IsConnected(),
	}

	return stats, nil
}

// ValidateCameraURL validates if a camera stream or snapshot URL is accessible
func (s *CameraService) ValidateCameraURL(ctx context.Context, url string) error {
	// Basic URL validation for now
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	s.logger.WithField("url", url).Info("Validating camera URL")

	// TODO: Implement actual HTTP validation
	// For now, just log the validation attempt
	return nil
}

// Helper methods

func (s *CameraService) getCameraByID(ctx context.Context, id int) *Camera {
	camera, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil
	}
	return camera
}

func (s *CameraService) getCameraByEntityID(ctx context.Context, entityID string) *Camera {
	camera, err := s.repo.GetByEntityID(ctx, entityID)
	if err != nil {
		return nil
	}
	return camera
}

func (s *CameraService) cameraToModel(camera *Camera) *models.Camera {
	modelCamera := &models.Camera{
		ID:        camera.ID,
		EntityID:  camera.EntityID,
		Name:      camera.Name,
		Type:      camera.Type,
		IsEnabled: camera.IsEnabled,
		CreatedAt: camera.CreatedAt,
		UpdatedAt: camera.UpdatedAt,
	}

	// Convert URLs
	if camera.StreamURL != nil {
		modelCamera.StreamURL.String = *camera.StreamURL
		modelCamera.StreamURL.Valid = true
	}

	if camera.SnapshotURL != nil {
		modelCamera.SnapshotURL.String = *camera.SnapshotURL
		modelCamera.SnapshotURL.Valid = true
	}

	// Convert JSON fields
	if camera.Capabilities != nil {
		if capabilitiesJSON, err := json.Marshal(camera.Capabilities); err == nil {
			modelCamera.Capabilities = capabilitiesJSON
		}
	}

	if camera.Settings != nil {
		if settingsJSON, err := json.Marshal(camera.Settings); err == nil {
			modelCamera.Settings = settingsJSON
		}
	}

	return modelCamera
}

func (s *CameraService) modelToCamera(modelCamera *models.Camera) *Camera {
	camera := &Camera{
		ID:        modelCamera.ID,
		EntityID:  modelCamera.EntityID,
		Name:      modelCamera.Name,
		Type:      modelCamera.Type,
		IsEnabled: modelCamera.IsEnabled,
		CreatedAt: modelCamera.CreatedAt,
		UpdatedAt: modelCamera.UpdatedAt,
	}

	// Convert URLs
	if modelCamera.StreamURL.Valid {
		camera.StreamURL = &modelCamera.StreamURL.String
	}

	if modelCamera.SnapshotURL.Valid {
		camera.SnapshotURL = &modelCamera.SnapshotURL.String
	}

	// Convert JSON fields
	if len(modelCamera.Capabilities) > 0 {
		if err := json.Unmarshal(modelCamera.Capabilities, &camera.Capabilities); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal camera capabilities")
			camera.Capabilities = make(map[string]interface{})
		}
	}

	if len(modelCamera.Settings) > 0 {
		if err := json.Unmarshal(modelCamera.Settings, &camera.Settings); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal camera settings")
			camera.Settings = make(map[string]interface{})
		}
	}

	return camera
}

func (s *CameraService) capabilitiesToMap(capabilities CameraCapabilities) map[string]interface{} {
	return map[string]interface{}{
		"has_video":         capabilities.HasVideo,
		"has_audio":         capabilities.HasAudio,
		"has_motion":        capabilities.HasMotion,
		"has_light":         capabilities.HasLight,
		"has_siren":         capabilities.HasSiren,
		"has_snapshot":      capabilities.HasSnapshot,
		"has_live_stream":   capabilities.HasLiveStream,
		"has_recording":     capabilities.HasRecording,
		"has_battery":       capabilities.HasBattery,
		"supported_formats": capabilities.SupportedFormats,
	}
}

func (s *CameraService) mapToCapabilities(m map[string]interface{}) CameraCapabilities {
	capabilities := CameraCapabilities{}

	if val, ok := m["has_video"].(bool); ok {
		capabilities.HasVideo = val
	}
	if val, ok := m["has_audio"].(bool); ok {
		capabilities.HasAudio = val
	}
	if val, ok := m["has_motion"].(bool); ok {
		capabilities.HasMotion = val
	}
	if val, ok := m["has_light"].(bool); ok {
		capabilities.HasLight = val
	}
	if val, ok := m["has_siren"].(bool); ok {
		capabilities.HasSiren = val
	}
	if val, ok := m["has_snapshot"].(bool); ok {
		capabilities.HasSnapshot = val
	}
	if val, ok := m["has_live_stream"].(bool); ok {
		capabilities.HasLiveStream = val
	}
	if val, ok := m["has_recording"].(bool); ok {
		capabilities.HasRecording = val
	}
	if val, ok := m["has_battery"].(bool); ok {
		capabilities.HasBattery = val
	}
	if val, ok := m["supported_formats"].([]string); ok {
		capabilities.SupportedFormats = val
	}

	return capabilities
}

func (s *CameraService) settingsToMap(settings CameraSettings) map[string]interface{} {
	return map[string]interface{}{
		"motion_detection":      settings.MotionDetection,
		"recording_enabled":     settings.RecordingEnabled,
		"night_vision":          settings.NightVision,
		"quality":               settings.Quality,
		"stream_format":         settings.StreamFormat,
		"notification_settings": settings.NotificationSettings,
		"privacy":               settings.Privacy,
	}
}

func (s *CameraService) mapToSettings(m map[string]interface{}) CameraSettings {
	settings := CameraSettings{}

	if val, ok := m["motion_detection"].(bool); ok {
		settings.MotionDetection = val
	}
	if val, ok := m["recording_enabled"].(bool); ok {
		settings.RecordingEnabled = val
	}
	if val, ok := m["night_vision"].(bool); ok {
		settings.NightVision = val
	}
	if val, ok := m["quality"].(string); ok {
		settings.Quality = val
	}
	if val, ok := m["stream_format"].(string); ok {
		settings.StreamFormat = val
	}
	if val, ok := m["notification_settings"].(map[string]interface{}); ok {
		settings.NotificationSettings = val
	}
	if val, ok := m["privacy"].(map[string]interface{}); ok {
		settings.Privacy = val
	}

	return settings
}
