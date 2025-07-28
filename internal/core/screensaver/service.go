package screensaver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/image/webp"
)

// Service manages screensaver images and configuration
type Service struct {
	repo   Repository
	logger *logrus.Logger
	config ScreensaverConfig
}

// NewService creates a new screensaver service
func NewService(repo Repository, logger *logrus.Logger, config ScreensaverConfig) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		config: config,
	}
}

// Initialize sets up the screensaver service
func (s *Service) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing ScreensaverService...")

	// Ensure images directory exists
	if err := os.MkdirAll(s.config.ImagesDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	s.logger.WithField("directory", s.config.ImagesDirectory).Info("ScreensaverService initialized successfully")
	return nil
}

// GetImages returns all screensaver images with storage info
func (s *Service) GetImages(ctx context.Context) (*ScreensaverImageListResponse, error) {
	images, err := s.repo.GetActiveImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get images: %w", err)
	}

	storageInfo, err := s.GetStorageInfo(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get storage info")
		// Continue without storage info
		storageInfo = &ScreensaverStorageInfo{}
	}

	return &ScreensaverImageListResponse{
		Images:  images,
		Total:   len(images),
		Storage: *storageInfo,
	}, nil
}

// GetImageByID returns a specific image by ID
func (s *Service) GetImageByID(ctx context.Context, id int) (*ScreensaverImage, error) {
	return s.repo.GetImageByID(ctx, id)
}

// GetImageByFilename returns a specific image by filename
func (s *Service) GetImageByFilename(ctx context.Context, filename string) (*ScreensaverImage, error) {
	return s.repo.GetImageByFilename(ctx, filename)
}

// UploadImages handles uploading multiple images from multipart form
func (s *Service) UploadImages(ctx context.Context, form *multipart.Form) (*ScreensaverUploadResponse, error) {
	files := form.File["images"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	response := &ScreensaverUploadResponse{
		UploadedImages: []ScreensaverImage{},
		Errors:         []string{},
	}

	// Check storage before starting uploads
	storageInfo, err := s.GetStorageInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check storage: %w", err)
	}

	if !storageInfo.Recommendations.CanUpload {
		return nil, fmt.Errorf("storage limit reached, cannot upload more images")
	}

	for _, fileHeader := range files {
		// Check individual file size
		if fileHeader.Size > s.config.MaxImageSize {
			response.FailedCount++
			response.Errors = append(response.Errors,
				fmt.Sprintf("%s: file too large (%d bytes > %d bytes)",
					fileHeader.Filename, fileHeader.Size, s.config.MaxImageSize))
			continue
		}

		// Process the file
		image, err := s.processUploadedFile(ctx, fileHeader)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors,
				fmt.Sprintf("%s: %v", fileHeader.Filename, err))
			continue
		}

		response.UploadedImages = append(response.UploadedImages, *image)
		response.UploadedCount++
		response.TotalSize += image.FileSize
	}

	if response.UploadedCount > 0 {
		response.Message = fmt.Sprintf("Successfully uploaded %d image(s)", response.UploadedCount)
		if response.FailedCount > 0 {
			response.Message += fmt.Sprintf(" (%d failed)", response.FailedCount)
		}
	} else {
		response.Message = "No images were uploaded"
	}

	return response, nil
}

// processUploadedFile handles processing a single uploaded file
func (s *Service) processUploadedFile(ctx context.Context, fileHeader *multipart.FileHeader) (*ScreensaverImage, error) {
	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate checksum
	checksum := s.calculateChecksum(data)

	// Check if image already exists
	existing, err := s.repo.GetImageByChecksum(ctx, checksum)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("image already exists (duplicate)")
	}

	// Validate content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = s.detectContentType(data)
	}

	if !s.isValidContentType(contentType) {
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	// Get image dimensions
	width, height, err := s.getImageDimensions(data, contentType)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get image dimensions")
		// Continue without dimensions
		width, height = 0, 0
	}

	// Generate unique filename
	filename := s.generateUniqueFilename(fileHeader.Filename, checksum)

	// Save file to disk
	filePath := filepath.Join(s.config.ImagesDirectory, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create database record
	screensaverImage := &ScreensaverImage{
		Filename:     filename,
		OriginalName: fileHeader.Filename,
		ContentType:  contentType,
		FileSize:     fileHeader.Size,
		Width:        width,
		Height:       height,
		Checksum:     checksum,
		UploadedAt:   time.Now(),
		UploadedBy:   "web_upload", // Could be enhanced to use actual user info
		Tags:         "[]",         // Empty JSON array
		Active:       true,
	}

	if err := s.repo.CreateImage(ctx, screensaverImage); err != nil {
		// Clean up the file if database insert fails
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save image record: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"filename": filename,
		"size":     fileHeader.Size,
		"type":     contentType,
	}).Info("Screensaver image uploaded successfully")

	return screensaverImage, nil
}

// DeleteImage deletes an image by ID
func (s *Service) DeleteImage(ctx context.Context, id int) error {
	// Get the image record first
	image, err := s.repo.GetImageByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	// Delete from database
	if err := s.repo.DeleteImage(ctx, id); err != nil {
		return fmt.Errorf("failed to delete image record: %w", err)
	}

	// Delete file from disk
	filePath := filepath.Join(s.config.ImagesDirectory, image.Filename)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		s.logger.WithError(err).WithField("filename", image.Filename).Warn("Failed to delete image file")
		// Don't return error - database record is already deleted
	}

	s.logger.WithField("filename", image.Filename).Info("Screensaver image deleted successfully")
	return nil
}

// GetStorageInfo returns storage information and recommendations
func (s *Service) GetStorageInfo(ctx context.Context) (*ScreensaverStorageInfo, error) {
	// Get image statistics
	totalCount, err := s.repo.GetTotalImageCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get image count: %w", err)
	}

	totalSize, err := s.repo.GetTotalImageSize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total size: %w", err)
	}

	// Get disk information
	diskInfo, err := s.getDiskInfo()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get disk info")
		// Use default values
		diskInfo = ScreensaverDiskInfo{
			Total: s.config.MaxTotalSize,
			Free:  s.config.MaxTotalSize - totalSize,
		}
	}

	// Calculate recommendations
	recommendedMaxSize := int64(float64(diskInfo.Total) * 0.20) // 20% of disk space
	if recommendedMaxSize > s.config.MaxTotalSize {
		recommendedMaxSize = s.config.MaxTotalSize
	}

	currentUsagePercentage := float64(totalSize) / float64(diskInfo.Total) * 100
	remainingSpace := recommendedMaxSize - totalSize
	canUpload := remainingSpace > 0 && totalSize < recommendedMaxSize
	maxUploadSize := recommendedMaxSize - totalSize
	if maxUploadSize < 0 {
		maxUploadSize = 0
	}

	return &ScreensaverStorageInfo{
		TotalImagesCount: totalCount,
		TotalImagesSize:  totalSize,
		DiskInfo:         diskInfo,
		Recommendations: ScreensaverRecommendations{
			RecommendedMaxSize:     recommendedMaxSize,
			CurrentUsagePercentage: currentUsagePercentage,
			RemainingSpace:         remainingSpace,
			CanUpload:              canUpload,
			MaxUploadSize:          maxUploadSize,
		},
	}, nil
}

// ServeImage serves an image file by filename
func (s *Service) ServeImage(ctx context.Context, filename string) ([]byte, string, error) {
	// Validate the image exists in database
	image, err := s.repo.GetImageByFilename(ctx, filename)
	if err != nil {
		return nil, "", fmt.Errorf("image not found: %w", err)
	}

	// Read file from disk
	filePath := filepath.Join(s.config.ImagesDirectory, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image file: %w", err)
	}

	return data, image.ContentType, nil
}

// CleanupOrphanedImages removes database records for images that don't exist on disk
func (s *Service) CleanupOrphanedImages(ctx context.Context) error {
	orphaned, err := s.repo.GetOrphanedImages(ctx, s.config.ImagesDirectory)
	if err != nil {
		return fmt.Errorf("failed to get orphaned images: %w", err)
	}

	if len(orphaned) == 0 {
		return nil
	}

	var ids []int
	for _, image := range orphaned {
		ids = append(ids, image.ID)
	}

	if err := s.repo.DeleteImagesByIDs(ctx, ids); err != nil {
		return fmt.Errorf("failed to delete orphaned images: %w", err)
	}

	s.logger.WithField("count", len(orphaned)).Info("Cleaned up orphaned image records")
	return nil
}

// Helper methods

func (s *Service) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *Service) detectContentType(data []byte) string {
	// Simple content type detection based on file headers
	if len(data) < 4 {
		return "application/octet-stream"
	}

	switch {
	case data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "image/png"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "image/gif"
	case data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46:
		// Could be WebP (need to check further)
		if len(data) >= 12 && string(data[8:12]) == "WEBP" {
			return "image/webp"
		}
	}

	return "application/octet-stream"
}

func (s *Service) isValidContentType(contentType string) bool {
	for _, validType := range s.config.SupportedFormats {
		if contentType == validType {
			return true
		}
	}
	return false
}

func (s *Service) getImageDimensions(data []byte, contentType string) (int, int, error) {
	reader := strings.NewReader(string(data))

	switch contentType {
	case "image/jpeg":
		img, err := jpeg.Decode(reader)
		if err != nil {
			return 0, 0, err
		}
		bounds := img.Bounds()
		return bounds.Dx(), bounds.Dy(), nil

	case "image/png":
		img, err := png.Decode(reader)
		if err != nil {
			return 0, 0, err
		}
		bounds := img.Bounds()
		return bounds.Dx(), bounds.Dy(), nil

	case "image/gif":
		img, err := gif.Decode(reader)
		if err != nil {
			return 0, 0, err
		}
		bounds := img.Bounds()
		return bounds.Dx(), bounds.Dy(), nil

	case "image/webp":
		img, err := webp.Decode(reader)
		if err != nil {
			return 0, 0, err
		}
		bounds := img.Bounds()
		return bounds.Dx(), bounds.Dy(), nil

	default:
		// Try generic image decode
		reader := strings.NewReader(string(data))
		img, _, err := image.Decode(reader)
		if err != nil {
			return 0, 0, err
		}
		bounds := img.Bounds()
		return bounds.Dx(), bounds.Dy(), nil
	}
}

func (s *Service) generateUniqueFilename(originalName, checksum string) string {
	ext := filepath.Ext(originalName)
	if ext == "" {
		ext = ".jpg" // default extension
	}

	// Use first 16 characters of checksum + timestamp for uniqueness
	timestamp := time.Now().Unix()
	return fmt.Sprintf("screensaver_%s_%d%s", checksum[:16], timestamp, ext)
}

func (s *Service) getDiskInfo() (ScreensaverDiskInfo, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(s.config.ImagesDirectory, &stat)
	if err != nil {
		return ScreensaverDiskInfo{}, err
	}

	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free
	percentage := float64(used) / float64(total) * 100

	return ScreensaverDiskInfo{
		Total:      total,
		Used:       used,
		Free:       free,
		Percentage: percentage,
	}, nil
}
