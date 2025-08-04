package speech

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// STTError represents different types of STT service errors
type STTError struct {
	Type        STTErrorType `json:"type"`
	Message     string       `json:"message"`
	Underlying  error        `json:"-"`
	Recoverable bool         `json:"recoverable"`
	Solution    string       `json:"solution,omitempty"`
}

// STTErrorType represents the category of STT error
type STTErrorType string

const (
	// Service-level errors
	STTErrorServiceDisabled  STTErrorType = "service_disabled"
	STTErrorServiceUnavailable STTErrorType = "service_unavailable"
	STTErrorDependencyMissing STTErrorType = "dependency_missing"
	
	// Audio-related errors
	STTErrorNoAudioDevices    STTErrorType = "no_audio_devices"
	STTErrorAudioDeviceFailed STTErrorType = "audio_device_failed"
	STTErrorNoAudioData       STTErrorType = "no_audio_data"
	STTErrorAudioFormatUnsupported STTErrorType = "audio_format_unsupported"
	STTErrorAudioTooLarge     STTErrorType = "audio_too_large"
	
	// Processing errors
	STTErrorModelLoadFailed   STTErrorType = "model_load_failed"
	STTErrorTranscriptionFailed STTErrorType = "transcription_failed"
	STTErrorTimeout           STTErrorType = "timeout"
	STTErrorInvalidInput      STTErrorType = "invalid_input"
	
	// System errors
	STTErrorPermissionDenied  STTErrorType = "permission_denied"
	STTErrorInsufficientMemory STTErrorType = "insufficient_memory"
	STTErrorPythonEnvironment STTErrorType = "python_environment"
	STTErrorUnknown           STTErrorType = "unknown"
)

// Error implements the error interface
func (e *STTError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("STT %s: %s (caused by: %v)", e.Type, e.Message, e.Underlying)
	}
	return fmt.Sprintf("STT %s: %s", e.Type, e.Message)
}

// NewSTTError creates a new STT error
func NewSTTError(errorType STTErrorType, message string, underlying error) *STTError {
	sttErr := &STTError{
		Type:       errorType,
		Message:    message,
		Underlying: underlying,
	}
	
	// Set recoverability and solutions based on error type
	switch errorType {
	case STTErrorServiceDisabled:
		sttErr.Recoverable = true
		sttErr.Solution = "Enable STT service in configuration"
	case STTErrorNoAudioDevices:
		sttErr.Recoverable = true
		sttErr.Solution = "Check audio device connections and drivers"
	case STTErrorAudioDeviceFailed:
		sttErr.Recoverable = true
		sttErr.Solution = "Try a different audio device or restart audio services"
	case STTErrorModelLoadFailed:
		sttErr.Recoverable = true
		sttErr.Solution = "Check Whisper model installation and available memory"
	case STTErrorTimeout:
		sttErr.Recoverable = true
		sttErr.Solution = "Try with shorter audio or increase timeout settings"
	case STTErrorAudioFormatUnsupported:
		sttErr.Recoverable = true
		sttErr.Solution = "Convert audio to supported format (WAV, MP3, M4A, OGG, WebM, MP4)"
	case STTErrorPermissionDenied:
		sttErr.Recoverable = true
		sttErr.Solution = "Check file permissions and audio device access"
	case STTErrorPythonEnvironment:
		sttErr.Recoverable = true
		sttErr.Solution = "Check Python virtual environment and dependencies"
	default:
		sttErr.Recoverable = false
		sttErr.Solution = "Contact system administrator"
	}
	
	return sttErr
}

// CategorizeError analyzes an error and categorizes it into the appropriate STT error type
func CategorizeError(err error) *STTError {
	if err == nil {
		return nil
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Check for specific error patterns
	switch {
	case strings.Contains(errStr, "stt service is disabled") || strings.Contains(errStr, "stt service not available"):
		return NewSTTError(STTErrorServiceDisabled, "STT service is not enabled", err)
		
	case strings.Contains(errStr, "no input device found") || strings.Contains(errStr, "no audio devices"):
		return NewSTTError(STTErrorNoAudioDevices, "No audio input devices found", err)
		
	case strings.Contains(errStr, "error opening audio stream") || strings.Contains(errStr, "audio device"):
		return NewSTTError(STTErrorAudioDeviceFailed, "Failed to access audio device", err)
		
	case strings.Contains(errStr, "no audio recorded") || strings.Contains(errStr, "no speech detected"):
		return NewSTTError(STTErrorNoAudioData, "No audio data captured", err)
		
	case strings.Contains(errStr, "unsupported audio format") || strings.Contains(errStr, "invalid format"):
		return NewSTTError(STTErrorAudioFormatUnsupported, "Audio format not supported", err)
		
	case strings.Contains(errStr, "file size exceeds") || strings.Contains(errStr, "too large"):
		return NewSTTError(STTErrorAudioTooLarge, "Audio file too large", err)
		
	case strings.Contains(errStr, "failed to load model") || strings.Contains(errStr, "model"):
		return NewSTTError(STTErrorModelLoadFailed, "Failed to load Whisper model", err)
		
	case strings.Contains(errStr, "transcription failed") || strings.Contains(errStr, "whisper"):
		return NewSTTError(STTErrorTranscriptionFailed, "Speech transcription failed", err)
		
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
		return NewSTTError(STTErrorTimeout, "Operation timed out", err)
		
	case strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "access denied"):
		return NewSTTError(STTErrorPermissionDenied, "Permission denied", err)
		
	case strings.Contains(errStr, "out of memory") || strings.Contains(errStr, "memory"):
		return NewSTTError(STTErrorInsufficientMemory, "Insufficient memory", err)
		
	case strings.Contains(errStr, "python") || strings.Contains(errStr, "venv") || strings.Contains(errStr, "import"):
		return NewSTTError(STTErrorPythonEnvironment, "Python environment issue", err)
		
	case strings.Contains(errStr, "invalid") || strings.Contains(errStr, "bad"):
		return NewSTTError(STTErrorInvalidInput, "Invalid input provided", err)
		
	default:
		return NewSTTError(STTErrorUnknown, err.Error(), err)
	}
}

// IsRecoverable checks if an error can potentially be recovered from
func IsRecoverable(err error) bool {
	var sttErr *STTError
	if errors.As(err, &sttErr) {
		return sttErr.Recoverable
	}
	return false
}

// GetSolution returns a suggested solution for an error
func GetSolution(err error) string {
	var sttErr *STTError
	if errors.As(err, &sttErr) {
		return sttErr.Solution
	}
	return "Check system logs for more details"
}

// STTHealthStatus represents the health of the STT service
type STTHealthStatus struct {
	Overall       string                    `json:"overall"`       // healthy, degraded, unhealthy
	Components    map[string]ComponentHealth `json:"components"`    // individual component health
	LastError     *STTError                 `json:"last_error,omitempty"`
	LastSuccess   string                    `json:"last_success,omitempty"`
	ErrorHistory  []*STTError               `json:"error_history,omitempty"`
	Suggestions   []string                  `json:"suggestions,omitempty"`
}

type ComponentHealth struct {
	Status      string    `json:"status"`       // healthy, degraded, unhealthy
	LastChecked string    `json:"last_checked"`
	Error       *STTError `json:"error,omitempty"`
}

// HealthChecker provides health checking capabilities for STT service
type HealthChecker struct {
	service        *Service
	lastHealthCheck *STTHealthStatus
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(service *Service) *HealthChecker {
	return &HealthChecker{
		service: service,
	}
}

// CheckHealth performs a comprehensive health check of the STT service
func (h *HealthChecker) CheckHealth() *STTHealthStatus {
	status := &STTHealthStatus{
		Components: make(map[string]ComponentHealth),
	}
	
	healthyComponents := 0
	totalComponents := 0
	
	// Check if STT is enabled
	totalComponents++
	if h.service.STTEnabled() {
		status.Components["stt_enabled"] = ComponentHealth{
			Status:      "healthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
		}
		healthyComponents++
	} else {
		status.Components["stt_enabled"] = ComponentHealth{
			Status:      "unhealthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
			Error:       NewSTTError(STTErrorServiceDisabled, "STT service is disabled", nil),
		}
	}
	
	// Check Python environment
	totalComponents++
	pythonPath := "/opt/pma/speech/venv_speech/bin/python3"
	if _, err := exec.LookPath(pythonPath); err == nil {
		status.Components["python_env"] = ComponentHealth{
			Status:      "healthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
		}
		healthyComponents++
	} else {
		status.Components["python_env"] = ComponentHealth{
			Status:      "unhealthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
			Error:       NewSTTError(STTErrorPythonEnvironment, "Python environment not found", err),
		}
	}
	
	// Check STT script
	totalComponents++
	if _, err := os.Stat(h.service.config.STT.PythonScriptPath); err == nil {
		status.Components["stt_script"] = ComponentHealth{
			Status:      "healthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
		}
		healthyComponents++
	} else {
		status.Components["stt_script"] = ComponentHealth{
			Status:      "unhealthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
			Error:       NewSTTError(STTErrorDependencyMissing, "STT script not found", err),
		}
	}
	
	// Check output directory
	totalComponents++
	if _, err := os.Stat(h.service.config.STT.OutputDir); err == nil {
		status.Components["output_dir"] = ComponentHealth{
			Status:      "healthy",
			LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
		}
		healthyComponents++
	} else {
		// Try to create it
		if err := os.MkdirAll(h.service.config.STT.OutputDir, 0755); err == nil {
			status.Components["output_dir"] = ComponentHealth{
				Status:      "healthy",
				LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
			}
			healthyComponents++
		} else {
			status.Components["output_dir"] = ComponentHealth{
				Status:      "unhealthy",
				LastChecked: fmt.Sprintf("%d", time.Now().Unix()),
				Error:       NewSTTError(STTErrorPermissionDenied, "Cannot access output directory", err),
			}
		}
	}
	
	// Determine overall health
	if healthyComponents == totalComponents {
		status.Overall = "healthy"
	} else if healthyComponents > 0 {
		status.Overall = "degraded"
		status.Suggestions = append(status.Suggestions, "Some STT components are not functioning properly")
	} else {
		status.Overall = "unhealthy"
		status.Suggestions = append(status.Suggestions, "STT service is not operational")
	}
	
	h.lastHealthCheck = status
	return status
}

// GetLastHealthCheck returns the last health check result
func (h *HealthChecker) GetLastHealthCheck() *STTHealthStatus {
	return h.lastHealthCheck
}