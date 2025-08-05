package providers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// MultiInstanceLlamaCppProvider integrates the multi-instance manager with the LlamaCppProvider interface
type MultiInstanceLlamaCppProvider struct {
	name              string
	config            config.AIProviderConfig
	logger            *logrus.Logger
	manager           *MultiInstanceManager
	client            *http.Client
	mu                sync.RWMutex
	isRunning         bool
	lastHealthCheck   time.Time
	errorCount        int64
	requestCount      int64
	totalResponseTime time.Duration
}

// NewMultiInstanceLlamaCppProvider creates a new multi-instance llamacpp provider
func NewMultiInstanceLlamaCppProvider(cfg config.AIProviderConfig, fullCfg *config.Config, logger *logrus.Logger) *MultiInstanceLlamaCppProvider {
	provider := &MultiInstanceLlamaCppProvider{
		name:    "multi-llamacpp",
		config:  cfg,
		logger:  logger,
		client:  &http.Client{Timeout: 60 * time.Second},
		manager: NewMultiInstanceManager(fullCfg, logger),
	}

	logger.WithField("provider", "multi-llamacpp").Info("🔧 Multi-Instance LlamaCpp Provider created")

	return provider
}

// Initialize starts the multi-instance manager
func (p *MultiInstanceLlamaCppProvider) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("🚀 Initializing Multi-Instance LlamaCpp Provider")

	if err := p.manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start multi-instance manager: %w", err)
	}

	p.isRunning = true
	p.lastHealthCheck = time.Now()

	p.logger.Info("✅ Multi-Instance LlamaCpp Provider initialized successfully")
	return nil
}

// Chat implements the chat interface using the appropriate model instance
func (p *MultiInstanceLlamaCppProvider) Chat(ctx context.Context, messages []ai.ChatMessage, opts ai.ChatOptions) (*ai.ChatResponse, error) {
	p.mu.Lock()
	p.requestCount++
	startTime := time.Now()
	p.mu.Unlock()

	// Get the appropriate instance for the model
	instance, err := p.manager.GetInstanceForModel(opts.Model)
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.mu.Unlock()
		return nil, fmt.Errorf("no available instance for model %s: %w", opts.Model, err)
	}

	p.logger.WithFields(logrus.Fields{
		"model":         opts.Model,
		"instance_id":   instance.ID,
		"instance_port": instance.Port,
		"message_count": len(messages),
	}).Info("🎯 Routing chat request to model instance")

	// Create a temporary LlamaCppProvider for this instance
	instanceProvider := &LlamaCppProvider{
		name:            fmt.Sprintf("instance-%s", instance.ID),
		config:          p.config,
		client:          p.client,
		logger:          p.logger,
		baseURL:         instance.BaseURL,
		defaultModel:    instance.ModelName,
		isRunning:       true,
		lastHealthCheck: time.Now(),
	}

	// Forward the request to the instance
	response, err := instanceProvider.Chat(ctx, messages, opts)

	// Update statistics
	p.mu.Lock()
	elapsed := time.Since(startTime)
	p.totalResponseTime += elapsed
	if err != nil {
		p.errorCount++
	}
	p.mu.Unlock()

	// Update instance statistics
	instance.mutex.Lock()
	instance.Stats.RequestCount++
	instance.Stats.LastRequestTime = time.Now()
	if err != nil {
		instance.Stats.ErrorCount++
	} else {
		// Update average response time
		if instance.Stats.RequestCount == 1 {
			instance.Stats.AvgResponseTime = elapsed
		} else {
			// Simple moving average
			instance.Stats.AvgResponseTime = (instance.Stats.AvgResponseTime + elapsed) / 2
		}
	}
	instance.mutex.Unlock()

	if err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"model":       opts.Model,
			"instance_id": instance.ID,
		}).Error("❌ Chat request failed")
		return nil, err
	}

	p.logger.WithFields(logrus.Fields{
		"model":         opts.Model,
		"instance_id":   instance.ID,
		"response_time": elapsed,
		"tokens_used":   response.TokensUsed.TotalTokens,
	}).Info("✅ Chat request completed successfully")

	return response, nil
}

// GetName returns the provider name
func (p *MultiInstanceLlamaCppProvider) GetName() string {
	return p.name
}

// IsAvailable checks if any instances are available
func (p *MultiInstanceLlamaCppProvider) IsAvailable(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isRunning {
		return false
	}

	// Check if any instances are healthy
	runningInstances := p.manager.getRunningInstances()
	return len(runningInstances) > 0
}

// GetModels returns all available models from all instances
func (p *MultiInstanceLlamaCppProvider) GetModels(ctx context.Context) ([]ai.ModelInfo, error) {
	p.manager.mutex.RLock()
	defer p.manager.mutex.RUnlock()

	var models []ai.ModelInfo

	for modelName, instance := range p.manager.instances {
		if instance.Status == StatusHealthy || instance.Status == StatusRunning {
			modelInfo := ai.ModelInfo{
				ID:          modelName,
				Name:        modelName,
				Description: fmt.Sprintf("LFM2 %s model running on multi-instance llamacpp", instance.Quantization),
				Provider:    "multi-llamacpp",
				Capabilities: []string{
					"chat",
					"completion",
					"tool_calling",
					"chatml_template",
				},
				MaxTokens:  instance.Config.ContextSize,
				Available:  true,
				LocalModel: true,
			}
			models = append(models, modelInfo)
		}
	}

	p.logger.WithField("model_count", len(models)).Info("📋 Retrieved available models from instances")

	return models, nil
}

// GetStats returns provider statistics
func (p *MultiInstanceLlamaCppProvider) GetStats() ai.ProviderStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var avgResponseTime time.Duration
	if p.requestCount > 0 {
		avgResponseTime = time.Duration(int64(p.totalResponseTime) / p.requestCount)
	}

	return ai.ProviderStats{
		RequestCount:        p.requestCount,
		ErrorCount:          p.errorCount,
		AverageResponseTime: avgResponseTime,
		LastUsed:            p.lastHealthCheck,
	}
}

// Complete implements text completion
func (p *MultiInstanceLlamaCppProvider) Complete(ctx context.Context, prompt string, opts ai.CompletionOptions) (*ai.CompletionResponse, error) {
	// Convert completion request to chat format
	messages := []ai.ChatMessage{
		{
			Role:      "user",
			Content:   prompt,
			Timestamp: time.Now(),
		},
	}

	chatOpts := ai.ChatOptions{
		Provider:    opts.Provider,
		Model:       opts.Model,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		TopP:        opts.TopP,
		Stream:      opts.Stream,
	}

	// Use chat interface
	chatResp, err := p.Chat(ctx, messages, chatOpts)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &ai.CompletionResponse{
		ID:               chatResp.ID,
		Text:             chatResp.Message.Content,
		FinishReason:     chatResp.FinishReason,
		Model:            chatResp.Model,
		Provider:         chatResp.Provider,
		ProcessingTimeMs: chatResp.ProcessingTimeMs,
		TokensUsed:       chatResp.TokensUsed,
		CreatedAt:        chatResp.CreatedAt,
	}, nil
}

// EstimateTokens provides a rough token count estimate
func (p *MultiInstanceLlamaCppProvider) EstimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English text
	return len(text) / 4
}

// GetRateLimit returns rate limit information
func (p *MultiInstanceLlamaCppProvider) GetRateLimit() ai.RateLimit {
	// Multi-instance setup can handle more requests
	return ai.RateLimit{
		RequestsPerMinute: 120, // Higher limit with multiple instances
		TokensPerMinute:   20000,
	}
}

// HealthCheck implements health checking
func (p *MultiInstanceLlamaCppProvider) HealthCheck(ctx context.Context) error {
	if !p.IsAvailable(ctx) {
		return fmt.Errorf("no healthy instances available")
	}

	p.mu.Lock()
	p.lastHealthCheck = time.Now()
	p.mu.Unlock()

	return nil
}

// Shutdown gracefully shuts down the multi-instance provider
func (p *MultiInstanceLlamaCppProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("🛑 Shutting down Multi-Instance LlamaCpp Provider")

	if err := p.manager.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown multi-instance manager: %w", err)
	}

	p.isRunning = false

	p.logger.Info("✅ Multi-Instance LlamaCpp Provider shutdown complete")
	return nil
}

// GetInstanceStats returns statistics for all model instances
func (p *MultiInstanceLlamaCppProvider) GetInstanceStats() map[string]InstanceStats {
	return p.manager.GetInstanceStats()
}

// GetManagerStatus returns overall manager status
func (p *MultiInstanceLlamaCppProvider) GetManagerStatus() map[string]interface{} {
	return p.manager.GetManagerStatus()
}

// ConfigureModels dynamically enables/disables models
func (p *MultiInstanceLlamaCppProvider) ConfigureModels(enabledModels map[string]bool) error {
	p.logger.WithField("enabled_models", enabledModels).Info("🔧 Configuring model availability")

	// Update enabled models and start/stop instances as needed
	for model, enabled := range enabledModels {
		p.manager.enabledModels[model] = enabled

		// Get the instance for this model
		instance, exists := p.manager.instances[model]
		if !exists {
			p.logger.WithField("model", model).Warn("Model instance not found, skipping start/stop")
			continue
		}

		if enabled {
			// Start the instance if it's not running
			if instance.Status != StatusRunning && instance.Status != StatusHealthy {
				p.logger.WithField("model", model).Info("🚀 Starting model instance")
				if err := p.manager.startInstance(context.Background(), instance); err != nil {
					p.logger.WithError(err).WithField("model", model).Error("❌ Failed to start model instance")
				}
			}
		} else {
			// Stop the instance if it's running
			if instance.Status == StatusRunning || instance.Status == StatusHealthy {
				p.logger.WithField("model", model).Info("🛑 Stopping model instance")
				if instance.Process != nil {
					instance.Process.Process.Kill()
					instance.Process.Wait()
					instance.Status = StatusStopped
				}
			}
		}
	}

	return nil
}

// RestartModel restarts a specific model instance
func (p *MultiInstanceLlamaCppProvider) RestartModel(ctx context.Context, modelName string) error {
	p.logger.WithField("model_name", modelName).Info("🔄 Restarting model instance")

	return p.manager.RestartModel(ctx, modelName)
}
