package debug

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DebugConfig holds debug logging configuration
type DebugConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	Level       string   `mapstructure:"level"` // debug, info, warn, error
	FileEnabled bool     `mapstructure:"file_enabled"`
	FilePath    string   `mapstructure:"file_path"`
	MaxFileSize int64    `mapstructure:"max_file_size"` // in bytes
	MaxFiles    int      `mapstructure:"max_files"`
	Console     bool     `mapstructure:"console"`
	Components  []string `mapstructure:"components"` // specific components to debug
}

// DebugLogger provides comprehensive debug logging capabilities
type DebugLogger struct {
	config     *DebugConfig
	logger     *logrus.Logger
	fileWriter io.WriteCloser
	mutex      sync.RWMutex
	components map[string]bool // enabled components
}

// DebugEntry represents a debug log entry with rich context
type DebugEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Function    string                 `json:"function,omitempty"`
	File        string                 `json:"file,omitempty"`
	Line        int                    `json:"line,omitempty"`
	GoroutineID string                 `json:"goroutine_id,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Error       *DebugError            `json:"error,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
}

// DebugError represents error information in debug logs
type DebugError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// NewDebugLogger creates a new debug logger instance
func NewDebugLogger(config *DebugConfig) (*DebugLogger, error) {
	if config == nil {
		config = &DebugConfig{
			Enabled:     false,
			Level:       "debug",
			FileEnabled: false,
			FilePath:    "./logs/debug.log",
			MaxFileSize: 10 * 1024 * 1024, // 10MB
			MaxFiles:    5,
			Console:     false,
			Components:  []string{},
		}
	}

	dl := &DebugLogger{
		config:     config,
		logger:     logrus.New(),
		components: make(map[string]bool),
	}

	// Setup components filter
	for _, component := range config.Components {
		dl.components[component] = true
	}

	// Configure logger
	if err := dl.setupLogger(); err != nil {
		return nil, fmt.Errorf("failed to setup debug logger: %w", err)
	}

	return dl, nil
}

// setupLogger configures the debug logger
func (dl *DebugLogger) setupLogger() error {
	// Set formatter
	dl.logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     false, // Keep compact for file logging
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "function",
		},
	})

	// Set level
	level, err := logrus.ParseLevel(dl.config.Level)
	if err != nil {
		level = logrus.DebugLevel
	}
	dl.logger.SetLevel(level)

	// Setup output
	var outputs []io.Writer

	// Console output
	if dl.config.Console {
		outputs = append(outputs, os.Stdout)
	}

	// File output
	if dl.config.FileEnabled {
		if err := dl.setupFileOutput(); err != nil {
			return fmt.Errorf("failed to setup file output: %w", err)
		}
		if dl.fileWriter != nil {
			outputs = append(outputs, dl.fileWriter)
		}
	}

	// Set multi-writer if we have outputs
	if len(outputs) > 0 {
		dl.logger.SetOutput(io.MultiWriter(outputs...))
	} else {
		// Default to stdout if no outputs configured
		dl.logger.SetOutput(os.Stdout)
	}

	return nil
}

// setupFileOutput sets up file output with rotation
func (dl *DebugLogger) setupFileOutput() error {
	// Ensure directory exists
	dir := filepath.Dir(dl.config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for writing
	file, err := os.OpenFile(dl.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open debug log file: %w", err)
	}

	dl.fileWriter = file

	// Check file size and rotate if needed
	if err := dl.checkAndRotate(); err != nil {
		return fmt.Errorf("failed to check/rotate log file: %w", err)
	}

	return nil
}

// checkAndRotate checks if log rotation is needed
func (dl *DebugLogger) checkAndRotate() error {
	if dl.fileWriter == nil {
		return nil
	}

	// Get file info
	file, ok := dl.fileWriter.(*os.File)
	if !ok {
		return nil
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Rotate if file is too large
	if info.Size() > dl.config.MaxFileSize {
		return dl.rotateLogFile()
	}

	return nil
}

// rotateLogFile rotates the log file
func (dl *DebugLogger) rotateLogFile() error {
	if dl.fileWriter == nil {
		return nil
	}

	// Close current file
	dl.fileWriter.Close()

	// Rotate existing files
	for i := dl.config.MaxFiles - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", dl.config.FilePath, i)
		newPath := fmt.Sprintf("%s.%d", dl.config.FilePath, i+1)

		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// Rename current file
	if _, err := os.Stat(dl.config.FilePath); err == nil {
		os.Rename(dl.config.FilePath, dl.config.FilePath+".1")
	}

	// Open new file
	file, err := os.OpenFile(dl.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	dl.fileWriter = file
	return nil
}

// IsComponentEnabled checks if debug logging is enabled for a specific component
func (dl *DebugLogger) IsComponentEnabled(component string) bool {
	dl.mutex.RLock()
	defer dl.mutex.RUnlock()

	// If no components specified, all are enabled
	if len(dl.components) == 0 {
		return dl.config.Enabled
	}

	return dl.config.Enabled && dl.components[component]
}

// Log creates a debug log entry
func (dl *DebugLogger) Log(level, component, message string, fields map[string]interface{}) {
	if !dl.IsComponentEnabled(component) {
		return
	}

	entry := dl.createDebugEntry(level, component, message, fields)
	dl.writeEntry(entry)
}

// LogWithContext creates a debug log entry with context
func (dl *DebugLogger) LogWithContext(ctx context.Context, level, component, message string, fields map[string]interface{}) {
	if !dl.IsComponentEnabled(component) {
		return
	}

	entry := dl.createDebugEntry(level, component, message, fields)

	// Add context information
	if ctx != nil {
		if requestID := ctx.Value("request_id"); requestID != nil {
			entry.RequestID = fmt.Sprintf("%v", requestID)
		}
		if userID := ctx.Value("user_id"); userID != nil {
			entry.UserID = fmt.Sprintf("%v", userID)
		}
	}

	dl.writeEntry(entry)
}

// LogError logs an error with debug information
func (dl *DebugLogger) LogError(component, message string, err error, fields map[string]interface{}) {
	if !dl.IsComponentEnabled(component) {
		return
	}

	entry := dl.createDebugEntry("error", component, message, fields)

	if err != nil {
		entry.Error = &DebugError{
			Type:    fmt.Sprintf("%T", err),
			Message: err.Error(),
		}
	}

	dl.writeEntry(entry)
}

// LogDuration logs a duration with debug information
func (dl *DebugLogger) LogDuration(component, message string, duration time.Duration, fields map[string]interface{}) {
	if !dl.IsComponentEnabled(component) {
		return
	}

	entry := dl.createDebugEntry("debug", component, message, fields)
	entry.Duration = &duration

	dl.writeEntry(entry)
}

// createDebugEntry creates a debug entry with caller information
func (dl *DebugLogger) createDebugEntry(level, component, message string, fields map[string]interface{}) *DebugEntry {
	entry := &DebugEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: component,
		Message:   message,
		Context:   fields,
	}

	// Get caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry.Function = fn.Name()
		}
		entry.File = filepath.Base(file)
		entry.Line = line
	}

	// Get goroutine ID
	entry.GoroutineID = fmt.Sprintf("%d", runtime.NumGoroutine())

	return entry
}

// writeEntry writes the debug entry to the logger
func (dl *DebugLogger) writeEntry(entry *DebugEntry) {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	// Check for rotation
	if err := dl.checkAndRotate(); err != nil {
		// Log rotation error to stderr
		fmt.Fprintf(os.Stderr, "Debug log rotation error: %v\n", err)
	}

	// Convert to logrus entry
	logrusEntry := dl.logger.WithFields(logrus.Fields{
		"component":    entry.Component,
		"function":     entry.Function,
		"file":         entry.File,
		"line":         entry.Line,
		"goroutine_id": entry.GoroutineID,
		"request_id":   entry.RequestID,
		"user_id":      entry.UserID,
	})

	// Add context fields
	for k, v := range entry.Context {
		logrusEntry = logrusEntry.WithField(k, v)
	}

	// Add error if present
	if entry.Error != nil {
		logrusEntry = logrusEntry.WithField("error", entry.Error)
	}

	// Add duration if present
	if entry.Duration != nil {
		logrusEntry = logrusEntry.WithField("duration", entry.Duration.String())
	}

	// Log based on level
	switch strings.ToLower(entry.Level) {
	case "debug":
		logrusEntry.Debug(entry.Message)
	case "info":
		logrusEntry.Info(entry.Message)
	case "warn":
		logrusEntry.Warn(entry.Message)
	case "error":
		logrusEntry.Error(entry.Message)
	default:
		logrusEntry.Debug(entry.Message)
	}
}

// Close closes the debug logger and any open files
func (dl *DebugLogger) Close() error {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	if dl.fileWriter != nil {
		return dl.fileWriter.Close()
	}
	return nil
}

// Flush flushes any buffered output
func (dl *DebugLogger) Flush() {
	// Logrus handles buffering internally, but we can force a sync
	if dl.fileWriter != nil {
		if file, ok := dl.fileWriter.(*os.File); ok {
			file.Sync()
		}
	}
}

// GetStats returns debug logger statistics
func (dl *DebugLogger) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":      dl.config.Enabled,
		"level":        dl.config.Level,
		"file_enabled": dl.config.FileEnabled,
		"console":      dl.config.Console,
		"components":   dl.config.Components,
	}

	if dl.config.FileEnabled {
		stats["file_path"] = dl.config.FilePath
		stats["max_file_size"] = dl.config.MaxFileSize
		stats["max_files"] = dl.config.MaxFiles
	}

	return stats
}
