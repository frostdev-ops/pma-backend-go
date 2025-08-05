package ai

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ModelComplexity represents the complexity level of a request
type ModelComplexity int

const (
	Simple ModelComplexity = iota
	Moderate
	Complex
)

// SmartModelSelector manages intelligent model selection based on request complexity
type SmartModelSelector struct {
	logger *logrus.Logger

	// Available model configurations
	models map[string]ModelConfig

	// Complexity detection patterns
	simplePatterns     []string
	complexPatterns    []string
	toolPatterns       []string
	automationPatterns []string
}

// ModelConfig defines a model's capabilities and characteristics
type ModelConfig struct {
	Name         string
	Quantization string
	MaxTokens    int
	Temperature  float64
	ResponseTime time.Duration
	Accuracy     float64
	MemoryUsage  int64
	IsLoaded     bool
}

// NewSmartModelSelector creates a new smart model selector
func NewSmartModelSelector(logger *logrus.Logger) *SmartModelSelector {
	selector := &SmartModelSelector{
		logger: logger,
		models: make(map[string]ModelConfig),
	}

	// Initialize LFM2 model configurations
	selector.initializeLFM2Models()

	// Initialize complexity detection patterns
	selector.initializeComplexityPatterns()

	return selector
}

// GetModelConfig returns the configuration for a specific model
func (s *SmartModelSelector) GetModelConfig(modelName string) (ModelConfig, bool) {
	config, exists := s.models[modelName]
	return config, exists
}

// initializeLFM2Models sets up the different LFM2 quantizations
func (s *SmartModelSelector) initializeLFM2Models() {
	// Fastest model for simple requests - using full 32K context window
	s.models["LFM2-1.2B-Q2"] = ModelConfig{
		Name:         "LFM2-1.2B-Q2",
		Quantization: "Q2",
		MaxTokens:    32768, // Full 32K context window - no artificial limits
		Temperature:  0.7,
		ResponseTime: 500 * time.Millisecond,
		Accuracy:     0.75,
		MemoryUsage:  512 * 1024 * 1024, // 512MB
		IsLoaded:     true,
	}

	// Balanced model for moderate complexity - using full 32K context window
	s.models["LFM2-1.2B-Q4"] = ModelConfig{
		Name:         "LFM2-1.2B-Q4",
		Quantization: "Q4",
		MaxTokens:    32768, // Full 32K context window - no artificial limits
		Temperature:  0.7,
		ResponseTime: 1 * time.Second,
		Accuracy:     0.85,
		MemoryUsage:  1 * 1024 * 1024 * 1024, // 1GB
		IsLoaded:     true,
	}

	// High precision model for complex requests - using full 32K context window
	s.models["LFM2-1.2B-Q8"] = ModelConfig{
		Name:         "LFM2-1.2B-Q8",
		Quantization: "Q8",
		MaxTokens:    32768, // Full 32K context window - no artificial limits
		Temperature:  0.7,
		ResponseTime: 2 * time.Second,
		Accuracy:     0.95,
		MemoryUsage:  2 * 1024 * 1024 * 1024, // 2GB
		IsLoaded:     true,
	}

	// Full precision model for tool use and automation
	s.models["LFM2-1.2B"] = ModelConfig{
		Name:         "LFM2-1.2B",
		Quantization: "FP16",
		MaxTokens:    32768, // Full 32K context window - no artificial limits
		Temperature:  0.7,
		ResponseTime: 3 * time.Second,
		Accuracy:     0.98,
		MemoryUsage:  3 * 1024 * 1024 * 1024, // 3GB
		IsLoaded:     true,
	}
}

// initializeComplexityPatterns sets up patterns for detecting request complexity
func (s *SmartModelSelector) initializeComplexityPatterns() {
	// Simple patterns - basic conversation, greetings, simple questions
	s.simplePatterns = []string{
		"hello", "hi", "hey", "good morning", "good afternoon", "good evening",
		"how are you", "what's up", "thanks", "thank you", "bye", "goodbye",
		"what time", "what day", "weather", "temperature", "simple", "basic",
		"yes", "no", "ok", "okay", "sure", "fine", "good", "bad",
	}

	// Complex patterns - detailed analysis, reasoning, explanations
	s.complexPatterns = []string{
		"explain", "analyze", "compare", "contrast", "why", "how does",
		"what is the difference", "describe", "elaborate", "detailed",
		"comprehensive", "thorough", "analysis", "reasoning", "logic",
		"complex", "complicated", "difficult", "challenging", "advanced",
		"technical", "sophisticated", "nuanced", "subtle", "intricate",
	}

	// Tool use patterns - commands that require tool execution AND tool inquiries
	s.toolPatterns = []string{
		"turn on", "turn off", "switch", "toggle", "activate", "deactivate",
		"set", "change", "adjust", "control", "command", "execute", "run",
		"start", "stop", "restart", "reboot", "shutdown", "power",
		"light", "switch", "device", "sensor", "camera", "thermostat",
		"automation", "rule", "trigger", "condition", "action",
		// Tool inquiry patterns - questions about available tools/capabilities
		"what tools", "which tools", "tools available", "what can you do",
		"what functions", "capabilities", "available functions", "help me",
		"what commands", "list tools", "show tools", "tool list",
	}

	// Automation patterns - automation creation and management
	s.automationPatterns = []string{
		"create automation", "make rule", "set up automation", "automate",
		"when", "if", "then", "trigger", "condition", "action", "rule",
		"schedule", "timer", "delay", "repeat", "loop", "sequence",
		"workflow", "process", "pipeline", "chain", "series", "sequence",
	}
}

// AnalyzeComplexity analyzes the complexity of a conversation
func (s *SmartModelSelector) AnalyzeComplexity(messages []ChatMessage) ModelComplexity {
	if len(messages) == 0 {
		s.logger.Warn("⚠️ No messages provided for complexity analysis")
		return Simple
	}

	// Get the latest user message
	latestMessage := messages[len(messages)-1]
	content := strings.ToLower(latestMessage.Content)

	s.logger.WithFields(logrus.Fields{
		"messageCount":  len(messages),
		"latestContent": latestMessage.Content,
		"contentLength": len(content),
	}).Info("🔍 Analyzing request complexity")

	// Check for complex patterns
	for _, pattern := range s.complexPatterns {
		if strings.Contains(content, pattern) {
			s.logger.WithField("pattern", pattern).Info("🎯 Complex pattern detected")
			return Complex
		}
	}

	// Check for tool usage patterns
	for _, pattern := range s.toolPatterns {
		if strings.Contains(content, pattern) {
			s.logger.WithField("pattern", pattern).Info("🔧 Tool usage pattern detected")
			return Complex
		}
	}

	// Check for automation patterns
	for _, pattern := range s.automationPatterns {
		if strings.Contains(content, pattern) {
			s.logger.WithField("pattern", pattern).Info("🤖 Automation pattern detected")
			return Complex
		}
	}

	// Check for simple patterns
	for _, pattern := range s.simplePatterns {
		if strings.Contains(content, pattern) {
			s.logger.WithField("pattern", pattern).Info("💬 Simple pattern detected")
			return Simple
		}
	}

	// Length-based heuristics
	if len(content) > 200 {
		s.logger.WithField("contentLength", len(content)).Info("📏 Long message - moderate complexity")
		return Moderate
	}

	if len(content) < 50 {
		s.logger.WithField("contentLength", len(content)).Info("📏 Short message - simple complexity")
		return Simple
	}

	s.logger.Info("📊 Default complexity - moderate")
	return Moderate
}

// SelectOptimalModel selects the optimal model based on complexity
func (s *SmartModelSelector) SelectOptimalModel(complexity ModelComplexity) string {
	var selectedModel string

	switch complexity {
	case Simple:
		selectedModel = "LFM2-1.2B-Q2"
		s.logger.WithField("model", selectedModel).Info("⚡ Simple request - using fastest model")
	case Moderate:
		selectedModel = "LFM2-1.2B-Q4"
		s.logger.WithField("model", selectedModel).Info("⚖️ Moderate request - using balanced model")
	case Complex:
		selectedModel = "LFM2-1.2B-Q8"
		s.logger.WithField("model", selectedModel).Info("🧠 Complex request - using high precision model")
	default:
		selectedModel = "LFM2-1.2B-Q4"
		s.logger.WithField("model", selectedModel).Warn("⚠️ Unknown complexity - using balanced model")
	}

	// Multi-instance model selection - let the manager handle fallbacks
	s.logger.WithFields(logrus.Fields{
		"selectedModel": selectedModel,
		"complexity":    complexity,
		"reason":        "multi_instance_selection",
	}).Info("🎯 Model selected for multi-instance deployment")

	return selectedModel
}

// UpdateModelStatus updates the loaded status of a model
func (s *SmartModelSelector) UpdateModelStatus(modelName string, isLoaded bool) {
	if config, exists := s.models[modelName]; exists {
		config.IsLoaded = isLoaded
		s.models[modelName] = config
		s.logger.WithFields(logrus.Fields{
			"model":    modelName,
			"isLoaded": isLoaded,
		}).Info("Updated model status")
	}
}

// GetAvailableModels returns all available models with their status
func (s *SmartModelSelector) GetAvailableModels() map[string]ModelConfig {
	return s.models
}

// GetModelStatistics returns statistics about model usage and performance
func (s *SmartModelSelector) GetModelStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	for name, config := range s.models {
		stats[name] = map[string]interface{}{
			"quantization": config.Quantization,
			"maxTokens":    config.MaxTokens,
			"temperature":  config.Temperature,
			"responseTime": config.ResponseTime.String(),
			"accuracy":     config.Accuracy,
			"memoryUsage":  config.MemoryUsage,
			"isLoaded":     config.IsLoaded,
		}
	}

	return stats
}

// OptimizeModelSelection provides recommendations for model loading
func (s *SmartModelSelector) OptimizeModelSelection() []string {
	var recommendations []string

	// Always recommend keeping the fastest model loaded for simple requests
	if config, exists := s.models["LFM2-1.2B-Q2"]; exists && !config.IsLoaded {
		recommendations = append(recommendations, "Load LFM2-1.2B-Q2 for fast simple responses")
	}

	// Recommend balanced model for moderate complexity
	if config, exists := s.models["LFM2-1.2B-Q4"]; exists && !config.IsLoaded {
		recommendations = append(recommendations, "Load LFM2-1.2B-Q4 for balanced performance")
	}

	// Recommend full model for complex tasks
	if config, exists := s.models["LFM2-1.2B"]; exists && !config.IsLoaded {
		recommendations = append(recommendations, "Load LFM2-1.2B for complex tasks and tool use")
	}

	return recommendations
}
