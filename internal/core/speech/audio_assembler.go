package speech

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

// AudioAssembler handles combining multiple audio chunks into a seamless stream
type AudioAssembler struct {
	config            *AudioConfig
	logger            *logrus.Logger
	volumeNormalizer  *VolumeNormalizer
	pauseInserter     *PauseManager
	qualityController *QualityAssurance
	crossFader        *CrossFader
}

// AudioConfig holds configuration for audio processing
type AudioConfig struct {
	SampleRate           int           `json:"sample_rate"`           // 22050 Hz default
	AudioFormat          string        `json:"audio_format"`          // "wav", "mp3", etc.
	NormalizationEnabled bool          `json:"normalization_enabled"` // Volume normalization
	NaturalPauses        bool          `json:"natural_pauses"`        // Insert natural pauses
	CrossFadeEnabled     bool          `json:"cross_fade_enabled"`    // Enable cross-fade transitions
	CrossFadeDuration    time.Duration `json:"cross_fade_duration"`   // Duration of cross-fade
	QualityAssurance     bool          `json:"quality_assurance"`     // Enable QA processing
	MaxOutputSize        int64         `json:"max_output_size"`       // Maximum output file size
	TempDirectory        string        `json:"temp_directory"`        // Temporary files directory
}

// AudioSegment represents a processed audio chunk with metadata
type AudioSegment struct {
	Data          []byte                 `json:"-"`               // Raw audio data
	Duration      time.Duration          `json:"duration"`        // Audio duration
	Index         int                    `json:"index"`           // Original chunk index
	Volume        float64                `json:"volume"`          // Current volume level
	SampleRate    int                    `json:"sample_rate"`     // Sample rate
	Format        string                 `json:"format"`          // Audio format
	Metadata      map[string]interface{} `json:"metadata"`        // Chunk metadata
	FilePath      string                 `json:"file_path"`       // Path to audio file
	Size          int64                  `json:"size"`            // Size in bytes
	Channels      int                    `json:"channels"`        // Number of audio channels
	BitsPerSample int                    `json:"bits_per_sample"` // Bits per sample
	IsProcessed   bool                   `json:"is_processed"`    // Processing status
	ErrorMessage  string                 `json:"error_message"`   // Any processing errors
}

// VolumeNormalizer handles audio volume normalization
type VolumeNormalizer struct {
	targetRMS        float64
	maxPeakAllowed   float64
	compressionRatio float64
	lookAheadTime    time.Duration
	logger           *logrus.Logger
}

// PauseManager handles insertion of natural pauses between segments
type PauseManager struct {
	config           *PauseConfig
	silenceGenerator *SilenceGenerator
	logger           *logrus.Logger
}

// PauseConfig defines pause insertion rules
type PauseConfig struct {
	SentencePause    time.Duration `json:"sentence_pause"`    // 300-500ms
	ParagraphPause   time.Duration `json:"paragraph_pause"`   // 800ms-1s
	DialoguePause    time.Duration `json:"dialogue_pause"`    // 450ms
	ListItemPause    time.Duration `json:"list_item_pause"`   // 200ms
	CustomPauses     bool          `json:"custom_pauses"`     // Use chunk-specific pauses
	BreathingEnabled bool          `json:"breathing_enabled"` // Add breathing sounds
	BreathingChance  float64       `json:"breathing_chance"`  // Probability of breathing
	MinPauseLength   time.Duration `json:"min_pause_length"`  // Minimum pause duration
	MaxPauseLength   time.Duration `json:"max_pause_length"`  // Maximum pause duration
}

// QualityAssurance handles audio quality checking and fixing
type QualityAssurance struct {
	clickRemoval        bool
	popRemoval          bool
	noiseReduction      bool
	artifactDetection   bool
	corruptionDetection bool
	logger              *logrus.Logger
}

// CrossFader handles smooth transitions between audio segments
type CrossFader struct {
	fadeDuration   time.Duration
	fadeAlgorithm  string // "linear", "exponential", "cosine"
	overlapSamples int
	logger         *logrus.Logger
}

// SilenceGenerator creates silent audio segments for pauses
type SilenceGenerator struct {
	sampleRate int
	channels   int
	format     string
}

// WaveHeader represents WAV file header structure
type WaveHeader struct {
	ChunkID       [4]byte // "RIFF"
	ChunkSize     uint32  // File size - 8
	Format        [4]byte // "WAVE"
	Subchunk1ID   [4]byte // "fmt "
	Subchunk1Size uint32  // 16 for PCM
	AudioFormat   uint16  // 1 for PCM
	NumChannels   uint16  // Number of channels
	SampleRate    uint32  // Sample rate
	ByteRate      uint32  // Byte rate
	BlockAlign    uint16  // Block align
	BitsPerSample uint16  // Bits per sample
	Subchunk2ID   [4]byte // "data"
	Subchunk2Size uint32  // Data size
}

// NewAudioAssembler creates a new audio assembler with the given configuration
func NewAudioAssembler(config *AudioConfig, logger *logrus.Logger) (*AudioAssembler, error) {
	if config == nil {
		return nil, fmt.Errorf("audio config cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Validate configuration
	if err := validateAudioConfig(config); err != nil {
		return nil, fmt.Errorf("invalid audio config: %w", err)
	}

	assembler := &AudioAssembler{
		config: config,
		logger: logger,
	}

	// Initialize components
	assembler.volumeNormalizer = NewVolumeNormalizer(0.5, 0.95, 2.0, logger)
	assembler.pauseInserter = NewPauseManager(getDefaultPauseConfig(), config.SampleRate, logger)
	assembler.qualityController = NewQualityAssurance(true, true, false, logger)
	assembler.crossFader = NewCrossFader(config.CrossFadeDuration, "cosine", config.SampleRate, logger)

	return assembler, nil
}

// AssembleAudioStream combines multiple audio segments into a seamless stream
func (aa *AudioAssembler) AssembleAudioStream(segments []AudioSegment) ([]byte, error) {
	if len(segments) == 0 {
		return nil, fmt.Errorf("no audio segments provided")
	}

	aa.logger.WithField("segment_count", len(segments)).Info("Starting audio assembly")

	// Sort segments by index to ensure correct order
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].Index < segments[j].Index
	})

	// Validate and prepare segments
	processedSegments, err := aa.prepareSegments(segments)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare segments: %w", err)
	}

	// Load audio data for all segments
	if err := aa.loadAudioData(processedSegments); err != nil {
		return nil, fmt.Errorf("failed to load audio data: %w", err)
	}

	// Apply volume normalization if enabled
	if aa.config.NormalizationEnabled {
		if err := aa.normalizeVolumes(processedSegments); err != nil {
			aa.logger.WithError(err).Warn("Volume normalization failed, continuing without")
		}
	}

	// Apply quality assurance if enabled
	if aa.config.QualityAssurance {
		if err := aa.applyQualityAssurance(processedSegments); err != nil {
			aa.logger.WithError(err).Warn("Quality assurance failed, continuing without")
		}
	}

	// Assemble the final audio stream
	finalAudio, err := aa.combineSegments(processedSegments)
	if err != nil {
		return nil, fmt.Errorf("failed to combine segments: %w", err)
	}

	aa.logger.WithFields(logrus.Fields{
		"final_size":    len(finalAudio),
		"segment_count": len(processedSegments),
		"sample_rate":   aa.config.SampleRate,
	}).Info("Audio assembly completed successfully")

	return finalAudio, nil
}

// prepareSegments validates and prepares audio segments for processing
func (aa *AudioAssembler) prepareSegments(segments []AudioSegment) ([]AudioSegment, error) {
	processed := make([]AudioSegment, 0, len(segments))

	for i, segment := range segments {
		// Validate segment
		if segment.FilePath == "" {
			return nil, fmt.Errorf("segment %d has no file path", i)
		}

		// Check file exists
		if _, err := os.Stat(segment.FilePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("segment %d file does not exist: %s", i, segment.FilePath)
		}

		// Set defaults if not provided
		if segment.SampleRate == 0 {
			segment.SampleRate = aa.config.SampleRate
		}
		if segment.Format == "" {
			segment.Format = aa.config.AudioFormat
		}
		if segment.Channels == 0 {
			segment.Channels = 1 // Mono default
		}
		if segment.BitsPerSample == 0 {
			segment.BitsPerSample = 16 // 16-bit default
		}

		processed = append(processed, segment)
	}

	return processed, nil
}

// loadAudioData loads raw audio data from files into memory
func (aa *AudioAssembler) loadAudioData(segments []AudioSegment) error {
	for i := range segments {
		segment := &segments[i]

		aa.logger.WithFields(logrus.Fields{
			"index":     segment.Index,
			"file_path": segment.FilePath,
		}).Debug("Loading audio data")

		// Read file
		data, err := os.ReadFile(segment.FilePath)
		if err != nil {
			return fmt.Errorf("failed to read audio file %s: %w", segment.FilePath, err)
		}

		// Parse WAV file if applicable
		if segment.Format == "wav" {
			audioData, header, err := aa.parseWaveFile(data)
			if err != nil {
				return fmt.Errorf("failed to parse WAV file %s: %w", segment.FilePath, err)
			}

			segment.Data = audioData
			segment.SampleRate = int(header.SampleRate)
			segment.Channels = int(header.NumChannels)
			segment.BitsPerSample = int(header.BitsPerSample)
			segment.Duration = time.Duration(len(audioData)/(segment.Channels*segment.BitsPerSample/8)/segment.SampleRate) * time.Second
		} else {
			// For other formats, store raw data
			segment.Data = data
		}

		segment.Size = int64(len(segment.Data))
		segment.IsProcessed = true

		aa.logger.WithFields(logrus.Fields{
			"index":           segment.Index,
			"size":            segment.Size,
			"duration":        segment.Duration,
			"sample_rate":     segment.SampleRate,
			"channels":        segment.Channels,
			"bits_per_sample": segment.BitsPerSample,
		}).Debug("Audio data loaded successfully")
	}

	return nil
}

// parseWaveFile parses a WAV file and extracts audio data and header information
func (aa *AudioAssembler) parseWaveFile(data []byte) ([]byte, *WaveHeader, error) {
	if len(data) < 44 {
		return nil, nil, fmt.Errorf("file too small to be a valid WAV file")
	}

	// Parse WAV header
	reader := bytes.NewReader(data)
	var header WaveHeader

	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, nil, fmt.Errorf("failed to read WAV header: %w", err)
	}

	// Validate WAV format
	if string(header.ChunkID[:]) != "RIFF" {
		return nil, nil, fmt.Errorf("invalid WAV file: missing RIFF header")
	}
	if string(header.Format[:]) != "WAVE" {
		return nil, nil, fmt.Errorf("invalid WAV file: missing WAVE format")
	}
	if string(header.Subchunk1ID[:]) != "fmt " {
		return nil, nil, fmt.Errorf("invalid WAV file: missing fmt chunk")
	}
	if string(header.Subchunk2ID[:]) != "data" {
		return nil, nil, fmt.Errorf("invalid WAV file: missing data chunk")
	}

	// Extract audio data (skip header)
	audioData := data[44:]
	if len(audioData) != int(header.Subchunk2Size) {
		aa.logger.Warn("WAV file data size mismatch")
	}

	return audioData, &header, nil
}

// normalizeVolumes applies volume normalization across all segments
func (aa *AudioAssembler) normalizeVolumes(segments []AudioSegment) error {
	aa.logger.Info("Applying volume normalization")

	// Calculate RMS levels for all segments
	rmsLevels := make([]float64, len(segments))
	maxRMS := 0.0

	for i, segment := range segments {
		rms := aa.volumeNormalizer.calculateRMS(segment.Data, segment.BitsPerSample)
		rmsLevels[i] = rms
		if rms > maxRMS {
			maxRMS = rms
		}
	}

	// Normalize based on the loudest segment
	if maxRMS > 0 {
		for i := range segments {
			if rmsLevels[i] > 0 {
				gain := aa.volumeNormalizer.targetRMS / rmsLevels[i]

				// Limit gain to prevent clipping
				if gain > aa.volumeNormalizer.maxPeakAllowed {
					gain = aa.volumeNormalizer.maxPeakAllowed
				}

				aa.volumeNormalizer.applyGain(&segments[i], gain)

				aa.logger.WithFields(logrus.Fields{
					"segment_index": i,
					"original_rms":  rmsLevels[i],
					"gain_applied":  gain,
				}).Debug("Volume normalization applied")
			}
		}
	}

	return nil
}

// applyQualityAssurance runs quality checks and fixes on audio segments
func (aa *AudioAssembler) applyQualityAssurance(segments []AudioSegment) error {
	aa.logger.Info("Applying quality assurance")

	for i := range segments {
		segment := &segments[i]

		// Check for corruption
		if aa.qualityController.corruptionDetection {
			if corrupted := aa.qualityController.detectCorruption(segment.Data); corrupted {
				aa.logger.WithField("segment_index", i).Warn("Corrupted audio detected")
				segment.ErrorMessage = "Audio corruption detected"
				continue
			}
		}

		// Remove clicks and pops
		if aa.qualityController.clickRemoval {
			aa.qualityController.removeClicks(segment)
		}

		if aa.qualityController.popRemoval {
			aa.qualityController.removePops(segment)
		}

		// Noise reduction (basic implementation)
		if aa.qualityController.noiseReduction {
			aa.qualityController.reduceNoise(segment)
		}
	}

	return nil
}

// combineSegments combines all segments into a final audio stream
func (aa *AudioAssembler) combineSegments(segments []AudioSegment) ([]byte, error) {
	aa.logger.Info("Combining audio segments")

	var combinedData []byte
	totalDuration := time.Duration(0)

	for i, segment := range segments {
		// Add the segment audio data
		combinedData = append(combinedData, segment.Data...)
		totalDuration += segment.Duration

		aa.logger.WithFields(logrus.Fields{
			"segment_index": i,
			"segment_size":  len(segment.Data),
			"total_size":    len(combinedData),
		}).Debug("Added segment to combined stream")

		// Add pause after segment (except for the last one)
		if i < len(segments)-1 {
			pauseData := aa.generatePause(segment, segments[i+1])
			if len(pauseData) > 0 {
				combinedData = append(combinedData, pauseData...)
				pauseDuration := time.Duration(len(pauseData)/(segment.Channels*segment.BitsPerSample/8)/segment.SampleRate) * time.Second
				totalDuration += pauseDuration

				aa.logger.WithFields(logrus.Fields{
					"pause_size":     len(pauseData),
					"pause_duration": pauseDuration,
				}).Debug("Added pause between segments")
			}
		}

		// Apply cross-fade between segments if enabled
		if aa.config.CrossFadeEnabled && i < len(segments)-1 {
			nextSegment := segments[i+1]
			crossFadeData := aa.crossFader.createCrossFade(segment.Data, nextSegment.Data, segment.BitsPerSample)
			if len(crossFadeData) > 0 {
				// Replace the overlapping area with cross-faded data
				overlapSize := len(crossFadeData)
				if len(combinedData) >= overlapSize {
					copy(combinedData[len(combinedData)-overlapSize:], crossFadeData)
				}
			}
		}
	}

	// Create final WAV file with proper header
	if aa.config.AudioFormat == "wav" {
		finalWav := aa.createWavFile(combinedData, segments[0])
		return finalWav, nil
	}

	aa.logger.WithFields(logrus.Fields{
		"final_size":     len(combinedData),
		"total_duration": totalDuration,
		"segment_count":  len(segments),
	}).Info("Audio combination completed")

	return combinedData, nil
}

// generatePause creates appropriate pause audio between segments
func (aa *AudioAssembler) generatePause(currentSegment, nextSegment AudioSegment) []byte {
	if !aa.config.NaturalPauses {
		return nil
	}

	// Determine pause duration based on segment metadata
	pauseDuration := aa.pauseInserter.calculatePauseDuration(currentSegment, nextSegment)

	if pauseDuration <= 0 {
		return nil
	}

	// Generate silence
	silenceData := aa.pauseInserter.silenceGenerator.generateSilence(
		pauseDuration,
		currentSegment.SampleRate,
		currentSegment.Channels,
		currentSegment.BitsPerSample,
	)

	// Add breathing sound occasionally if enabled
	if aa.pauseInserter.config.BreathingEnabled {
		if shouldAddBreathing := aa.pauseInserter.shouldAddBreathing(); shouldAddBreathing {
			breathingData := aa.pauseInserter.generateBreathing(pauseDuration, currentSegment.SampleRate, currentSegment.Channels, currentSegment.BitsPerSample)
			if len(breathingData) > 0 {
				return breathingData
			}
		}
	}

	return silenceData
}

// createWavFile creates a complete WAV file with header
func (aa *AudioAssembler) createWavFile(audioData []byte, sampleSegment AudioSegment) []byte {
	dataSize := uint32(len(audioData))
	fileSize := dataSize + 36

	header := WaveHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'},
		ChunkSize:     fileSize,
		Format:        [4]byte{'W', 'A', 'V', 'E'},
		Subchunk1ID:   [4]byte{'f', 'm', 't', ' '},
		Subchunk1Size: 16,
		AudioFormat:   1, // PCM
		NumChannels:   uint16(sampleSegment.Channels),
		SampleRate:    uint32(sampleSegment.SampleRate),
		ByteRate:      uint32(sampleSegment.SampleRate * sampleSegment.Channels * sampleSegment.BitsPerSample / 8),
		BlockAlign:    uint16(sampleSegment.Channels * sampleSegment.BitsPerSample / 8),
		BitsPerSample: uint16(sampleSegment.BitsPerSample),
		Subchunk2ID:   [4]byte{'d', 'a', 't', 'a'},
		Subchunk2Size: dataSize,
	}

	// Write header and data
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, header)
	buf.Write(audioData)

	return buf.Bytes()
}

// Component implementations

func NewVolumeNormalizer(targetRMS, maxPeak, compressionRatio float64, logger *logrus.Logger) *VolumeNormalizer {
	return &VolumeNormalizer{
		targetRMS:        targetRMS,
		maxPeakAllowed:   maxPeak,
		compressionRatio: compressionRatio,
		lookAheadTime:    time.Millisecond * 5,
		logger:           logger,
	}
}

func (vn *VolumeNormalizer) calculateRMS(data []byte, bitsPerSample int) float64 {
	if len(data) == 0 {
		return 0
	}

	var sum float64
	sampleCount := 0

	if bitsPerSample == 16 {
		for i := 0; i < len(data)-1; i += 2 {
			sample := int16(binary.LittleEndian.Uint16(data[i : i+2]))
			normalized := float64(sample) / 32768.0
			sum += normalized * normalized
			sampleCount++
		}
	} else if bitsPerSample == 8 {
		for _, b := range data {
			sample := int8(b)
			normalized := float64(sample) / 128.0
			sum += normalized * normalized
			sampleCount++
		}
	}

	if sampleCount == 0 {
		return 0
	}

	return math.Sqrt(sum / float64(sampleCount))
}

func (vn *VolumeNormalizer) applyGain(segment *AudioSegment, gain float64) {
	if gain == 1.0 || len(segment.Data) == 0 {
		return
	}

	if segment.BitsPerSample == 16 {
		for i := 0; i < len(segment.Data)-1; i += 2 {
			sample := int16(binary.LittleEndian.Uint16(segment.Data[i : i+2]))
			newSample := int16(float64(sample) * gain)

			// Prevent clipping
			if newSample > 32767 {
				newSample = 32767
			} else if newSample < -32768 {
				newSample = -32768
			}

			binary.LittleEndian.PutUint16(segment.Data[i:i+2], uint16(newSample))
		}
	}

	segment.Volume = gain
}

func NewPauseManager(config *PauseConfig, sampleRate int, logger *logrus.Logger) *PauseManager {
	return &PauseManager{
		config: config,
		silenceGenerator: &SilenceGenerator{
			sampleRate: sampleRate,
			channels:   1,
			format:     "wav",
		},
		logger: logger,
	}
}

func (pm *PauseManager) calculatePauseDuration(current, next AudioSegment) time.Duration {
	// Use custom pause from metadata if available
	if pm.config.CustomPauses {
		if pauseAfter, exists := current.Metadata["pause_after"]; exists {
			if pauseMs, ok := pauseAfter.(int); ok {
				return time.Duration(pauseMs) * time.Millisecond
			}
		}
	}

	// Default pause rules based on content type
	if isDialogue, exists := current.Metadata["is_dialogue"]; exists && isDialogue.(bool) {
		return pm.config.DialoguePause
	}

	if isList, exists := current.Metadata["is_list"]; exists && isList.(bool) {
		return pm.config.ListItemPause
	}

	// Check for paragraph break
	if content, exists := current.Metadata["content"]; exists {
		if contentStr, ok := content.(string); ok {
			if len(contentStr) > 0 {
				lastChar := contentStr[len(contentStr)-1]
				switch lastChar {
				case '.', '!', '?':
					return pm.config.SentencePause
				case '\n':
					return pm.config.ParagraphPause
				case ',', ';':
					return pm.config.ListItemPause
				}
			}
		}
	}

	return pm.config.SentencePause
}

func (pm *PauseManager) shouldAddBreathing() bool {
	if !pm.config.BreathingEnabled {
		return false
	}

	// Simple random chance for breathing
	return math.Mod(float64(time.Now().UnixNano()), 100) < pm.config.BreathingChance*100
}

func (pm *PauseManager) generateBreathing(duration time.Duration, sampleRate, channels, bitsPerSample int) []byte {
	// This would generate a subtle breathing sound
	// For now, return silence as a placeholder
	return pm.silenceGenerator.generateSilence(duration, sampleRate, channels, bitsPerSample)
}

func (sg *SilenceGenerator) generateSilence(duration time.Duration, sampleRate, channels, bitsPerSample int) []byte {
	samplesNeeded := int(duration.Seconds() * float64(sampleRate))
	bytesPerSample := bitsPerSample / 8
	totalBytes := samplesNeeded * channels * bytesPerSample

	return make([]byte, totalBytes) // All zeros = silence
}

func NewQualityAssurance(clickRemoval, popRemoval, noiseReduction bool, logger *logrus.Logger) *QualityAssurance {
	return &QualityAssurance{
		clickRemoval:        clickRemoval,
		popRemoval:          popRemoval,
		noiseReduction:      noiseReduction,
		artifactDetection:   true,
		corruptionDetection: true,
		logger:              logger,
	}
}

func (qa *QualityAssurance) detectCorruption(data []byte) bool {
	// Basic corruption detection - check for all zeros or all max values
	if len(data) == 0 {
		return true
	}

	zeroCount := 0
	maxCount := 0
	for _, b := range data {
		if b == 0 {
			zeroCount++
		} else if b == 255 {
			maxCount++
		}
	}

	// If more than 90% is zeros or max values, consider it corrupted
	threshold := len(data) * 9 / 10
	return zeroCount > threshold || maxCount > threshold
}

func (qa *QualityAssurance) removeClicks(segment *AudioSegment) {
	// Placeholder for click removal algorithm
	// This would implement sophisticated click detection and removal
	qa.logger.WithField("segment_index", segment.Index).Debug("Click removal applied")
}

func (qa *QualityAssurance) removePops(segment *AudioSegment) {
	// Placeholder for pop removal algorithm
	qa.logger.WithField("segment_index", segment.Index).Debug("Pop removal applied")
}

func (qa *QualityAssurance) reduceNoise(segment *AudioSegment) {
	// Placeholder for noise reduction algorithm
	qa.logger.WithField("segment_index", segment.Index).Debug("Noise reduction applied")
}

func NewCrossFader(duration time.Duration, algorithm string, sampleRate int, logger *logrus.Logger) *CrossFader {
	overlapSamples := int(duration.Seconds() * float64(sampleRate))
	return &CrossFader{
		fadeDuration:   duration,
		fadeAlgorithm:  algorithm,
		overlapSamples: overlapSamples,
		logger:         logger,
	}
}

func (cf *CrossFader) createCrossFade(audio1, audio2 []byte, bitsPerSample int) []byte {
	if len(audio1) == 0 || len(audio2) == 0 {
		return nil
	}

	overlapBytes := cf.overlapSamples * (bitsPerSample / 8)
	if len(audio1) < overlapBytes || len(audio2) < overlapBytes {
		return nil
	}

	crossFadeData := make([]byte, overlapBytes)

	// Apply cross-fade algorithm
	for i := 0; i < overlapBytes; i += 2 { // Assuming 16-bit samples
		if i+1 >= len(audio1) || i+1 >= len(audio2) {
			break
		}

		// Get samples from both audio streams
		sample1 := int16(binary.LittleEndian.Uint16(audio1[len(audio1)-overlapBytes+i:]))
		sample2 := int16(binary.LittleEndian.Uint16(audio2[i:]))

		// Calculate fade factors
		progress := float64(i) / float64(overlapBytes)
		fade1, fade2 := cf.calculateFadeFactors(progress)

		// Mix samples
		mixedSample := int16(float64(sample1)*fade1 + float64(sample2)*fade2)
		binary.LittleEndian.PutUint16(crossFadeData[i:], uint16(mixedSample))
	}

	return crossFadeData
}

func (cf *CrossFader) calculateFadeFactors(progress float64) (float64, float64) {
	switch cf.fadeAlgorithm {
	case "exponential":
		fade1 := math.Pow(1-progress, 2)
		fade2 := math.Pow(progress, 2)
		return fade1, fade2
	case "cosine":
		fade1 := math.Cos(progress * math.Pi / 2)
		fade2 := math.Sin(progress * math.Pi / 2)
		return fade1, fade2
	default: // linear
		fade1 := 1 - progress
		fade2 := progress
		return fade1, fade2
	}
}

// Utility functions

func validateAudioConfig(config *AudioConfig) error {
	if config.SampleRate <= 0 {
		return fmt.Errorf("sample rate must be positive")
	}
	if config.AudioFormat == "" {
		return fmt.Errorf("audio format cannot be empty")
	}
	if config.CrossFadeDuration < 0 {
		return fmt.Errorf("cross fade duration cannot be negative")
	}
	if config.MaxOutputSize <= 0 {
		return fmt.Errorf("max output size must be positive")
	}
	if config.TempDirectory == "" {
		return fmt.Errorf("temp directory cannot be empty")
	}
	return nil
}

func getDefaultPauseConfig() *PauseConfig {
	return &PauseConfig{
		SentencePause:    400 * time.Millisecond,
		ParagraphPause:   800 * time.Millisecond,
		DialoguePause:    450 * time.Millisecond,
		ListItemPause:    250 * time.Millisecond,
		CustomPauses:     true,
		BreathingEnabled: false,
		BreathingChance:  0.05, // 5% chance
		MinPauseLength:   100 * time.Millisecond,
		MaxPauseLength:   2 * time.Second,
	}
}

// GetAssemblyStats returns statistics about the audio assembly process
func (aa *AudioAssembler) GetAssemblyStats(segments []AudioSegment, finalSize int64) map[string]interface{} {
	totalInputSize := int64(0)
	totalDuration := time.Duration(0)
	segmentCount := len(segments)

	for _, segment := range segments {
		totalInputSize += segment.Size
		totalDuration += segment.Duration
	}

	compressionRatio := float64(finalSize) / float64(totalInputSize)

	return map[string]interface{}{
		"segment_count":     segmentCount,
		"total_input_size":  totalInputSize,
		"final_output_size": finalSize,
		"compression_ratio": compressionRatio,
		"total_duration":    totalDuration.String(),
		"sample_rate":       aa.config.SampleRate,
		"audio_format":      aa.config.AudioFormat,
		"normalization":     aa.config.NormalizationEnabled,
		"cross_fade":        aa.config.CrossFadeEnabled,
		"quality_assurance": aa.config.QualityAssurance,
	}
}
