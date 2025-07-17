package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// ClaudeProvider implements the LLMProvider interface for Anthropic Claude
type ClaudeProvider struct {
	name              string
	config            config.AIProviderConfig
	client            *http.Client
	logger            *logrus.Logger
	apiKey            string
	baseURL           string
	defaultModel      string
	mu                sync.RWMutex
	errorCount        int64
	requestCount      int64
	totalResponseTime time.Duration
	lastHealthCheck   time.Time
	isHealthy         bool
	rateLimit         ai.RateLimit
	rateLimiter       *rateLimiter
}

// NewClaudeProvider creates a new Claude provider instance
func NewClaudeProvider(cfg config.AIProviderConfig, logger *logrus.Logger) *ClaudeProvider {
	baseURL := "https://api.anthropic.com/v1"

	return &ClaudeProvider{
		name:         "claude",
		config:       cfg,
		client:       &http.Client{Timeout: 60 * time.Second},
		logger:       logger,
		apiKey:       cfg.APIKey,
		baseURL:      baseURL,
		defaultModel: cfg.DefaultModel,
		rateLimit: ai.RateLimit{
			RequestsPerMinute: 1000,
			RequestsPerHour:   5000,
			RequestsPerDay:    50000,
			TokensPerMinute:   40000,
			TokensPerHour:     240000,
			TokensPerDay:      1200000,
		},
		rateLimiter: newRateLimiter(1000, 40000), // requests/min, tokens/min
	}
}

// Initialize initializes the Claude provider
func (c *ClaudeProvider) Initialize(ctx context.Context) error {
	if c.apiKey == "" {
		return &ai.ProviderError{
			Provider: c.name,
			Type:     "auth",
			Message:  "Claude API key is required",
		}
	}

	return c.HealthCheck(ctx)
}

// Shutdown cleans up the provider
func (c *ClaudeProvider) Shutdown(ctx context.Context) error {
	return nil
}

// GetName returns the provider name
func (c *ClaudeProvider) GetName() string {
	return c.name
}

// IsAvailable checks if the provider is available
func (c *ClaudeProvider) IsAvailable(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.apiKey == "" {
		return false
	}

	// Check if we need to refresh health status
	if time.Since(c.lastHealthCheck) > 5*time.Minute {
		go func() {
			if err := c.HealthCheck(ctx); err != nil {
				c.logger.WithError(err).Debug("Claude health check failed")
			}
		}()
	}

	return c.isHealthy
}

// HealthCheck performs a health check on the Claude API
func (c *ClaudeProvider) HealthCheck(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.apiKey == "" {
		c.isHealthy = false
		return &ai.ProviderError{
			Provider: c.name,
			Type:     "auth",
			Message:  "Claude API key is not configured",
		}
	}

	// Make a simple completion request to validate API key
	testRequest := map[string]interface{}{
		"model":      c.defaultModel,
		"max_tokens": 1,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Hi",
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", nil)
	if err != nil {
		c.isHealthy = false
		return err
	}

	jsonBody, _ := json.Marshal(testRequest)
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.isHealthy = false
		c.errorCount++
		return &ai.ProviderError{
			Provider:   c.name,
			Type:       "network",
			Message:    "Failed to connect to Claude API",
			Retryable:  true,
			Underlying: err,
		}
	}
	defer resp.Body.Close()

	c.lastHealthCheck = time.Now()

	if resp.StatusCode == 401 {
		c.isHealthy = false
		return &ai.ProviderError{
			Provider: c.name,
			Type:     "auth",
			Message:  "Invalid Claude API key",
		}
	}

	if resp.StatusCode >= 400 && resp.StatusCode != 400 { // 400 might be expected for test request
		c.isHealthy = false
		c.errorCount++
		return &ai.ProviderError{
			Provider:  c.name,
			Type:      "internal",
			Message:   fmt.Sprintf("Claude API returned status %d", resp.StatusCode),
			Retryable: resp.StatusCode >= 500,
		}
	}

	c.isHealthy = true
	return nil
}

// Complete performs text completion using Claude
func (c *ClaudeProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOptions) (*ai.CompletionResponse, error) {
	if !c.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  c.name,
			Type:      "unavailable",
			Message:   "Claude provider is not available",
			Retryable: true,
		}
	}

	// Convert completion to chat format (Claude's approach)
	messages := []ai.ChatMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	chatOpts := ai.ChatOptions{
		Model:       opts.Model,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		TopP:        opts.TopP,
		Stream:      opts.Stream,
		Stop:        opts.Stop,
		Metadata:    opts.Metadata,
	}

	chatResp, err := c.Chat(ctx, messages, chatOpts)
	if err != nil {
		return nil, err
	}

	return &ai.CompletionResponse{
		ID:               chatResp.ID,
		Text:             chatResp.Message.Content,
		FinishReason:     chatResp.FinishReason,
		TokensUsed:       chatResp.TokensUsed,
		Model:            chatResp.Model,
		Provider:         chatResp.Provider,
		ProcessingTimeMs: chatResp.ProcessingTimeMs,
		Metadata:         chatResp.Metadata,
		CreatedAt:        chatResp.CreatedAt,
	}, nil
}

// Chat performs chat completion using Claude
func (c *ClaudeProvider) Chat(ctx context.Context, messages []ai.ChatMessage, opts ai.ChatOptions) (*ai.ChatResponse, error) {
	if !c.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  c.name,
			Type:      "unavailable",
			Message:   "Claude provider is not available",
			Retryable: true,
		}
	}

	model := opts.Model
	if model == "" {
		model = c.defaultModel
	}

	// Estimate tokens for rate limiting
	totalPrompt := ""
	for _, msg := range messages {
		totalPrompt += msg.Content + "\n"
	}
	if opts.SystemPrompt != "" {
		totalPrompt += opts.SystemPrompt + "\n"
	}

	// Check rate limits
	if err := c.rateLimiter.checkRequest(ctx, c.EstimateTokens(totalPrompt)); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Convert messages to Claude format
	claudeMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		claudeMessages = append(claudeMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	request := map[string]interface{}{
		"model":      model,
		"messages":   claudeMessages,
		"max_tokens": 4096, // Claude requires max_tokens
	}

	if opts.MaxTokens > 0 {
		request["max_tokens"] = opts.MaxTokens
	}
	if opts.Temperature > 0 {
		request["temperature"] = opts.Temperature
	}
	if opts.TopP > 0 {
		request["top_p"] = opts.TopP
	}
	if opts.SystemPrompt != "" {
		request["system"] = opts.SystemPrompt
	}
	if len(opts.Stop) > 0 {
		request["stop_sequences"] = opts.Stop
	}

	resp, err := c.makeRequest(ctx, "POST", "/messages", request)
	if err != nil {
		c.mu.Lock()
		c.errorCount++
		c.mu.Unlock()
		return nil, err
	}

	c.mu.Lock()
	c.requestCount++
	c.totalResponseTime += time.Since(startTime)
	c.mu.Unlock()

	var claudeResp struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Role  string `json:"role"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.Unmarshal(resp, &claudeResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   c.name,
			Type:       "parse_error",
			Message:    "Failed to parse Claude response",
			Underlying: err,
		}
	}

	if len(claudeResp.Content) == 0 {
		return nil, &ai.ProviderError{
			Provider: c.name,
			Type:     "internal",
			Message:  "No content returned from Claude",
		}
	}

	responseMessage := ai.ChatMessage{
		Role:      "assistant",
		Content:   claudeResp.Content[0].Text,
		Timestamp: time.Now(),
		Metadata:  opts.Metadata,
	}

	return &ai.ChatResponse{
		ID:               claudeResp.ID,
		Message:          responseMessage,
		FinishReason:     claudeResp.StopReason,
		Model:            claudeResp.Model,
		Provider:         c.name,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		TokensUsed: ai.TokenUsage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
		CreatedAt: time.Now(),
		Metadata:  opts.Metadata,
	}, nil
}

// GetModels returns available Claude models
func (c *ClaudeProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	// Claude doesn't have a models endpoint, so we return known models
	models := []ai.ModelInfo{
		{
			ID:              "claude-3-haiku-20240307",
			Name:            "Claude 3 Haiku",
			Description:     "Fast and cost-effective Claude 3 model",
			Provider:        c.name,
			MaxTokens:       200000,
			InputCostPer1K:  0.00025,
			OutputCostPer1K: 0.00125,
			Capabilities:    []string{"chat", "completion"},
			Available:       c.IsAvailable(ctx),
			LocalModel:      false,
		},
		{
			ID:              "claude-3-sonnet-20240229",
			Name:            "Claude 3 Sonnet",
			Description:     "Balanced performance Claude 3 model",
			Provider:        c.name,
			MaxTokens:       200000,
			InputCostPer1K:  0.003,
			OutputCostPer1K: 0.015,
			Capabilities:    []string{"chat", "completion"},
			Available:       c.IsAvailable(ctx),
			LocalModel:      false,
		},
		{
			ID:              "claude-3-opus-20240229",
			Name:            "Claude 3 Opus",
			Description:     "Most powerful Claude 3 model",
			Provider:        c.name,
			MaxTokens:       200000,
			InputCostPer1K:  0.015,
			OutputCostPer1K: 0.075,
			Capabilities:    []string{"chat", "completion"},
			Available:       c.IsAvailable(ctx),
			LocalModel:      false,
		},
	}

	return models, nil
}

// EstimateTokens provides an estimate of token count for Claude
func (c *ClaudeProvider) EstimateTokens(text string) int {
	// Claude uses similar tokenization to OpenAI
	words := len(strings.Fields(text))
	chars := len(text)

	// Use the higher of word-based or character-based estimation
	wordTokens := int(float64(words) * 1.3)
	charTokens := chars / 3

	if wordTokens > charTokens {
		return wordTokens
	}
	return charTokens
}

// GetRateLimit returns current rate limiting information
func (c *ClaudeProvider) GetRateLimit() ai.RateLimit {
	return c.rateLimit
}

// Private methods

func (c *ClaudeProvider) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, &ai.ProviderError{
				Provider:   c.name,
				Type:       "internal",
				Message:    "Failed to marshal request body",
				Underlying: err,
			}
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   c.name,
			Type:       "internal",
			Message:    "Failed to create request",
			Underlying: err,
		}
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("User-Agent", "PMA-Backend/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   c.name,
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
			Provider:   c.name,
			Type:       "network",
			Message:    "Failed to read response body",
			Underlying: err,
		}
	}

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(responseBody, &errorResp)

		errorType := "internal"
		retryable := false
		var retryAfter time.Duration

		switch resp.StatusCode {
		case 401:
			errorType = "auth"
		case 429:
			errorType = "rate_limit"
			retryable = true
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				if seconds, err := strconv.Atoi(retryHeader); err == nil {
					retryAfter = time.Duration(seconds) * time.Second
				}
			}
		case 400:
			errorType = "model_error"
		default:
			if resp.StatusCode >= 500 {
				retryable = true
			}
		}

		return nil, &ai.ProviderError{
			Provider:   c.name,
			Type:       errorType,
			Message:    fmt.Sprintf("Claude API error: %s", errorResp.Error.Message),
			Code:       strconv.Itoa(resp.StatusCode),
			Retryable:  retryable,
			RetryAfter: retryAfter,
		}
	}

	return responseBody, nil
}
