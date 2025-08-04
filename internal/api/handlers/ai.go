package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// StreamlinedAI handlers for the simplified llama.cpp AI system

// ChatWithAI handles basic chat requests using the streamlined AI service with smart model selection
func (h *Handlers) ChatWithAI(c *gin.Context) {
	h.log.Info("🎯 ChatWithAI request received")

	var req ai.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.WithError(err).Error("❌ Failed to bind chat request JSON")
		utils.SendError(c, http.StatusBadRequest, "Invalid request")
		return
	}

	h.log.WithFields(logrus.Fields{
		"messageCount": len(req.Messages),
		"model":        req.Model,
		"maxTokens":    req.MaxTokens,
		"temperature":  req.Temperature,
		"stream":       req.Stream,
	}).Info("📝 Chat request details")

	if h.llmManager == nil {
		h.log.Error("❌ LLM Manager is nil - AI service not available")
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	// Use smart model selection if available
	var selectedModel string
	if h.smartModelSelector != nil {
		h.log.Info("🧠 Smart model selector available - analyzing complexity")

		// Analyze request complexity
		complexity := h.smartModelSelector.AnalyzeComplexity(req.Messages)
		selectedModel = h.smartModelSelector.SelectOptimalModel(complexity)

		h.log.WithFields(logrus.Fields{
			"complexity":    complexity,
			"selectedModel": selectedModel,
			"messageCount":  len(req.Messages),
			"originalModel": req.Model,
		}).Info("🎯 Smart model selection applied")
	} else {
		// Fallback to default model
		h.log.Warn("⚠️ Smart model selector not available - using fallback")
		selectedModel = req.Model
		if selectedModel == "" {
			selectedModel = "LFM2-1.2B"
		}
		h.log.WithField("selectedModel", selectedModel).Info("🔄 Using fallback model")
	}

	// Get available MCP tools from database - only for Q4, Q8, and full models
	var tools []ai.Tool
	var mcpTools []*ai.MCPTool
	shouldUseMCPTools := selectedModel != "LFM2-1.2B-Q2" // Q2 is too fast/simple for tools
	
	if shouldUseMCPTools && h.mcpToolExecutor != nil && h.repos.MCP != nil {
		toolCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		
		var err error
		mcpTools, err = h.mcpToolExecutor.GetAvailableTools(toolCtx, h.repos.MCP)
		if err != nil {
			h.log.WithError(err).Warn("Failed to get MCP tools, continuing without tools")
		} else {
			// Convert MCP tools to LLM tools format
			tools = ai.ConvertMCPToolsToLLMTools(mcpTools)
			h.log.WithFields(logrus.Fields{
				"tool_count": len(tools),
				"model":      selectedModel,
			}).Info("🔧 Loaded MCP tools for AI chat")
		}
	} else {
		h.log.WithField("model", selectedModel).Info("🚫 Skipping MCP tools for Q2 model (optimized for speed)")
	}

	// Create system prompt with tool descriptions
	systemPrompt := req.SystemPrompt
	if len(tools) > 0 {
		toolsSystemPrompt := ai.CreateToolsSystemPrompt(tools)
		if systemPrompt != "" {
			systemPrompt = systemPrompt + toolsSystemPrompt
		} else {
			systemPrompt = "You are PMA (Personal Management Assistant), a smart home automation assistant." + toolsSystemPrompt
		}
	}

	// Validate system prompt fits within model's context window
	systemPromptTokens := len(systemPrompt) / 4 // Rough estimate: ~4 chars per token
	modelContextWindow := 32768                 // LFM2-1.2B has 32K context window
	maxSystemPromptTokens := modelContextWindow / 4 // Reserve 75% for input/output, 25% for system prompt

	if systemPromptTokens > maxSystemPromptTokens {
		h.log.WithFields(logrus.Fields{
			"systemPromptTokens":    systemPromptTokens,
			"maxSystemPromptTokens": maxSystemPromptTokens,
			"toolCount":            len(tools),
			"model":                selectedModel,
		}).Warn("⚠️ System prompt exceeds recommended size, may impact response quality")
		
		// Truncate system prompt if it's too large (emergency fallback)
		if systemPromptTokens > modelContextWindow/2 { // If over 50% of context
			maxChars := (modelContextWindow / 2) * 4
			systemPrompt = systemPrompt[:maxChars] + "\n[System prompt truncated due to length]"
			h.log.Warn("🔪 System prompt truncated to prevent context overflow")
		}
	}

	h.log.WithFields(logrus.Fields{
		"systemPromptTokens": systemPromptTokens,
		"toolCount":         len(tools),
		"model":             selectedModel,
	}).Info("📊 System prompt token analysis")

	// Set appropriate MaxTokens based on model selection, tools, and system prompt size
	maxTokens := req.MaxTokens
	if h.smartModelSelector != nil {
		// Get model config to use its optimized MaxTokens
		if modelConfig, exists := h.smartModelSelector.GetModelConfig(selectedModel); exists {
			// Use model's MaxTokens if it's higher than request, especially when tools are present
			if len(tools) > 0 || modelConfig.MaxTokens > maxTokens {
				maxTokens = modelConfig.MaxTokens
				
				// Adjust maxTokens to account for system prompt size
				// Leave room for: system prompt + user input + response
				availableTokens := modelContextWindow - systemPromptTokens - 1000 // Reserve 1000 for user input
				if maxTokens > availableTokens && availableTokens > 0 {
					maxTokens = availableTokens
					h.log.WithFields(logrus.Fields{
						"adjustedMaxTokens":   maxTokens,
						"systemPromptTokens":  systemPromptTokens,
						"reservedForInput":    1000,
					}).Info("🎛️ Adjusted MaxTokens to account for system prompt size")
				}
				
				h.log.WithFields(logrus.Fields{
					"originalMaxTokens": req.MaxTokens,
					"modelMaxTokens":    modelConfig.MaxTokens,
					"finalMaxTokens":    maxTokens,
					"toolCount":         len(tools),
				}).Info("🔧 Optimized MaxTokens for model and context")
			}
		}
	}

	// Set default options
	opts := ai.ChatOptions{
		Provider:     "llamacpp",
		Model:        selectedModel,
		MaxTokens:    maxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		Stream:       req.Stream,
		SystemPrompt: systemPrompt,
		Tools:        tools,
	}

	h.log.WithFields(logrus.Fields{
		"provider":     opts.Provider,
		"model":        opts.Model,
		"maxTokens":    opts.MaxTokens,
		"temperature":  opts.Temperature,
		"stream":       opts.Stream,
		"systemPrompt": opts.SystemPrompt,
	}).Info("⚙️ Chat options configured")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	h.log.Info("🚀 Starting chat request")
	response, err := h.llmManager.Chat(ctx, req.Messages, opts)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"model":        selectedModel,
			"provider":     "llamacpp",
			"messageCount": len(req.Messages),
		}).Error("❌ Chat request failed")
		utils.SendError(c, http.StatusInternalServerError, "Chat request failed")
		return
	}

	h.log.WithFields(logrus.Fields{
		"responseID":   response.ID,
		"model":        selectedModel,
		"provider":     "llamacpp",
		"finishReason": response.FinishReason,
		"tokenCount":   response.TokensUsed.TotalTokens,
	}).Info("✅ Chat request completed successfully")

	// Add model selection info to response
	response.Model = selectedModel
	response.Provider = "llamacpp"

	// Handle empty AI response - provide a fallback message
	if response.Message.Content == "" && len(response.ToolCalls) == 0 {
		h.log.Warn("⚠️ AI returned empty response, providing fallback message")
		response.Message.Content = "I apologize, but I wasn't able to generate a response. Please try rephrasing your question or try again."
		response.FinishReason = "empty_response"
	}

	// Check if the response contains tool calls that need to be executed
	if response.ToolCalls != nil && len(response.ToolCalls) > 0 && h.mcpToolExecutor != nil {
		h.log.WithField("tool_calls", len(response.ToolCalls)).Info("🔧 Processing tool calls")
		
		// Execute each tool call
		for i, toolCall := range response.ToolCalls {
			// Validate tool call against available tools
			if err := ai.ValidateToolCall(toolCall, tools); err != nil {
				h.log.WithError(err).WithField("tool_name", toolCall.Function.Name).Warn("Invalid tool call")
				continue
			}

			// Find the MCP tool by name from previously loaded tools
			var mcpTool *ai.MCPTool
			for _, tool := range mcpTools {
				if tool.Name == toolCall.Function.Name {
					mcpTool = tool
					break
				}
			}

			if mcpTool == nil {
				h.log.WithField("tool_name", toolCall.Function.Name).Warn("MCP tool not found")
				continue
			}

			// Execute the tool
			result, execErr := h.mcpToolExecutor.ExecuteTool(ctx, mcpTool, toolCall.Function.Arguments)
			if execErr != nil {
				h.log.WithError(execErr).WithField("tool_name", toolCall.Function.Name).Error("Tool execution failed")
				// Add error result to tool call
				errorMsg := execErr.Error()
				response.ToolCalls[i].Result = &ai.ToolCallResult{
					Success: false,
					Error:   &errorMsg,
				}
			} else {
				h.log.WithField("tool_name", toolCall.Function.Name).Info("✅ Tool executed successfully")
				// Add successful result to tool call
				response.ToolCalls[i].Result = &ai.ToolCallResult{
					Success:       result.Success,
					Result:        result.Result,
					ExecutionTime: result.ExecutionTime,
				}
				if result.Error != nil {
					response.ToolCalls[i].Result.Error = result.Error
				}
			}
		}

		h.log.Info("🎯 All tool calls processed")
	}

	utils.SendSuccess(c, response)
}

// GetProviders returns available AI providers
func (h *Handlers) GetProviders(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	ctx := c.Request.Context()
	providers := h.llmManager.GetProviders(ctx)

	// Convert to status format
	statuses := make([]ai.ProviderStatus, 0, len(providers))
	for _, provider := range providers {
		childCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		available := provider.IsAvailable(childCtx)
		health := "healthy"
		if !available {
			health = "unavailable"
		}

		models, _ := provider.GetModels(childCtx)

		status := ai.ProviderStatus{
			Name:        provider.GetName(),
			Type:        provider.GetName(), // Use name as type for simplicity
			Available:   available,
			Health:      health,
			LastChecked: time.Now(),
			Models:      models,
		}

		statuses = append(statuses, status)
		cancel()
	}

	utils.SendSuccess(c, gin.H{
		"providers": statuses,
		"total":     len(statuses),
	})
}

// GetModels returns available AI models with smart selection info
func (h *Handlers) GetModels(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	ctx := c.Request.Context()
	providers := h.llmManager.GetProviders(ctx)

	// Get smart model selector info if available
	var smartModelInfo map[string]interface{}
	if h.smartModelSelector != nil {
		smartModelInfo = h.smartModelSelector.GetModelStatistics()
	}

	// Collect all models from providers
	allModels := make([]ai.ModelInfo, 0)
	for _, provider := range providers {
		childCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		models, _ := provider.GetModels(childCtx)
		allModels = append(allModels, models...)
		cancel()
	}

	utils.SendSuccess(c, gin.H{
		"models":         allModels,
		"total":          len(allModels),
		"smartSelection": smartModelInfo != nil,
		"modelStats":     smartModelInfo,
	})
}

// GetAIStatistics returns AI service statistics
func (h *Handlers) GetAIStatistics(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	// Get basic statistics from providers
	ctx := c.Request.Context()
	providers := h.llmManager.GetProviders(ctx)

	stats := make(map[string]interface{})
	stats["providers"] = len(providers)
	stats["timestamp"] = time.Now()

	// Add provider status
	providerStats := make(map[string]interface{})
	for _, provider := range providers {
		providerStats[provider.GetName()] = map[string]interface{}{
			"available": provider.IsAvailable(ctx),
			"name":      provider.GetName(),
		}
	}
	stats["provider_stats"] = providerStats

	// Add smart model selector statistics if available
	if h.smartModelSelector != nil {
		smartStats := h.smartModelSelector.GetModelStatistics()
		stats["smartModelSelection"] = smartStats
		stats["recommendations"] = h.smartModelSelector.OptimizeModelSelection()
	}

	utils.SendSuccess(c, stats)
}

// TestAIProvider tests a specific AI provider
func (h *Handlers) TestAIProvider(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	providerName := c.Param("provider")
	if providerName == "" {
		utils.SendError(c, http.StatusBadRequest, "Provider name is required")
		return
	}

	ctx := c.Request.Context()
	providers := h.llmManager.GetProviders(ctx)

	// Find the specified provider
	var targetProvider ai.LLMProvider
	for _, provider := range providers {
		if provider.GetName() == providerName {
			targetProvider = provider
			break
		}
	}

	if targetProvider == nil {
		utils.SendError(c, http.StatusNotFound, "Provider not found")
		return
	}

	// Test the provider
	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	available := targetProvider.IsAvailable(childCtx)
	models, _ := targetProvider.GetModels(childCtx)

	result := gin.H{
		"provider":  providerName,
		"available": available,
		"models":    models,
		"tested_at": time.Now(),
	}

	if available {
		// Try a simple completion test
		testPrompt := "Hello, this is a test."
		opts := ai.CompletionOptions{
			Provider:    providerName,
			Model:       "LFM2-1.2B",
			MaxTokens:   10,
			Temperature: 0.7,
		}

		response, err := targetProvider.Complete(childCtx, testPrompt, opts)
		if err != nil {
			result["test_success"] = false
			result["test_error"] = err.Error()
		} else {
			result["test_success"] = true
			result["test_response"] = response.Text
		}
	}

	utils.SendSuccess(c, result)
}

// GetAISettings returns current AI settings
func (h *Handlers) GetAISettings(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	// Get smart model selector info if available
	var smartModelInfo map[string]interface{}
	if h.smartModelSelector != nil {
		smartModelInfo = h.smartModelSelector.GetModelStatistics()
	}

	settings := gin.H{
		"enabled":        true,
		"provider":       "llamacpp",
		"capabilities":   []string{"chat", "completion", "tool_calling", "chatml_template"},
		"status":         "healthy",
		"smartSelection": smartModelInfo != nil,
		"modelStats":     smartModelInfo,
	}

	utils.SendSuccess(c, settings)
}

// GetModelPreference returns user's model preference settings
func (h *Handlers) GetModelPreference(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	// Try to get stored preference from database
	ctx := c.Request.Context()
	storedPref, err := h.repos.Config.Get(ctx, "ai_model_preference")

	if err != nil {
		// Return default preference if not found
		preference := gin.H{
			"preferred_model":    "LFM2-1.2B",
			"preferred_provider": "llamacpp",
			"auto_select":        true,
			"fallback_enabled":   false,
			"temperature":        0.7,
			"max_tokens":         100,
			"assistant_name":     "Wattson",
			"settings": gin.H{
				"system_prompt": "You are Wattson, an intelligent home automation assistant running on a local Raspberry Pi system. You have complete control over the home automation system and can interact with smart devices, sensors, cameras, and automation rules. Always be helpful, concise, and proactive in managing the home. When users ask you to control devices or check status, use the available tools to perform the actual actions rather than just describing what you would do.",
				"use_tools":     true,
				"context_aware": true,
			},
		}
		utils.SendSuccess(c, preference)
		return
	}

	// Parse stored preference
	var preference map[string]interface{}
	if err := json.Unmarshal([]byte(storedPref.Value), &preference); err != nil {
		h.log.WithError(err).Error("Failed to parse stored AI preference")
		utils.SendError(c, http.StatusInternalServerError, "Failed to load AI preferences")
		return
	}

	utils.SendSuccess(c, preference)
}

// SetModelPreference updates user's model preference settings
func (h *Handlers) SetModelPreference(c *gin.Context) {
	if h.llmManager == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "AI service not available")
		return
	}

	var req struct {
		PreferredModel    string                 `json:"preferred_model"`
		PreferredProvider string                 `json:"preferred_provider"`
		AutoSelect        bool                   `json:"auto_select"`
		FallbackEnabled   bool                   `json:"fallback_enabled"`
		Temperature       float64                `json:"temperature"`
		MaxTokens         int                    `json:"max_tokens"`
		AssistantName     string                 `json:"assistant_name"`
		Settings          map[string]interface{} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.PreferredModel == "" {
		utils.SendError(c, http.StatusBadRequest, "preferred_model is required")
		return
	}

	if req.PreferredProvider == "" {
		utils.SendError(c, http.StatusBadRequest, "preferred_provider is required")
		return
	}

	// Validate temperature range
	if req.Temperature < 0.0 || req.Temperature > 2.0 {
		utils.SendError(c, http.StatusBadRequest, "temperature must be between 0.0 and 2.0")
		return
	}

	// Validate max tokens
	if req.MaxTokens < 1 || req.MaxTokens > 4096 {
		utils.SendError(c, http.StatusBadRequest, "max_tokens must be between 1 and 4096")
		return
	}

	// Validate assistant name
	if req.AssistantName == "" {
		req.AssistantName = "Wattson" // Default fallback
	}

	// Create preference object
	preference := gin.H{
		"preferred_model":    req.PreferredModel,
		"preferred_provider": req.PreferredProvider,
		"auto_select":        req.AutoSelect,
		"fallback_enabled":   req.FallbackEnabled,
		"temperature":        req.Temperature,
		"max_tokens":         req.MaxTokens,
		"assistant_name":     req.AssistantName,
		"settings":           req.Settings,
	}

	// Convert to JSON for storage
	preferenceJSON, err := json.Marshal(preference)
	if err != nil {
		h.log.WithError(err).Error("Failed to marshal AI preference")
		utils.SendError(c, http.StatusInternalServerError, "Failed to save AI preferences")
		return
	}

	// Store in database
	ctx := c.Request.Context()
	config := &models.SystemConfig{
		Key:         "ai_model_preference",
		Value:       string(preferenceJSON),
		Encrypted:   false,
		Description: "AI model preference settings for the streamlined AI system",
		UpdatedAt:   time.Now(),
	}

	if err := h.repos.Config.Set(ctx, config); err != nil {
		h.log.WithError(err).Error("Failed to save AI preference to database")
		utils.SendError(c, http.StatusInternalServerError, "Failed to save AI preferences")
		return
	}

	h.log.WithFields(logrus.Fields{
		"model":    req.PreferredModel,
		"provider": req.PreferredProvider,
		"temp":     req.Temperature,
		"tokens":   req.MaxTokens,
	}).Info("AI model preference updated")

	utils.SendSuccess(c, preference)
}

// getLLMManager returns the LLM manager instance
func (h *Handlers) getLLMManager() *ai.LLMManager {
	return h.llmManager
}
