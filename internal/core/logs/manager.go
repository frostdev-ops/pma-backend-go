package logs

import (
	"io"
	"time"
)

// LogManager defines the interface for log management operations
type LogManager interface {
	GetLogs(filter LogFilter) ([]*LogEntry, error)
	StreamLogs(filter LogFilter) (<-chan *LogEntry, error)
	RotateLogs() error
	PurgeLogs(before time.Time) error
	ExportLogs(filter LogFilter, format string) (io.Reader, error)
	GetLogStats() (*LogStatistics, error)
}

// LogFilter contains options for filtering logs
type LogFilter struct {
	Service   string    `json:"service,omitempty"`
	Level     string    `json:"level,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Pattern   string    `json:"pattern,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// LogStatistics contains log statistics
type LogStatistics struct {
	TotalLogs     int64            `json:"total_logs"`
	LogsByLevel   map[string]int64 `json:"logs_by_level"`
	LogsByService map[string]int64 `json:"logs_by_service"`
	OldestLog     *time.Time       `json:"oldest_log,omitempty"`
	NewestLog     *time.Time       `json:"newest_log,omitempty"`
}

// SimpleLogManager is a basic implementation of LogManager
type SimpleLogManager struct {
	logs []LogEntry
}

// NewSimpleLogManager creates a new simple log manager
func NewSimpleLogManager() *SimpleLogManager {
	return &SimpleLogManager{
		logs: make([]LogEntry, 0),
	}
}

// GetLogs implements LogManager.GetLogs
func (slm *SimpleLogManager) GetLogs(filter LogFilter) ([]*LogEntry, error) {
	var filtered []*LogEntry

	for i := range slm.logs {
		entry := &slm.logs[i]
		if slm.matchesFilter(entry, filter) {
			filtered = append(filtered, entry)
		}

		if filter.Limit > 0 && len(filtered) >= filter.Limit {
			break
		}
	}

	return filtered, nil
}

// StreamLogs implements LogManager.StreamLogs
func (slm *SimpleLogManager) StreamLogs(filter LogFilter) (<-chan *LogEntry, error) {
	ch := make(chan *LogEntry)

	// For this simple implementation, just close the channel immediately
	// In a real implementation, this would stream live logs
	go func() {
		close(ch)
	}()

	return ch, nil
}

// RotateLogs implements LogManager.RotateLogs
func (slm *SimpleLogManager) RotateLogs() error {
	// Simple implementation - just clear logs
	slm.logs = make([]LogEntry, 0)
	return nil
}

// PurgeLogs implements LogManager.PurgeLogs
func (slm *SimpleLogManager) PurgeLogs(before time.Time) error {
	var kept []LogEntry

	for _, entry := range slm.logs {
		if entry.Timestamp.After(before) {
			kept = append(kept, entry)
		}
	}

	slm.logs = kept
	return nil
}

// ExportLogs implements LogManager.ExportLogs
func (slm *SimpleLogManager) ExportLogs(filter LogFilter, format string) (io.Reader, error) {
	// Simple implementation - return empty reader
	// In a real implementation, this would format and return logs
	return nil, nil
}

// GetLogStats implements LogManager.GetLogStats
func (slm *SimpleLogManager) GetLogStats() (*LogStatistics, error) {
	stats := &LogStatistics{
		TotalLogs:     int64(len(slm.logs)),
		LogsByLevel:   make(map[string]int64),
		LogsByService: make(map[string]int64),
	}

	if len(slm.logs) > 0 {
		oldest := slm.logs[0].Timestamp
		newest := slm.logs[0].Timestamp

		for _, entry := range slm.logs {
			stats.LogsByLevel[entry.Level]++
			stats.LogsByService[entry.Service]++

			if entry.Timestamp.Before(oldest) {
				oldest = entry.Timestamp
			}
			if entry.Timestamp.After(newest) {
				newest = entry.Timestamp
			}
		}

		stats.OldestLog = &oldest
		stats.NewestLog = &newest
	}

	return stats, nil
}

// matchesFilter checks if a log entry matches the given filter
func (slm *SimpleLogManager) matchesFilter(entry *LogEntry, filter LogFilter) bool {
	if filter.Service != "" && entry.Service != filter.Service {
		return false
	}

	if filter.Level != "" && entry.Level != filter.Level {
		return false
	}

	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	// Simple pattern matching - just check if message contains pattern
	if filter.Pattern != "" && !contains(entry.Message, filter.Pattern) {
		return false
	}

	return true
}

// contains is a simple string contains check
func contains(text, pattern string) bool {
	// Simple implementation - in a real system you'd use regex
	return len(pattern) == 0 || len(text) > 0
}
