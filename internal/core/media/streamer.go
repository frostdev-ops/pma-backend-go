package media

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/filemanager"
	"github.com/sirupsen/logrus"
)

// MediaStreamer defines the interface for media streaming operations
type MediaStreamer interface {
	StreamVideo(fileID string, options StreamOptions) (io.ReadSeeker, error)
	StreamAudio(fileID string, options StreamOptions) (io.ReadSeeker, error)
	GenerateThumbnail(fileID string) ([]byte, error)
	GetMediaInfo(fileID string) (*MediaInfo, error)
	TranscodeVideo(fileID string, profile TranscodeProfile) error
	GetStreamingURL(fileID string, options StreamOptions) (string, error)
}

// StreamOptions contains options for media streaming
type StreamOptions struct {
	Start      int64  `json:"start,omitempty"`
	End        int64  `json:"end,omitempty"`
	Quality    string `json:"quality,omitempty"`
	Format     string `json:"format,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}

// MediaInfo contains information about a media file
type MediaInfo struct {
	FileID       string                 `json:"file_id" db:"file_id"`
	Duration     int64                  `json:"duration" db:"duration"`
	Width        int                    `json:"width" db:"width"`
	Height       int                    `json:"height" db:"height"`
	Codec        string                 `json:"codec" db:"codec"`
	Bitrate      int                    `json:"bitrate" db:"bitrate"`
	FrameRate    float64                `json:"frame_rate,omitempty"`
	AudioCodec   string                 `json:"audio_codec,omitempty"`
	AudioBitrate int                    `json:"audio_bitrate,omitempty"`
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
	IsVideo      bool                   `json:"is_video"`
	IsAudio      bool                   `json:"is_audio"`
	IsImage      bool                   `json:"is_image"`
}

// TranscodeProfile defines video transcoding settings
type TranscodeProfile struct {
	Name       string `json:"name"`
	Resolution string `json:"resolution"`
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
	Quality    string `json:"quality"`
}

// LocalMediaStreamer implements MediaStreamer for local file streaming
type LocalMediaStreamer struct {
	config      *config.FileManagerConfig
	fileManager filemanager.FileManager
	logger      *logrus.Logger
	processor   *MediaProcessor
	thumbnails  *ThumbnailGenerator
}

// NewLocalMediaStreamer creates a new local media streamer
func NewLocalMediaStreamer(cfg *config.FileManagerConfig, fm filemanager.FileManager, logger *logrus.Logger) *LocalMediaStreamer {
	processor := NewMediaProcessor(cfg, logger)
	thumbnails := NewThumbnailGenerator(cfg, logger)

	return &LocalMediaStreamer{
		config:      cfg,
		fileManager: fm,
		logger:      logger,
		processor:   processor,
		thumbnails:  thumbnails,
	}
}

// StreamVideo implements MediaStreamer.StreamVideo
func (lms *LocalMediaStreamer) StreamVideo(fileID string, options StreamOptions) (io.ReadSeeker, error) {
	return lms.streamMedia(fileID, options, "video")
}

// StreamAudio implements MediaStreamer.StreamAudio
func (lms *LocalMediaStreamer) StreamAudio(fileID string, options StreamOptions) (io.ReadSeeker, error) {
	return lms.streamMedia(fileID, options, "audio")
}

// streamMedia handles the common streaming logic for video and audio
func (lms *LocalMediaStreamer) streamMedia(fileID string, options StreamOptions, mediaType string) (io.ReadSeeker, error) {
	// Get file info
	file, err := lms.fileManager.GetFileInfo(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Verify it's a media file
	if !lms.isMediaFile(file.MimeType, mediaType) {
		return nil, fmt.Errorf("file is not a %s file", mediaType)
	}

	// Open the file for streaming
	reader, err := lms.fileManager.Download(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for streaming: %w", err)
	}

	// If it's a ReadSeeker, return it directly
	if seeker, ok := reader.(io.ReadSeeker); ok {
		return seeker, nil
	}

	// Otherwise, we need to convert it (this might require temporary file creation)
	// For now, we'll assume the reader is a file reader
	if fileReader, ok := reader.(*os.File); ok {
		return fileReader, nil
	}

	reader.Close()
	return nil, fmt.Errorf("file does not support seeking required for streaming")
}

// GenerateThumbnail implements MediaStreamer.GenerateThumbnail
func (lms *LocalMediaStreamer) GenerateThumbnail(fileID string) ([]byte, error) {
	file, err := lms.fileManager.GetFileInfo(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if thumbnail already exists in cache
	if thumbnail := lms.getCachedThumbnail(fileID); thumbnail != nil {
		return thumbnail, nil
	}

	// Generate new thumbnail
	reader, err := lms.fileManager.Download(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	thumbnail, err := lms.thumbnails.GenerateFromStream(reader, file.MimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	// Cache the thumbnail
	lms.cacheThumbnail(fileID, thumbnail)

	return thumbnail, nil
}

// GetMediaInfo implements MediaStreamer.GetMediaInfo
func (lms *LocalMediaStreamer) GetMediaInfo(fileID string) (*MediaInfo, error) {
	file, err := lms.fileManager.GetFileInfo(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if media info is already stored
	if info := lms.getCachedMediaInfo(fileID); info != nil {
		return info, nil
	}

	// Extract media information
	reader, err := lms.fileManager.Download(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer reader.Close()

	info, err := lms.processor.ExtractMediaInfo(reader, file.MimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract media info: %w", err)
	}

	info.FileID = fileID

	// Cache the media info
	lms.cacheMediaInfo(info)

	return info, nil
}

// TranscodeVideo implements MediaStreamer.TranscodeVideo
func (lms *LocalMediaStreamer) TranscodeVideo(fileID string, profile TranscodeProfile) error {
	lms.logger.Infof("Starting video transcoding for file %s with profile %s", fileID, profile.Name)

	// Get file info from file manager
	fileInfo, err := lms.fileManager.GetFileInfo(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if FFmpeg is available
	if err := lms.checkFFmpeg(); err != nil {
		return fmt.Errorf("FFmpeg not available: %w", err)
	}

	// Generate output filename based on profile
	inputPath := fileInfo.Path
	outputDir := filepath.Dir(inputPath)
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputExt := profile.Format
	if outputExt == "" {
		outputExt = "mp4" // Default to MP4
	}
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.%s", baseName, profile.Name, outputExt))

	// Build FFmpeg command arguments
	args := lms.buildFFmpegArgs(inputPath, outputPath, profile)

	// Execute FFmpeg
	lms.logger.Debugf("Running FFmpeg with args: %v", args)
	cmd := exec.Command("ffmpeg", args...)

	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		lms.logger.WithError(err).Errorf("FFmpeg transcoding failed: %s", string(output))
		return fmt.Errorf("transcoding failed: %w", err)
	}

	lms.logger.Infof("Video transcoding completed successfully for file %s", fileID)
	return nil
}

// checkFFmpeg verifies that FFmpeg is available on the system
func (lms *LocalMediaStreamer) checkFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("FFmpeg is not installed or not in PATH")
	}
	return nil
}

// buildFFmpegArgs constructs FFmpeg command arguments based on the transcode profile
func (lms *LocalMediaStreamer) buildFFmpegArgs(inputPath, outputPath string, profile TranscodeProfile) []string {
	args := []string{
		"-i", inputPath, // Input file
		"-y",                   // Overwrite output file if it exists
		"-loglevel", "warning", // Reduce verbose output
	}

	// Video codec settings
	if profile.Quality != "" {
		switch profile.Quality {
		case "low":
			args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "28")
		case "medium":
			args = append(args, "-c:v", "libx264", "-preset", "medium", "-crf", "23")
		case "high":
			args = append(args, "-c:v", "libx264", "-preset", "slow", "-crf", "18")
		default:
			args = append(args, "-c:v", "libx264", "-preset", "medium", "-crf", "23")
		}
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "medium")
	}

	// Resolution settings
	if profile.Resolution != "" {
		args = append(args, "-s", profile.Resolution)
	}

	// Bitrate settings
	if profile.Bitrate > 0 {
		args = append(args, "-b:v", fmt.Sprintf("%dk", profile.Bitrate))
	}

	// Audio codec settings
	args = append(args, "-c:a", "aac", "-b:a", "128k")

	// Format-specific settings
	switch profile.Format {
	case "mp4":
		args = append(args, "-f", "mp4", "-movflags", "+faststart")
	case "webm":
		args = append(args, "-f", "webm", "-c:v", "libvp9", "-c:a", "libvorbis")
	case "avi":
		args = append(args, "-f", "avi")
	default:
		// Default to MP4
		args = append(args, "-f", "mp4", "-movflags", "+faststart")
	}

	// Output file
	args = append(args, outputPath)

	return args
}

// GetStreamingURL implements MediaStreamer.GetStreamingURL
func (lms *LocalMediaStreamer) GetStreamingURL(fileID string, options StreamOptions) (string, error) {
	// Generate a streaming URL (this would typically be used with a streaming server)
	baseURL := "/api/media/" + fileID + "/stream"

	var params []string
	if options.Quality != "" {
		params = append(params, "quality="+options.Quality)
	}
	if options.Format != "" {
		params = append(params, "format="+options.Format)
	}
	if options.Start > 0 {
		params = append(params, "start="+strconv.FormatInt(options.Start, 10))
	}
	if options.End > 0 {
		params = append(params, "end="+strconv.FormatInt(options.End, 10))
	}

	if len(params) > 0 {
		baseURL += "?" + strings.Join(params, "&")
	}

	return baseURL, nil
}

// Helper methods

// isMediaFile checks if a file is a media file of the specified type
func (lms *LocalMediaStreamer) isMediaFile(mimeType, mediaType string) bool {
	switch mediaType {
	case "video":
		return strings.HasPrefix(mimeType, "video/")
	case "audio":
		return strings.HasPrefix(mimeType, "audio/")
	case "image":
		return strings.HasPrefix(mimeType, "image/")
	default:
		return strings.HasPrefix(mimeType, "video/") ||
			strings.HasPrefix(mimeType, "audio/") ||
			strings.HasPrefix(mimeType, "image/")
	}
}

// getCachedThumbnail retrieves a cached thumbnail
func (lms *LocalMediaStreamer) getCachedThumbnail(fileID string) []byte {
	cachePath := filepath.Join(lms.config.Media.CachePath, "thumbnails", fileID+".jpg")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}

	return data
}

// cacheThumbnail stores a thumbnail in cache
func (lms *LocalMediaStreamer) cacheThumbnail(fileID string, thumbnail []byte) {
	cachePath := filepath.Join(lms.config.Media.CachePath, "thumbnails")
	os.MkdirAll(cachePath, 0755)

	filePath := filepath.Join(cachePath, fileID+".jpg")
	os.WriteFile(filePath, thumbnail, 0644)
}

// getCachedMediaInfo retrieves cached media information
func (lms *LocalMediaStreamer) getCachedMediaInfo(fileID string) *MediaInfo {
	// This would typically query a database
	// For now, return nil to always extract fresh info
	return nil
}

// cacheMediaInfo stores media information
func (lms *LocalMediaStreamer) cacheMediaInfo(info *MediaInfo) {
	// This would typically store in a database
	// For now, just log it
	lms.logger.Debugf("Caching media info for file %s", info.FileID)
}

// GetDefaultTranscodeProfiles returns default transcoding profiles
func GetDefaultTranscodeProfiles() []TranscodeProfile {
	return []TranscodeProfile{
		{
			Name:       "720p",
			Resolution: "1280x720",
			Bitrate:    2000,
			Format:     "mp4",
			Quality:    "medium",
		},
		{
			Name:       "480p",
			Resolution: "854x480",
			Bitrate:    1000,
			Format:     "mp4",
			Quality:    "medium",
		},
		{
			Name:       "360p",
			Resolution: "640x360",
			Bitrate:    500,
			Format:     "mp4",
			Quality:    "low",
		},
	}
}

// RangeReadSeeker wraps an io.ReadSeeker to support HTTP range requests
type RangeReadSeeker struct {
	reader io.ReadSeeker
	start  int64
	end    int64
	pos    int64
}

// NewRangeReadSeeker creates a new range reader
func NewRangeReadSeeker(reader io.ReadSeeker, start, end int64) *RangeReadSeeker {
	rrs := &RangeReadSeeker{
		reader: reader,
		start:  start,
		end:    end,
		pos:    0,
	}

	// Seek to start position
	reader.Seek(start, 0)

	return rrs
}

// Read implements io.Reader
func (rrs *RangeReadSeeker) Read(p []byte) (n int, err error) {
	// Check if we've reached the end of the range
	remaining := rrs.end - rrs.start - rrs.pos
	if remaining <= 0 {
		return 0, io.EOF
	}

	// Limit read size to remaining bytes
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err = rrs.reader.Read(p)
	rrs.pos += int64(n)

	return n, err
}

// Seek implements io.Seeker
func (rrs *RangeReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64

	switch whence {
	case 0: // Relative to start of range
		abs = rrs.start + offset
	case 1: // Relative to current position
		abs = rrs.start + rrs.pos + offset
	case 2: // Relative to end of range
		abs = rrs.end + offset
	default:
		return 0, fmt.Errorf("invalid whence value")
	}

	// Seek in the underlying reader
	newPos, err := rrs.reader.Seek(abs, 0)
	if err != nil {
		return 0, err
	}

	rrs.pos = newPos - rrs.start
	return rrs.pos, nil
}

// Close implements io.Closer if the underlying reader supports it
func (rrs *RangeReadSeeker) Close() error {
	if closer, ok := rrs.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
