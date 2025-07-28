package ai

import (
	"context"
	"time"
)

// LLMProvider defines the interface that all AI providers must implement
type LLMProvider interface {
	// Core operations
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (*CompletionResponse, error)
	Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (*ChatResponse, error)

	// Provider info
	GetName() string
	IsAvailable(ctx context.Context) bool
	GetModels(ctx context.Context) ([]ModelInfo, error)

	// Resource management
	EstimateTokens(text string) int
	GetRateLimit() RateLimit

	// Health and lifecycle
	HealthCheck(ctx context.Context) error
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// CompletionOptions holds options for text completion requests
type CompletionOptions struct {
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Stop        []string          `json:"stop,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// LLMTool represents a tool/function available to the LLM for function calling
type LLMTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON schema for parameters
}

// ChatOptions holds options for chat completion requests
type ChatOptions struct {
	Model        string            `json:"model,omitempty"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	Temperature  float64           `json:"temperature,omitempty"`
	TopP         float64           `json:"top_p,omitempty"`
	Stream       bool              `json:"stream,omitempty"`
	Stop         []string          `json:"stop,omitempty"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Tools        []LLMTool         `json:"tools,omitempty"`       // MCP tools converted to LLM format
	ToolChoice   string            `json:"tool_choice,omitempty"` // "auto", "none", or specific tool name
}

// ChatMessage represents a single message in a chat conversation
type ChatMessage struct {
	Role       string            `json:"role"` // "system", "user", "assistant", "tool"
	Content    string            `json:"content"`
	Name       string            `json:"name,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	ToolCalls  []ToolCall        `json:"tool_calls,omitempty"`   // For assistant messages with tool calls
	ToolCallID string            `json:"tool_call_id,omitempty"` // For tool response messages
}

// CompletionResponse represents the response from a completion request
type CompletionResponse struct {
	ID               string            `json:"id"`
	Text             string            `json:"text"`
	FinishReason     string            `json:"finish_reason"`
	TokensUsed       TokenUsage        `json:"tokens_used"`
	Model            string            `json:"model"`
	Provider         string            `json:"provider"`
	ProcessingTimeMs int64             `json:"processing_time_ms"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// ChatResponse represents the response from a chat request
type ChatResponse struct {
	ID               string            `json:"id"`
	Message          ChatMessage       `json:"message"`
	FinishReason     string            `json:"finish_reason"`
	TokensUsed       TokenUsage        `json:"tokens_used"`
	Model            string            `json:"model"`
	Provider         string            `json:"provider"`
	ProcessingTimeMs int64             `json:"processing_time_ms"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// TokenUsage represents token consumption statistics
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo represents information about an available model
type ModelInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Provider        string            `json:"provider"`
	MaxTokens       int               `json:"max_tokens,omitempty"`
	InputCostPer1K  float64           `json:"input_cost_per_1k,omitempty"`
	OutputCostPer1K float64           `json:"output_cost_per_1k,omitempty"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Available       bool              `json:"available"`
	LocalModel      bool              `json:"local_model"`
}

// RateLimit represents rate limiting information for a provider
type RateLimit struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	RequestsPerDay    int           `json:"requests_per_day"`
	TokensPerMinute   int           `json:"tokens_per_minute"`
	TokensPerHour     int           `json:"tokens_per_hour"`
	TokensPerDay      int           `json:"tokens_per_day"`
	ResetTime         time.Time     `json:"reset_time,omitempty"`
	RetryAfter        time.Duration `json:"retry_after,omitempty"`
}

// ProviderStatus represents the current status of a provider
type ProviderStatus struct {
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	Available         bool              `json:"available"`
	Healthy           bool              `json:"healthy"`
	LastHealthCheck   time.Time         `json:"last_health_check"`
	ErrorCount        int64             `json:"error_count"`
	RequestCount      int64             `json:"request_count"`
	AverageResponseMs int64             `json:"average_response_ms"`
	Models            []ModelInfo       `json:"models"`
	RateLimit         RateLimit         `json:"rate_limit"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// ProviderError represents errors from AI providers
type ProviderError struct {
	Provider   string        `json:"provider"`
	Type       string        `json:"type"` // "rate_limit", "auth", "network", "model_error", "internal"
	Message    string        `json:"message"`
	Code       string        `json:"code,omitempty"`
	Retryable  bool          `json:"retryable"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	Underlying error         `json:"-"`
}

func (e *ProviderError) Error() string {
	return e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns true if the error is retryable
func (e *ProviderError) IsRetryable() bool {
	return e.Retryable
}
