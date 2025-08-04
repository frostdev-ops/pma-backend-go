package repositories

import (
	"context"
	"time"
)

// ConversationRepository defines conversation data access methods
// Using interface{} temporarily to avoid import cycles with ai package
type ConversationRepository interface {
	// Conversation management
	CreateConversation(ctx context.Context, conv interface{}) error
	GetConversation(ctx context.Context, id string, userID string) (interface{}, error)
	GetConversations(ctx context.Context, filter interface{}) ([]interface{}, error)
	GetConversationCount(ctx context.Context, filter interface{}) (int, error)
	UpdateConversation(ctx context.Context, conv interface{}) error
	DeleteConversation(ctx context.Context, id string, userID string) error
	ArchiveConversation(ctx context.Context, id string, userID string) error
	UnarchiveConversation(ctx context.Context, id string, userID string) error

	// Message management
	CreateMessage(ctx context.Context, msg interface{}) error
	GetMessage(ctx context.Context, id string) (interface{}, error)
	GetMessages(ctx context.Context, filter interface{}) ([]interface{}, error)
	GetMessageCount(ctx context.Context, filter interface{}) (int, error)
	GetConversationMessages(ctx context.Context, conversationID string, limit int, offset int) ([]interface{}, error)
	UpdateMessage(ctx context.Context, msg interface{}) error
	DeleteMessage(ctx context.Context, id string) error
	DeleteConversationMessages(ctx context.Context, conversationID string) error

	// Conversation analytics
	CreateOrUpdateAnalytics(ctx context.Context, analytics interface{}) error
	GetConversationAnalytics(ctx context.Context, conversationID string, date time.Time) (interface{}, error)
	GetAnalyticsByDateRange(ctx context.Context, conversationID string, startDate, endDate time.Time) ([]interface{}, error)
	GetGlobalStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (interface{}, error)

	// Cleanup and maintenance
	CleanupOldConversations(ctx context.Context, days int) error
	CleanupOldMessages(ctx context.Context, days int) error
	CleanupOldAnalytics(ctx context.Context, days int) error
}

// MCPRepository defines MCP (Model Context Protocol) data access methods
// Using interface{} temporarily to avoid import cycles with ai package
type MCPRepository interface {
	// Tool management
	CreateTool(ctx context.Context, tool interface{}) error
	GetTool(ctx context.Context, id string) (interface{}, error)
	GetToolByName(ctx context.Context, name string) (interface{}, error)
	GetTools(ctx context.Context, filter interface{}) ([]interface{}, error)
	GetAllTools(ctx context.Context) ([]interface{}, error)
	GetToolCount(ctx context.Context, filter interface{}) (int, error)
	GetEnabledTools(ctx context.Context, category string) ([]interface{}, error)
	UpdateTool(ctx context.Context, tool interface{}) error
	DeleteTool(ctx context.Context, id string) error
	EnableTool(ctx context.Context, id string) error
	DisableTool(ctx context.Context, id string) error

	// Tool execution tracking
	CreateToolExecution(ctx context.Context, execution interface{}) error
	GetToolExecution(ctx context.Context, id string) (interface{}, error)
	GetToolExecutions(ctx context.Context, conversationID string, limit int, offset int) ([]interface{}, error)
	GetToolExecutionsByTool(ctx context.Context, toolID string, limit int, offset int) ([]interface{}, error)
	UpdateToolExecution(ctx context.Context, execution interface{}) error
	IncrementToolUsage(ctx context.Context, toolID string) error

	// Tool statistics and analytics
	GetToolUsageStats(ctx context.Context, startDate, endDate time.Time) ([]interface{}, error)
	GetToolSuccessRate(ctx context.Context, toolID string, days int) (float64, error)
	GetMostUsedTools(ctx context.Context, limit int, days int) ([]interface{}, error)
	GetRecentToolExecutions(ctx context.Context, limit int) ([]interface{}, error)

	// Cleanup
	CleanupOldExecutions(ctx context.Context, days int) error
}
