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
}

// NewFileHandler creates a new file handler
func NewFileHandler(logger *log.Logger, uploadsDir string) *FileHandler {
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
		images: make([]ImageInfo, 0),
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
    <title>PMA Screensaver Upload</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        :root {
            --pma-primary: #2563eb;
            --pma-accent: #f97316;
            --pma-success: #10b981;
            --pma-error: #ef4444;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
            background-color: #f9fafb;
            color: #111827;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            padding: 20px;
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
            box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
            border: 1px solid #e5e7eb;
            color: var(--pma-primary);
            font-size: 24px;
            font-weight: 700;
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 10px;
            color: #111827;
        }
        .subtitle {
            font-size: 14px;
            color: #6b7280;
        }
        .info-box {
            background: #ffffff;
            border: 1px solid #e5e7eb;
            border-radius: 16px;
            padding: 20px;
            margin-bottom: 24px;
            font-size: 14px;
            text-align: center;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }
        .limit-info {
            font-weight: 600;
            color: var(--pma-accent);
        }
        .upload-area {
            background: #ffffff;
            border: 2px dashed #d1d5db;
            border-radius: 20px;
            padding: 40px 20px;
            text-align: center;
            margin-bottom: 24px;
            transition: all 0.3s ease;
            cursor: pointer;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }
        .upload-area:hover {
            border-color: var(--pma-primary);
            background-color: #f8fafc;
        }
        .upload-area.dragover {
            background: #eff6ff;
            border-color: var(--pma-primary);
            transform: scale(1.02);
        }
        .upload-text {
            font-size: 16px;
            font-weight: 500;
            margin-bottom: 10px;
            color: #374151;
        }
        .upload-hint {
            font-size: 12px;
            color: #6b7280;
        }
        input[type="file"] {
            display: none;
        }
        .upload-button {
            background: var(--pma-primary);
            color: white;
            border: none;
            border-radius: 16px;
            padding: 18px 40px;
            font-size: 16px;
            font-weight: 600;
            width: 100%;
            cursor: pointer;
            transition: all 0.3s ease;
            display: none;
        }
        .upload-button:hover {
            background: #1d4ed8;
            transform: translateY(-1px);
        }
        .status-message {
            border-radius: 12px;
            padding: 16px;
            margin-top: 20px;
            font-size: 14px;
            text-align: center;
            display: none;
            border: 1px solid;
        }
        .status-message.success {
            background: #ecfdf5;
            border-color: #10b981;
            color: #065f46;
        }
        .status-message.error {
            background: #fef2f2;
            border-color: #ef4444;
            color: #991b1b;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo">PMA</div>
        <h1>Screensaver Upload</h1>
        <p class="subtitle">Add images to your PMA screensaver</p>
    </div>
    
    <div class="info-box">
        <div>Maximum file size: <span class="limit-info">50MB per image</span></div>
        <div>Maximum files: <span class="limit-info">10 images at once</span></div>
        <div style="color: #6b7280;">Supported formats: JPG, PNG, GIF, WebP</div>
    </div>
    
    <div class="upload-area" id="uploadArea">
        <p class="upload-text">📸 Tap to select images</p>
        <p class="upload-hint">or drag and drop files here</p>
        <input type="file" id="fileInput" multiple accept="image/*">
    </div>
    
    <button class="upload-button" id="uploadButton">Upload Images</button>
    
    <div class="status-message" id="statusMessage"></div>
    
    <script>
        const uploadArea = document.getElementById('uploadArea');
        const fileInput = document.getElementById('fileInput');
        const uploadButton = document.getElementById('uploadButton');
        const statusMessage = document.getElementById('statusMessage');
        
        let filesToUpload = [];
        
        uploadArea.addEventListener('click', () => fileInput.click());
        
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });
        
        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('dragover');
        });
        
        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            handleFiles(e.dataTransfer.files);
        });
        
        fileInput.addEventListener('change', (e) => {
            handleFiles(e.target.files);
        });
        
        function handleFiles(files) {
            const imageFiles = Array.from(files).filter(file => file.type.startsWith('image/'));
            
            if (imageFiles.length === 0) {
                showStatus('Please select only image files', 'error');
                return;
            }
            
                         if (imageFiles.length > 10) {
                 imageFiles = imageFiles.slice(0, 10);
                 showStatus('Only first 10 files selected (maximum allowed)', 'error');
             }
             
             filesToUpload = imageFiles;
             uploadButton.style.display = 'block';
             uploadButton.textContent = '📸 Upload ' + imageFiles.length + ' Image' + (imageFiles.length > 1 ? 's' : '');
         }
        
        uploadButton.addEventListener('click', async () => {
            if (filesToUpload.length === 0) return;
            
            uploadButton.disabled = true;
            uploadButton.textContent = 'Uploading...';
            statusMessage.style.display = 'none';
            
            const formData = new FormData();
            filesToUpload.forEach(file => {
                formData.append('images', file);
            });
            
            try {
                const response = await fetch('/api/screensaver/images/upload', {
                    method: 'POST',
                    body: formData
                });
                
                const result = await response.json();
                
                if (result.success) {
                    showStatus(result.data.message || 'Images uploaded successfully!', 'success');
                    filesToUpload = [];
                    fileInput.value = '';
                    uploadButton.style.display = 'none';
                } else {
                    showStatus(result.error || 'Upload failed', 'error');
                }
            } catch (error) {
                showStatus('Network error: ' + error.message, 'error');
            }
            
            uploadButton.disabled = false;
            uploadButton.textContent = 'Upload Images';
        });
        
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
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
