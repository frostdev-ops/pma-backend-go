package speech

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// StreamingTTSService extends the basic speech service with multi-instance streaming capabilities
type StreamingTTSService struct {
	*Service        // Embed the existing service for backward compatibility
	textSplitter    *TextSplitter
	clusterManager  *PiperCluster
	audioAssembler  *AudioAssembler
	streamManager   *StreamManager
	cacheManager    *TTSCache
	streamingConfig *StreamingConfig
	metrics         *StreamingMetrics
	mu              sync.RWMutex
	isInitialized   bool
}

// NewStreamingTTSService creates a new streaming TTS service
func NewStreamingTTSService(cfg *config.SpeechConfig, logger *logrus.Logger) (*StreamingTTSService, error) {
	// Create the base service first
	baseService, err := NewService(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base service: %w", err)
	}

	streamingService := &StreamingTTSService{
		Service:         baseService,
		streamingConfig: DefaultStreamingConfig,
		metrics:         NewStreamingMetrics(),
	}

	// Initialize streaming components
	if err := streamingService.initializeComponents(cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to initialize streaming components: %w", err)
	}

	return streamingService, nil
}

// initializeComponents sets up all the streaming components
func (sts *StreamingTTSService) initializeComponents(cfg *config.SpeechConfig, logger *logrus.Logger) error {
	// Initialize text splitter
	sts.textSplitter = NewTextSplitter()
	sts.textSplitter.MaxChunkSize = sts.streamingConfig.MaxChunkSize
	sts.textSplitter.MinChunkSize = sts.streamingConfig.MinChunkSize
	sts.textSplitter.ContextOverlap = sts.streamingConfig.ContextOverlap

	// Skip cluster manager initialization - use direct daemon processing instead
	// We'll process chunks sequentially through the existing TTS daemon on port 8001
	logger.Info("Using single-daemon intelligent processing mode")

	// Skip audio assembler initialization for single-daemon mode
	// Audio will be handled per-chunk without assembly

	// Initialize stream manager
	sts.streamManager = &StreamManager{
		streams:    make(map[string]*StreamSession),
		maxStreams: sts.streamingConfig.MaxConcurrentStreams,
		logger:     logger,
	}

	// Initialize cache manager
	sts.cacheManager = NewTTSCache(logger)

	sts.isInitialized = true
	logger.Info("Streaming TTS service initialized successfully")

	return nil
}

// StartStreaming initializes the streaming infrastructure
func (sts *StreamingTTSService) StartStreaming(ctx context.Context) error {
	sts.mu.Lock()
	defer sts.mu.Unlock()

	if !sts.isInitialized {
		return fmt.Errorf("service not initialized")
	}

	// Skip cluster startup - using single-daemon processing
	sts.logger.Info("Single-daemon streaming mode - no cluster to start")

	// Start background cleanup routine
	go sts.runCleanupRoutine(ctx)

	// Start metrics collection routine
	go sts.runMetricsCollector(ctx)

	sts.logger.Info("Streaming TTS service started successfully")
	return nil
}

// StopStreaming gracefully shuts down the streaming infrastructure
func (sts *StreamingTTSService) StopStreaming(ctx context.Context) error {
	sts.mu.Lock()
	defer sts.mu.Unlock()

	if sts.clusterManager != nil {
		if err := sts.clusterManager.Stop(ctx); err != nil {
			sts.logger.WithError(err).Error("Failed to stop cluster manager")
		}
	}

	// Cancel all active streams
	sts.streamManager.mu.Lock()
	for _, session := range sts.streamManager.streams {
		if session.CancelFunc != nil {
			session.CancelFunc()
		}
	}
	sts.streamManager.streams = make(map[string]*StreamSession)
	sts.streamManager.mu.Unlock()

	sts.logger.Info("Streaming TTS service stopped")
	return nil
}

// TextToSpeechStreaming processes text using multi-instance streaming
func (sts *StreamingTTSService) TextToSpeechStreaming(ctx context.Context, req *StreamingTTSRequest) (*StreamingTTSResponse, error) {
	// Check if cluster processing is available and enabled
	if sts.clusterManager != nil && sts.clusterManager.IsAvailable() {
		sts.logger.Info("🌐 Using cluster-based processing")
		return sts.processClusterBased(ctx, req)
	}

	// Fallback to single instance processing
	sts.logger.Info("🔄 Using single-instance fallback processing")
	return sts.fallbackToSingleInstance(ctx, req)
}

// createStreamSession creates a new streaming session
func (sts *StreamingTTSService) createStreamSession(ctx context.Context, req *StreamingTTSRequest) (*StreamSession, error) {
	// Check if we've reached the maximum number of concurrent streams
	sts.streamManager.mu.Lock()
	if len(sts.streamManager.streams) >= sts.streamManager.maxStreams {
		sts.streamManager.mu.Unlock()
		return nil, fmt.Errorf("maximum concurrent streams reached: %d", sts.streamManager.maxStreams)
	}

	// Generate stream ID if not provided
	streamID := req.RequestID
	if streamID == "" {
		streamID = generateRequestID()
	}

	// Create session context with timeout
	sessionCtx, cancelFunc := context.WithTimeout(ctx, sts.streamingConfig.StreamTimeout)

	// Create response channels
	response := &StreamingTTSResponse{
		StreamID:       streamID,
		AudioStream:    make(chan AudioChunk, sts.streamingConfig.BufferSize),
		ProgressStream: make(chan ProgressUpdate, 10),
		ErrorStream:    make(chan error, 5),
		CompletionChan: make(chan StreamCompletion, 1),
	}

	// Create session
	session := &StreamSession{
		ID:                streamID,
		Request:           req,
		Response:          response,
		Context:           sessionCtx,
		CancelFunc:        cancelFunc,
		StartTime:         time.Now(),
		LastActivity:      time.Now(),
		State:             StreamStateInitializing,
		WorkerAssignments: make(map[int]int),
	}

	// Store session
	sts.streamManager.streams[streamID] = session
	sts.streamManager.mu.Unlock()

	sts.logger.WithFields(logrus.Fields{
		"stream_id":      streamID,
		"text_length":    len(req.Text),
		"active_streams": len(sts.streamManager.streams),
	}).Info("Created new streaming session")

	return session, nil
}

// processStreamingPipeline runs the complete streaming processing pipeline
func (sts *StreamingTTSService) processStreamingPipeline(session *StreamSession) {
	defer func() {
		// Cleanup on completion or error
		sts.cleanupSession(session)
	}()

	startTime := time.Now()
	statistics := ProcessingStatistics{
		StartTime: startTime,
	}

	// Phase 1: Text Splitting
	sts.updateSessionState(session, StreamStateSplitting)
	sts.sendProgressUpdate(session, PhaseInitializing, 0, 0)

	splitStart := time.Now()
	chunks, err := sts.textSplitter.SplitIntelligently(session.Request.Text)
	if err != nil {
		sts.handleSessionError(session, fmt.Errorf("text splitting failed: %w", err))
		return
	}
	statistics.TextSplittingTime = time.Since(splitStart)
	statistics.TotalChunks = len(chunks)

	session.mu.Lock()
	session.Chunks = chunks
	session.mu.Unlock()

	// Update response with total chunks
	session.Response.TotalChunks = len(chunks)

	sts.logger.WithFields(logrus.Fields{
		"stream_id":      session.ID,
		"total_chunks":   len(chunks),
		"splitting_time": statistics.TextSplittingTime,
	}).Info("Text splitting completed")

	// Send progress update
	sts.sendProgressUpdate(session, PhaseSplitting, 0, len(chunks))

	// Phase 2: Parallel Processing
	sts.updateSessionState(session, StreamStateProcessing)
	sts.sendProgressUpdate(session, PhaseProcessing, 0, len(chunks))

	processingStart := time.Now()
	resultChan, err := sts.clusterManager.ProcessChunks(
		session.Context,
		chunks,
		session.Request.Voice,
		session.Request.Language,
		session.Request.Speed,
	)
	if err != nil {
		sts.handleSessionError(session, fmt.Errorf("failed to start parallel processing: %w", err))
		return
	}

	// Collect results from parallel processing
	processedChunks := make([]AudioSegment, len(chunks))
	completedCount := 0

	go func() {
		for result := range resultChan {
			if result.Success {
				// Convert ChunkResult to AudioSegment
				audioSegment := AudioSegment{
					Data:          result.AudioData,
					Duration:      result.Duration,
					Index:         result.ChunkIndex,
					FilePath:      result.AudioFile,
					SampleRate:    22050, // Default sample rate
					Format:        "wav",
					Channels:      1,
					BitsPerSample: 16,
					IsProcessed:   true,
					Metadata:      result.Metadata,
				}

				processedChunks[result.ChunkIndex] = audioSegment
				completedCount++
				statistics.SuccessfulChunks++

				// Send progress update
				sts.sendProgressUpdate(session, PhaseProcessing, completedCount, len(chunks))

				// Stream individual chunks if streaming mode is enabled
				if session.Request.StreamingMode {
					audioChunk := AudioChunk{
						ChunkIndex: result.ChunkIndex,
						AudioData:  result.AudioData,
						AudioFile:  result.AudioFile,
						Duration:   result.Duration,
						Timestamp:  time.Now(),
						IsComplete: false,
						Metadata:   result.Metadata,
					}

					select {
					case session.Response.AudioStream <- audioChunk:
					case <-session.Context.Done():
						return
					}
				}
			} else {
				statistics.FailedChunks++
				sts.logger.WithFields(logrus.Fields{
					"stream_id":   session.ID,
					"chunk_index": result.ChunkIndex,
					"error":       result.ErrorMessage,
				}).Error("Chunk processing failed")
			}

			// Check if we're done
			if completedCount+statistics.FailedChunks >= len(chunks) {
				break
			}
		}

		statistics.ParallelProcessingTime = time.Since(processingStart)

		// Phase 3: Audio Assembly
		sts.updateSessionState(session, StreamStateAssembling)
		sts.sendProgressUpdate(session, PhaseAssembling, completedCount, len(chunks))

		assemblyStart := time.Now()
		finalAudio, err := sts.audioAssembler.AssembleAudioStream(processedChunks)
		if err != nil {
			sts.handleSessionError(session, fmt.Errorf("audio assembly failed: %w", err))
			return
		}
		statistics.AudioAssemblyTime = time.Since(assemblyStart)

		// Save final audio file
		outputPath := filepath.Join(sts.config.TTS.OutputDir, fmt.Sprintf("stream_%s.wav", session.ID))
		if err := os.WriteFile(outputPath, finalAudio, 0644); err != nil {
			sts.logger.WithError(err).Error("Failed to save final audio file")
		} else {
			session.Response.FinalAudioFile = outputPath
		}

		// Calculate final statistics
		statistics.EndTime = time.Now()
		statistics.TotalProcessingTime = time.Since(startTime)
		statistics.AverageChunkSize = float64(len(session.Request.Text)) / float64(len(chunks))
		if statistics.SuccessfulChunks > 0 {
			statistics.AverageProcessingTime = statistics.ParallelProcessingTime / time.Duration(statistics.SuccessfulChunks)
		}

		// Send completion
		completion := StreamCompletion{
			StreamID:        session.ID,
			Success:         statistics.FailedChunks == 0,
			TotalChunks:     statistics.TotalChunks,
			ProcessedChunks: statistics.SuccessfulChunks,
			FinalAudioFile:  session.Response.FinalAudioFile,
			TotalDuration:   sts.calculateTotalDuration(processedChunks),
			ProcessingTime:  statistics.TotalProcessingTime,
			Statistics:      statistics,
			Timestamp:       time.Now(),
		}

		if statistics.FailedChunks > 0 {
			completion.Error = fmt.Errorf("failed to process %d out of %d chunks", statistics.FailedChunks, statistics.TotalChunks)
			completion.ErrorMessage = completion.Error.Error()
		}

		session.Response.ProcessingStats = statistics

		select {
		case session.Response.CompletionChan <- completion:
		case <-session.Context.Done():
		}

		sts.updateSessionState(session, StreamStateCompleted)
		sts.sendProgressUpdate(session, PhaseCompleted, statistics.SuccessfulChunks, statistics.TotalChunks)

		// Update metrics
		if completion.Success {
			sts.metrics.IncrementCompletedStreams()
		} else {
			sts.metrics.IncrementFailedStreams()
		}
		sts.metrics.UpdateLatency(statistics.TotalProcessingTime)

		sts.logger.WithFields(logrus.Fields{
			"stream_id":             session.ID,
			"total_processing_time": statistics.TotalProcessingTime,
			"successful_chunks":     statistics.SuccessfulChunks,
			"failed_chunks":         statistics.FailedChunks,
			"final_audio_size":      len(finalAudio),
		}).Info("Streaming TTS processing completed")
	}()
}

// Helper methods

func (sts *StreamingTTSService) validateStreamingRequest(req *StreamingTTSRequest) error {
	if req.Text == "" {
		return fmt.Errorf("text is required")
	}
	if len(req.Text) > 10000 { // 10K character limit for streaming
		return fmt.Errorf("text too long for streaming: %d characters (max: 10000)", len(req.Text))
	}
	if req.Quality != "" {
		if _, exists := sts.streamingConfig.QualityPresets[req.Quality]; !exists {
			return fmt.Errorf("invalid quality preset: %s", req.Quality)
		}
	}
	return nil
}

func (sts *StreamingTTSService) updateSessionState(session *StreamSession, state StreamState) {
	session.mu.Lock()
	session.State = state
	session.LastActivity = time.Now()
	session.mu.Unlock()
}

func (sts *StreamingTTSService) sendProgressUpdate(session *StreamSession, phase ProcessingPhase, processed, total int) {
	if !session.Request.ProgressUpdates {
		return
	}

	update := ProgressUpdate{
		StreamID:        session.ID,
		TotalChunks:     total,
		ChunksProcessed: processed,
		ChunksCompleted: processed,
		Phase:           phase,
		Timestamp:       time.Now(),
	}

	// Calculate estimated time remaining
	if processed > 0 && total > 0 {
		elapsed := time.Since(session.StartTime)
		rate := float64(processed) / elapsed.Seconds()
		remaining := float64(total-processed) / rate
		update.EstimatedTimeRemaining = time.Duration(remaining) * time.Second
		update.ProcessingSpeed = rate
	}

	select {
	case session.Response.ProgressStream <- update:
	case <-session.Context.Done():
	}
}

func (sts *StreamingTTSService) handleSessionError(session *StreamSession, err error) {
	sts.logger.WithError(err).WithField("stream_id", session.ID).Error("Session error")

	session.mu.Lock()
	session.State = StreamStateError
	session.mu.Unlock()

	// Send error to streams
	select {
	case session.Response.ErrorStream <- err:
	case <-session.Context.Done():
	}

	// Send completion with error
	completion := StreamCompletion{
		StreamID:     session.ID,
		Success:      false,
		Error:        err,
		ErrorMessage: err.Error(),
		Timestamp:    time.Now(),
	}

	select {
	case session.Response.CompletionChan <- completion:
	case <-session.Context.Done():
	}

	sts.metrics.IncrementFailedStreams()
}

func (sts *StreamingTTSService) cleanupSession(session *StreamSession) {
	// Remove from active sessions
	sts.streamManager.mu.Lock()
	delete(sts.streamManager.streams, session.ID)
	sts.streamManager.mu.Unlock()

	// Close channels
	close(session.Response.AudioStream)
	close(session.Response.ProgressStream)
	close(session.Response.ErrorStream)
	close(session.Response.CompletionChan)

	// Cancel context
	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	sts.logger.WithField("stream_id", session.ID).Debug("Session cleaned up")
}

func (sts *StreamingTTSService) calculateTotalDuration(segments []AudioSegment) time.Duration {
	total := time.Duration(0)
	for _, segment := range segments {
		total += segment.Duration
	}
	return total
}

// fallbackToSingleInstance processes the request using a single TTS instance
func (sts *StreamingTTSService) fallbackToSingleInstance(ctx context.Context, req *StreamingTTSRequest) (*StreamingTTSResponse, error) {
	sts.logger.Info("🔄 Using single-instance fallback processing")
	return sts.processSingleChunk(ctx, req)
}

// processClusterBased processes the request using the cluster manager
func (sts *StreamingTTSService) processClusterBased(ctx context.Context, req *StreamingTTSRequest) (*StreamingTTSResponse, error) {
	if !sts.isInitialized {
		return nil, fmt.Errorf("streaming service not initialized")
	}

	// Validate request
	if err := sts.validateStreamingRequest(req); err != nil {
		return nil, fmt.Errorf("invalid streaming request: %w", err)
	}

	sts.logger.WithFields(logrus.Fields{
		"text_length":    len(req.Text),
		"voice":          req.Voice,
		"language":       req.Language,
		"streaming_mode": req.StreamingMode,
		"quality":        req.Quality,
	}).Info("Processing cluster-based TTS request")

	// Split text into chunks
	chunks, err := sts.textSplitter.SplitIntelligently(req.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to split text: %w", err)
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks generated from text")
	}

	sts.logger.WithField("total_chunks", len(chunks)).Info("Text split into chunks")

	// Create response
	streamResponse := &StreamingTTSResponse{
		StreamID:       req.RequestID,
		AudioStream:    make(chan AudioChunk, len(chunks)),
		ProgressStream: make(chan ProgressUpdate, 10),
		ErrorStream:    make(chan error, 5),
		CompletionChan: make(chan StreamCompletion, 1),
	}

	// Process chunks through cluster
	go func() {
		defer close(streamResponse.AudioStream)
		defer close(streamResponse.ProgressStream)
		defer close(streamResponse.ErrorStream)
		defer close(streamResponse.CompletionChan)

		// Convert chunks to cluster format
		clusterChunks := make([]TextChunk, len(chunks))
		for i, chunk := range chunks {
			clusterChunks[i] = TextChunk{
				Index:   i,
				Content: chunk.Content,
			}
		}

		// Process through cluster
		resultChan, err := sts.clusterManager.ProcessChunks(ctx, clusterChunks, req.Voice, req.Language, req.Speed)
		if err != nil {
			select {
			case streamResponse.ErrorStream <- fmt.Errorf("cluster processing failed: %w", err):
			case <-ctx.Done():
			}
			return
		}

		// Collect results
		processedChunks := 0
		for result := range resultChan {
			if result.Success {
				audioChunk := AudioChunk{
					ChunkIndex: result.ChunkIndex,
					AudioData:  result.AudioData,
					AudioFile:  result.AudioFile,
					Duration:   result.Duration,
					Timestamp:  time.Now(),
					IsComplete: true,
					Metadata:   result.Metadata,
				}

				select {
				case streamResponse.AudioStream <- audioChunk:
				case <-ctx.Done():
					return
				}

				processedChunks++
			} else {
				select {
				case streamResponse.ErrorStream <- fmt.Errorf("chunk %d failed: %s", result.ChunkIndex, result.ErrorMessage):
				case <-ctx.Done():
				}
			}

			// Send progress update
			progress := ProgressUpdate{
				TotalChunks:     len(chunks),
				ChunksProcessed: processedChunks,
				Timestamp:       time.Now(),
				Phase:           "processing",
			}

			select {
			case streamResponse.ProgressStream <- progress:
			case <-ctx.Done():
				return
			}
		}

		// Send completion
		completion := StreamCompletion{
			Success:         processedChunks == len(chunks),
			TotalChunks:     len(chunks),
			ProcessedChunks: processedChunks,
			Timestamp:       time.Now(),
		}

		select {
		case streamResponse.CompletionChan <- completion:
		case <-ctx.Done():
		}
	}()

	return streamResponse, nil
}

func (sts *StreamingTTSService) runCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(sts.streamingConfig.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sts.cleanupExpiredSessions()
		case <-ctx.Done():
			return
		}
	}
}

func (sts *StreamingTTSService) cleanupExpiredSessions() {
	sts.streamManager.mu.Lock()
	defer sts.streamManager.mu.Unlock()

	now := time.Now()
	for sessionID, session := range sts.streamManager.streams {
		// Clean up sessions that have been inactive for too long
		if now.Sub(session.LastActivity) > sts.streamingConfig.StreamTimeout {
			sts.logger.WithField("stream_id", sessionID).Info("Cleaning up expired session")
			if session.CancelFunc != nil {
				session.CancelFunc()
			}
			delete(sts.streamManager.streams, sessionID)
		}
	}
}

func (sts *StreamingTTSService) runMetricsCollector(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sts.updateMetrics()
		case <-ctx.Done():
			return
		}
	}
}

func (sts *StreamingTTSService) updateMetrics() {
	// Update worker utilization
	if sts.clusterManager != nil {
		stats := sts.clusterManager.GetClusterStats()
		if activeWorkers, ok := stats["active_workers"].(int64); ok {
			if totalWorkers, ok := stats["total_workers"].(int); ok && totalWorkers > 0 {
				utilization := float64(activeWorkers) / float64(totalWorkers)
				sts.metrics.mu.Lock()
				sts.metrics.WorkerUtilization = utilization
				sts.metrics.mu.Unlock()
			}
		}
	}

	// Update queue depth
	sts.streamManager.mu.RLock()
	queueDepth := len(sts.streamManager.streams)
	sts.streamManager.mu.RUnlock()

	sts.metrics.mu.Lock()
	sts.metrics.QueueDepth = queueDepth
	sts.metrics.mu.Unlock()
}

// Public API methods

func (sts *StreamingTTSService) GetStreamingMetrics() map[string]interface{} {
	return sts.metrics.GetSnapshot()
}

func (sts *StreamingTTSService) GetActiveStreams() []string {
	sts.streamManager.mu.RLock()
	defer sts.streamManager.mu.RUnlock()

	streams := make([]string, 0, len(sts.streamManager.streams))
	for id := range sts.streamManager.streams {
		streams = append(streams, id)
	}
	return streams
}

func (sts *StreamingTTSService) CancelStream(streamID string) error {
	sts.streamManager.mu.Lock()
	defer sts.streamManager.mu.Unlock()

	session, exists := sts.streamManager.streams[streamID]
	if !exists {
		return fmt.Errorf("stream not found: %s", streamID)
	}

	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	session.mu.Lock()
	session.State = StreamStateCanceled
	session.mu.Unlock()

	sts.logger.WithField("stream_id", streamID).Info("Stream canceled")
	return nil
}

func (sts *StreamingTTSService) IsStreamingEnabled() bool {
	return sts.streamingConfig.Enabled && sts.isInitialized
}

func (sts *StreamingTTSService) processSingleChunk(ctx context.Context, req *StreamingTTSRequest) (*StreamingTTSResponse, error) {
	// Convert streaming request to regular request
	regularReq := &TTSRequest{
		Text:     req.Text,
		Voice:    req.Voice,
		Language: req.Language,
		Speed:    req.Speed,
	}

	// Process using the existing single-instance method
	response, err := sts.Service.TextToSpeech(ctx, regularReq)
	if err != nil {
		return nil, err
	}

	// Convert to streaming response
	streamResponse := &StreamingTTSResponse{
		StreamID:       generateRequestID(),
		TotalChunks:    1,
		FinalAudioFile: response.OutputFile,
		AudioStream:    make(chan AudioChunk, 1),
		ProgressStream: make(chan ProgressUpdate, 1),
		ErrorStream:    make(chan error, 1),
		CompletionChan: make(chan StreamCompletion, 1),
	}

	// Send single chunk
	go func() {
		defer close(streamResponse.AudioStream)
		defer close(streamResponse.ProgressStream)
		defer close(streamResponse.ErrorStream)
		defer close(streamResponse.CompletionChan)

		// Read audio file
		var audioData []byte
		if response.OutputFile != "" {
			data, err := os.ReadFile(response.OutputFile)
			if err == nil {
				audioData = data
			}
		}

		chunk := AudioChunk{
			ChunkIndex: 0,
			AudioData:  audioData,
			AudioFile:  response.OutputFile,
			Duration:   time.Duration(response.AudioDuration * float64(time.Second)),
			Timestamp:  time.Now(),
			IsComplete: true,
		}

		streamResponse.AudioStream <- chunk

		completion := StreamCompletion{
			StreamID:        streamResponse.StreamID,
			Success:         response.Success,
			TotalChunks:     1,
			ProcessedChunks: 1,
			FinalAudioFile:  response.OutputFile,
			Timestamp:       time.Now(),
		}

		if !response.Success {
			completion.Error = fmt.Errorf("%s", response.Error)
			completion.ErrorMessage = response.Error
		}

		streamResponse.CompletionChan <- completion
	}()

	return streamResponse, nil
}

func (sts *StreamingTTSService) processIntelligentChunking(ctx context.Context, req *StreamingTTSRequest) (*StreamingTTSResponse, error) {
	sts.logger.WithField("text_length", len(req.Text)).Info("Processing with intelligent chunking")

	// Split text into intelligent chunks
	chunks, err := sts.textSplitter.SplitIntelligently(req.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to split text: %w", err)
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks generated from text")
	}

	sts.logger.WithField("total_chunks", len(chunks)).Info("Text split into chunks")

	// Create streaming response
	streamResponse := &StreamingTTSResponse{
		StreamID:       generateRequestID(),
		TotalChunks:    len(chunks),
		AudioStream:    make(chan AudioChunk, len(chunks)),
		ProgressStream: make(chan ProgressUpdate, len(chunks)*2),
		ErrorStream:    make(chan error, 1),
		CompletionChan: make(chan StreamCompletion, 1),
	}

	// Process chunks sequentially through single daemon
	go func() {
		defer close(streamResponse.AudioStream)
		defer close(streamResponse.ProgressStream)
		defer close(streamResponse.ErrorStream)
		defer close(streamResponse.CompletionChan)

		var audioSegments []AudioSegment
		var totalDuration float64
		startTime := time.Now()
		var processingTimes []time.Duration

		for i, chunk := range chunks {
			chunkStartTime := time.Now()

			sts.logger.WithFields(logrus.Fields{
				"chunk_index": i,
				"chunk_text":  chunk.Content[:minInt(50, len(chunk.Content))],
			}).Info("Processing chunk")

			// Calculate estimated time remaining and processing speed
			var estimatedTimeRemaining time.Duration
			var processingSpeed float64

			if len(processingTimes) > 0 {
				avgProcessingTime := time.Duration(0)
				for _, pt := range processingTimes {
					avgProcessingTime += pt
				}
				avgProcessingTime = avgProcessingTime / time.Duration(len(processingTimes))

				remainingChunks := len(chunks) - i
				estimatedTimeRemaining = avgProcessingTime * time.Duration(remainingChunks)
				processingSpeed = float64(len(chunks)) / time.Since(startTime).Seconds()
			}

			// Send progress update
			progress := ProgressUpdate{
				TotalChunks:            len(chunks),
				ChunksProcessed:        i,
				EstimatedTimeRemaining: estimatedTimeRemaining,
				ProcessingSpeed:        processingSpeed,
				BufferHealth:           1.0,
				Timestamp:              time.Now(),
				Phase:                  "processing",
			}

			select {
			case streamResponse.ProgressStream <- progress:
			case <-ctx.Done():
				return
			}

			// Process chunk through existing TTS daemon
			chunkReq := &TTSRequest{
				Text:     chunk.Content,
				Voice:    req.Voice,
				Language: req.Language,
				Speed:    req.Speed,
			}

			response, err := sts.Service.TextToSpeech(ctx, chunkReq)
			if err != nil {
				sts.logger.WithError(err).WithField("chunk_index", i).Error("Failed to process chunk")
				select {
				case streamResponse.ErrorStream <- fmt.Errorf("chunk %d failed: %w", i, err):
				case <-ctx.Done():
				}
				return
			}

			// Calculate actual duration from audio file
			audioDuration := response.AudioDuration
			if audioDuration <= 0 {
				// Fallback: estimate duration based on text length
				audioDuration = float64(len(chunk.Content)) / 15.0 // Rough estimate: 15 chars per second
			}

			// Create audio segment
			segment := AudioSegment{
				Index:    i,
				FilePath: response.OutputFile,
				Duration: time.Duration(audioDuration * float64(time.Second)),
				Metadata: map[string]interface{}{
					"text":       chunk.Content,
					"voice":      req.Voice,
					"chunk_size": len(chunk.Content),
				},
			}

			audioSegments = append(audioSegments, segment)
			totalDuration += audioDuration

			// Record processing time
			processingTime := time.Since(chunkStartTime)
			processingTimes = append(processingTimes, processingTime)

			// Send audio chunk
			audioChunk := AudioChunk{
				ChunkIndex: i,
				AudioData:  nil, // File path will be used
				AudioFile:  response.OutputFile,
				Duration:   segment.Duration,
				Timestamp:  time.Now(),
				IsComplete: true,
				Metadata:   segment.Metadata,
			}

			select {
			case streamResponse.AudioStream <- audioChunk:
			case <-ctx.Done():
				return
			}

			sts.logger.WithField("chunk_index", i).Info("Chunk processed successfully")
		}

		// Implement audio assembly to create final combined file
		finalAudioFile := ""
		if len(audioSegments) > 0 {
			if sts.audioAssembler != nil {
				// Use audio assembler to combine files
				combinedAudio, err := sts.audioAssembler.AssembleAudioStream(audioSegments)
				if err != nil {
					sts.logger.WithError(err).Warn("Failed to combine audio files, using last chunk")
					finalAudioFile = audioSegments[len(audioSegments)-1].FilePath
				} else {
					// Save combined audio to a temporary file
					tempFile := fmt.Sprintf("/tmp/combined_audio_%s.wav", generateRequestID())
					if err := os.WriteFile(tempFile, combinedAudio, 0644); err == nil {
						finalAudioFile = tempFile
					} else {
						sts.logger.WithError(err).Warn("Failed to save combined audio, using last chunk")
						finalAudioFile = audioSegments[len(audioSegments)-1].FilePath
					}
				}
			} else {
				// Fallback: use last chunk if no assembler available
				finalAudioFile = audioSegments[len(audioSegments)-1].FilePath
			}
		}

		// Calculate actual processing time
		totalProcessingTime := time.Since(startTime)

		// Send completion
		completion := StreamCompletion{
			Success:         true,
			TotalChunks:     len(chunks),
			ProcessedChunks: len(chunks),
			FinalAudioFile:  finalAudioFile,
			TotalDuration:   time.Duration(totalDuration * float64(time.Second)),
			ProcessingTime:  totalProcessingTime,
			Timestamp:       time.Now(),
		}

		select {
		case streamResponse.CompletionChan <- completion:
		case <-ctx.Done():
		}

		sts.logger.WithFields(logrus.Fields{
			"stream_id":       streamResponse.StreamID,
			"total_chunks":    len(chunks),
			"final_file":      finalAudioFile,
			"processing_time": totalProcessingTime,
		}).Info("Intelligent chunking completed")
	}()

	return streamResponse, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
