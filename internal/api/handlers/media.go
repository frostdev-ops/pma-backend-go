package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/media"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MediaHandler handles media processing API requests
type MediaHandler struct {
	mediaProcessor     *media.MediaProcessor
	mediaStreamer      media.MediaStreamer
	thumbnailGenerator *media.ThumbnailGenerator
	logger             *logrus.Logger
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(
	processor *media.MediaProcessor,
	streamer media.MediaStreamer,
	thumbnailGen *media.ThumbnailGenerator,
	logger *logrus.Logger,
) *MediaHandler {
	return &MediaHandler{
		mediaProcessor:     processor,
		mediaStreamer:      streamer,
		thumbnailGenerator: thumbnailGen,
		logger:             logger,
	}
}

// ProcessMedia processes an uploaded media file and extracts metadata
func (mh *MediaHandler) ProcessMedia(c *gin.Context) {
	file, header, err := c.Request.FormFile("media")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	mh.logger.WithFields(logrus.Fields{
		"filename":  header.Filename,
		"mime_type": mimeType,
		"size":      header.Size,
	}).Info("Processing media file")

	// Extract media information
	mediaInfo, err := mh.mediaProcessor.ExtractMediaInfo(file, mimeType)
	if err != nil {
		mh.logger.WithError(err).Error("Failed to extract media info")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to process media: %v", err))
		return
	}

	// Create response with file information and media info
	response := gin.H{
		"message": "Media processed successfully",
		"file_info": gin.H{
			"filename":  header.Filename,
			"size":      header.Size,
			"mime_type": mimeType,
		},
		"media_info": mediaInfo,
	}

	utils.SendSuccess(c, response)
}

// GetMediaInfo retrieves media information for a file
func (mh *MediaHandler) GetMediaInfo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.SendError(c, http.StatusBadRequest, "File ID is required")
		return
	}

	mediaInfo, err := mh.mediaStreamer.GetMediaInfo(fileID)
	if err != nil {
		mh.logger.WithError(err).WithField("file_id", fileID).Error("Failed to get media info")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Media not found: %v", err))
		return
	}

	utils.SendSuccess(c, mediaInfo)
}

// GenerateThumbnail generates a thumbnail for uploaded media
func (mh *MediaHandler) GenerateThumbnail(c *gin.Context) {
	file, header, err := c.Request.FormFile("media")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Parse thumbnail options from query parameters
	var options media.ThumbnailOptions = mh.thumbnailGenerator.GetDefaultThumbnailOptions()

	if widthStr := c.Query("width"); widthStr != "" {
		if width, err := strconv.Atoi(widthStr); err == nil {
			options.Width = width
		}
	}

	if heightStr := c.Query("height"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			options.Height = height
		}
	}

	if quality := c.Query("quality"); quality != "" {
		if q, err := strconv.Atoi(quality); err == nil {
			options.Quality = q
		}
	}

	mh.logger.WithFields(logrus.Fields{
		"filename":  header.Filename,
		"mime_type": mimeType,
		"width":     options.Width,
		"height":    options.Height,
	}).Info("Generating thumbnail")

	thumbnail, err := mh.thumbnailGenerator.GenerateWithOptions(file, mimeType, options)
	if err != nil {
		mh.logger.WithError(err).Error("Failed to generate thumbnail")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate thumbnail: %v", err))
		return
	}

	// Return thumbnail as image
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", len(thumbnail)))
	c.Data(http.StatusOK, "image/jpeg", thumbnail)
}

// GetThumbnail retrieves an existing thumbnail for a file
func (mh *MediaHandler) GetThumbnail(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.SendError(c, http.StatusBadRequest, "File ID is required")
		return
	}

	thumbnail, err := mh.mediaStreamer.GenerateThumbnail(fileID)
	if err != nil {
		mh.logger.WithError(err).WithField("file_id", fileID).Error("Failed to get thumbnail")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Thumbnail not found: %v", err))
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", len(thumbnail)))
	c.Data(http.StatusOK, "image/jpeg", thumbnail)
}

// StreamVideo streams a video file
func (mh *MediaHandler) StreamVideo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.SendError(c, http.StatusBadRequest, "File ID is required")
		return
	}

	// Parse streaming options
	var options media.StreamOptions

	if qualityStr := c.Query("quality"); qualityStr != "" {
		options.Quality = qualityStr
	}

	if formatStr := c.Query("format"); formatStr != "" {
		options.Format = formatStr
	}

	stream, err := mh.mediaStreamer.StreamVideo(fileID, options)
	if err != nil {
		mh.logger.WithError(err).WithField("file_id", fileID).Error("Failed to stream video")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Video not found: %v", err))
		return
	}

	// Set appropriate headers for video streaming
	c.Header("Content-Type", "video/mp4")
	c.Header("Accept-Ranges", "bytes")

	// Stream the video content using ReadSeeker
	http.ServeContent(c.Writer, c.Request, fileID, time.Now(), stream)
}

// StreamAudio streams an audio file
func (mh *MediaHandler) StreamAudio(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.SendError(c, http.StatusBadRequest, "File ID is required")
		return
	}

	// Parse streaming options
	var options media.StreamOptions

	if qualityStr := c.Query("quality"); qualityStr != "" {
		options.Quality = qualityStr
	}

	if formatStr := c.Query("format"); formatStr != "" {
		options.Format = formatStr
	}

	stream, err := mh.mediaStreamer.StreamAudio(fileID, options)
	if err != nil {
		mh.logger.WithError(err).WithField("file_id", fileID).Error("Failed to stream audio")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Audio not found: %v", err))
		return
	}

	// Set appropriate headers for audio streaming
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Accept-Ranges", "bytes")

	// Stream the audio content using ReadSeeker
	http.ServeContent(c.Writer, c.Request, fileID, time.Now(), stream)
}

// TranscodeVideo transcodes a video to different quality/format
func (mh *MediaHandler) TranscodeVideo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.SendError(c, http.StatusBadRequest, "File ID is required")
		return
	}

	var profile media.TranscodeProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid transcode profile")
		return
	}

	mh.logger.WithFields(logrus.Fields{
		"file_id": fileID,
		"profile": profile.Name,
	}).Info("Starting video transcoding")

	err := mh.mediaStreamer.TranscodeVideo(fileID, profile)
	if err != nil {
		mh.logger.WithError(err).WithField("file_id", fileID).Error("Failed to transcode video")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to transcode video: %v", err))
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Video transcoding started",
		"file_id": fileID,
		"profile": profile.Name,
	})
}

// GetStreamingURL returns a streaming URL for a media file
func (mh *MediaHandler) GetStreamingURL(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		utils.SendError(c, http.StatusBadRequest, "File ID is required")
		return
	}

	var options media.StreamOptions
	if qualityStr := c.Query("quality"); qualityStr != "" {
		options.Quality = qualityStr
	}
	if formatStr := c.Query("format"); formatStr != "" {
		options.Format = formatStr
	}

	url, err := mh.mediaStreamer.GetStreamingURL(fileID, options)
	if err != nil {
		mh.logger.WithError(err).WithField("file_id", fileID).Error("Failed to get streaming URL")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Failed to get streaming URL: %v", err))
		return
	}

	utils.SendSuccess(c, gin.H{
		"streaming_url": url,
		"file_id":       fileID,
		"options":       options,
	})
}

// ValidateMedia validates an uploaded media file
func (mh *MediaHandler) ValidateMedia(c *gin.Context) {
	file, header, err := c.Request.FormFile("media")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Validate with media processor
	if err := mh.mediaProcessor.ValidateMediaFile(mimeType); err != nil {
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Invalid media file: %v", err))
		return
	}

	// If it's an image, also validate with thumbnail generator
	if mimeType[:6] == "image/" {
		if err := mh.thumbnailGenerator.ValidateImage(file, mimeType); err != nil {
			utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Invalid image file: %v", err))
			return
		}
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Media file is valid",
		"mime_type": mimeType,
		"filename":  header.Filename,
		"size":      header.Size,
	})
}

// GetSupportedFormats returns supported media formats
func (mh *MediaHandler) GetSupportedFormats(c *gin.Context) {
	formats := mh.mediaProcessor.GetSupportedFormats()
	thumbnailFormats := mh.thumbnailGenerator.GetSupportedFormats()

	utils.SendSuccess(c, gin.H{
		"media_formats":     formats,
		"thumbnail_formats": thumbnailFormats,
	})
}

// GenerateMultipleThumbnails generates thumbnails in multiple sizes
func (mh *MediaHandler) GenerateMultipleThumbnails(c *gin.Context) {
	file, header, err := c.Request.FormFile("media")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	mh.logger.WithFields(logrus.Fields{
		"filename":  header.Filename,
		"mime_type": mimeType,
	}).Info("Generating multiple thumbnail sizes")

	thumbnails, err := mh.thumbnailGenerator.GenerateMultipleSizes(file, mimeType)
	if err != nil {
		mh.logger.WithError(err).Error("Failed to generate multiple thumbnails")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate thumbnails: %v", err))
		return
	}

	// Convert thumbnails to base64 for JSON response
	result := make(map[string]string)
	for size, data := range thumbnails {
		result[size] = fmt.Sprintf("data:image/jpeg;base64,%s", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":    "Multiple thumbnails generated successfully",
		"thumbnails": result,
		"count":      len(result),
	})
}

// GetMediaStats returns statistics about media processing
func (mh *MediaHandler) GetMediaStats(c *gin.Context) {
	// This is a placeholder for media statistics
	// In a real implementation, you'd gather stats from a database or cache
	utils.SendSuccess(c, gin.H{
		"supported_formats": mh.mediaProcessor.GetSupportedFormats(),
		"thumbnail_formats": mh.thumbnailGenerator.GetSupportedFormats(),
		"processor_status":  "active",
		"streaming_status":  "active",
	})
}
