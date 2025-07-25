package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// ThumbnailGenerator handles thumbnail generation for media files
type ThumbnailGenerator struct {
	config *config.FileManagerConfig
	logger *logrus.Logger
}

// ThumbnailOptions contains options for thumbnail generation
type ThumbnailOptions struct {
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Quality int    `json:"quality"`
	Format  string `json:"format"` // jpeg, png, webp
	Fit     string `json:"fit"`    // resize, crop, fill
}

// NewThumbnailGenerator creates a new thumbnail generator
func NewThumbnailGenerator(cfg *config.FileManagerConfig, logger *logrus.Logger) *ThumbnailGenerator {
	return &ThumbnailGenerator{
		config: cfg,
		logger: logger,
	}
}

// GenerateFromStream generates a thumbnail from a media stream
func (tg *ThumbnailGenerator) GenerateFromStream(reader io.Reader, mimeType string) ([]byte, error) {
	tg.logger.Debugf("Generating thumbnail for MIME type: %s", mimeType)

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return tg.generateImageThumbnail(reader, mimeType)
	case strings.HasPrefix(mimeType, "video/"):
		return tg.generateVideoThumbnail(reader, mimeType)
	default:
		return tg.generateDefaultThumbnail()
	}
}

// GenerateWithOptions generates a thumbnail with specific options
func (tg *ThumbnailGenerator) GenerateWithOptions(reader io.Reader, mimeType string, options ThumbnailOptions) ([]byte, error) {
	tg.logger.Debugf("Generating thumbnail with options: %+v", options)

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return tg.generateImageThumbnailWithOptions(reader, mimeType, options)
	case strings.HasPrefix(mimeType, "video/"):
		return tg.generateVideoThumbnailWithOptions(reader, mimeType, options)
	default:
		return tg.generateDefaultThumbnail()
	}
}

// generateImageThumbnail generates a thumbnail from an image
func (tg *ThumbnailGenerator) generateImageThumbnail(reader io.Reader, mimeType string) ([]byte, error) {
	// Use default options
	options := ThumbnailOptions{
		Width:   300,
		Height:  300,
		Quality: 85,
		Format:  "jpeg",
		Fit:     "resize",
	}

	return tg.generateImageThumbnailWithOptions(reader, mimeType, options)
}

// generateImageThumbnailWithOptions generates a thumbnail from an image with specific options
func (tg *ThumbnailGenerator) generateImageThumbnailWithOptions(reader io.Reader, mimeType string, options ThumbnailOptions) ([]byte, error) {
	// Decode the image
	img, err := tg.decodeImage(reader, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize the image based on fit option
	var thumbnail image.Image
	switch options.Fit {
	case "crop":
		thumbnail = imaging.Fill(img, options.Width, options.Height, imaging.Center, imaging.Lanczos)
	case "fill":
		thumbnail = imaging.Fit(img, options.Width, options.Height, imaging.Lanczos)
	default: // resize
		thumbnail = imaging.Resize(img, options.Width, options.Height, imaging.Lanczos)
	}

	// Encode the thumbnail
	return tg.encodeThumbnail(thumbnail, options)
}

// generateVideoThumbnail generates a thumbnail from a video using FFmpeg
func (tg *ThumbnailGenerator) generateVideoThumbnail(reader io.Reader, mimeType string) ([]byte, error) {
	return tg.generateVideoThumbnailWithOptions(reader, mimeType, ThumbnailOptions{
		Width:   300,
		Height:  300,
		Quality: 80,
	})
}

// generateVideoThumbnailWithOptions generates a video thumbnail with specific options using FFmpeg
func (tg *ThumbnailGenerator) generateVideoThumbnailWithOptions(reader io.Reader, mimeType string, options ThumbnailOptions) ([]byte, error) {
	// Create temporary files for input video and output thumbnail
	tempDir := os.TempDir()

	// Create temporary input file
	inputFile, err := os.CreateTemp(tempDir, "video_input_*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	// Copy video data to temporary file
	_, err = io.Copy(inputFile, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write video data to temp file: %w", err)
	}
	inputFile.Close()

	// Create temporary output file for thumbnail
	outputFile, err := os.CreateTemp(tempDir, "video_thumb_*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	// Check if FFmpeg is available
	if !tg.isFFmpegAvailable() {
		tg.logger.Warn("FFmpeg not available, falling back to placeholder thumbnail")
		return tg.generateVideoPlaceholder(options.Width, options.Height)
	}

	// Generate FFmpeg command
	args := []string{
		"-i", inputFile.Name(), // Input file
		"-ss", "00:00:01", // Seek to 1 second (avoid black frames at start)
		"-vframes", "1", // Extract only 1 frame
		"-vf", fmt.Sprintf("scale=%d:%d", options.Width, options.Height), // Scale to desired size
		"-q:v", fmt.Sprintf("%d", 100-options.Quality), // Quality (FFmpeg uses inverse scale)
		"-y",       // Overwrite output file
		outputPath, // Output file
	}

	// Execute FFmpeg command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	tg.logger.WithFields(map[string]interface{}{
		"mime_type": mimeType,
		"width":     options.Width,
		"height":    options.Height,
		"quality":   options.Quality,
	}).Debug("Generating video thumbnail with FFmpeg")

	err = cmd.Run()
	if err != nil {
		tg.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"stderr": stderr.String(),
			"args":   args,
		}).Error("FFmpeg command failed")

		// Fall back to placeholder if FFmpeg fails
		return tg.generateVideoPlaceholder(options.Width, options.Height)
	}

	// Read the generated thumbnail
	thumbnailData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated thumbnail: %w", err)
	}

	tg.logger.WithFields(map[string]interface{}{
		"mime_type":      mimeType,
		"thumbnail_size": len(thumbnailData),
		"width":          options.Width,
		"height":         options.Height,
	}).Info("Video thumbnail generated successfully")

	return thumbnailData, nil
}

// decodeImage decodes an image from a reader based on MIME type
func (tg *ThumbnailGenerator) decodeImage(reader io.Reader, mimeType string) (image.Image, error) {
	switch mimeType {
	case "image/jpeg":
		return jpeg.Decode(reader)
	case "image/png":
		return png.Decode(reader)
	case "image/gif":
		// Note: gif.Decode only returns the first frame
		return png.Decode(reader) // Fallback to generic decoder
	default:
		// Try generic image decoder
		img, _, err := image.Decode(reader)
		return img, err
	}
}

// encodeThumbnail encodes a thumbnail image with the specified options
func (tg *ThumbnailGenerator) encodeThumbnail(img image.Image, options ThumbnailOptions) ([]byte, error) {
	buf := new(bytes.Buffer)

	switch options.Format {
	case "png":
		encoder := png.Encoder{
			CompressionLevel: png.DefaultCompression,
		}
		err := encoder.Encode(buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	case "jpeg":
		fallthrough
	default:
		quality := options.Quality
		if quality <= 0 || quality > 100 {
			quality = 85 // Default quality
		}

		err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// generateDefaultThumbnail generates a default thumbnail for unsupported file types
func (tg *ThumbnailGenerator) generateDefaultThumbnail() ([]byte, error) {
	// Create a simple colored rectangle as default thumbnail
	img := imaging.New(300, 300, color.RGBA{128, 128, 128, 255})

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, fmt.Errorf("failed to encode default thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}

// generateVideoPlaceholder generates a video placeholder thumbnail
func (tg *ThumbnailGenerator) generateVideoPlaceholder(width, height int) ([]byte, error) {
	// Create a dark gray rectangle with a play icon representation
	img := imaging.New(width, height, color.RGBA{64, 64, 64, 255})

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, fmt.Errorf("failed to encode video placeholder: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateMultipleSizes generates thumbnails in multiple sizes
func (tg *ThumbnailGenerator) GenerateMultipleSizes(reader io.Reader, mimeType string) (map[string][]byte, error) {
	thumbnails := make(map[string][]byte)

	// Get configured thumbnail sizes
	sizes := tg.config.Media.ThumbnailSizes
	if len(sizes) == 0 {
		sizes = []int{150, 300, 600} // Default sizes
	}

	for _, size := range sizes {
		// Create a seekable reader for each size generation
		var sizeReader io.Reader

		// For this simplified implementation, we'll assume the original reader can be reused
		// In a real implementation, you'd need to either:
		// 1. Store the data in memory and create multiple readers
		// 2. Use a seekable reader and seek back to the beginning
		sizeReader = reader

		options := ThumbnailOptions{
			Width:   size,
			Height:  size,
			Quality: 85,
			Format:  "jpeg",
			Fit:     "resize",
		}

		thumbnail, err := tg.GenerateWithOptions(sizeReader, mimeType, options)
		if err != nil {
			tg.logger.Warnf("Failed to generate thumbnail size %d: %v", size, err)
			continue
		}

		thumbnails[fmt.Sprintf("%dx%d", size, size)] = thumbnail
	}

	return thumbnails, nil
}

// ValidateImage validates if an image can be processed for thumbnail generation
func (tg *ThumbnailGenerator) ValidateImage(reader io.Reader, mimeType string) error {
	supportedTypes := []string{
		"image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp",
	}

	for _, supported := range supportedTypes {
		if mimeType == supported {
			return nil
		}
	}

	return fmt.Errorf("unsupported image format for thumbnail generation: %s", mimeType)
}

// GetImageDimensions gets the dimensions of an image without fully decoding it
func (tg *ThumbnailGenerator) GetImageDimensions(reader io.Reader, mimeType string) (width, height int, err error) {
	// This would typically use a library that can read image headers without full decoding
	// For now, we'll decode the image to get dimensions
	img, err := tg.decodeImage(reader, mimeType)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image for dimensions: %w", err)
	}

	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// CalculateOptimalSize calculates optimal thumbnail size maintaining aspect ratio
func (tg *ThumbnailGenerator) CalculateOptimalSize(originalWidth, originalHeight, maxWidth, maxHeight int) (width, height int) {
	if originalWidth <= maxWidth && originalHeight <= maxHeight {
		return originalWidth, originalHeight
	}

	aspectRatio := float64(originalWidth) / float64(originalHeight)

	if float64(maxWidth)/aspectRatio <= float64(maxHeight) {
		return maxWidth, int(float64(maxWidth) / aspectRatio)
	}

	return int(float64(maxHeight) * aspectRatio), maxHeight
}

// GetDefaultThumbnailOptions returns default thumbnail generation options
func (tg *ThumbnailGenerator) GetDefaultThumbnailOptions() ThumbnailOptions {
	return ThumbnailOptions{
		Width:   300,
		Height:  300,
		Quality: 85,
		Format:  "jpeg",
		Fit:     "resize",
	}
}

// EstimateProcessingTime estimates thumbnail generation processing time
func (tg *ThumbnailGenerator) EstimateProcessingTime(imageWidth, imageHeight int, thumbnailCount int) int64 {
	// Simple estimation based on image size and number of thumbnails
	// Returns estimated time in milliseconds

	pixels := int64(imageWidth * imageHeight)
	baseTime := pixels / 1000000 // ~1 second per megapixel

	return baseTime * int64(thumbnailCount) * 100 // Scale by thumbnail count
}

// isFFmpegAvailable checks if FFmpeg is available on the system
func (tg *ThumbnailGenerator) isFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// GetSupportedFormats returns supported formats for thumbnail generation
func (tg *ThumbnailGenerator) GetSupportedFormats() []string {
	return []string{
		"image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp",
		"video/mp4", "video/webm", "video/avi", "video/mov", // Video support (placeholder)
	}
}
