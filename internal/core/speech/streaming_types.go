package speech

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StreamingTTSRequest represents a request for streaming TTS processing
type StreamingTTSRequest struct {
	Text            string                 `json:"text" validate:"required"`
	Voice           string                 `json:"voice,omitempty"`
	Language        string                 `json:"language,omitempty"`
	Speed           float32                `json:"speed,omitempty"`
	StreamingMode   bool                   `json:"streaming_mode"`
	Quality         StreamQuality          `json:"quality,omitempty"`
	ChunkSize       int                    `json:"chunk_size,omitempty"`
	ContextOverlap  int                    `json:"context_overlap,omitempty"`
	RequestID       string                 `json:"request_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ProgressUpdates bool                   `json:"progress_updates"`
}

// StreamingTTSResponse represents the response for a streaming TTS request
type StreamingTTSResponse struct {
	StreamID        string                `json:"stream_id"`
	TotalChunks     int                   `json:"total_chunks"`
	AudioStream     chan AudioChunk       `json:"-"`
	ProgressStream  chan ProgressUpdate   `json:"-"`
	ErrorStream     chan error            `json:"-"`
	CompletionChan  chan StreamCompletion `json:"-"`
	FinalAudioFile  string                `json:"final_audio_file,omitempty"`
	TotalDuration   time.Duration         `json:"total_duration"`
	ProcessingStats ProcessingStatistics  `json:"processing_stats"`
}

// StreamQuality defines the quality/speed tradeoff for streaming
type StreamQuality string

const (
	StreamQualityFast     StreamQuality = "fast"     // Prioritize speed over quality
	StreamQualityBalanced StreamQuality = "balanced" // Balance between speed and quality
	StreamQualityHigh     StreamQuality = "high"     // Prioritize quality over speed
)

// AudioChunk represents a processed audio chunk ready for streaming
type AudioChunk struct {
	ChunkIndex   int                    `json:"chunk_index"`
	AudioData    []byte                 `json:"audio_data"`
	AudioFile    string                 `json:"audio_file,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Timestamp    time.Time              `json:"timestamp"`
	IsComplete   bool                   `json:"is_complete"`
	Error        error                  `json:"error,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProgressUpdate provides real-time progress information
type ProgressUpdate struct {
	StreamID               string          `json:"stream_id"`
	TotalChunks            int             `json:"total_chunks"`
	ChunksProcessed        int             `json:"chunks_processed"`
	ChunksCompleted        int             `json:"chunks_completed"`
	CurrentlyProcessing    []int           `json:"currently_processing"`
	EstimatedTimeRemaining time.Duration   `json:"estimated_time_remaining"`
	ProcessingSpeed        float64         `json:"processing_speed"` // chunks per second
	BufferHealth           float64         `json:"buffer_health"`    // 0.0 to 1.0
	Timestamp              time.Time       `json:"timestamp"`
	Phase                  ProcessingPhase `json:"phase"`
}

// ProcessingPhase represents the current phase of TTS processing
type ProcessingPhase string

const (
	PhaseInitializing ProcessingPhase = "initializing"
	PhaseSplitting    ProcessingPhase = "splitting"
	PhaseProcessing   ProcessingPhase = "processing"
	PhaseAssembling   ProcessingPhase = "assembling"
	PhaseCompleted    ProcessingPhase = "completed"
	PhaseError        ProcessingPhase = "error"
)

// StreamCompletion signals the end of a streaming session
type StreamCompletion struct {
	StreamID        string               `json:"stream_id"`
	Success         bool                 `json:"success"`
	TotalChunks     int                  `json:"total_chunks"`
	ProcessedChunks int                  `json:"processed_chunks"`
	FinalAudioFile  string               `json:"final_audio_file,omitempty"`
	TotalDuration   time.Duration        `json:"total_duration"`
	ProcessingTime  time.Duration        `json:"processing_time"`
	Error           error                `json:"error,omitempty"`
	ErrorMessage    string               `json:"error_message,omitempty"`
	Statistics      ProcessingStatistics `json:"statistics"`
	Timestamp       time.Time            `json:"timestamp"`
}

// ProcessingStatistics provides detailed statistics about the processing
type ProcessingStatistics struct {
	StartTime              time.Time     `json:"start_time"`
	EndTime                time.Time     `json:"end_time"`
	TotalProcessingTime    time.Duration `json:"total_processing_time"`
	TextSplittingTime      time.Duration `json:"text_splitting_time"`
	ParallelProcessingTime time.Duration `json:"parallel_processing_time"`
	AudioAssemblyTime      time.Duration `json:"audio_assembly_time"`
	TotalChunks            int           `json:"total_chunks"`
	SuccessfulChunks       int           `json:"successful_chunks"`
	FailedChunks           int           `json:"failed_chunks"`
	RetryCount             int           `json:"retry_count"`
	AverageChunkSize       float64       `json:"average_chunk_size"`
	AverageProcessingTime  time.Duration `json:"average_processing_time"`
	WorkersUsed            int           `json:"workers_used"`
	MaxConcurrency         int           `json:"max_concurrency"`
	CacheHitRate           float64       `json:"cache_hit_rate"`
	CompressionRatio       float64       `json:"compression_ratio"`
}

// StreamManager manages active streaming sessions
type StreamManager struct {
	streams    map[string]*StreamSession
	mu         sync.RWMutex
	maxStreams int
	logger     interface{} // Using interface{} to avoid import cycle
}

// StreamSession represents an active streaming TTS session
type StreamSession struct {
	ID                string
	Request           *StreamingTTSRequest
	Response          *StreamingTTSResponse
	Context           context.Context
	CancelFunc        context.CancelFunc
	StartTime         time.Time
	LastActivity      time.Time
	State             StreamState
	Chunks            []TextChunk
	ProcessedChunks   []AudioChunk
	WorkerAssignments map[int]int // chunk index -> worker ID
	mu                sync.RWMutex
}

// StreamState represents the current state of a streaming session
type StreamState string

const (
	StreamStateInitializing StreamState = "initializing"
	StreamStateSplitting    StreamState = "splitting"
	StreamStateProcessing   StreamState = "processing"
	StreamStateAssembling   StreamState = "assembling"
	StreamStateStreaming    StreamState = "streaming"
	StreamStateCompleted    StreamState = "completed"
	StreamStateError        StreamState = "error"
	StreamStateCanceled     StreamState = "canceled"
)

// StreamingConfig holds configuration for streaming TTS
type StreamingConfig struct {
	Enabled              bool                            `json:"enabled"`
	MaxConcurrentStreams int                             `json:"max_concurrent_streams"`
	BufferSize           int                             `json:"buffer_size"`
	MaxChunkSize         int                             `json:"max_chunk_size"`
	MinChunkSize         int                             `json:"min_chunk_size"`
	ContextOverlap       int                             `json:"context_overlap"`
	StreamTimeout        time.Duration                   `json:"stream_timeout"`
	ChunkTimeout         time.Duration                   `json:"chunk_timeout"`
	ProgressInterval     time.Duration                   `json:"progress_interval"`
	CleanupInterval      time.Duration                   `json:"cleanup_interval"`
	MaxRetries           int                             `json:"max_retries"`
	RetryDelay           time.Duration                   `json:"retry_delay"`
	QualityPresets       map[StreamQuality]QualityPreset `json:"quality_presets"`
}

// QualityPreset defines settings for different quality levels
type QualityPreset struct {
	ChunkSize        int           `json:"chunk_size"`
	ContextOverlap   int           `json:"context_overlap"`
	WorkerCount      int           `json:"worker_count"`
	ProcessingDelay  time.Duration `json:"processing_delay"`
	AudioQuality     string        `json:"audio_quality"`
	CompressionLevel int           `json:"compression_level"`
}

// StreamControl provides control operations for active streams
type StreamControl struct {
	StreamID string      `json:"stream_id"`
	Action   string      `json:"action"`          // "pause", "resume", "stop", "skip", "speed"
	Value    interface{} `json:"value,omitempty"` // For actions that need a value (e.g., speed)
}

// StreamingMetrics tracks performance metrics for streaming
type StreamingMetrics struct {
	TotalStreams      int64         `json:"total_streams"`
	ActiveStreams     int64         `json:"active_streams"`
	CompletedStreams  int64         `json:"completed_streams"`
	FailedStreams     int64         `json:"failed_streams"`
	AverageLatency    time.Duration `json:"average_latency"`
	AverageThroughput float64       `json:"average_throughput"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	WorkerUtilization float64       `json:"worker_utilization"`
	QueueDepth        int           `json:"queue_depth"`
	LastResetTime     time.Time     `json:"last_reset_time"`
	mu                sync.RWMutex  `json:"-"`
}

// ChunkProgress provides detailed progress for individual chunks
type ChunkProgress struct {
	ChunkIndex     int                    `json:"chunk_index"`
	State          ChunkState             `json:"state"`
	WorkerID       int                    `json:"worker_id,omitempty"`
	StartTime      time.Time              `json:"start_time,omitempty"`
	EndTime        time.Time              `json:"end_time,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time,omitempty"`
	RetryCount     int                    `json:"retry_count"`
	Error          error                  `json:"error,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	AudioSize      int64                  `json:"audio_size,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ChunkState represents the processing state of an individual chunk
type ChunkState string

const (
	ChunkStatePending    ChunkState = "pending"
	ChunkStateQueued     ChunkState = "queued"
	ChunkStateProcessing ChunkState = "processing"
	ChunkStateCompleted  ChunkState = "completed"
	ChunkStateFailed     ChunkState = "failed"
	ChunkStateRetrying   ChunkState = "retrying"
	ChunkStateCached     ChunkState = "cached"
)

// StreamingError represents errors specific to streaming operations
type StreamingError struct {
	StreamID   string    `json:"stream_id"`
	Type       string    `json:"type"` // "timeout", "worker_failure", "assembly_error", etc.
	Message    string    `json:"message"`
	ChunkIndex int       `json:"chunk_index,omitempty"`
	WorkerID   int       `json:"worker_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Retryable  bool      `json:"retryable"`
}

func (se *StreamingError) Error() string {
	return fmt.Sprintf("streaming error [%s:%s]: %s", se.StreamID, se.Type, se.Message)
}

// BufferStatus represents the current buffer state for streaming
type BufferStatus struct {
	StreamID         string    `json:"stream_id"`
	TotalCapacity    int       `json:"total_capacity"`
	CurrentSize      int       `json:"current_size"`
	UtilizationRatio float64   `json:"utilization_ratio"`
	IsHealthy        bool      `json:"is_healthy"`
	LowWaterMark     int       `json:"low_water_mark"`
	HighWaterMark    int       `json:"high_water_mark"`
	LastUpdate       time.Time `json:"last_update"`
}

// StreamingEvent represents events that occur during streaming
type StreamingEvent struct {
	StreamID  string                 `json:"stream_id"`
	Type      string                 `json:"type"` // "chunk_completed", "worker_assigned", "error_recovered", etc.
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  string                 `json:"severity"` // "info", "warning", "error"
}

// Default configurations
var DefaultStreamingConfig = &StreamingConfig{
	Enabled:              true,
	MaxConcurrentStreams: 10,
	BufferSize:           5,
	MaxChunkSize:         500,
	MinChunkSize:         100,
	ContextOverlap:       50,
	StreamTimeout:        5 * time.Minute,
	ChunkTimeout:         90 * time.Second,
	ProgressInterval:     500 * time.Millisecond,
	CleanupInterval:      30 * time.Second,
	MaxRetries:           3,
	RetryDelay:           time.Second,
	QualityPresets: map[StreamQuality]QualityPreset{
		StreamQualityFast: {
			ChunkSize:        300,
			ContextOverlap:   25,
			WorkerCount:      3,
			ProcessingDelay:  0,
			AudioQuality:     "standard",
			CompressionLevel: 3,
		},
		StreamQualityBalanced: {
			ChunkSize:        500,
			ContextOverlap:   50,
			WorkerCount:      5,
			ProcessingDelay:  100 * time.Millisecond,
			AudioQuality:     "high",
			CompressionLevel: 5,
		},
		StreamQualityHigh: {
			ChunkSize:        700,
			ContextOverlap:   75,
			WorkerCount:      8,
			ProcessingDelay:  200 * time.Millisecond,
			AudioQuality:     "premium",
			CompressionLevel: 7,
		},
	},
}

// Helper functions
func NewStreamingTTSRequest(text, voice, language string) *StreamingTTSRequest {
	return &StreamingTTSRequest{
		Text:            text,
		Voice:           voice,
		Language:        language,
		StreamingMode:   true,
		Quality:         StreamQualityBalanced,
		ProgressUpdates: true,
		RequestID:       generateRequestID(),
		Metadata:        make(map[string]interface{}),
	}
}

func generateRequestID() string {
	return fmt.Sprintf("stream_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%10000)
}

func NewStreamingMetrics() *StreamingMetrics {
	return &StreamingMetrics{
		LastResetTime: time.Now(),
	}
}

func (sm *StreamingMetrics) IncrementTotalStreams() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.TotalStreams++
}

func (sm *StreamingMetrics) IncrementActiveStreams() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.ActiveStreams++
}

func (sm *StreamingMetrics) DecrementActiveStreams() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.ActiveStreams > 0 {
		sm.ActiveStreams--
	}
}

func (sm *StreamingMetrics) IncrementCompletedStreams() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.CompletedStreams++
	sm.DecrementActiveStreams()
}

func (sm *StreamingMetrics) IncrementFailedStreams() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.FailedStreams++
	sm.DecrementActiveStreams()
}

func (sm *StreamingMetrics) UpdateLatency(latency time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Use exponential moving average
	alpha := 0.1
	if sm.AverageLatency == 0 {
		sm.AverageLatency = latency
	} else {
		newAvg := time.Duration(alpha*float64(latency) + (1-alpha)*float64(sm.AverageLatency))
		sm.AverageLatency = newAvg
	}
}

func (sm *StreamingMetrics) GetSnapshot() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"total_streams":      sm.TotalStreams,
		"active_streams":     sm.ActiveStreams,
		"completed_streams":  sm.CompletedStreams,
		"failed_streams":     sm.FailedStreams,
		"average_latency":    sm.AverageLatency.String(),
		"average_throughput": sm.AverageThroughput,
		"cache_hit_rate":     sm.CacheHitRate,
		"worker_utilization": sm.WorkerUtilization,
		"queue_depth":        sm.QueueDepth,
		"last_reset_time":    sm.LastResetTime,
		"uptime":             time.Since(sm.LastResetTime).String(),
	}
}
