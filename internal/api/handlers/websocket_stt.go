package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/speech"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocket upgrader for STT streaming
var sttUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
	ReadBufferSize:  1024 * 4,  // 4KB read buffer
	WriteBufferSize: 1024 * 4,  // 4KB write buffer
}

// STTWebSocketMessage represents messages sent over the STT WebSocket
type STTWebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	TrackID string      `json:"track_id,omitempty"`
}

// STTStreamConfig represents configuration for STT streaming session
type STTStreamConfig struct {
	Model           string  `json:"model,omitempty"`
	Language        string  `json:"language,omitempty"`
	UseAutocorrect  bool    `json:"use_autocorrect,omitempty"`
	ChunkDuration   int     `json:"chunk_duration,omitempty"`   // milliseconds
	SilenceTimeout  int     `json:"silence_timeout,omitempty"`  // milliseconds
	MaxDuration     int     `json:"max_duration,omitempty"`     // seconds
	AutoFinalize    bool    `json:"auto_finalize,omitempty"`
}

// STTStreamSession represents an active STT streaming session
type STTStreamSession struct {
	ID              string
	Config          STTStreamConfig
	Conn            *websocket.Conn
	Service         *speech.Service
	Logger          *logrus.Logger
	StartTime       time.Time
	LastActivity    time.Time
	AudioBuffer     []byte
	IsProcessing    bool
	TempFilePath    string
	ResultChannel   chan *speech.STTResponse
	ErrorChannel    chan error
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// STTWebSocketHandler handles WebSocket connections for real-time STT
// @Summary WebSocket endpoint for real-time speech-to-text
// @Description Establishes WebSocket connection for streaming audio data and receiving transcriptions
// @Tags Speech
// @Param model query string false "STT model to use"
// @Param language query string false "Language code"
// @Param use_autocorrect query boolean false "Enable autocorrect"
// @Success 101 {string} string "WebSocket connection established"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 503 {object} utils.ErrorResponse
// @Router /api/v1/speech/stt/stream [get]
func (h *SpeechHandlers) STTWebSocketHandler(c *gin.Context) {
	// Check if STT is enabled
	if !h.getBaseService().STTEnabled() {
		utils.SendError(c, http.StatusServiceUnavailable, "STT service not available")
		return
	}

	// Parse query parameters for configuration
	config := STTStreamConfig{
		Model:          c.DefaultQuery("model", "base"),
		Language:       c.DefaultQuery("language", "en"),
		UseAutocorrect: c.DefaultQuery("use_autocorrect", "false") == "true",
		ChunkDuration:  1000, // 1 second chunks
		SilenceTimeout: 3000, // 3 seconds of silence
		MaxDuration:    300,  // 5 minutes max
		AutoFinalize:   true,
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := sttUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		utils.SendError(c, http.StatusBadRequest, "Failed to upgrade WebSocket connection")
		return
	}

	// Create session
	sessionID := generateSessionID()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.MaxDuration)*time.Second)
	
	session := &STTStreamSession{
		ID:            sessionID,
		Config:        config,
		Conn:          conn,
		Service:       h.getBaseService(),
		Logger:        h.logger,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		AudioBuffer:   make([]byte, 0),
		ResultChannel: make(chan *speech.STTResponse, 10),
		ErrorChannel:  make(chan error, 10),
		ctx:           ctx,
		cancel:        cancel,
	}

	h.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"config":     config,
	}).Info("STT WebSocket session started")

	// Handle the session
	h.handleSTTWebSocketSession(session)
}

// handleSTTWebSocketSession manages the WebSocket session lifecycle
func (h *SpeechHandlers) handleSTTWebSocketSession(session *STTStreamSession) {
	defer func() {
		session.cancel()
		session.Conn.Close()
		
		// Clean up temporary files
		if session.TempFilePath != "" {
			os.Remove(session.TempFilePath)
		}
		
		h.logger.WithField("session_id", session.ID).Info("STT WebSocket session ended")
	}()

	// Send initial connection confirmation
	h.sendSTTMessage(session, STTWebSocketMessage{
		Type: "connected",
		Data: map[string]interface{}{
			"session_id": session.ID,
			"config":     session.Config,
		},
	})

	// Start goroutines for handling messages
	go h.handleSTTResults(session)
	go h.monitorSTTSession(session)

	// Main message loop
	for {
		select {
		case <-session.ctx.Done():
			h.sendSTTMessage(session, STTWebSocketMessage{
				Type:  "error",
				Error: "Session timeout",
			})
			return

		default:
			// Set read deadline
			session.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			
			// Read message
			_, messageData, err := session.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.WithError(err).Error("WebSocket read error")
				}
				return
			}

			// Parse message
			var msg STTWebSocketMessage
			if err := json.Unmarshal(messageData, &msg); err != nil {
				h.logger.WithError(err).Error("Failed to parse WebSocket message")
				h.sendSTTMessage(session, STTWebSocketMessage{
					Type:  "error",
					Error: "Invalid message format",
				})
				continue
			}

			// Handle message
			if err := h.handleSTTWebSocketMessage(session, &msg); err != nil {
				h.logger.WithError(err).Error("Failed to handle WebSocket message")
				h.sendSTTMessage(session, STTWebSocketMessage{
					Type:  "error",
					Error: err.Error(),
				})
			}
		}
	}
}

// handleSTTWebSocketMessage processes incoming WebSocket messages
func (h *SpeechHandlers) handleSTTWebSocketMessage(session *STTStreamSession, msg *STTWebSocketMessage) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.LastActivity = time.Now()

	switch msg.Type {
	case "audio_data":
		return h.handleAudioData(session, msg)
	
	case "finalize":
		return h.finalizeTranscription(session)
	
	case "reset":
		return h.resetSession(session)
	
	case "ping":
		h.sendSTTMessage(session, STTWebSocketMessage{Type: "pong"})
		return nil
	
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleAudioData processes incoming audio data
func (h *SpeechHandlers) handleAudioData(session *STTStreamSession, msg *STTWebSocketMessage) error {
	// Extract audio data
	audioDataStr, ok := msg.Data.(string)
	if !ok {
		return fmt.Errorf("invalid audio data format")
	}

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(audioDataStr)
	if err != nil {
		return fmt.Errorf("failed to decode audio data: %w", err)
	}

	// Append to buffer
	session.AudioBuffer = append(session.AudioBuffer, audioData...)

	// Check if we have enough data to process
	if len(session.AudioBuffer) >= session.Config.ChunkDuration*16*2 { // Assuming 16kHz, 16-bit
		return h.processAudioChunk(session)
	}

	return nil
}

// processAudioChunk processes accumulated audio data
func (h *SpeechHandlers) processAudioChunk(session *STTStreamSession) error {
	if session.IsProcessing {
		return nil // Skip if already processing
	}

	session.IsProcessing = true
	defer func() {
		session.IsProcessing = false
	}()

	// Create temporary file for audio data
	tempFile, err := h.createTempAudioFile(session.AudioBuffer)
	if err != nil {
		return fmt.Errorf("failed to create temp audio file: %w", err)
	}
	defer os.Remove(tempFile)

	// Process with STT
	go func() {
		sttReq := &speech.STTRequest{
			AudioFile:      tempFile,
			Model:          session.Config.Model,
			Language:       session.Config.Language,
			UseAutocorrect: session.Config.UseAutocorrect,
		}

		result, err := session.Service.SpeechToText(session.ctx, sttReq)
		if err != nil {
			session.ErrorChannel <- err
		} else {
			session.ResultChannel <- result
		}
	}()

	// Clear processed audio from buffer
	session.AudioBuffer = session.AudioBuffer[:0]

	return nil
}

// finalizeTranscription processes final transcription
func (h *SpeechHandlers) finalizeTranscription(session *STTStreamSession) error {
	if len(session.AudioBuffer) > 0 {
		return h.processAudioChunk(session)
	}
	
	h.sendSTTMessage(session, STTWebSocketMessage{
		Type: "finalized",
		Data: map[string]interface{}{
			"session_id": session.ID,
			"timestamp":  time.Now().Unix(),
		},
	})
	
	return nil
}

// resetSession resets the session state
func (h *SpeechHandlers) resetSession(session *STTStreamSession) error {
	session.AudioBuffer = session.AudioBuffer[:0]
	session.IsProcessing = false
	
	if session.TempFilePath != "" {
		os.Remove(session.TempFilePath)
		session.TempFilePath = ""
	}
	
	h.sendSTTMessage(session, STTWebSocketMessage{
		Type: "reset",
		Data: map[string]interface{}{
			"session_id": session.ID,
		},
	})
	
	return nil
}

// handleSTTResults processes STT results
func (h *SpeechHandlers) handleSTTResults(session *STTStreamSession) {
	for {
		select {
		case <-session.ctx.Done():
			return
			
		case result := <-session.ResultChannel:
			h.sendSTTMessage(session, STTWebSocketMessage{
				Type: "transcription",
				Data: result,
			})
			
		case err := <-session.ErrorChannel:
			h.sendSTTMessage(session, STTWebSocketMessage{
				Type:  "error",
				Error: err.Error(),
			})
		}
	}
}

// monitorSTTSession monitors session health and timeouts
func (h *SpeechHandlers) monitorSTTSession(session *STTStreamSession) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-session.ctx.Done():
			return
			
		case <-ticker.C:
			session.mu.RLock()
			timeSinceActivity := time.Since(session.LastActivity)
			session.mu.RUnlock()
			
			// Check for inactivity timeout
			if timeSinceActivity > time.Duration(session.Config.SilenceTimeout)*time.Millisecond {
				if session.Config.AutoFinalize {
					h.finalizeTranscription(session)
				}
			}
			
			// Send periodic heartbeat
			h.sendSTTMessage(session, STTWebSocketMessage{
				Type: "heartbeat",
				Data: map[string]interface{}{
					"session_time": time.Since(session.StartTime).Seconds(),
					"buffer_size":  len(session.AudioBuffer),
				},
			})
		}
	}
}

// sendSTTMessage sends a message to the WebSocket client
func (h *SpeechHandlers) sendSTTMessage(session *STTStreamSession, msg STTWebSocketMessage) {
	session.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	if err := session.Conn.WriteJSON(msg); err != nil {
		h.logger.WithError(err).Error("Failed to send WebSocket message")
	}
}

// createTempAudioFile creates a temporary WAV file from audio data
func (h *SpeechHandlers) createTempAudioFile(audioData []byte) (string, error) {
	// Create temp file
	tempFile, err := os.CreateTemp("", "stt_stream_*.wav")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Write WAV header (assuming 16kHz, 16-bit mono)
	wavHeader := h.createWAVHeader(len(audioData))
	if _, err := tempFile.Write(wavHeader); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	// Write audio data
	if _, err := tempFile.Write(audioData); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// createWAVHeader creates a WAV file header
func (h *SpeechHandlers) createWAVHeader(dataSize int) []byte {
	header := make([]byte, 44)
	
	// RIFF header
	copy(header[0:4], "RIFF")
	// File size - 8
	fileSize := uint32(dataSize + 36)
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)
	
	// WAVE format
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	
	// fmt chunk size (16)
	header[16] = 16
	
	// Audio format (1 = PCM)
	header[20] = 1
	
	// Number of channels (1 = mono)
	header[22] = 1
	
	// Sample rate (16000)
	sampleRate := uint32(16000)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	
	// Byte rate (sample rate * channels * bits per sample / 8)
	byteRate := uint32(32000)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	
	// Block align (channels * bits per sample / 8)
	header[32] = 2
	
	// Bits per sample
	header[34] = 16
	
	// Data header
	copy(header[36:40], "data")
	
	// Data size
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)
	
	return header
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("stt_%d", time.Now().UnixNano())
}