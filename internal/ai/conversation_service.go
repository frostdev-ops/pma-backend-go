package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ConversationRepositoryInterface defines the methods needed for conversation persistence
type ConversationRepositoryInterface interface {
	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, id string, userID string) (*Conversation, error)
	GetConversations(ctx context.Context, filter *ConversationFilter) ([]*Conversation, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	DeleteConversation(ctx context.Context, id string, userID string) error
	ArchiveConversation(ctx context.Context, id string, userID string) error
	UnarchiveConversation(ctx context.Context, id string, userID string) error
	CreateMessage(ctx context.Context, msg *ConversationMessage) error
	GetConversationMessages(ctx context.Context, conversationID string, limit int, offset int) ([]*ConversationMessage, error)
	CreateOrUpdateAnalytics(ctx context.Context, analytics *ConversationAnalytics) error
	GetConversationAnalytics(ctx context.Context, conversationID string, date time.Time) (*ConversationAnalytics, error)
	GetGlobalStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (*ConversationStatistics, error)
	CleanupOldConversations(ctx context.Context, days int) error
	CleanupOldMessages(ctx context.Context, days int) error
	CleanupOldAnalytics(ctx context.Context, days int) error
}

// MCPRepositoryInterface defines the methods needed for MCP tool management
type MCPRepositoryInterface interface {
	GetToolByName(ctx context.Context, name string) (*MCPTool, error)
	GetEnabledTools(ctx context.Context, category string) ([]*MCPTool, error)
	CreateToolExecution(ctx context.Context, execution *MCPToolExecution) error
	IncrementToolUsage(ctx context.Context, toolID string) error
	CleanupOldExecutions(ctx context.Context, days int) error
}

// ConversationService provides enhanced conversation management with persistence and MCP support
type ConversationService struct {
	llmManager       *LLMManager
	conversationRepo ConversationRepositoryInterface
	mcpRepo          MCPRepositoryInterface
	toolExecutor     *MCPToolExecutor
	logger           *logrus.Logger
	contextExtractor ContextExtractor
	defaultProvider  string
	defaultModel     string
	systemPrompt     string
}

// NewConversationService creates a new conversation service
func NewConversationService(
	llmManager *LLMManager,
	conversationRepo ConversationRepositoryInterface,
	mcpRepo MCPRepositoryInterface,
	toolExecutor *MCPToolExecutor,
	logger *logrus.Logger,
) *ConversationService {
	systemPrompt := `You are the PMA smart home assistant. You manage a complete Home Assistant system with 28+ tools for device control, room management, automation, and monitoring.

IMPORTANT: This system is already set up with devices and rooms. Use tools to discover everything - never ask users to describe their setup.

Key tools include:
- Device discovery: find_devices_by_name, search_devices, get_device_details
- Room control: get_all_rooms, get_room_status, control_room  
- Device control: control_multiple_devices, toggle_devices, set_brightness
- System setup: analyze_system_setup, suggest_automations, assign_entity_to_room
- Home Assistant: get_entity_state, set_entity_state, execute_scene
- Monitoring: check_device_connectivity, get_sensor_readings

Always use tools first to get current information, then provide specific recommendations based on real data. Be proactive in using tools to help users manage their smart home effectively.`

	return &ConversationService{
		llmManager:       llmManager,
		conversationRepo: conversationRepo,
		mcpRepo:          mcpRepo,
		toolExecutor:     toolExecutor,
		logger:           logger,
		systemPrompt:     systemPrompt,
		defaultProvider:  "auto",
		defaultModel:     "",
	}
}

// SetContextExtractor sets the context extractor for enriching conversations
func (cs *ConversationService) SetContextExtractor(extractor ContextExtractor) {
	cs.contextExtractor = extractor
}

// SetDefaults sets default provider and model
func (cs *ConversationService) SetDefaults(provider, model string) {
	cs.defaultProvider = provider
	cs.defaultModel = model
}

// CreateConversation creates a new persistent conversation
func (cs *ConversationService) CreateConversation(ctx context.Context, userID string, req *CreateConversationRequest) (*Conversation, error) {
	conversation := &Conversation{
		ID:           uuid.New().String(),
		UserID:       userID,
		Title:        req.Title,
		Provider:     cs.defaultProvider,
		Temperature:  0.7,
		MaxTokens:    2000,
		ContextData:  req.ContextData,
		Metadata:     req.Metadata,
		MessageCount: 0,
		Archived:     false,
	}

	// Apply request overrides
	if req.SystemPrompt != nil {
		conversation.SystemPrompt = req.SystemPrompt
	}
	if req.Provider != nil {
		conversation.Provider = *req.Provider
	}
	if req.Model != nil {
		conversation.Model = req.Model
	}
	if req.Temperature != nil {
		conversation.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		conversation.MaxTokens = *req.MaxTokens
	}

	err := cs.conversationRepo.CreateConversation(ctx, conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	cs.logger.WithFields(logrus.Fields{
		"conversation_id": conversation.ID,
		"user_id":         userID,
		"title":           conversation.Title,
	}).Info("Created new conversation")

	return conversation, nil
}

// GetConversation retrieves a conversation by ID
func (cs *ConversationService) GetConversation(ctx context.Context, userID, conversationID string) (*Conversation, error) {
	return cs.conversationRepo.GetConversation(ctx, conversationID, userID)
}

// GetConversations retrieves conversations for a user with filtering
func (cs *ConversationService) GetConversations(ctx context.Context, userID string, filter *ConversationFilter) ([]*Conversation, error) {
	if filter == nil {
		filter = &ConversationFilter{}
	}
	filter.UserID = &userID

	return cs.conversationRepo.GetConversations(ctx, filter)
}

// UpdateConversation updates conversation settings
func (cs *ConversationService) UpdateConversation(ctx context.Context, userID, conversationID string, req *UpdateConversationRequest) (*Conversation, error) {
	conversation, err := cs.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Apply updates
	if req.Title != nil {
		conversation.Title = *req.Title
	}
	if req.SystemPrompt != nil {
		conversation.SystemPrompt = req.SystemPrompt
	}
	if req.Provider != nil {
		conversation.Provider = *req.Provider
	}
	if req.Model != nil {
		conversation.Model = req.Model
	}
	if req.Temperature != nil {
		conversation.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		conversation.MaxTokens = *req.MaxTokens
	}
	if req.ContextData != nil {
		conversation.ContextData = req.ContextData
	}
	if req.Metadata != nil {
		conversation.Metadata = req.Metadata
	}
	if req.Archived != nil {
		conversation.Archived = *req.Archived
	}

	err = cs.conversationRepo.UpdateConversation(ctx, conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	return conversation, nil
}

// DeleteConversation deletes a conversation
func (cs *ConversationService) DeleteConversation(ctx context.Context, userID, conversationID string) error {
	return cs.conversationRepo.DeleteConversation(ctx, conversationID, userID)
}

// GetConversationMessages retrieves messages for a conversation
func (cs *ConversationService) GetConversationMessages(ctx context.Context, userID, conversationID string, limit, offset int) ([]*ConversationMessage, error) {
	// Verify user has access to conversation
	_, err := cs.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("access denied or conversation not found: %w", err)
	}

	return cs.conversationRepo.GetConversationMessages(ctx, conversationID, limit, offset)
}

// SendMessage sends a message in a conversation and gets an AI response
func (cs *ConversationService) SendMessage(ctx context.Context, userID, conversationID string, req *SendMessageRequest) (*EnhancedChatResponse, error) {
	startTime := time.Now()

	// Get conversation
	conversation, err := cs.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Create user message
	userMessage := &ConversationMessage{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "user",
		Content:        req.Content,
		Metadata:       req.Metadata,
	}

	if req.Role != nil {
		userMessage.Role = *req.Role
	}

	// Save user message
	err = cs.conversationRepo.CreateMessage(ctx, userMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Build conversation history for AI context
	messages, err := cs.buildConversationHistory(ctx, conversation, userMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to build conversation history: %w", err)
	}

	// Set up chat options
	chatOpts := ChatOptions{
		MaxTokens:    conversation.MaxTokens,
		Temperature:  conversation.Temperature,
		Provider:     conversation.Provider,
		SystemPrompt: cs.getSystemPrompt(conversation),
	}

	// Set model (handle pointer vs string)
	if conversation.Model != nil {
		chatOpts.Model = *conversation.Model
	}

	// Apply request overrides
	if req.Temperature != nil {
		chatOpts.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		chatOpts.MaxTokens = *req.MaxTokens
	}

	// Use defaults if not set
	if chatOpts.Provider == "" || chatOpts.Provider == "auto" {
		chatOpts.Provider = cs.defaultProvider
	}
	if chatOpts.Model == "" && cs.defaultModel != "" {
		chatOpts.Model = cs.defaultModel
	}

	// Note: Tool support would be added here when the ChatOptions structure supports it

	// Get AI response
	response, err := cs.llmManager.Chat(ctx, messages, chatOpts)
	if err != nil {
		return nil, fmt.Errorf("AI chat failed: %w", err)
	}

	responseTime := time.Since(startTime)

	// Create assistant message
	assistantMessage := &ConversationMessage{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response.Message.Content,
		TokensUsed:     response.TokensUsed.TotalTokens,
		ModelUsed:      &response.Model,
		ProviderUsed:   &response.Provider,
		ResponseTimeMs: int(responseTime.Milliseconds()),
		Metadata:       make(map[string]interface{}),
	}

	// Handle tool calls if present (placeholder for future enhancement)
	var toolExecutions []MCPToolExecution
	// Note: Tool call support would be added here when the ChatResponse structure supports it

	// Save assistant message
	err = cs.conversationRepo.CreateMessage(ctx, assistantMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	// Calculate cost (placeholder - would integrate with actual pricing)
	cost := float64(response.TokensUsed.TotalTokens) * 0.0001 // $0.0001 per token

	// Create enhanced response
	enhancedResponse := &EnhancedChatResponse{
		ConversationID: conversationID,
		Message:        *assistantMessage,
		Response:       *response,
		ToolExecutions: toolExecutions,
		TokensUsed:     response.TokensUsed.TotalTokens,
		Cost:           cost,
		ResponseTime:   responseTime,
		Provider:       response.Provider,
		Model:          response.Model,
	}

	// Update analytics
	go cs.updateConversationAnalytics(conversationID, response.TokensUsed.TotalTokens, cost, responseTime)

	cs.logger.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"tokens_used":     response.TokensUsed,
		"response_time":   responseTime.Milliseconds(),
		"tool_calls":      len(toolExecutions),
	}).Info("Processed conversation message")

	return enhancedResponse, nil
}

// buildConversationHistory builds the message history for AI context
func (cs *ConversationService) buildConversationHistory(ctx context.Context, conversation *Conversation, newMessage *ConversationMessage) ([]ChatMessage, error) {
	// Get recent messages for context (limit to last 20 messages to manage token usage)
	recentMessages, err := cs.conversationRepo.GetConversationMessages(ctx, conversation.ID, 20, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	var messages []ChatMessage

	// Add system message if present
	systemPrompt := cs.getSystemPrompt(conversation)
	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add conversation history (reverse order since we got them DESC)
	for i := len(recentMessages) - 1; i >= 0; i-- {
		msg := recentMessages[i]
		chatMsg := ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Note: Tool calls would be added here when ChatMessage supports them

		messages = append(messages, chatMsg)
	}

	// Add new user message
	messages = append(messages, ChatMessage{
		Role:    newMessage.Role,
		Content: newMessage.Content,
	})

	return messages, nil
}

// getSystemPrompt gets the system prompt for a conversation
func (cs *ConversationService) getSystemPrompt(conversation *Conversation) string {
	if conversation.SystemPrompt != nil && *conversation.SystemPrompt != "" {
		return *conversation.SystemPrompt
	}
	return cs.systemPrompt
}

// executeToolCall executes an MCP tool call
func (cs *ConversationService) executeToolCall(ctx context.Context, conversationID, messageID string, toolCall ToolCall) (*MCPToolExecution, error) {
	if cs.toolExecutor == nil {
		return nil, fmt.Errorf("tool executor not available")
	}

	// Get tool definition
	tool, err := cs.mcpRepo.GetToolByName(ctx, toolCall.Name)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Execute tool
	execution, err := cs.toolExecutor.ExecuteTool(ctx, tool, toolCall.Function.Arguments)
	if err != nil {
		cs.logger.WithError(err).WithField("tool", toolCall.Name).Error("Tool execution failed")
	}

	// Create execution record
	mcpExecution := &MCPToolExecution{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		MessageID:      messageID,
		ToolID:         tool.ID,
		ToolName:       tool.Name,
		Parameters:     toolCall.Function.Arguments,
		Success:        execution != nil && execution.Success,
		CreatedAt:      time.Now(),
	}

	if execution != nil {
		mcpExecution.ExecutionTimeMs = execution.ExecutionTime
		if execution.Success {
			if resultStr, ok := execution.Result.(string); ok {
				mcpExecution.Result = &resultStr
			}
		} else if execution.Error != nil {
			mcpExecution.Error = execution.Error
		}
	}

	// Save execution record
	if saveErr := cs.mcpRepo.CreateToolExecution(ctx, mcpExecution); saveErr != nil {
		cs.logger.WithError(saveErr).Error("Failed to save tool execution")
	}

	// Increment tool usage
	if updateErr := cs.mcpRepo.IncrementToolUsage(ctx, tool.ID); updateErr != nil {
		cs.logger.WithError(updateErr).Error("Failed to increment tool usage")
	}

	return mcpExecution, err
}

// Note: convertMCPToolsToLLMTools would be implemented when LLMTool types are available

// updateConversationAnalytics updates conversation analytics asynchronously
func (cs *ConversationService) updateConversationAnalytics(conversationID string, tokensUsed int, cost float64, responseTime time.Duration) {
	ctx := context.Background()
	today := time.Now().Truncate(24 * time.Hour)

	// Get existing analytics or create new
	analytics, err := cs.conversationRepo.GetConversationAnalytics(ctx, conversationID, today)
	if err != nil {
		// Create new analytics
		analytics = &ConversationAnalytics{
			ID:                uuid.New().String(),
			ConversationID:    conversationID,
			TotalMessages:     1,
			TotalTokens:       tokensUsed,
			TotalCost:         cost,
			AvgResponseTimeMs: float64(responseTime.Milliseconds()),
			Date:              today,
		}
	} else {
		// Update existing analytics
		analytics.TotalMessages++
		analytics.TotalTokens += tokensUsed
		analytics.TotalCost += cost
		analytics.AvgResponseTimeMs = (analytics.AvgResponseTimeMs + float64(responseTime.Milliseconds())) / 2
	}

	if err := cs.conversationRepo.CreateOrUpdateAnalytics(ctx, analytics); err != nil {
		cs.logger.WithError(err).Error("Failed to update conversation analytics")
	}
}

// GetConversationStatistics retrieves conversation statistics
func (cs *ConversationService) GetConversationStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (*ConversationStatistics, error) {
	return cs.conversationRepo.GetGlobalStatistics(ctx, userID, startDate, endDate)
}

// ArchiveConversation archives a conversation
func (cs *ConversationService) ArchiveConversation(ctx context.Context, userID, conversationID string) error {
	return cs.conversationRepo.ArchiveConversation(ctx, conversationID, userID)
}

// UnarchiveConversation unarchives a conversation
func (cs *ConversationService) UnarchiveConversation(ctx context.Context, userID, conversationID string) error {
	return cs.conversationRepo.UnarchiveConversation(ctx, conversationID, userID)
}

// CleanupOldData cleans up old conversation data
func (cs *ConversationService) CleanupOldData(ctx context.Context, days int) error {
	if err := cs.conversationRepo.CleanupOldConversations(ctx, days); err != nil {
		return fmt.Errorf("failed to cleanup old conversations: %w", err)
	}

	if err := cs.conversationRepo.CleanupOldMessages(ctx, days*2); err != nil {
		return fmt.Errorf("failed to cleanup old messages: %w", err)
	}

	if err := cs.conversationRepo.CleanupOldAnalytics(ctx, days*3); err != nil {
		return fmt.Errorf("failed to cleanup old analytics: %w", err)
	}

	if err := cs.mcpRepo.CleanupOldExecutions(ctx, days); err != nil {
		return fmt.Errorf("failed to cleanup old tool executions: %w", err)
	}

	return nil
}

// GenerateConversationTitle generates a title for a conversation based on its content
func (cs *ConversationService) GenerateConversationTitle(ctx context.Context, conversationID string) (string, error) {
	// Get first few messages
	messages, err := cs.conversationRepo.GetConversationMessages(ctx, conversationID, 5, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) == 0 {
		return "New Conversation", nil
	}

	// Build content summary
	var content []string
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 0 {
			content = append(content, msg.Content)
		}
	}

	if len(content) == 0 {
		return "New Conversation", nil
	}

	// Create a prompt to generate title
	titlePrompt := fmt.Sprintf("Generate a short, descriptive title (3-6 words) for a conversation that starts with: %s", strings.Join(content[:1], " "))

	messages_for_ai := []ChatMessage{
		{Role: "system", Content: "You are a helpful assistant that generates concise, descriptive titles for conversations. Return only the title, no quotes or extra text."},
		{Role: "user", Content: titlePrompt},
	}

	chatOpts := ChatOptions{
		Provider:    cs.defaultProvider,
		MaxTokens:   50,
		Temperature: 0.7,
	}

	response, err := cs.llmManager.Chat(ctx, messages_for_ai, chatOpts)
	if err != nil {
		// Fallback to simple title
		firstMessage := content[0]
		if len(firstMessage) > 30 {
			firstMessage = firstMessage[:30] + "..."
		}
		return firstMessage, nil
	}

	title := strings.TrimSpace(response.Message.Content)
	if title == "" {
		return "New Conversation", nil
	}

	return title, nil
}
