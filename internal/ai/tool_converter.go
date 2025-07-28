package ai

import (
	"fmt"
)

// ConvertMCPToolsToLLMTools converts MCP tools to LLM function calling format
func ConvertMCPToolsToLLMTools(mcpTools []*MCPTool) []LLMTool {
	var llmTools []LLMTool

	for _, mcpTool := range mcpTools {
		if !mcpTool.Enabled {
			continue
		}

		llmTool := LLMTool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Parameters:  mcpTool.Schema,
		}

		llmTools = append(llmTools, llmTool)
	}

	return llmTools
}

// ConvertMCPToolToLLMTool converts a single MCP tool to LLM format
func ConvertMCPToolToLLMTool(mcpTool *MCPTool) LLMTool {
	return LLMTool{
		Name:        mcpTool.Name,
		Description: mcpTool.Description,
		Parameters:  mcpTool.Schema,
	}
}

// CreateToolsSystemPrompt creates a system prompt that describes available tools
func CreateToolsSystemPrompt(tools []LLMTool) string {
	if len(tools) == 0 {
		return ""
	}

	prompt := "\n\nYou have access to the following tools for controlling the PMA home automation system:\n\n"

	for _, tool := range tools {
		prompt += fmt.Sprintf("**%s**: %s\n", tool.Name, tool.Description)

		// Add parameter information if available
		if tool.Parameters != nil {
			if properties, ok := tool.Parameters["properties"].(map[string]interface{}); ok {
				prompt += "  Parameters:\n"
				for paramName, paramInfo := range properties {
					if paramMap, ok := paramInfo.(map[string]interface{}); ok {
						if desc, ok := paramMap["description"].(string); ok {
							prompt += fmt.Sprintf("  - %s: %s\n", paramName, desc)
						}
					}
				}
			}
		}
		prompt += "\n"
	}

	prompt += "Use these tools whenever you need to interact with the home automation system. Always call the appropriate tool rather than just describing what you would do.\n"

	return prompt
}

// ValidateToolCall validates that a tool call matches an available tool
func ValidateToolCall(toolCall ToolCall, availableTools []LLMTool) error {
	// Find the tool
	var foundTool *LLMTool
	for i := range availableTools {
		if availableTools[i].Name == toolCall.Function.Name {
			foundTool = &availableTools[i]
			break
		}
	}

	if foundTool == nil {
		return fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}

	// Basic validation - could be extended to validate parameters against schema
	if toolCall.Function.Arguments == nil {
		toolCall.Function.Arguments = make(map[string]interface{})
	}

	return nil
}
