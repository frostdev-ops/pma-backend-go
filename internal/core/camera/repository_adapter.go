package camera

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// RepositoryAdapter adapts the database repository to the camera service interface
type RepositoryAdapter struct {
	repo repositories.CameraRepository
}

// NewRepositoryAdapter creates a new repository adapter
func NewRepositoryAdapter(repo repositories.CameraRepository) CameraRepository {
	return &RepositoryAdapter{repo: repo}
}

// Create creates a new camera
func (a *RepositoryAdapter) Create(ctx context.Context, camera *Camera) error {
	modelCamera := a.cameraToModel(camera)
	err := a.repo.Create(ctx, modelCamera)
	if err != nil {
		return err
	}

	// Update the camera with the generated ID and timestamps
	camera.ID = modelCamera.ID
	camera.CreatedAt = modelCamera.CreatedAt
	camera.UpdatedAt = modelCamera.UpdatedAt

	return nil
}

// GetByID retrieves a camera by ID
func (a *RepositoryAdapter) GetByID(ctx context.Context, id int) (*Camera, error) {
	modelCamera, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return a.modelToCamera(modelCamera), nil
}

// GetByEntityID retrieves a camera by entity ID
func (a *RepositoryAdapter) GetByEntityID(ctx context.Context, entityID string) (*Camera, error) {
	modelCamera, err := a.repo.GetByEntityID(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return a.modelToCamera(modelCamera), nil
}

// GetAll retrieves all cameras
func (a *RepositoryAdapter) GetAll(ctx context.Context) ([]*Camera, error) {
	modelCameras, err := a.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	cameras := make([]*Camera, len(modelCameras))
	for i, modelCamera := range modelCameras {
		cameras[i] = a.modelToCamera(modelCamera)
	}

	return cameras, nil
}

// GetEnabled retrieves all enabled cameras
func (a *RepositoryAdapter) GetEnabled(ctx context.Context) ([]*Camera, error) {
	modelCameras, err := a.repo.GetEnabled(ctx)
	if err != nil {
		return nil, err
	}

	cameras := make([]*Camera, len(modelCameras))
	for i, modelCamera := range modelCameras {
		cameras[i] = a.modelToCamera(modelCamera)
	}

	return cameras, nil
}

// Update updates an existing camera
func (a *RepositoryAdapter) Update(ctx context.Context, camera *Camera) error {
	modelCamera := a.cameraToModel(camera)
	err := a.repo.Update(ctx, modelCamera)
	if err != nil {
		return err
	}

	// Update the camera with the new timestamp
	camera.UpdatedAt = modelCamera.UpdatedAt

	return nil
}

// Delete deletes a camera by ID
func (a *RepositoryAdapter) Delete(ctx context.Context, id int) error {
	return a.repo.Delete(ctx, id)
}

// GetByType retrieves cameras by type
func (a *RepositoryAdapter) GetByType(ctx context.Context, cameraType string) ([]*Camera, error) {
	modelCameras, err := a.repo.GetByType(ctx, cameraType)
	if err != nil {
		return nil, err
	}

	cameras := make([]*Camera, len(modelCameras))
	for i, modelCamera := range modelCameras {
		cameras[i] = a.modelToCamera(modelCamera)
	}

	return cameras, nil
}

// SearchCameras searches cameras by name or entity ID
func (a *RepositoryAdapter) SearchCameras(ctx context.Context, query string) ([]*Camera, error) {
	modelCameras, err := a.repo.SearchCameras(ctx, query)
	if err != nil {
		return nil, err
	}

	cameras := make([]*Camera, len(modelCameras))
	for i, modelCamera := range modelCameras {
		cameras[i] = a.modelToCamera(modelCamera)
	}

	return cameras, nil
}

// CountCameras returns the total number of cameras
func (a *RepositoryAdapter) CountCameras(ctx context.Context) (int, error) {
	return a.repo.CountCameras(ctx)
}

// CountEnabledCameras returns the number of enabled cameras
func (a *RepositoryAdapter) CountEnabledCameras(ctx context.Context) (int, error) {
	return a.repo.CountEnabledCameras(ctx)
}

// UpdateStatus updates camera enabled status
func (a *RepositoryAdapter) UpdateStatus(ctx context.Context, id int, enabled bool) error {
	return a.repo.UpdateStatus(ctx, id, enabled)
}

// UpdateStreamURL updates camera stream URL
func (a *RepositoryAdapter) UpdateStreamURL(ctx context.Context, id int, streamURL string) error {
	return a.repo.UpdateStreamURL(ctx, id, streamURL)
}

// UpdateSnapshotURL updates camera snapshot URL
func (a *RepositoryAdapter) UpdateSnapshotURL(ctx context.Context, id int, snapshotURL string) error {
	return a.repo.UpdateSnapshotURL(ctx, id, snapshotURL)
}

// Helper methods for conversion

func (a *RepositoryAdapter) cameraToModel(camera *Camera) *models.Camera {
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
		modelCamera.StreamURL = sql.NullString{
			String: *camera.StreamURL,
			Valid:  true,
		}
	}

	if camera.SnapshotURL != nil {
		modelCamera.SnapshotURL = sql.NullString{
			String: *camera.SnapshotURL,
			Valid:  true,
		}
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

func (a *RepositoryAdapter) modelToCamera(modelCamera *models.Camera) *Camera {
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
		if err := json.Unmarshal(modelCamera.Capabilities, &camera.Capabilities); err == nil {
			// Successfully unmarshaled
		} else {
			camera.Capabilities = make(map[string]interface{})
		}
	} else {
		camera.Capabilities = make(map[string]interface{})
	}

	if len(modelCamera.Settings) > 0 {
		if err := json.Unmarshal(modelCamera.Settings, &camera.Settings); err == nil {
			// Successfully unmarshaled
		} else {
			camera.Settings = make(map[string]interface{})
		}
	} else {
		camera.Settings = make(map[string]interface{})
	}

	return camera
}
