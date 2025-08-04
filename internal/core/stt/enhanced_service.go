package stt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/sirupsen/logrus"
)

// EnhancedSTTService provides 24/7 speech-to-text capabilities with wake word detection
type EnhancedSTTService struct {
	logger         *logrus.Logger
	config         *STTConfig
	audioManager   *AudioManager
	wakeWordEngine *WakeWordEngine
	continuousSTT  *ContinuousSTTProcessor
	wsHub          *websocket.Hub
	isRunning      bool
	mu             sync.RWMutex

	// Event channels
	transcriptionEvents chan TranscriptionEvent
	wakeWordEvents      chan WakeWordEvent
	audioEvents         chan AudioDeviceEvent
	systemEvents        chan SystemEvent
}

// STTConfig contains configuration for the enhanced STT service
type STTConfig struct {
	Enabled             bool                   `json:"enabled"`
	ContinuousListening bool                   `json:"continuous_listening"`
	WakeWordConfig      *WakeWordConfig        `json:"wake_word_config"`
	AudioConfig         *AudioConfig           `json:"audio_config"`
	STTEngineConfig     *STTEngineConfig       `json:"stt_engine_config"`
	PowerManagement     *PowerManagementConfig `json:"power_management"`
	PrivacyConfig       *PrivacyConfig         `json:"privacy_config"`
	NotificationConfig  *NotificationConfig    `json:"notification_config"`
}

// STTEngineConfig contains configuration for the STT processing engine
type STTEngineConfig struct {
	Model               string        `json:"model"`                // whisper model to use
	Language            string        `json:"language"`             // primary language
	AutoLanguageDetect  bool          `json:"auto_language_detect"` // detect language automatically
	ChunkDuration       time.Duration `json:"chunk_duration"`       // duration of audio chunks for processing
	OverlapDuration     time.Duration `json:"overlap_duration"`     // overlap between chunks
	MaxProcessingDelay  time.Duration `json:"max_processing_delay"` // max delay before processing
	ConfidenceThreshold float64       `json:"confidence_threshold"` // minimum confidence for transcription
	NoiseReduction      bool          `json:"noise_reduction"`      // enable noise reduction
	AutoCorrect         bool          `json:"auto_correct"`         // enable auto-correction
	RealTimeMode        bool          `json:"real_time_mode"`       // process audio in real-time
}

// PowerManagementConfig contains power saving settings
type PowerManagementConfig struct {
	Enabled           bool          `json:"enabled"`
	BatteryThreshold  float64       `json:"battery_threshold"`   // battery level to enter power saving
	IdleTimeout       time.Duration `json:"idle_timeout"`        // timeout before entering idle mode
	SleepModeEnabled  bool          `json:"sleep_mode_enabled"`  // enable sleep mode during quiet hours
	SleepStartHour    int           `json:"sleep_start_hour"`    // hour to start sleep mode (24h format)
	SleepEndHour      int           `json:"sleep_end_hour"`      // hour to end sleep mode
	ReducedProcessing bool          `json:"reduced_processing"`  // reduce processing frequency in power saving
	WakeWordOnlyMode  bool          `json:"wake_word_only_mode"` // only wake word detection during power saving
}

// PrivacyConfig contains privacy and security settings
type PrivacyConfig struct {
	LocalProcessingOnly  bool          `json:"local_processing_only"`  // never send audio to external services
	AutoDeleteRecordings bool          `json:"auto_delete_recordings"` // automatically delete recordings
	RetentionPeriod      time.Duration `json:"retention_period"`       // how long to keep recordings
	EncryptionEnabled    bool          `json:"encryption_enabled"`     // encrypt stored audio
	AnonymizeTranscripts bool          `json:"anonymize_transcripts"`  // remove personal information from transcripts
	ConsentRequired      bool          `json:"consent_required"`       // require explicit consent for recording
}

// NotificationConfig contains settings for STT event notifications
type NotificationConfig struct {
	WakeWordDetected    bool `json:"wake_word_detected"`   // notify on wake word detection
	TranscriptionReady  bool `json:"transcription_ready"`  // notify when transcription is complete
	ErrorOccurred       bool `json:"error_occurred"`       // notify on errors
	DeviceConnected     bool `json:"device_connected"`     // notify on audio device changes
	ConfidenceThreshold bool `json:"confidence_threshold"` // notify on low confidence transcriptions
}

// TranscriptionEvent represents a completed transcription
type TranscriptionEvent struct {
	ID                string                 `json:"id"`
	Text              string                 `json:"text"`
	Confidence        float64                `json:"confidence"`
	Language          string                 `json:"language"`
	Timestamp         time.Time              `json:"timestamp"`
	Duration          time.Duration          `json:"duration"`
	Source            AudioSource            `json:"source"`
	DeviceID          string                 `json:"device_id"`
	WakeWordTriggered bool                   `json:"wake_word_triggered"`
	PreContext        string                 `json:"pre_context"`  // context before wake word
	PostContext       string                 `json:"post_context"` // context after wake word
	ProcessingTime    time.Duration          `json:"processing_time"`
	ChunkCount        int                    `json:"chunk_count"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// AudioDeviceEvent represents audio device status changes
type AudioDeviceEvent struct {
	Type       string                 `json:"type"` // "connected", "disconnected", "quality_change", "primary_changed"
	DeviceID   string                 `json:"device_id"`
	DeviceName string                 `json:"device_name"`
	Source     AudioSource            `json:"source"`
	Quality    float64                `json:"quality"`
	Timestamp  time.Time              `json:"timestamp"`
	Details    map[string]interface{} `json:"details"`
}

// SystemEvent represents system-level STT events
type SystemEvent struct {
	Type      string                 `json:"type"` // "started", "stopped", "error", "config_changed"
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  string                 `json:"severity"` // "info", "warning", "error"
	Details   map[string]interface{} `json:"details"`
}

// NewEnhancedSTTService creates a new enhanced STT service
func NewEnhancedSTTService(logger *logrus.Logger, config *STTConfig, wsHub *websocket.Hub) *EnhancedSTTService {
	return &EnhancedSTTService{
		logger:              logger,
		config:              config,
		wsHub:               wsHub,
		transcriptionEvents: make(chan TranscriptionEvent, 100),
		wakeWordEvents:      make(chan WakeWordEvent, 100),
		audioEvents:         make(chan AudioDeviceEvent, 100),
		systemEvents:        make(chan SystemEvent, 100),
	}
}

// Start begins the enhanced STT service
func (ess *EnhancedSTTService) Start(ctx context.Context) error {
	ess.mu.Lock()
	defer ess.mu.Unlock()

	if ess.isRunning {
		return fmt.Errorf("STT service is already running")
	}

	ess.logger.Info("Starting Enhanced STT Service...")

	// Initialize audio manager
	ess.audioManager = NewAudioManager(ess.logger, ess.config.AudioConfig)
	if err := ess.audioManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start audio manager: %w", err)
	}

	// Initialize wake word engine if enabled
	if ess.config.WakeWordConfig.Enabled {
		audioStream := ess.audioManager.GetAudioStream()
		ess.wakeWordEngine = NewWakeWordEngine(ess.logger, ess.config.WakeWordConfig, audioStream)
		if err := ess.wakeWordEngine.Start(ctx); err != nil {
			return fmt.Errorf("failed to start wake word engine: %w", err)
		}
	}

	// Initialize continuous STT processor
	ess.continuousSTT = NewContinuousSTTProcessor(ess.logger, ess.config.STTEngineConfig)
	if err := ess.continuousSTT.Start(ctx); err != nil {
		return fmt.Errorf("failed to start continuous STT processor: %w", err)
	}

	// Start event processing
	go ess.processEvents(ctx)

	// Start audio processing pipeline
	go ess.processAudioPipeline(ctx)

	// Start WebSocket broadcasting
	go ess.broadcastEvents(ctx)

	// Start periodic maintenance
	go ess.periodicMaintenance(ctx)

	ess.isRunning = true

	// Send system started event
	ess.systemEvents <- SystemEvent{
		Type:      "started",
		Message:   "Enhanced STT Service started successfully",
		Timestamp: time.Now(),
		Severity:  "info",
		Details: map[string]interface{}{
			"wake_word_enabled":    ess.config.WakeWordConfig.Enabled,
			"continuous_listening": ess.config.ContinuousListening,
			"power_management":     ess.config.PowerManagement.Enabled,
		},
	}

	ess.logger.Info("Enhanced STT Service started successfully")
	return nil
}

// Stop stops the enhanced STT service
func (ess *EnhancedSTTService) Stop(ctx context.Context) error {
	ess.mu.Lock()
	defer ess.mu.Unlock()

	if !ess.isRunning {
		return nil
	}

	ess.logger.Info("Stopping Enhanced STT Service...")

	// Stop components
	if ess.continuousSTT != nil {
		ess.continuousSTT.Stop(ctx)
	}

	if ess.wakeWordEngine != nil {
		// Wake word engine cleanup is handled by context cancellation
	}

	if ess.audioManager != nil {
		// Audio manager cleanup is handled by context cancellation
	}

	ess.isRunning = false

	// Send system stopped event
	ess.systemEvents <- SystemEvent{
		Type:      "stopped",
		Message:   "Enhanced STT Service stopped",
		Timestamp: time.Now(),
		Severity:  "info",
	}

	ess.logger.Info("Enhanced STT Service stopped")
	return nil
}

// processAudioPipeline handles the main audio processing pipeline
func (ess *EnhancedSTTService) processAudioPipeline(ctx context.Context) {
	audioStream := ess.audioManager.GetAudioStream()
	var wakeWordEvents <-chan WakeWordEvent

	if ess.wakeWordEngine != nil {
		wakeWordEvents = ess.wakeWordEngine.GetDetectionEvents()
	}

	isListening := ess.config.ContinuousListening
	var currentSession *STTSession

	for {
		select {
		case <-ctx.Done():
			return

		case audioChunk := <-audioStream:
			// Handle continuous listening mode
			if isListening {
				if currentSession == nil {
					currentSession = ess.continuousSTT.StartSession(&STTSessionConfig{
						Source:              audioChunk.Source,
						DeviceID:            audioChunk.DeviceID,
						Language:            ess.config.STTEngineConfig.Language,
						AutoLanguageDetect:  ess.config.STTEngineConfig.AutoLanguageDetect,
						ConfidenceThreshold: ess.config.STTEngineConfig.ConfidenceThreshold,
					})
				}

				// Process audio chunk
				if err := currentSession.ProcessAudio(audioChunk.Data); err != nil {
					ess.logger.WithError(err).Error("Error processing audio chunk")
				}

				// Check for completed transcriptions
				if transcription := currentSession.GetCompletedTranscription(); transcription != nil {
					ess.handleTranscription(transcription, false)
					currentSession = nil // Start new session for next utterance
				}
			}

		case wakeWordEvent := <-wakeWordEvents:
			ess.logger.WithFields(logrus.Fields{
				"keyword":    wakeWordEvent.Keyword,
				"confidence": wakeWordEvent.Confidence,
				"source":     wakeWordEvent.Source,
			}).Info("Wake word detected")

			// Handle wake word detection
			ess.handleWakeWordDetection(wakeWordEvent)

			// Start listening session if not already listening
			if !isListening {
				isListening = true

				// Start STT session with wake word context
				currentSession = ess.continuousSTT.StartSession(&STTSessionConfig{
					Source:              wakeWordEvent.Source,
					DeviceID:            wakeWordEvent.DeviceID,
					Language:            ess.config.STTEngineConfig.Language,
					AutoLanguageDetect:  ess.config.STTEngineConfig.AutoLanguageDetect,
					ConfidenceThreshold: ess.config.STTEngineConfig.ConfidenceThreshold,
					WakeWordTriggered:   true,
					PreWakeAudio:        wakeWordEvent.PreWakeAudio,
				})

				// Process the post-wake audio
				if len(wakeWordEvent.PostWakeAudio) > 0 {
					if err := currentSession.ProcessAudio(wakeWordEvent.PostWakeAudio); err != nil {
						ess.logger.WithError(err).Error("Error processing post-wake audio")
					}
				}

				// Set timeout to stop listening after a period of inactivity
				go func() {
					timer := time.NewTimer(30 * time.Second) // 30 second timeout
					defer timer.Stop()

					select {
					case <-timer.C:
						isListening = ess.config.ContinuousListening // Reset to config default
						if currentSession != nil {
							if transcription := currentSession.ForceComplete(); transcription != nil {
								ess.handleTranscription(transcription, true)
							}
							currentSession = nil
						}
					case <-ctx.Done():
						return
					}
				}()
			}
		}
	}
}

// handleWakeWordDetection processes a wake word detection event
func (ess *EnhancedSTTService) handleWakeWordDetection(event WakeWordEvent) {
	// Forward wake word event
	select {
	case ess.wakeWordEvents <- event:
	default:
		ess.logger.Warn("Wake word event buffer full")
	}

	// Handle wake word actions
	switch event.Action {
	case "start_listening":
		ess.logger.Info("Starting listening session due to wake word")
	case "stop_listening":
		ess.logger.Info("Stopping listening session due to wake word")
	case "command_mode":
		ess.logger.Info("Entering command mode due to wake word")
	}
}

// handleTranscription processes a completed transcription
func (ess *EnhancedSTTService) handleTranscription(transcription *STTTranscription, wakeWordTriggered bool) {
	if transcription.Confidence < ess.config.STTEngineConfig.ConfidenceThreshold {
		ess.logger.WithFields(logrus.Fields{
			"confidence": transcription.Confidence,
			"threshold":  ess.config.STTEngineConfig.ConfidenceThreshold,
		}).Debug("Transcription below confidence threshold")
		return
	}

	// Create transcription event
	event := TranscriptionEvent{
		ID:                fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		Text:              transcription.Text,
		Confidence:        transcription.Confidence,
		Language:          transcription.Language,
		Timestamp:         transcription.Timestamp,
		Duration:          transcription.Duration,
		Source:            transcription.Source,
		DeviceID:          transcription.DeviceID,
		WakeWordTriggered: wakeWordTriggered,
		ProcessingTime:    transcription.ProcessingTime,
		ChunkCount:        transcription.ChunkCount,
		Metadata:          transcription.Metadata,
	}

	// Forward transcription event
	select {
	case ess.transcriptionEvents <- event:
	default:
		ess.logger.Warn("Transcription event buffer full")
	}

	ess.logger.WithFields(logrus.Fields{
		"text":       transcription.Text,
		"confidence": transcription.Confidence,
		"source":     transcription.Source,
	}).Info("Transcription completed")
}

// processEvents handles all event processing and forwarding
func (ess *EnhancedSTTService) processEvents(ctx context.Context) {
	deviceEvents := ess.audioManager.GetDeviceEvents()

	for {
		select {
		case <-ctx.Done():
			return

		case deviceEvent := <-deviceEvents:
			// Convert device event to audio device event
			audioEvent := AudioDeviceEvent{
				Type:       deviceEvent.Type,
				DeviceID:   deviceEvent.Device.ID,
				DeviceName: deviceEvent.Device.Name,
				Source:     deviceEvent.Device.Source,
				Quality:    deviceEvent.Device.Quality,
				Timestamp:  deviceEvent.Timestamp,
				Details: map[string]interface{}{
					"capabilities": deviceEvent.Device.Capabilities,
					"format":       deviceEvent.Device.Format,
				},
			}

			// Forward audio device event
			select {
			case ess.audioEvents <- audioEvent:
			default:
				ess.logger.Warn("Audio device event buffer full")
			}
		}
	}
}

// broadcastEvents broadcasts events via WebSocket
func (ess *EnhancedSTTService) broadcastEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case transcriptionEvent := <-ess.transcriptionEvents:
			if ess.config.NotificationConfig.TranscriptionReady {
				ess.wsHub.BroadcastToTopic("stt:transcription", "stt_transcription", transcriptionEvent)
			}

		case wakeWordEvent := <-ess.wakeWordEvents:
			if ess.config.NotificationConfig.WakeWordDetected {
				ess.wsHub.BroadcastToTopic("stt:wake_word", "stt_wake_word", wakeWordEvent)
			}

		case audioEvent := <-ess.audioEvents:
			if ess.config.NotificationConfig.DeviceConnected {
				ess.wsHub.BroadcastToTopic("stt:audio_device", "stt_audio_device", audioEvent)
			}

		case systemEvent := <-ess.systemEvents:
			ess.wsHub.BroadcastToTopic("stt:system", "stt_system", systemEvent)
		}
	}
}

// periodicMaintenance performs periodic maintenance tasks
func (ess *EnhancedSTTService) periodicMaintenance(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check power management
			if ess.config.PowerManagement.Enabled {
				ess.checkPowerSaving()
			}

			// Clean up old recordings if privacy settings require it
			if ess.config.PrivacyConfig.AutoDeleteRecordings {
				ess.cleanupOldRecordings()
			}

			// Update statistics
			ess.updateStatistics()
		}
	}
}

// API methods for external control
func (ess *EnhancedSTTService) GetStatus() map[string]interface{} {
	ess.mu.RLock()
	defer ess.mu.RUnlock()

	status := map[string]interface{}{
		"is_running":           ess.isRunning,
		"continuous_listening": ess.config.ContinuousListening,
		"wake_word_enabled":    ess.config.WakeWordConfig.Enabled,
		"active_devices":       len(ess.audioManager.GetActiveDevices()),
		"timestamp":            time.Now(),
	}

	if ess.wakeWordEngine != nil {
		status["wake_word_stats"] = ess.wakeWordEngine.GetStats()
	}

	return status
}

func (ess *EnhancedSTTService) SetContinuousListening(enabled bool) error {
	ess.mu.Lock()
	defer ess.mu.Unlock()

	ess.config.ContinuousListening = enabled

	ess.systemEvents <- SystemEvent{
		Type:      "config_changed",
		Message:   fmt.Sprintf("Continuous listening %s", map[bool]string{true: "enabled", false: "disabled"}[enabled]),
		Timestamp: time.Now(),
		Severity:  "info",
		Details: map[string]interface{}{
			"continuous_listening": enabled,
		},
	}

	return nil
}

// Helper methods
func (ess *EnhancedSTTService) checkPowerSaving() {
	// Implementation for power management checks
}

func (ess *EnhancedSTTService) cleanupOldRecordings() {
	// Implementation for cleaning up old recordings based on retention policy
}

func (ess *EnhancedSTTService) updateStatistics() {
	// Implementation for updating service statistics
}
