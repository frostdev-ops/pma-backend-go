package repositories

import (
	"context"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
)

// ConversationRepository defines conversation data access methods
type ConversationRepository interface {
	// Conversation management
	CreateConversation(ctx context.Context, conv *ai.Conversation) error
	GetConversation(ctx context.Context, id string, userID string) (*ai.Conversation, error)
	GetConversations(ctx context.Context, filter *ai.ConversationFilter) ([]*ai.Conversation, error)
	GetConversationCount(ctx context.Context, filter *ai.ConversationFilter) (int, error)
	UpdateConversation(ctx context.Context, conv *ai.Conversation) error
	DeleteConversation(ctx context.Context, id string, userID string) error
	ArchiveConversation(ctx context.Context, id string, userID string) error
	UnarchiveConversation(ctx context.Context, id string, userID string) error

	// Message management
	CreateMessage(ctx context.Context, msg *ai.ConversationMessage) error
	GetMessage(ctx context.Context, id string) (*ai.ConversationMessage, error)
	GetMessages(ctx context.Context, filter *ai.MessageFilter) ([]*ai.ConversationMessage, error)
	GetMessageCount(ctx context.Context, filter *ai.MessageFilter) (int, error)
	GetConversationMessages(ctx context.Context, conversationID string, limit int, offset int) ([]*ai.ConversationMessage, error)
	UpdateMessage(ctx context.Context, msg *ai.ConversationMessage) error
	DeleteMessage(ctx context.Context, id string) error
	DeleteConversationMessages(ctx context.Context, conversationID string) error

	// Conversation analytics
	CreateOrUpdateAnalytics(ctx context.Context, analytics *ai.ConversationAnalytics) error
	GetConversationAnalytics(ctx context.Context, conversationID string, date time.Time) (*ai.ConversationAnalytics, error)
	GetAnalyticsByDateRange(ctx context.Context, conversationID string, startDate, endDate time.Time) ([]*ai.ConversationAnalytics, error)
	GetGlobalStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (*ai.ConversationStatistics, error)

	// Cleanup and maintenance
	CleanupOldConversations(ctx context.Context, days int) error
	CleanupOldMessages(ctx context.Context, days int) error
	CleanupOldAnalytics(ctx context.Context, days int) error
}

// MCPRepository defines MCP (Model Context Protocol) data access methods
type MCPRepository interface {
	// Tool management
	CreateTool(ctx context.Context, tool *ai.MCPTool) error
	GetTool(ctx context.Context, id string) (*ai.MCPTool, error)
	GetToolByName(ctx context.Context, name string) (*ai.MCPTool, error)
	GetTools(ctx context.Context, filter *ai.MCPToolFilter) ([]*ai.MCPTool, error)
	GetToolCount(ctx context.Context, filter *ai.MCPToolFilter) (int, error)
	GetEnabledTools(ctx context.Context, category string) ([]*ai.MCPTool, error)
	UpdateTool(ctx context.Context, tool *ai.MCPTool) error
	DeleteTool(ctx context.Context, id string) error
	EnableTool(ctx context.Context, id string) error
	DisableTool(ctx context.Context, id string) error

	// Tool execution tracking
	CreateToolExecution(ctx context.Context, execution *ai.MCPToolExecution) error
	GetToolExecution(ctx context.Context, id string) (*ai.MCPToolExecution, error)
	GetToolExecutions(ctx context.Context, conversationID string, limit int, offset int) ([]*ai.MCPToolExecution, error)
	GetToolExecutionsByTool(ctx context.Context, toolID string, limit int, offset int) ([]*ai.MCPToolExecution, error)
	UpdateToolExecution(ctx context.Context, execution *ai.MCPToolExecution) error
	IncrementToolUsage(ctx context.Context, toolID string) error

	// Tool statistics and analytics
	GetToolUsageStats(ctx context.Context, startDate, endDate time.Time) ([]*ai.ToolUsageStats, error)
	GetToolSuccessRate(ctx context.Context, toolID string, days int) (float64, error)
	GetMostUsedTools(ctx context.Context, limit int, days int) ([]*ai.MCPTool, error)
	GetRecentToolExecutions(ctx context.Context, limit int) ([]*ai.MCPToolExecution, error)

	// Cleanup
	CleanupOldExecutions(ctx context.Context, days int) error
}
