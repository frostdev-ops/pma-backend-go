package camera

import (
	"context"
	"time"
)

// Camera represents a camera device for the service layer
type Camera struct {
	ID           int                    `json:"id"`
	EntityID     string                 `json:"entity_id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	StreamURL    *string                `json:"stream_url,omitempty"`
	SnapshotURL  *string                `json:"snapshot_url,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
	IsEnabled    bool                   `json:"is_enabled"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CameraCapabilities represents camera capabilities
type CameraCapabilities struct {
	HasVideo         bool     `json:"has_video"`
	HasAudio         bool     `json:"has_audio"`
	HasMotion        bool     `json:"has_motion"`
	HasLight         bool     `json:"has_light"`
	HasSiren         bool     `json:"has_siren"`
	HasSnapshot      bool     `json:"has_snapshot"`
	HasLiveStream    bool     `json:"has_live_stream"`
	HasRecording     bool     `json:"has_recording"`
	HasBattery       bool     `json:"has_battery"`
	SupportedFormats []string `json:"supported_formats,omitempty"`
}

// CameraSettings represents camera configuration settings
type CameraSettings struct {
	MotionDetection      bool                   `json:"motion_detection"`
	RecordingEnabled     bool                   `json:"recording_enabled"`
	NightVision          bool                   `json:"night_vision"`
	Quality              string                 `json:"quality"`
	StreamFormat         string                 `json:"stream_format"`
	NotificationSettings map[string]interface{} `json:"notification_settings,omitempty"`
	Privacy              map[string]interface{} `json:"privacy,omitempty"`
}

// RingDeviceInfo represents Ring device information
type RingDeviceInfo struct {
	ID               int                    `json:"id"`
	Description      string                 `json:"description"`
	Kind             string                 `json:"kind"`
	BatteryLife      *int                   `json:"battery_life"`
	MotionDetection  bool                   `json:"motion_detection"`
	StreamingEnabled bool                   `json:"streaming_enabled"`
	HasSubscription  bool                   `json:"has_subscription"`
	Features         map[string]interface{} `json:"features"`
	Settings         map[string]interface{} `json:"settings"`
}

// RingIntegration defines the interface for Ring camera integration
type RingIntegration interface {
	// Device discovery and management
	GetCameras(ctx context.Context) ([]*RingDeviceInfo, error)
	GetCamera(ctx context.Context, cameraID string) (*RingDeviceInfo, error)

	// Camera control
	GetSnapshot(ctx context.Context, cameraID string) ([]byte, error)
	GetStreamURL(ctx context.Context, cameraID string) (string, error)

	// Device features
	SetLights(ctx context.Context, cameraID string, enabled bool) error
	SetSiren(ctx context.Context, cameraID string, enabled bool) error

	// Status and health
	IsConnected() bool
	GetDeviceHealth(ctx context.Context, cameraID string) (map[string]interface{}, error)
}

// CameraRepository defines the interface for camera data access
type CameraRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, camera *Camera) error
	GetByID(ctx context.Context, id int) (*Camera, error)
	GetByEntityID(ctx context.Context, entityID string) (*Camera, error)
	GetAll(ctx context.Context) ([]*Camera, error)
	GetEnabled(ctx context.Context) ([]*Camera, error)
	Update(ctx context.Context, camera *Camera) error
	Delete(ctx context.Context, id int) error

	// Advanced queries
	GetByType(ctx context.Context, cameraType string) ([]*Camera, error)
	SearchCameras(ctx context.Context, query string) ([]*Camera, error)
	CountCameras(ctx context.Context) (int, error)
	CountEnabledCameras(ctx context.Context) (int, error)

	// Status and URL management
	UpdateStatus(ctx context.Context, id int, enabled bool) error
	UpdateStreamURL(ctx context.Context, id int, streamURL string) error
	UpdateSnapshotURL(ctx context.Context, id int, snapshotURL string) error
}

// Service defines the camera service interface
type Service interface {
	// Camera management
	RegisterRingCamera(ctx context.Context, ringDevice *RingDeviceInfo) (*Camera, error)
	RegisterGenericCamera(ctx context.Context, name, streamURL, snapshotURL string, capabilities CameraCapabilities, settings CameraSettings) (*Camera, error)

	// Ring integration
	SyncRingCameras(ctx context.Context) error
	DiscoverRingCameras(ctx context.Context) error

	// Camera operations
	GetCameraCapabilities(ctx context.Context, cameraID int) (*CameraCapabilities, error)
	GetCameraSettings(ctx context.Context, cameraID int) (*CameraSettings, error)
	UpdateCameraSettings(ctx context.Context, cameraID int, settings CameraSettings) error

	// Health and stats
	GetCameraHealth(ctx context.Context, cameraID int) (map[string]interface{}, error)
	GetCameraStats(ctx context.Context) (map[string]interface{}, error)
	ValidateCameraURL(ctx context.Context, url string) error
}
