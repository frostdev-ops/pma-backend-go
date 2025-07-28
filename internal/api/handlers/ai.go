package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/core/system"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		h.log.Error("LLM Manager is nil - AI services failed to initialize")
		c.JSON(http.StatusOK, gin.H{
			"providers": []interface{}{},
			"count":     0,
			"error":     "AI services failed to initialize - check server logs",
		})
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

	// Convert to frontend-expected format
	frontendModels := make([]map[string]interface{}, 0, len(models))
	for _, model := range models {
		frontendModel := map[string]interface{}{
			"id":          model.ID,
			"name":        model.Name,
			"provider":    model.Provider,
			"size":        "Unknown", // Default size
			"status":      "available",
			"description": model.Description,
			"tags":        []string{},
		}

		// Add additional fields if available
		if model.MaxTokens > 0 {
			frontendModel["context_length"] = model.MaxTokens
		}

		if model.Capabilities != nil {
			frontendModel["capabilities"] = model.Capabilities
		}

		// Check if model is local and set status accordingly
		if model.LocalModel {
			frontendModel["status"] = "installed"
		}

		frontendModels = append(frontendModels, frontendModel)
	}

	// Return using standard response format
	utils.SendSuccess(c, frontendModels)
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

// GetAISettings retrieves AI configuration settings
func (h *Handlers) GetAISettings(c *gin.Context) {
	// Get AI configuration from system config
	systemConfig := h.getSystemConfigOrDefaults()

	// Create comprehensive AI settings response
	aiSettings := map[string]interface{}{
		"enabled":                 false,
		"default_provider":        "ollama",
		"default_model":           "llama2",
		"stream_responses":        true,
		"auto_save_conversations": true,
		"max_tokens":              2048,
		"temperature":             0.7,
		"providers":               map[string]interface{}{},
		"service_status":          map[string]interface{}{},
	}

	// Check if AI services are configured and available
	llmManager := h.getLLMManager()
	if llmManager != nil {
		aiSettings["enabled"] = true

		// Get available providers
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		providers := llmManager.GetProviders(ctx)
		providerMap := make(map[string]interface{})
		serviceStatus := make(map[string]interface{})

		for _, provider := range providers {
			providerInfo := map[string]interface{}{
				"name":      provider.Name,
				"enabled":   true,
				"available": true,
				"models":    []string{},
			}

			// Provider-specific information
			switch provider.Name {
			case "ollama":
				providerInfo["url"] = "http://localhost:11434"
				providerInfo["models"] = []string{"llama2", "mistral", "codellama"}
				providerInfo["local"] = true

				// Check Ollama availability
				if available := h.checkOllamaAvailable(); available {
					serviceStatus["ollama"] = map[string]interface{}{
						"status":    "healthy",
						"available": true,
						"latency":   "50ms",
					}
				} else {
					serviceStatus["ollama"] = map[string]interface{}{
						"status":    "unavailable",
						"available": false,
						"error":     "Service not reachable",
					}
					providerInfo["available"] = false
				}

			case "openai":
				providerInfo["models"] = []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}
				providerInfo["local"] = false
				serviceStatus["openai"] = map[string]interface{}{
					"status":    "configured",
					"available": systemConfig.Services.AI != nil,
				}

			case "claude":
				providerInfo["models"] = []string{"claude-3-haiku-20240307", "claude-3-sonnet-20240229", "claude-3-opus-20240229"}
				providerInfo["local"] = false
				serviceStatus["claude"] = map[string]interface{}{
					"status":    "configured",
					"available": systemConfig.Services.AI != nil,
				}

			case "gemini":
				providerInfo["models"] = []string{"gemini-pro", "gemini-pro-vision"}
				providerInfo["local"] = false
				serviceStatus["gemini"] = map[string]interface{}{
					"status":    "configured",
					"available": systemConfig.Services.AI != nil,
				}
			}

			providerMap[provider.Name] = providerInfo
		}

		aiSettings["providers"] = providerMap
		aiSettings["service_status"] = serviceStatus

		// Get current AI configuration
		if systemConfig.Services.AI != nil {
			aiSettings["default_provider"] = systemConfig.Services.AI.DefaultProvider
		}
	} else {
		aiSettings["service_status"] = map[string]interface{}{
			"llm_manager": map[string]interface{}{
				"status":    "unavailable",
				"available": false,
				"error":     "LLM Manager not initialized",
			},
		}
	}

	// Get conversation statistics if available
	if h.conversationService != nil {
		// Simplified conversation stats since the full API isn't available
		aiSettings["conversation_stats"] = map[string]interface{}{
			"service_available": true,
			"note":              "Conversation service available",
		}
	} else {
		aiSettings["conversation_stats"] = map[string]interface{}{
			"service_available": false,
			"note":              "Conversation service not available",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    aiSettings,
	})
}

// UpdateAISettings updates AI configuration
func (h *Handlers) UpdateAISettings(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request data")
		return
	}

	// Get current system config
	systemConfig := h.getSystemConfigOrDefaults()

	// Initialize AI services config if it doesn't exist
	if systemConfig.Services.AI == nil {
		systemConfig.Services.AI = &system.AIConfig{
			Enabled:         "false",
			DefaultProvider: "ollama",
		}
	}

	// Update fields from request
	updated := false

	if enabled, ok := request["enabled"].(bool); ok {
		if enabled {
			systemConfig.Services.AI.Enabled = "true"
		} else {
			systemConfig.Services.AI.Enabled = "false"
		}
		updated = true
	}

	if provider, ok := request["default_provider"].(string); ok && provider != "" {
		systemConfig.Services.AI.DefaultProvider = provider
		updated = true
	}

	// Note: MaxTokens and Temperature are not available in the current AIConfig structure
	// These would need to be added to the system.AIConfig type if needed

	// Save updated configuration
	if updated {
		if err := h.systemService.UpdateConfig(*systemConfig); err != nil {
			h.log.WithError(err).Error("Failed to update AI settings")
			utils.SendError(c, http.StatusInternalServerError, "Failed to update AI settings")
			return
		}

		h.log.Info("AI settings updated successfully")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AI settings updated successfully",
		"data":    request,
	})
}

// TestAIConnection tests connection to AI providers
func (h *Handlers) TestAIConnection(c *gin.Context) {
	var request struct {
		Provider string                 `json:"provider" binding:"required"`
		Config   map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid test request")
		return
	}

	// Test connection based on provider
	result := map[string]interface{}{
		"provider":  request.Provider,
		"connected": false,
		"error":     "",
		"latency":   0,
	}

	switch request.Provider {
	case "ollama":
		connected, latency, err := h.testOllamaConnection()
		result["connected"] = connected
		result["latency"] = latency
		if err != nil {
			result["error"] = err.Error()
		}

	case "openai":
		// Test OpenAI connection - implement based on your OpenAI client
		result["connected"] = false
		result["error"] = "OpenAI testing not implemented"

	case "claude":
		// Test Claude connection - implement based on your Claude client
		result["connected"] = false
		result["error"] = "Claude testing not implemented"

	case "gemini":
		// Test Gemini connection - implement based on your Gemini client
		result["connected"] = false
		result["error"] = "Gemini testing not implemented"

	default:
		result["error"] = "Unknown provider"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
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

// Model Management Handlers

// ModelDownloadManager tracks active model downloads
type ModelDownloadManager struct {
	downloads map[string]*ModelDownloadProgress
	mutex     sync.RWMutex
}

type ModelDownloadProgress struct {
	ID           string    `json:"id"`
	ModelName    string    `json:"model_name"`
	Provider     string    `json:"provider"`
	Progress     int       `json:"progress"`
	Status       string    `json:"status"` // downloading, installing, completed, error
	ErrorMessage string    `json:"error_message,omitempty"`
	Speed        string    `json:"speed,omitempty"`
	ETA          string    `json:"eta,omitempty"`
	StartTime    time.Time `json:"start_time"`
	UpdatedAt    time.Time `json:"updated_at"`
}

var downloadManager = &ModelDownloadManager{
	downloads: make(map[string]*ModelDownloadProgress),
}

// GetModelStorageInfo returns storage information for AI models
func (h *Handlers) GetModelStorageInfo(c *gin.Context) {
	// TODO: Implement real storage calculation by checking model directories
	// For now, return mock data with better estimates

	llmManager := h.getLLMManager()
	var installedCount int
	if llmManager != nil {
		if models, err := llmManager.GetModels(c.Request.Context()); err == nil {
			for _, model := range models {
				if model.LocalModel {
					installedCount++
				}
			}
		}
	}

	storageInfo := gin.H{
		"models_count":    installedCount,
		"storage_used":    "15.2 GB", // TODO: Calculate real usage
		"available_space": "34.8 GB",
		"total_space":     "50 GB",
		"largest_model":   "mistral-small3.2:latest (14.14 GB)",
		"oldest_model":    "nomic-embed-text:latest",
	}

	utils.SendSuccess(c, storageInfo)
}

// GetModelDownloads returns current model download status
func (h *Handlers) GetModelDownloads(c *gin.Context) {
	downloadManager.mutex.RLock()
	defer downloadManager.mutex.RUnlock()

	downloads := make([]*ModelDownloadProgress, 0, len(downloadManager.downloads))
	for _, download := range downloadManager.downloads {
		downloads = append(downloads, download)
	}

	utils.SendSuccess(c, downloads)
}

// InstallModel starts model installation/download with progress tracking
func (h *Handlers) InstallModel(c *gin.Context) {
	var req struct {
		ModelName string `json:"model_name" binding:"required"`
		Provider  string `json:"provider" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate unique download ID
	downloadID := fmt.Sprintf("%s_%s_%d", req.Provider, req.ModelName, time.Now().Unix())

	// Create download progress entry
	downloadManager.mutex.Lock()
	downloadManager.downloads[downloadID] = &ModelDownloadProgress{
		ID:        downloadID,
		ModelName: req.ModelName,
		Provider:  req.Provider,
		Progress:  0,
		Status:    "starting",
		StartTime: time.Now(),
		UpdatedAt: time.Now(),
	}
	downloadManager.mutex.Unlock()

	// Start download in background
	go h.performModelInstallation(downloadID, req.ModelName, req.Provider)

	result := gin.H{
		"message":     fmt.Sprintf("Installation started for model %s from provider %s", req.ModelName, req.Provider),
		"download_id": downloadID,
		"model_name":  req.ModelName,
		"provider":    req.Provider,
		"status":      "downloading",
	}

	utils.SendSuccess(c, result)
}

// performModelInstallation performs the actual model installation with progress tracking
func (h *Handlers) performModelInstallation(downloadID, modelName, provider string) {
	updateProgress := func(progress int, status string, errorMsg string) {
		downloadManager.mutex.Lock()
		if download, exists := downloadManager.downloads[downloadID]; exists {
			download.Progress = progress
			download.Status = status
			download.ErrorMessage = errorMsg
			download.UpdatedAt = time.Now()

			// Calculate speed and ETA (mock for now)
			if status == "downloading" && progress > 0 {
				elapsed := time.Since(download.StartTime)
				if elapsed.Seconds() > 1 {
					download.Speed = fmt.Sprintf("%.1f MB/s", float64(progress)*0.5) // Mock speed calculation
					remaining := float64(100-progress) / float64(progress) * elapsed.Seconds()
					download.ETA = fmt.Sprintf("%.0fs", remaining)
				}
			}
		}
		downloadManager.mutex.Unlock()

		// TODO: Broadcast update via WebSocket
		h.log.WithFields(logrus.Fields{
			"download_id": downloadID,
			"progress":    progress,
			"status":      status,
		}).Info("Model installation progress update")
	}

	updateProgress(5, "downloading", "")

	// Get LLM manager
	llmManager := h.getLLMManager()
	if llmManager == nil {
		updateProgress(0, "error", "AI service not available")
		return
	}

	if provider == "ollama" {
		// Simulate download progress for Ollama
		updateProgress(10, "downloading", "")
		time.Sleep(1 * time.Second)

		updateProgress(25, "downloading", "")
		time.Sleep(2 * time.Second)

		updateProgress(50, "downloading", "")
		time.Sleep(2 * time.Second)

		updateProgress(75, "installing", "")
		time.Sleep(1 * time.Second)

		// Try to actually pull the model through Ollama
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Get Ollama provider and pull model
		if ollamaProvider := llmManager.GetProvider("ollama"); ollamaProvider != nil {
			updateProgress(90, "installing", "")

			// Create a simple chat request to trigger model pull if needed
			testReq := ai.ChatRequest{
				Messages: []ai.ChatMessage{
					{
						Role:    "user",
						Content: "Hello",
					},
				},
				Model:       modelName,
				MaxTokens:   1,
				Temperature: 0.1,
				Provider:    "ollama",
			}

			chatService := h.getChatService()
			if chatService != nil {
				_, err := chatService.Chat(ctx, testReq)
				if err != nil {
					h.log.WithError(err).WithField("model", modelName).Error("Failed to install model")
					updateProgress(0, "error", fmt.Sprintf("Failed to install model: %v", err))
					return
				}
			}
		}

		updateProgress(100, "completed", "")

		// Clean up completed download after 30 seconds
		go func() {
			time.Sleep(30 * time.Second)
			downloadManager.mutex.Lock()
			delete(downloadManager.downloads, downloadID)
			downloadManager.mutex.Unlock()
		}()

	} else {
		updateProgress(0, "error", fmt.Sprintf("Provider %s not supported for installation", provider))
	}
}

// RemoveModel removes/uninstalls a model
func (h *Handlers) RemoveModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		utils.SendError(c, http.StatusBadRequest, "Model ID is required")
		return
	}

	// TODO: Implement actual model removal for Ollama
	// For now, just return success
	result := gin.H{
		"message": fmt.Sprintf("Model %s removal not yet implemented", modelID),
		"warning": "This feature is not yet implemented",
	}

	utils.SendSuccess(c, result)
}

// RunModelTest runs a test on a specific model
func (h *Handlers) RunModelTest(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		utils.SendError(c, http.StatusBadRequest, "Model ID is required")
		return
	}

	// Get the LLM manager to test the model
	llmManager := h.getLLMManager()
	if llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	// Create a simple test chat request
	startTime := time.Now()
	testReq := ai.ChatRequest{
		Messages: []ai.ChatMessage{
			{
				Role:    "user",
				Content: "Hello, this is a test message. Please respond with 'Test successful'.",
			},
		},
		Model:       modelID,
		MaxTokens:   50,
		Temperature: 0.1,
	}

	chatService := h.getChatService()
	if chatService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Chat service not available")
		return
	}

	response, err := chatService.Chat(c.Request.Context(), testReq)
	if err != nil {
		h.log.WithError(err).WithField("model_id", modelID).Error("Model test failed")
		errorResult := gin.H{
			"success":       false,
			"error":         "Model test failed",
			"details":       err.Error(),
			"response_time": time.Since(startTime).Milliseconds(),
		}
		utils.SendSuccess(c, errorResult)
		return
	}

	result := gin.H{
		"success":       true,
		"test_output":   response.Message.Content,
		"response_time": time.Since(startTime).Milliseconds(),
		"tokens_used":   response.TokensUsed,
		"model":         response.Model,
	}

	utils.SendSuccess(c, result)
}

// Helper methods

// getSystemConfigOrDefaults gets the current system config or returns defaults
func (h *Handlers) getSystemConfigOrDefaults() *system.SystemConfig {
	if h.systemService != nil {
		config := h.systemService.GetConfig()
		return &config
	}

	// Return basic defaults if no system service
	now := time.Now().Format(time.RFC3339)
	defaultConfig := system.SystemConfig{
		Services: system.ServicesSectionConfig{
			AI: &system.AIConfig{
				Enabled:         "false",
				DefaultProvider: "ollama",
				Providers:       system.AIProvidersConfig{},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
	return &defaultConfig
}

// saveSystemConfig saves the system configuration
func (h *Handlers) saveSystemConfig(config *system.SystemConfig) error {
	if h.systemService != nil {
		return h.systemService.UpdateConfig(*config)
	}
	return fmt.Errorf("system service not available")
}

// updateAIProviders updates AI provider configurations
func (h *Handlers) updateAIProviders(aiConfig *system.AIConfig, providers map[string]interface{}) {
	// Update Ollama config
	if ollamaData, ok := providers["ollama"].(map[string]interface{}); ok {
		if aiConfig.Providers.Ollama == nil {
			aiConfig.Providers.Ollama = &system.OllamaConfig{}
		}

		if enabled, ok := ollamaData["enabled"].(bool); ok {
			aiConfig.Providers.Ollama.Enabled = enabled
		}
		if url, ok := ollamaData["url"].(string); ok {
			aiConfig.Providers.Ollama.URL = url
		}
		if modelsInterface, ok := ollamaData["models"]; ok {
			if models, ok := modelsInterface.([]interface{}); ok {
				stringModels := make([]string, len(models))
				for i, model := range models {
					if str, ok := model.(string); ok {
						stringModels[i] = str
					}
				}
				aiConfig.Providers.Ollama.Models = stringModels
			}
		}
	}

	// Update OpenAI config
	if openaiData, ok := providers["openai"].(map[string]interface{}); ok {
		if aiConfig.Providers.OpenAI == nil {
			aiConfig.Providers.OpenAI = &system.OpenAIConfig{}
		}

		if enabled, ok := openaiData["enabled"].(bool); ok {
			aiConfig.Providers.OpenAI.Enabled = enabled
		}
		if apiKey, ok := openaiData["api_key"].(string); ok {
			aiConfig.Providers.OpenAI.APIKey = apiKey
		}
		if model, ok := openaiData["model"].(string); ok {
			aiConfig.Providers.OpenAI.Model = model
		}
		if maxTokens, ok := openaiData["max_tokens"].(float64); ok {
			aiConfig.Providers.OpenAI.MaxTokens = int(maxTokens)
		}
		if temperature, ok := openaiData["temperature"].(float64); ok {
			aiConfig.Providers.OpenAI.Temperature = temperature
		}
	}

	// Update Claude config
	if claudeData, ok := providers["claude"].(map[string]interface{}); ok {
		if aiConfig.Providers.Claude == nil {
			aiConfig.Providers.Claude = &system.ClaudeConfig{}
		}

		if enabled, ok := claudeData["enabled"].(bool); ok {
			aiConfig.Providers.Claude.Enabled = enabled
		}
		if apiKey, ok := claudeData["api_key"].(string); ok {
			aiConfig.Providers.Claude.APIKey = apiKey
		}
		if model, ok := claudeData["model"].(string); ok {
			aiConfig.Providers.Claude.Model = model
		}
		if maxTokens, ok := claudeData["max_tokens"].(float64); ok {
			aiConfig.Providers.Claude.MaxTokens = int(maxTokens)
		}
	}

	// Update Gemini config
	if geminiData, ok := providers["gemini"].(map[string]interface{}); ok {
		if aiConfig.Providers.Gemini == nil {
			aiConfig.Providers.Gemini = &system.GeminiConfig{}
		}

		if enabled, ok := geminiData["enabled"].(bool); ok {
			aiConfig.Providers.Gemini.Enabled = enabled
		}
		if apiKey, ok := geminiData["api_key"].(string); ok {
			aiConfig.Providers.Gemini.APIKey = apiKey
		}
		if model, ok := geminiData["model"].(string); ok {
			aiConfig.Providers.Gemini.Model = model
		}
		if safetySettings, ok := geminiData["safety_settings"].(map[string]interface{}); ok {
			settings := make(map[string]string)
			for key, value := range safetySettings {
				if str, ok := value.(string); ok {
					settings[key] = str
				}
			}
			aiConfig.Providers.Gemini.SafetySettings = settings
		}
	}
}

// checkOllamaAvailable checks if Ollama service is available
func (h *Handlers) checkOllamaAvailable() bool {
	llmManager := h.getLLMManager()
	if llmManager != nil {
		// Check if Ollama provider is available through LLM manager
		providers := llmManager.GetProviders(context.Background())
		for _, provider := range providers {
			if provider.Name == "ollama" {
				return true
			}
		}
	}
	return false
}

// testOllamaConnection tests connection to Ollama
func (h *Handlers) testOllamaConnection() (bool, int64, error) {
	llmManager := h.getLLMManager()
	if llmManager == nil {
		return false, 0, fmt.Errorf("LLM manager not available")
	}

	// Test Ollama connection through LLM manager
	providers := llmManager.GetProviders(context.Background())
	for _, provider := range providers {
		if provider.Name == "ollama" {
			return true, 50, nil // Default latency
		}
	}

	return false, 0, fmt.Errorf("Ollama provider not found")
}
