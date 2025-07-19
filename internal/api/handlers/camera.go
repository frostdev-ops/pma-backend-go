package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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

// GetCameras returns all cameras
func (h *Handlers) GetCameras(c *gin.Context) {
	cameras, err := h.repos.Camera.GetAll(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve cameras")
		return
	}

	response := make([]*CameraResponse, len(cameras))
	for i, camera := range cameras {
		response[i] = convertCameraToResponse(camera)
	}

	utils.SendSuccess(c, response)
}

// GetEnabledCameras returns only enabled cameras
func (h *Handlers) GetEnabledCameras(c *gin.Context) {
	cameras, err := h.repos.Camera.GetEnabled(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get enabled cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve enabled cameras")
		return
	}

	response := make([]*CameraResponse, len(cameras))
	for i, camera := range cameras {
		response[i] = convertCameraToResponse(camera)
	}

	utils.SendSuccess(c, response)
}

// GetCamera returns a specific camera by ID
func (h *Handlers) GetCamera(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid camera ID")
		return
	}

	camera, err := h.repos.Camera.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to get camera")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	utils.SendSuccess(c, convertCameraToResponse(camera))
}

// GetCameraByEntityID returns a specific camera by entity ID
func (h *Handlers) GetCameraByEntityID(c *gin.Context) {
	entityID := c.Param("entityId")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	camera, err := h.repos.Camera.GetByEntityID(c.Request.Context(), entityID)
	if err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to get camera by entity ID")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	utils.SendSuccess(c, convertCameraToResponse(camera))
}

// CreateCamera creates a new camera
func (h *Handlers) CreateCamera(c *gin.Context) {
	var req CameraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Check if camera with entity ID already exists
	existing, _ := h.repos.Camera.GetByEntityID(c.Request.Context(), req.EntityID)
	if existing != nil {
		utils.SendError(c, http.StatusConflict, "Camera with this entity ID already exists")
		return
	}

	camera := &models.Camera{
		EntityID:     req.EntityID,
		Name:         req.Name,
		Type:         req.Type,
		Capabilities: req.Capabilities,
		Settings:     req.Settings,
		IsEnabled:    req.IsEnabled,
	}

	// Set stream URL if provided
	if req.StreamURL != nil {
		camera.StreamURL.String = *req.StreamURL
		camera.StreamURL.Valid = true
	}

	// Set snapshot URL if provided
	if req.SnapshotURL != nil {
		camera.SnapshotURL.String = *req.SnapshotURL
		camera.SnapshotURL.Valid = true
	}

	err := h.repos.Camera.Create(c.Request.Context(), camera)
	if err != nil {
		h.log.WithError(err).Error("Failed to create camera")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create camera")
		return
	}

	h.log.WithField("camera_id", camera.ID).Info("Camera created successfully")
	utils.SendSuccess(c, convertCameraToResponse(camera))
}

// UpdateCamera updates an existing camera
func (h *Handlers) UpdateCamera(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid camera ID")
		return
	}

	var req CameraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Get existing camera
	camera, err := h.repos.Camera.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to get camera for update")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	// Update fields
	camera.Name = req.Name
	camera.Type = req.Type
	camera.Capabilities = req.Capabilities
	camera.Settings = req.Settings
	camera.IsEnabled = req.IsEnabled

	// Update stream URL
	if req.StreamURL != nil {
		camera.StreamURL.String = *req.StreamURL
		camera.StreamURL.Valid = true
	} else {
		camera.StreamURL.Valid = false
	}

	// Update snapshot URL
	if req.SnapshotURL != nil {
		camera.SnapshotURL.String = *req.SnapshotURL
		camera.SnapshotURL.Valid = true
	} else {
		camera.SnapshotURL.Valid = false
	}

	err = h.repos.Camera.Update(c.Request.Context(), camera)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to update camera")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update camera")
		return
	}

	h.log.WithField("camera_id", id).Info("Camera updated successfully")
	utils.SendSuccess(c, convertCameraToResponse(camera))
}

// DeleteCamera deletes a camera
func (h *Handlers) DeleteCamera(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid camera ID")
		return
	}

	err = h.repos.Camera.Delete(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("camera_id", id).Error("Failed to delete camera")
		utils.SendError(c, http.StatusNotFound, "Camera not found")
		return
	}

	h.log.WithField("camera_id", id).Info("Camera deleted successfully")
	utils.SendSuccess(c, gin.H{"message": "Camera deleted successfully"})
}

// GetCamerasByType returns cameras by type
func (h *Handlers) GetCamerasByType(c *gin.Context) {
	cameraType := c.Param("type")
	if cameraType == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera type is required")
		return
	}

	cameras, err := h.repos.Camera.GetByType(c.Request.Context(), cameraType)
	if err != nil {
		h.log.WithError(err).WithField("camera_type", cameraType).Error("Failed to get cameras by type")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve cameras")
		return
	}

	response := make([]*CameraResponse, len(cameras))
	for i, camera := range cameras {
		response[i] = convertCameraToResponse(camera)
	}

	utils.SendSuccess(c, response)
}

// SearchCameras searches cameras by name or entity ID
func (h *Handlers) SearchCameras(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.SendError(c, http.StatusBadRequest, "Search query is required")
		return
	}

	cameras, err := h.repos.Camera.SearchCameras(c.Request.Context(), query)
	if err != nil {
		h.log.WithError(err).WithField("query", query).Error("Failed to search cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to search cameras")
		return
	}

	response := make([]*CameraResponse, len(cameras))
	for i, camera := range cameras {
		response[i] = convertCameraToResponse(camera)
	}

	utils.SendSuccess(c, response)
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

// GetCameraStats returns camera statistics
func (h *Handlers) GetCameraStats(c *gin.Context) {
	ctx := c.Request.Context()

	totalCameras, err := h.repos.Camera.CountCameras(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to count total cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get camera statistics")
		return
	}

	enabledCameras, err := h.repos.Camera.CountEnabledCameras(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to count enabled cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get camera statistics")
		return
	}

	ringCameras, err := h.repos.Camera.GetByType(ctx, "ring")
	if err != nil {
		h.log.WithError(err).Error("Failed to get Ring cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get camera statistics")
		return
	}

	genericCameras, err := h.repos.Camera.GetByType(ctx, "generic")
	if err != nil {
		h.log.WithError(err).Error("Failed to get generic cameras")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get camera statistics")
		return
	}

	// Count online cameras (enabled cameras for now)
	onlineCameras := enabledCameras

	stats := CameraStatsResponse{
		TotalCameras:   totalCameras,
		EnabledCameras: enabledCameras,
		RingCameras:    len(ringCameras),
		GenericCameras: len(genericCameras),
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
