package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OllamaProvider implements the LLMProvider interface for Ollama
type OllamaProvider struct {
	name              string
	config            config.AIProviderConfig
	client            *http.Client
	logger            *logrus.Logger
	baseURL           string
	defaultModel      string
	mu                sync.RWMutex
	isRunning         bool
	processCmd        *exec.Cmd
	lastHealthCheck   time.Time
	errorCount        int64
	requestCount      int64
	totalResponseTime time.Duration
	availableModels   []ai.ModelInfo
	modelCache        map[string]ai.ModelInfo
	cacheTTL          time.Duration
	lastCacheUpdate   time.Time
}

// NewOllamaProvider creates a new Ollama provider instance
func NewOllamaProvider(cfg config.AIProviderConfig, logger *logrus.Logger) *OllamaProvider {
	baseURL := cfg.URL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		name:         "ollama",
		config:       cfg,
		client:       &http.Client{Timeout: 60 * time.Second},
		logger:       logger,
		baseURL:      baseURL,
		defaultModel: cfg.DefaultModel,
		modelCache:   make(map[string]ai.ModelInfo),
		cacheTTL:     5 * time.Minute,
	}
}

// Initialize starts the Ollama service if auto_start is enabled
func (o *OllamaProvider) Initialize(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if already running
	if err := o.healthCheck(ctx); err == nil {
		o.isRunning = true
		o.logger.Info("Ollama service already running")
		return nil
	}

	// Start service if auto_start is enabled
	if o.config.AutoStart {
		if err := o.startService(ctx); err != nil {
			return fmt.Errorf("failed to start Ollama service: %w", err)
		}
	}

	// Wait for service to be ready
	return o.waitForReady(ctx, 30*time.Second)
}

// Shutdown stops the Ollama service if it was started by this provider
func (o *OllamaProvider) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.processCmd != nil && o.processCmd.Process != nil {
		o.logger.Info("Stopping Ollama service")
		if err := o.processCmd.Process.Kill(); err != nil {
			o.logger.WithError(err).Warn("Failed to kill Ollama process")
		}
		o.processCmd.Wait()
		o.processCmd = nil
	}

	o.isRunning = false
	return nil
}

// GetName returns the provider name
func (o *OllamaProvider) GetName() string {
	return o.name
}

// IsAvailable checks if the provider is available
func (o *OllamaProvider) IsAvailable(ctx context.Context) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Check if we need to refresh health status
	if time.Since(o.lastHealthCheck) > 30*time.Second {
		go func() {
			if err := o.HealthCheck(ctx); err != nil {
				o.logger.WithError(err).Debug("Ollama health check failed")
			}
		}()
	}

	return o.isRunning
}

// HealthCheck performs a health check on the Ollama service
func (o *OllamaProvider) HealthCheck(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	err := o.healthCheck(ctx)
	o.lastHealthCheck = time.Now()

	if err != nil {
		o.isRunning = false
		o.errorCount++
		return err
	}

	o.isRunning = true
	return nil
}

// Complete performs text completion
func (o *OllamaProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOptions) (*ai.CompletionResponse, error) {
	if !o.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      "unavailable",
			Message:   "Ollama service is not available",
			Retryable: true,
		}
	}

	model := opts.Model
	if model == "" {
		model = o.defaultModel
	}

	// Ensure model is available
	if err := o.ensureModelAvailable(ctx, model); err != nil {
		return nil, err
	}

	startTime := time.Now()

	request := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	if opts.MaxTokens > 0 {
		request["options"] = map[string]interface{}{
			"num_predict": opts.MaxTokens,
		}
	}

	if opts.Temperature > 0 {
		if request["options"] == nil {
			request["options"] = make(map[string]interface{})
		}
		request["options"].(map[string]interface{})["temperature"] = opts.Temperature
	}

	resp, err := o.makeRequest(ctx, "POST", "/api/generate", request)
	if err != nil {
		o.mu.Lock()
		o.errorCount++
		o.mu.Unlock()
		return nil, err
	}

	o.mu.Lock()
	o.requestCount++
	o.totalResponseTime += time.Since(startTime)
	o.mu.Unlock()

	var ollamaResp struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
		Model    string `json:"model"`
	}

	if err := json.Unmarshal(resp, &ollamaResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "parse_error",
			Message:    "Failed to parse Ollama response",
			Underlying: err,
		}
	}

	return &ai.CompletionResponse{
		ID:               uuid.New().String(),
		Text:             ollamaResp.Response,
		FinishReason:     "stop",
		Model:            ollamaResp.Model,
		Provider:         o.name,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		TokensUsed: ai.TokenUsage{
			PromptTokens:     o.EstimateTokens(prompt),
			CompletionTokens: o.EstimateTokens(ollamaResp.Response),
			TotalTokens:      o.EstimateTokens(prompt + ollamaResp.Response),
		},
		CreatedAt: time.Now(),
		Metadata:  opts.Metadata,
	}, nil
}

// Chat performs chat completion
func (o *OllamaProvider) Chat(ctx context.Context, messages []ai.ChatMessage, opts ai.ChatOptions) (*ai.ChatResponse, error) {
	if !o.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      "unavailable",
			Message:   "Ollama service is not available",
			Retryable: true,
		}
	}

	model := opts.Model
	if model == "" {
		model = o.defaultModel
	}

	// Ensure model is available
	if err := o.ensureModelAvailable(ctx, model); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Convert messages to Ollama format
	ollamaMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	request := map[string]interface{}{
		"model":    model,
		"messages": ollamaMessages,
		"stream":   false,
	}

	if opts.MaxTokens > 0 {
		request["options"] = map[string]interface{}{
			"num_predict": opts.MaxTokens,
		}
	}

	if opts.Temperature > 0 {
		if request["options"] == nil {
			request["options"] = make(map[string]interface{})
		}
		request["options"].(map[string]interface{})["temperature"] = opts.Temperature
	}

	resp, err := o.makeRequest(ctx, "POST", "/api/chat", request)
	if err != nil {
		o.mu.Lock()
		o.errorCount++
		o.mu.Unlock()
		return nil, err
	}

	o.mu.Lock()
	o.requestCount++
	o.totalResponseTime += time.Since(startTime)
	o.mu.Unlock()

	var ollamaResp struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done  bool   `json:"done"`
		Model string `json:"model"`
	}

	if err := json.Unmarshal(resp, &ollamaResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "parse_error",
			Message:    "Failed to parse Ollama response",
			Underlying: err,
		}
	}

	responseMessage := ai.ChatMessage{
		Role:      ollamaResp.Message.Role,
		Content:   ollamaResp.Message.Content,
		Timestamp: time.Now(),
		Metadata:  opts.Metadata,
	}

	totalPrompt := ""
	for _, msg := range messages {
		totalPrompt += msg.Content + "\n"
	}

	return &ai.ChatResponse{
		ID:               uuid.New().String(),
		Message:          responseMessage,
		FinishReason:     "stop",
		Model:            ollamaResp.Model,
		Provider:         o.name,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		TokensUsed: ai.TokenUsage{
			PromptTokens:     o.EstimateTokens(totalPrompt),
			CompletionTokens: o.EstimateTokens(ollamaResp.Message.Content),
			TotalTokens:      o.EstimateTokens(totalPrompt + ollamaResp.Message.Content),
		},
		CreatedAt: time.Now(),
		Metadata:  opts.Metadata,
	}, nil
}

// GetModels returns available models
func (o *OllamaProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	o.mu.RLock()
	if time.Since(o.lastCacheUpdate) < o.cacheTTL && len(o.availableModels) > 0 {
		defer o.mu.RUnlock()
		return o.availableModels, nil
	}
	o.mu.RUnlock()

	if !o.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      "unavailable",
			Message:   "Ollama service is not available",
			Retryable: true,
		}
	}

	resp, err := o.makeRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		return nil, err
	}

	var ollamaResp struct {
		Models []struct {
			Name       string    `json:"name"`
			Size       int64     `json:"size"`
			ModifiedAt time.Time `json:"modified_at"`
			Digest     string    `json:"digest"`
		} `json:"models"`
	}

	if err := json.Unmarshal(resp, &ollamaResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "parse_error",
			Message:    "Failed to parse models response",
			Underlying: err,
		}
	}

	models := make([]ai.ModelInfo, len(ollamaResp.Models))
	for i, model := range ollamaResp.Models {
		models[i] = ai.ModelInfo{
			ID:           model.Name,
			Name:         model.Name,
			Description:  fmt.Sprintf("Ollama model %s (%.2f GB)", model.Name, float64(model.Size)/(1024*1024*1024)),
			Provider:     o.name,
			Available:    true,
			LocalModel:   true,
			Capabilities: []string{"chat", "completion"},
			Metadata: map[string]string{
				"size":        strconv.FormatInt(model.Size, 10),
				"digest":      model.Digest,
				"modified_at": model.ModifiedAt.Format(time.RFC3339),
			},
		}
	}

	o.mu.Lock()
	o.availableModels = models
	o.lastCacheUpdate = time.Now()

	// Update model cache
	for _, model := range models {
		o.modelCache[model.ID] = model
	}
	o.mu.Unlock()

	return models, nil
}

// EstimateTokens provides a rough estimate of token count
func (o *OllamaProvider) EstimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

// GetRateLimit returns rate limiting information
func (o *OllamaProvider) GetRateLimit() ai.RateLimit {
	// Ollama running locally typically has no rate limits
	return ai.RateLimit{
		RequestsPerMinute: 1000,
		RequestsPerHour:   60000,
		RequestsPerDay:    1440000,
		TokensPerMinute:   100000,
		TokensPerHour:     6000000,
		TokensPerDay:      144000000,
	}
}

// Private methods

func (o *OllamaProvider) healthCheck(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (o *OllamaProvider) startService(ctx context.Context) error {
	o.logger.Info("Starting Ollama service")

	// Check if ollama command exists
	if _, err := exec.LookPath("ollama"); err != nil {
		return fmt.Errorf("ollama command not found in PATH: %w", err)
	}

	// Start ollama serve
	cmd := exec.CommandContext(ctx, "ollama", "serve")

	// Set environment variables for resource limits if specified
	env := os.Environ()
	if o.config.ResourceLimits.MaxMemory != "" {
		env = append(env, "OLLAMA_MAX_MEMORY="+o.config.ResourceLimits.MaxMemory)
	}
	if o.config.ResourceLimits.MaxCPU > 0 {
		env = append(env, "OLLAMA_NUM_THREADS="+strconv.Itoa(o.config.ResourceLimits.MaxCPU))
	}
	cmd.Env = env

	// Capture logs
	cmd.Stdout = &logWriter{logger: o.logger, level: logrus.InfoLevel}
	cmd.Stderr = &logWriter{logger: o.logger, level: logrus.ErrorLevel}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama service: %w", err)
	}

	o.processCmd = cmd
	o.logger.WithField("pid", cmd.Process.Pid).Info("Ollama service started")

	return nil
}

func (o *OllamaProvider) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Ollama service to be ready")
		case <-ticker.C:
			if err := o.healthCheck(ctx); err == nil {
				o.logger.Info("Ollama service is ready")
				return nil
			}
		}
	}
}

func (o *OllamaProvider) ensureModelAvailable(ctx context.Context, model string) error {
	// Check if model is in cache
	o.mu.RLock()
	_, exists := o.modelCache[model]
	o.mu.RUnlock()

	if exists {
		return nil
	}

	// Refresh model list
	models, err := o.GetModels(ctx)
	if err != nil {
		return err
	}

	// Check if model is available
	for _, m := range models {
		if m.ID == model {
			return nil
		}
	}

	// Model not found, try to pull it
	o.logger.WithField("model", model).Info("Model not found, attempting to pull")
	return o.pullModel(ctx, model)
}

func (o *OllamaProvider) pullModel(ctx context.Context, model string) error {
	request := map[string]interface{}{
		"name": model,
	}

	_, err := o.makeRequest(ctx, "POST", "/api/pull", request)
	if err != nil {
		return &ai.ProviderError{
			Provider:   o.name,
			Type:       "model_error",
			Message:    fmt.Sprintf("Failed to pull model %s", model),
			Underlying: err,
		}
	}

	o.logger.WithField("model", model).Info("Model pulled successfully")

	// Invalidate cache to refresh model list
	o.mu.Lock()
	o.lastCacheUpdate = time.Time{}
	o.mu.Unlock()

	return nil
}

func (o *OllamaProvider) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, &ai.ProviderError{
				Provider:   o.name,
				Type:       "internal",
				Message:    "Failed to marshal request body",
				Underlying: err,
			}
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "internal",
			Message:    "Failed to create request",
			Underlying: err,
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "network",
			Message:    "Network error during request",
			Retryable:  true,
			Underlying: err,
		}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "network",
			Message:    "Failed to read response body",
			Underlying: err,
		}
	}

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(responseBody, &errorResp)

		errorType := "internal"
		retryable := false
		if resp.StatusCode >= 500 {
			errorType = "internal"
			retryable = true
		} else if resp.StatusCode == 429 {
			errorType = "rate_limit"
			retryable = true
		} else if resp.StatusCode == 404 {
			errorType = "model_error"
		}

		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      errorType,
			Message:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errorResp.Error),
			Code:      strconv.Itoa(resp.StatusCode),
			Retryable: retryable,
		}
	}

	return responseBody, nil
}

// logWriter implements io.Writer to capture ollama logs
type logWriter struct {
	logger *logrus.Logger
	level  logrus.Level
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		lw.logger.WithField("source", "ollama").Log(lw.level, msg)
	}
	return len(p), nil
}
