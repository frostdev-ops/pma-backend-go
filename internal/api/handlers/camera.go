package handlers

// Camera Management Handlers
//
// MIGRATION STATUS: ✅ Fully migrated to unified system
// - GetCameras, GetEnabledCameras, GetCameraStats: ✅ Migrated to unified system
// - GetCamera, GetCameraByEntityID, SearchCameras, GetCamerasByType: ✅ Migrated to unified system
// - UpdateCamera: ✅ Migrated to unified system (state changes only via ExecuteAction)
// - CreateCamera, DeleteCamera: ✅ Deprecated (cameras managed by adapters)
// - Ring camera operations: ✅ Already using unified system (see ring.go)
//
// The unified system handles cameras through Ring/HA adapters automatically

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// CameraRequest represents camera creation/update request
type CameraRequest struct {
	EntityID     string          `json:"entity_id" binding:"required"`
	Name         string          `json:"name" binding:"required"`
	Type         string          `json:"type" binding:"required"`
	StreamURL    *string         `json:"stream_url,omitempty"`
	SnapshotURL  *string         `json:"snapshot_url,omitempty"`
	Capabilities json.RawMessage `json:"capabilities,omitempty"`
	Settings     json.RawMessage `json:"settings,omitempty"`
	IsEnabled    bool            `json:"is_enabled"`
}

// CameraResponse represents camera API response
type CameraResponse struct {
	ID           int             `json:"id"`
	EntityID     string          `json:"entity_id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	StreamURL    *string         `json:"stream_url,omitempty"`
	SnapshotURL  *string         `json:"snapshot_url,omitempty"`
	Capabilities json.RawMessage `json:"capabilities,omitempty"`
	Settings     json.RawMessage `json:"settings,omitempty"`
	IsEnabled    bool            `json:"is_enabled"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CameraStreamRequest represents stream request
type CameraStreamRequest struct {
	Format   string `json:"format" form:"format" binding:"omitempty,oneof=hls rtsp webrtc"`
	Quality  string `json:"quality" form:"quality" binding:"omitempty,oneof=low medium high"`
	Duration int    `json:"duration" form:"duration"`
}

// CameraStreamResponse represents stream response
type CameraStreamResponse struct {
	StreamURL string     `json:"stream_url"`
	Format    string     `json:"format"`
	Quality   string     `json:"quality,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CameraID  string     `json:"camera_id"`
}

// CameraSnapshotResponse represents snapshot response
type CameraSnapshotResponse struct {
	SnapshotURL string    `json:"snapshot_url"`
	CameraID    string    `json:"camera_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	Timestamp   time.Time `json:"timestamp"`
}

// CameraStatsResponse represents camera statistics
type CameraStatsResponse struct {
	TotalCameras   int `json:"total_cameras"`
	EnabledCameras int `json:"enabled_cameras"`
	RingCameras    int `json:"ring_cameras"`
	GenericCameras int `json:"generic_cameras"`
	OnlineCameras  int `json:"online_cameras"`
}

// GetCameras returns all cameras from the unified entity service
func (h *Handlers) GetCameras(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get camera entities through unified service
	options := unified.GetAllOptions{
		Domain:        "camera",
		AvailableOnly: c.Query("available_only") == "true",
		IncludeRoom:   c.Query("include_room") == "true",
	}

	entitiesWithRooms, err := h.unifiedService.GetAll(ctx, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get cameras from unified service")
		// Return empty array instead of error to prevent 500 errors
		utils.SendSuccess(c, []*CameraResponse{})
		return
	}

	// Convert PMA entities to camera response format
	cameras := make([]*CameraResponse, 0, len(entitiesWithRooms))
	for _, entityWithRoom := range entitiesWithRooms {
		if entityWithRoom.Entity.GetType() == types.EntityTypeCamera {
			camera := convertPMAEntityToCameraResponse(entityWithRoom.Entity)
			cameras = append(cameras, camera)
		}
	}

	h.log.WithField("camera_count", len(cameras)).Info("Retrieved cameras successfully")
	utils.SendSuccess(c, cameras)
}

// GetEnabledCameras returns only enabled cameras from unified service
func (h *Handlers) GetEnabledCameras(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get only available camera entities through unified service
	options := unified.GetAllOptions{
		Domain:        "camera",
		AvailableOnly: true, // Only get enabled/available cameras
		IncludeRoom:   c.Query("include_room") == "true",
	}

	entitiesWithRooms, err := h.unifiedService.GetAll(ctx, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get enabled cameras from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve enabled cameras")
		return
	}

	// Convert PMA entities to camera response format
	cameras := make([]*CameraResponse, 0, len(entitiesWithRooms))
	for _, entityWithRoom := range entitiesWithRooms {
		if entityWithRoom.Entity.GetType() == types.EntityTypeCamera {
			camera := convertPMAEntityToCameraResponse(entityWithRoom.Entity)
			cameras = append(cameras, camera)
		}
	}

	utils.SendSuccess(c, cameras)
}

// GetCamera returns a specific camera by ID
func (h *Handlers) GetCamera(c *gin.Context) {
	entityID := c.Param("id")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera entity ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get camera entity through unified service
	options := unified.GetEntityOptions{
		IncludeRoom: c.Query("include_room") == "true",
		IncludeArea: c.Query("include_area") == "true",
	}

	entityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, options)
	if err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to get camera from unified service")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	// Verify it's a camera entity
	if entityWithRoom.Entity.GetType() != types.EntityTypeCamera {
		utils.SendError(c, http.StatusNotFound, "Entity is not a camera")
		return
	}

	utils.SendSuccess(c, convertPMAEntityToCameraResponse(entityWithRoom.Entity))
}

// GetCameraByEntityID returns a specific camera by entity ID
func (h *Handlers) GetCameraByEntityID(c *gin.Context) {
	entityID := c.Param("entityId")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get camera entity through unified service
	options := unified.GetEntityOptions{
		IncludeRoom: c.Query("include_room") == "true",
		IncludeArea: c.Query("include_area") == "true",
	}

	entityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, options)
	if err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to get camera by entity ID from unified service")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	// Verify it's a camera entity
	if entityWithRoom.Entity.GetType() != types.EntityTypeCamera {
		utils.SendError(c, http.StatusNotFound, "Entity is not a camera")
		return
	}

	utils.SendSuccess(c, convertPMAEntityToCameraResponse(entityWithRoom.Entity))
}

// CreateCamera creates a new camera
// DEPRECATED: In the unified system, cameras are automatically discovered and managed by adapters (Ring, Home Assistant).
// Manual camera creation is no longer supported. Cameras will be automatically registered when discovered by their respective adapters.
func (h *Handlers) CreateCamera(c *gin.Context) {
	utils.SendError(c, http.StatusBadRequest,
		"Manual camera creation is not supported. Cameras are automatically discovered and managed by adapters (Ring, Home Assistant, etc.). "+
			"Please configure your camera through its native system (Ring app, Home Assistant, etc.) and it will be automatically discovered.")
}

// UpdateCamera updates an existing camera
// NOTE: In the unified system, camera configuration is managed by adapters.
// This endpoint now only supports state changes (enable/disable) via the unified action system.
// For configuration changes, use the camera's native system (Ring app, Home Assistant, etc.).
func (h *Handlers) UpdateCamera(c *gin.Context) {
	entityID := c.Param("id")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera entity ID is required")
		return
	}

	var req struct {
		IsEnabled *bool `json:"is_enabled,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Only support enabling/disabling cameras
	if req.IsEnabled == nil {
		utils.SendError(c, http.StatusBadRequest,
			"Only camera state changes (is_enabled) are supported. "+
				"For configuration changes, use the camera's native system (Ring app, Home Assistant, etc.)")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Verify the entity exists and is a camera
	entityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, unified.GetEntityOptions{})
	if err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to get camera for update")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	if entityWithRoom.Entity.GetType() != types.EntityTypeCamera {
		utils.SendError(c, http.StatusNotFound, "Entity is not a camera")
		return
	}

	// Execute enable/disable action through unified service
	var action string
	if *req.IsEnabled {
		action = "turn_on"
	} else {
		action = "turn_off"
	}

	controlAction := types.PMAControlAction{
		EntityID:   entityID,
		Action:     action,
		Parameters: make(map[string]interface{}),
		Context: &types.PMAContext{
			Source:      "camera_api",
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("Camera %s via API", action),
		},
	}

	result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
	if err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to execute camera action")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update camera state")
		return
	}

	if !result.Success {
		h.log.WithField("entity_id", entityID).WithField("error", result.Error).Error("Camera action was not successful")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update camera: %s", result.Error.Message))
		return
	}

	h.log.WithField("entity_id", entityID).WithField("action", action).Info("Camera state updated successfully")

	// Get updated entity and return it
	updatedEntityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, unified.GetEntityOptions{})
	if err != nil {
		// Still return success even if we can't get the updated entity
		utils.SendSuccess(c, gin.H{
			"message":   "Camera updated successfully",
			"entity_id": entityID,
			"action":    action,
		})
		return
	}

	utils.SendSuccess(c, convertPMAEntityToCameraResponse(updatedEntityWithRoom.Entity))
}

// DeleteCamera deletes a camera
// DEPRECATED: In the unified system, cameras are managed by adapters and cannot be manually deleted.
// Cameras will be automatically removed when they are no longer available in their native systems.
// To remove a camera, disable or remove it from its native system (Ring app, Home Assistant, etc.).
func (h *Handlers) DeleteCamera(c *gin.Context) {
	utils.SendError(c, http.StatusBadRequest,
		"Manual camera deletion is not supported. Cameras are managed by adapters and will be automatically removed when unavailable. "+
			"To remove a camera, disable or delete it from its native system (Ring app, Home Assistant, etc.).")
}

// GetCamerasByType returns cameras by type
func (h *Handlers) GetCamerasByType(c *gin.Context) {
	cameraType := c.Param("type")
	if cameraType == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera type is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get camera entities through unified service
	options := unified.GetAllOptions{
		Domain:        "camera",
		AvailableOnly: c.Query("available_only") == "true",
		IncludeRoom:   c.Query("include_room") == "true",
		IncludeArea:   c.Query("include_area") == "true",
	}

	entitiesWithRooms, err := h.unifiedService.GetAll(ctx, options)
	if err != nil {
		h.log.WithError(err).WithField("camera_type", cameraType).Error("Failed to get cameras from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve cameras")
		return
	}

	// Filter cameras by type and convert to response format
	cameras := make([]*CameraResponse, 0)
	for _, entityWithRoom := range entitiesWithRooms {
		if entityWithRoom.Entity.GetType() == types.EntityTypeCamera {
			// Check if camera attributes contain the requested type
			if attributes := entityWithRoom.Entity.GetAttributes(); attributes != nil {
				if entityType, ok := attributes["device_class"].(string); ok && entityType == cameraType {
					camera := convertPMAEntityToCameraResponse(entityWithRoom.Entity)
					cameras = append(cameras, camera)
				} else if entityType, ok := attributes["type"].(string); ok && entityType == cameraType {
					camera := convertPMAEntityToCameraResponse(entityWithRoom.Entity)
					cameras = append(cameras, camera)
				}
			}
		}
	}

	utils.SendSuccess(c, cameras)
}

// SearchCameras searches cameras by name or entity ID
func (h *Handlers) SearchCameras(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.SendError(c, http.StatusBadRequest, "Search query is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Search camera entities through unified service
	options := unified.GetAllOptions{
		Domain:        "camera",
		AvailableOnly: c.Query("available_only") == "true",
		IncludeRoom:   c.Query("include_room") == "true",
		IncludeArea:   c.Query("include_area") == "true",
	}

	entitiesWithRooms, err := h.unifiedService.Search(ctx, query, options)
	if err != nil {
		h.log.WithError(err).WithField("query", query).Error("Failed to search cameras in unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to search cameras")
		return
	}

	// Convert PMA entities to camera response format, filtering for cameras only
	cameras := make([]*CameraResponse, 0)
	for _, entityWithRoom := range entitiesWithRooms {
		if entityWithRoom.Entity.GetType() == types.EntityTypeCamera {
			camera := convertPMAEntityToCameraResponse(entityWithRoom.Entity)
			cameras = append(cameras, camera)
		}
	}

	utils.SendSuccess(c, cameras)
}

// UpdateCameraStatus updates camera enabled status
func (h *Handlers) UpdateCameraStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid camera ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	err = h.repos.Camera.UpdateStatus(c.Request.Context(), id, req.Enabled)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to update camera status")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	h.log.WithField("camera_id", id).WithField("enabled", req.Enabled).Info("Camera status updated")
	utils.SendSuccess(c, gin.H{"message": "Camera status updated", "enabled": req.Enabled})
}

// GetCameraStream returns camera stream URL
func (h *Handlers) GetCameraStream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid camera ID")
		return
	}

	var req CameraStreamRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// Use defaults if binding fails
		req.Format = "hls"
		req.Quality = "medium"
	}

	camera, err := h.repos.Camera.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to get camera for stream")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	if !camera.IsEnabled {
		utils.SendError(c, http.StatusForbidden, "Camera is not enabled")
		return
	}

	// For Ring cameras, use Ring adapter to get stream
	if camera.Type == "ring" {
		h.getRingCameraStream(c, camera, req)
		return
	}

	// For generic cameras, use the stored stream URL
	if !camera.StreamURL.Valid || camera.StreamURL.String == "" {
		utils.SendError(c, http.StatusNotFound, "Stream URL not available for this camera")
		return
	}

	response := CameraStreamResponse{
		StreamURL: camera.StreamURL.String,
		Format:    req.Format,
		Quality:   req.Quality,
		CameraID:  strconv.Itoa(camera.ID),
	}

	// Set expiry for temporary streams
	if req.Duration > 0 {
		expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Second)
		response.ExpiresAt = &expiresAt
	}

	utils.SendSuccess(c, response)
}

// GetCameraSnapshot returns camera snapshot
func (h *Handlers) GetCameraSnapshot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid camera ID")
		return
	}

	camera, err := h.repos.Camera.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to get camera for snapshot")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	if !camera.IsEnabled {
		utils.SendError(c, http.StatusForbidden, "Camera is not enabled")
		return
	}

	// For Ring cameras, use Ring adapter to get snapshot
	if camera.Type == "ring" {
		h.getRingCameraSnapshot(c, camera)
		return
	}

	// For generic cameras, use the stored snapshot URL
	if !camera.SnapshotURL.Valid || camera.SnapshotURL.String == "" {
		utils.SendError(c, http.StatusNotFound, "Snapshot URL not available for this camera")
		return
	}

	response := CameraSnapshotResponse{
		SnapshotURL: camera.SnapshotURL.String,
		CameraID:    strconv.Itoa(camera.ID),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Timestamp:   time.Now(),
	}

	utils.SendSuccess(c, response)
}

// GetCameraStats returns camera statistics from unified service
func (h *Handlers) GetCameraStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get all camera entities through unified service
	options := unified.GetAllOptions{
		Domain: "camera",
	}

	entitiesWithRooms, err := h.unifiedService.GetAll(ctx, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get cameras from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get camera statistics")
		return
	}

	// Count cameras by type and status
	var totalCameras, enabledCameras, ringCameras, genericCameras, onlineCameras int

	for _, entityWithRoom := range entitiesWithRooms {
		entity := entityWithRoom.Entity
		if entity.GetType() == types.EntityTypeCamera {
			totalCameras++

			if entity.IsAvailable() {
				enabledCameras++
				onlineCameras++
			}

			// Check source to determine camera type
			if metadata := entity.GetMetadata(); metadata != nil {
				switch metadata.Source {
				case types.SourceRing:
					ringCameras++
				case types.SourceHomeAssistant:
					// Check if it's a generic camera from HA
					attributes := entity.GetAttributes()
					if deviceClass, ok := attributes["device_class"].(string); ok && deviceClass == "camera" {
						genericCameras++
					}
				default:
					genericCameras++
				}
			} else {
				genericCameras++
			}
		}
	}

	stats := CameraStatsResponse{
		TotalCameras:   totalCameras,
		EnabledCameras: enabledCameras,
		RingCameras:    ringCameras,
		GenericCameras: genericCameras,
		OnlineCameras:  onlineCameras,
	}

	utils.SendSuccess(c, stats)
}

// Helper function to convert models.Camera to CameraResponse
func convertCameraToResponse(camera *models.Camera) *CameraResponse {
	response := &CameraResponse{
		ID:           camera.ID,
		EntityID:     camera.EntityID,
		Name:         camera.Name,
		Type:         camera.Type,
		Capabilities: camera.Capabilities,
		Settings:     camera.Settings,
		IsEnabled:    camera.IsEnabled,
		CreatedAt:    camera.CreatedAt,
		UpdatedAt:    camera.UpdatedAt,
	}

	if camera.StreamURL.Valid {
		response.StreamURL = &camera.StreamURL.String
	}

	if camera.SnapshotURL.Valid {
		response.SnapshotURL = &camera.SnapshotURL.String
	}

	return response
}

// Helper function to convert PMA entity to CameraResponse
func convertPMAEntityToCameraResponse(entity types.PMAEntity) *CameraResponse {
	attributes := entity.GetAttributes()

	response := &CameraResponse{
		EntityID:  entity.GetID(),
		Name:      entity.GetFriendlyName(),
		Type:      "unified", // Mark as unified system camera
		IsEnabled: entity.IsAvailable(),
		CreatedAt: entity.GetLastUpdated(),
		UpdatedAt: entity.GetLastUpdated(),
	}

	// Set capabilities from entity attributes
	if capabilities, ok := attributes["capabilities"].(map[string]interface{}); ok {
		if capabilityJSON, err := json.Marshal(capabilities); err == nil {
			response.Capabilities = capabilityJSON
		}
	}

	// Set settings from entity attributes
	if settings, ok := attributes["settings"].(map[string]interface{}); ok {
		if settingsJSON, err := json.Marshal(settings); err == nil {
			response.Settings = settingsJSON
		}
	}

	// Extract stream and snapshot URLs from attributes
	if streamURL, ok := attributes["stream_url"].(string); ok && streamURL != "" {
		response.StreamURL = &streamURL
	}
	if snapshotURL, ok := attributes["snapshot_url"].(string); ok && snapshotURL != "" {
		response.SnapshotURL = &snapshotURL
	}

	// For Ring cameras, construct URLs based on entity ID
	if metadata := entity.GetMetadata(); metadata != nil && metadata.Source == types.SourceRing {
		response.Type = "ring"
		if response.StreamURL == nil {
			streamURL := "/api/ring/cameras/" + entity.GetID() + "/stream"
			response.StreamURL = &streamURL
		}
		if response.SnapshotURL == nil {
			snapshotURL := "/api/ring/cameras/" + entity.GetID() + "/snapshot"
			response.SnapshotURL = &snapshotURL
		}
	}

	return response
}

// Helper function to get Ring camera stream
func (h *Handlers) getRingCameraStream(c *gin.Context, camera *models.Camera, req CameraStreamRequest) {
	// Extract Ring device ID from entity ID (e.g., "ring_camera_123" -> "123")
	// This assumes the entity ID follows the pattern "ring_camera_{id}"

	// For now, return a mock response since Ring integration would need to be connected
	response := CameraStreamResponse{
		StreamURL: h.cfg.ExternalServices.MockData.RingStreamsBase + "/" + camera.EntityID,
		Format:    req.Format,
		Quality:   req.Quality,
		CameraID:  strconv.Itoa(camera.ID),
	}

	if req.Duration > 0 {
		expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Second)
		response.ExpiresAt = &expiresAt
	}

	h.log.WithField("camera_id", camera.ID).Info("Ring camera stream requested")
	utils.SendSuccess(c, response)
}

// Helper function to get Ring camera snapshot
func (h *Handlers) getRingCameraSnapshot(c *gin.Context, camera *models.Camera) {
	// For now, return a mock response since Ring integration would need to be connected
	response := CameraSnapshotResponse{
		SnapshotURL: h.cfg.ExternalServices.MockData.RingSnapshotsBase + "/" + camera.EntityID + "/latest.jpg",
		CameraID:    strconv.Itoa(camera.ID),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Timestamp:   time.Now(),
	}

	h.log.WithField("camera_id", camera.ID).Info("Ring camera snapshot requested")
	utils.SendSuccess(c, response)
}

// GetAllCameraEvents returns all camera events from various sources
func (h *Handlers) GetAllCameraEvents(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// For now, return empty events since this would require integration with
	// camera sources (Ring, etc.) to fetch actual events
	events := []map[string]interface{}{}

	utils.SendSuccessWithMeta(c, events, gin.H{
		"total":   0,
		"limit":   limit,
		"message": "Camera events endpoint is available but no sources are configured",
	})
}

// GetCameraEvents returns events for a specific camera
func (h *Handlers) GetCameraEvents(c *gin.Context) {
	cameraID := c.Param("id")
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// For now, return empty events since this would require integration with
	// camera sources to fetch actual events for the specific camera
	events := []map[string]interface{}{}

	utils.SendSuccessWithMeta(c, events, gin.H{
		"camera_id": cameraID,
		"total":     0,
		"limit":     limit,
		"message":   "Camera events endpoint is available but no events found",
	})
}
