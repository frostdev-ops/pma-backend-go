package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create cameras table
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS cameras (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			type TEXT DEFAULT 'generic',
			stream_url TEXT,
			snapshot_url TEXT,
			capabilities TEXT,
			settings TEXT,
			is_enabled BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

func TestCameraRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	capabilities := map[string]interface{}{
		"has_video":  true,
		"has_audio":  true,
		"has_motion": true,
	}
	capabilitiesJSON, _ := json.Marshal(capabilities)

	settings := map[string]interface{}{
		"motion_detection": true,
		"quality":          "high",
	}
	settingsJSON, _ := json.Marshal(settings)

	camera := &models.Camera{
		EntityID:     "test_camera_1",
		Name:         "Test Camera",
		Type:         "generic",
		Capabilities: capabilitiesJSON,
		Settings:     settingsJSON,
		IsEnabled:    true,
	}
	camera.StreamURL.String = "rtsp://test.example.com/stream"
	camera.StreamURL.Valid = true

	err := repo.Create(ctx, camera)
	if err != nil {
		t.Fatalf("Failed to create camera: %v", err)
	}

	if camera.ID == 0 {
		t.Error("Expected camera ID to be set after creation")
	}

	if camera.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}
}

func TestCameraRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	// Create a camera first
	capabilities := map[string]interface{}{"has_video": true}
	capabilitiesJSON, _ := json.Marshal(capabilities)

	settings := map[string]interface{}{"quality": "medium"}
	settingsJSON, _ := json.Marshal(settings)

	camera := &models.Camera{
		EntityID:     "test_camera_2",
		Name:         "Test Camera 2",
		Type:         "ring",
		Capabilities: capabilitiesJSON,
		Settings:     settingsJSON,
		IsEnabled:    true,
	}

	err := repo.Create(ctx, camera)
	if err != nil {
		t.Fatalf("Failed to create camera: %v", err)
	}

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, camera.ID)
	if err != nil {
		t.Fatalf("Failed to get camera by ID: %v", err)
	}

	if retrieved.EntityID != camera.EntityID {
		t.Errorf("Expected entity_id %s, got %s", camera.EntityID, retrieved.EntityID)
	}

	if retrieved.Name != camera.Name {
		t.Errorf("Expected name %s, got %s", camera.Name, retrieved.Name)
	}

	if retrieved.Type != camera.Type {
		t.Errorf("Expected type %s, got %s", camera.Type, retrieved.Type)
	}
}

func TestCameraRepository_GetByEntityID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	capabilities := map[string]interface{}{"has_motion": true}
	capabilitiesJSON, _ := json.Marshal(capabilities)

	settings := map[string]interface{}{"motion_detection": false}
	settingsJSON, _ := json.Marshal(settings)

	camera := &models.Camera{
		EntityID:     "unique_entity_id",
		Name:         "Entity Test Camera",
		Type:         "generic",
		Capabilities: capabilitiesJSON,
		Settings:     settingsJSON,
		IsEnabled:    false,
	}

	err := repo.Create(ctx, camera)
	if err != nil {
		t.Fatalf("Failed to create camera: %v", err)
	}

	// Test GetByEntityID
	retrieved, err := repo.GetByEntityID(ctx, "unique_entity_id")
	if err != nil {
		t.Fatalf("Failed to get camera by entity ID: %v", err)
	}

	if retrieved.ID != camera.ID {
		t.Errorf("Expected ID %d, got %d", camera.ID, retrieved.ID)
	}

	if retrieved.IsEnabled != camera.IsEnabled {
		t.Errorf("Expected is_enabled %v, got %v", camera.IsEnabled, retrieved.IsEnabled)
	}
}

func TestCameraRepository_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	// Create multiple cameras
	for i := 0; i < 3; i++ {
		capabilities := map[string]interface{}{"index": i}
		capabilitiesJSON, _ := json.Marshal(capabilities)

		settings := map[string]interface{}{"test": true}
		settingsJSON, _ := json.Marshal(settings)

		camera := &models.Camera{
			EntityID:     fmt.Sprintf("camera_%d", i),
			Name:         fmt.Sprintf("Camera %d", i),
			Type:         "test",
			Capabilities: capabilitiesJSON,
			Settings:     settingsJSON,
			IsEnabled:    true,
		}

		err := repo.Create(ctx, camera)
		if err != nil {
			t.Fatalf("Failed to create camera %d: %v", i, err)
		}
	}

	// Test GetAll
	cameras, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all cameras: %v", err)
	}

	if len(cameras) != 3 {
		t.Errorf("Expected 3 cameras, got %d", len(cameras))
	}

	// Cameras should be ordered by name
	for i, camera := range cameras {
		expectedName := fmt.Sprintf("Camera %d", i)
		if camera.Name != expectedName {
			t.Errorf("Expected camera %d name %s, got %s", i, expectedName, camera.Name)
		}
	}
}

func TestCameraRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	// Create a camera
	capabilities := map[string]interface{}{"original": true}
	capabilitiesJSON, _ := json.Marshal(capabilities)

	settings := map[string]interface{}{"original": "value"}
	settingsJSON, _ := json.Marshal(settings)

	camera := &models.Camera{
		EntityID:     "update_test",
		Name:         "Original Name",
		Type:         "generic",
		Capabilities: capabilitiesJSON,
		Settings:     settingsJSON,
		IsEnabled:    true,
	}

	err := repo.Create(ctx, camera)
	if err != nil {
		t.Fatalf("Failed to create camera: %v", err)
	}

	// Update the camera
	camera.Name = "Updated Name"
	camera.Type = "ring"
	camera.IsEnabled = false

	updatedCapabilities := map[string]interface{}{"updated": true}
	camera.Capabilities, _ = json.Marshal(updatedCapabilities)

	updatedSettings := map[string]interface{}{"updated": "new_value"}
	camera.Settings, _ = json.Marshal(updatedSettings)

	oldUpdatedAt := camera.UpdatedAt

	// Small delay to ensure updated_at changes
	time.Sleep(time.Millisecond * 10)

	err = repo.Update(ctx, camera)
	if err != nil {
		t.Fatalf("Failed to update camera: %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByID(ctx, camera.ID)
	if err != nil {
		t.Fatalf("Failed to get updated camera: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", retrieved.Name)
	}

	if retrieved.Type != "ring" {
		t.Errorf("Expected type 'ring', got %s", retrieved.Type)
	}

	if retrieved.IsEnabled != false {
		t.Errorf("Expected is_enabled false, got %v", retrieved.IsEnabled)
	}

	if !retrieved.UpdatedAt.After(oldUpdatedAt) {
		t.Error("Expected updated_at to be more recent")
	}
}

func TestCameraRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	// Create a camera
	capabilities := map[string]interface{}{"temp": true}
	capabilitiesJSON, _ := json.Marshal(capabilities)

	settings := map[string]interface{}{"temp": "value"}
	settingsJSON, _ := json.Marshal(settings)

	camera := &models.Camera{
		EntityID:     "delete_test",
		Name:         "To Be Deleted",
		Type:         "generic",
		Capabilities: capabilitiesJSON,
		Settings:     settingsJSON,
		IsEnabled:    true,
	}

	err := repo.Create(ctx, camera)
	if err != nil {
		t.Fatalf("Failed to create camera: %v", err)
	}

	// Delete the camera
	err = repo.Delete(ctx, camera.ID)
	if err != nil {
		t.Fatalf("Failed to delete camera: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, camera.ID)
	if err == nil {
		t.Error("Expected error when getting deleted camera")
	}
}

func TestCameraRepository_GetByType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	// Create cameras of different types
	types := []string{"ring", "ring", "generic", "ip"}
	for i, cameraType := range types {
		capabilities := map[string]interface{}{"test": i}
		capabilitiesJSON, _ := json.Marshal(capabilities)

		settings := map[string]interface{}{"test": i}
		settingsJSON, _ := json.Marshal(settings)

		camera := &models.Camera{
			EntityID:     fmt.Sprintf("type_test_%d", i),
			Name:         fmt.Sprintf("Type Test %d", i),
			Type:         cameraType,
			Capabilities: capabilitiesJSON,
			Settings:     settingsJSON,
			IsEnabled:    true,
		}

		err := repo.Create(ctx, camera)
		if err != nil {
			t.Fatalf("Failed to create camera %d: %v", i, err)
		}
	}

	// Test GetByType for "ring"
	ringCameras, err := repo.GetByType(ctx, "ring")
	if err != nil {
		t.Fatalf("Failed to get Ring cameras: %v", err)
	}

	if len(ringCameras) != 2 {
		t.Errorf("Expected 2 Ring cameras, got %d", len(ringCameras))
	}

	for _, camera := range ringCameras {
		if camera.Type != "ring" {
			t.Errorf("Expected camera type 'ring', got %s", camera.Type)
		}
	}
}

func TestCameraRepository_CountMethods(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCameraRepository(db)
	ctx := context.Background()

	// Create cameras with different enabled states
	enabledStates := []bool{true, true, false, true}
	for i, enabled := range enabledStates {
		capabilities := map[string]interface{}{"count_test": i}
		capabilitiesJSON, _ := json.Marshal(capabilities)

		settings := map[string]interface{}{"count_test": i}
		settingsJSON, _ := json.Marshal(settings)

		camera := &models.Camera{
			EntityID:     fmt.Sprintf("count_test_%d", i),
			Name:         fmt.Sprintf("Count Test %d", i),
			Type:         "test",
			Capabilities: capabilitiesJSON,
			Settings:     settingsJSON,
			IsEnabled:    enabled,
		}

		err := repo.Create(ctx, camera)
		if err != nil {
			t.Fatalf("Failed to create camera %d: %v", i, err)
		}
	}

	// Test CountCameras
	totalCount, err := repo.CountCameras(ctx)
	if err != nil {
		t.Fatalf("Failed to count total cameras: %v", err)
	}

	if totalCount != 4 {
		t.Errorf("Expected total count 4, got %d", totalCount)
	}

	// Test CountEnabledCameras
	enabledCount, err := repo.CountEnabledCameras(ctx)
	if err != nil {
		t.Fatalf("Failed to count enabled cameras: %v", err)
	}

	if enabledCount != 3 {
		t.Errorf("Expected enabled count 3, got %d", enabledCount)
	}
}
