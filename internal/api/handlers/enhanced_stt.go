package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/stt"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Enhanced STT API endpoints

// GetSTTStatus returns the status of the enhanced STT service
// GET /api/v1/stt/status
func (h *Handlers) GetSTTStatus(c *gin.Context) {
	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	status := h.enhancedSTTService.GetStatus()
	utils.SendSuccess(c, status)
}

// SetContinuousListening enables or disables continuous listening
// POST /api/v1/stt/continuous-listening
func (h *Handlers) SetContinuousListening(c *gin.Context) {
	var request struct {
		Enabled bool `json:"enabled" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	if err := h.enhancedSTTService.SetContinuousListening(request.Enabled); err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to update continuous listening")
		return
	}

	utils.SendSuccess(c, gin.H{
		"enabled": request.Enabled,
	})
}

// GetAudioDevices returns available audio input devices
// GET /api/v1/speech/devices
func (h *Handlers) GetAudioDevices(c *gin.Context) {
	if h.speechHandler == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Speech service not available")
		return
	}
	h.speechHandler.GetAudioDevices(c)
}

// SetPrimaryAudioDevice sets the primary audio input device
// POST /api/v1/stt/audio-devices/primary
func (h *Handlers) SetPrimaryAudioDevice(c *gin.Context) {
	var request struct {
		DeviceID string `json:"device_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Implementation would call the audio manager to set primary device
	// For now, we'll simulate success

	utils.SendSuccess(c, gin.H{
		"device_id": request.DeviceID,
	})
}

// GetWakeWordStats returns wake word detection statistics
// GET /api/v1/stt/wake-word/stats
func (h *Handlers) GetWakeWordStats(c *gin.Context) {
	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Get wake word statistics from the service
	status := h.enhancedSTTService.GetStatus()
	if stats, exists := status["wake_word_stats"]; exists {
		utils.SendSuccess(c, stats)
	} else {
		utils.SendError(c, http.StatusNotFound, "Wake word statistics not available")
	}
}

// UpdateWakeWordConfig updates wake word configuration
// PUT /api/v1/stt/wake-word/config
func (h *Handlers) UpdateWakeWordConfig(c *gin.Context) {
	var request struct {
		Keywords          []stt.WakeWordKeyword `json:"keywords"`
		SensitivityLevel  float64               `json:"sensitivity_level"`
		MinConfidence     float64               `json:"min_confidence"`
		CooldownPeriod    string                `json:"cooldown_period"`
		PowerSavingMode   bool                  `json:"power_saving_mode"`
		AdaptiveThreshold bool                  `json:"adaptive_threshold"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Update the configuration (this would need to be implemented in the service)
	// For now, we'll simulate success

	utils.SendSuccess(c, gin.H{
		"keywords_count":    len(request.Keywords),
		"sensitivity_level": request.SensitivityLevel,
		"power_saving_mode": request.PowerSavingMode,
	})
}

// AddWakeWordKeyword adds a new wake word keyword
// POST /api/v1/stt/wake-word/keywords
func (h *Handlers) AddWakeWordKeyword(c *gin.Context) {
	var request stt.WakeWordKeyword

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Validate keyword
	if request.Word == "" {
		utils.SendError(c, http.StatusBadRequest, "Keyword word is required")
		return
	}

	if request.Sensitivity < 0.0 || request.Sensitivity > 1.0 {
		utils.SendError(c, http.StatusBadRequest, "Sensitivity must be between 0.0 and 1.0")
		return
	}

	// Set defaults if not provided
	if request.Action == "" {
		request.Action = "start_listening"
	}

	// Implementation would call the wake word engine to add the keyword
	// For now, we'll simulate success

	utils.SendSuccess(c, gin.H{
		"keyword": request.Word,
		"enabled": request.Enabled,
		"action":  request.Action,
	})
}

// RemoveWakeWordKeyword removes a wake word keyword
// DELETE /api/v1/stt/wake-word/keywords/:word
func (h *Handlers) RemoveWakeWordKeyword(c *gin.Context) {
	word := c.Param("word")
	if word == "" {
		utils.SendError(c, http.StatusBadRequest, "Keyword word is required")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Implementation would call the wake word engine to remove the keyword
	// For now, we'll simulate success

	utils.SendSuccess(c, gin.H{
		"keyword": word,
	})
}

// StartListeningSession starts a new listening session
// POST /api/v1/stt/sessions/start
func (h *Handlers) StartListeningSession(c *gin.Context) {
	var request struct {
		Source              string  `json:"source"`
		DeviceID            string  `json:"device_id"`
		Language            string  `json:"language"`
		AutoLanguageDetect  bool    `json:"auto_language_detect"`
		ConfidenceThreshold float64 `json:"confidence_threshold"`
		MaxDuration         string  `json:"max_duration"`
		SilenceTimeout      string  `json:"silence_timeout"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Parse durations
	maxDuration := 5 * time.Minute // Default
	if request.MaxDuration != "" {
		if d, err := time.ParseDuration(request.MaxDuration); err == nil {
			maxDuration = d
		}
	}

	silenceTimeout := 3 * time.Second // Default
	if request.SilenceTimeout != "" {
		if d, err := time.ParseDuration(request.SilenceTimeout); err == nil {
			silenceTimeout = d
		}
	}

	// Set defaults
	if request.Language == "" {
		request.Language = "en"
	}
	if request.ConfidenceThreshold == 0 {
		request.ConfidenceThreshold = 0.7
	}

	// Implementation would start a new STT session
	// For now, we'll simulate session creation
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	utils.SendSuccess(c, gin.H{
		"session_id":      sessionID,
		"source":          request.Source,
		"device_id":       request.DeviceID,
		"language":        request.Language,
		"max_duration":    maxDuration.String(),
		"silence_timeout": silenceTimeout.String(),
	})
}

// StopListeningSession stops an active listening session
// POST /api/v1/stt/sessions/:session_id/stop
func (h *Handlers) StopListeningSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.SendError(c, http.StatusBadRequest, "Session ID is required")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Implementation would stop the specified session
	// For now, we'll simulate success

	utils.SendSuccess(c, gin.H{
		"session_id": sessionID,
	})
}

// GetTranscriptionHistory returns recent transcriptions
// GET /api/v1/stt/transcriptions
func (h *Handlers) GetTranscriptionHistory(c *gin.Context) {
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	source := c.Query("source")
	minConfidence := c.Query("min_confidence")

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Implementation would query transcription history from database
	// For now, we'll simulate some data
	transcriptions := []gin.H{
		{
			"id":                  "txn_1",
			"text":                "Hey assistant, turn on the living room lights",
			"confidence":          0.95,
			"language":            "en",
			"timestamp":           time.Now().Add(-5 * time.Minute),
			"source":              "browser",
			"wake_word_triggered": true,
		},
		{
			"id":                  "txn_2",
			"text":                "What's the weather like today",
			"confidence":          0.87,
			"language":            "en",
			"timestamp":           time.Now().Add(-10 * time.Minute),
			"source":              "usb",
			"wake_word_triggered": true,
		},
	}

	utils.SendSuccess(c, gin.H{
		"transcriptions": transcriptions,
		"count":          len(transcriptions),
		"limit":          limit,
		"offset":         offset,
		"filters": gin.H{
			"source":         source,
			"min_confidence": minConfidence,
		},
	})
}

// DeleteTranscriptionHistory deletes transcription history
// DELETE /api/v1/stt/transcriptions
func (h *Handlers) DeleteTranscriptionHistory(c *gin.Context) {
	var request struct {
		OlderThan       string   `json:"older_than"`       // Duration string like "24h"
		Sources         []string `json:"sources"`          // Filter by sources
		ConfidenceBelow float64  `json:"confidence_below"` // Delete low confidence transcriptions
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.enhancedSTTService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Implementation would delete transcriptions based on criteria
	// For now, we'll simulate deletion

	deletedCount := 15 // Simulated

	utils.SendSuccess(c, gin.H{
		"deleted_count": deletedCount,
		"criteria": gin.H{
			"older_than":       request.OlderThan,
			"sources":          request.Sources,
			"confidence_below": request.ConfidenceBelow,
		},
	})
}

// WebSocket endpoint for real-time audio streaming
// WebSocket /api/v1/stt/stream
func (h *Handlers) STTWebSocketStream(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	if h.enhancedSTTService == nil {
		conn.WriteJSON(gin.H{
			"error": "STT service not available",
		})
		return
	}

	// Generate connection ID
	connectionID := fmt.Sprintf("ws_%d", time.Now().UnixNano())

	h.log.WithField("connection_id", connectionID).Info("STT WebSocket connection established")

	// Send welcome message
	conn.WriteJSON(gin.H{
		"type":          "connection_established",
		"connection_id": connectionID,
		"timestamp":     time.Now(),
		"capabilities": []string{
			"audio_streaming",
			"real_time_transcription",
			"wake_word_detection",
		},
	})

	// Register connection with audio manager
	// This would need to be implemented in the enhanced STT service

	// Handle WebSocket messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			h.log.WithError(err).Debug("WebSocket connection closed")
			break
		}

		// Handle different message types
		msgType, exists := message["type"].(string)
		if !exists {
			conn.WriteJSON(gin.H{
				"error": "Message type required",
			})
			continue
		}

		switch msgType {
		case "audio_data":
			// Handle audio data streaming
			h.handleWebSocketAudioData(conn, message, connectionID)

		case "start_listening":
			// Start listening session
			h.handleWebSocketStartListening(conn, message, connectionID)

		case "stop_listening":
			// Stop listening session
			h.handleWebSocketStopListening(conn, message, connectionID)

		case "configure_wake_word":
			// Configure wake word detection
			h.handleWebSocketConfigureWakeWord(conn, message, connectionID)

		default:
			conn.WriteJSON(gin.H{
				"error": fmt.Sprintf("Unknown message type: %s", msgType),
			})
		}
	}

	// Unregister connection
	h.log.WithField("connection_id", connectionID).Info("STT WebSocket connection closed")
}

// Helper methods for WebSocket handling
func (h *Handlers) handleWebSocketAudioData(conn *websocket.Conn, message map[string]interface{}, connectionID string) {
	// Extract audio data and forward to audio manager
	// Implementation would depend on the audio data format

	conn.WriteJSON(gin.H{
		"type":      "audio_received",
		"timestamp": time.Now(),
	})
}

func (h *Handlers) handleWebSocketStartListening(conn *websocket.Conn, message map[string]interface{}, connectionID string) {
	// Start a listening session for this WebSocket connection

	conn.WriteJSON(gin.H{
		"type":       "listening_started",
		"session_id": fmt.Sprintf("ws_session_%s", connectionID),
		"timestamp":  time.Now(),
	})
}

func (h *Handlers) handleWebSocketStopListening(conn *websocket.Conn, message map[string]interface{}, connectionID string) {
	// Stop the listening session for this WebSocket connection

	conn.WriteJSON(gin.H{
		"type":      "listening_stopped",
		"timestamp": time.Now(),
	})
}

func (h *Handlers) handleWebSocketConfigureWakeWord(conn *websocket.Conn, message map[string]interface{}, connectionID string) {
	// Configure wake word detection for this connection

	conn.WriteJSON(gin.H{
		"type":      "wake_word_configured",
		"timestamp": time.Now(),
	})
}
