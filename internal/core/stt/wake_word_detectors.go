package stt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PorcupineDetector implements wake word detection using Porcupine
type PorcupineDetector struct {
	logger       *logrus.Logger
	config       *WakeWordConfig
	isActive     bool
	keywordPaths map[string]string
	mu           sync.RWMutex
}

// OpenWakeWordDetector implements wake word detection using OpenWakeWord
type OpenWakeWordDetector struct {
	logger     *logrus.Logger
	config     *WakeWordConfig
	serverURL  string
	httpClient *http.Client
	isActive   bool
	mu         sync.RWMutex
}

// SimpleKeywordDetector implements basic keyword matching as fallback
type SimpleKeywordDetector struct {
	logger          *logrus.Logger
	config          *WakeWordConfig
	audioBuffer     []byte
	isActive        bool
	mu              sync.RWMutex
	energyThreshold float64
}

// NewPorcupineDetector creates a new Porcupine-based wake word detector
func NewPorcupineDetector(logger *logrus.Logger, config *WakeWordConfig) (WakeWordDetector, error) {
	detector := &PorcupineDetector{
		logger:       logger,
		config:       config,
		keywordPaths: make(map[string]string),
	}

	// Check if Porcupine is available
	if _, err := exec.LookPath("python3"); err != nil {
		return nil, fmt.Errorf("python3 not found: %w", err)
	}

	return detector, nil
}

// Initialize sets up the Porcupine detector
func (pd *PorcupineDetector) Initialize(config *WakeWordConfig) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.config = config

	// Set up keyword model paths
	for _, keyword := range config.Keywords {
		if keyword.Enabled {
			// In a real implementation, these would be actual Porcupine .ppn files
			pd.keywordPaths[keyword.Word] = fmt.Sprintf("%s/%s.ppn", config.ModelPath, keyword.Word)
		}
	}

	pd.isActive = true
	pd.logger.Info("Porcupine detector initialized")
	return nil
}

// DetectWakeWord performs wake word detection on audio data
func (pd *PorcupineDetector) DetectWakeWord(audioData []byte) (*WakeWordDetection, error) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	if !pd.isActive {
		return &WakeWordDetection{Detected: false}, nil
	}

	// Convert audio data to int16 PCM format for Porcupine
	samples := make([]int16, len(audioData)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(audioData[i*2]) | (int16(audioData[i*2+1]) << 8)
	}

	// In a real implementation, this would call the Porcupine library
	// For now, we'll simulate detection based on energy and simple pattern matching
	energy := pd.calculateAudioEnergy(samples)

	if energy < 0.01 { // Too quiet
		return &WakeWordDetection{Detected: false}, nil
	}

	// Simulate Porcupine detection logic
	for _, keyword := range pd.config.Keywords {
		if keyword.Enabled {
			// Simple simulation - in reality this would use Porcupine's neural network
			confidence := pd.simulatePorcupineDetection(audioData, keyword.Word)
			if confidence >= keyword.Sensitivity {
				return &WakeWordDetection{
					Detected:   true,
					Keyword:    keyword.Word,
					Confidence: confidence,
					StartTime:  0,
					EndTime:    time.Duration(len(audioData)/2/16000) * time.Second,
				}, nil
			}
		}
	}

	return &WakeWordDetection{Detected: false}, nil
}

// UpdateSensitivity updates the detection sensitivity
func (pd *PorcupineDetector) UpdateSensitivity(sensitivity float64) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.config.SensitivityLevel = sensitivity
	pd.logger.WithField("sensitivity", sensitivity).Info("Porcupine sensitivity updated")
	return nil
}

// AddKeyword adds a new keyword to the detector
func (pd *PorcupineDetector) AddKeyword(keyword WakeWordKeyword) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if keyword.Enabled {
		pd.keywordPaths[keyword.Word] = fmt.Sprintf("%s/%s.ppn", pd.config.ModelPath, keyword.Word)
		pd.logger.WithField("keyword", keyword.Word).Info("Keyword added to Porcupine detector")
	}

	return nil
}

// RemoveKeyword removes a keyword from the detector
func (pd *PorcupineDetector) RemoveKeyword(word string) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	delete(pd.keywordPaths, word)
	pd.logger.WithField("keyword", word).Info("Keyword removed from Porcupine detector")
	return nil
}

// GetSupportedKeywords returns the list of supported keywords
func (pd *PorcupineDetector) GetSupportedKeywords() []string {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	keywords := make([]string, 0, len(pd.keywordPaths))
	for keyword := range pd.keywordPaths {
		keywords = append(keywords, keyword)
	}

	return keywords
}

// Cleanup cleans up the detector resources
func (pd *PorcupineDetector) Cleanup() error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.isActive = false
	pd.logger.Info("Porcupine detector cleaned up")
	return nil
}

// Helper methods for PorcupineDetector
func (pd *PorcupineDetector) calculateAudioEnergy(samples []int16) float64 {
	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	return sum / float64(len(samples)) / (32768.0 * 32768.0)
}

func (pd *PorcupineDetector) simulatePorcupineDetection(audioData []byte, keyword string) float64 {
	// This is a simulation - real Porcupine would use trained neural networks
	// For demonstration, we'll use simple audio characteristics
	energy := pd.calculateAudioEnergy(pd.bytesToInt16(audioData))

	// Simulate confidence based on energy and keyword
	baseConfidence := energy * 10.0

	// Add some keyword-specific logic
	switch strings.ToLower(keyword) {
	case "hey assistant", "hey pma":
		return math.Min(baseConfidence*1.2, 1.0)
	case "wake up":
		return math.Min(baseConfidence*1.1, 1.0)
	default:
		return math.Min(baseConfidence, 1.0)
	}
}

func (pd *PorcupineDetector) bytesToInt16(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(data[i*2]) | (int16(data[i*2+1]) << 8)
	}
	return samples
}

// NewOpenWakeWordDetector creates a new OpenWakeWord-based detector
func NewOpenWakeWordDetector(logger *logrus.Logger, config *WakeWordConfig) (WakeWordDetector, error) {
	detector := &OpenWakeWordDetector{
		logger:    logger,
		config:    config,
		serverURL: "http://localhost:10400", // Default OpenWakeWord server port
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Test if OpenWakeWord server is available
	resp, err := detector.httpClient.Get(detector.serverURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("OpenWakeWord server not available: %w", err)
	}
	resp.Body.Close()

	return detector, nil
}

// Initialize sets up the OpenWakeWord detector
func (owd *OpenWakeWordDetector) Initialize(config *WakeWordConfig) error {
	owd.mu.Lock()
	defer owd.mu.Unlock()

	owd.config = config
	owd.isActive = true

	// Configure OpenWakeWord with our keywords
	configPayload := map[string]interface{}{
		"sensitivity": config.SensitivityLevel,
		"keywords":    config.Keywords,
	}

	jsonData, _ := json.Marshal(configPayload)
	resp, err := owd.httpClient.Post(owd.serverURL+"/configure", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to configure OpenWakeWord: %w", err)
	}
	resp.Body.Close()

	owd.logger.Info("OpenWakeWord detector initialized")
	return nil
}

// DetectWakeWord performs wake word detection using OpenWakeWord
func (owd *OpenWakeWordDetector) DetectWakeWord(audioData []byte) (*WakeWordDetection, error) {
	owd.mu.RLock()
	defer owd.mu.RUnlock()

	if !owd.isActive {
		return &WakeWordDetection{Detected: false}, nil
	}

	// Send audio data to OpenWakeWord server
	resp, err := owd.httpClient.Post(owd.serverURL+"/detect", "application/octet-stream", bytes.NewBuffer(audioData))
	if err != nil {
		return nil, fmt.Errorf("OpenWakeWord detection request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Detected   bool    `json:"detected"`
		Keyword    string  `json:"keyword"`
		Confidence float64 `json:"confidence"`
		StartTime  float64 `json:"start_time"`
		EndTime    float64 `json:"end_time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode OpenWakeWord response: %w", err)
	}

	return &WakeWordDetection{
		Detected:   result.Detected,
		Keyword:    result.Keyword,
		Confidence: result.Confidence,
		StartTime:  time.Duration(result.StartTime * float64(time.Second)),
		EndTime:    time.Duration(result.EndTime * float64(time.Second)),
	}, nil
}

// Other OpenWakeWordDetector methods (UpdateSensitivity, AddKeyword, etc.)
func (owd *OpenWakeWordDetector) UpdateSensitivity(sensitivity float64) error {
	owd.mu.Lock()
	defer owd.mu.Unlock()

	configPayload := map[string]interface{}{
		"sensitivity": sensitivity,
	}

	jsonData, _ := json.Marshal(configPayload)
	resp, err := owd.httpClient.Post(owd.serverURL+"/configure", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	resp.Body.Close()

	owd.config.SensitivityLevel = sensitivity
	return nil
}

func (owd *OpenWakeWordDetector) AddKeyword(keyword WakeWordKeyword) error {
	// Implementation for adding keywords to OpenWakeWord
	return nil
}

func (owd *OpenWakeWordDetector) RemoveKeyword(word string) error {
	// Implementation for removing keywords from OpenWakeWord
	return nil
}

func (owd *OpenWakeWordDetector) GetSupportedKeywords() []string {
	// Return supported keywords from OpenWakeWord
	return []string{"hey assistant", "wake up", "hey pma"}
}

func (owd *OpenWakeWordDetector) Cleanup() error {
	owd.mu.Lock()
	defer owd.mu.Unlock()

	owd.isActive = false
	return nil
}

// NewSimpleKeywordDetector creates a fallback simple keyword detector
func NewSimpleKeywordDetector(logger *logrus.Logger, config *WakeWordConfig) WakeWordDetector {
	return &SimpleKeywordDetector{
		logger:          logger,
		config:          config,
		audioBuffer:     make([]byte, 0, 8192),
		energyThreshold: 0.01,
	}
}

// Initialize sets up the simple keyword detector
func (skd *SimpleKeywordDetector) Initialize(config *WakeWordConfig) error {
	skd.mu.Lock()
	defer skd.mu.Unlock()

	skd.config = config
	skd.isActive = true
	skd.logger.Info("Simple keyword detector initialized")
	return nil
}

// DetectWakeWord performs simple energy-based wake word detection
func (skd *SimpleKeywordDetector) DetectWakeWord(audioData []byte) (*WakeWordDetection, error) {
	skd.mu.Lock()
	defer skd.mu.Unlock()

	if !skd.isActive {
		return &WakeWordDetection{Detected: false}, nil
	}

	// Add to rolling buffer
	skd.audioBuffer = append(skd.audioBuffer, audioData...)

	// Keep buffer to reasonable size (about 2 seconds of audio)
	maxBufferSize := 16000 * 2 * 2 // 16kHz, 2 seconds, 16-bit
	if len(skd.audioBuffer) > maxBufferSize {
		skd.audioBuffer = skd.audioBuffer[len(skd.audioBuffer)-maxBufferSize:]
	}

	// Calculate audio energy
	energy := skd.calculateEnergy(audioData)

	// Simple threshold-based detection
	if energy > skd.energyThreshold {
		// Check if we have enough audio for analysis
		if len(skd.audioBuffer) >= 16000 { // 1 second of audio
			// Simple pattern matching could go here
			// For now, detect based on energy patterns

			for _, keyword := range skd.config.Keywords {
				if keyword.Enabled {
					confidence := skd.analyzeForKeyword(skd.audioBuffer, keyword.Word)
					if confidence >= keyword.Sensitivity {
						return &WakeWordDetection{
							Detected:   true,
							Keyword:    keyword.Word,
							Confidence: confidence,
							StartTime:  0,
							EndTime:    time.Duration(len(audioData)/2/16000) * time.Second,
						}, nil
					}
				}
			}
		}
	}

	return &WakeWordDetection{Detected: false}, nil
}

// Helper methods for SimpleKeywordDetector
func (skd *SimpleKeywordDetector) calculateEnergy(data []byte) float64 {
	if len(data) < 2 {
		return 0.0
	}

	var sum float64
	for i := 0; i < len(data)-1; i += 2 {
		sample := int16(data[i]) | (int16(data[i+1]) << 8)
		sum += float64(sample * sample)
	}

	return sum / float64(len(data)/2) / (32768.0 * 32768.0)
}

func (skd *SimpleKeywordDetector) analyzeForKeyword(audioData []byte, keyword string) float64 {
	// Very simple analysis - in reality this would use more sophisticated techniques
	energy := skd.calculateEnergy(audioData)

	// Base confidence on energy level and some keyword-specific logic
	baseConfidence := energy * 5.0

	// Keyword-specific adjustments
	switch strings.ToLower(keyword) {
	case "hey assistant", "hey pma":
		// Look for energy patterns that might indicate speech
		return math.Min(baseConfidence*1.5, 1.0)
	case "wake up":
		return math.Min(baseConfidence*1.3, 1.0)
	default:
		return math.Min(baseConfidence, 1.0)
	}
}

// Implement remaining methods for SimpleKeywordDetector
func (skd *SimpleKeywordDetector) UpdateSensitivity(sensitivity float64) error {
	skd.mu.Lock()
	defer skd.mu.Unlock()

	skd.energyThreshold = 0.01 * (1.0 - sensitivity) // Invert sensitivity for threshold
	return nil
}

func (skd *SimpleKeywordDetector) AddKeyword(keyword WakeWordKeyword) error {
	// Simple detector can handle any keyword
	return nil
}

func (skd *SimpleKeywordDetector) RemoveKeyword(word string) error {
	// Simple detector doesn't need to track specific keywords
	return nil
}

func (skd *SimpleKeywordDetector) GetSupportedKeywords() []string {
	// Simple detector can "support" any keyword (though with limited accuracy)
	keywords := make([]string, 0, len(skd.config.Keywords))
	for _, kw := range skd.config.Keywords {
		keywords = append(keywords, kw.Word)
	}
	return keywords
}

func (skd *SimpleKeywordDetector) Cleanup() error {
	skd.mu.Lock()
	defer skd.mu.Unlock()

	skd.isActive = false
	skd.audioBuffer = nil
	return nil
}
