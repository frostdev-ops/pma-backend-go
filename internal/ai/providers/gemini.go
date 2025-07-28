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
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GeminiProvider implements the LLMProvider interface for Google Gemini
type GeminiProvider struct {
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

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider(cfg config.AIProviderConfig, logger *logrus.Logger) *GeminiProvider {
	baseURL := "https://generativelanguage.googleapis.com/v1beta"

	return &GeminiProvider{
		name:         "gemini",
		config:       cfg,
		client:       &http.Client{Timeout: 60 * time.Second},
		logger:       logger,
		apiKey:       cfg.APIKey,
		baseURL:      baseURL,
		defaultModel: cfg.DefaultModel,
		rateLimit: ai.RateLimit{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
			TokensPerMinute:   32000,
			TokensPerHour:     200000,
			TokensPerDay:      1000000,
		},
		rateLimiter: newRateLimiter(60, 32000), // requests/min, tokens/min
	}
}

// Initialize initializes the Gemini provider
func (g *GeminiProvider) Initialize(ctx context.Context) error {
	if g.apiKey == "" {
		return &ai.ProviderError{
			Provider: g.name,
			Type:     "auth",
			Message:  "Gemini API key is required",
		}
	}

	return g.HealthCheck(ctx)
}

// Shutdown cleans up the provider
func (g *GeminiProvider) Shutdown(ctx context.Context) error {
	return nil
}

// GetName returns the provider name
func (g *GeminiProvider) GetName() string {
	return g.name
}

// IsAvailable checks if the provider is available
func (g *GeminiProvider) IsAvailable(ctx context.Context) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.apiKey == "" {
		return false
	}

	// Check if we need to refresh health status
	if time.Since(g.lastHealthCheck) > 5*time.Minute {
		go func() {
			if err := g.HealthCheck(ctx); err != nil {
				g.logger.WithError(err).Debug("Gemini health check failed")
			}
		}()
	}

	return g.isHealthy
}

// HealthCheck performs a health check on the Gemini API
func (g *GeminiProvider) HealthCheck(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.apiKey == "" {
		g.isHealthy = false
		return &ai.ProviderError{
			Provider: g.name,
			Type:     "auth",
			Message:  "Gemini API key is not configured",
		}
	}

	// Make a simple models request to validate API key
	url := fmt.Sprintf("%s/models?key=%s", g.baseURL, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		g.isHealthy = false
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		g.isHealthy = false
		g.errorCount++
		return &ai.ProviderError{
			Provider:   g.name,
			Type:       "network",
			Message:    "Failed to connect to Gemini API",
			Retryable:  true,
			Underlying: err,
		}
	}
	defer resp.Body.Close()

	g.lastHealthCheck = time.Now()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		g.isHealthy = false
		return &ai.ProviderError{
			Provider: g.name,
			Type:     "auth",
			Message:  "Invalid Gemini API key",
		}
	}

	if resp.StatusCode >= 400 {
		g.isHealthy = false
		g.errorCount++
		return &ai.ProviderError{
			Provider:  g.name,
			Type:      "internal",
			Message:   fmt.Sprintf("Gemini API returned status %d", resp.StatusCode),
			Retryable: resp.StatusCode >= 500,
		}
	}

	g.isHealthy = true
	return nil
}

// Complete performs text completion using Gemini
func (g *GeminiProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOptions) (*ai.CompletionResponse, error) {
	if !g.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  g.name,
			Type:      "unavailable",
			Message:   "Gemini provider is not available",
			Retryable: true,
		}
	}

	// Convert completion to chat format (Gemini's approach)
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

	chatResp, err := g.Chat(ctx, messages, chatOpts)
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

// Chat performs chat completion using Gemini
func (g *GeminiProvider) Chat(ctx context.Context, messages []ai.ChatMessage, opts ai.ChatOptions) (*ai.ChatResponse, error) {
	if !g.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  g.name,
			Type:      "unavailable",
			Message:   "Gemini provider is not available",
			Retryable: true,
		}
	}

	model := opts.Model
	if model == "" {
		model = g.defaultModel
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
	if err := g.rateLimiter.checkRequest(ctx, g.EstimateTokens(totalPrompt)); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Convert messages to Gemini format
	geminiContents := make([]map[string]interface{}, 0, len(messages))

	// Add system prompt as first message if provided
	if opts.SystemPrompt != "" {
		geminiContents = append(geminiContents, map[string]interface{}{
			"role": "user",
			"parts": []map[string]interface{}{
				{"text": opts.SystemPrompt},
			},
		})
		geminiContents = append(geminiContents, map[string]interface{}{
			"role": "model",
			"parts": []map[string]interface{}{
				{"text": "I understand. How can I help you?"},
			},
		})
	}

	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		} else if msg.Role == "system" {
			// Skip system messages if already handled above
			continue
		}

		geminiContents = append(geminiContents, map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		})
	}

	request := map[string]interface{}{
		"contents": geminiContents,
	}

	// Add generation config
	genConfig := make(map[string]interface{})
	if opts.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = opts.MaxTokens
	}
	if opts.Temperature > 0 {
		genConfig["temperature"] = opts.Temperature
	}
	if opts.TopP > 0 {
		genConfig["topP"] = opts.TopP
	}
	if len(opts.Stop) > 0 {
		genConfig["stopSequences"] = opts.Stop
	}

	if len(genConfig) > 0 {
		request["generationConfig"] = genConfig
	}

	// Add function calling tools if provided
	if len(opts.Tools) > 0 {
		geminiTools := g.convertToolsToGeminiFormat(opts.Tools)
		request["tools"] = geminiTools

		// Add tool config for function calling mode
		if opts.ToolChoice != "" {
			toolConfig := map[string]interface{}{
				"functionCallingConfig": map[string]interface{}{
					"mode": g.convertToolChoiceToGeminiMode(opts.ToolChoice),
				},
			}
			request["toolConfig"] = toolConfig
		}
	}

	url := fmt.Sprintf("/models/%s:generateContent", model)
	resp, err := g.makeRequest(ctx, "POST", url, request)
	if err != nil {
		g.mu.Lock()
		g.errorCount++
		g.mu.Unlock()
		return nil, err
	}

	g.mu.Lock()
	g.requestCount++
	g.totalResponseTime += time.Since(startTime)
	g.mu.Unlock()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string                 `json:"text,omitempty"`
					FunctionCall map[string]interface{} `json:"functionCall,omitempty"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
			Index        int    `json:"index"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
		ModelVersion string `json:"modelVersion"`
	}

	if err := json.Unmarshal(resp, &geminiResp); err != nil {
		return nil, &ai.ProviderError{
			Provider:   g.name,
			Type:       "parse_error",
			Message:    "Failed to parse Gemini response",
			Underlying: err,
		}
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, &ai.ProviderError{
			Provider: g.name,
			Type:     "internal",
			Message:  "No candidates returned from Gemini",
		}
	}

	candidate := geminiResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, &ai.ProviderError{
			Provider: g.name,
			Type:     "internal",
			Message:  "No content parts returned from Gemini",
		}
	}

	// Extract content and function calls from response
	var content string
	var toolCalls []ai.ToolCall

	for i, part := range candidate.Content.Parts {
		if part.Text != "" {
			content += part.Text
		}

		if part.FunctionCall != nil {
			toolCall, err := g.convertGeminiFunctionCallToToolCall(part.FunctionCall, i)
			if err != nil {
				g.logger.WithError(err).Warn("Failed to convert Gemini function call")
				continue
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	responseMessage := ai.ChatMessage{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
		Metadata:  opts.Metadata,
	}

	return &ai.ChatResponse{
		ID:               uuid.New().String(),
		Message:          responseMessage,
		FinishReason:     candidate.FinishReason,
		Model:            model,
		Provider:         g.name,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		TokensUsed: ai.TokenUsage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
		CreatedAt: time.Now(),
		Metadata:  opts.Metadata,
	}, nil
}

// GetModels returns available Gemini models
func (g *GeminiProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	if !g.IsAvailable(ctx) {
		return nil, &ai.ProviderError{
			Provider:  g.name,
			Type:      "unavailable",
			Message:   "Gemini provider is not available",
			Retryable: true,
		}
	}

	resp, err := g.makeRequest(ctx, "GET", "/models", nil)
	if err != nil {
		// Return known models if API call fails
		return g.getKnownModels(ctx), nil
	}

	var geminiResp struct {
		Models []struct {
			Name                string   `json:"name"`
			DisplayName         string   `json:"displayName"`
			Description         string   `json:"description"`
			InputTokenLimit     int      `json:"inputTokenLimit"`
			OutputTokenLimit    int      `json:"outputTokenLimit"`
			SupportedGeneration []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}

	if err := json.Unmarshal(resp, &geminiResp); err != nil {
		// Return known models if parsing fails
		return g.getKnownModels(ctx), nil
	}

	models := make([]ai.ModelInfo, 0, len(geminiResp.Models))
	for _, model := range geminiResp.Models {
		// Filter to only include generative models
		supportsGeneration := false
		for _, method := range model.SupportedGeneration {
			if method == "generateContent" {
				supportsGeneration = true
				break
			}
		}

		if !supportsGeneration {
			continue
		}

		// Extract model ID from name (e.g., "models/gemini-pro" -> "gemini-pro")
		modelID := model.Name
		if strings.HasPrefix(modelID, "models/") {
			modelID = strings.TrimPrefix(modelID, "models/")
		}

		maxTokens := model.OutputTokenLimit
		if maxTokens == 0 {
			maxTokens = 8192 // Default for most Gemini models
		}

		models = append(models, ai.ModelInfo{
			ID:           modelID,
			Name:         model.DisplayName,
			Description:  model.Description,
			Provider:     g.name,
			MaxTokens:    maxTokens,
			Capabilities: []string{"chat", "completion"},
			Available:    true,
			LocalModel:   false,
			Metadata: map[string]string{
				"input_token_limit":  strconv.Itoa(model.InputTokenLimit),
				"output_token_limit": strconv.Itoa(model.OutputTokenLimit),
			},
		})
	}

	if len(models) == 0 {
		return g.getKnownModels(ctx), nil
	}

	return models, nil
}

// getKnownModels returns a fallback list of known Gemini models
func (g *GeminiProvider) getKnownModels(ctx context.Context) []ai.ModelInfo {
	return []ai.ModelInfo{
		{
			ID:           "gemini-pro",
			Name:         "Gemini Pro",
			Description:  "Google's most capable generative AI model",
			Provider:     g.name,
			MaxTokens:    8192,
			Capabilities: []string{"chat", "completion"},
			Available:    g.IsAvailable(ctx),
			LocalModel:   false,
		},
		{
			ID:           "gemini-pro-vision",
			Name:         "Gemini Pro Vision",
			Description:  "Gemini Pro with vision capabilities",
			Provider:     g.name,
			MaxTokens:    8192,
			Capabilities: []string{"chat", "completion", "vision"},
			Available:    g.IsAvailable(ctx),
			LocalModel:   false,
		},
	}
}

// EstimateTokens provides an estimate of token count for Gemini
func (g *GeminiProvider) EstimateTokens(text string) int {
	// Gemini uses similar tokenization patterns
	words := len(strings.Fields(text))
	chars := len(text)

	// Use the higher of word-based or character-based estimation
	wordTokens := int(float64(words) * 1.3)
	charTokens := chars / 4

	if wordTokens > charTokens {
		return wordTokens
	}
	return charTokens
}

// GetRateLimit returns current rate limiting information
func (g *GeminiProvider) GetRateLimit() ai.RateLimit {
	return g.rateLimit
}

// Private methods

func (g *GeminiProvider) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, &ai.ProviderError{
				Provider:   g.name,
				Type:       "internal",
				Message:    "Failed to marshal request body",
				Underlying: err,
			}
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := g.baseURL + endpoint + "?key=" + g.apiKey
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   g.name,
			Type:       "internal",
			Message:    "Failed to create request",
			Underlying: err,
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "PMA-Backend/1.0")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, &ai.ProviderError{
			Provider:   g.name,
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
			Provider:   g.name,
			Type:       "network",
			Message:    "Failed to read response body",
			Underlying: err,
		}
	}

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		json.Unmarshal(responseBody, &errorResp)

		errorType := "internal"
		retryable := false
		var retryAfter time.Duration

		switch resp.StatusCode {
		case 401, 403:
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
			Provider:   g.name,
			Type:       errorType,
			Message:    fmt.Sprintf("Gemini API error: %s", errorResp.Error.Message),
			Code:       strconv.Itoa(resp.StatusCode),
			Retryable:  retryable,
			RetryAfter: retryAfter,
		}
	}

	return responseBody, nil
}

// convertToolsToGeminiFormat converts LLMTool slice to Gemini function calling format
func (g *GeminiProvider) convertToolsToGeminiFormat(tools []ai.LLMTool) []map[string]interface{} {
	geminiTools := make([]map[string]interface{}, 0, len(tools))

	for _, tool := range tools {
		// Fix schema for Gemini compatibility (add missing items fields for arrays)
		fixedParams := g.fixSchemaForGemini(tool.Parameters)

		geminiTool := map[string]interface{}{
			"functionDeclarations": []map[string]interface{}{
				{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  fixedParams,
				},
			},
		}
		geminiTools = append(geminiTools, geminiTool)
	}

	return geminiTools
}

// fixSchemaForGemini recursively fixes JSON schema to be Gemini compatible
func (g *GeminiProvider) fixSchemaForGemini(schema interface{}) interface{} {
	switch v := schema.(type) {
	case map[string]interface{}:
		fixed := make(map[string]interface{})
		for key, value := range v {
			if key == "properties" {
				// Fix properties recursively
				if props, ok := value.(map[string]interface{}); ok {
					fixedProps := make(map[string]interface{})
					for propName, propDef := range props {
						fixedProps[propName] = g.fixSchemaForGemini(propDef)
					}
					fixed[key] = fixedProps
				} else {
					fixed[key] = g.fixSchemaForGemini(value)
				}
			} else if key == "type" && value == "array" {
				// This is an array property - ensure it has items
				fixed[key] = value
				// Check if items is missing
				if _, hasItems := v["items"]; !hasItems {
					// Add default items schema
					fixed["items"] = map[string]interface{}{
						"type":        "object",
						"description": "Array item",
					}
				}
			} else {
				fixed[key] = g.fixSchemaForGemini(value)
			}
		}
		return fixed
	case []interface{}:
		fixed := make([]interface{}, len(v))
		for i, item := range v {
			fixed[i] = g.fixSchemaForGemini(item)
		}
		return fixed
	default:
		return v
	}
}

// convertToolChoiceToGeminiMode converts LLM tool choice to Gemini function calling mode
func (g *GeminiProvider) convertToolChoiceToGeminiMode(toolChoice string) string {
	switch strings.ToLower(toolChoice) {
	case "auto":
		return "AUTO"
	case "none":
		return "NONE"
	case "required", "any":
		return "ANY"
	default:
		return "AUTO" // Default to AUTO mode
	}
}

// convertGeminiFunctionCallToToolCall converts Gemini function call response to ToolCall format
func (g *GeminiProvider) convertGeminiFunctionCallToToolCall(functionCall map[string]interface{}, index int) (ai.ToolCall, error) {
	// Extract function name
	name, ok := functionCall["name"].(string)
	if !ok {
		return ai.ToolCall{}, fmt.Errorf("function call missing name")
	}

	// Extract arguments
	args, ok := functionCall["args"].(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	// Generate unique ID for this tool call
	toolCallID := fmt.Sprintf("call_%s_%d_%d", name, time.Now().UnixNano(), index)

	return ai.ToolCall{
		ID:   toolCallID,
		Name: name,
		Function: ai.ToolFunction{
			Name:      name,
			Arguments: args,
		},
	}, nil
}
