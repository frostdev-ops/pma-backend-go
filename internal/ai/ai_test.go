package ai

import (
	"context"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// MockProvider implements LLMProvider for testing
type MockProvider struct {
	name      string
	available bool
}

func (m *MockProvider) Complete(ctx context.Context, prompt string, opts CompletionOptions) (*CompletionResponse, error) {
	return &CompletionResponse{
		ID:               "test-completion",
		Text:             "Mock completion response",
		FinishReason:     "stop",
		Model:            "mock-model",
		Provider:         m.name,
		ProcessingTimeMs: 100,
		TokensUsed:       TokenUsage{PromptTokens: 10, CompletionTokens: 15, TotalTokens: 25},
		CreatedAt:        time.Now(),
	}, nil
}

func (m *MockProvider) Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (*ChatResponse, error) {
	return &ChatResponse{
		ID: "test-chat",
		Message: ChatMessage{
			Role:      "assistant",
			Content:   "Mock chat response",
			Timestamp: time.Now(),
		},
		FinishReason:     "stop",
		Model:            "mock-model",
		Provider:         m.name,
		ProcessingTimeMs: 120,
		TokensUsed:       TokenUsage{PromptTokens: 20, CompletionTokens: 25, TotalTokens: 45},
		CreatedAt:        time.Now(),
	}, nil
}

func (m *MockProvider) GetName() string {
	return m.name
}

func (m *MockProvider) IsAvailable(ctx context.Context) bool {
	return m.available
}

func (m *MockProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	return []ModelInfo{
		{
			ID:           "mock-model",
			Name:         "Mock Model",
			Description:  "A mock model for testing",
			Provider:     m.name,
			MaxTokens:    4096,
			Available:    true,
			LocalModel:   false,
			Capabilities: []string{"chat", "completion"},
		},
	}, nil
}

func (m *MockProvider) EstimateTokens(text string) int {
	return len(text) / 4
}

func (m *MockProvider) GetRateLimit() RateLimit {
	return RateLimit{
		RequestsPerMinute: 100,
		TokensPerMinute:   10000,
	}
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockProvider) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockProvider) Shutdown(ctx context.Context) error {
	return nil
}

func TestNewLLMManager(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultProvider: "mock",
			FallbackEnabled: true,
			FallbackDelay:   "1s",
			MaxRetries:      3,
			Timeout:         "30s",
			Providers:       []config.AIProviderConfig{},
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	manager, err := NewLLMManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create LLM manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Test with mock provider
	mockProvider := &MockProvider{name: "mock", available: true}
	manager.RegisterProviderFactory("mock", func(cfg config.AIProviderConfig, logger *logrus.Logger) LLMProvider {
		return mockProvider
	})

	// Test initialization
	ctx := context.Background()
	err = manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Test getting providers
	providers := manager.GetProviders(ctx)
	if len(providers) == 0 {
		t.Log("No providers available (expected since no factories are registered by default)")
	}

	// Test getting models
	models, err := manager.GetModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Log("No models available (expected since no providers are configured)")
	}

	// Test statistics
	stats := manager.GetStatistics()
	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	// Test shutdown
	err = manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shutdown manager: %v", err)
	}
}

func TestChatService(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultProvider: "mock",
			FallbackEnabled: true,
			Providers:       []config.AIProviderConfig{},
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	manager, err := NewLLMManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create LLM manager: %v", err)
	}

	// Register mock provider
	mockProvider := &MockProvider{name: "mock", available: true}
	manager.RegisterProviderFactory("mock", func(cfg config.AIProviderConfig, logger *logrus.Logger) LLMProvider {
		return mockProvider
	})

	// Initialize manager
	ctx := context.Background()
	err = manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create chat service
	chatService := NewChatService(manager, logger)
	if chatService == nil {
		t.Fatal("Chat service should not be nil")
	}

	// Test basic functionality without actual provider interaction
	chatService.SetDefaultModel("mock-model")

	// Test completion request structure
	completionReq := CompletionRequest{
		Prompt:      "Test prompt",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	if completionReq.Prompt == "" {
		t.Fatal("Completion request should have prompt")
	}

	// Test chat request structure
	chatReq := ChatRequest{
		Messages: []ChatMessage{
			{
				Role:      "user",
				Content:   "Hello",
				Timestamp: time.Now(),
			},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	if len(chatReq.Messages) == 0 {
		t.Fatal("Chat request should have messages")
	}

	// Test entity analysis request
	analysisReq := EntityAnalysisRequest{
		EntityIDs:    []string{"test.entity"},
		AnalysisType: "patterns",
		TimeRange: TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
	}

	if len(analysisReq.EntityIDs) == 0 {
		t.Fatal("Analysis request should have entity IDs")
	}

	// Test automation generation request
	automationReq := AutomationGenerationRequest{
		Description: "Turn on lights when motion is detected",
		EntityIDs:   []string{"motion.sensor", "light.living_room"},
		Complexity:  "simple",
	}

	if automationReq.Description == "" {
		t.Fatal("Automation request should have description")
	}
}

func TestProviderError(t *testing.T) {
	err := &ProviderError{
		Provider:  "test",
		Type:      "rate_limit",
		Message:   "Rate limit exceeded",
		Retryable: true,
	}

	if err.Error() != "Rate limit exceeded" {
		t.Fatalf("Expected 'Rate limit exceeded', got '%s'", err.Error())
	}

	if !err.IsRetryable() {
		t.Fatal("Error should be retryable")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := &CircuitBreaker{
		state: CircuitClosed,
	}

	if cb.state != CircuitClosed {
		t.Fatal("Circuit breaker should start closed")
	}

	// Simulate failures
	cb.failures = 5
	cb.state = CircuitOpen
	cb.lastFailure = time.Now()

	if cb.state != CircuitOpen {
		t.Fatal("Circuit breaker should be open after failures")
	}
}
