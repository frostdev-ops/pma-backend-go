package stt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// AudioSource represents different types of audio input sources
type AudioSource string

const (
	SourceBrowser   AudioSource = "browser"
	SourceUSB       AudioSource = "usb"
	SourceBluetooth AudioSource = "bluetooth"
	SourcePhone     AudioSource = "phone"
	SourceInternal  AudioSource = "internal"
)

// AudioFormat defines the standardized audio format for processing
type AudioFormat struct {
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	BitDepth   int    `json:"bit_depth"`
	Format     string `json:"format"` // "wav", "pcm", "mp3"
}

// AudioDevice represents a specific audio input device
type AudioDevice struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Source       AudioSource `json:"source"`
	IsActive     bool        `json:"is_active"`
	Priority     int         `json:"priority"` // Higher = more priority
	Quality      float64     `json:"quality"`  // Signal quality 0-1
	Format       AudioFormat `json:"format"`
	LastSeen     time.Time   `json:"last_seen"`
	Capabilities []string    `json:"capabilities"` // "wake_word", "full_duplex", "noise_canceling"
}

// AudioChunk represents a piece of audio data with metadata
type AudioChunk struct {
	Data        []byte        `json:"data"`
	Timestamp   time.Time     `json:"timestamp"`
	Source      AudioSource   `json:"source"`
	DeviceID    string        `json:"device_id"`
	Format      AudioFormat   `json:"format"`
	Duration    time.Duration `json:"duration"`
	VolumeLevel float64       `json:"volume_level"`
	Quality     float64       `json:"quality"`
}

// AudioManager manages multiple audio input sources and routing
type AudioManager struct {
	logger        *logrus.Logger
	devices       map[string]*AudioDevice
	activeStreams map[string]*AudioStream
	mixer         *AudioMixer
	config        *AudioConfig
	mu            sync.RWMutex

	// Event channels
	deviceEvents chan DeviceEvent
	audioOutput  chan AudioChunk
	errorOutput  chan error

	// WebSocket connections for browser audio
	wsConnections map[string]*websocket.Conn
	wsMu          sync.RWMutex
}

// AudioStream represents an active audio stream from a source
type AudioStream struct {
	DeviceID      string
	Source        AudioSource
	StartTime     time.Time
	ChunkCount    int64
	BytesReceived int64
	IsActive      bool
	Context       context.Context
	Cancel        context.CancelFunc
}

// DeviceEvent represents device connection/disconnection events
type DeviceEvent struct {
	Type      string       `json:"type"` // "connected", "disconnected", "quality_change"
	Device    *AudioDevice `json:"device"`
	Timestamp time.Time    `json:"timestamp"`
}

// AudioConfig holds configuration for the audio manager
type AudioConfig struct {
	MaxConcurrentStreams int           `json:"max_concurrent_streams"`
	BufferSize           int           `json:"buffer_size"`
	DefaultSampleRate    int           `json:"default_sample_rate"`
	DefaultChannels      int           `json:"default_channels"`
	QualityThreshold     float64       `json:"quality_threshold"`
	AutoSourceSwitch     bool          `json:"auto_source_switch"`
	PowerSavingMode      bool          `json:"power_saving_mode"`
	NoiseReduction       bool          `json:"noise_reduction"`
	EchoCancellation     bool          `json:"echo_cancellation"`
	DeviceTimeout        time.Duration `json:"device_timeout"`
}

// NewAudioManager creates a new audio manager instance
func NewAudioManager(logger *logrus.Logger, config *AudioConfig) *AudioManager {
	return &AudioManager{
		logger:        logger,
		devices:       make(map[string]*AudioDevice),
		activeStreams: make(map[string]*AudioStream),
		mixer:         NewAudioMixer(logger, config),
		config:        config,
		deviceEvents:  make(chan DeviceEvent, 100),
		audioOutput:   make(chan AudioChunk, 1000),
		errorOutput:   make(chan error, 100),
		wsConnections: make(map[string]*websocket.Conn),
	}
}

// Start begins the audio manager operation
func (am *AudioManager) Start(ctx context.Context) error {
	am.logger.Info("Starting Audio Manager...")

	// Start device discovery
	go am.discoverDevices(ctx)

	// Start event processing
	go am.processEvents(ctx)

	// Start audio processing
	go am.processAudio(ctx)

	// Start USB device monitoring
	go am.monitorUSBDevices(ctx)

	// Start Bluetooth device monitoring
	go am.monitorBluetoothDevices(ctx)

	am.logger.Info("Audio Manager started successfully")
	return nil
}

// RegisterWebSocketConnection registers a browser WebSocket connection for audio streaming
func (am *AudioManager) RegisterWebSocketConnection(connectionID string, conn *websocket.Conn) error {
	am.wsMu.Lock()
	defer am.wsMu.Unlock()

	am.wsConnections[connectionID] = conn
	am.logger.WithField("connection_id", connectionID).Info("WebSocket audio connection registered")

	// Start listening for audio data from this connection
	go am.handleWebSocketAudio(connectionID, conn)

	return nil
}

// UnregisterWebSocketConnection removes a WebSocket connection
func (am *AudioManager) UnregisterWebSocketConnection(connectionID string) {
	am.wsMu.Lock()
	defer am.wsMu.Unlock()

	if conn, exists := am.wsConnections[connectionID]; exists {
		conn.Close()
		delete(am.wsConnections, connectionID)
		am.logger.WithField("connection_id", connectionID).Info("WebSocket audio connection unregistered")
	}
}

// handleWebSocketAudio processes audio data from browser WebSocket connection
func (am *AudioManager) handleWebSocketAudio(connectionID string, conn *websocket.Conn) {
	defer am.UnregisterWebSocketConnection(connectionID)

	// Register browser audio device
	device := &AudioDevice{
		ID:       fmt.Sprintf("browser_%s", connectionID),
		Name:     "Browser Microphone",
		Source:   SourceBrowser,
		IsActive: true,
		Priority: 3, // Medium priority
		Quality:  0.8,
		Format: AudioFormat{
			SampleRate: 16000,
			Channels:   1,
			BitDepth:   16,
			Format:     "pcm",
		},
		LastSeen:     time.Now(),
		Capabilities: []string{"wake_word", "full_duplex"},
	}

	am.addDevice(device)
	defer am.removeDevice(device.ID)

	for {
		var audioData map[string]interface{}
		err := conn.ReadJSON(&audioData)
		if err != nil {
			am.logger.WithError(err).Error("Error reading WebSocket audio data")
			break
		}

		// Process the audio data
		if data, ok := audioData["audio_data"].([]byte); ok {
			chunk := AudioChunk{
				Data:        data,
				Timestamp:   time.Now(),
				Source:      SourceBrowser,
				DeviceID:    device.ID,
				Format:      device.Format,
				Duration:    time.Duration(len(data)/2/16000) * time.Second, // Approximate
				VolumeLevel: am.calculateVolumeLevel(data),
				Quality:     device.Quality,
			}

			// Send to audio processing pipeline
			select {
			case am.audioOutput <- chunk:
			default:
				am.logger.Warn("Audio output buffer full, dropping chunk")
			}
		}
	}
}

// discoverDevices continuously discovers available audio devices
func (am *AudioManager) discoverDevices(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.scanForDevices()
		}
	}
}

// scanForDevices scans for available audio input devices
func (am *AudioManager) scanForDevices() {
	// USB Microphone discovery
	am.discoverUSBMicrophones()

	// Bluetooth audio device discovery
	am.discoverBluetoothDevices()

	// Internal microphone discovery
	am.discoverInternalMicrophones()
}

// GetActiveDevices returns currently active audio devices
func (am *AudioManager) GetActiveDevices() []*AudioDevice {
	am.mu.RLock()
	defer am.mu.RUnlock()

	devices := make([]*AudioDevice, 0, len(am.devices))
	for _, device := range am.devices {
		if device.IsActive {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetAudioStream returns the audio output channel for processing
func (am *AudioManager) GetAudioStream() <-chan AudioChunk {
	return am.audioOutput
}

// GetDeviceEvents returns the device event channel
func (am *AudioManager) GetDeviceEvents() <-chan DeviceEvent {
	return am.deviceEvents
}

// SetPrimaryDevice sets a device as the primary audio source
func (am *AudioManager) SetPrimaryDevice(deviceID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Find the device
	device, exists := am.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	// Reset all priorities
	for _, d := range am.devices {
		if d.Priority > 5 {
			d.Priority = 3 // Reset to medium priority
		}
	}

	// Set new primary device
	device.Priority = 10 // Highest priority

	am.logger.WithFields(logrus.Fields{
		"device_id":   deviceID,
		"device_name": device.Name,
	}).Info("Primary audio device set")

	return nil
}

// Helper methods
func (am *AudioManager) addDevice(device *AudioDevice) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.devices[device.ID] = device

	// Send device event
	select {
	case am.deviceEvents <- DeviceEvent{
		Type:      "connected",
		Device:    device,
		Timestamp: time.Now(),
	}:
	default:
	}

	am.logger.WithFields(logrus.Fields{
		"device_id":   device.ID,
		"device_name": device.Name,
		"source":      device.Source,
	}).Info("Audio device added")
}

func (am *AudioManager) removeDevice(deviceID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if device, exists := am.devices[deviceID]; exists {
		delete(am.devices, deviceID)

		// Send device event
		select {
		case am.deviceEvents <- DeviceEvent{
			Type:      "disconnected",
			Device:    device,
			Timestamp: time.Now(),
		}:
		default:
		}

		am.logger.WithField("device_id", deviceID).Info("Audio device removed")
	}
}

func (am *AudioManager) calculateVolumeLevel(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Simple RMS calculation for volume level
	var sum float64
	for i := 0; i < len(data)-1; i += 2 {
		sample := int16(data[i]) | (int16(data[i+1]) << 8)
		sum += float64(sample * sample)
	}

	rms := sum / float64(len(data)/2)
	return rms / (32768.0 * 32768.0) // Normalize to 0-1
}

// Additional methods for USB, Bluetooth, and internal device discovery will be implemented
// These will integrate with system APIs for device enumeration and monitoring
