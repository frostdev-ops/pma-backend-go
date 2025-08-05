package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// LLMManager manages multiple AI providers with fallback capabilities
type LLMManager struct {
	providers       []LLMProvider
	providersByName map[string]LLMProvider
	primaryProvider string
	fallbackEnabled bool
	fallbackDelay   time.Duration
	maxRetries      int
	timeout         time.Duration
	logger          *logrus.Logger
	mu              sync.RWMutex

	// Statistics and monitoring
	requestCount   map[string]int64
	errorCount     map[string]int64
	responseTime   map[string]time.Duration
	lastUsage      map[string]time.Time
	circuitBreaker map[string]*CircuitBreaker

	// Context awareness
	contextExtractor ContextExtractor

	// Provider factories
	providerFactories map[string]ProviderFactory

	// Database configuration storage
	configRepo repositories.ConfigRepository
}

// ProviderFactory creates new provider instances
type ProviderFactory func(cfg config.AIProviderConfig, logger *logrus.Logger) LLMProvider

// CircuitBreaker implements circuit breaker pattern for providers
type CircuitBreaker struct {
	failures    int
	lastFailure time.Time
	state       CircuitState
	mu          sync.RWMutex
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// ContextExtractor extracts context for AI requests
type ContextExtractor interface {
	ExtractContext(ctx context.Context, userID string) (*ConversationContext, error)
}

// NewLLMManager creates a new LLM manager with configured providers
func NewLLMManager(cfg *config.Config, logger *logrus.Logger, configRepo repositories.ConfigRepository) (*LLMManager, error) {
	manager := &LLMManager{
		providers:         make([]LLMProvider, 0),
		providersByName:   make(map[string]LLMProvider),
		primaryProvider:   "",   // Will be set dynamically based on available providers
		fallbackEnabled:   true, // Enable fallback between providers
		maxRetries:        3,    // Default max retries
		logger:            logger,
		requestCount:      make(map[string]int64),
		errorCount:        make(map[string]int64),
		responseTime:      make(map[string]time.Duration),
		lastUsage:         make(map[string]time.Time),
		circuitBreaker:    make(map[string]*CircuitBreaker),
		providerFactories: make(map[string]ProviderFactory),
		configRepo:        configRepo,
		fallbackDelay:     0, // No fallback delay needed
	}

	// Parse timeout from LlamaCpp config or use default
	if cfg.AI.LlamaCpp.Timeout != "" {
		if timeout, err := time.ParseDuration(cfg.AI.LlamaCpp.Timeout); err == nil {
			manager.timeout = timeout
		} else {
			manager.timeout = 60 * time.Second
		}
	} else {
		manager.timeout = 60 * time.Second
	}

	// Update max retries from LlamaCpp config if available
	if cfg.AI.LlamaCpp.MaxRetries > 0 {
		manager.maxRetries = cfg.AI.LlamaCpp.MaxRetries
	}

	// Provider factories will be registered separately to avoid import cycles

	// Build provider configs from configuration
	var providerConfigs []config.AIProviderConfig

	// Add LlamaCpp provider if enabled (PRIMARY PROVIDER)
	if cfg.AI.Enabled && cfg.AI.LlamaCpp.Enabled {
		// Check if multi-instance mode is enabled via environment or config
		useMultiInstance := cfg.AI.LlamaCpp.MultiInstance || true // Default to multi-instance for now
		
		if useMultiInstance {
			// Use multi-instance provider
			providerConfigs = append(providerConfigs, config.AIProviderConfig{
				Type:         "multi-llamacpp",
				Enabled:      cfg.AI.LlamaCpp.Enabled,
				URL:          cfg.AI.LlamaCpp.BaseURL,
				APIKey:       cfg.AI.LlamaCpp.APIKey,
				DefaultModel: cfg.AI.LlamaCpp.DefaultModel,
				MaxTokens:    32768, // Full 32K context for multi-instance
				AutoStart:    true,  // Always auto-start multi-instance manager
			})
		} else {
			// Use single-instance provider
			providerConfigs = append(providerConfigs, config.AIProviderConfig{
				Type:         "llamacpp",
				Enabled:      cfg.AI.LlamaCpp.Enabled,
				URL:          cfg.AI.LlamaCpp.BaseURL,
				APIKey:       cfg.AI.LlamaCpp.APIKey,
				DefaultModel: cfg.AI.LlamaCpp.DefaultModel,
				MaxTokens:    4096, // Default for LLMs
				AutoStart:    true, // Always auto-start llama.cpp for LFM2
			})
		}
	}

	// Initialize providers if any are configured
	if len(providerConfigs) > 0 {
		if err := manager.initializeProviders(providerConfigs, logger); err != nil {
			return nil, fmt.Errorf("failed to initialize providers: %w", err)
		}

		// Set primary provider based on what's available
		if len(manager.providers) > 0 {
			// Prefer Multi-Instance LlamaCpp if available, then LlamaCpp, otherwise use the first available provider
			for _, provider := range manager.providers {
				if provider.GetName() == "multi-llamacpp" {
					manager.primaryProvider = "multi-llamacpp"
					break
				} else if provider.GetName() == "llamacpp" {
					manager.primaryProvider = "llamacpp"
				}
			}
			if manager.primaryProvider == "" {
				manager.primaryProvider = manager.providers[0].GetName()
			}
		}
	}

	return manager, nil
}

// RegisterProviderFactory registers a provider factory function
func (m *LLMManager) RegisterProviderFactory(providerType string, factory ProviderFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providerFactories[providerType] = factory
}

// Initialize initializes all providers
func (m *LLMManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var initErrors []error
	for _, provider := range m.providers {
		if err := provider.Initialize(ctx); err != nil {
			m.logger.WithError(err).WithField("provider", provider.GetName()).Warn("Failed to initialize provider")
			initErrors = append(initErrors, err)
		} else {
			m.logger.WithField("provider", provider.GetName()).Info("Provider initialized successfully")
		}
	}

	// Return error only if no providers were successfully initialized
	if len(initErrors) == len(m.providers) && len(m.providers) > 0 {
		return fmt.Errorf("failed to initialize any providers: %v", initErrors)
	}

	return nil
}

// Shutdown shuts down all providers
func (m *LLMManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var shutdownErrors []error
	for _, provider := range m.providers {
		if err := provider.Shutdown(ctx); err != nil {
			m.logger.WithError(err).WithField("provider", provider.GetName()).Warn("Failed to shutdown provider")
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", shutdownErrors)
	}

	return nil
}

// Chat performs chat completion with the primary provider (LlamaCpp)
func (m *LLMManager) Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (*ChatResponse, error) {
	m.logger.WithFields(logrus.Fields{
		"messageCount": len(messages),
		"provider":     opts.Provider,
		"model":        opts.Model,
		"maxTokens":    opts.MaxTokens,
		"temperature":  opts.Temperature,
		"timeout":      m.timeout,
	}).Info("🚀 LLM Manager Chat request started")

	// Add timeout to context
	if m.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
		defer cancel()
		m.logger.WithField("timeout", m.timeout).Info("⏱️ Applied timeout to context")
	}

	// Use primary provider (LlamaCpp) if not specified
	if opts.Provider == "" {
		opts.Provider = m.primaryProvider
		m.logger.WithField("provider", opts.Provider).Info("🔄 Using primary provider")
	}

	m.logger.WithField("requestedProvider", opts.Provider).Info("🔍 Looking up provider")

	// Get the requested provider
	provider, exists := m.providersByName[opts.Provider]
	if !exists {
		m.logger.WithFields(logrus.Fields{
			"requestedProvider":  opts.Provider,
			"availableProviders": len(m.providersByName),
		}).Error("❌ Provider not found")
		return nil, fmt.Errorf("provider %s not found", opts.Provider)
	}

	m.logger.WithField("provider", provider.GetName()).Info("✅ Provider found")

	// Check if provider is available
	m.logger.Info("🔍 Checking provider availability")
	if !provider.IsAvailable(ctx) {
		m.logger.WithField("provider", provider.GetName()).Error("❌ Provider is not available")
		return nil, fmt.Errorf("provider %s is not available", opts.Provider)
	}

	m.logger.WithField("provider", provider.GetName()).Info("✅ Provider is available")

	// Make the request using the provider interface
	m.logger.Info("📤 Sending chat request to provider")
	startTime := time.Now()
	response, err := provider.Chat(ctx, messages, opts)
	duration := time.Since(startTime)

	if err != nil {
		m.logger.WithError(err).WithFields(logrus.Fields{
			"provider": provider.GetName(),
			"duration": duration,
			"model":    opts.Model,
		}).Error("❌ Chat request failed")
		m.recordError(provider.GetName())
		return nil, err
	}

	m.logger.WithFields(logrus.Fields{
		"provider":     provider.GetName(),
		"duration":     duration,
		"responseID":   response.ID,
		"finishReason": response.FinishReason,
		"tokenCount":   response.TokensUsed.TotalTokens,
	}).Info("✅ Chat request completed successfully")

	m.recordSuccess(provider.GetName(), duration)

	// Return the response directly (it's already in the correct format)
	return response, nil
}

// GetProviders returns all available providers
func (m *LLMManager) GetProviders(ctx context.Context) []LLMProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]LLMProvider, 0, len(m.providers))
	providers = append(providers, m.providers...)
	return providers
}

// GetProvider returns a specific provider by name
func (m *LLMManager) GetProvider(name string) (LLMProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.providersByName[name]
	return provider, exists
}

// GetPrimaryProvider returns the name of the primary provider
func (m *LLMManager) GetPrimaryProvider() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.primaryProvider
}

// GetModels returns all available models from all providers
func (m *LLMManager) GetModels(ctx context.Context) ([]ModelInfo, error) {
	m.mu.RLock()
	providers := make([]LLMProvider, len(m.providers))
	copy(providers, m.providers)
	m.mu.RUnlock()

	var allModels []ModelInfo
	for _, provider := range providers {
		if provider.IsAvailable(ctx) {
			models, err := provider.GetModels(ctx)
			if err != nil {
				m.logger.WithError(err).WithField("provider", provider.GetName()).Warn("Failed to get models")
				continue
			}
			allModels = append(allModels, models...)
		}
	}

	return allModels, nil
}

// initializeProviders initializes providers from configuration
func (m *LLMManager) initializeProviders(configs []config.AIProviderConfig, logger *logrus.Logger) error {
	for _, cfg := range configs {
		factory, exists := m.providerFactories[cfg.Type]
		if !exists {
			logger.WithField("type", cfg.Type).Warn("No factory registered for provider type")
			continue
		}

		provider := factory(cfg, logger)
		m.providers = append(m.providers, provider)
		m.providersByName[provider.GetName()] = provider
		m.circuitBreaker[provider.GetName()] = &CircuitBreaker{
			state: CircuitClosed,
		}

		logger.WithField("provider", provider.GetName()).Info("Provider registered")
	}

	return nil
}

// Complete performs text completion using the configured providers
func (m *LLMManager) Complete(ctx context.Context, prompt string, opts CompletionOptions) (*CompletionResponse, error) {
	if opts.Provider == "" {
		opts.Provider = m.primaryProvider
	}

	provider, exists := m.providersByName[opts.Provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", opts.Provider)
	}

	// Check if provider is available
	if !provider.IsAvailable(ctx) {
		return nil, fmt.Errorf("provider %s is not available", opts.Provider)
	}

	// Make the request using the provider interface
	response, err := provider.Complete(ctx, prompt, opts)
	if err != nil {
		m.recordError(provider.GetName())
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	m.recordSuccess(provider.GetName(), time.Since(time.Now()))
	return response, nil
}

// recordSuccess records successful provider usage
func (m *LLMManager) recordSuccess(providerName string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCount[providerName]++
	m.responseTime[providerName] += duration
	m.lastUsage[providerName] = time.Now()
}

// recordError records provider error

func (m *LLMManager) recordError(providerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorCount[providerName]++
}

// ReinitializeProviders reinitializes providers with new configuration
func (m *LLMManager) ReinitializeProviders(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing providers
	m.providers = nil
	m.providersByName = make(map[string]LLMProvider)
	m.circuitBreaker = make(map[string]*CircuitBreaker)

	// Build new provider configs
	var providerConfigs []config.AIProviderConfig

	// Add LlamaCpp provider if enabled (PRIMARY PROVIDER)
	m.logger.WithFields(logrus.Fields{
		"ai_enabled":       cfg.AI.Enabled,
		"llamacpp_enabled": cfg.AI.LlamaCpp.Enabled,
		"auto_start":       cfg.AI.LlamaCpp.AutoStart,
		"binary_path":      cfg.AI.LlamaCpp.BinaryPath,
		"model_path":       cfg.AI.LlamaCpp.ModelPath,
	}).Info("Checking LlamaCpp provider configuration")

	if cfg.AI.Enabled && cfg.AI.LlamaCpp.Enabled {
		m.logger.Info("Adding LlamaCpp provider to configuration list")
		providerConfigs = append(providerConfigs, config.AIProviderConfig{
			Type:         "llamacpp",
			Enabled:      cfg.AI.LlamaCpp.Enabled,
			URL:          cfg.AI.LlamaCpp.BaseURL,
			APIKey:       cfg.AI.LlamaCpp.APIKey,
			DefaultModel: cfg.AI.LlamaCpp.DefaultModel,
			MaxTokens:    4096, // Default for LLMs
			AutoStart:    cfg.AI.LlamaCpp.AutoStart,
		})
	} else {
		m.logger.WithFields(logrus.Fields{
			"ai_enabled":       cfg.AI.Enabled,
			"llamacpp_enabled": cfg.AI.LlamaCpp.Enabled,
		}).Warn("LlamaCpp provider not enabled, skipping")
	}

	// Initialize providers
	if len(providerConfigs) > 0 {
		if err := m.initializeProviders(providerConfigs, m.logger); err != nil {
			return fmt.Errorf("failed to initialize providers: %w", err)
		}

		// Set primary provider based on what's available
		if len(m.providers) > 0 {
			// Prefer LlamaCpp if available, otherwise use the first available provider
			for _, provider := range m.providers {
				if provider.GetName() == "llamacpp" {
					m.primaryProvider = "llamacpp"
					break
				}
			}
			if m.primaryProvider == "" {
				m.primaryProvider = m.providers[0].GetName()
			}
		}
	}

	return nil
}
