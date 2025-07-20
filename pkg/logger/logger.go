package logger

import (
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RequestMetrics holds metrics for a specific endpoint
type RequestMetrics struct {
	Count       int           `json:"count"`
	TotalTime   time.Duration `json:"total_time"`
	MinLatency  time.Duration `json:"min_latency"`
	MaxLatency  time.Duration `json:"max_latency"`
	AvgLatency  time.Duration `json:"avg_latency"`
}

// BatchLogger wraps logrus.Logger with batching capabilities for 200 status codes
type BatchLogger struct {
	*logrus.Logger
	metrics     map[string]*RequestMetrics
	batchCount  int
	mutex       sync.RWMutex
	batchSize   int
}

// New creates a new logger instance with batching for 200 status codes
func New() *BatchLogger {
	log := logrus.New()

	// Always use JSON formatter for clean, consistent output
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     true, // Makes JSON human-readable with indentation
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "msg",
			logrus.FieldKeyFunc:  "func",
		},
	})

	// Set output
	log.SetOutput(os.Stdout)

	// Set level based on environment
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	return &BatchLogger{
		Logger:     log,
		metrics:    make(map[string]*RequestMetrics),
		batchCount: 0,
		batchSize:  100,
	}
}

// LogRequest logs a request, batching 200 status codes
func (bl *BatchLogger) LogRequest(method, endpoint string, statusCode int, latency time.Duration, fields logrus.Fields) {
	if statusCode == 200 {
		bl.batchSuccess(method, endpoint, latency)
		return
	}

	// Log non-200 status codes immediately
	entry := bl.WithFields(fields)
	if statusCode >= 400 {
		entry.Errorf("%s %s - Status: %d, Latency: %v", method, endpoint, statusCode, latency)
	} else {
		entry.Infof("%s %s - Status: %d, Latency: %v", method, endpoint, statusCode, latency)
	}
}

// batchSuccess adds a successful request to the batch
func (bl *BatchLogger) batchSuccess(method, endpoint string, latency time.Duration) {
	bl.mutex.Lock()
	defer bl.mutex.Unlock()

	key := method + " " + endpoint
	
	if bl.metrics[key] == nil {
		bl.metrics[key] = &RequestMetrics{
			MinLatency: latency,
			MaxLatency: latency,
		}
	}

	metrics := bl.metrics[key]
	metrics.Count++
	metrics.TotalTime += latency

	if latency < metrics.MinLatency {
		metrics.MinLatency = latency
	}
	if latency > metrics.MaxLatency {
		metrics.MaxLatency = latency
	}
	metrics.AvgLatency = metrics.TotalTime / time.Duration(metrics.Count)

	bl.batchCount++

	// Send summary when batch is full
	if bl.batchCount >= bl.batchSize {
		bl.flushBatch()
	}
}

// flushBatch sends a summary of batched requests
func (bl *BatchLogger) flushBatch() {
	if bl.batchCount == 0 {
		return
	}

	summary := make(map[string]interface{})
	summary["batch_summary"] = true
	summary["total_requests"] = bl.batchCount
	summary["endpoints"] = bl.metrics

	bl.WithFields(summary).Info("Request batch summary (200 status codes)")

	// Reset batch
	bl.metrics = make(map[string]*RequestMetrics)
	bl.batchCount = 0
}

// FlushPending forces a flush of any pending batch data
func (bl *BatchLogger) FlushPending() {
	bl.mutex.Lock()
	defer bl.mutex.Unlock()
	bl.flushBatch()
}

// WithContext returns a logger with common context fields
func WithContext(log *BatchLogger, fields map[string]interface{}) *logrus.Entry {
	return log.WithFields(fields)
}
