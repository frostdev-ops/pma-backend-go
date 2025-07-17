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

// OpenAIProvider implements the LLMProvider interface for OpenAI
type OpenAIProvider struct {
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

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(cfg config.AIProviderConfig, logger *logrus.Logger) *OpenAIProvider {
	baseURL := "https://api.openai.com/v1"

	return &OpenAIProvider{
		name:         "openai",
		config:       cfg,
		client:       &http.Client{Timeout: 60 * time.Second},
		logger:       logger,
		apiKey:       cfg.APIKey,
		baseURL:      baseURL,
		defaultModel: cfg.DefaultModel,
		rateLimit: ai.RateLimit{
			RequestsPerMinute: 3500,
			RequestsPerHour:   10000,
			RequestsPerDay:    100000,
			TokensPerMinute:   90000,
			TokensPerHour:     540000,
			TokensPerDay:      2000000,
		},
		rateLimiter: newRateLimiter(3500, 90000), // requests/min, tokens/min
	}
}

// Initialize initializes the OpenAI provider
func (o *OpenAIProvider) Initialize(ctx context.Context) error {
	if o.apiKey == "" {
		return &ai.ProviderError{
			Provider: o.name,
			Type:     "auth",
			Message:  "OpenAI API key is required",
		}
	}

	// Validate API key with a simple request
	return o.HealthCheck(ctx)
}

// Shutdown cleans up the provider
func (o *OpenAIProvider) Shutdown(ctx context.Context) error {
	return nil
}

// GetName returns the provider name
func (o *OpenAIProvider) GetName() string {
	return o.name
}

// IsAvailable checks if the provider is available
func (o *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if o.apiKey == "" {
		return false
	}

	// Check if we need to refresh health status
	if time.Since(o.lastHealthCheck) > 5*time.Minute {
		go func() {
			if err := o.HealthCheck(ctx); err != nil {
				o.logger.WithError(err).Debug("OpenAI health check failed")
			}
		}()
	}

	return o.isHealthy
}

// HealthCheck performs a health check on the OpenAI API
func (o *OpenAIProvider) HealthCheck(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.apiKey == "" {
		o.isHealthy = false
		return &ai.ProviderError{
			Provider: o.name,
			Type:     "auth",
			Message:  "OpenAI API key is not configured",
		}
	}

	// Make a simple models request to validate API key
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/models", nil)
	if err != nil {
		o.isHealthy = false
		return err
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		o.isHealthy = false
		o.errorCount++
		return &ai.ProviderError{
			Provider:   o.name,
			Type:       "network",
			Message:    "Failed to connect to OpenAI API",
			Retryable:  true,
			Underlying: err,
		}
	}
	defer resp.Body.Close()

	o.lastHealthCheck = time.Now()

	if resp.StatusCode == 401 {
		o.isHealthy = false
		return &ai.ProviderError{
			Provider: o.name,
			Type:     "auth",
			Message:  "Invalid OpenAI API key",
		}
	}

	if resp.StatusCode >= 400 {
		o.isHealthy = false
		o.errorCount++
		return &ai.ProviderError{
			Provider:  o.name,
			Type:      "internal",
			Message:   fmt.Sprintf("OpenAI API returned status %d", resp.StatusCode),
			Retryable: resp.StatusCode >= 500,
		}
	}

	o.isHealthy = true
	return nil
}

// Complete performs text completion using OpenAI's completion endpoint
func (o *OpenAIProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOptions) (*ai.CompletionResponse, error) {
	if !o.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      "unavailable",
			Message:   "OpenAI provider is not available",
			Retryable: true,
		}
	}

	model := opts.Model
	if model == "" {
		model = o.defaultModel
	}

	// Check rate limits
	if err := o.rateLimiter.checkRequest(ctx, o.EstimateTokens(prompt)); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Convert completion to chat format (OpenAI's preferred approach)
	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": prompt,
		},
	}

	request := map[string]interface{}{
		"model":    model,
		"messages": messages,
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
	if len(opts.Stop) > 0 {
		request["stop"] = opts.Stop
	}

	resp, err := o.makeRequest(ctx, "POST", "/chat/completions", request)
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

	var openaiResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Usage   struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp, &openaiResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "parse_error",
			Message:    "Failed to parse OpenAI response",
			Underlying: err,
		}
	}

	if len(openaiResp.Choices) == 0 {
		return nil, &ai.ProviderError{
			Provider: o.name,
			Type:     "internal",
			Message:  "No choices returned from OpenAI",
		}
	}

	choice := openaiResp.Choices[0]

	return &ai.CompletionResponse{
		ID:               openaiResp.ID,
		Text:             choice.Message.Content,
		FinishReason:     choice.FinishReason,
		Model:            openaiResp.Model,
		Provider:         o.name,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		TokensUsed: ai.TokenUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
		CreatedAt: time.Unix(openaiResp.Created, 0),
		Metadata:  opts.Metadata,
	}, nil
}

// Chat performs chat completion using OpenAI's chat endpoint
func (o *OpenAIProvider) Chat(ctx context.Context, messages []ai.ChatMessage, opts ai.ChatOptions) (*ai.ChatResponse, error) {
	if !o.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      "unavailable",
			Message:   "OpenAI provider is not available",
			Retryable: true,
		}
	}

	model := opts.Model
	if model == "" {
		model = o.defaultModel
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
	if err := o.rateLimiter.checkRequest(ctx, o.EstimateTokens(totalPrompt)); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Convert messages to OpenAI format
	openaiMessages := make([]map[string]interface{}, 0, len(messages)+1)

	// Add system prompt if provided
	if opts.SystemPrompt != "" {
		openaiMessages = append(openaiMessages, map[string]interface{}{
			"role":    "system",
			"content": opts.SystemPrompt,
		})
	}

	for _, msg := range messages {
		openaiMessages = append(openaiMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	request := map[string]interface{}{
		"model":    model,
		"messages": openaiMessages,
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
	if len(opts.Stop) > 0 {
		request["stop"] = opts.Stop
	}

	resp, err := o.makeRequest(ctx, "POST", "/chat/completions", request)
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

	var openaiResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Usage   struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp, &openaiResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "parse_error",
			Message:    "Failed to parse OpenAI response",
			Underlying: err,
		}
	}

	if len(openaiResp.Choices) == 0 {
		return nil, &ai.ProviderError{
			Provider: o.name,
			Type:     "internal",
			Message:  "No choices returned from OpenAI",
		}
	}

	choice := openaiResp.Choices[0]
	responseMessage := ai.ChatMessage{
		Role:      choice.Message.Role,
		Content:   choice.Message.Content,
		Timestamp: time.Now(),
		Metadata:  opts.Metadata,
	}

	return &ai.ChatResponse{
		ID:               openaiResp.ID,
		Message:          responseMessage,
		FinishReason:     choice.FinishReason,
		Model:            openaiResp.Model,
		Provider:         o.name,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		TokensUsed: ai.TokenUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
		CreatedAt: time.Unix(openaiResp.Created, 0),
		Metadata:  opts.Metadata,
	}, nil
}

// GetModels returns available models from OpenAI
func (o *OpenAIProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	if !o.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  o.name,
			Type:      "unavailable",
			Message:   "OpenAI provider is not available",
			Retryable: true,
		}
	}

	resp, err := o.makeRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, err
	}

	var openaiResp struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &openaiResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   o.name,
			Type:       "parse_error",
			Message:    "Failed to parse models response",
			Underlying: err,
		}
	}

	models := make([]ai.ModelInfo, 0, len(openaiResp.Data))
	for _, model := range openaiResp.Data {
		// Filter to only include relevant models
		if strings.Contains(model.ID, "gpt") || strings.Contains(model.ID, "davinci") || strings.Contains(model.ID, "text") {
			maxTokens := 4096
			if strings.Contains(model.ID, "gpt-4") {
				maxTokens = 8192
				if strings.Contains(model.ID, "32k") {
					maxTokens = 32768
				}
			} else if strings.Contains(model.ID, "16k") {
				maxTokens = 16384
			}

			capabilities := []string{"chat"}
			if strings.Contains(model.ID, "text-") || strings.Contains(model.ID, "davinci") {
				capabilities = append(capabilities, "completion")
			}

			// Estimate costs (rough estimates, should be updated with actual pricing)
			inputCost := 0.0015 // per 1K tokens
			outputCost := 0.002 // per 1K tokens
			if strings.Contains(model.ID, "gpt-4") {
				inputCost = 0.03
				outputCost = 0.06
			}

			models = append(models, ai.ModelInfo{
				ID:              model.ID,
				Name:            model.ID,
				Description:     fmt.Sprintf("OpenAI %s model (max %d tokens)", model.ID, maxTokens),
				Provider:        o.name,
				MaxTokens:       maxTokens,
				InputCostPer1K:  inputCost,
				OutputCostPer1K: outputCost,
				Capabilities:    capabilities,
				Available:       true,
				LocalModel:      false,
				Metadata: map[string]string{
					"owned_by": model.OwnedBy,
					"created":  strconv.FormatInt(model.Created, 10),
				},
			})
		}
	}

	return models, nil
}

// EstimateTokens provides an estimate of token count using OpenAI's rules
func (o *OpenAIProvider) EstimateTokens(text string) int {
	// More accurate estimation for OpenAI models
	// Rough estimation: 1 token ≈ 3.5 characters for English text
	// Plus account for special tokens, spaces, etc.
	words := len(strings.Fields(text))
	chars := len(text)

	// Use the higher of word-based or character-based estimation
	wordTokens := int(float64(words) * 1.3) // Account for subword tokenization
	charTokens := chars / 3

	if wordTokens > charTokens {
		return wordTokens
	}
	return charTokens
}

// GetRateLimit returns current rate limiting information
func (o *OpenAIProvider) GetRateLimit() ai.RateLimit {
	return o.rateLimit
}

// Private methods

func (o *OpenAIProvider) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
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

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PMA-Backend/1.0")

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

	// Update rate limit info from headers
	o.updateRateLimitFromHeaders(resp.Header)

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
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
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
			Provider:   o.name,
			Type:       errorType,
			Message:    fmt.Sprintf("OpenAI API error: %s", errorResp.Error.Message),
			Code:       strconv.Itoa(resp.StatusCode),
			Retryable:  retryable,
			RetryAfter: retryAfter,
		}
	}

	return responseBody, nil
}

func (o *OpenAIProvider) updateRateLimitFromHeaders(headers http.Header) {
	// OpenAI provides rate limit info in response headers
	if limit := headers.Get("x-ratelimit-limit-requests"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			o.rateLimit.RequestsPerMinute = val
		}
	}
	if limit := headers.Get("x-ratelimit-limit-tokens"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			o.rateLimit.TokensPerMinute = val
		}
	}
	if reset := headers.Get("x-ratelimit-reset-requests"); reset != "" {
		if duration, err := time.ParseDuration(reset); err == nil {
			o.rateLimit.ResetTime = time.Now().Add(duration)
		}
	}
}

// Simple rate limiter implementation
type rateLimiter struct {
	requestsPerMin int
	tokensPerMin   int
	requestWindow  []time.Time
	tokenUsage     []tokenUsage
	mu             sync.Mutex
}

type tokenUsage struct {
	timestamp time.Time
	tokens    int
}

func newRateLimiter(requestsPerMin, tokensPerMin int) *rateLimiter {
	return &rateLimiter{
		requestsPerMin: requestsPerMin,
		tokensPerMin:   tokensPerMin,
		requestWindow:  make([]time.Time, 0),
		tokenUsage:     make([]tokenUsage, 0),
	}
}

func (r *rateLimiter) checkRequest(ctx context.Context, estimatedTokens int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	// Clean old entries
	r.cleanOldEntries(oneMinuteAgo)

	// Check request rate
	if len(r.requestWindow) >= r.requestsPerMin {
		return &ai.ProviderError{
			Provider:   "openai",
			Type:       "rate_limit",
			Message:    "Request rate limit exceeded",
			Retryable:  true,
			RetryAfter: time.Minute,
		}
	}

	// Check token rate
	totalTokens := estimatedTokens
	for _, usage := range r.tokenUsage {
		totalTokens += usage.tokens
	}
	if totalTokens > r.tokensPerMin {
		return &ai.ProviderError{
			Provider:   "openai",
			Type:       "rate_limit",
			Message:    "Token rate limit exceeded",
			Retryable:  true,
			RetryAfter: time.Minute,
		}
	}

	// Record this request
	r.requestWindow = append(r.requestWindow, now)
	r.tokenUsage = append(r.tokenUsage, tokenUsage{
		timestamp: now,
		tokens:    estimatedTokens,
	})

	return nil
}

func (r *rateLimiter) cleanOldEntries(cutoff time.Time) {
	// Clean request window
	newRequests := make([]time.Time, 0)
	for _, t := range r.requestWindow {
		if t.After(cutoff) {
			newRequests = append(newRequests, t)
		}
	}
	r.requestWindow = newRequests

	// Clean token usage
	newTokenUsage := make([]tokenUsage, 0)
	for _, usage := range r.tokenUsage {
		if usage.timestamp.After(cutoff) {
			newTokenUsage = append(newTokenUsage, usage)
		}
	}
	r.tokenUsage = newTokenUsage
}
