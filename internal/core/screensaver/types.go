package screensaver

import (
	"context"
	"time"
)

// ScreensaverImage represents a screensaver image with metadata
type ScreensaverImage struct {
	ID           int       `json:"id" db:"id"`
	Filename     string    `json:"filename" db:"filename"`
	OriginalName string    `json:"original_name" db:"original_name"`
	ContentType  string    `json:"content_type" db:"content_type"`
	FileSize     int64     `json:"file_size" db:"file_size"`
	Width        int       `json:"width" db:"width"`
	Height       int       `json:"height" db:"height"`
	Checksum     string    `json:"checksum" db:"checksum"`
	UploadedAt   time.Time `json:"uploaded_at" db:"uploaded_at"`
	UploadedBy   string    `json:"uploaded_by" db:"uploaded_by"`
	Tags         string    `json:"tags" db:"tags"` // JSON array of tags
	Active       bool      `json:"active" db:"active"`
}

// ScreensaverImageRequest represents an image upload request
type ScreensaverImageRequest struct {
	Images []ScreensaverImageFile `json:"images"`
}

// ScreensaverImageFile represents a single image file in upload
type ScreensaverImageFile struct {
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type"`
	Data        []byte   `json:"data"`
	Tags        []string `json:"tags,omitempty"`
}

// ScreensaverStorageInfo represents storage information and recommendations
type ScreensaverStorageInfo struct {
	TotalImagesCount int                        `json:"totalImagesCount"`
	TotalImagesSize  int64                      `json:"totalImagesSize"`
	DiskInfo         ScreensaverDiskInfo        `json:"diskInfo"`
	Recommendations  ScreensaverRecommendations `json:"recommendations"`
}

// ScreensaverDiskInfo represents disk usage information
type ScreensaverDiskInfo struct {
	Total      int64   `json:"total"`
	Used       int64   `json:"used"`
	Free       int64   `json:"free"`
	Percentage float64 `json:"percentage"`
}

// ScreensaverRecommendations represents storage recommendations
type ScreensaverRecommendations struct {
	RecommendedMaxSize     int64   `json:"recommendedMaxSize"`
	CurrentUsagePercentage float64 `json:"currentUsagePercentage"`
	RemainingSpace         int64   `json:"remainingSpace"`
	CanUpload              bool    `json:"canUpload"`
	MaxUploadSize          int64   `json:"maxUploadSize"`
}

// ScreensaverImageListResponse represents the response for listing images
type ScreensaverImageListResponse struct {
	Images  []ScreensaverImage     `json:"images"`
	Total   int                    `json:"total"`
	Storage ScreensaverStorageInfo `json:"storage"`
}

// ScreensaverUploadResponse represents the response after uploading images
type ScreensaverUploadResponse struct {
	Message        string             `json:"message"`
	UploadedCount  int                `json:"uploadedCount"`
	FailedCount    int                `json:"failedCount"`
	TotalSize      int64              `json:"totalSize"`
	UploadedImages []ScreensaverImage `json:"uploadedImages"`
	Errors         []string           `json:"errors,omitempty"`
}

// ScreensaverConfig represents screensaver configuration
type ScreensaverConfig struct {
	ImagesDirectory    string   `json:"images_directory"`
	MaxImageSize       int64    `json:"max_image_size"` // bytes per image
	MaxTotalSize       int64    `json:"max_total_size"` // total storage limit
	SupportedFormats   []string `json:"supported_formats"`
	CompressionEnabled bool     `json:"compression_enabled"`
	CompressionQuality int      `json:"compression_quality"`
}

// Constants for screensaver configuration
const (
	DefaultMaxImageSize       = 100 * 1024 * 1024      // 100MB per image
	DefaultMaxTotalSize       = 2 * 1024 * 1024 * 1024 // 2GB total
	DefaultCompressionQuality = 85
)

var (
	DefaultSupportedFormats = []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
)

// Repository defines the interface for screensaver image database operations
type Repository interface {
	// Image management
	CreateImage(ctx context.Context, image *ScreensaverImage) error
	GetImageByID(ctx context.Context, id int) (*ScreensaverImage, error)
	GetImageByFilename(ctx context.Context, filename string) (*ScreensaverImage, error)
	GetImageByChecksum(ctx context.Context, checksum string) (*ScreensaverImage, error)
	GetAllImages(ctx context.Context) ([]ScreensaverImage, error)
	GetActiveImages(ctx context.Context) ([]ScreensaverImage, error)
	UpdateImage(ctx context.Context, image *ScreensaverImage) error
	DeleteImage(ctx context.Context, id int) error
	DeleteImageByFilename(ctx context.Context, filename string) error

	// Bulk operations
	GetImagesByIDs(ctx context.Context, ids []int) ([]ScreensaverImage, error)
	DeleteImagesByIDs(ctx context.Context, ids []int) error
	SetImagesActive(ctx context.Context, ids []int, active bool) error

	// Storage statistics
	GetTotalImageCount(ctx context.Context) (int, error)
	GetActiveImageCount(ctx context.Context) (int, error)
	GetTotalImageSize(ctx context.Context) (int64, error)
	GetActiveImageSize(ctx context.Context) (int64, error)
	GetImageCountByContentType(ctx context.Context, contentType string) (int, error)

	// Cleanup operations
	DeleteInactiveImages(ctx context.Context) error
	GetOrphanedImages(ctx context.Context, imageDirectory string) ([]ScreensaverImage, error)
}
