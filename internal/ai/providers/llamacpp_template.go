package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Chat implements the chat interface with LFM2 ChatML template and tool calling
func (p *LlamaCppProvider) Chat(ctx context.Context, messages []ai.ChatMessage, opts ai.ChatOptions) (*ai.ChatResponse, error) {
	p.logger.WithFields(logrus.Fields{
		"messageCount": len(messages),
		"model":        opts.Model,
		"maxTokens":    opts.MaxTokens,
		"temperature":  opts.Temperature,
		"baseURL":      p.baseURL,
	}).Info("🎯 LlamaCpp Chat request started")

	p.mu.Lock()
	p.requestCount++
	p.mu.Unlock()

	startTime := time.Now()
	defer func() {
		p.mu.Lock()
		p.totalResponseTime += time.Since(startTime)
		p.mu.Unlock()
	}()

	// Format messages using LFM2 ChatML template
	p.logger.Info("📝 Formatting messages with LFM2 ChatML template")
	prompt, err := p.templateEngine.FormatMessages(messages, opts.Tools)
	if err != nil {
		p.logger.WithError(err).Error("❌ Failed to format messages")
		p.mu.Lock()
		p.errorCount++
		p.mu.Unlock()
		return nil, fmt.Errorf("failed to format messages: %w", err)
	}

	p.logger.WithField("promptLength", len(prompt)).Info("✅ Messages formatted successfully")

	// Prepare completion request with LFM2-optimized parameters
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	completionReq := map[string]interface{}{
		"prompt":             prompt,
		"n_predict":          maxTokens,
		"temperature":        0.3,  // Optimal for LFM2
		"min_p":              0.15, // LFM2-specific parameter
		"repetition_penalty": 1.05, // LFM2-specific parameter
		"stop":               []string{"<|im_end|>", "<|tool_response_end|>"},
		"stream":             false,
	}

	p.logger.WithFields(logrus.Fields{
		"maxTokens":         maxTokens,
		"temperature":       0.3,
		"minP":              0.15,
		"repetitionPenalty": 1.05,
	}).Info("⚙️ Completion request parameters configured")

	// Send request to llama.cpp server
	p.logger.Info("📤 Sending completion request to llama.cpp server")
	completionResult, err := p.sendCompletionRequest(ctx, completionReq)
	if err != nil {
		p.logger.WithError(err).Error("❌ Completion request failed")
		p.mu.Lock()
		p.errorCount++
		p.mu.Unlock()
		return nil, err
	}

	p.logger.WithField("responseLength", len(completionResult.Content)).Info("✅ Completion request successful")

	// Parse response for tool calls and content
	p.logger.Info("🔍 Parsing response for tool calls and content")
	response, err := p.templateEngine.ParseResponse(completionResult.Content, opts.Tools, completionResult)
	if err != nil {
		p.logger.WithError(err).Error("❌ Failed to parse response")
		p.mu.Lock()
		p.errorCount++
		p.mu.Unlock()
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"responseID":   response.ID,
		"finishReason": response.FinishReason,
		"tokenCount":   response.TokensUsed.TotalTokens,
		"duration":     time.Since(startTime),
	}).Info("✅ Chat request completed successfully")

	return response, nil
}

// CompletionResult holds the result from llama.cpp completion
type CompletionResult struct {
	Content          string
	TokensPredicted  int
	TokensEvaluated  int
	PromptTokens     int
	CompletionTokens int
}

// sendCompletionRequest sends a completion request to llama.cpp server
func (p *LlamaCppProvider) sendCompletionRequest(ctx context.Context, request map[string]interface{}) (*CompletionResult, error) {
	p.logger.WithField("url", p.baseURL+"/completion").Info("🌐 Preparing HTTP request to llama.cpp server")

	jsonData, err := json.Marshal(request)
	if err != nil {
		p.logger.WithError(err).Error("❌ Failed to marshal request JSON")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	p.logger.WithField("requestSize", len(jsonData)).Info("📦 Request JSON marshaled successfully")

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/completion", bytes.NewBuffer(jsonData))
	if err != nil {
		p.logger.WithError(err).Error("❌ Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	p.logger.Info("📤 Sending HTTP request")

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.WithError(err).Error("❌ HTTP request failed")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	p.logger.WithField("statusCode", resp.StatusCode).Info("📥 HTTP response received")

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"body":       string(body),
		}).Error("❌ Server returned error status")
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var completionResp struct {
		Content          string `json:"content"`
		Stop             bool   `json:"stop"`
		TokensPredicted  int    `json:"tokens_predicted"`
		TokensEvaluated  int    `json:"tokens_evaluated"`
		Timings          struct {
			PromptN    int `json:"prompt_n"`
			PredictedN int `json:"predicted_n"`
		} `json:"timings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		p.logger.WithError(err).Error("❌ Failed to decode response JSON")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &CompletionResult{
		Content:          completionResp.Content,
		TokensPredicted:  completionResp.TokensPredicted,
		TokensEvaluated:  completionResp.TokensEvaluated,
		PromptTokens:     completionResp.Timings.PromptN,
		CompletionTokens: completionResp.Timings.PredictedN,
	}

	p.logger.WithFields(logrus.Fields{
		"contentLength":    len(result.Content),
		"promptTokens":     result.PromptTokens,
		"completionTokens": result.CompletionTokens,
		"totalTokens":      result.PromptTokens + result.CompletionTokens,
	}).Info("✅ Response decoded successfully with token counts")
	
	return result, nil
}

// FormatMessages formats messages using LFM2 ChatML template
func (te *LFM2TemplateEngine) FormatMessages(messages []ai.ChatMessage, tools []ai.Tool) (string, error) {
	var prompt strings.Builder

	// Get dynamic system prompt and assistant name from database
	systemPrompt, _ := te.getDynamicSystemPrompt()

	// Start with the LFM2 ChatML format
	prompt.WriteString("<|startoftext|><|im_start|>system\n")
	prompt.WriteString(systemPrompt)

	// Add tool definitions if provided
	if len(tools) > 0 {
		prompt.WriteString("\n\nList of tools: <|tool_list_start|>")
		toolsJSON, err := json.Marshal(tools)
		if err != nil {
			return "", fmt.Errorf("failed to marshal tools: %w", err)
		}
		prompt.Write(toolsJSON)
		prompt.WriteString("<|tool_list_end|>")
	}

	prompt.WriteString("<|im_end|>\n")

	// Add conversation messages
	for _, msg := range messages {
		prompt.WriteString(fmt.Sprintf("<|im_start|>%s\n%s<|im_end|>\n", msg.Role, msg.Content))
	}

	// Start assistant response
	prompt.WriteString("<|im_start|>assistant\n")

	return prompt.String(), nil
}

// ParseResponse parses the LFM2 response for tool calls and content
func (te *LFM2TemplateEngine) ParseResponse(response string, tools []ai.Tool, tokenCounts *CompletionResult) (*ai.ChatResponse, error) {
	// Use actual token counts from llama.cpp if available, otherwise fallback to estimates
	var tokenUsage ai.TokenUsage
	if tokenCounts != nil {
		tokenUsage = ai.TokenUsage{
			PromptTokens:     tokenCounts.PromptTokens,
			CompletionTokens: tokenCounts.CompletionTokens,
			TotalTokens:      tokenCounts.PromptTokens + tokenCounts.CompletionTokens,
		}
	} else {
		// Fallback to estimates
		tokenUsage = ai.TokenUsage{
			PromptTokens:     len(response) / 8, // Rough estimate: ~8 chars per prompt token
			CompletionTokens: len(response) / 4, // Rough estimate: ~4 chars per completion token
			TotalTokens:      len(response) / 4, // Rough estimate for total tokens
		}
	}

	chatResponse := &ai.ChatResponse{
		ID: uuid.New().String(),
		Message: ai.ChatMessage{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now(),
		},
		FinishReason: "stop",
		Model:        "LFM2-llama.cpp",
		Provider:     "llamacpp",
		TokensUsed:   tokenUsage,
		CreatedAt:    time.Now(),
	}

	// Check for tool calls in the response
	if strings.Contains(response, "<|tool_call_start|>") && strings.Contains(response, "<|tool_call_end|>") {
		toolCalls, content, err := te.extractToolCalls(response)
		if err != nil {
			return nil, fmt.Errorf("failed to extract tool calls: %w", err)
		}

		chatResponse.ToolCalls = toolCalls
		chatResponse.Message.Content = content
		chatResponse.FinishReason = "tool_calls"
	}

	return chatResponse, nil
}

// extractToolCalls extracts tool calls from LFM2 response
func (te *LFM2TemplateEngine) extractToolCalls(response string) ([]ai.ToolCall, string, error) {
	// Find tool call section
	startIdx := strings.Index(response, "<|tool_call_start|>")
	endIdx := strings.Index(response, "<|tool_call_end|>")

	if startIdx == -1 || endIdx == -1 {
		return nil, response, nil
	}

	// Extract tool call content
	toolCallContent := response[startIdx+len("<|tool_call_start|>") : endIdx]

	// Extract remaining content (before and after tool calls)
	beforeContent := response[:startIdx]
	afterContent := response[endIdx+len("<|tool_call_end|>"):]
	content := strings.TrimSpace(beforeContent + afterContent)

	// Parse tool calls (they should be in Python list format)
	toolCalls, err := te.parseToolCallList(toolCallContent)
	if err != nil {
		return nil, content, fmt.Errorf("failed to parse tool call list: %w", err)
	}

	return toolCalls, content, nil
}

// parseToolCallList parses the Python-style tool call list
func (te *LFM2TemplateEngine) parseToolCallList(content string) ([]ai.ToolCall, error) {
	// This is a simplified parser for Python-style function calls
	// In a production system, you might want to use a proper Python AST parser

	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "[") || !strings.HasSuffix(content, "]") {
		return nil, fmt.Errorf("tool calls must be in list format")
	}

	// Remove brackets
	content = content[1 : len(content)-1]
	content = strings.TrimSpace(content)

	var toolCalls []ai.ToolCall

	// Simple parsing for function calls like: function_name(arg1="value1", arg2="value2")
	// This is a basic implementation - you may need more sophisticated parsing
	if content != "" {
		parts := strings.Split(content, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "(") && strings.Contains(part, ")") {
				toolCall, err := te.parseSingleToolCall(part)
				if err != nil {
					continue // Skip invalid tool calls
				}
				toolCalls = append(toolCalls, toolCall)
			}
		}
	}

	return toolCalls, nil
}

// parseSingleToolCall parses a single tool call
func (te *LFM2TemplateEngine) parseSingleToolCall(call string) (ai.ToolCall, error) {
	// Basic parsing for: function_name(arg1="value1", arg2="value2")
	parenIdx := strings.Index(call, "(")
	if parenIdx == -1 {
		return ai.ToolCall{}, fmt.Errorf("invalid function call format")
	}

	functionName := strings.TrimSpace(call[:parenIdx])
	argsStr := call[parenIdx+1:]
	if strings.HasSuffix(argsStr, ")") {
		argsStr = argsStr[:len(argsStr)-1]
	}

	// Parse arguments (simplified)
	args := make(map[string]interface{})
	if argsStr != "" {
		// Very basic argument parsing - in production, use a proper parser
		argPairs := strings.Split(argsStr, ",")
		for _, pair := range argPairs {
			if strings.Contains(pair, "=") {
				parts := strings.SplitN(pair, "=", 2)
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Remove quotes
				if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
					value = value[1 : len(value)-1]
				}

				args[key] = value
			}
		}
	}

	return ai.ToolCall{
		ID:   uuid.New().String(),
		Name: functionName,
		Function: ai.ToolFunction{
			Name:      functionName,
			Arguments: args,
		},
	}, nil
}

// ProcessToolResponse processes a tool response and formats it for LFM2
func (te *LFM2TemplateEngine) ProcessToolResponse(toolCallID string, result interface{}) string {
	var response strings.Builder

	response.WriteString("<|tool_response_start|>")

	if result != nil {
		if resultStr, ok := result.(string); ok {
			response.WriteString(resultStr)
		} else {
			// Try to marshal to JSON
			if jsonBytes, err := json.Marshal(result); err == nil {
				response.WriteString(string(jsonBytes))
			} else {
				response.WriteString(fmt.Sprintf("%v", result))
			}
		}
	}

	response.WriteString("<|tool_response_end|>")

	return response.String()
}

// getDynamicSystemPrompt fetches the system prompt and assistant name from user preferences
func (te *LFM2TemplateEngine) getDynamicSystemPrompt() (string, string) {
	// Default values
	defaultSystemPrompt := "You are Wattson, an intelligent home automation assistant running on a local Raspberry Pi system. You have complete control over the home automation system and can interact with smart devices, sensors, cameras, and automation rules. Always be helpful, concise, and proactive in managing the home. When users ask you to control devices or check status, use the available tools to perform the actual actions rather than just describing what you would do."
	defaultAssistantName := "Wattson"

	// If no config repo available, return defaults
	if te.configRepo == nil {
		return defaultSystemPrompt, defaultAssistantName
	}

	// Try to fetch user preferences from database
	ctx := context.Background()
	storedPref, err := te.configRepo.Get(ctx, "ai_model_preference")
	if err != nil {
		if te.logger != nil {
			te.logger.WithError(err).Debug("Could not fetch AI preferences, using defaults")
		}
		return defaultSystemPrompt, defaultAssistantName
	}

	// Parse the stored preferences
	var preferences map[string]interface{}
	if err := json.Unmarshal([]byte(storedPref.Value), &preferences); err != nil {
		if te.logger != nil {
			te.logger.WithError(err).Debug("Could not parse AI preferences, using defaults")
		}
		return defaultSystemPrompt, defaultAssistantName
	}

	// Extract assistant name
	assistantName := defaultAssistantName
	if name, ok := preferences["assistant_name"].(string); ok && name != "" {
		assistantName = name
	}

	// Extract system prompt from settings
	systemPrompt := defaultSystemPrompt
	if settings, ok := preferences["settings"].(map[string]interface{}); ok {
		if prompt, ok := settings["system_prompt"].(string); ok && prompt != "" {
			systemPrompt = prompt
		}
	}

	// If the system prompt doesn't contain the assistant name, inject it
	if !strings.Contains(systemPrompt, assistantName) && assistantName != "Wattson" {
		// Replace "Wattson" or "assistant" with the custom name
		systemPrompt = strings.ReplaceAll(systemPrompt, "Wattson", assistantName)
		if !strings.Contains(systemPrompt, assistantName) {
			systemPrompt = strings.ReplaceAll(systemPrompt, "assistant", assistantName)
		}
	}

	return systemPrompt, assistantName
}
