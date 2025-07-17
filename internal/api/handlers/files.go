package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/backup"
	"github.com/frostdev-ops/pma-backend-go/internal/core/filemanager"
	"github.com/frostdev-ops/pma-backend-go/internal/core/logs"
	"github.com/frostdev-ops/pma-backend-go/internal/core/media"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FileHandler handles file management API endpoints
type FileHandler struct {
	fileManager   filemanager.FileManager
	mediaStreamer media.MediaStreamer
	backupManager backup.BackupManager
	logManager    logs.LogManager
	logger        *logrus.Logger
}

// NewFileHandler creates a new file handler
func NewFileHandler(
	fm filemanager.FileManager,
	ms media.MediaStreamer,
	bm backup.BackupManager,
	lm logs.LogManager,
	logger *logrus.Logger,
) *FileHandler {
	return &FileHandler{
		fileManager:   fm,
		mediaStreamer: ms,
		backupManager: bm,
		logManager:    lm,
		logger:        logger,
	}
}

// File Management Endpoints

// UploadFile handles file uploads
func (fh *FileHandler) UploadFile(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	// Parse metadata
	var metadata filemanager.FileMetadata
	if metadataStr := c.PostForm("metadata"); metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
			return
		}
	}

	// Set category from form or default
	if category := c.PostForm("category"); category != "" {
		metadata.Category = category
	}
	if metadata.Category == "" {
		metadata.Category = filemanager.CategoryUpload
	}

	// Set description
	if description := c.PostForm("description"); description != "" {
		metadata.Description = description
	}

	// Set tags
	if tagsStr := c.PostForm("tags"); tagsStr != "" {
		metadata.Tags = strings.Split(tagsStr, ",")
	}

	// TODO: Get user ID from authentication context
	// metadata.UploadedBy = getUserID(c)

	// Upload file
	uploadedFile, err := fh.fileManager.Upload(header.Filename, file, metadata)
	if err != nil {
		fh.logger.Errorf("File upload failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"file":    uploadedFile,
	})
}

// DownloadFile handles file downloads
func (fh *FileHandler) DownloadFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	// Get file info
	fileInfo, err := fh.fileManager.GetFileInfo(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// TODO: Check permissions
	// if !hasPermission(c, fileID, "read") {
	//     c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
	//     return
	// }

	// Open file for reading
	reader, err := fh.fileManager.Download(fileID)
	if err != nil {
		fh.logger.Errorf("Failed to open file %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer reader.Close()

	// Set headers
	c.Header("Content-Type", fileInfo.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileInfo.Name))
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size, 10))

	// Stream file content
	io.Copy(c.Writer, reader)
}

// DeleteFile handles file deletion
func (fh *FileHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	// TODO: Check permissions
	// if !hasPermission(c, fileID, "delete") {
	//     c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
	//     return
	// }

	if err := fh.fileManager.Delete(fileID); err != nil {
		fh.logger.Errorf("Failed to delete file %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListFiles handles file listing with pagination and filtering
func (fh *FileHandler) ListFiles(c *gin.Context) {
	var filter filemanager.FileFilter

	// Parse query parameters
	if category := c.Query("category"); category != "" {
		filter.Category = category
	}

	if tags := c.Query("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
	}

	if mimeTypes := c.Query("mime_types"); mimeTypes != "" {
		filter.MimeTypes = strings.Split(mimeTypes, ",")
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if parsed, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = parsed
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if parsed, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = parsed
		}
	}

	if minSize := c.Query("min_size"); minSize != "" {
		if parsed, err := strconv.ParseInt(minSize, 10, 64); err == nil {
			filter.MinSize = parsed
		}
	}

	if maxSize := c.Query("max_size"); maxSize != "" {
		if parsed, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			filter.MaxSize = parsed
		}
	}

	if search := c.Query("search"); search != "" {
		filter.NameSearch = search
	}

	if limit := c.Query("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			filter.Limit = parsed
		}
	} else {
		filter.Limit = 50 // Default limit
	}

	if offset := c.Query("offset"); offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil {
			filter.Offset = parsed
		}
	}

	// Get files
	files, err := fh.fileManager.List(filter)
	if err != nil {
		fh.logger.Errorf("Failed to list files: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  files,
		"filter": filter,
		"count":  len(files),
	})
}

// GetFileMetadata handles metadata retrieval
func (fh *FileHandler) GetFileMetadata(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	metadata, err := fh.fileManager.GetMetadata(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metadata": metadata})
}

// UpdateFileMetadata handles metadata updates
func (fh *FileHandler) UpdateFileMetadata(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	var metadata filemanager.FileMetadata
	if err := c.ShouldBindJSON(&metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	if err := fh.fileManager.UpdateMetadata(fileID, metadata); err != nil {
		fh.logger.Errorf("Failed to update metadata for file %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStorageStats returns storage usage statistics
func (fh *FileHandler) GetStorageStats(c *gin.Context) {
	stats, err := fh.fileManager.GetStorageStats()
	if err != nil {
		fh.logger.Errorf("Failed to get storage stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get storage stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Media Streaming Endpoints

// StreamMedia handles media streaming with range support
func (fh *FileHandler) StreamMedia(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	// Parse streaming options
	var options media.StreamOptions
	if quality := c.Query("quality"); quality != "" {
		options.Quality = quality
	}
	if format := c.Query("format"); format != "" {
		options.Format = format
	}
	if start := c.Query("start"); start != "" {
		if parsed, err := strconv.ParseInt(start, 10, 64); err == nil {
			options.Start = parsed
		}
	}
	if end := c.Query("end"); end != "" {
		if parsed, err := strconv.ParseInt(end, 10, 64); err == nil {
			options.End = parsed
		}
	}

	// Get file info first
	fileInfo, err := fh.fileManager.GetFileInfo(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Determine if it's video or audio
	var reader io.ReadSeeker
	if strings.HasPrefix(fileInfo.MimeType, "video/") {
		reader, err = fh.mediaStreamer.StreamVideo(fileID, options)
	} else if strings.HasPrefix(fileInfo.MimeType, "audio/") {
		reader, err = fh.mediaStreamer.StreamAudio(fileID, options)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is not a media file"})
		return
	}

	if err != nil {
		fh.logger.Errorf("Failed to stream media %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream media"})
		return
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	// Handle range requests
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		fh.handleRangeRequest(c, reader, fileInfo, rangeHeader)
		return
	}

	// Set headers for streaming
	c.Header("Content-Type", fileInfo.MimeType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size, 10))

	// Stream content
	io.Copy(c.Writer, reader)
}

// handleRangeRequest handles HTTP range requests for media streaming
func (fh *FileHandler) handleRangeRequest(c *gin.Context, reader io.ReadSeeker, fileInfo *filemanager.File, rangeHeader string) {
	// Parse range header (e.g., "bytes=0-1023")
	ranges := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(ranges, "-")

	if len(parts) != 2 {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	var start, end int64
	var err error

	if parts[0] != "" {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	} else {
		end = fileInfo.Size - 1
	}

	// Validate range
	if start > end || start >= fileInfo.Size {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1

	// Seek to start position
	if _, err := reader.Seek(start, 0); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Set partial content headers
	c.Header("Content-Type", fileInfo.MimeType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size))
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Status(http.StatusPartialContent)

	// Stream the requested range
	io.CopyN(c.Writer, reader, contentLength)
}

// GetThumbnail handles thumbnail generation and retrieval
func (fh *FileHandler) GetThumbnail(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	thumbnail, err := fh.mediaStreamer.GenerateThumbnail(fileID)
	if err != nil {
		fh.logger.Errorf("Failed to generate thumbnail for %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate thumbnail"})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Data(http.StatusOK, "image/jpeg", thumbnail)
}

// GetMediaInfo handles media information retrieval
func (fh *FileHandler) GetMediaInfo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	mediaInfo, err := fh.mediaStreamer.GetMediaInfo(fileID)
	if err != nil {
		fh.logger.Errorf("Failed to get media info for %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get media info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"media_info": mediaInfo})
}

// TranscodeVideo handles video transcoding requests
func (fh *FileHandler) TranscodeVideo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID required"})
		return
	}

	var profile media.TranscodeProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transcode profile"})
		return
	}

	// Start transcoding (this would typically be async)
	err := fh.mediaStreamer.TranscodeVideo(fileID, profile)
	if err != nil {
		fh.logger.Errorf("Failed to start transcoding for %s: %v", fileID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transcoding"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Transcoding started",
		"profile": profile,
	})
}

// Backup Management Endpoints

// CreateBackup handles backup creation
func (fh *FileHandler) CreateBackup(c *gin.Context) {
	var options backup.BackupOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup options"})
		return
	}

	backupRecord, err := fh.backupManager.CreateBackup(options)
	if err != nil {
		fh.logger.Errorf("Failed to create backup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"backup":  backupRecord,
	})
}

// ListBackups handles backup listing
func (fh *FileHandler) ListBackups(c *gin.Context) {
	backups, err := fh.backupManager.ListBackups()
	if err != nil {
		fh.logger.Errorf("Failed to list backups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

// RestoreBackup handles backup restoration
func (fh *FileHandler) RestoreBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Backup ID required"})
		return
	}

	var options backup.RestoreOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid restore options"})
		return
	}

	if err := fh.backupManager.RestoreBackup(backupID, options); err != nil {
		fh.logger.Errorf("Failed to restore backup %s: %v", backupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteBackup handles backup deletion
func (fh *FileHandler) DeleteBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Backup ID required"})
		return
	}

	if err := fh.backupManager.DeleteBackup(backupID); err != nil {
		fh.logger.Errorf("Failed to delete backup %s: %v", backupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DownloadBackup handles backup download
func (fh *FileHandler) DownloadBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Backup ID required"})
		return
	}

	// Get backup info
	_, err := fh.backupManager.GetBackupInfo(backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
		return
	}

	// Set headers
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="backup_%s.tar"`, backupID))

	// Stream backup
	if err := fh.backupManager.ExportBackup(backupID, c.Writer); err != nil {
		fh.logger.Errorf("Failed to export backup %s: %v", backupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export backup"})
		return
	}
}

// ScheduleBackup handles backup scheduling
func (fh *FileHandler) ScheduleBackup(c *gin.Context) {
	var schedule backup.BackupSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule format"})
		return
	}

	if err := fh.backupManager.ScheduleBackup(schedule); err != nil {
		fh.logger.Errorf("Failed to schedule backup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Log Management Endpoints

// GetLogs handles log retrieval with filtering
func (fh *FileHandler) GetLogs(c *gin.Context) {
	var filter logs.LogFilter

	// Parse query parameters
	if service := c.Query("service"); service != "" {
		filter.Service = service
	}
	if level := c.Query("level"); level != "" {
		filter.Level = level
	}
	if startTime := c.Query("start_time"); startTime != "" {
		if parsed, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = parsed
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if parsed, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = parsed
		}
	}
	if pattern := c.Query("pattern"); pattern != "" {
		filter.Pattern = pattern
	}
	if limit := c.Query("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			filter.Limit = parsed
		}
	} else {
		filter.Limit = 100 // Default limit
	}

	logs, err := fh.logManager.GetLogs(filter)
	if err != nil {
		fh.logger.Errorf("Failed to get logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"filter": filter,
		"count":  len(logs),
	})
}

// StreamLogs handles real-time log streaming via WebSocket
func (fh *FileHandler) StreamLogs(c *gin.Context) {
	// This would typically upgrade to WebSocket
	// For now, return an error indicating WebSocket is required
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "WebSocket connection required",
		"upgrade": "Use WebSocket endpoint for log streaming",
	})
}

// ExportLogs handles log export
func (fh *FileHandler) ExportLogs(c *gin.Context) {
	var filter logs.LogFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filter format"})
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	reader, err := fh.logManager.ExportLogs(filter, format)
	if err != nil {
		fh.logger.Errorf("Failed to export logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export logs"})
		return
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	// Set headers
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="logs_%s.%s"`, time.Now().Format("20060102_150405"), format))

	// Stream logs
	io.Copy(c.Writer, reader)
}

// PurgeLogs handles old log deletion
func (fh *FileHandler) PurgeLogs(c *gin.Context) {
	beforeStr := c.Query("before")
	if beforeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Before timestamp required"})
		return
	}

	before, err := time.Parse(time.RFC3339, beforeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format"})
		return
	}

	if err := fh.logManager.PurgeLogs(before); err != nil {
		fh.logger.Errorf("Failed to purge logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetLogStats handles log statistics retrieval
func (fh *FileHandler) GetLogStats(c *gin.Context) {
	stats, err := fh.logManager.GetLogStats()
	if err != nil {
		fh.logger.Errorf("Failed to get log stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get log stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}
