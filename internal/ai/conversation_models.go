package ai

import (
	"encoding/json"
	"time"
)

// Conversation represents a persistent conversation session
type Conversation struct {
	ID            string                 `json:"id" db:"id"`
	UserID        string                 `json:"user_id" db:"user_id"`
	Title         string                 `json:"title" db:"title"`
	SystemPrompt  *string                `json:"system_prompt,omitempty" db:"system_prompt"`
	Provider      string                 `json:"provider" db:"provider"`
	Model         *string                `json:"model,omitempty" db:"model"`
	Temperature   float64                `json:"temperature" db:"temperature"`
	MaxTokens     int                    `json:"max_tokens" db:"max_tokens"`
	ContextData   map[string]interface{} `json:"context_data,omitempty" db:"context_data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	MessageCount  int                    `json:"message_count" db:"message_count"`
	LastMessageAt *time.Time             `json:"last_message_at,omitempty" db:"last_message_at"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	Archived      bool                   `json:"archived" db:"archived"`
}

// ConversationMessage represents a message within a conversation
type ConversationMessage struct {
	ID             string                 `json:"id" db:"id"`
	ConversationID string                 `json:"conversation_id" db:"conversation_id"`
	Role           string                 `json:"role" db:"role"` // user, assistant, system, tool
	Content        string                 `json:"content" db:"content"`
	ToolCalls      []ToolCall             `json:"tool_calls,omitempty" db:"tool_calls"`
	ToolCallID     *string                `json:"tool_call_id,omitempty" db:"tool_call_id"`
	TokensUsed     int                    `json:"tokens_used" db:"tokens_used"`
	ModelUsed      *string                `json:"model_used,omitempty" db:"model_used"`
	ProviderUsed   *string                `json:"provider_used,omitempty" db:"provider_used"`
	ResponseTimeMs int                    `json:"response_time_ms" db:"response_time_ms"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// ToolCall represents an MCP tool call within a message
type ToolCall struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Function ToolFunction    `json:"function"`
	Result   *ToolCallResult `json:"result,omitempty"`
}

// ToolFunction represents the function details of a tool call
type ToolFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResult represents the result of a tool call execution
type ToolCallResult struct {
	Success       bool        `json:"success"`
	Result        interface{} `json:"result,omitempty"`
	Error         *string     `json:"error,omitempty"`
	ExecutionTime int         `json:"execution_time_ms"`
}

// MCPTool represents an available MCP (Model Context Protocol) tool
type MCPTool struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Schema      map[string]interface{} `json:"schema" db:"schema"`
	Handler     string                 `json:"handler" db:"handler"`
	Category    string                 `json:"category" db:"category"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	UsageCount  int                    `json:"usage_count" db:"usage_count"`
	LastUsed    *time.Time             `json:"last_used,omitempty" db:"last_used"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// MCPToolExecution represents a tool execution record
type MCPToolExecution struct {
	ID              string                 `json:"id" db:"id"`
	ConversationID  string                 `json:"conversation_id" db:"conversation_id"`
	MessageID       string                 `json:"message_id" db:"message_id"`
	ToolID          string                 `json:"tool_id" db:"tool_id"`
	ToolName        string                 `json:"tool_name" db:"tool_name"`
	Parameters      map[string]interface{} `json:"parameters" db:"parameters"`
	Result          *string                `json:"result,omitempty" db:"result"`
	Error           *string                `json:"error,omitempty" db:"error"`
	ExecutionTimeMs int                    `json:"execution_time_ms" db:"execution_time_ms"`
	Success         bool                   `json:"success" db:"success"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// ConversationAnalytics represents analytics data for a conversation
type ConversationAnalytics struct {
	ID                 string    `json:"id" db:"id"`
	ConversationID     string    `json:"conversation_id" db:"conversation_id"`
	TotalMessages      int       `json:"total_messages" db:"total_messages"`
	TotalTokens        int       `json:"total_tokens" db:"total_tokens"`
	TotalCost          float64   `json:"total_cost" db:"total_cost"`
	AvgResponseTimeMs  float64   `json:"avg_response_time_ms" db:"avg_response_time_ms"`
	ToolsUsed          int       `json:"tools_used" db:"tools_used"`
	ProvidersUsed      []string  `json:"providers_used" db:"providers_used"`
	ModelsUsed         []string  `json:"models_used" db:"models_used"`
	SentimentScore     *float64  `json:"sentiment_score,omitempty" db:"sentiment_score"`
	ComplexityScore    *float64  `json:"complexity_score,omitempty" db:"complexity_score"`
	SatisfactionRating *int      `json:"satisfaction_rating,omitempty" db:"satisfaction_rating"`
	Date               time.Time `json:"date" db:"date"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// Request and Response models for conversation API

// CreateConversationRequest represents a request to create a new conversation
type CreateConversationRequest struct {
	Title        string                 `json:"title" binding:"required"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Provider     *string                `json:"provider,omitempty"`
	Model        *string                `json:"model,omitempty"`
	Temperature  *float64               `json:"temperature,omitempty"`
	MaxTokens    *int                   `json:"max_tokens,omitempty"`
	ContextData  map[string]interface{} `json:"context_data,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateConversationRequest represents a request to update conversation settings
type UpdateConversationRequest struct {
	Title        *string                `json:"title,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Provider     *string                `json:"provider,omitempty"`
	Model        *string                `json:"model,omitempty"`
	Temperature  *float64               `json:"temperature,omitempty"`
	MaxTokens    *int                   `json:"max_tokens,omitempty"`
	ContextData  map[string]interface{} `json:"context_data,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Archived     *bool                  `json:"archived,omitempty"`
}

// SendMessageRequest represents a request to send a message in a conversation
type SendMessageRequest struct {
	Content     string                 `json:"content" binding:"required"`
	Role        *string                `json:"role,omitempty"`
	ToolCalls   []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID  *string                `json:"tool_call_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
}

// ConversationResponse represents an enhanced conversation response with full context
type ConversationResponse struct {
	Conversation *Conversation          `json:"conversation"`
	Messages     []ConversationMessage  `json:"messages,omitempty"`
	Analytics    *ConversationAnalytics `json:"analytics,omitempty"`
	ToolsUsed    []MCPTool              `json:"tools_used,omitempty"`
}

// ConversationFilter represents filters for conversation queries
type ConversationFilter struct {
	UserID      *string    `json:"user_id,omitempty"`
	Archived    *bool      `json:"archived,omitempty"`
	Provider    *string    `json:"provider,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	HasMessages *bool      `json:"has_messages,omitempty"`
	SearchQuery *string    `json:"search_query,omitempty"` // Search in title or messages
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
	OrderBy     string     `json:"order_by,omitempty"`  // created_at, updated_at, last_message_at
	OrderDir    string     `json:"order_dir,omitempty"` // asc, desc
}

// MessageFilter represents filters for message queries
type MessageFilter struct {
	ConversationID *string    `json:"conversation_id,omitempty"`
	Role           *string    `json:"role,omitempty"`
	HasToolCalls   *bool      `json:"has_tool_calls,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	SearchQuery    *string    `json:"search_query,omitempty"` // Search in content
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
	OrderBy        string     `json:"order_by,omitempty"`  // created_at
	OrderDir       string     `json:"order_dir,omitempty"` // asc, desc
}

// MCPToolFilter represents filters for MCP tool queries
type MCPToolFilter struct {
	Category    *string `json:"category,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	SearchQuery *string `json:"search_query,omitempty"` // Search in name or description
	Limit       int     `json:"limit,omitempty"`
	Offset      int     `json:"offset,omitempty"`
	OrderBy     string  `json:"order_by,omitempty"`  // name, usage_count, created_at
	OrderDir    string  `json:"order_dir,omitempty"` // asc, desc
}

// ConversationStatistics represents overall conversation statistics
type ConversationStatistics struct {
	TotalConversations    int                  `json:"total_conversations"`
	ActiveConversations   int                  `json:"active_conversations"`
	ArchivedConversations int                  `json:"archived_conversations"`
	TotalMessages         int                  `json:"total_messages"`
	TotalTokensUsed       int                  `json:"total_tokens_used"`
	TotalCost             float64              `json:"total_cost"`
	AvgMessagesPerConv    float64              `json:"avg_messages_per_conversation"`
	AvgResponseTime       float64              `json:"avg_response_time_ms"`
	TopProviders          []ProviderUsageStats `json:"top_providers"`
	TopModels             []ModelUsageStats    `json:"top_models"`
	TopTools              []ToolUsageStats     `json:"top_tools"`
	DailyActivity         []DailyActivityStats `json:"daily_activity"`
}

// ProviderUsageStats represents usage statistics for AI providers
type ProviderUsageStats struct {
	Provider     string  `json:"provider"`
	MessageCount int     `json:"message_count"`
	TokenCount   int     `json:"token_count"`
	Cost         float64 `json:"cost"`
	AvgLatency   float64 `json:"avg_latency_ms"`
}

// ModelUsageStats represents usage statistics for AI models
type ModelUsageStats struct {
	Model        string  `json:"model"`
	Provider     string  `json:"provider"`
	MessageCount int     `json:"message_count"`
	TokenCount   int     `json:"token_count"`
	Cost         float64 `json:"cost"`
	AvgLatency   float64 `json:"avg_latency_ms"`
}

// ToolUsageStats represents usage statistics for MCP tools
type ToolUsageStats struct {
	ToolName    string  `json:"tool_name"`
	Category    string  `json:"category"`
	UsageCount  int     `json:"usage_count"`
	SuccessRate float64 `json:"success_rate"`
	AvgExecTime float64 `json:"avg_execution_time_ms"`
}

// DailyActivityStats represents daily conversation activity
type DailyActivityStats struct {
	Date          time.Time `json:"date"`
	Conversations int       `json:"conversations"`
	Messages      int       `json:"messages"`
	Tokens        int       `json:"tokens"`
	ToolCalls     int       `json:"tool_calls"`
}

// Enhanced chat request with conversation context
type EnhancedChatRequest struct {
	ConversationID string                 `json:"conversation_id,omitempty"`
	Content        string                 `json:"content" binding:"required"`
	Role           string                 `json:"role,omitempty"`
	Provider       *string                `json:"provider,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Temperature    *float64               `json:"temperature,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	ToolCalls      []ToolCall             `json:"tool_calls,omitempty"`
	EnableTools    bool                   `json:"enable_tools,omitempty"`
	ToolChoice     string                 `json:"tool_choice,omitempty"` // auto, none, specific tool name
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Context        *ConversationContext   `json:"context,omitempty"`
}

// Enhanced chat response with conversation persistence
type EnhancedChatResponse struct {
	ConversationID string              `json:"conversation_id"`
	Message        ConversationMessage `json:"message"`
	Response       ChatResponse        `json:"response"`
	ToolExecutions []MCPToolExecution  `json:"tool_executions,omitempty"`
	TokensUsed     int                 `json:"tokens_used"`
	Cost           float64             `json:"cost"`
	ResponseTime   time.Duration       `json:"response_time"`
	Provider       string              `json:"provider"`
	Model          string              `json:"model"`
}

// Helper methods for JSON marshaling/unmarshaling

// MarshalContextData marshals context data to JSON string for database storage
func (c *Conversation) MarshalContextData() (string, error) {
	if c.ContextData == nil {
		return "", nil
	}
	data, err := json.Marshal(c.ContextData)
	return string(data), err
}

// UnmarshalContextData unmarshals context data from JSON string
func (c *Conversation) UnmarshalContextData(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), &c.ContextData)
}

// MarshalMetadata marshals metadata to JSON string for database storage
func (c *Conversation) MarshalMetadata() (string, error) {
	if c.Metadata == nil {
		return "", nil
	}
	data, err := json.Marshal(c.Metadata)
	return string(data), err
}

// UnmarshalMetadata unmarshals metadata from JSON string
func (c *Conversation) UnmarshalMetadata(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), &c.Metadata)
}

// MarshalToolCalls marshals tool calls to JSON string for database storage
func (cm *ConversationMessage) MarshalToolCalls() (string, error) {
	if cm.ToolCalls == nil {
		return "", nil
	}
	data, err := json.Marshal(cm.ToolCalls)
	return string(data), err
}

// UnmarshalToolCalls unmarshals tool calls from JSON string
func (cm *ConversationMessage) UnmarshalToolCalls(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), &cm.ToolCalls)
}
