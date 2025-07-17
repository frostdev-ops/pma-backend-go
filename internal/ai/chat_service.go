package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ChatService provides high-level AI chat functionality with PMA integration
type ChatService struct {
	manager          *LLMManager
	logger           *logrus.Logger
	contextExtractor ContextExtractor
	defaultModel     string
	systemPrompt     string
}

// NewChatService creates a new chat service instance
func NewChatService(manager *LLMManager, logger *logrus.Logger) *ChatService {
	systemPrompt := `You are the PMA (Personal Management Assistant) AI assistant. You help users manage their smart home, analyze data, and provide insights about their devices and automation.

You have access to:
- Real-time entity states from Home Assistant
- Room information and organization
- Historical data and patterns
- System status and health information
- User preferences and recent actions

Your responses should be:
- Helpful and informative
- Focused on home automation and management
- Practical and actionable
- Concise but complete
- Friendly and professional

When analyzing entities or suggesting automations, consider:
- User patterns and preferences
- Energy efficiency
- Safety and security
- Convenience and comfort
- Cost implications

Always provide specific, implementable suggestions when possible.`

	return &ChatService{
		manager:      manager,
		logger:       logger,
		systemPrompt: systemPrompt,
	}
}

// SetContextExtractor sets the context extractor for enriching chat requests
func (cs *ChatService) SetContextExtractor(extractor ContextExtractor) {
	cs.contextExtractor = extractor
}

// SetDefaultModel sets the default model to use for chat requests
func (cs *ChatService) SetDefaultModel(model string) {
	cs.defaultModel = model
}

// Chat performs a context-aware chat interaction
func (cs *ChatService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Enrich the request with context if available
	if cs.contextExtractor != nil && req.Context != nil && req.Context.UserID != "" {
		enrichedContext, err := cs.contextExtractor.ExtractContext(ctx, req.Context.UserID)
		if err != nil {
			cs.logger.WithError(err).Warn("Failed to extract context, continuing without enrichment")
		} else {
			req.Context = enrichedContext
		}
	}

	// Build the message chain with system prompt and context
	messages := cs.buildMessageChain(req)

	// Set up chat options
	opts := ChatOptions{
		Model:        req.Model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		Stream:       req.Stream,
		SystemPrompt: cs.systemPrompt,
		Provider:     req.Provider,
		Metadata:     req.Metadata,
	}

	// Use default model if none specified
	if opts.Model == "" && cs.defaultModel != "" {
		opts.Model = cs.defaultModel
	}

	// Call the LLM manager
	response, err := cs.manager.Chat(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("chat failed: %w", err)
	}

	// Add context information to response metadata
	if response.Metadata == nil {
		response.Metadata = make(map[string]string)
	}

	if req.Context != nil {
		response.Metadata["context_provided"] = "true"
		response.Metadata["entity_count"] = fmt.Sprintf("%d", len(req.Context.Entities))
		response.Metadata["room_count"] = fmt.Sprintf("%d", len(req.Context.Rooms))
	}

	return response, nil
}

// Complete performs a context-aware text completion
func (cs *ChatService) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Enrich the prompt with context information
	enrichedPrompt := cs.enrichPrompt(req.Prompt)

	// Set up completion options
	opts := CompletionOptions{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Stop:        req.Stop,
		Provider:    req.Provider,
		Metadata:    req.Metadata,
	}

	// Use default model if none specified
	if opts.Model == "" && cs.defaultModel != "" {
		opts.Model = cs.defaultModel
	}

	// Call the LLM manager
	response, err := cs.manager.Complete(ctx, enrichedPrompt, opts)
	if err != nil {
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	return response, nil
}

// AnalyzeEntity analyzes an entity and provides insights
func (cs *ChatService) AnalyzeEntity(ctx context.Context, req EntityAnalysisRequest) (*EntityAnalysisResponse, error) {
	// Build analysis prompt
	prompt := cs.buildEntityAnalysisPrompt(req)

	// Perform completion
	completionReq := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   2048,
		Temperature: 0.3, // Lower temperature for more consistent analysis
	}

	resp, err := cs.Complete(ctx, completionReq)
	if err != nil {
		return nil, err
	}

	// Parse the response into structured insights
	// For now, return a basic response - this could be enhanced with structured parsing
	analysis := &EntityAnalysisResponse{
		EntityID:     req.EntityIDs[0], // Simplified for single entity
		AnalysisType: req.AnalysisType,
		Insights: []AnalysisInsight{
			{
				Type:        "general",
				Title:       "AI Analysis",
				Description: resp.Text,
				Confidence:  0.8,
				Actionable:  true,
			},
		},
		Confidence:  0.8,
		ProcessedAt: time.Now(),
	}

	return analysis, nil
}

// GenerateAutomation generates automation rules from natural language description
func (cs *ChatService) GenerateAutomation(ctx context.Context, req AutomationGenerationRequest) (*AutomationGenerationResponse, error) {
	// Build automation generation prompt
	prompt := cs.buildAutomationPrompt(req)

	// Perform completion with higher token limit for detailed automations
	completionReq := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.4, // Balanced for creativity and consistency
	}

	resp, err := cs.Complete(ctx, completionReq)
	if err != nil {
		return nil, err
	}

	// Parse response into automation structure
	// For now, return a basic response - this could be enhanced with structured parsing
	automation := &AutomationGenerationResponse{
		Automations: []GeneratedAutomation{
			{
				ID:          fmt.Sprintf("ai_generated_%d", time.Now().Unix()),
				Name:        "AI Generated Automation",
				Description: req.Description,
				Complexity:  req.Complexity,
				Benefits:    []string{"Automated based on AI analysis"},
				HAConfig: map[string]interface{}{
					"automation": map[string]interface{}{
						"alias":       "AI Generated Automation",
						"description": resp.Text,
						"trigger":     []interface{}{},
						"action":      []interface{}{},
					},
				},
			},
		},
		Summary:     "Generated automation based on: " + req.Description,
		Confidence:  0.7,
		GeneratedAt: time.Now(),
	}

	return automation, nil
}

// SummarizeSystem generates a system status summary
func (cs *ChatService) SummarizeSystem(ctx context.Context, req SystemSummaryRequest) (*SystemSummaryResponse, error) {
	// Build system summary prompt
	prompt := cs.buildSystemSummaryPrompt(req)

	// Perform completion
	completionReq := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   1024,
		Temperature: 0.3, // Lower temperature for factual summaries
	}

	resp, err := cs.Complete(ctx, completionReq)
	if err != nil {
		return nil, err
	}

	// Create structured summary response
	summary := &SystemSummaryResponse{
		Summary:     resp.Text,
		Highlights:  []string{"System operational", "AI analysis available"},
		GeneratedAt: time.Now(),
	}

	return summary, nil
}

// Private helper methods

func (cs *ChatService) buildMessageChain(req ChatRequest) []ChatMessage {
	messages := make([]ChatMessage, 0)

	// Add context information as a system message if available
	if req.Context != nil {
		contextMsg := cs.buildContextMessage(req.Context)
		if contextMsg != "" {
			messages = append(messages, ChatMessage{
				Role:      "system",
				Content:   contextMsg,
				Timestamp: time.Now(),
			})
		}
	}

	// Add the user's messages
	messages = append(messages, req.Messages...)

	return messages
}

func (cs *ChatService) buildContextMessage(ctx *ConversationContext) string {
	if ctx == nil {
		return ""
	}

	var parts []string

	// Add Home Assistant context
	if ctx.HomeAssistant != nil {
		parts = append(parts, fmt.Sprintf("Home Assistant: %d entities, connected: %v, last sync: %s",
			ctx.HomeAssistant.EntityCount,
			ctx.HomeAssistant.Connected,
			ctx.HomeAssistant.LastSync.Format("2006-01-02 15:04:05")))
	}

	// Add entity information
	if len(ctx.Entities) > 0 {
		entitySummary := make([]string, 0, len(ctx.Entities))
		for _, entity := range ctx.Entities {
			entitySummary = append(entitySummary, fmt.Sprintf("%s (%s): %s", entity.Name, entity.Type, entity.State))
		}
		parts = append(parts, fmt.Sprintf("Current entities: %s", strings.Join(entitySummary, ", ")))
	}

	// Add room information
	if len(ctx.Rooms) > 0 {
		roomSummary := make([]string, 0, len(ctx.Rooms))
		for _, room := range ctx.Rooms {
			roomSummary = append(roomSummary, fmt.Sprintf("%s (%d entities)", room.Name, room.EntityCount))
		}
		parts = append(parts, fmt.Sprintf("Rooms: %s", strings.Join(roomSummary, ", ")))
	}

	// Add system status
	if ctx.SystemStatus != nil {
		parts = append(parts, fmt.Sprintf("System: CPU %.1f%%, Memory %.1f%%, Network %s",
			ctx.SystemStatus.CPUUsage,
			ctx.SystemStatus.MemoryUsage,
			ctx.SystemStatus.NetworkStatus))
	}

	if len(parts) == 0 {
		return ""
	}

	return "Current system context:\n" + strings.Join(parts, "\n")
}

func (cs *ChatService) enrichPrompt(prompt string) string {
	// Simple prompt enrichment - could be enhanced with actual context
	return fmt.Sprintf("Context: You are the PMA assistant helping with home automation.\n\nUser request: %s", prompt)
}

func (cs *ChatService) buildEntityAnalysisPrompt(req EntityAnalysisRequest) string {
	prompt := fmt.Sprintf(`Analyze the following entities for %s:

Entity IDs: %s
Analysis Type: %s
Time Range: %s to %s

Please provide insights about:
- Current state and behavior patterns
- Potential issues or anomalies
- Optimization opportunities
- Automation suggestions

Be specific and actionable in your analysis.`,
		req.AnalysisType,
		strings.Join(req.EntityIDs, ", "),
		req.AnalysisType,
		req.TimeRange.Start.Format("2006-01-02 15:04:05"),
		req.TimeRange.End.Format("2006-01-02 15:04:05"))

	return prompt
}

func (cs *ChatService) buildAutomationPrompt(req AutomationGenerationRequest) string {
	prompt := fmt.Sprintf(`Generate a Home Assistant automation based on this description:
"%s"

Requirements:
- Complexity level: %s
- Target entities: %s
- Target rooms: %s

Please provide:
1. A clear automation name and description
2. Appropriate triggers and conditions
3. Actions to perform
4. Benefits and potential risks
5. Testing recommendations

Format the response as a structured automation configuration.`,
		req.Description,
		req.Complexity,
		strings.Join(req.EntityIDs, ", "),
		strings.Join(req.RoomIDs, ", "))

	return prompt
}

func (cs *ChatService) buildSystemSummaryPrompt(req SystemSummaryRequest) string {
	prompt := "Provide a comprehensive system summary including:\n\n"

	if req.IncludeEntities {
		prompt += "- Entity status and availability\n"
	}
	if req.IncludeRooms {
		prompt += "- Room organization and occupancy\n"
	}
	if req.IncludeAutomation {
		prompt += "- Automation status and recent activity\n"
	}
	if req.IncludeAlerts {
		prompt += "- System alerts and notifications\n"
	}

	prompt += fmt.Sprintf("\nDetail level: %s", req.DetailLevel)
	prompt += "\n\nPlease provide a clear, organized summary with actionable insights."

	return prompt
}
