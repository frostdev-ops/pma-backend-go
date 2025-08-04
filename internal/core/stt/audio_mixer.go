package stt

import (
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AudioMixer handles mixing and processing audio from multiple sources
type AudioMixer struct {
	logger        *logrus.Logger
	config        *AudioConfig
	activeStreams map[string]*MixerStream
	mu            sync.RWMutex
	outputFormat  AudioFormat
}

// MixerStream represents an active audio stream in the mixer
type MixerStream struct {
	Source     AudioSource
	DeviceID   string
	Priority   int
	Volume     float64
	Muted      bool
	LastSeen   time.Time
	BufferSize int
	Quality    float64
}

// NewAudioMixer creates a new audio mixer
func NewAudioMixer(logger *logrus.Logger, config *AudioConfig) *AudioMixer {
	return &AudioMixer{
		logger:        logger,
		config:        config,
		activeStreams: make(map[string]*MixerStream),
		outputFormat: AudioFormat{
			SampleRate: config.DefaultSampleRate,
			Channels:   config.DefaultChannels,
			BitDepth:   16,
			Format:     "pcm",
		},
	}
}

// ProcessAudioChunk processes and normalizes an audio chunk
func (am *AudioMixer) ProcessAudioChunk(chunk AudioChunk) AudioChunk {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Update or create stream info
	streamKey := string(chunk.Source) + "_" + chunk.DeviceID
	stream, exists := am.activeStreams[streamKey]
	if !exists {
		stream = &MixerStream{
			Source:     chunk.Source,
			DeviceID:   chunk.DeviceID,
			Priority:   am.getSourcePriority(chunk.Source),
			Volume:     1.0,
			Muted:      false,
			BufferSize: 4096,
			Quality:    chunk.Quality,
		}
		am.activeStreams[streamKey] = stream
	}

	stream.LastSeen = chunk.Timestamp
	stream.Quality = chunk.Quality

	// Skip processing if muted
	if stream.Muted {
		return chunk
	}

	// Normalize audio format
	normalizedData := am.normalizeAudioFormat(chunk.Data, chunk.Format, am.outputFormat)

	// Apply volume control
	if stream.Volume != 1.0 {
		normalizedData = am.applyVolumeControl(normalizedData, stream.Volume)
	}

	// Apply noise reduction if enabled
	if am.config.NoiseReduction {
		normalizedData = am.applyNoiseReduction(normalizedData)
	}

	// Apply echo cancellation if enabled
	if am.config.EchoCancellation {
		normalizedData = am.applyEchoCancellation(normalizedData, streamKey)
	}

	// Update chunk with processed data
	processedChunk := chunk
	processedChunk.Data = normalizedData
	processedChunk.Format = am.outputFormat
	processedChunk.VolumeLevel = am.calculateVolumeLevel(normalizedData)

	return processedChunk
}

// normalizeAudioFormat converts audio data to the target format
func (am *AudioMixer) normalizeAudioFormat(data []byte, sourceFormat, targetFormat AudioFormat) []byte {
	// If formats match, return as-is
	if sourceFormat.SampleRate == targetFormat.SampleRate &&
		sourceFormat.Channels == targetFormat.Channels &&
		sourceFormat.BitDepth == targetFormat.BitDepth {
		return data
	}

	// Convert sample rate if needed
	if sourceFormat.SampleRate != targetFormat.SampleRate {
		data = am.resampleAudio(data, sourceFormat.SampleRate, targetFormat.SampleRate)
	}

	// Convert channels if needed
	if sourceFormat.Channels != targetFormat.Channels {
		data = am.convertChannels(data, sourceFormat.Channels, targetFormat.Channels)
	}

	// Convert bit depth if needed
	if sourceFormat.BitDepth != targetFormat.BitDepth {
		data = am.convertBitDepth(data, sourceFormat.BitDepth, targetFormat.BitDepth)
	}

	return data
}

// resampleAudio performs simple linear interpolation resampling
func (am *AudioMixer) resampleAudio(data []byte, sourceSampleRate, targetSampleRate int) []byte {
	if sourceSampleRate == targetSampleRate {
		return data
	}

	// Convert bytes to int16 samples
	sourceLength := len(data) / 2
	sourceSamples := make([]int16, sourceLength)
	for i := 0; i < sourceLength; i++ {
		sourceSamples[i] = int16(data[i*2]) | (int16(data[i*2+1]) << 8)
	}

	// Calculate target length
	ratio := float64(targetSampleRate) / float64(sourceSampleRate)
	targetLength := int(float64(sourceLength) * ratio)
	targetSamples := make([]int16, targetLength)

	// Linear interpolation resampling
	for i := 0; i < targetLength; i++ {
		sourcePos := float64(i) / ratio
		sourceIndex := int(sourcePos)
		fraction := sourcePos - float64(sourceIndex)

		if sourceIndex >= sourceLength-1 {
			targetSamples[i] = sourceSamples[sourceLength-1]
		} else {
			// Linear interpolation
			sample1 := float64(sourceSamples[sourceIndex])
			sample2 := float64(sourceSamples[sourceIndex+1])
			targetSamples[i] = int16(sample1 + fraction*(sample2-sample1))
		}
	}

	// Convert back to bytes
	result := make([]byte, targetLength*2)
	for i, sample := range targetSamples {
		result[i*2] = byte(sample & 0xff)
		result[i*2+1] = byte((sample >> 8) & 0xff)
	}

	return result
}

// convertChannels converts between mono and stereo
func (am *AudioMixer) convertChannels(data []byte, sourceChannels, targetChannels int) []byte {
	if sourceChannels == targetChannels {
		return data
	}

	sampleCount := len(data) / 2 / sourceChannels

	if sourceChannels == 1 && targetChannels == 2 {
		// Mono to stereo - duplicate each sample
		result := make([]byte, len(data)*2)
		for i := 0; i < sampleCount; i++ {
			sample := data[i*2 : i*2+2]
			copy(result[i*4:i*4+2], sample)   // Left channel
			copy(result[i*4+2:i*4+4], sample) // Right channel
		}
		return result
	} else if sourceChannels == 2 && targetChannels == 1 {
		// Stereo to mono - average the channels
		result := make([]byte, len(data)/2)
		for i := 0; i < sampleCount; i++ {
			left := int16(data[i*4]) | (int16(data[i*4+1]) << 8)
			right := int16(data[i*4+2]) | (int16(data[i*4+3]) << 8)
			mono := (left + right) / 2
			result[i*2] = byte(mono & 0xff)
			result[i*2+1] = byte((mono >> 8) & 0xff)
		}
		return result
	}

	// For other channel configurations, return as-is
	return data
}

// convertBitDepth converts between different bit depths (basic implementation)
func (am *AudioMixer) convertBitDepth(data []byte, sourceBitDepth, targetBitDepth int) []byte {
	if sourceBitDepth == targetBitDepth {
		return data
	}

	// For now, we only support 16-bit
	// In a full implementation, you'd handle 8-bit, 24-bit, 32-bit, etc.
	return data
}

// applyVolumeControl applies volume adjustment to audio data
func (am *AudioMixer) applyVolumeControl(data []byte, volume float64) []byte {
	if volume == 1.0 {
		return data
	}

	result := make([]byte, len(data))

	for i := 0; i < len(data)-1; i += 2 {
		sample := int16(data[i]) | (int16(data[i+1]) << 8)
		adjustedSample := int16(float64(sample) * volume)

		// Clamp to prevent overflow
		if adjustedSample > 32767 {
			adjustedSample = 32767
		} else if adjustedSample < -32768 {
			adjustedSample = -32768
		}

		result[i] = byte(adjustedSample & 0xff)
		result[i+1] = byte((adjustedSample >> 8) & 0xff)
	}

	return result
}

// applyNoiseReduction applies basic noise reduction
func (am *AudioMixer) applyNoiseReduction(data []byte) []byte {
	if len(data) < 4 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	// Simple noise gate - zero out samples below threshold
	noiseThreshold := int16(100) // Adjust based on needs

	for i := 0; i < len(result)-1; i += 2 {
		sample := int16(result[i]) | (int16(result[i+1]) << 8)
		if sample < noiseThreshold && sample > -noiseThreshold {
			result[i] = 0
			result[i+1] = 0
		}
	}

	return result
}

// applyEchoCancellation applies basic echo cancellation
func (am *AudioMixer) applyEchoCancellation(data []byte, streamKey string) []byte {
	// This is a placeholder for echo cancellation
	// Real echo cancellation would require complex algorithms like adaptive filters
	// For now, we'll just return the data as-is
	return data
}

// calculateVolumeLevel calculates the RMS volume level
func (am *AudioMixer) calculateVolumeLevel(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var sum float64
	sampleCount := len(data) / 2

	for i := 0; i < sampleCount; i++ {
		sample := int16(data[i*2]) | (int16(data[i*2+1]) << 8)
		sum += float64(sample * sample)
	}

	rms := math.Sqrt(sum / float64(sampleCount))
	return rms / 32768.0 // Normalize to 0-1 range
}

// getSourcePriority returns the priority for a given audio source
func (am *AudioMixer) getSourcePriority(source AudioSource) int {
	switch source {
	case SourceBrowser:
		return 3
	case SourceUSB:
		return 4
	case SourceBluetooth:
		return 2
	case SourcePhone:
		return 5
	case SourceInternal:
		return 1
	default:
		return 1
	}
}

// SetStreamVolume sets the volume for a specific stream
func (am *AudioMixer) SetStreamVolume(source AudioSource, deviceID string, volume float64) {
	am.mu.Lock()
	defer am.mu.Unlock()

	streamKey := string(source) + "_" + deviceID
	if stream, exists := am.activeStreams[streamKey]; exists {
		stream.Volume = math.Max(0.0, math.Min(1.0, volume)) // Clamp between 0 and 1
	}
}

// MuteStream mutes or unmutes a specific stream
func (am *AudioMixer) MuteStream(source AudioSource, deviceID string, muted bool) {
	am.mu.Lock()
	defer am.mu.Unlock()

	streamKey := string(source) + "_" + deviceID
	if stream, exists := am.activeStreams[streamKey]; exists {
		stream.Muted = muted
	}
}

// GetActiveStreams returns information about currently active streams
func (am *AudioMixer) GetActiveStreams() map[string]*MixerStream {
	am.mu.RLock()
	defer am.mu.RUnlock()

	result := make(map[string]*MixerStream)
	for key, stream := range am.activeStreams {
		// Create a copy to avoid race conditions
		streamCopy := *stream
		result[key] = &streamCopy
	}

	return result
}

// CleanupStaleStreams removes streams that haven't been seen recently
func (am *AudioMixer) CleanupStaleStreams(timeout time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	staleStreams := make([]string, 0)

	for key, stream := range am.activeStreams {
		if now.Sub(stream.LastSeen) > timeout {
			staleStreams = append(staleStreams, key)
		}
	}

	for _, key := range staleStreams {
		delete(am.activeStreams, key)
		am.logger.WithField("stream", key).Debug("Cleaned up stale audio stream")
	}
}
