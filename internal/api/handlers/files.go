package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/filemanager"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ImageInfo represents uploaded image metadata
type ImageInfo struct {
	ID           string    `json:"id" db:"id"`
	Filename     string    `json:"filename" db:"filename"`
	OriginalName string    `json:"originalName" db:"original_name"`
	Size         int64     `json:"size" db:"size"`
	MimeType     string    `json:"mimeType" db:"mime_type"`
	Width        int       `json:"width,omitempty" db:"width"`
	Height       int       `json:"height,omitempty" db:"height"`
	Hash         string    `json:"hash,omitempty" db:"hash"`
	UploadedAt   time.Time `json:"uploadedAt" db:"uploaded_at"`
}

// StorageInfo represents storage information
type StorageInfo struct {
	TotalImagesCount int                    `json:"totalImagesCount"`
	TotalImagesSize  int64                  `json:"totalImagesSize"`
	DiskInfo         DiskInfo               `json:"diskInfo"`
	Recommendations  StorageRecommendations `json:"recommendations"`
}

// DiskInfo represents disk usage information
type DiskInfo struct {
	Total      int64   `json:"total"`
	Used       int64   `json:"used"`
	Free       int64   `json:"free"`
	Percentage float64 `json:"percentage"`
}

// StorageRecommendations represents storage recommendations
type StorageRecommendations struct {
	RecommendedMaxSize     int64   `json:"recommendedMaxSize"`
	CurrentUsagePercentage float64 `json:"currentUsagePercentage"`
	RemainingSpace         int64   `json:"remainingSpace"`
	CanUpload              bool    `json:"canUpload"`
}

// FileHandler handles file operations
type FileHandler struct {
	log               *log.Logger
	uploadsDir        string
	maxFileSize       int64
	maxFiles          int
	allowedMimeTypes  map[string]bool
	allowedExtensions map[string]bool
	images            []ImageInfo // In-memory storage for demo - use database in production
	securityManager   *filemanager.SecurityManager
}

// NewFileHandler creates a new file handler
func NewFileHandler(logger *log.Logger, uploadsDir string, securityManager *filemanager.SecurityManager) *FileHandler {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		logger.Printf("Warning: Could not create uploads directory: %v", err)
	}

	return &FileHandler{
		log:         logger,
		uploadsDir:  uploadsDir,
		maxFileSize: 50 * 1024 * 1024, // 50MB
		maxFiles:    10,
		allowedMimeTypes: map[string]bool{
			"image/jpeg": true,
			"image/jpg":  true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
		},
		allowedExtensions: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		},
		images:          make([]ImageInfo, 0),
		securityManager: securityManager,
	}
}

// GetScreensaverImages returns all uploaded screensaver images
func (h *FileHandler) GetScreensaverImages(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      h.images,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetScreensaverStorage returns storage information
func (h *FileHandler) GetScreensaverStorage(c *gin.Context) {
	diskInfo, err := h.getDiskInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get disk information: " + err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	var totalSize int64
	for _, img := range h.images {
		totalSize += img.Size
	}

	// Calculate 20% of total disk space as recommended max for screensaver images
	recommendedMaxSize := int64(float64(diskInfo.Total) * 0.2)
	currentUsagePercentage := float64(totalSize) / float64(diskInfo.Total) * 100
	remainingSpace := recommendedMaxSize - totalSize
	if remainingSpace < 0 {
		remainingSpace = 0
	}

	// Can upload if there's at least 100MB free space remaining
	canUpload := remainingSpace > 100*1024*1024

	storageInfo := StorageInfo{
		TotalImagesCount: len(h.images),
		TotalImagesSize:  totalSize,
		DiskInfo:         diskInfo,
		Recommendations: StorageRecommendations{
			RecommendedMaxSize:     recommendedMaxSize,
			CurrentUsagePercentage: currentUsagePercentage,
			RemainingSpace:         remainingSpace,
			CanUpload:              canUpload,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      storageInfo,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// UploadScreensaverImages handles image upload
func (h *FileHandler) UploadScreensaverImages(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(h.maxFileSize * int64(h.maxFiles)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Failed to parse form: " + err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	files := c.Request.MultipartForm.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "No images uploaded",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	if len(files) > h.maxFiles {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     fmt.Sprintf("Too many files. Maximum %d files allowed", h.maxFiles),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	uploadedImages := []ImageInfo{}
	failedUploads := []map[string]string{}

	for _, fileHeader := range files {
		imageInfo, err := h.processUploadedFile(fileHeader)
		if err != nil {
			h.log.Printf("Failed to process file %s: %v", fileHeader.Filename, err)
			failedUploads = append(failedUploads, map[string]string{
				"filename": fileHeader.Filename,
				"error":    err.Error(),
			})
			continue
		}

		uploadedImages = append(uploadedImages, *imageInfo)
		h.images = append(h.images, *imageInfo)
	}

	// Prepare response
	if len(uploadedImages) == 0 {
		errorMessages := make([]string, len(failedUploads))
		for i, failed := range failedUploads {
			errorMessages[i] = fmt.Sprintf("%s: %s", failed["filename"], failed["error"])
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "All uploads failed. " + strings.Join(errorMessages, ", "),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	responseData := gin.H{
		"images":        uploadedImages,
		"uploadedCount": len(uploadedImages),
		"failedCount":   len(failedUploads),
	}

	if len(failedUploads) > 0 {
		responseData["message"] = fmt.Sprintf("Uploaded %d images successfully, %d failed",
			len(uploadedImages), len(failedUploads))
		responseData["failures"] = failedUploads
	} else {
		responseData["message"] = fmt.Sprintf("All %d images uploaded successfully", len(uploadedImages))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      responseData,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// DeleteScreensaverImage deletes an uploaded image
func (h *FileHandler) DeleteScreensaverImage(c *gin.Context) {
	imageID := c.Param("id")
	if imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Image ID is required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Find image
	var imageIndex = -1
	var imageInfo ImageInfo
	for i, img := range h.images {
		if img.ID == imageID {
			imageIndex = i
			imageInfo = img
			break
		}
	}

	if imageIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Image not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Delete file from disk
	filePath := filepath.Join(h.uploadsDir, imageInfo.Filename)
	if err := os.Remove(filePath); err != nil {
		h.log.Printf("Warning: Could not delete file %s: %v", filePath, err)
	}

	// Remove from slice
	h.images = append(h.images[:imageIndex], h.images[imageIndex+1:]...)

	h.log.Printf("Deleted image: %s (%s)", imageInfo.OriginalName, imageID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Image deleted successfully",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetScreensaverImage serves an uploaded image
func (h *FileHandler) GetScreensaverImage(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Filename is required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Find image info for mime type
	var imageInfo *ImageInfo
	for _, img := range h.images {
		if img.Filename == filename {
			imageInfo = &img
			break
		}
	}

	filePath := filepath.Join(h.uploadsDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Image not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Set content type if we have image info
	if imageInfo != nil && imageInfo.MimeType != "" {
		c.Header("Content-Type", imageInfo.MimeType)
	}

	// Set cache headers
	c.Header("Cache-Control", "public, max-age=86400") // 24 hours

	c.File(filePath)
}

// processUploadedFile processes a single uploaded file
func (h *FileHandler) processUploadedFile(fileHeader *multipart.FileHeader) (*ImageInfo, error) {
	// Validate file size
	if fileHeader.Size > h.maxFileSize {
		return nil, fmt.Errorf("file too large (max %d MB)", h.maxFileSize/(1024*1024))
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !h.allowedExtensions[ext] {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Open uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer src.Close()

	// Read file content for validation and hash calculation
	content, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Validate MIME type
	mimeType := http.DetectContentType(content)
	if !h.allowedMimeTypes[mimeType] {
		return nil, fmt.Errorf("unsupported MIME type: %s", mimeType)
	}

	// Security validation
	if h.securityManager != nil {
		securityConfig := filemanager.SecurityConfig{
			EncryptionEnabled: false,
			VirusScanEnabled:  true,
			AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
			BlockedExtensions: []string{".exe", ".bat", ".com", ".scr", ".pif", ".cmd"},
			MaxFileSize:       h.maxFileSize,
			ScanOnUpload:      true,
		}

		contentReader := strings.NewReader(string(content))
		if err := h.securityManager.ValidateFileUpload(fileHeader.Filename, contentReader, securityConfig); err != nil {
			h.log.Printf("File upload blocked by security validation: %s - %v", fileHeader.Filename, err)
			return nil, fmt.Errorf("security validation failed: %v", err)
		}
	}

	// Validate that it's actually an image by trying to decode it
	_, format, err := image.DecodeConfig(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("invalid image file: %v", err)
	}

	// Calculate file hash
	hash := md5.Sum(content)
	hashStr := hex.EncodeToString(hash[:])

	// Check for duplicates
	for _, img := range h.images {
		if img.Hash == hashStr {
			return nil, fmt.Errorf("duplicate image")
		}
	}

	// Generate unique filename
	imageID := uuid.New().String()
	filename := fmt.Sprintf("%s%s", imageID, ext)
	filePath := filepath.Join(h.uploadsDir, filename)

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	if _, err := dst.Write(content); err != nil {
		os.Remove(filePath) // Cleanup on error
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	// Get image dimensions
	img, _, err := image.Decode(strings.NewReader(string(content)))
	width, height := 0, 0
	if err == nil && img != nil {
		bounds := img.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	// Create image info
	imageInfo := &ImageInfo{
		ID:           imageID,
		Filename:     filename,
		OriginalName: fileHeader.Filename,
		Size:         fileHeader.Size,
		MimeType:     mimeType,
		Width:        width,
		Height:       height,
		Hash:         hashStr,
		UploadedAt:   time.Now(),
	}

	h.log.Printf("Uploaded image: %s -> %s (%s, %dx%d, %d bytes)",
		fileHeader.Filename, filename, format, width, height, fileHeader.Size)

	return imageInfo, nil
}

// getDiskInfo gets disk usage information
func (h *FileHandler) getDiskInfo() (DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(h.uploadsDir, &stat); err != nil {
		return DiskInfo{}, fmt.Errorf("failed to get disk info: %v", err)
	}

	// Calculate disk usage
	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free
	percentage := float64(used) / float64(total) * 100

	return DiskInfo{
		Total:      total,
		Used:       used,
		Free:       free,
		Percentage: percentage,
	}, nil
}

// GetMobileUploadPage returns the mobile upload HTML page
func (h *FileHandler) GetMobileUploadPage(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <meta name="theme-color" content="#2563eb">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <title>PMA Screensaver Upload</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        :root {
            --pma-primary: #2563eb;
            --pma-accent: #f97316;
            --pma-success: #10b981;
            --pma-error: #ef4444;
            --pma-warning: #f59e0b;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #111827;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            padding: 20px;
            line-height: 1.5;
        }
        .container {
            max-width: 500px;
            margin: 0 auto;
            width: 100%;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            width: 80px;
            height: 80px;
            margin: 0 auto 20px;
            background: white;
            border-radius: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            color: var(--pma-primary);
            font-size: 24px;
            font-weight: 700;
        }
        h1 {
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 10px;
            color: white;
            text-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }
        .subtitle {
            font-size: 16px;
            color: rgba(255,255,255,0.9);
            font-weight: 500;
        }
        .info-box {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            padding: 20px;
            margin-bottom: 24px;
            font-size: 14px;
            text-align: center;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
        }
        .limit-info {
            font-weight: 600;
            color: var(--pma-accent);
        }
        .upload-container {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 24px;
            margin-bottom: 24px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
        }
        .upload-area {
            border: 2px dashed #d1d5db;
            border-radius: 16px;
            padding: 40px 20px;
            text-align: center;
            transition: all 0.3s ease;
            cursor: pointer;
            position: relative;
            overflow: hidden;
            min-height: 160px;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
        }
        .upload-area:hover {
            border-color: var(--pma-primary);
            background: rgba(37, 99, 235, 0.05);
            transform: translateY(-2px);
        }
        .upload-area.dragover {
            background: rgba(37, 99, 235, 0.1);
            border-color: var(--pma-primary);
            transform: scale(1.02);
        }
        .upload-icon {
            font-size: 48px;
            margin-bottom: 16px;
            opacity: 0.7;
            transition: all 0.3s ease;
        }
        .upload-area:hover .upload-icon {
            transform: scale(1.1);
            opacity: 1;
        }
        .upload-text {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: 8px;
            color: #374151;
        }
        .upload-hint {
            font-size: 14px;
            color: #6b7280;
            margin-bottom: 16px;
        }
        input[type="file"] {
            display: none;
        }
        .button-group {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
            margin-bottom: 24px;
        }
        .btn {
            border: none;
            border-radius: 16px;
            padding: 18px 24px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            min-height: 56px;
        }
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
        .btn-primary {
            background: var(--pma-primary);
            color: white;
        }
        .btn-primary:hover:not(:disabled) {
            background: #1d4ed8;
            transform: translateY(-2px);
            box-shadow: 0 10px 25px rgba(37, 99, 235, 0.3);
        }
        .btn-secondary {
            background: white;
            color: var(--pma-primary);
            border: 2px solid var(--pma-primary);
        }
        .btn-secondary:hover:not(:disabled) {
            background: var(--pma-primary);
            color: white;
            transform: translateY(-1px);
        }
        .upload-button, .camera-button {
            display: none;
        }
        .preview-container {
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            padding: 20px;
            margin-bottom: 24px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            display: none;
        }
        .preview-title {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: 16px;
            color: #374151;
        }
        .preview-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
            gap: 16px;
            margin-bottom: 20px;
        }
        .preview-item {
            position: relative;
            aspect-ratio: 1;
            border-radius: 12px;
            overflow: hidden;
            background: #f3f4f6;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .preview-image {
            width: 100%;
            height: 100%;
            object-fit: cover;
        }
        .preview-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.7);
            display: flex;
            align-items: center;
            justify-content: center;
            opacity: 0;
            transition: opacity 0.3s ease;
        }
        .preview-item:hover .preview-overlay {
            opacity: 1;
        }
        .remove-btn {
            background: var(--pma-error);
            color: white;
            border: none;
            border-radius: 50%;
            width: 32px;
            height: 32px;
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s ease;
        }
        .remove-btn:hover {
            transform: scale(1.1);
        }
        .preview-info {
            position: absolute;
            bottom: 0;
            left: 0;
            right: 0;
            background: rgba(0, 0, 0, 0.8);
            color: white;
            padding: 4px 8px;
            font-size: 10px;
            text-align: center;
        }
        .status-message {
            border-radius: 16px;
            padding: 20px;
            margin-bottom: 20px;
            font-size: 14px;
            display: none;
            border: 1px solid;
            background: rgba(255,255,255,0.95);
            backdrop-filter: blur(10px);
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
        }
        .status-message.success {
            border-color: var(--pma-success);
            color: var(--pma-success);
        }
        .status-message.error {
            border-color: var(--pma-error);
            color: var(--pma-error);
        }
        .status-message.warning {
            border-color: var(--pma-warning);
            color: var(--pma-warning);
        }
        .spinner {
            width: 20px;
            height: 20px;
            border: 2px solid transparent;
            border-top: 2px solid currentColor;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .quality-selector {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #e5e7eb;
        }
        .quality-label {
            font-size: 14px;
            font-weight: 500;
            color: #374151;
            margin-bottom: 12px;
        }
        .quality-options {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 8px;
        }
        .quality-option {
            padding: 12px;
            border: 2px solid #e5e7eb;
            border-radius: 12px;
            text-align: center;
            cursor: pointer;
            transition: all 0.3s ease;
            background: #f9fafb;
        }
        .quality-option:hover {
            border-color: var(--pma-primary);
        }
        .quality-option.selected {
            border-color: var(--pma-primary);
            background: rgba(37, 99, 235, 0.1);
        }
        .quality-title {
            font-size: 12px;
            font-weight: 600;
            color: #374151;
        }
        .quality-desc {
            font-size: 10px;
            color: #6b7280;
            margin-top: 2px;
        }
        @media (max-width: 480px) {
            body {
                padding: 16px;
            }
            h1 {
                font-size: 24px;
            }
            .upload-area {
                padding: 30px 16px;
                min-height: 140px;
            }
            .upload-icon {
                font-size: 36px;
            }
            .upload-text {
                font-size: 16px;
            }
            .button-group {
                grid-template-columns: 1fr;
            }
            .preview-grid {
                grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">PMA</div>
            <h1>Screensaver Upload</h1>
            <p class="subtitle">Add images to your PMA screensaver collection</p>
        </div>
        
        <div class="info-box">
            <div><strong>Max size:</strong> <span class="limit-info">50MB per image</span></div>
            <div><strong>Max files:</strong> <span class="limit-info">10 images at once</span></div>
            <div style="color: #6b7280; margin-top: 8px;">
                📱 Supports: JPG, PNG, GIF, WebP • Camera capture • Auto-compression
            </div>
        </div>
        
        <div class="upload-container">
            <div class="upload-area" id="uploadArea" tabindex="0" role="button" 
                 aria-label="Select images to upload or drag and drop files here">
                <div class="upload-icon">📸</div>
                <p class="upload-text">Tap to select images</p>
                <p class="upload-hint">or drag and drop files here</p>
                <input type="file" id="fileInput" multiple accept="image/*" aria-label="Select image files">
            </div>
            
            <div class="quality-selector">
                <div class="quality-label">Compression Quality:</div>
                <div class="quality-options">
                    <div class="quality-option" data-quality="0.6">
                        <div class="quality-title">Small</div>
                        <div class="quality-desc">Faster upload</div>
                    </div>
                    <div class="quality-option selected" data-quality="0.8">
                        <div class="quality-title">Balanced</div>
                        <div class="quality-desc">Recommended</div>
                    </div>
                    <div class="quality-option" data-quality="0.95">
                        <div class="quality-title">High</div>
                        <div class="quality-desc">Best quality</div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="button-group">
            <button class="btn btn-secondary camera-button" id="cameraButton" type="button">
                📷 Take Photo
            </button>
            <button class="btn btn-primary upload-button" id="uploadButton" type="button">
                <span class="spinner" style="display: none;"></span>
                <span id="uploadButtonText">📤 Upload Images</span>
            </button>
        </div>
        
        <div class="preview-container" id="previewContainer">
            <div class="preview-title">Selected Images (<span id="imageCount">0</span>)</div>
            <div class="preview-grid" id="previewGrid"></div>
        </div>
        
        <div class="status-message" id="statusMessage" role="alert" aria-live="polite"></div>
    </div>
    
    <script>
        // Global variables
        const uploadArea = document.getElementById('uploadArea');
        const fileInput = document.getElementById('fileInput');
        const uploadButton = document.getElementById('uploadButton');
        const uploadButtonText = document.getElementById('uploadButtonText');
        const cameraButton = document.getElementById('cameraButton');
        const statusMessage = document.getElementById('statusMessage');
        const previewContainer = document.getElementById('previewContainer');
        const previewGrid = document.getElementById('previewGrid');
        const imageCount = document.getElementById('imageCount');
        const qualityOptions = document.querySelectorAll('.quality-option');
        
        let filesToUpload = [];
        let compressionQuality = 0.8;
        let isUploading = false;
        
        // Check for camera support
        if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
            cameraButton.style.display = 'flex';
        }
        
        // Quality selector handlers
        qualityOptions.forEach(option => {
            option.addEventListener('click', () => {
                qualityOptions.forEach(opt => opt.classList.remove('selected'));
                option.classList.add('selected');
                compressionQuality = parseFloat(option.dataset.quality);
            });
        });
        
        // Upload area handlers
        uploadArea.addEventListener('click', () => fileInput.click());
        uploadArea.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                fileInput.click();
            }
        });
        
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });
        
        uploadArea.addEventListener('dragleave', (e) => {
            if (!uploadArea.contains(e.relatedTarget)) {
                uploadArea.classList.remove('dragover');
            }
        });
        
        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            handleFiles(e.dataTransfer.files);
        });
        
        fileInput.addEventListener('change', (e) => {
            handleFiles(e.target.files);
        });
        
        // Camera button handler
        cameraButton.addEventListener('click', async () => {
            try {
                const stream = await navigator.mediaDevices.getUserMedia({ 
                    video: { 
                        facingMode: 'environment',
                        width: { ideal: 1920 },
                        height: { ideal: 1080 }
                    } 
                });
                
                createCameraInterface(stream);
            } catch (error) {
                console.error('Camera access error:', error);
                showStatus('❌ Camera access denied or not available', 'error');
            }
        });
        
        // Upload button handler
        uploadButton.addEventListener('click', async () => {
            if (filesToUpload.length === 0 || isUploading) return;
            await uploadFiles();
        });
        
        // File handling functions
        async function handleFiles(files) {
            const imageFiles = Array.from(files).filter(file => file.type.startsWith('image/'));
            
            if (imageFiles.length === 0) {
                showStatus('⚠️ Please select only image files', 'warning');
                return;
            }
            
            if (imageFiles.length > 10) {
                showStatus('⚠️ Maximum 10 files allowed. First 10 files selected.', 'warning');
                imageFiles.splice(10);
            }
            
            // Process and compress images
            const processedFiles = [];
            for (const file of imageFiles) {
                try {
                    const compressedFile = await compressImage(file, compressionQuality);
                    processedFiles.push(compressedFile);
                } catch (error) {
                    console.error('Compression error for', file.name, error);
                    processedFiles.push(file); // Use original if compression fails
                }
            }
            
            filesToUpload = processedFiles;
            updatePreview();
            uploadButton.style.display = 'flex';
            updateUploadButtonText();
        }
        
        async function compressImage(file, quality) {
            return new Promise((resolve) => {
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                const img = new Image();
                
                img.onload = () => {
                    // Calculate new dimensions (max 1920x1080 while maintaining aspect ratio)
                    const maxWidth = 1920;
                    const maxHeight = 1080;
                    let { width, height } = img;
                    
                    if (width > maxWidth || height > maxHeight) {
                        const ratio = Math.min(maxWidth / width, maxHeight / height);
                        width *= ratio;
                        height *= ratio;
                    }
                    
                    canvas.width = width;
                    canvas.height = height;
                    
                    // Draw and compress
                    ctx.drawImage(img, 0, 0, width, height);
                    
                    canvas.toBlob((blob) => {
                        const compressedFile = new File([blob], file.name, {
                            type: file.type,
                            lastModified: file.lastModified
                        });
                        resolve(compressedFile);
                    }, file.type, quality);
                };
                
                img.src = URL.createObjectURL(file);
            });
        }
        
        function updatePreview() {
            if (filesToUpload.length === 0) {
                previewContainer.style.display = 'none';
                return;
            }
            
            previewContainer.style.display = 'block';
            imageCount.textContent = filesToUpload.length;
            
            previewGrid.innerHTML = '';
            filesToUpload.forEach((file, index) => {
                const previewItem = createPreviewItem(file, index);
                previewGrid.appendChild(previewItem);
            });
        }
        
        function createPreviewItem(file, index) {
            const item = document.createElement('div');
            item.className = 'preview-item';
            
            const img = document.createElement('img');
            img.className = 'preview-image';
            img.src = URL.createObjectURL(file);
            img.alt = file.name;
            
            const overlay = document.createElement('div');
            overlay.className = 'preview-overlay';
            
            const removeBtn = document.createElement('button');
            removeBtn.className = 'remove-btn';
            removeBtn.innerHTML = '✕';
            removeBtn.title = 'Remove image';
            removeBtn.addEventListener('click', () => removeFile(index));
            
            const info = document.createElement('div');
            info.className = 'preview-info';
            info.textContent = formatFileSize(file.size);
            
            overlay.appendChild(removeBtn);
            item.appendChild(img);
            item.appendChild(overlay);
            item.appendChild(info);
            
            return item;
        }
        
        function removeFile(index) {
            filesToUpload.splice(index, 1);
            updatePreview();
            
            if (filesToUpload.length === 0) {
                uploadButton.style.display = 'none';
                fileInput.value = '';
            } else {
                updateUploadButtonText();
            }
        }
        
        function updateUploadButtonText() {
            const count = filesToUpload.length;
            uploadButtonText.textContent = '📤 Upload ' + count + ' Image' + (count > 1 ? 's' : '');
        }
        
        async function uploadFiles() {
            if (isUploading) return;
            
            isUploading = true;
            uploadButton.disabled = true;
            uploadButton.querySelector('.spinner').style.display = 'block';
            uploadButtonText.textContent = 'Uploading...';
            
            try {
                const formData = new FormData();
                filesToUpload.forEach(file => {
                    formData.append('images', file);
                });
                
                const response = await fetch('/api/screensaver/images/upload', {
                    method: 'POST',
                    body: formData
                });
                
                const result = await response.json();
                
                if (result.success) {
                    showStatus('✅ ' + (result.data.message || 'Images uploaded successfully!'), 'success');
                    filesToUpload = [];
                    fileInput.value = '';
                    previewContainer.style.display = 'none';
                    uploadButton.style.display = 'none';
                } else {
                    showStatus('❌ ' + (result.error || 'Upload failed'), 'error');
                }
            } catch (error) {
                console.error('Upload error:', error);
                showStatus('❌ Network error: ' + error.message, 'error');
            }
            
            isUploading = false;
            uploadButton.disabled = false;
            uploadButton.querySelector('.spinner').style.display = 'none';
            updateUploadButtonText();
        }
        
        function createCameraInterface(stream) {
            // Create modal overlay
            const modal = document.createElement('div');
            modal.style.position = 'fixed';
            modal.style.top = '0';
            modal.style.left = '0';
            modal.style.right = '0';
            modal.style.bottom = '0';
            modal.style.background = 'rgba(0, 0, 0, 0.9)';
            modal.style.display = 'flex';
            modal.style.flexDirection = 'column';
            modal.style.alignItems = 'center';
            modal.style.justifyContent = 'center';
            modal.style.zIndex = '1000';
            modal.style.padding = '20px';
            
            // Create video element
            const video = document.createElement('video');
            video.style.maxWidth = '100%';
            video.style.maxHeight = '70vh';
            video.style.borderRadius = '12px';
            video.srcObject = stream;
            video.autoplay = true;
            video.playsInline = true;
            
            // Create button container
            const buttonContainer = document.createElement('div');
            buttonContainer.style.display = 'flex';
            buttonContainer.style.gap = '20px';
            buttonContainer.style.marginTop = '20px';
            buttonContainer.style.flexWrap = 'wrap';
            buttonContainer.style.justifyContent = 'center';
            
            // Create capture button
            const captureBtn = document.createElement('button');
            captureBtn.className = 'btn btn-primary';
            captureBtn.innerHTML = '📸 Capture';
            captureBtn.onclick = () => capturePhoto(video, modal, stream);
            
            // Create close button
            const closeBtn = document.createElement('button');
            closeBtn.className = 'btn btn-secondary';
            closeBtn.innerHTML = '✕ Close';
            closeBtn.onclick = () => {
                stream.getTracks().forEach(track => track.stop());
                document.body.removeChild(modal);
            };
            
            buttonContainer.appendChild(captureBtn);
            buttonContainer.appendChild(closeBtn);
            modal.appendChild(video);
            modal.appendChild(buttonContainer);
            document.body.appendChild(modal);
        }
        
        function capturePhoto(video, modal, stream) {
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            ctx.drawImage(video, 0, 0);
            
            canvas.toBlob(async (blob) => {
                const timestamp = Date.now();
                const file = new File([blob], 'camera-' + timestamp + '.jpg', { type: 'image/jpeg' });
                
                stream.getTracks().forEach(track => track.stop());
                document.body.removeChild(modal);
                
                const compressedFile = await compressImage(file, compressionQuality);
                filesToUpload.push(compressedFile);
                updatePreview();
                uploadButton.style.display = 'flex';
                updateUploadButtonText();
                
                showStatus('📸 Photo captured! Ready to upload.', 'success');
            }, 'image/jpeg', 0.9);
        }
        
        function showStatus(message, type) {
            statusMessage.textContent = message;
            statusMessage.className = 'status-message ' + type;
            statusMessage.style.display = 'block';
            
            if (type === 'success') {
                setTimeout(() => {
                    statusMessage.style.display = 'none';
                }, 5000);
            }
        }
        
        function formatFileSize(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
        }
        
        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
                    case 'o':
                        e.preventDefault();
                        fileInput.click();
                        break;
                    case 'u':
                        if (filesToUpload.length > 0 && !isUploading) {
                            e.preventDefault();
                            uploadFiles();
                        }
                        break;
                }
            }
        });
        
        // Initialize
        console.log('🚀 PMA Mobile Upload initialized');
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.String(http.StatusOK, html)
}
