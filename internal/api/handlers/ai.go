package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/gin-gonic/gin"
)

// AI-related handlers

// ChatWithAI handles chat requests to the AI system
func (h *Handlers) ChatWithAI(c *gin.Context) {
	var req ai.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get chat service from context or handlers
	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Perform chat
	response, err := chatService.Chat(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("Chat request failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat request failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CompleteText handles text completion requests
func (h *Handlers) CompleteText(c *gin.Context) {
	var req ai.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get chat service from context or handlers
	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Perform completion
	response, err := chatService.Complete(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("Completion request failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Completion request failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProviders returns information about available AI providers
func (h *Handlers) GetProviders(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	providers := llmManager.GetProviders(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetModels returns available models from all providers
func (h *Handlers) GetModels(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	models, err := llmManager.GetModels(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve models"})
		return
	}

	// Group models by provider
	modelsByProvider := make(map[string][]ai.ModelInfo)
	for _, model := range models {
		modelsByProvider[model.Provider] = append(modelsByProvider[model.Provider], model)
	}

	c.JSON(http.StatusOK, gin.H{
		"models":             models,
		"models_by_provider": modelsByProvider,
		"total_count":        len(models),
	})
}

// AnalyzeEntity analyzes specific entities and returns insights
func (h *Handlers) AnalyzeEntity(c *gin.Context) {
	entityID := c.Param("id")
	if entityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity ID is required"})
		return
	}

	var req ai.EntityAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Add the entity ID from the URL parameter
	if len(req.EntityIDs) == 0 {
		req.EntityIDs = []string{entityID}
	}

	// Set default analysis type if not specified
	if req.AnalysisType == "" {
		req.AnalysisType = "general"
	}

	// Get chat service
	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Perform analysis
	response, err := chatService.AnalyzeEntity(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("Entity analysis failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Entity analysis failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GenerateAutomation generates automation rules from natural language
func (h *Handlers) GenerateAutomation(c *gin.Context) {
	var req ai.AutomationGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description is required"})
		return
	}

	// Set default complexity if not specified
	if req.Complexity == "" {
		req.Complexity = "simple"
	}

	// Get chat service
	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Generate automation
	response, err := chatService.GenerateAutomation(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("Automation generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Automation generation failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSystemSummary generates an AI-powered system summary
func (h *Handlers) GetSystemSummary(c *gin.Context) {
	var req ai.SystemSummaryRequest

	// Parse query parameters
	req.IncludeEntities = c.DefaultQuery("include_entities", "true") == "true"
	req.IncludeRooms = c.DefaultQuery("include_rooms", "true") == "true"
	req.IncludeAutomation = c.DefaultQuery("include_automation", "true") == "true"
	req.IncludeAlerts = c.DefaultQuery("include_alerts", "true") == "true"
	req.DetailLevel = c.DefaultQuery("detail_level", "normal")

	// Parse entity types if provided
	if entityTypes := c.QueryArray("entity_types"); len(entityTypes) > 0 {
		req.EntityTypes = entityTypes
	}

	// Get chat service
	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Generate summary
	response, err := chatService.SummarizeSystem(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("System summary generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "System summary generation failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetAIStatistics returns AI system usage statistics
func (h *Handlers) GetAIStatistics(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	stats := llmManager.GetStatistics()
	c.JSON(http.StatusOK, stats)
}

// TestAIProvider tests connectivity to a specific AI provider
func (h *Handlers) TestAIProvider(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider name is required"})
		return
	}

	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Simple test chat request
	testReq := ai.ChatRequest{
		Messages: []ai.ChatMessage{
			{
				Role:    "user",
				Content: "Hello, this is a test message. Please respond with 'Test successful'.",
			},
		},
		Provider:    provider,
		MaxTokens:   50,
		Temperature: 0.1,
	}

	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Chat service not available"})
		return
	}

	response, err := chatService.Chat(c.Request.Context(), testReq)
	if err != nil {
		h.log.WithError(err).WithField("provider", provider).Error("Provider test failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":    "Provider test failed",
			"provider": provider,
			"details":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             "success",
		"provider":           provider,
		"response":           response.Message.Content,
		"processing_time_ms": response.ProcessingTimeMs,
		"tokens_used":        response.TokensUsed,
		"model":              response.Model,
	})
}

// ChatWithContext performs a chat with enhanced context from PMA system
func (h *Handlers) ChatWithContext(c *gin.Context) {
	var req ai.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Extract user ID from token/context if available
	userID := h.getUserIDFromContext(c)

	// Build conversation context from PMA system
	context := &ai.ConversationContext{
		UserID:    userID,
		SessionID: c.GetHeader("X-Session-ID"),
		Timestamp: time.Now(),
	}

	// Enhance context with actual entity and room data
	entities, err := h.repos.Entity.GetAll(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Warn("Failed to fetch entities for AI context")
	} else {
		// Convert entities to EntityContext slice
		entityContexts := make([]ai.EntityContext, 0, len(entities))
		for _, entity := range entities {
			entityContext := ai.EntityContext{
				ID:          entity.EntityID,
				Name:        entity.FriendlyName.String,
				Type:        entity.Domain,
				State:       entity.State.String,
				LastChanged: entity.LastUpdated,
			}

			// Parse attributes if they exist
			if len(entity.Attributes) > 0 {
				var attributes map[string]interface{}
				if err := json.Unmarshal(entity.Attributes, &attributes); err == nil {
					entityContext.Attributes = attributes
				}
			}

			entityContexts = append(entityContexts, entityContext)
		}
		context.Entities = entityContexts
	}

	rooms, err := h.repos.Room.GetAll(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Warn("Failed to fetch rooms for AI context")
	} else {
		// Convert rooms to RoomContext slice
		roomContexts := make([]ai.RoomContext, 0, len(rooms))
		for _, room := range rooms {
			roomContext := ai.RoomContext{
				ID:   fmt.Sprintf("room_%d", room.ID),
				Name: room.Name,
			}

			// Get entities in this room for entity count
			roomEntities, err := h.repos.Entity.GetByRoom(c.Request.Context(), room.ID)
			if err == nil {
				roomContext.EntityCount = len(roomEntities)

				// Convert room entities to EntityContext
				roomEntityContexts := make([]ai.EntityContext, 0, len(roomEntities))
				for _, entity := range roomEntities {
					entityContext := ai.EntityContext{
						ID:          entity.EntityID,
						Name:        entity.FriendlyName.String,
						Type:        entity.Domain,
						State:       entity.State.String,
						Room:        room.Name,
						LastChanged: entity.LastUpdated,
					}

					if len(entity.Attributes) > 0 {
						var attributes map[string]interface{}
						if err := json.Unmarshal(entity.Attributes, &attributes); err == nil {
							entityContext.Attributes = attributes
						}
					}

					roomEntityContexts = append(roomEntityContexts, entityContext)
				}
				roomContext.Entities = roomEntityContexts
			}

			roomContexts = append(roomContexts, roomContext)
		}
		context.Rooms = roomContexts
	}

	if req.Context == nil {
		req.Context = context
	} else {
		// Merge with provided context
		if req.Context.UserID == "" {
			req.Context.UserID = userID
		}
		if req.Context.SessionID == "" {
			req.Context.SessionID = context.SessionID
		}
		// Merge entity and room data
		if req.Context.Entities == nil {
			req.Context.Entities = context.Entities
		}
		if req.Context.Rooms == nil {
			req.Context.Rooms = context.Rooms
		}
	}

	// Get chat service
	chatService := h.getChatService()
	if chatService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Perform chat with enhanced context
	response, err := chatService.Chat(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("Context-aware chat failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat request failed"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AI Settings & Management Handlers

// GetAISettings retrieves AI provider configurations
func (h *Handlers) GetAISettings(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Get current AI configuration
	settings, err := llmManager.GetSettings(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get AI settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve AI settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// SaveAISettings saves AI provider settings
func (h *Handlers) SaveAISettings(c *gin.Context) {
	var req ai.AISettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Save settings
	if err := llmManager.SaveSettings(c.Request.Context(), req); err != nil {
		h.log.WithError(err).Error("Failed to save AI settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save AI settings", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "AI settings saved successfully",
		"updated_at": time.Now(),
	})
}

// TestAIConnection tests AI provider connectivity
func (h *Handlers) TestAIConnection(c *gin.Context) {
	var req ai.AIConnectionTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	// Test connection
	result, err := llmManager.TestConnection(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).Error("Connection test failed")
		c.JSON(http.StatusOK, gin.H{
			"success":   false,
			"message":   "Connection test failed",
			"error":     err.Error(),
			"tested_at": time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Ollama Process Management Handlers

// GetOllamaStatus gets Ollama process status
func (h *Handlers) GetOllamaStatus(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	status, err := llmManager.GetOllamaStatus(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get Ollama status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Ollama status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetOllamaMetrics gets Ollama resource usage metrics
func (h *Handlers) GetOllamaMetrics(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	metrics, err := llmManager.GetOllamaMetrics(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get Ollama metrics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Ollama metrics", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetOllamaHealth performs health check for Ollama service
func (h *Handlers) GetOllamaHealth(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	health, err := llmManager.GetOllamaHealth(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get Ollama health")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Ollama health", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// StartOllamaProcess starts Ollama process
func (h *Handlers) StartOllamaProcess(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	result, err := llmManager.StartOllamaProcess(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to start Ollama process")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start Ollama process", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// StopOllamaProcess stops Ollama process
func (h *Handlers) StopOllamaProcess(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	result, err := llmManager.StopOllamaProcess(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to stop Ollama process")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop Ollama process", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RestartOllamaProcess restarts Ollama process
func (h *Handlers) RestartOllamaProcess(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	result, err := llmManager.RestartOllamaProcess(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to restart Ollama process")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart Ollama process", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOllamaMonitoring gets comprehensive monitoring data
func (h *Handlers) GetOllamaMonitoring(c *gin.Context) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service not available"})
		return
	}

	monitoring, err := llmManager.GetOllamaMonitoring(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get Ollama monitoring data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Ollama monitoring data", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, monitoring)
}

// Helper methods

// getChatService returns the chat service instance
func (h *Handlers) getChatService() *ai.ChatService {
	return h.chatService
}

// getLLMManager returns the LLM manager instance
func (h *Handlers) getLLMManager() *ai.LLMManager {
	return h.llmManager
}

// getUserIDFromContext extracts user ID from the request context
func (h *Handlers) getUserIDFromContext(c *gin.Context) string {
	// This would typically extract from JWT token or session
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

// Helper function to parse int query parameter
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// Helper function to parse float query parameter
func parseFloatQuery(c *gin.Context, key string, defaultValue float64) float64 {
	if value := c.Query(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
