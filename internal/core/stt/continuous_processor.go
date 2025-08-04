package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ContinuousSTTProcessor handles continuous speech-to-text processing
type ContinuousSTTProcessor struct {
	logger           *logrus.Logger
	config           *STTEngineConfig
	activeSessions   map[string]*STTSession
	sessionMutex     sync.RWMutex
	isRunning        bool
	pythonScriptPath string
	tempDir          string
}

// STTSession represents an active STT processing session
type STTSession struct {
	ID                     string
	Config                 *STTSessionConfig
	StartTime              time.Time
	AudioBuffer            []byte
	ChunksProcessed        int
	IsCompleted            bool
	CompletedTranscription *STTTranscription
	mutex                  sync.RWMutex
	processor              *ContinuousSTTProcessor
	lastActivity           time.Time
	silenceThreshold       time.Duration
}

// STTSessionConfig contains configuration for an STT session
type STTSessionConfig struct {
	Source              AudioSource   `json:"source"`
	DeviceID            string        `json:"device_id"`
	Language            string        `json:"language"`
	AutoLanguageDetect  bool          `json:"auto_language_detect"`
	ConfidenceThreshold float64       `json:"confidence_threshold"`
	WakeWordTriggered   bool          `json:"wake_word_triggered"`
	PreWakeAudio        []byte        `json:"pre_wake_audio,omitempty"`
	MaxDuration         time.Duration `json:"max_duration"`
	SilenceTimeout      time.Duration `json:"silence_timeout"`
}

// STTTranscription represents a completed transcription result
type STTTranscription struct {
	SessionID      string                 `json:"session_id"`
	Text           string                 `json:"text"`
	Confidence     float64                `json:"confidence"`
	Language       string                 `json:"language"`
	LanguageProb   float64                `json:"language_prob"`
	Timestamp      time.Time              `json:"timestamp"`
	Duration       time.Duration          `json:"duration"`
	Source         AudioSource            `json:"source"`
	DeviceID       string                 `json:"device_id"`
	ProcessingTime time.Duration          `json:"processing_time"`
	ChunkCount     int                    `json:"chunk_count"`
	Segments       []TranscriptionSegment `json:"segments"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// TranscriptionSegment represents a segment of the transcription with timing
type TranscriptionSegment struct {
	Text       string     `json:"text"`
	Start      float64    `json:"start"`
	End        float64    `json:"end"`
	Confidence float64    `json:"confidence"`
	Words      []WordInfo `json:"words,omitempty"`
}

// WordInfo contains detailed information about individual words
type WordInfo struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence"`
}

// WhisperResult represents the result from Whisper STT processing
type WhisperResult struct {
	Text           string                 `json:"text"`
	Language       string                 `json:"language"`
	LanguageProb   float64                `json:"language_probability"`
	Duration       float64                `json:"duration"`
	Segments       []TranscriptionSegment `json:"segments"`
	AvgConfidence  float64                `json:"avg_confidence"`
	ProcessingTime float64                `json:"processing_time"`
	Model          string                 `json:"model"`
	Error          string                 `json:"error,omitempty"`
}

// NewContinuousSTTProcessor creates a new continuous STT processor
func NewContinuousSTTProcessor(logger *logrus.Logger, config *STTEngineConfig) *ContinuousSTTProcessor {
	return &ContinuousSTTProcessor{
		logger:           logger,
		config:           config,
		activeSessions:   make(map[string]*STTSession),
		pythonScriptPath: "../../pma-ai/pma-speech/stt.py", // Path to existing STT script
		tempDir:          "./data/temp/stt",
	}
}

// Start initializes the continuous STT processor
func (csp *ContinuousSTTProcessor) Start(ctx context.Context) error {
	csp.logger.Info("Starting Continuous STT Processor...")

	// Ensure temp directory exists
	if err := os.MkdirAll(csp.tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Verify Python script exists
	if _, err := os.Stat(csp.pythonScriptPath); os.IsNotExist(err) {
		return fmt.Errorf("STT Python script not found at %s", csp.pythonScriptPath)
	}

	// Test STT functionality
	if err := csp.testSTTCapability(); err != nil {
		return fmt.Errorf("STT capability test failed: %w", err)
	}

	csp.isRunning = true

	// Start session cleanup routine
	go csp.sessionCleanup(ctx)

	csp.logger.Info("Continuous STT Processor started successfully")
	return nil
}

// Stop shuts down the continuous STT processor
func (csp *ContinuousSTTProcessor) Stop(ctx context.Context) error {
	csp.logger.Info("Stopping Continuous STT Processor...")

	csp.sessionMutex.Lock()
	defer csp.sessionMutex.Unlock()

	// Complete all active sessions
	for _, session := range csp.activeSessions {
		if !session.IsCompleted {
			session.ForceComplete()
		}
	}

	csp.activeSessions = make(map[string]*STTSession)
	csp.isRunning = false

	csp.logger.Info("Continuous STT Processor stopped")
	return nil
}

// StartSession creates a new STT processing session
func (csp *ContinuousSTTProcessor) StartSession(config *STTSessionConfig) *STTSession {
	sessionID := uuid.New().String()

	// Set defaults if not provided
	if config.MaxDuration == 0 {
		config.MaxDuration = 5 * time.Minute // Default 5 minute max
	}
	if config.SilenceTimeout == 0 {
		config.SilenceTimeout = 3 * time.Second // Default 3 second silence timeout
	}

	session := &STTSession{
		ID:               sessionID,
		Config:           config,
		StartTime:        time.Now(),
		AudioBuffer:      make([]byte, 0),
		ChunksProcessed:  0,
		IsCompleted:      false,
		processor:        csp,
		lastActivity:     time.Now(),
		silenceThreshold: config.SilenceTimeout,
	}

	// Include pre-wake audio if provided
	if len(config.PreWakeAudio) > 0 {
		session.AudioBuffer = append(session.AudioBuffer, config.PreWakeAudio...)
	}

	csp.sessionMutex.Lock()
	csp.activeSessions[sessionID] = session
	csp.sessionMutex.Unlock()

	csp.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"source":     config.Source,
		"device_id":  config.DeviceID,
		"language":   config.Language,
	}).Info("Started new STT session")

	return session
}

// ProcessAudio adds audio data to the session for processing
func (session *STTSession) ProcessAudio(audioData []byte) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.IsCompleted {
		return fmt.Errorf("session %s is already completed", session.ID)
	}

	// Add audio to buffer
	session.AudioBuffer = append(session.AudioBuffer, audioData...)
	session.ChunksProcessed++
	session.lastActivity = time.Now()

	// Check for voice activity (simple energy-based detection)
	if session.hasVoiceActivity(audioData) {
		session.lastActivity = time.Now()
	}

	// Process if we have enough audio or if it's been too long
	bufferDuration := session.getBufferDuration()
	timeSinceStart := time.Since(session.StartTime)

	shouldProcess := bufferDuration >= session.processor.config.ChunkDuration ||
		timeSinceStart >= session.processor.config.MaxProcessingDelay ||
		time.Since(session.lastActivity) >= session.silenceThreshold

	if shouldProcess {
		return session.processCurrentBuffer()
	}

	return nil
}

// GetCompletedTranscription returns the completed transcription if available
func (session *STTSession) GetCompletedTranscription() *STTTranscription {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	if session.IsCompleted && session.CompletedTranscription != nil {
		return session.CompletedTranscription
	}

	return nil
}

// ForceComplete forces completion of the session
func (session *STTSession) ForceComplete() *STTTranscription {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.IsCompleted {
		return session.CompletedTranscription
	}

	// Process any remaining audio
	if len(session.AudioBuffer) > 0 {
		session.processCurrentBuffer()
	}

	session.IsCompleted = true

	// Remove from active sessions
	session.processor.sessionMutex.Lock()
	delete(session.processor.activeSessions, session.ID)
	session.processor.sessionMutex.Unlock()

	return session.CompletedTranscription
}

// processCurrentBuffer processes the current audio buffer
func (session *STTSession) processCurrentBuffer() error {
	if len(session.AudioBuffer) == 0 {
		return nil
	}

	startTime := time.Now()

	// Create temporary audio file
	audioFile := filepath.Join(session.processor.tempDir, fmt.Sprintf("session_%s_%d.wav", session.ID, time.Now().UnixNano()))

	// Write audio data to file (assuming PCM format)
	if err := session.writeAudioToFile(audioFile, session.AudioBuffer); err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	defer os.Remove(audioFile) // Clean up temp file

	// Process with Whisper STT
	result, err := session.processor.processWithWhisper(audioFile, session.Config)
	if err != nil {
		return fmt.Errorf("STT processing failed: %w", err)
	}

	processingTime := time.Since(startTime)

	// Create transcription result
	transcription := &STTTranscription{
		SessionID:      session.ID,
		Text:           result.Text,
		Confidence:     result.AvgConfidence,
		Language:       result.Language,
		LanguageProb:   result.LanguageProb,
		Timestamp:      session.StartTime,
		Duration:       time.Duration(result.Duration * float64(time.Second)),
		Source:         session.Config.Source,
		DeviceID:       session.Config.DeviceID,
		ProcessingTime: processingTime,
		ChunkCount:     session.ChunksProcessed,
		Segments:       result.Segments,
		Metadata: map[string]interface{}{
			"model":               result.Model,
			"wake_word_triggered": session.Config.WakeWordTriggered,
			"buffer_size":         len(session.AudioBuffer),
		},
	}

	// Check confidence threshold
	if transcription.Confidence >= session.Config.ConfidenceThreshold {
		session.CompletedTranscription = transcription
		session.IsCompleted = true

		session.processor.logger.WithFields(logrus.Fields{
			"session_id":      session.ID,
			"text":            transcription.Text,
			"confidence":      transcription.Confidence,
			"processing_time": processingTime,
		}).Info("STT transcription completed")
	} else {
		session.processor.logger.WithFields(logrus.Fields{
			"session_id": session.ID,
			"confidence": transcription.Confidence,
			"threshold":  session.Config.ConfidenceThreshold,
		}).Debug("Transcription below confidence threshold")
	}

	// Clear buffer after processing
	session.AudioBuffer = make([]byte, 0)

	return nil
}

// processWithWhisper processes audio using the Whisper STT engine
func (csp *ContinuousSTTProcessor) processWithWhisper(audioFile string, config *STTSessionConfig) (*WhisperResult, error) {
	// Prepare command arguments
	args := []string{
		csp.pythonScriptPath,
		"--input", audioFile,
		"--model", csp.config.Model,
		"--output-format", "json",
	}

	if config.Language != "" && !config.AutoLanguageDetect {
		args = append(args, "--language", config.Language)
	}

	if csp.config.NoiseReduction {
		args = append(args, "--noise-reduction")
	}

	// Execute STT processing
	cmd := exec.Command("python3", args...)
	cmd.Env = os.Environ()

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("whisper processing failed: %w", err)
	}

	// Parse result
	var result WhisperResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse whisper result: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("whisper error: %s", result.Error)
	}

	return &result, nil
}

// Helper methods
func (session *STTSession) hasVoiceActivity(audioData []byte) bool {
	if len(audioData) < 2 {
		return false
	}

	// Simple energy-based voice activity detection
	var sum float64
	for i := 0; i < len(audioData)-1; i += 2 {
		sample := int16(audioData[i]) | (int16(audioData[i+1]) << 8)
		sum += float64(sample * sample)
	}

	energy := sum / float64(len(audioData)/2) / (32768.0 * 32768.0)
	return energy > 0.001 // Simple threshold
}

func (session *STTSession) getBufferDuration() time.Duration {
	// Assuming 16kHz, 16-bit, mono audio
	sampleRate := 16000
	bytesPerSample := 2
	samplesPerSecond := sampleRate * bytesPerSample

	seconds := float64(len(session.AudioBuffer)) / float64(samplesPerSecond)
	return time.Duration(seconds * float64(time.Second))
}

func (session *STTSession) writeAudioToFile(filename string, audioData []byte) error {
	// This is a simplified WAV file writer
	// In a production system, you'd use a proper audio library
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write basic WAV header (44 bytes)
	sampleRate := 16000
	channels := 1
	bitsPerSample := 16
	dataSize := len(audioData)
	fileSize := 36 + dataSize

	// WAV header
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	header[4] = byte(fileSize & 0xff)
	header[5] = byte((fileSize >> 8) & 0xff)
	header[6] = byte((fileSize >> 16) & 0xff)
	header[7] = byte((fileSize >> 24) & 0xff)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	header[16] = 16 // PCM format chunk size
	header[20] = 1  // PCM format
	header[22] = byte(channels)
	header[24] = byte(sampleRate & 0xff)
	header[25] = byte((sampleRate >> 8) & 0xff)
	header[26] = byte((sampleRate >> 16) & 0xff)
	header[27] = byte((sampleRate >> 24) & 0xff)

	byteRate := sampleRate * channels * bitsPerSample / 8
	header[28] = byte(byteRate & 0xff)
	header[29] = byte((byteRate >> 8) & 0xff)
	header[30] = byte((byteRate >> 16) & 0xff)
	header[31] = byte((byteRate >> 24) & 0xff)

	blockAlign := channels * bitsPerSample / 8
	header[32] = byte(blockAlign)
	header[34] = byte(bitsPerSample)

	copy(header[36:40], "data")
	header[40] = byte(dataSize & 0xff)
	header[41] = byte((dataSize >> 8) & 0xff)
	header[42] = byte((dataSize >> 16) & 0xff)
	header[43] = byte((dataSize >> 24) & 0xff)

	// Write header and data
	if _, err := file.Write(header); err != nil {
		return err
	}

	if _, err := file.Write(audioData); err != nil {
		return err
	}

	return nil
}

// testSTTCapability tests if STT processing is working
func (csp *ContinuousSTTProcessor) testSTTCapability() error {
	// Create a small test audio file (silence)
	testFile := filepath.Join(csp.tempDir, "test_stt.wav")
	testAudio := make([]byte, 16000*2) // 1 second of silence

	session := &STTSession{AudioBuffer: testAudio}
	if err := session.writeAudioToFile(testFile, testAudio); err != nil {
		return fmt.Errorf("failed to create test audio file: %w", err)
	}
	defer os.Remove(testFile)

	// Test STT processing
	config := &STTSessionConfig{
		Language:            csp.config.Language,
		AutoLanguageDetect:  csp.config.AutoLanguageDetect,
		ConfidenceThreshold: 0.0, // Low threshold for test
	}

	_, err := csp.processWithWhisper(testFile, config)
	if err != nil {
		return fmt.Errorf("STT test failed: %w", err)
	}

	csp.logger.Info("STT capability test passed")
	return nil
}

// sessionCleanup periodically cleans up old or stale sessions
func (csp *ContinuousSTTProcessor) sessionCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			csp.sessionMutex.Lock()

			now := time.Now()
			staleSessions := make([]string, 0)

			for sessionID, session := range csp.activeSessions {
				session.mutex.RLock()
				isStale := now.Sub(session.StartTime) > session.Config.MaxDuration ||
					(!session.IsCompleted && now.Sub(session.lastActivity) > session.silenceThreshold*2)
				session.mutex.RUnlock()

				if isStale {
					staleSessions = append(staleSessions, sessionID)
				}
			}

			// Clean up stale sessions
			for _, sessionID := range staleSessions {
				if session, exists := csp.activeSessions[sessionID]; exists {
					session.ForceComplete()
					delete(csp.activeSessions, sessionID)
					csp.logger.WithField("session_id", sessionID).Debug("Cleaned up stale STT session")
				}
			}

			csp.sessionMutex.Unlock()
		}
	}
}
