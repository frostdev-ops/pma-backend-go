package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/speech"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SpeechHandlers contains speech-related HTTP handlers
type SpeechHandlers struct {
	service    interface{} // Can be *speech.Service or *speech.StreamingTTSService
	logger     *logrus.Logger
	configRepo repositories.ConfigRepository
}

// NewSpeechHandlers creates new speech handlers
func NewSpeechHandlers(service interface{}, logger *logrus.Logger, configRepo repositories.ConfigRepository) *SpeechHandlers {
	return &SpeechHandlers{
		service:    service,
		logger:     logger,
		configRepo: configRepo,
	}
}

// getBaseService returns the underlying *speech.Service regardless of whether it's embedded in StreamingTTSService
func (h *SpeechHandlers) getBaseService() *speech.Service {
	switch s := h.service.(type) {
	case *speech.Service:
		return s
	case *speech.StreamingTTSService:
		return s.Service // Access the embedded service
	default:
		return nil
	}
}

// TextToSpeech handles TTS requests
// @Summary Convert text to speech
// @Description Converts provided text to speech audio using TTS models
// @Tags Speech
// @Accept json
// @Produce json
// @Param request body speech.TTSRequest true "TTS request"
// @Success 200 {object} speech.TTSResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/tts [post]
func (h *SpeechHandlers) TextToSpeech(c *gin.Context) {
	if !h.getBaseService().TTSEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "TTS service is disabled")
		return
	}

	var req speech.TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate text length
	if len(req.Text) == 0 {
		utils.SendError(c, http.StatusBadRequest, "Text is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	h.logger.WithFields(logrus.Fields{
		"text_length": len(req.Text),
		"voice":       req.Voice,
		"language":    req.Language,
	}).Info("Processing TTS request")

	response, err := h.getBaseService().TextToSpeech(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("TTS request failed")
		utils.SendError(c, http.StatusInternalServerError, "TTS processing failed")
		return
	}

	utils.SendSuccess(c, response)
}

// TextToSpeechStreaming handles streaming TTS requests
// @Summary Convert text to speech with streaming support
// @Description Converts provided text to speech using multi-instance streaming for improved performance
// @Tags Speech
// @Accept json
// @Produce json
// @Param request body speech.StreamingTTSRequest true "Streaming TTS request"
// @Success 200 {object} speech.StreamingTTSResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/tts/streaming [post]
func (h *SpeechHandlers) TextToSpeechStreaming(c *gin.Context) {
	// Check if streaming service is available
	streamingService, ok := h.service.(*speech.StreamingTTSService)
	if !ok || !streamingService.IsStreamingEnabled() {
		// Fall back to regular TTS
		h.TextToSpeech(c)
		return
	}

	var req speech.StreamingTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate text length
	if len(req.Text) == 0 {
		utils.SendError(c, http.StatusBadRequest, "Text is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute) // Longer timeout for streaming
	defer cancel()

	h.logger.WithFields(logrus.Fields{
		"text_length":    len(req.Text),
		"voice":          req.Voice,
		"language":       req.Language,
		"streaming_mode": req.StreamingMode,
		"quality":        req.Quality,
	}).Info("Processing streaming TTS request")

	response, err := streamingService.TextToSpeechStreaming(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Streaming TTS request failed")
		utils.SendError(c, http.StatusInternalServerError, "Streaming TTS processing failed")
		return
	}

	utils.SendSuccess(c, response)
}

// GetStreamingStatus returns the status of a streaming TTS session
// @Summary Get streaming TTS status
// @Description Returns the current status and progress of a streaming TTS session
// @Tags Speech
// @Param stream_id path string true "Stream ID"
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Router /api/v1/speech/tts/streaming/{stream_id}/status [get]
func (h *SpeechHandlers) GetStreamingStatus(c *gin.Context) {
	streamID := c.Param("stream_id")
	if streamID == "" {
		utils.SendError(c, http.StatusBadRequest, "Stream ID is required")
		return
	}

	streamingService, ok := h.service.(*speech.StreamingTTSService)
	if !ok {
		utils.SendError(c, http.StatusServiceUnavailable, "Streaming service not available")
		return
	}

	// Get active streams
	activeStreams := streamingService.GetActiveStreams()
	isActive := false
	for _, id := range activeStreams {
		if id == streamID {
			isActive = true
			break
		}
	}

	status := gin.H{
		"stream_id": streamID,
		"is_active": isActive,
		"timestamp": time.Now(),
	}

	utils.SendSuccess(c, status)
}

// CancelStreamingSession cancels an active streaming TTS session
// @Summary Cancel streaming TTS session
// @Description Cancels an active streaming TTS session
// @Tags Speech
// @Param stream_id path string true "Stream ID"
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Router /api/v1/speech/tts/streaming/{stream_id}/cancel [post]
func (h *SpeechHandlers) CancelStreamingSession(c *gin.Context) {
	streamID := c.Param("stream_id")
	if streamID == "" {
		utils.SendError(c, http.StatusBadRequest, "Stream ID is required")
		return
	}

	streamingService, ok := h.service.(*speech.StreamingTTSService)
	if !ok {
		utils.SendError(c, http.StatusServiceUnavailable, "Streaming service not available")
		return
	}

	if err := streamingService.CancelStream(streamID); err != nil {
		h.logger.WithError(err).WithField("stream_id", streamID).Error("Failed to cancel stream")
		utils.SendError(c, http.StatusNotFound, "Stream not found or could not be canceled")
		return
	}

	h.logger.WithField("stream_id", streamID).Info("Stream canceled successfully")
	utils.SendSuccess(c, gin.H{"message": "Stream canceled successfully"})
}

// GetStreamingMetrics returns streaming performance metrics
// @Summary Get streaming TTS metrics
// @Description Returns performance metrics and statistics for the streaming TTS system
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Router /api/v1/speech/tts/streaming/metrics [get]
func (h *SpeechHandlers) GetStreamingMetrics(c *gin.Context) {
	streamingService, ok := h.service.(*speech.StreamingTTSService)
	if !ok {
		utils.SendError(c, http.StatusServiceUnavailable, "Streaming service not available")
		return
	}

	metrics := streamingService.GetStreamingMetrics()
	utils.SendSuccess(c, metrics)
}

// SpeechToText handles STT requests for uploaded audio files
// @Summary Convert speech to text
// @Description Converts uploaded audio file to text using STT models
// @Tags Speech
// @Accept multipart/form-data
// @Produce json
// @Param audio_file formData file true "Audio file"
// @Param model formData string false "STT model"
// @Param language formData string false "Language code"
// @Param use_autocorrect formData boolean false "Use autocorrect"
// @Success 200 {object} speech.STTResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/stt [post]
func (h *SpeechHandlers) SpeechToText(c *gin.Context) {
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service is disabled")
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Audio file is required")
		return
	}
	defer file.Close()

	// Validate file size (10MB max)
	if header.Size > 10*1024*1024 {
		utils.SendError(c, http.StatusBadRequest, "File size exceeds 10MB limit")
		return
	}

	// Save uploaded file
	audioPath, err := h.getBaseService().SaveUploadedAudio(header.Filename, file)
	if err != nil {
		h.logger.WithError(err).Error("Failed to save uploaded audio")
		utils.SendError(c, http.StatusInternalServerError, "Failed to save audio file")
		return
	}

	// Clean up uploaded file after processing
	defer func() {
		if err := os.Remove(audioPath); err != nil {
			h.logger.WithError(err).WithField("file", audioPath).Warn("Failed to clean up uploaded file")
		}
	}()

	// Build STT request
	req := speech.STTRequest{
		AudioFile: audioPath,
		Model:     c.PostForm("model"),
		Language:  c.PostForm("language"),
	}

	// Parse autocorrect flag
	if autocorrectStr := c.PostForm("use_autocorrect"); autocorrectStr != "" {
		if autocorrect, err := strconv.ParseBool(autocorrectStr); err == nil {
			req.UseAutocorrect = autocorrect
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	h.logger.WithFields(logrus.Fields{
		"filename":        header.Filename,
		"size":            header.Size,
		"model":           req.Model,
		"language":        req.Language,
		"use_autocorrect": req.UseAutocorrect,
	}).Info("Processing STT request")

	response, err := h.getBaseService().SpeechToText(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("STT request failed")
		utils.SendError(c, http.StatusInternalServerError, "STT processing failed")
		return
	}

	utils.SendSuccess(c, response)
}

// RecordAndTranscribe handles live recording and transcription
// @Summary Record audio and transcribe to text
// @Description Records audio from microphone and transcribes to text
// @Tags Speech
// @Accept json
// @Produce json
// @Param model query string false "STT model"
// @Param language query string false "Language code"
// @Param use_autocorrect query boolean false "Use autocorrect"
// @Success 200 {object} speech.STTResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/record [post]
func (h *SpeechHandlers) RecordAndTranscribe(c *gin.Context) {
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service is disabled")
		return
	}

	// Get query parameters
	model := c.Query("model")
	language := c.Query("language")
	useAutocorrect := c.Query("use_autocorrect") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second) // Longer timeout for recording
	defer cancel()

	h.logger.WithFields(logrus.Fields{
		"model":           model,
		"language":        language,
		"use_autocorrect": useAutocorrect,
	}).Info("Starting audio recording and transcription")

	response, err := h.getBaseService().RecordAndTranscribe(ctx, model, language, useAutocorrect)
	if err != nil {
		h.logger.WithError(err).Error("Recording and transcription failed")
		utils.SendError(c, http.StatusInternalServerError, "Recording failed")
		return
	}

	utils.SendSuccess(c, response)
}

// GetTTSModels returns available TTS models
// @Summary Get available TTS models
// @Description Returns a list of available TTS models
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=[]string}
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/tts/models [get]
func (h *SpeechHandlers) GetTTSModels(c *gin.Context) {
	if !h.getBaseService().TTSEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "TTS service is disabled")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	models, err := h.getBaseService().GetTTSModels(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get TTS models")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve TTS models")
		return
	}

	utils.SendSuccess(c, gin.H{"models": models})
}

// GetAudioDevices returns available audio devices
// @Summary Get available audio devices
// @Description Returns a list of available audio input/output devices
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=[]map[string]interface{}}
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/devices [get]
func (h *SpeechHandlers) GetAudioDevices(c *gin.Context) {
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	devices, err := h.getBaseService().GetAudioDevices(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audio devices")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve audio devices")
		return
	}

	utils.SendSuccess(c, gin.H{"devices": devices})
}

// GetVoices returns available TTS voices
// @Summary Get available TTS voices
// @Description Returns a list of available TTS voices
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=[]map[string]interface{}}
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/voices [get]
func (h *SpeechHandlers) GetVoices(c *gin.Context) {
	if !h.getBaseService().TTSEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "TTS service is disabled")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	voices, err := h.getBaseService().GetVoices(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get voices")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve voices")
		return
	}

	utils.SendSuccess(c, voices)
}

// GetVoiceSpeakers handles requests for speakers of a specific voice
// @Summary Get speakers for a voice
// @Description Get available speakers for a multi-speaker voice model
// @Tags Speech
// @Param voice_name path string true "Voice name"
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 503 {object} utils.ErrorResponse
// @Router /api/v1/speech/voices/{voice_name}/speakers [get]
func (h *SpeechHandlers) GetVoiceSpeakers(c *gin.Context) {
	if !h.getBaseService().TTSEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "TTS service is disabled")
		return
	}

	voiceName := c.Param("voice_name")
	if voiceName == "" {
		utils.SendError(c, http.StatusBadRequest, "Voice name is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	speakers, err := h.getBaseService().GetVoiceSpeakers(ctx, voiceName)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to get speakers for voice: %s", voiceName)
		if strings.Contains(err.Error(), "not found") {
			utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Voice '%s' not found", voiceName))
		} else if strings.Contains(err.Error(), "not a multi-speaker") {
			utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Voice '%s' is not a multi-speaker model", voiceName))
		} else {
			utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve speakers")
		}
		return
	}

	utils.SendSuccess(c, speakers)
}

// DownloadAudio serves generated TTS audio files
// @Summary Download TTS audio file
// @Description Downloads generated TTS audio file
// @Tags Speech
// @Param filename path string true "Audio filename"
// @Produce audio/wav
// @Success 200 {file} file
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// REMOVED @Router annotation to prevent auto-registration in protected group
func (h *SpeechHandlers) DownloadAudio(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.SendError(c, http.StatusBadRequest, "Filename is required")
		return
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		utils.SendError(c, http.StatusBadRequest, "Invalid filename")
		return
	}

	// Try TTS output directory first
	ttsDir := h.GetTTSOutputDir()
	filePath := filepath.Join(ttsDir, filename)
	h.logger.WithFields(logrus.Fields{
		"filename":  filename,
		"tts_dir":   ttsDir,
		"file_path": filePath,
	}).Info("🔍 Checking TTS file path")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		h.logger.WithFields(logrus.Fields{
			"file_path": filePath,
			"error":     err,
		}).Warn("❌ TTS file not found, trying STT directory")

		// Try STT output directory
		sttDir := h.GetSTTOutputDir()
		filePath = filepath.Join(sttDir, filename)
		h.logger.WithFields(logrus.Fields{
			"stt_dir":   sttDir,
			"file_path": filePath,
		}).Info("🔍 Checking STT file path")

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			h.logger.WithFields(logrus.Fields{
				"file_path": filePath,
				"error":     err,
			}).Error("❌ Audio file not found in either directory")
			utils.SendError(c, http.StatusNotFound, "Audio file not found")
			return
		}
	}

	h.logger.WithFields(logrus.Fields{
		"file_path": filePath,
	}).Info("✅ Audio file found, serving")

	// Determine content type
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := "audio/wav" // default
	switch ext {
	case ".mp3":
		contentType = "audio/mpeg"
	case ".m4a":
		contentType = "audio/mp4"
	case ".ogg":
		contentType = "audio/ogg"
	}

	h.logger.WithFields(logrus.Fields{
		"filename":     filename,
		"path":         filePath,
		"content_type": contentType,
	}).Debug("Serving audio file")

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.File(filePath)
}

// GetSpeechStatus returns the status of speech services
// @Summary Get speech services status
// @Description Returns the status and configuration of speech services
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Router /api/v1/speech/status [get]
func (h *SpeechHandlers) GetSpeechStatus(c *gin.Context) {
	status := gin.H{
		"enabled":     h.getBaseService().Enabled(),
		"tts_enabled": h.getBaseService().TTSEnabled(),
		"stt_enabled": h.getBaseService().STTEnabled(),
	}

	utils.SendSuccess(c, status)
}

// CleanupTempFiles cleans up old temporary files
// @Summary Cleanup temporary audio files
// @Description Removes old temporary audio files older than specified age
// @Tags Speech
// @Param max_age query string false "Maximum age (e.g., '24h', '7d')" default("24h")
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/cleanup [post]
func (h *SpeechHandlers) CleanupTempFiles(c *gin.Context) {
	maxAgeStr := c.DefaultQuery("max_age", "24h")

	maxAge, err := time.ParseDuration(maxAgeStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid max_age format")
		return
	}

	if err := h.getBaseService().CleanupTempFiles(maxAge); err != nil {
		h.logger.WithError(err).Error("Failed to cleanup temp files")
		utils.SendError(c, http.StatusInternalServerError, "Cleanup failed")
		return
	}

	h.logger.WithField("max_age", maxAge).Info("Cleaned up temp files")
	utils.SendSuccess(c, gin.H{"message": "Cleanup completed"})
}

// SpeechHealthCheck performs a health check of speech services
// @Summary Health check for speech services
// @Description Performs a health check of the speech services
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/health [get]
func (h *SpeechHandlers) SpeechHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	health := gin.H{
		"status":      "healthy",
		"enabled":     h.getBaseService().Enabled(),
		"tts_enabled": h.getBaseService().TTSEnabled(),
		"stt_enabled": h.getBaseService().STTEnabled(),
		"timestamp":   time.Now().UTC(),
	}

	overallHealthy := true

	// Test TTS if enabled
	if h.getBaseService().TTSEnabled() {
		ttsHealth := gin.H{"status": "healthy"}

		// Try to get models as a health check
		if _, err := h.getBaseService().GetTTSModels(ctx); err != nil {
			ttsHealth["status"] = "unhealthy"
			ttsHealth["error"] = err.Error()
			overallHealthy = false
		}

		health["tts"] = ttsHealth
	}

	// Enhanced STT health check
	if h.getBaseService().STTEnabled() {
		sttHealth := gin.H{"status": "healthy"}
		
		// Check STT components
		healthChecker := speech.NewHealthChecker(h.getBaseService())
		sttHealthStatus := healthChecker.CheckHealth()
		
		if sttHealthStatus.Overall != "healthy" {
			sttHealth["status"] = sttHealthStatus.Overall
			sttHealth["components"] = sttHealthStatus.Components
			sttHealth["suggestions"] = sttHealthStatus.Suggestions
			
			if sttHealthStatus.LastError != nil {
				sttHealth["last_error"] = sttHealthStatus.LastError
			}
			
			if sttHealthStatus.Overall == "unhealthy" {
				overallHealthy = false
			}
		}

		health["stt"] = sttHealth
	}

	// Set overall status
	if !overallHealthy {
		health["status"] = "degraded"
	}

	utils.SendSuccess(c, health)
}

// SpeechSettings represents the speech configuration that can be returned to frontend
type SpeechSettings struct {
	TTSEnabled          bool     `json:"tts_enabled"`
	STTEnabled          bool     `json:"stt_enabled"`
	DefaultTTSModel     string   `json:"default_tts_model"` // Legacy field for backward compatibility
	DefaultVoice        string   `json:"default_voice"`     // New Piper voice field
	DefaultSTTModel     string   `json:"default_stt_model"`
	DefaultLanguage     string   `json:"default_language"`
	AutoCorrect         bool     `json:"autocorrect"`
	MaxTextLength       int      `json:"max_text_length"`
	Timeout             string   `json:"timeout"`
	AudioFormats        []string `json:"audio_formats"`
	SampleRate          int      `json:"sample_rate"`
	DefaultInputDevice  string   `json:"default_input_device"`
	DefaultOutputDevice string   `json:"default_output_device"`
	AutoPlayResponses   bool     `json:"auto_play_responses"` // Legacy field for backward compatibility
	ConversationalFlow  bool     `json:"conversational_flow"` // New conversational flow setting
	SpeechRate          float64  `json:"speech_rate"`         // TTS speech rate
	Volume              float64  `json:"volume"`              // TTS volume
}

// GetSpeechSettings handles retrieving speech configuration
// @Summary Get speech settings
// @Description Retrieves current speech service settings and configuration
// @Tags Speech
// @Produce json
// @Success 200 {object} SpeechSettings
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/settings [get]
func (h *SpeechHandlers) GetSpeechSettings(c *gin.Context) {
	ctx := c.Request.Context()

	// Start with defaults from config
	settings := SpeechSettings{
		TTSEnabled:          h.getBaseService().TTSEnabled(),
		STTEnabled:          h.getBaseService().STTEnabled(),
		DefaultTTSModel:     h.getBaseService().GetDefaultTTSModel(), // Legacy field
		DefaultVoice:        "lessac-high",                           // Default Piper voice
		DefaultSTTModel:     h.getBaseService().GetDefaultSTTModel(),
		DefaultLanguage:     h.getBaseService().GetDefaultLanguage(),
		AutoCorrect:         h.getBaseService().GetAutoCorrectEnabled(),
		MaxTextLength:       h.getBaseService().GetMaxTextLength(),
		Timeout:             h.getBaseService().GetTimeout(),
		AudioFormats:        h.getBaseService().GetSupportedAudioFormats(),
		SampleRate:          h.getBaseService().GetSampleRate(),
		DefaultInputDevice:  "",    // Default to empty (system default)
		DefaultOutputDevice: "",    // Default to empty (system default)
		AutoPlayResponses:   false, // Legacy field - default to disabled
		ConversationalFlow:  false, // Default to disabled
		SpeechRate:          1.0,   // Default speech rate
		Volume:              0.8,   // Default volume
	}

	// Override with any saved user preferences from database
	settingsKeys := []string{
		"speech.tts_enabled",
		"speech.stt_enabled",
		"speech.default_tts_model",
		"speech.default_voice",
		"speech.default_stt_model",
		"speech.default_language",
		"speech.autocorrect",
		"speech.max_text_length",
		"speech.default_input_device",
		"speech.default_output_device",
		"speech.auto_play_responses",
		"speech.conversational_flow",
		"speech.speech_rate",
		"speech.volume",
	}

	for _, key := range settingsKeys {
		if config, err := h.configRepo.Get(ctx, key); err == nil {
			// Parse the JSON value and apply it to settings
			switch key {
			case "speech.tts_enabled":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.TTSEnabled = value
				}
			case "speech.stt_enabled":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.STTEnabled = value
				}
			case "speech.default_tts_model":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil && value != "" {
					settings.DefaultTTSModel = value
				}
			case "speech.default_stt_model":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil && value != "" {
					settings.DefaultSTTModel = value
				}
			case "speech.default_language":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil && value != "" {
					settings.DefaultLanguage = value
				}
			case "speech.autocorrect":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.AutoCorrect = value
				}
			case "speech.max_text_length":
				var value int
				if json.Unmarshal([]byte(config.Value), &value) == nil && value > 0 {
					settings.MaxTextLength = value
				}
			case "speech.default_input_device":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.DefaultInputDevice = value
				}
			case "speech.default_output_device":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.DefaultOutputDevice = value
				}
			case "speech.auto_play_responses":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.AutoPlayResponses = value
				}
			case "speech.default_voice":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil && value != "" {
					settings.DefaultVoice = value
				}
			case "speech.conversational_flow":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					settings.ConversationalFlow = value
				}
			case "speech.speech_rate":
				var value float64
				if json.Unmarshal([]byte(config.Value), &value) == nil && value > 0 {
					settings.SpeechRate = value
				}
			case "speech.volume":
				var value float64
				if json.Unmarshal([]byte(config.Value), &value) == nil && value >= 0 && value <= 1 {
					settings.Volume = value
				}
			}
		}
	}

	utils.SendSuccess(c, settings)
}

// UpdateSpeechSettings handles updating speech configuration
// @Summary Update speech settings
// @Description Updates speech service settings and configuration
// @Tags Speech
// @Accept json
// @Produce json
// @Param settings body SpeechSettings true "Updated speech settings"
// @Success 200 {object} SpeechSettings
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/speech/settings [put]
func (h *SpeechHandlers) UpdateSpeechSettings(c *gin.Context) {
	// Use a map to handle partial updates instead of binding to struct
	var rawSettings map[string]interface{}
	if err := c.ShouldBindJSON(&rawSettings); err != nil {
		h.logger.WithError(err).Error("Failed to bind speech settings JSON")
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"raw_settings": rawSettings,
	}).Info("Speech settings update requested")

	// Save settings to database for persistence
	ctx := c.Request.Context()

	// Only update the fields that were actually provided
	settingsMap := make(map[string]interface{})

	// Map frontend field names to database keys and only include provided fields
	fieldMappings := map[string]string{
		"tts_enabled":           "speech.tts_enabled",
		"stt_enabled":           "speech.stt_enabled",
		"default_tts_model":     "speech.default_tts_model",
		"default_voice":         "speech.default_voice",
		"default_stt_model":     "speech.default_stt_model",
		"default_language":      "speech.default_language",
		"autocorrect":           "speech.autocorrect",
		"max_text_length":       "speech.max_text_length",
		"default_input_device":  "speech.default_input_device",
		"default_output_device": "speech.default_output_device",
		"auto_play_responses":   "speech.auto_play_responses",
		"conversational_flow":   "speech.conversational_flow",
		"speech_rate":           "speech.speech_rate",
		"volume":                "speech.volume",
	}

	// Only add settings that were actually provided in the request
	for frontendKey, dbKey := range fieldMappings {
		if value, exists := rawSettings[frontendKey]; exists {
			settingsMap[dbKey] = value
			h.logger.WithFields(logrus.Fields{
				"field": frontendKey,
				"value": value,
			}).Info("Updating speech setting")
		}
	}

	for key, value := range settingsMap {
		// Convert value to JSON string for storage
		valueBytes, err := json.Marshal(value)
		if err != nil {
			h.logger.WithError(err).WithField("key", key).Error("Failed to marshal setting value")
			continue
		}

		config := &models.SystemConfig{
			Key:         key,
			Value:       string(valueBytes),
			Encrypted:   false,
			Description: fmt.Sprintf("Speech setting: %s", key),
		}

		if err := h.configRepo.Set(ctx, config); err != nil {
			h.logger.WithError(err).WithField("key", key).Error("Failed to save speech setting")
			utils.SendError(c, http.StatusInternalServerError, "Failed to save speech settings")
			return
		}
	}

	h.logger.Info("Speech settings saved successfully")

	// Return the complete current settings by re-reading from database
	// This ensures we return the actual current state, not just the partial update

	// Start with defaults from config
	updatedSettings := SpeechSettings{
		TTSEnabled:          h.getBaseService().TTSEnabled(),
		STTEnabled:          h.getBaseService().STTEnabled(),
		DefaultTTSModel:     h.getBaseService().GetDefaultTTSModel(), // Legacy field
		DefaultVoice:        "lessac-high",                           // Default Piper voice
		DefaultSTTModel:     h.getBaseService().GetDefaultSTTModel(),
		DefaultLanguage:     h.getBaseService().GetDefaultLanguage(),
		AutoCorrect:         h.getBaseService().GetAutoCorrectEnabled(),
		MaxTextLength:       h.getBaseService().GetMaxTextLength(),
		Timeout:             h.getBaseService().GetTimeout(),
		AudioFormats:        h.getBaseService().GetSupportedAudioFormats(),
		SampleRate:          h.getBaseService().GetSampleRate(),
		DefaultInputDevice:  "",    // Default to empty (system default)
		DefaultOutputDevice: "",    // Default to empty (system default)
		AutoPlayResponses:   false, // Legacy field - default to disabled
		ConversationalFlow:  false, // Default to disabled
		SpeechRate:          1.0,   // Default speech rate
		Volume:              0.8,   // Default volume
	}

	// Override with any saved user preferences from database
	settingsKeys := []string{
		"speech.tts_enabled",
		"speech.stt_enabled",
		"speech.default_tts_model",
		"speech.default_voice",
		"speech.default_stt_model",
		"speech.default_language",
		"speech.autocorrect",
		"speech.max_text_length",
		"speech.default_input_device",
		"speech.default_output_device",
		"speech.auto_play_responses",
		"speech.conversational_flow",
		"speech.speech_rate",
		"speech.volume",
	}

	for _, key := range settingsKeys {
		if config, err := h.configRepo.Get(ctx, key); err == nil {
			// Parse the JSON value and apply it to settings
			switch key {
			case "speech.tts_enabled":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					updatedSettings.TTSEnabled = value
				}
			case "speech.stt_enabled":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					updatedSettings.STTEnabled = value
				}
			case "speech.default_voice":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil && value != "" {
					updatedSettings.DefaultVoice = value
				}
			case "speech.conversational_flow":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					updatedSettings.ConversationalFlow = value
				}
			case "speech.speech_rate":
				var value float64
				if json.Unmarshal([]byte(config.Value), &value) == nil && value > 0 {
					updatedSettings.SpeechRate = value
				}
			case "speech.volume":
				var value float64
				if json.Unmarshal([]byte(config.Value), &value) == nil && value > 0 {
					updatedSettings.Volume = value
				}
			case "speech.default_language":
				var value string
				if json.Unmarshal([]byte(config.Value), &value) == nil && value != "" {
					updatedSettings.DefaultLanguage = value
				}
			case "speech.autocorrect":
				var value bool
				if json.Unmarshal([]byte(config.Value), &value) == nil {
					updatedSettings.AutoCorrect = value
				}
				// Add other cases as needed
			}
		}
	}

	utils.SendSuccess(c, updatedSettings)
}

// GetSTTMetrics returns STT service performance metrics
// @Summary Get STT service metrics
// @Description Returns performance metrics and statistics for the STT service
// @Tags Speech 
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 503 {object} utils.ErrorResponse
// @Router /api/v1/speech/stt/metrics [get]
func (h *SpeechHandlers) GetSTTMetrics(c *gin.Context) {
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// For now, return basic metrics based on health status
	healthChecker := speech.NewHealthChecker(h.getBaseService())
	healthStatus := healthChecker.CheckHealth()
	
	metrics := gin.H{
		"service_status": healthStatus.Overall,
		"components":     healthStatus.Components,
		"timestamp":      time.Now().UTC(),
		"suggestions":    healthStatus.Suggestions,
	}
	
	if healthStatus.LastError != nil {
		metrics["last_error"] = healthStatus.LastError
	}

	utils.SendSuccess(c, metrics)
}

// RestartSTTService restarts the STT service components
// @Summary Restart STT service
// @Description Restarts STT service components and clears error states
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 503 {object} utils.ErrorResponse
// @Router /api/v1/speech/stt/restart [post]
func (h *SpeechHandlers) RestartSTTService(c *gin.Context) {
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	h.logger.Info("STT service restart requested")

	// Perform restart operations
	// 1. Clear temporary files
	if err := h.getBaseService().CleanupTempFiles(0); err != nil {
		h.logger.WithError(err).Warn("Failed to cleanup temp files during restart")
	}

	// 2. Perform health check
	healthChecker := speech.NewHealthChecker(h.getBaseService())
	newHealthStatus := healthChecker.CheckHealth()

	result := gin.H{
		"message":      "STT service restart completed",
		"timestamp":    time.Now().UTC(),
		"health_status": newHealthStatus,
	}

	h.logger.WithField("health_status", newHealthStatus.Overall).Info("STT service restart completed")
	utils.SendSuccess(c, result)
}

// ValidateSTTConfiguration validates STT service configuration
// @Summary Validate STT configuration
// @Description Validates STT service configuration and dependencies
// @Tags Speech
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 503 {object} utils.ErrorResponse
// @Router /api/v1/speech/stt/validate [get]
func (h *SpeechHandlers) ValidateSTTConfiguration(c *gin.Context) {
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	healthChecker := speech.NewHealthChecker(h.getBaseService())
	healthStatus := healthChecker.CheckHealth()

	validation := gin.H{
		"overall_status": healthStatus.Overall,
		"components":     healthStatus.Components,
		"suggestions":    healthStatus.Suggestions,
		"timestamp":      time.Now().UTC(),
	}

	// Add configuration details
	validation["configuration"] = gin.H{
		"enabled":            h.getBaseService().STTEnabled(),
		"python_script_path": "/opt/pma/speech/stt.py", // From config
		"default_model":      h.getBaseService().GetDefaultSTTModel(),
		"default_language":   h.getBaseService().GetDefaultLanguage(),
		"timeout":           h.getBaseService().GetTimeout(),
		"autocorrect":       h.getBaseService().GetAutoCorrectEnabled(),
	}

	if healthStatus.Overall == "healthy" {
		validation["message"] = "STT service configuration is valid and ready"
	} else {
		validation["message"] = "STT service configuration has issues that need attention"
	}

	utils.SendSuccess(c, validation)
}

// Helper methods to access service configuration

func (h *SpeechHandlers) GetTTSOutputDir() string {
	return "/opt/pma/data/temp/tts"
}

func (h *SpeechHandlers) GetSTTOutputDir() string {
	return "/opt/pma/data/temp/stt"
}
