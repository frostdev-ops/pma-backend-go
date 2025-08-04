package speech

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// PiperCluster manages multiple Piper TTS daemon instances for parallel processing
type PiperCluster struct {
	workers         []*PiperWorker
	chunkQueue      chan ChunkTask
	resultCollector chan ChunkResult
	errorHandler    *ErrorRecovery
	loadBalancer    *LoadBalancer
	healthMonitor   *HealthMonitor
	logger          *logrus.Logger
	config          *ClusterConfig
	mu              sync.RWMutex
	isRunning       bool
	shutdownChan    chan struct{}
	wg              sync.WaitGroup
	metrics         *ClusterMetrics
}

// ClusterConfig holds configuration for the Piper cluster
type ClusterConfig struct {
	BasePort            int           `json:"base_port"`
	PortRange           int           `json:"port_range"`
	MinWorkers          int           `json:"min_workers"`
	MaxWorkers          int           `json:"max_workers"`
	DefaultWorkers      int           `json:"default_workers"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	WorkerTimeout       time.Duration `json:"worker_timeout"`
	AutoScaling         bool          `json:"auto_scaling"`
	QueueBuffer         int           `json:"queue_buffer"`
	ResultBuffer        int           `json:"result_buffer"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	ScaleUpThreshold    float64       `json:"scale_up_threshold"`    // Queue utilization %
	ScaleDownThreshold  float64       `json:"scale_down_threshold"`  // Queue utilization %
	LoadBalanceStrategy string        `json:"load_balance_strategy"` // round_robin, least_busy, fastest
}

// PiperWorker represents a single Piper TTS daemon instance
type PiperWorker struct {
	ID              int            `json:"id"`
	Port            int            `json:"port"`
	URL             string         `json:"url"`
	Status          WorkerStatus   `json:"status"`
	LastUsed        time.Time      `json:"last_used"`
	ProcessingQueue chan ChunkTask `json:"-"`
	httpClient      *http.Client   `json:"-"`
	ProcessingCount int64          `json:"processing_count"`
	TotalProcessed  int64          `json:"total_processed"`
	TotalErrors     int64          `json:"total_errors"`
	AverageLatency  float64        `json:"average_latency"`
	mu              sync.RWMutex   `json:"-"`
	HealthyCount    int64          `json:"healthy_count"`
	UnhealthyCount  int64          `json:"unhealthy_count"`
	LastHealthCheck time.Time      `json:"last_health_check"`
}

// WorkerStatus represents the current status of a worker
type WorkerStatus string

const (
	WorkerStatusAvailable WorkerStatus = "available"
	WorkerStatusBusy      WorkerStatus = "busy"
	WorkerStatusError     WorkerStatus = "error"
	WorkerStatusOffline   WorkerStatus = "offline"
	WorkerStatusStarting  WorkerStatus = "starting"
	WorkerStatusStopping  WorkerStatus = "stopping"
)

// ChunkTask represents a TTS processing task for a worker
type ChunkTask struct {
	ID         string                 `json:"id"`
	RequestID  string                 `json:"request_id"`
	Chunk      TextChunk              `json:"chunk"`
	Voice      string                 `json:"voice"`
	Language   string                 `json:"language"`
	Speed      float32                `json:"speed"`
	Priority   int                    `json:"priority"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	CreatedAt  time.Time              `json:"created_at"`
	StartedAt  time.Time              `json:"started_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ChunkResult represents the result of a TTS processing task
type ChunkResult struct {
	TaskID       string                 `json:"task_id"`
	RequestID    string                 `json:"request_id"`
	ChunkIndex   int                    `json:"chunk_index"`
	AudioData    []byte                 `json:"audio_data"`
	AudioFile    string                 `json:"audio_file"`
	Duration     time.Duration          `json:"duration"`
	ProcessTime  time.Duration          `json:"process_time"`
	WorkerID     int                    `json:"worker_id"`
	Success      bool                   `json:"success"`
	Error        error                  `json:"error"`
	ErrorMessage string                 `json:"error_message"`
	Metadata     map[string]interface{} `json:"metadata"`
	CompletedAt  time.Time              `json:"completed_at"`
}

// ErrorRecovery handles worker failures and recovery
type ErrorRecovery struct {
	maxRetries     int
	retryDelay     time.Duration
	backoffFactor  float64
	circuitBreaker map[int]*CircuitBreaker
	mu             sync.RWMutex
	logger         *logrus.Logger
}

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	failureCount    int64
	successCount    int64
	lastFailureTime time.Time
	state           CircuitState
	threshold       int64
	timeout         time.Duration
	mu              sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// LoadBalancer manages work distribution across workers
type LoadBalancer struct {
	strategy   string
	lastWorker int64
	mu         sync.RWMutex
	logger     *logrus.Logger
}

// HealthMonitor continuously monitors worker health
type HealthMonitor struct {
	interval       time.Duration
	timeout        time.Duration
	unhealthyLimit int
	mu             sync.RWMutex
	logger         *logrus.Logger
}

// ClusterMetrics tracks performance metrics
type ClusterMetrics struct {
	TotalTasks     int64     `json:"total_tasks"`
	CompletedTasks int64     `json:"completed_tasks"`
	FailedTasks    int64     `json:"failed_tasks"`
	QueueLength    int64     `json:"queue_length"`
	ActiveWorkers  int64     `json:"active_workers"`
	AverageLatency float64   `json:"average_latency"`
	Throughput     float64   `json:"throughput"`
	StartTime      time.Time `json:"start_time"`
	LastUpdate     time.Time `json:"last_update"`
	mu             sync.RWMutex
}

// NewPiperCluster creates a new Piper TTS cluster
func NewPiperCluster(config *ClusterConfig, logger *logrus.Logger) (*PiperCluster, error) {
	if config == nil {
		return nil, fmt.Errorf("cluster config cannot be nil")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Validate configuration
	if err := validateClusterConfig(config); err != nil {
		return nil, fmt.Errorf("invalid cluster config: %w", err)
	}

	cluster := &PiperCluster{
		workers:         make([]*PiperWorker, 0, config.MaxWorkers),
		chunkQueue:      make(chan ChunkTask, config.QueueBuffer),
		resultCollector: make(chan ChunkResult, config.ResultBuffer),
		logger:          logger,
		config:          config,
		shutdownChan:    make(chan struct{}),
		metrics: &ClusterMetrics{
			StartTime: time.Now(),
		},
	}

	// Initialize components
	cluster.errorHandler = NewErrorRecovery(config.MaxRetries, config.RetryDelay, logger)
	cluster.loadBalancer = NewLoadBalancer(config.LoadBalanceStrategy, logger)
	cluster.healthMonitor = NewHealthMonitor(config.HealthCheckInterval, config.WorkerTimeout, logger)

	return cluster, nil
}

// Start initializes and starts the cluster
func (pc *PiperCluster) Start(ctx context.Context) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.isRunning {
		return fmt.Errorf("cluster is already running")
	}

	pc.logger.WithFields(logrus.Fields{
		"min_workers":     pc.config.MinWorkers,
		"default_workers": pc.config.DefaultWorkers,
		"max_workers":     pc.config.MaxWorkers,
	}).Info("Starting Piper TTS cluster")

	// Start with default number of workers
	for i := 0; i < pc.config.DefaultWorkers; i++ {
		if err := pc.addWorker(); err != nil {
			pc.logger.WithError(err).WithField("worker_id", i).Error("Failed to add worker during startup")
			// Continue with other workers
		}
	}

	// Start background goroutines
	pc.isRunning = true
	pc.wg.Add(4)

	go pc.runTaskDispatcher(ctx)
	go pc.runResultCollector(ctx)
	go pc.runHealthMonitor(ctx)
	go pc.runAutoScaler(ctx)

	pc.logger.WithField("active_workers", len(pc.workers)).Info("Piper TTS cluster started successfully")

	return nil
}

// Stop gracefully shuts down the cluster
func (pc *PiperCluster) Stop(ctx context.Context) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isRunning {
		return nil
	}

	pc.logger.Info("Stopping Piper TTS cluster")
	pc.isRunning = false

	// Signal shutdown
	close(pc.shutdownChan)

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		pc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		pc.logger.Info("All cluster goroutines stopped")
	case <-ctx.Done():
		pc.logger.Warn("Cluster shutdown timed out")
	}

	// Stop all workers
	for _, worker := range pc.workers {
		pc.stopWorker(worker)
	}

	// Close channels
	close(pc.chunkQueue)
	close(pc.resultCollector)

	pc.logger.Info("Piper TTS cluster stopped successfully")
	return nil
}

// ProcessChunks processes multiple text chunks in parallel
func (pc *PiperCluster) ProcessChunks(ctx context.Context, chunks []TextChunk, voice, language string, speed float32) (chan ChunkResult, error) {
	if !pc.isRunning {
		return nil, fmt.Errorf("cluster is not running")
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to process")
	}

	pc.logger.WithFields(logrus.Fields{
		"chunk_count": len(chunks),
		"voice":       voice,
		"language":    language,
		"speed":       speed,
	}).Info("Processing text chunks in parallel")

	// Create result channel
	resultChan := make(chan ChunkResult, len(chunks))

	// Generate request ID
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	// Submit all chunks for processing
	for _, chunk := range chunks {
		task := ChunkTask{
			ID:        fmt.Sprintf("%s_chunk_%d", requestID, chunk.Index),
			RequestID: requestID,
			Chunk:     chunk,
			Voice:     voice,
			Language:  language,
			Speed:     speed,
			Priority:  chunk.Priority,
			Timeout:   pc.config.WorkerTimeout,
			CreatedAt: time.Now(),
			Metadata:  chunk.Metadata,
		}

		select {
		case pc.chunkQueue <- task:
			atomic.AddInt64(&pc.metrics.TotalTasks, 1)
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Queue is full
			pc.logger.Warn("Chunk queue is full, task will be delayed")
			select {
			case pc.chunkQueue <- task:
				atomic.AddInt64(&pc.metrics.TotalTasks, 1)
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	// Start a goroutine to collect results for this request
	go pc.collectRequestResults(ctx, requestID, len(chunks), resultChan)

	return resultChan, nil
}

// GetWorkerStats returns statistics for all workers
func (pc *PiperCluster) GetWorkerStats() []map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	stats := make([]map[string]interface{}, len(pc.workers))
	for i, worker := range pc.workers {
		worker.mu.RLock()
		stats[i] = map[string]interface{}{
			"id":                worker.ID,
			"port":              worker.Port,
			"status":            worker.Status,
			"processing_count":  worker.ProcessingCount,
			"total_processed":   worker.TotalProcessed,
			"total_errors":      worker.TotalErrors,
			"average_latency":   worker.AverageLatency,
			"healthy_count":     worker.HealthyCount,
			"unhealthy_count":   worker.UnhealthyCount,
			"last_used":         worker.LastUsed,
			"last_health_check": worker.LastHealthCheck,
		}
		worker.mu.RUnlock()
	}
	return stats
}

// GetClusterStats returns overall cluster statistics
func (pc *PiperCluster) GetClusterStats() map[string]interface{} {
	pc.metrics.mu.RLock()
	defer pc.metrics.mu.RUnlock()

	uptime := time.Since(pc.metrics.StartTime)
	queueLength := int64(len(pc.chunkQueue))
	activeWorkers := pc.countActiveWorkers()

	return map[string]interface{}{
		"total_tasks":       pc.metrics.TotalTasks,
		"completed_tasks":   pc.metrics.CompletedTasks,
		"failed_tasks":      pc.metrics.FailedTasks,
		"queue_length":      queueLength,
		"active_workers":    activeWorkers,
		"total_workers":     len(pc.workers),
		"average_latency":   pc.metrics.AverageLatency,
		"throughput":        pc.metrics.Throughput,
		"uptime":            uptime.String(),
		"start_time":        pc.metrics.StartTime,
		"last_update":       pc.metrics.LastUpdate,
		"queue_capacity":    cap(pc.chunkQueue),
		"queue_utilization": float64(queueLength) / float64(cap(pc.chunkQueue)),
	}
}

// Internal methods

func (pc *PiperCluster) addWorker() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if len(pc.workers) >= pc.config.MaxWorkers {
		return fmt.Errorf("maximum number of workers reached: %d", pc.config.MaxWorkers)
	}

	workerID := len(pc.workers)
	port := pc.config.BasePort + workerID

	// Check if port is in range
	if port >= pc.config.BasePort+pc.config.PortRange {
		return fmt.Errorf("port %d exceeds allowed range", port)
	}

	worker := &PiperWorker{
		ID:              workerID,
		Port:            port,
		URL:             fmt.Sprintf("http://127.0.0.1:%d", port),
		Status:          WorkerStatusStarting,
		ProcessingQueue: make(chan ChunkTask, 10),
		httpClient: &http.Client{
			Timeout: pc.config.WorkerTimeout,
		},
		LastHealthCheck: time.Now(),
	}

	pc.workers = append(pc.workers, worker)

	// Start worker goroutine
	pc.wg.Add(1)
	go pc.runWorker(worker)

	pc.logger.WithFields(logrus.Fields{
		"worker_id": workerID,
		"port":      port,
		"url":       worker.URL,
	}).Info("Added new Piper worker")

	return nil
}

func (pc *PiperCluster) stopWorker(worker *PiperWorker) {
	worker.mu.Lock()
	defer worker.mu.Unlock()

	if worker.Status == WorkerStatusOffline {
		return
	}

	worker.Status = WorkerStatusStopping
	close(worker.ProcessingQueue)

	pc.logger.WithField("worker_id", worker.ID).Info("Stopped Piper worker")
}

func (pc *PiperCluster) runTaskDispatcher(ctx context.Context) {
	defer pc.wg.Done()

	pc.logger.Info("Starting task dispatcher")

	for {
		select {
		case task := <-pc.chunkQueue:
			pc.dispatchTask(task)
		case <-pc.shutdownChan:
			pc.logger.Info("Task dispatcher shutting down")
			return
		case <-ctx.Done():
			pc.logger.Info("Task dispatcher canceled by context")
			return
		}
	}
}

func (pc *PiperCluster) dispatchTask(task ChunkTask) {
	// Select best worker using load balancer
	worker := pc.loadBalancer.SelectWorker(pc.workers)
	if worker == nil {
		pc.logger.Error("No available workers for task")
		// TODO: Handle no workers available (queue for retry, scale up, etc.)
		return
	}

	// Submit task to worker
	select {
	case worker.ProcessingQueue <- task:
		pc.logger.WithFields(logrus.Fields{
			"task_id":     task.ID,
			"worker_id":   worker.ID,
			"chunk_index": task.Chunk.Index,
		}).Debug("Dispatched task to worker")
	default:
		pc.logger.WithFields(logrus.Fields{
			"task_id":   task.ID,
			"worker_id": worker.ID,
		}).Warn("Worker queue is full, will retry")
		// TODO: Implement retry logic or queue overflow handling
	}
}

func (pc *PiperCluster) runWorker(worker *PiperWorker) {
	defer pc.wg.Done()

	pc.logger.WithField("worker_id", worker.ID).Info("Starting worker")

	// Set worker as available
	worker.mu.Lock()
	worker.Status = WorkerStatusAvailable
	worker.mu.Unlock()

	for {
		select {
		case task, ok := <-worker.ProcessingQueue:
			if !ok {
				pc.logger.WithField("worker_id", worker.ID).Info("Worker queue closed")
				return
			}
			pc.processTask(worker, task)
		case <-pc.shutdownChan:
			pc.logger.WithField("worker_id", worker.ID).Info("Worker shutting down")
			return
		}
	}
}

func (pc *PiperCluster) processTask(worker *PiperWorker, task ChunkTask) {
	startTime := time.Now()

	// Update worker status
	worker.mu.Lock()
	worker.Status = WorkerStatusBusy
	worker.LastUsed = startTime
	atomic.AddInt64(&worker.ProcessingCount, 1)
	worker.mu.Unlock()

	defer func() {
		worker.mu.Lock()
		worker.Status = WorkerStatusAvailable
		atomic.AddInt64(&worker.ProcessingCount, -1)
		atomic.AddInt64(&worker.TotalProcessed, 1)
		worker.mu.Unlock()
	}()

	pc.logger.WithFields(logrus.Fields{
		"worker_id":   worker.ID,
		"task_id":     task.ID,
		"chunk_index": task.Chunk.Index,
	}).Debug("Processing task")

	// Create TTS daemon request
	daemonReq := TTSDaemonRequest{
		Text:         task.Chunk.Content,
		Voice:        task.Voice,
		Language:     task.Language,
		Speed:        task.Speed,
		OutputFormat: "wav",
	}

	// Process the task
	result := pc.callWorkerTTS(worker, task, daemonReq)
	result.ProcessTime = time.Since(startTime)
	result.CompletedAt = time.Now()
	result.WorkerID = worker.ID

	// Update worker metrics
	pc.updateWorkerMetrics(worker, result)

	// Send result
	select {
	case pc.resultCollector <- result:
	default:
		pc.logger.WithField("task_id", task.ID).Warn("Result collector channel is full")
	}
}

func (pc *PiperCluster) callWorkerTTS(worker *PiperWorker, task ChunkTask, daemonReq TTSDaemonRequest) ChunkResult {
	// Implementation of the actual TTS call to the worker daemon
	// This would be similar to the existing textToSpeechDaemon method
	// but adapted for the worker context

	result := ChunkResult{
		TaskID:     task.ID,
		RequestID:  task.RequestID,
		ChunkIndex: task.Chunk.Index,
		Success:    false,
	}

	// Marshal request
	reqBody, err := json.Marshal(daemonReq)
	if err != nil {
		result.Error = err
		result.ErrorMessage = fmt.Sprintf("failed to marshal request: %v", err)
		return result
	}

	// Create HTTP request with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()

	url := worker.URL + "/tts"
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		result.Error = err
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		return result
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := worker.httpClient.Do(req)
	if err != nil {
		result.Error = err
		result.ErrorMessage = fmt.Sprintf("TTS request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		result.ErrorMessage = fmt.Sprintf("failed to read response: %v", err)
		return result
	}

	// Parse response
	var daemonResp TTSDaemonResponse
	if err := json.Unmarshal(respBody, &daemonResp); err != nil {
		result.Error = err
		result.ErrorMessage = fmt.Sprintf("failed to parse response: %v", err)
		return result
	}

	if !daemonResp.Success {
		result.Error = fmt.Errorf("TTS daemon error: %s", daemonResp.Error)
		result.ErrorMessage = daemonResp.Error
		return result
	}

	// Success
	result.Success = true
	result.AudioFile = daemonResp.AudioFile
	result.Duration = time.Duration(daemonResp.Duration * float64(time.Second))

	return result
}

func (pc *PiperCluster) updateWorkerMetrics(worker *PiperWorker, result ChunkResult) {
	worker.mu.Lock()
	defer worker.mu.Unlock()

	if result.Success {
		// Update average latency using exponential moving average
		alpha := 0.1 // Smoothing factor
		latency := float64(result.ProcessTime.Milliseconds())
		if worker.AverageLatency == 0 {
			worker.AverageLatency = latency
		} else {
			worker.AverageLatency = alpha*latency + (1-alpha)*worker.AverageLatency
		}
	} else {
		atomic.AddInt64(&worker.TotalErrors, 1)
	}
}

func (pc *PiperCluster) runResultCollector(ctx context.Context) {
	defer pc.wg.Done()

	pc.logger.Info("Starting result collector")

	for {
		select {
		case result := <-pc.resultCollector:
			pc.handleResult(result)
		case <-pc.shutdownChan:
			pc.logger.Info("Result collector shutting down")
			return
		case <-ctx.Done():
			pc.logger.Info("Result collector canceled by context")
			return
		}
	}
}

func (pc *PiperCluster) handleResult(result ChunkResult) {
	if result.Success {
		atomic.AddInt64(&pc.metrics.CompletedTasks, 1)
		pc.logger.WithFields(logrus.Fields{
			"task_id":      result.TaskID,
			"chunk_index":  result.ChunkIndex,
			"worker_id":    result.WorkerID,
			"process_time": result.ProcessTime,
		}).Debug("Task completed successfully")
	} else {
		atomic.AddInt64(&pc.metrics.FailedTasks, 1)
		pc.logger.WithFields(logrus.Fields{
			"task_id":     result.TaskID,
			"chunk_index": result.ChunkIndex,
			"worker_id":   result.WorkerID,
			"error":       result.ErrorMessage,
		}).Error("Task failed")
	}

	// Update metrics
	pc.updateClusterMetrics()
}

func (pc *PiperCluster) collectRequestResults(ctx context.Context, requestID string, expectedCount int, resultChan chan ChunkResult) {
	defer close(resultChan)

	collected := 0
	timeout := time.NewTimer(time.Minute * 5) // 5 minute timeout for request
	defer timeout.Stop()

	for collected < expectedCount {
		select {
		case result := <-pc.resultCollector:
			if result.RequestID == requestID {
				resultChan <- result
				collected++
			} else {
				// Not our result, put it back (this is a simplified approach)
				// In production, you'd want a more sophisticated routing mechanism
				select {
				case pc.resultCollector <- result:
				default:
					pc.logger.Warn("Dropped result due to channel overflow")
				}
			}
		case <-timeout.C:
			pc.logger.WithField("request_id", requestID).Error("Request timed out")
			return
		case <-ctx.Done():
			pc.logger.WithField("request_id", requestID).Info("Request canceled")
			return
		}
	}

	pc.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"collected":  collected,
		"expected":   expectedCount,
	}).Info("Request results collection completed")
}

func (pc *PiperCluster) runHealthMonitor(ctx context.Context) {
	defer pc.wg.Done()

	pc.logger.Info("Starting health monitor")
	ticker := time.NewTicker(pc.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.checkAllWorkersHealth()
		case <-pc.shutdownChan:
			pc.logger.Info("Health monitor shutting down")
			return
		case <-ctx.Done():
			pc.logger.Info("Health monitor canceled by context")
			return
		}
	}
}

func (pc *PiperCluster) checkAllWorkersHealth() {
	pc.mu.RLock()
	workers := make([]*PiperWorker, len(pc.workers))
	copy(workers, pc.workers)
	pc.mu.RUnlock()

	for _, worker := range workers {
		pc.checkWorkerHealth(worker)
	}
}

func (pc *PiperCluster) checkWorkerHealth(worker *PiperWorker) {
	ctx, cancel := context.WithTimeout(context.Background(), pc.healthMonitor.timeout)
	defer cancel()

	healthy := pc.performHealthCheck(ctx, worker)

	worker.mu.Lock()
	worker.LastHealthCheck = time.Now()
	if healthy {
		atomic.AddInt64(&worker.HealthyCount, 1)
		if worker.Status == WorkerStatusError {
			worker.Status = WorkerStatusAvailable
			pc.logger.WithField("worker_id", worker.ID).Info("Worker recovered")
		}
	} else {
		atomic.AddInt64(&worker.UnhealthyCount, 1)
		if worker.Status != WorkerStatusError {
			worker.Status = WorkerStatusError
			pc.logger.WithField("worker_id", worker.ID).Warn("Worker marked as unhealthy")
		}
	}
	worker.mu.Unlock()
}

func (pc *PiperCluster) performHealthCheck(ctx context.Context, worker *PiperWorker) bool {
	url := worker.URL + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := worker.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (pc *PiperCluster) runAutoScaler(ctx context.Context) {
	defer pc.wg.Done()

	if !pc.config.AutoScaling {
		pc.logger.Info("Auto-scaling is disabled")
		return
	}

	pc.logger.Info("Starting auto-scaler")
	ticker := time.NewTicker(time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pc.performAutoScaling()
		case <-pc.shutdownChan:
			pc.logger.Info("Auto-scaler shutting down")
			return
		case <-ctx.Done():
			pc.logger.Info("Auto-scaler canceled by context")
			return
		}
	}
}

func (pc *PiperCluster) performAutoScaling() {
	queueLength := len(pc.chunkQueue)
	queueCapacity := cap(pc.chunkQueue)
	utilization := float64(queueLength) / float64(queueCapacity)

	activeWorkers := pc.countActiveWorkers()

	pc.logger.WithFields(logrus.Fields{
		"queue_length":   queueLength,
		"queue_capacity": queueCapacity,
		"utilization":    utilization,
		"active_workers": activeWorkers,
		"total_workers":  len(pc.workers),
	}).Debug("Auto-scaling check")

	// Scale up if queue utilization is high and we have capacity
	if utilization > pc.config.ScaleUpThreshold && len(pc.workers) < pc.config.MaxWorkers {
		if err := pc.addWorker(); err != nil {
			pc.logger.WithError(err).Warn("Failed to scale up")
		} else {
			pc.logger.WithField("total_workers", len(pc.workers)).Info("Scaled up workers")
		}
	}

	// Scale down if queue utilization is low and we have more than minimum workers
	if utilization < pc.config.ScaleDownThreshold && len(pc.workers) > pc.config.MinWorkers {
		// Remove the least recently used worker
		pc.removeIdleWorker()
	}
}

func (pc *PiperCluster) countActiveWorkers() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	active := 0
	for _, worker := range pc.workers {
		worker.mu.RLock()
		if worker.Status == WorkerStatusAvailable || worker.Status == WorkerStatusBusy {
			active++
		}
		worker.mu.RUnlock()
	}
	return active
}

func (pc *PiperCluster) removeIdleWorker() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Find the least recently used worker that's not busy
	var oldestWorker *PiperWorker
	oldestTime := time.Now()

	for _, worker := range pc.workers {
		worker.mu.RLock()
		if worker.Status == WorkerStatusAvailable && worker.LastUsed.Before(oldestTime) {
			oldestWorker = worker
			oldestTime = worker.LastUsed
		}
		worker.mu.RUnlock()
	}

	if oldestWorker != nil {
		pc.stopWorker(oldestWorker)
		// Remove from slice
		for i, worker := range pc.workers {
			if worker == oldestWorker {
				pc.workers = append(pc.workers[:i], pc.workers[i+1:]...)
				break
			}
		}
		pc.logger.WithField("worker_id", oldestWorker.ID).Info("Scaled down worker")
	}
}

func (pc *PiperCluster) updateClusterMetrics() {
	pc.metrics.mu.Lock()
	defer pc.metrics.mu.Unlock()

	pc.metrics.LastUpdate = time.Now()

	// Calculate throughput (tasks per second)
	uptime := time.Since(pc.metrics.StartTime)
	if uptime.Seconds() > 0 {
		pc.metrics.Throughput = float64(pc.metrics.CompletedTasks) / uptime.Seconds()
	}

	// Calculate average latency across all workers
	pc.mu.RLock()
	totalLatency := 0.0
	activeWorkers := 0
	for _, worker := range pc.workers {
		worker.mu.RLock()
		if worker.AverageLatency > 0 {
			totalLatency += worker.AverageLatency
			activeWorkers++
		}
		worker.mu.RUnlock()
	}
	pc.mu.RUnlock()

	if activeWorkers > 0 {
		pc.metrics.AverageLatency = totalLatency / float64(activeWorkers)
	}
}

// Utility functions and helpers

func validateClusterConfig(config *ClusterConfig) error {
	if config.BasePort <= 0 || config.BasePort > 65535 {
		return fmt.Errorf("invalid base port: %d", config.BasePort)
	}
	if config.PortRange <= 0 {
		return fmt.Errorf("invalid port range: %d", config.PortRange)
	}
	if config.MinWorkers < 1 {
		return fmt.Errorf("min workers must be at least 1")
	}
	if config.MaxWorkers < config.MinWorkers {
		return fmt.Errorf("max workers must be >= min workers")
	}
	if config.DefaultWorkers < config.MinWorkers || config.DefaultWorkers > config.MaxWorkers {
		return fmt.Errorf("default workers must be between min and max")
	}
	if config.HealthCheckInterval <= 0 {
		return fmt.Errorf("health check interval must be positive")
	}
	if config.WorkerTimeout <= 0 {
		return fmt.Errorf("worker timeout must be positive")
	}
	return nil
}

func NewErrorRecovery(maxRetries int, retryDelay time.Duration, logger *logrus.Logger) *ErrorRecovery {
	return &ErrorRecovery{
		maxRetries:     maxRetries,
		retryDelay:     retryDelay,
		backoffFactor:  1.5,
		circuitBreaker: make(map[int]*CircuitBreaker),
		logger:         logger,
	}
}

func NewLoadBalancer(strategy string, logger *logrus.Logger) *LoadBalancer {
	if strategy == "" {
		strategy = "round_robin"
	}
	return &LoadBalancer{
		strategy: strategy,
		logger:   logger,
	}
}

func (lb *LoadBalancer) SelectWorker(workers []*PiperWorker) *PiperWorker {
	availableWorkers := make([]*PiperWorker, 0, len(workers))

	// Filter available workers
	for _, worker := range workers {
		worker.mu.RLock()
		if worker.Status == WorkerStatusAvailable {
			availableWorkers = append(availableWorkers, worker)
		}
		worker.mu.RUnlock()
	}

	if len(availableWorkers) == 0 {
		return nil
	}

	switch lb.strategy {
	case "least_busy":
		return lb.selectLeastBusy(availableWorkers)
	case "fastest":
		return lb.selectFastest(availableWorkers)
	default: // round_robin
		return lb.selectRoundRobin(availableWorkers)
	}
}

func (lb *LoadBalancer) selectRoundRobin(workers []*PiperWorker) *PiperWorker {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	index := atomic.AddInt64(&lb.lastWorker, 1) % int64(len(workers))
	return workers[index]
}

func (lb *LoadBalancer) selectLeastBusy(workers []*PiperWorker) *PiperWorker {
	var bestWorker *PiperWorker
	minLoad := int64(9999999)

	for _, worker := range workers {
		worker.mu.RLock()
		load := atomic.LoadInt64(&worker.ProcessingCount)
		worker.mu.RUnlock()

		if load < minLoad {
			minLoad = load
			bestWorker = worker
		}
	}

	return bestWorker
}

func (lb *LoadBalancer) selectFastest(workers []*PiperWorker) *PiperWorker {
	var bestWorker *PiperWorker
	bestLatency := float64(9999999)

	for _, worker := range workers {
		worker.mu.RLock()
		latency := worker.AverageLatency
		worker.mu.RUnlock()

		if latency > 0 && latency < bestLatency {
			bestLatency = latency
			bestWorker = worker
		}
	}

	// If no worker has latency data, fall back to round robin
	if bestWorker == nil {
		return lb.selectRoundRobin(workers)
	}

	return bestWorker
}

func NewHealthMonitor(interval, timeout time.Duration, logger *logrus.Logger) *HealthMonitor {
	return &HealthMonitor{
		interval:       interval,
		timeout:        timeout,
		unhealthyLimit: 3,
		logger:         logger,
	}
}
