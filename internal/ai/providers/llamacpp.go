package providers

import (
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// LlamaCppProvider implements the LLMProvider interface for llama.cpp server
type LlamaCppProvider struct {
	name              string
	config            config.AIProviderConfig
	llamaCppConfig    config.LlamaCppConfig // Specific llamacpp configuration
	client            *http.Client
	logger            *logrus.Logger
	baseURL           string
	defaultModel      string
	mu                sync.RWMutex
	isRunning         bool
	processCmd        *exec.Cmd
	lastHealthCheck   time.Time
	errorCount        int64
	requestCount      int64
	totalResponseTime time.Duration
	availableModels   []ai.ModelInfo
	modelCache        map[string]ai.ModelInfo
	cacheTTL          time.Duration
	lastCacheUpdate   time.Time

	// LFM2-specific fields
	currentModel string
	modelPath    string
	serverPort   int
	processReady chan bool

	// Template engine for LFM2 ChatML
	templateEngine *LFM2TemplateEngine
}

// LFM2TemplateEngine handles LFM2 ChatML template formatting and tool calling
type LFM2TemplateEngine struct {
	systemPrompt string
	configRepo   repositories.ConfigRepository
	logger       *logrus.Logger
}

// LFM2ToolCall represents a tool call in LFM2 format
type LFM2ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// NewLlamaCppProvider creates a new llama.cpp provider instance
func NewLlamaCppProvider(cfg config.AIProviderConfig, logger *logrus.Logger) *LlamaCppProvider {
	baseURL := cfg.URL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	return &LlamaCppProvider{
		name:           "llamacpp",
		config:         cfg,
		client:         &http.Client{Timeout: 300 * time.Second}, // Increased for F16 model
		logger:         logger,
		baseURL:        baseURL,
		defaultModel:   cfg.DefaultModel,
		modelCache:     make(map[string]ai.ModelInfo),
		cacheTTL:       5 * time.Minute,
		serverPort:     8000,
		processReady:   make(chan bool, 1),
		templateEngine: NewLFM2TemplateEngine(),
	}
}

// NewLlamaCppProviderWithConfig creates a new llama.cpp provider instance with full config access
func NewLlamaCppProviderWithConfig(providerCfg config.AIProviderConfig, fullCfg *config.Config, logger *logrus.Logger) *LlamaCppProvider {
	baseURL := providerCfg.URL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	// Extract llamacpp-specific config
	llamaCppCfg := fullCfg.AI.LlamaCpp

	// Set server port from config if available
	serverPort := 8000
	if llamaCppCfg.ServerPort > 0 {
		serverPort = llamaCppCfg.ServerPort
	}

	return &LlamaCppProvider{
		name:           "llamacpp",
		config:         providerCfg,
		llamaCppConfig: llamaCppCfg,                              // Store the specific config
		client:         &http.Client{Timeout: 300 * time.Second}, // Increased for F16 model
		logger:         logger,
		baseURL:        baseURL,
		defaultModel:   providerCfg.DefaultModel,
		modelCache:     make(map[string]ai.ModelInfo),
		cacheTTL:       5 * time.Minute,
		serverPort:     serverPort,
		processReady:   make(chan bool, 1),
		templateEngine: NewLFM2TemplateEngine(),
	}
}

// NewLlamaCppProviderWithConfigAndRepo creates a new llama.cpp provider instance with full config and repository access
func NewLlamaCppProviderWithConfigAndRepo(providerCfg config.AIProviderConfig, fullCfg *config.Config, logger *logrus.Logger, configRepo repositories.ConfigRepository) *LlamaCppProvider {
	baseURL := providerCfg.URL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	// Extract llamacpp-specific config
	llamaCppCfg := fullCfg.AI.LlamaCpp

	// Set server port from config if available
	serverPort := 8000
	if llamaCppCfg.ServerPort > 0 {
		serverPort = llamaCppCfg.ServerPort
	}

	return &LlamaCppProvider{
		name:           "llamacpp",
		config:         providerCfg,
		llamaCppConfig: llamaCppCfg,                              // Store the specific config
		client:         &http.Client{Timeout: 300 * time.Second}, // Increased for F16 model
		logger:         logger,
		baseURL:        baseURL,
		defaultModel:   providerCfg.DefaultModel,
		modelCache:     make(map[string]ai.ModelInfo),
		cacheTTL:       5 * time.Minute,
		serverPort:     serverPort,
		processReady:   make(chan bool, 1),
		templateEngine: NewLFM2TemplateEngineWithConfig(configRepo, logger),
	}
}

// NewLFM2TemplateEngine creates a new LFM2 template engine
func NewLFM2TemplateEngine() *LFM2TemplateEngine {
	return &LFM2TemplateEngine{
		systemPrompt: "You are Wattson, an intelligent home automation assistant.", // Default fallback
	}
}

// NewLFM2TemplateEngineWithConfig creates a new LFM2 template engine with config repository access
func NewLFM2TemplateEngineWithConfig(configRepo repositories.ConfigRepository, logger *logrus.Logger) *LFM2TemplateEngine {
	return &LFM2TemplateEngine{
		systemPrompt: "You are Wattson, an intelligent home automation assistant.", // Default fallback
		configRepo:   configRepo,
		logger:       logger,
	}
}
