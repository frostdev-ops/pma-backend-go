package ai

import (
	"context"
	"time"
)

// LLMProvider defines the interface that all AI providers must implement
type LLMProvider interface {
	// Core identification
	GetName() string
	IsAvailable(ctx context.Context) bool
	HealthCheck(ctx context.Context) error

	// Lifecycle management
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error

	// Chat completion (using the existing ChatRequest/ChatResponse format)
	Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (*ChatResponse, error)

	// Completion
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (*CompletionResponse, error)

	// Model management
	GetModels(ctx context.Context) ([]ModelInfo, error)

	// Utilities
	EstimateTokens(text string) int
	GetRateLimit() RateLimit
	GetStats() ProviderStats
}

// ChatOptions represents options for chat requests
type ChatOptions struct {
	Provider     string            `json:"provider,omitempty"`
	Model        string            `json:"model,omitempty"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	Temperature  float64           `json:"temperature,omitempty"`
	TopP         float64           `json:"top_p,omitempty"`
	Stream       bool              `json:"stream,omitempty"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
	Tools        []Tool            `json:"tools,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// CompletionOptions represents options for completion requests
type CompletionOptions struct {
	Provider    string            `json:"provider,omitempty"`
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	Stop        []string          `json:"stop,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ChatMessage represents a single message in a chat
type ChatMessage struct {
	Role      string    `json:"role"` // "user", "assistant", "system", "tool"
	Content   string    `json:"content"`
	Name      string    `json:"name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatResponse represents a response from chat completion
type ChatResponse struct {
	ID               string            `json:"id"`
	Message          ChatMessage       `json:"message"`
	ToolCalls        []ToolCall        `json:"tool_calls,omitempty"`
	FinishReason     string            `json:"finish_reason"`
	Model            string            `json:"model"`
	Provider         string            `json:"provider"`
	ProcessingTimeMs int               `json:"processing_time_ms"`
	TokensUsed       TokenUsage        `json:"tokens_used"`
	CreatedAt        time.Time         `json:"created_at"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// CompletionResponse represents a response from text completion
type CompletionResponse struct {
	ID               string     `json:"id"`
	Text             string     `json:"text"`
	FinishReason     string     `json:"finish_reason"`
	Model            string     `json:"model"`
	Provider         string     `json:"provider"`
	ProcessingTimeMs int        `json:"processing_time_ms"`
	TokensUsed       TokenUsage `json:"tokens_used"`
	CreatedAt        time.Time  `json:"created_at"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo represents information about an available model
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Provider     string   `json:"provider"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Available    bool     `json:"available"`
	LocalModel   bool     `json:"local_model"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// RateLimit represents provider rate limits
type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
}

// ProviderStatus represents the status of a provider
type ProviderStatus struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Available   bool        `json:"available"`
	Health      string      `json:"health"`
	LastChecked time.Time   `json:"last_checked"`
	Errors      []string    `json:"errors,omitempty"`
	Models      []ModelInfo `json:"models,omitempty"`
}

// ProviderStats represents provider usage statistics
type ProviderStats struct {
	RequestCount        int64         `json:"request_count"`
	ErrorCount          int64         `json:"error_count"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastUsed            time.Time     `json:"last_used"`
}

// Tool represents a tool/function available to the AI (for future LFM2 support)
type Tool struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition defines a function signature
type ToolFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}
