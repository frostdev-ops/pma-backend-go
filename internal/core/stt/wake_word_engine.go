package stt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// WakeWordEngine handles continuous wake word detection
type WakeWordEngine struct {
	logger          *logrus.Logger
	config          *WakeWordConfig
	detector        WakeWordDetector
	circularBuffer  *CircularAudioBuffer
	audioInput      <-chan AudioChunk
	detectionOutput chan WakeWordEvent
	mu              sync.RWMutex
	isActive        bool
	stats           *WakeWordStats
}

// WakeWordConfig contains configuration for wake word detection
type WakeWordConfig struct {
	Enabled            bool              `json:"enabled"`
	Keywords           []WakeWordKeyword `json:"keywords"`
	SensitivityLevel   float64           `json:"sensitivity_level"`  // 0.0 - 1.0
	BufferDuration     time.Duration     `json:"buffer_duration"`    // How long to keep audio before wake word
	PostWakeDuration   time.Duration     `json:"post_wake_duration"` // How long to capture after wake word
	MinConfidence      float64           `json:"min_confidence"`     // Minimum confidence threshold
	CooldownPeriod     time.Duration     `json:"cooldown_period"`    // Time between detections
	PowerSavingMode    bool              `json:"power_saving_mode"`
	AdaptiveThreshold  bool              `json:"adaptive_threshold"`
	NoiseGateThreshold float64           `json:"noise_gate_threshold"`
	ModelPath          string            `json:"model_path"`
}

// WakeWordKeyword represents a configured wake word
type WakeWordKeyword struct {
	Word        string  `json:"word"`
	Enabled     bool    `json:"enabled"`
	Sensitivity float64 `json:"sensitivity"`
	Action      string  `json:"action"`  // "start_listening", "stop_listening", "command_mode"
	Context     string  `json:"context"` // Additional context for processing
}

// WakeWordEvent represents a detected wake word event
type WakeWordEvent struct {
	Keyword       string        `json:"keyword"`
	Confidence    float64       `json:"confidence"`
	Timestamp     time.Time     `json:"timestamp"`
	AudioContext  []byte        `json:"audio_context"` // Audio before and after detection
	AudioDuration time.Duration `json:"audio_duration"`
	Source        AudioSource   `json:"source"`
	DeviceID      string        `json:"device_id"`
	Action        string        `json:"action"`
	PreWakeAudio  []byte        `json:"pre_wake_audio"`  // Audio before wake word
	PostWakeAudio []byte        `json:"post_wake_audio"` // Audio after wake word
}

// WakeWordStats tracks detection statistics
type WakeWordStats struct {
	TotalDetections     int64            `json:"total_detections"`
	FalsePositives      int64            `json:"false_positives"`
	LastDetection       time.Time        `json:"last_detection"`
	AverageConfidence   float64          `json:"average_confidence"`
	DetectionsByKeyword map[string]int64 `json:"detections_by_keyword"`
	DetectionsByHour    map[int]int64    `json:"detections_by_hour"`
	PowerSavingActive   bool             `json:"power_saving_active"`
	ProcessingLatency   time.Duration    `json:"processing_latency"`
}

// WakeWordDetector interface for different wake word detection implementations
type WakeWordDetector interface {
	Initialize(config *WakeWordConfig) error
	DetectWakeWord(audioData []byte) (*WakeWordDetection, error)
	UpdateSensitivity(sensitivity float64) error
	AddKeyword(keyword WakeWordKeyword) error
	RemoveKeyword(word string) error
	GetSupportedKeywords() []string
	Cleanup() error
}

// WakeWordDetection represents the result of wake word detection
type WakeWordDetection struct {
	Detected   bool          `json:"detected"`
	Keyword    string        `json:"keyword"`
	Confidence float64       `json:"confidence"`
	StartTime  time.Duration `json:"start_time"`
	EndTime    time.Duration `json:"end_time"`
}

// CircularAudioBuffer maintains a rolling window of audio data
type CircularAudioBuffer struct {
	buffer     []byte
	size       int
	writePos   int
	isFull     bool
	mu         sync.RWMutex
	duration   time.Duration
	sampleRate int
}

// NewWakeWordEngine creates a new wake word detection engine
func NewWakeWordEngine(logger *logrus.Logger, config *WakeWordConfig, audioInput <-chan AudioChunk) *WakeWordEngine {
	bufferSize := int(config.BufferDuration.Seconds() * float64(16000) * 2) // 16kHz, 16-bit

	return &WakeWordEngine{
		logger:          logger,
		config:          config,
		circularBuffer:  NewCircularAudioBuffer(bufferSize, config.BufferDuration, 16000),
		audioInput:      audioInput,
		detectionOutput: make(chan WakeWordEvent, 100),
		stats: &WakeWordStats{
			DetectionsByKeyword: make(map[string]int64),
			DetectionsByHour:    make(map[int]int64),
		},
	}
}

// Start begins wake word detection
func (wwe *WakeWordEngine) Start(ctx context.Context) error {
	if !wwe.config.Enabled {
		wwe.logger.Info("Wake word detection is disabled")
		return nil
	}

	wwe.mu.Lock()
	wwe.isActive = true
	wwe.mu.Unlock()

	// Initialize the wake word detector
	var err error
	wwe.detector, err = NewPorcupineDetector(wwe.logger, wwe.config)
	if err != nil {
		// Fallback to OpenWakeWord or simple keyword matching
		wwe.detector, err = NewOpenWakeWordDetector(wwe.logger, wwe.config)
		if err != nil {
			wwe.detector = NewSimpleKeywordDetector(wwe.logger, wwe.config)
		}
	}

	if err := wwe.detector.Initialize(wwe.config); err != nil {
		return fmt.Errorf("failed to initialize wake word detector: %w", err)
	}

	wwe.logger.Info("Wake word engine started")

	// Start audio processing loop
	go wwe.processAudio(ctx)

	// Start periodic maintenance
	go wwe.periodicMaintenance(ctx)

	return nil
}

// processAudio continuously processes incoming audio for wake word detection
func (wwe *WakeWordEngine) processAudio(ctx context.Context) {
	defer wwe.detector.Cleanup()

	lastDetection := time.Time{}
	processingBuffer := make([]byte, 0, 4096) // 4KB processing chunks

	for {
		select {
		case <-ctx.Done():
			return
		case audioChunk := <-wwe.audioInput:
			// Add audio to circular buffer
			wwe.circularBuffer.Write(audioChunk.Data)

			// Add to processing buffer
			processingBuffer = append(processingBuffer, audioChunk.Data...)

			// Process in chunks suitable for the detector
			for len(processingBuffer) >= 1024 { // Process 1KB chunks
				chunkToProcess := processingBuffer[:1024]
				processingBuffer = processingBuffer[1024:]

				// Check cooldown period
				if time.Since(lastDetection) < wwe.config.CooldownPeriod {
					continue
				}

				// Apply noise gate
				if wwe.calculateVolumeLevel(chunkToProcess) < wwe.config.NoiseGateThreshold {
					continue
				}

				// Power saving mode - reduce processing frequency
				if wwe.config.PowerSavingMode && wwe.shouldSkipProcessing() {
					continue
				}

				// Perform wake word detection
				startTime := time.Now()
				detection, err := wwe.detector.DetectWakeWord(chunkToProcess)
				processingLatency := time.Since(startTime)

				if err != nil {
					wwe.logger.WithError(err).Error("Wake word detection error")
					continue
				}

				wwe.stats.ProcessingLatency = processingLatency

				if detection.Detected && detection.Confidence >= wwe.config.MinConfidence {
					// Wake word detected!
					wwe.handleWakeWordDetection(detection, audioChunk, lastDetection)
					lastDetection = time.Now()
				}
			}
		}
	}
}

// handleWakeWordDetection processes a detected wake word
func (wwe *WakeWordEngine) handleWakeWordDetection(detection *WakeWordDetection, sourceChunk AudioChunk, lastDetection time.Time) {
	wwe.logger.WithFields(logrus.Fields{
		"keyword":    detection.Keyword,
		"confidence": detection.Confidence,
		"source":     sourceChunk.Source,
		"device_id":  sourceChunk.DeviceID,
	}).Info("Wake word detected")

	// Get audio context from circular buffer
	preWakeAudio := wwe.circularBuffer.ReadAll()

	// Find the keyword configuration
	var keywordConfig *WakeWordKeyword
	for _, kw := range wwe.config.Keywords {
		if kw.Word == detection.Keyword && kw.Enabled {
			keywordConfig = &kw
			break
		}
	}

	if keywordConfig == nil {
		wwe.logger.WithField("keyword", detection.Keyword).Warn("Detected keyword not in configuration")
		return
	}

	// Start capturing post-wake audio
	postWakeAudio := make([]byte, 0)
	go func() {
		// Capture audio for the post-wake duration
		captureStart := time.Now()
		for time.Since(captureStart) < wwe.config.PostWakeDuration {
			select {
			case chunk := <-wwe.audioInput:
				postWakeAudio = append(postWakeAudio, chunk.Data...)
			case <-time.After(100 * time.Millisecond):
				// Timeout to prevent blocking
			}
		}
	}()

	// Create wake word event
	event := WakeWordEvent{
		Keyword:       detection.Keyword,
		Confidence:    detection.Confidence,
		Timestamp:     time.Now(),
		AudioContext:  append(preWakeAudio, sourceChunk.Data...),
		AudioDuration: wwe.config.BufferDuration + sourceChunk.Duration,
		Source:        sourceChunk.Source,
		DeviceID:      sourceChunk.DeviceID,
		Action:        keywordConfig.Action,
		PreWakeAudio:  preWakeAudio,
		PostWakeAudio: postWakeAudio,
	}

	// Update statistics
	wwe.updateStats(detection.Keyword, detection.Confidence)

	// Send event
	select {
	case wwe.detectionOutput <- event:
	default:
		wwe.logger.Warn("Wake word event buffer full, dropping event")
	}
}

// GetDetectionEvents returns the channel for wake word detection events
func (wwe *WakeWordEngine) GetDetectionEvents() <-chan WakeWordEvent {
	return wwe.detectionOutput
}

// UpdateConfig updates the wake word configuration
func (wwe *WakeWordEngine) UpdateConfig(config *WakeWordConfig) error {
	wwe.mu.Lock()
	defer wwe.mu.Unlock()

	wwe.config = config

	if wwe.detector != nil {
		return wwe.detector.UpdateSensitivity(config.SensitivityLevel)
	}

	return nil
}

// AddKeyword adds a new wake word keyword
func (wwe *WakeWordEngine) AddKeyword(keyword WakeWordKeyword) error {
	wwe.mu.Lock()
	defer wwe.mu.Unlock()

	// Add to configuration
	wwe.config.Keywords = append(wwe.config.Keywords, keyword)

	// Add to detector
	if wwe.detector != nil {
		return wwe.detector.AddKeyword(keyword)
	}

	return nil
}

// RemoveKeyword removes a wake word keyword
func (wwe *WakeWordEngine) RemoveKeyword(word string) error {
	wwe.mu.Lock()
	defer wwe.mu.Unlock()

	// Remove from configuration
	for i, kw := range wwe.config.Keywords {
		if kw.Word == word {
			wwe.config.Keywords = append(wwe.config.Keywords[:i], wwe.config.Keywords[i+1:]...)
			break
		}
	}

	// Remove from detector
	if wwe.detector != nil {
		return wwe.detector.RemoveKeyword(word)
	}

	return nil
}

// GetStats returns current wake word detection statistics
func (wwe *WakeWordEngine) GetStats() *WakeWordStats {
	wwe.mu.RLock()
	defer wwe.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := *wwe.stats
	stats.DetectionsByKeyword = make(map[string]int64)
	stats.DetectionsByHour = make(map[int]int64)

	for k, v := range wwe.stats.DetectionsByKeyword {
		stats.DetectionsByKeyword[k] = v
	}

	for k, v := range wwe.stats.DetectionsByHour {
		stats.DetectionsByHour[k] = v
	}

	return &stats
}

// Helper methods
func (wwe *WakeWordEngine) shouldSkipProcessing() bool {
	// In power saving mode, process every other chunk during quiet periods
	return wwe.stats.TotalDetections%2 == 0 &&
		time.Since(wwe.stats.LastDetection) > 10*time.Minute
}

func (wwe *WakeWordEngine) calculateVolumeLevel(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var sum float64
	for i := 0; i < len(data)-1; i += 2 {
		sample := int16(data[i]) | (int16(data[i+1]) << 8)
		sum += float64(sample * sample)
	}

	rms := sum / float64(len(data)/2)
	return rms / (32768.0 * 32768.0)
}

func (wwe *WakeWordEngine) updateStats(keyword string, confidence float64) {
	wwe.stats.TotalDetections++
	wwe.stats.LastDetection = time.Now()
	wwe.stats.DetectionsByKeyword[keyword]++

	hour := time.Now().Hour()
	wwe.stats.DetectionsByHour[hour]++

	// Update average confidence
	wwe.stats.AverageConfidence = (wwe.stats.AverageConfidence*float64(wwe.stats.TotalDetections-1) + confidence) / float64(wwe.stats.TotalDetections)
}

func (wwe *WakeWordEngine) periodicMaintenance(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Adaptive threshold adjustment
			if wwe.config.AdaptiveThreshold {
				wwe.adjustThresholdBasedOnEnvironment()
			}

			// Log statistics periodically
			if wwe.stats.TotalDetections > 0 {
				wwe.logger.WithFields(logrus.Fields{
					"total_detections":   wwe.stats.TotalDetections,
					"avg_confidence":     wwe.stats.AverageConfidence,
					"processing_latency": wwe.stats.ProcessingLatency,
				}).Debug("Wake word detection stats")
			}
		}
	}
}

func (wwe *WakeWordEngine) adjustThresholdBasedOnEnvironment() {
	// Implement adaptive threshold logic based on environment noise
	// This could analyze background noise levels and adjust sensitivity accordingly
}

// NewCircularAudioBuffer creates a new circular audio buffer
func NewCircularAudioBuffer(size int, duration time.Duration, sampleRate int) *CircularAudioBuffer {
	return &CircularAudioBuffer{
		buffer:     make([]byte, size),
		size:       size,
		duration:   duration,
		sampleRate: sampleRate,
	}
}

// Write adds audio data to the circular buffer
func (cab *CircularAudioBuffer) Write(data []byte) {
	cab.mu.Lock()
	defer cab.mu.Unlock()

	for _, b := range data {
		cab.buffer[cab.writePos] = b
		cab.writePos = (cab.writePos + 1) % cab.size

		if cab.writePos == 0 {
			cab.isFull = true
		}
	}
}

// ReadAll returns all audio data in the buffer
func (cab *CircularAudioBuffer) ReadAll() []byte {
	cab.mu.RLock()
	defer cab.mu.RUnlock()

	if !cab.isFull {
		// Buffer not full yet, return from start to writePos
		result := make([]byte, cab.writePos)
		copy(result, cab.buffer[:cab.writePos])
		return result
	}

	// Buffer is full, return in correct order
	result := make([]byte, cab.size)
	copy(result, cab.buffer[cab.writePos:])
	copy(result[cab.size-cab.writePos:], cab.buffer[:cab.writePos])
	return result
}
