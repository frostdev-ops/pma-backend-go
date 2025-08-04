package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// Service manages text-to-speech and speech-to-text operations
type Service struct {
	config          *config.SpeechConfig
	logger          *logrus.Logger
	httpClient      *http.Client
	daemonHealthy   bool
	healthCheckMux  sync.RWMutex
	lastHealthCheck time.Time
}

// TTSDaemonRequest represents a request to the TTS daemon
type TTSDaemonRequest struct {
	Text         string  `json:"text"`
	Voice        string  `json:"voice,omitempty"`
	Speaker      string  `json:"speaker,omitempty"`
	SpeakerIndex *int    `json:"speaker_index,omitempty"`
	Language     string  `json:"language,omitempty"`
	Speed        float32 `json:"speed,omitempty"`
	Emotion      string  `json:"emotion,omitempty"`
	OutputFormat string  `json:"output_format,omitempty"`
}

// TTSDaemonResponse represents a response from the TTS daemon
type TTSDaemonResponse struct {
	Success        bool    `json:"success"`
	AudioFile      string  `json:"audio_file,omitempty"`
	Duration       float64 `json:"duration,omitempty"`
	ProcessingTime float64 `json:"processing_time,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// TTSDaemonHealthResponse represents a health check response from the TTS daemon
type TTSDaemonHealthResponse struct {
	Status       string                 `json:"status"`
	Uptime       float64                `json:"uptime"`
	ModelsLoaded int                    `json:"models_loaded"`
	AudioFiles   int                    `json:"audio_files"`
	MemoryUsage  map[string]interface{} `json:"memory_usage"`
	Version      string                 `json:"version"`
}

// TTSRequest represents a text-to-speech request
type TTSRequest struct {
	Text         string  `json:"text" validate:"required,max=5000"`
	Voice        string  `json:"voice,omitempty"`
	Speed        float32 `json:"speed,omitempty"`
	Language     string  `json:"language,omitempty"`
	Speaker      string  `json:"speaker,omitempty"`
	SpeakerIndex *int    `json:"speaker_index,omitempty"`
	SpeakerWav   string  `json:"speaker_wav,omitempty"`
	OutputFile   string  `json:"output_file,omitempty"`
	PlayDirectly bool    `json:"play_directly,omitempty"`
}

// TTSResponse represents a text-to-speech response
type TTSResponse struct {
	Success       bool    `json:"success"`
	OutputFile    string  `json:"output_file,omitempty"`
	AudioDuration float64 `json:"audio_duration,omitempty"`
	Error         string  `json:"error,omitempty"`
}

// STTRequest represents a speech-to-text request
type STTRequest struct {
	AudioFile        string `json:"audio_file" validate:"required"`
	Model            string `json:"model,omitempty"`
	Language         string `json:"language,omitempty"`
	UseAutocorrect   bool   `json:"use_autocorrect,omitempty"`
	SilenceThreshold int    `json:"silence_threshold,omitempty"`
}

// STTResponse represents a speech-to-text response
type STTResponse struct {
	Success               bool    `json:"success"`
	Transcription         string  `json:"transcription,omitempty"`
	Confidence            float64 `json:"confidence,omitempty"`
	Language              string  `json:"language,omitempty"`
	Model                 string  `json:"model,omitempty"`
	Autocorrected         string  `json:"autocorrected,omitempty"`
	AutocorrectConfidence float64 `json:"autocorrect_confidence,omitempty"`
	Error                 string  `json:"error,omitempty"`
}

// NewService creates a new speech service
func NewService(cfg *config.SpeechConfig, logger *logrus.Logger) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("speech config is required")
	}

	// Create HTTP client with timeout for daemon communication
	var daemonTimeout time.Duration = 30 * time.Second
	if cfg.TTS.DaemonTimeout != "" {
		if parsedTimeout, err := time.ParseDuration(cfg.TTS.DaemonTimeout); err == nil {
			daemonTimeout = parsedTimeout
		}
	}

	service := &Service{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: daemonTimeout,
		},
		daemonHealthy: false,
	}

	// Ensure output directories exist
	if err := service.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create output directories: %w", err)
	}

	// Start daemon health checking if enabled
	if cfg.TTS.UseDaemon {
		go service.startDaemonHealthCheck()
	}

	return service, nil
}

// ensureDirectories creates necessary output directories
func (s *Service) ensureDirectories() error {
	dirs := []string{
		s.config.TTS.OutputDir,
		s.config.STT.OutputDir,
	}

	for _, dir := range dirs {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	return nil
}

// Enabled returns whether the speech service is enabled
func (s *Service) Enabled() bool {
	return s.config.Enabled
}

// TTSEnabled returns whether TTS is enabled
func (s *Service) TTSEnabled() bool {
	return s.config.Enabled && s.config.TTS.Enabled
}

// STTEnabled returns whether STT is enabled
func (s *Service) STTEnabled() bool {
	return s.config.Enabled && s.config.STT.Enabled
}

// TextToSpeech converts text to speech using TTS daemon or fallback to Python script
func (s *Service) TextToSpeech(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	if !s.TTSEnabled() {
		return nil, fmt.Errorf("TTS service is disabled")
	}

	// Validate request
	if err := s.validateTTSRequest(req); err != nil {
		return nil, fmt.Errorf("invalid TTS request: %w", err)
	}

	// Try TTS daemon first if enabled and healthy
	if s.config.TTS.UseDaemon && s.isDaemonHealthy() {
		s.logger.Debug("Using TTS daemon for synthesis")
		response, err := s.textToSpeechDaemon(ctx, req)
		if err == nil {
			return response, nil
		}

		s.logger.WithError(err).Warn("TTS daemon failed, falling back to subprocess")
		// Mark daemon as unhealthy temporarily
		s.setDaemonHealthy(false)
	}

	// Fallback to subprocess execution
	s.logger.Debug("Using Python subprocess for TTS synthesis")
	return s.textToSpeechSubprocess(ctx, req)
}

// textToSpeechDaemon uses the TTS daemon HTTP API
func (s *Service) textToSpeechDaemon(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	// Build daemon request
	// Use Voice if provided, otherwise fallback to Speaker (they're aliases)
	voice := req.Voice
	if voice == "" {
		voice = req.Speaker
	}

	daemonReq := TTSDaemonRequest{
		Text:         req.Text,
		Voice:        voice,
		Speaker:      req.Speaker,
		SpeakerIndex: req.SpeakerIndex,
		Language:     req.Language,
		Speed:        req.Speed,
		OutputFormat: "wav",
	}

	// Marshal request
	reqBody, err := json.Marshal(daemonReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal daemon request: %w", err)
	}

	// Create HTTP request
	url := s.config.TTS.DaemonURL + "/tts"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("TTS daemon request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read daemon response: %w", err)
	}

	// Parse response
	var daemonResp TTSDaemonResponse
	if err := json.Unmarshal(respBody, &daemonResp); err != nil {
		return nil, fmt.Errorf("failed to parse daemon response: %w", err)
	}

	// Check for daemon error
	if !daemonResp.Success {
		return &TTSResponse{
			Success: false,
			Error:   daemonResp.Error,
		}, nil
	}

	// Download audio file from daemon
	audioURL := s.config.TTS.DaemonURL + "/audio/" + daemonResp.AudioFile
	outputFile := filepath.Join(s.config.TTS.OutputDir, daemonResp.AudioFile)

	if err := s.downloadAudioFile(ctx, audioURL, outputFile); err != nil {
		return nil, fmt.Errorf("failed to download audio file: %w", err)
	}

	return &TTSResponse{
		Success:       true,
		OutputFile:    outputFile,
		AudioDuration: daemonResp.Duration,
	}, nil
}

// textToSpeechSubprocess uses the original Python subprocess approach
func (s *Service) textToSpeechSubprocess(ctx context.Context, req *TTSRequest) (*TTSResponse, error) {
	// Prepare output file if not specified
	if req.OutputFile == "" {
		timestamp := time.Now().Format("20060102_150405")
		req.OutputFile = filepath.Join(s.config.TTS.OutputDir, fmt.Sprintf("tts_%s.wav", timestamp))
	}

	// Build command arguments
	args := s.buildTTSArgs(req)

	// Execute Python script
	response, err := s.executeTTSScript(ctx, args)
	if err != nil {
		s.logger.WithError(err).Error("TTS script execution failed")
		return &TTSResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return response, nil
}

// downloadAudioFile downloads an audio file from the daemon
func (s *Service) downloadAudioFile(ctx context.Context, url, outputFile string) error {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy audio data
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	return nil
}

// startDaemonHealthCheck starts periodic health checking of the TTS daemon
func (s *Service) startDaemonHealthCheck() {
	healthCheckInterval := 60 * time.Second
	if s.config.TTS.DaemonHealthCheckInterval != "" {
		if parsedInterval, err := time.ParseDuration(s.config.TTS.DaemonHealthCheckInterval); err == nil {
			healthCheckInterval = parsedInterval
		}
	}

	// Initial health check
	s.checkDaemonHealth()

	// Periodic health checks
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkDaemonHealth()
		}
	}
}

// checkDaemonHealth performs a health check on the TTS daemon
func (s *Service) checkDaemonHealth() {
	healthy := false

	defer func() {
		s.setDaemonHealthy(healthy)
		s.healthCheckMux.Lock()
		s.lastHealthCheck = time.Now()
		s.healthCheckMux.Unlock()
	}()

	if s.config.TTS.DaemonURL == "" {
		return
	}

	// Create health check request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := s.config.TTS.DaemonURL + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.logger.WithError(err).Debug("Failed to create health check request")
		return
	}

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.WithError(err).Debug("TTS daemon health check failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.WithField("status", resp.StatusCode).Debug("TTS daemon health check returned non-200 status")
		return
	}

	// Parse health response
	var healthResp TTSDaemonHealthResponse
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.WithError(err).Debug("Failed to read health check response")
		return
	}

	if err := json.Unmarshal(respBody, &healthResp); err != nil {
		s.logger.WithError(err).Debug("Failed to parse health check response")
		return
	}

	if healthResp.Status == "healthy" {
		healthy = true
		s.logger.WithFields(logrus.Fields{
			"uptime":        healthResp.Uptime,
			"models_loaded": healthResp.ModelsLoaded,
		}).Debug("TTS daemon is healthy")
	}
}

// isDaemonHealthy returns the current daemon health status
func (s *Service) isDaemonHealthy() bool {
	s.healthCheckMux.RLock()
	defer s.healthCheckMux.RUnlock()
	return s.daemonHealthy
}

// setDaemonHealthy sets the daemon health status
func (s *Service) setDaemonHealthy(healthy bool) {
	s.healthCheckMux.Lock()
	defer s.healthCheckMux.Unlock()

	if s.daemonHealthy != healthy {
		s.logger.WithField("healthy", healthy).Info("TTS daemon health status changed")
		s.daemonHealthy = healthy
	}
}

// SpeechToText converts speech to text using the Python STT script
func (s *Service) SpeechToText(ctx context.Context, req *STTRequest) (*STTResponse, error) {
	if !s.STTEnabled() {
		return nil, fmt.Errorf("STT service is disabled")
	}

	// Validate request
	if err := s.validateSTTRequest(req); err != nil {
		return nil, fmt.Errorf("invalid STT request: %w", err)
	}

	// Build command arguments
	args := s.buildSTTArgs(req)

	// Execute Python script
	response, err := s.executeSTTScript(ctx, args)
	if err != nil {
		s.logger.WithError(err).Error("STT script execution failed")
		return &STTResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return response, nil
}

// RecordAndTranscribe records audio and transcribes it to text
func (s *Service) RecordAndTranscribe(ctx context.Context, model, language string, useAutocorrect bool) (*STTResponse, error) {
	if !s.STTEnabled() {
		return nil, fmt.Errorf("STT service is disabled")
	}

	// Build arguments for recording
	args := []string{
		s.config.STT.PythonScriptPath,
	}

	if model != "" {
		args = append(args, "--model", model)
	} else {
		args = append(args, "--model", s.config.STT.DefaultModel)
	}

	if language != "" {
		args = append(args, "--language", language)
	} else {
		args = append(args, "--language", s.config.STT.Language)
	}

	if useAutocorrect {
		args = append(args, "--ac")
	}

	args = append(args, "--silence-threshold", fmt.Sprintf("%d", s.config.STT.SilenceThreshold))

	// Execute Python script for recording and transcription
	response, err := s.executeSTTScript(ctx, args)
	if err != nil {
		s.logger.WithError(err).Error("STT recording script execution failed")
		return &STTResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return response, nil
}

// validateTTSRequest validates a TTS request
func (s *Service) validateTTSRequest(req *TTSRequest) error {
	if req.Text == "" {
		return fmt.Errorf("text is required")
	}

	if len(req.Text) > s.config.TTS.MaxTextLength {
		return fmt.Errorf("text length exceeds maximum of %d characters", s.config.TTS.MaxTextLength)
	}

	return nil
}

// validateSTTRequest validates an STT request
func (s *Service) validateSTTRequest(req *STTRequest) error {
	if req.AudioFile == "" {
		return fmt.Errorf("audio file is required")
	}

	// Check if file exists
	if _, err := os.Stat(req.AudioFile); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s", req.AudioFile)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(req.AudioFile))
	validFormats := map[string]bool{
		".wav":  true,
		".mp3":  true,
		".m4a":  true,
		".ogg":  true,
		".webm": true, // Support browser WebM recording
		".mp4":  true, // Support MP4 audio
	}

	if !validFormats[ext] {
		return fmt.Errorf("unsupported audio format: %s", ext)
	}

	return nil
}

// buildTTSArgs builds command line arguments for the TTS script
func (s *Service) buildTTSArgs(req *TTSRequest) []string {
	args := []string{
		s.config.TTS.PythonScriptPath,
		"--text", req.Text,
		"--output", req.OutputFile,
	}

	// Add voice (Piper TTS uses voice instead of model)
	if req.Voice != "" {
		args = append(args, "--voice", req.Voice)
	} else if req.Speaker != "" {
		args = append(args, "--voice", req.Speaker)
	} else {
		args = append(args, "--voice", s.config.TTS.DefaultModel) // Fallback to config default
	}

	// Add device
	if s.config.TTS.Device != "" {
		args = append(args, "--device", s.config.TTS.Device)
	}

	// Add optional parameters
	if req.Speed > 0 {
		args = append(args, "--speed", fmt.Sprintf("%.2f", req.Speed))
	}

	if req.Language != "" {
		args = append(args, "--language", req.Language)
	}

	if req.Speaker != "" {
		args = append(args, "--speaker", req.Speaker)
	}

	if req.SpeakerWav != "" {
		args = append(args, "--speaker-wav", req.SpeakerWav)
	}

	// Force GPU if configured
	if s.config.TTS.ForceGPU {
		args = append(args, "--gpu")
	}

	// Don't play audio directly (we handle that separately)
	if !req.PlayDirectly {
		args = append(args, "--no-play")
	}

	return args
}

// buildSTTArgs builds command line arguments for the STT script
func (s *Service) buildSTTArgs(req *STTRequest) []string {
	args := []string{
		s.config.STT.PythonScriptPath,
	}

	// For file-based STT, we need to modify the script to accept file input
	// For now, assume the script supports a --file parameter
	if req.AudioFile != "" {
		args = append(args, "--file", req.AudioFile)
	}

	// Add model
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	} else {
		args = append(args, "--model", s.config.STT.DefaultModel)
	}

	// Add language
	if req.Language != "" {
		args = append(args, "--language", req.Language)
	} else {
		args = append(args, "--language", s.config.STT.Language)
	}

	// Add autocorrect
	if req.UseAutocorrect {
		args = append(args, "--ac")
	}

	// Add silence threshold
	threshold := req.SilenceThreshold
	if threshold == 0 {
		threshold = s.config.STT.SilenceThreshold
	}
	args = append(args, "--silence-threshold", fmt.Sprintf("%d", threshold))

	return args
}

// executeTTSScript executes the TTS Python script
func (s *Service) executeTTSScript(ctx context.Context, args []string) (*TTSResponse, error) {
	// Parse timeout
	timeout, err := time.ParseDuration(s.config.TTS.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute command using virtual environment Python
	pythonPath := "/opt/pma/speech/venv_speech/bin/python3"
	cmd := exec.CommandContext(execCtx, pythonPath, args...)

	s.logger.WithField("args", args).Debug("Executing TTS script")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("TTS script failed: %w, output: %s", err, string(output))
	}

	// Parse JSON response
	var response TTSResponse
	if err := json.Unmarshal(output, &response); err != nil {
		// If JSON parsing fails, assume it's a simple success case
		s.logger.WithField("output", string(output)).Debug("TTS script output (non-JSON)")

		// Check if output file exists
		outputFile := args[4] // Assuming --output is at position 4
		if _, err := os.Stat(outputFile); err == nil {
			return &TTSResponse{
				Success:    true,
				OutputFile: outputFile,
			}, nil
		}

		return nil, fmt.Errorf("failed to parse TTS script output: %w", err)
	}

	return &response, nil
}

// executeSTTScript executes the STT Python script with enhanced error handling
func (s *Service) executeSTTScript(ctx context.Context, args []string) (*STTResponse, error) {
	// Parse timeout
	timeout, err := time.ParseDuration(s.config.STT.Timeout)
	if err != nil {
		s.logger.WithError(err).Warn("Invalid STT timeout format, using default")
		timeout = 30 * time.Second
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Validate Python environment
	pythonPath := "/opt/pma/speech/venv_speech/bin/python3"
	if _, err := os.Stat(pythonPath); err != nil {
		s.logger.WithError(err).Error("Python environment not found")
		return nil, CategorizeError(fmt.Errorf("python environment not found: %w", err))
	}

	// Validate STT script
	if _, err := os.Stat(s.config.STT.PythonScriptPath); err != nil {
		s.logger.WithError(err).Error("STT script not found")
		return nil, CategorizeError(fmt.Errorf("STT script not found: %w", err))
	}

	// Execute command
	cmd := exec.CommandContext(execCtx, pythonPath, args...)
	
	s.logger.WithFields(logrus.Fields{
		"args":        args,
		"timeout":     timeout,
		"python_path": pythonPath,
	}).Info("Executing STT script")

	output, err := cmd.CombinedOutput()
	
	// Enhanced error handling
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":       err,
			"output":      string(output),
			"exit_code":   cmd.ProcessState.ExitCode(),
		}).Error("STT script execution failed")

		// Check for specific error conditions
		outputStr := string(output)
		
		// Check for timeout
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, CategorizeError(fmt.Errorf("STT processing timed out after %v", timeout))
		}
		
		// Parse Python script errors from output
		if strings.Contains(outputStr, "ModuleNotFoundError") || strings.Contains(outputStr, "ImportError") {
			return nil, CategorizeError(fmt.Errorf("missing Python dependencies: %s", outputStr))
		}
		
		if strings.Contains(outputStr, "No input device found") {
			return nil, CategorizeError(fmt.Errorf("no audio input devices available"))
		}
		
		if strings.Contains(outputStr, "Error opening audio stream") {
			return nil, CategorizeError(fmt.Errorf("failed to access audio device: %s", outputStr))
		}
		
		if strings.Contains(outputStr, "OutOfMemoryError") {
			return nil, CategorizeError(fmt.Errorf("insufficient memory for STT processing"))
		}

		// Generic execution error
		return nil, CategorizeError(fmt.Errorf("STT script execution failed: %w, output: %s", err, outputStr))
	}

	s.logger.WithField("output_length", len(output)).Debug("STT script completed successfully")

	// Parse JSON response
	var response STTResponse
	if err := json.Unmarshal(output, &response); err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":  err,
			"output": string(output),
		}).Error("Failed to parse STT script JSON output")
		
		// Try to extract any error from the output
		if strings.Contains(string(output), "error") {
			return nil, CategorizeError(fmt.Errorf("STT processing error: %s", string(output)))
		}
		
		return nil, CategorizeError(fmt.Errorf("failed to parse STT script output as JSON: %w", err))
	}

	// Check if the response indicates an error
	if !response.Success && response.Error != "" {
		s.logger.WithField("stt_error", response.Error).Error("STT script reported error")
		return nil, CategorizeError(fmt.Errorf("STT script error: %s", response.Error))
	}

	return &response, nil
}

// GetTTSModels returns available TTS models
func (s *Service) GetTTSModels(ctx context.Context) ([]string, error) {
	if !s.TTSEnabled() {
		return nil, fmt.Errorf("TTS service is disabled")
	}

	// Try daemon first if available and healthy
	if s.config.TTS.UseDaemon && s.isDaemonHealthy() {
		models, err := s.getTTSModelsFromDaemon(ctx)
		if err == nil {
			return models, nil
		}
		s.logger.WithError(err).Warn("Failed to get models from daemon, falling back to subprocess")
	}

	// Fallback to subprocess
	return s.getTTSModelsFromSubprocess(ctx)
}

// getTTSModelsFromDaemon gets models from the TTS daemon
func (s *Service) getTTSModelsFromDaemon(ctx context.Context) ([]string, error) {
	url := s.config.TTS.DaemonURL + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon models request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	// Parse response - daemon returns array of model objects directly
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read models response: %w", err)
	}

	// Try parsing as direct array (daemon format)
	var modelObjects []struct {
		Name     string  `json:"name"`
		Language string  `json:"language"`
		Type     string  `json:"type"`
		Loaded   bool    `json:"loaded"`
		SizeMB   float64 `json:"size_mb"`
	}

	if err := json.Unmarshal(respBody, &modelObjects); err != nil {
		// Fallback: try parsing as wrapped format for compatibility
		var modelsResp struct {
			AvailableModels []string `json:"available_models"`
			Error           string   `json:"error,omitempty"`
		}

		if err := json.Unmarshal(respBody, &modelsResp); err != nil {
			return nil, fmt.Errorf("failed to parse models response: %w", err)
		}

		if modelsResp.Error != "" {
			return nil, fmt.Errorf("daemon error: %s", modelsResp.Error)
		}

		return modelsResp.AvailableModels, nil
	}

	// Extract model names from daemon response
	var modelNames []string
	for _, model := range modelObjects {
		modelNames = append(modelNames, model.Name)
	}

	return modelNames, nil
}

// getTTSModelsFromSubprocess gets models using Python subprocess
func (s *Service) getTTSModelsFromSubprocess(ctx context.Context) ([]string, error) {
	args := []string{
		s.config.TTS.PythonScriptPath,
		"--list-models",
	}

	// Use the virtual environment Python interpreter
	pythonPath := "/opt/pma/speech/venv_speech/bin/python3"
	cmd := exec.CommandContext(ctx, pythonPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTS models: %w", err)
	}

	// Parse the output to extract model names
	lines := strings.Split(string(output), "\n")
	var models []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "tts_models/") {
			models = append(models, line)
		}
	}

	return models, nil
}

// GetVoices returns available TTS voices
func (s *Service) GetVoices(ctx context.Context) ([]map[string]interface{}, error) {
	if !s.TTSEnabled() {
		return nil, fmt.Errorf("TTS service is disabled")
	}

	// Try daemon first if available and healthy
	if s.config.TTS.UseDaemon && s.isDaemonHealthy() {
		voices, err := s.getVoicesFromDaemon(ctx)
		if err == nil {
			return voices, nil
		}
		s.logger.WithError(err).Warn("Failed to get voices from daemon, falling back to default voices")
	}

	// Fallback to default voices
	return s.getDefaultVoices(), nil
}

// getVoicesFromDaemon gets voices from the TTS daemon
func (s *Service) getVoicesFromDaemon(ctx context.Context) ([]map[string]interface{}, error) {
	url := s.config.TTS.DaemonURL + "/voices"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create voices request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon voices request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	// Parse response - daemon returns array of voice objects directly
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read voices response: %w", err)
	}

	var voiceObjects []map[string]interface{}
	if err := json.Unmarshal(respBody, &voiceObjects); err != nil {
		return nil, fmt.Errorf("failed to parse voices response: %w", err)
	}

	return voiceObjects, nil
}

// getDefaultVoices returns fallback voice options
func (s *Service) getDefaultVoices() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "default",
			"model":       "espeak",
			"language":    "en",
			"gender":      "neutral",
			"description": "Default system voice",
		},
		{
			"name":        "male",
			"model":       "espeak",
			"language":    "en",
			"gender":      "male",
			"description": "Male system voice",
		},
		{
			"name":        "female",
			"model":       "espeak",
			"language":    "en",
			"gender":      "female",
			"description": "Female system voice",
		},
	}
}

// GetVoiceSpeakers returns speakers for a specific multi-speaker voice
func (s *Service) GetVoiceSpeakers(ctx context.Context, voiceName string) (map[string]interface{}, error) {
	if !s.TTSEnabled() {
		return nil, fmt.Errorf("TTS service is disabled")
	}

	// Try daemon first if available and healthy
	if s.config.TTS.UseDaemon && s.isDaemonHealthy() {
		speakers, err := s.getVoiceSpeakersFromDaemon(ctx, voiceName)
		if err == nil {
			return speakers, nil
		}
		s.logger.WithError(err).Warnf("Failed to get speakers from daemon for voice %s", voiceName)
	}

	// Fallback: voice not found or not multi-speaker
	return nil, fmt.Errorf("voice '%s' not found or not a multi-speaker model", voiceName)
}

// getVoiceSpeakersFromDaemon gets speakers for a voice from the TTS daemon
func (s *Service) getVoiceSpeakersFromDaemon(ctx context.Context, voiceName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/voices/%s/speakers", s.config.TTS.DaemonURL, voiceName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create speakers request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon speakers request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("voice '%s' not found", voiceName)
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("voice '%s' is not a multi-speaker model", voiceName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read speakers response: %w", err)
	}

	var speakersData map[string]interface{}
	if err := json.Unmarshal(respBody, &speakersData); err != nil {
		return nil, fmt.Errorf("failed to parse speakers response: %w", err)
	}

	return speakersData, nil
}

// GetAudioDevices returns available audio devices
func (s *Service) GetAudioDevices(ctx context.Context) ([]map[string]interface{}, error) {
	if !s.STTEnabled() {
		return nil, fmt.Errorf("STT service is disabled")
	}

	args := []string{
		s.config.STT.PythonScriptPath,
		"--list-devices",
	}

	// Use the virtual environment Python interpreter
	pythonPath := "/opt/pma/speech/venv_speech/bin/python3"
	cmd := exec.CommandContext(ctx, pythonPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log the error but provide default audio devices as fallback
		s.logger.WithError(err).WithField("output", string(output)).Warn("Failed to get audio devices from system, providing default devices")

		// Return default audio devices for Raspberry Pi 5
		return s.getDefaultAudioDevices(), nil
	}

	// Parse the output to extract device information
	lines := strings.Split(string(output), "\n")
	var devices []map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "Available") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				deviceIndex := strings.TrimSpace(parts[0])
				deviceName := strings.TrimSpace(parts[1])

				// Determine device type based on name keywords
				deviceType := "output"
				if strings.Contains(strings.ToLower(deviceName), "mic") ||
					strings.Contains(strings.ToLower(deviceName), "input") ||
					strings.Contains(strings.ToLower(deviceName), "capture") ||
					strings.Contains(strings.ToLower(deviceName), "record") {
					deviceType = "input"
				}

				devices = append(devices, map[string]interface{}{
					"id":         deviceIndex,
					"name":       deviceName,
					"type":       deviceType,
					"is_default": deviceIndex == "0",
					"channels":   2, // Default to stereo
				})
			}
		}
	}

	// If no devices were parsed, return default devices
	if len(devices) == 0 {
		s.logger.WithField("output", string(output)).Warn("No devices parsed from script output, providing default devices")
		return s.getDefaultAudioDevices(), nil
	}

	return devices, nil
}

// getDefaultAudioDevices returns a set of fallback audio devices with proper structure
func (s *Service) getDefaultAudioDevices() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":         "default-output",
			"name":       "Default Audio Output",
			"type":       "output",
			"is_default": true,
			"channels":   2,
		},
		{
			"id":         "hdmi",
			"name":       "HDMI Audio Output",
			"type":       "output",
			"is_default": false,
			"channels":   2,
		},
		{
			"id":         "usb-output",
			"name":       "USB Audio Output",
			"type":       "output",
			"is_default": false,
			"channels":   2,
		},
		{
			"id":         "default-input",
			"name":       "Default Microphone",
			"type":       "input",
			"is_default": true,
			"channels":   1,
		},
		{
			"id":         "usb-input",
			"name":       "USB Microphone",
			"type":       "input",
			"is_default": false,
			"channels":   1,
		},
	}
}

// SaveUploadedAudio saves an uploaded audio file for STT processing
func (s *Service) SaveUploadedAudio(fileName string, data io.Reader) (string, error) {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(fileName))
	validFormats := map[string]bool{}
	for _, format := range s.config.Audio.AllowedFormats {
		validFormats["."+format] = true
	}

	if !validFormats[ext] {
		return "", fmt.Errorf("unsupported audio format: %s", ext)
	}

	// Generate unique filename
	timestamp := time.Now().Format("20060102_150405")
	basename := strings.TrimSuffix(fileName, ext)
	safeName := fmt.Sprintf("upload_%s_%s%s", timestamp, basename, ext)
	outputPath := filepath.Join(s.config.STT.OutputDir, safeName)

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy data
	_, err = io.Copy(outFile, data)
	if err != nil {
		os.Remove(outputPath) // Clean up on error
		return "", fmt.Errorf("failed to copy audio data: %w", err)
	}

	return outputPath, nil
}

// CleanupTempFiles removes old temporary files
func (s *Service) CleanupTempFiles(maxAge time.Duration) error {
	dirs := []string{
		s.config.TTS.OutputDir,
		s.config.STT.OutputDir,
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && time.Since(info.ModTime()) > maxAge {
				s.logger.WithField("file", path).Debug("Removing old temp file")
				return os.Remove(path)
			}

			return nil
		})

		if err != nil {
			s.logger.WithError(err).WithField("dir", dir).Error("Failed to cleanup temp files")
		}
	}

	return nil
}

// Configuration getter methods

// GetDefaultTTSModel returns the default TTS model from configuration
func (s *Service) GetDefaultTTSModel() string {
	if s.config != nil && s.config.TTS.DefaultModel != "" {
		return s.config.TTS.DefaultModel
	}
	return "tts_models/en/ljspeech/tacotron2-DDC"
}

// GetDefaultSTTModel returns the default STT model from configuration
func (s *Service) GetDefaultSTTModel() string {
	if s.config != nil && s.config.STT.DefaultModel != "" {
		return s.config.STT.DefaultModel
	}
	return "base"
}

// GetDefaultLanguage returns the default language from configuration
func (s *Service) GetDefaultLanguage() string {
	if s.config != nil && s.config.STT.Language != "" {
		return s.config.STT.Language
	}
	return "en"
}

// GetAutoCorrectEnabled returns whether autocorrect is enabled from configuration
func (s *Service) GetAutoCorrectEnabled() bool {
	if s.config != nil {
		return s.config.STT.Autocorrect
	}
	return false
}

// GetMaxTextLength returns the maximum text length from configuration
func (s *Service) GetMaxTextLength() int {
	if s.config != nil && s.config.TTS.MaxTextLength > 0 {
		return s.config.TTS.MaxTextLength
	}
	return 5000
}

// GetTimeout returns the timeout setting from configuration
func (s *Service) GetTimeout() string {
	if s.config != nil && s.config.TTS.Timeout != "" {
		return s.config.TTS.Timeout
	}
	return "30s"
}

// GetSupportedAudioFormats returns the supported audio formats from configuration
func (s *Service) GetSupportedAudioFormats() []string {
	if s.config != nil && len(s.config.Audio.AllowedFormats) > 0 {
		return s.config.Audio.AllowedFormats
	}
	return []string{"wav", "mp3", "ogg", "webm", "m4a"}
}

// GetSampleRate returns the sample rate from configuration
func (s *Service) GetSampleRate() int {
	if s.config != nil && s.config.Audio.SampleRate > 0 {
		return s.config.Audio.SampleRate
	}
	return 16000
}
