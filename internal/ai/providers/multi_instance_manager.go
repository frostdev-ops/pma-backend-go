package providers

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// ModelInstance represents a single llama.cpp server instance
type ModelInstance struct {
	ID            string                 `json:"id"`
	ModelPath     string                 `json:"model_path"`
	ModelName     string                 `json:"model_name"`
	Quantization  string                 `json:"quantization"`
	Port          int                    `json:"port"`
	BaseURL       string                 `json:"base_url"`
	Status        InstanceStatus         `json:"status"`
	Process       *exec.Cmd              `json:"-"`
	Provider      *LlamaCppProvider      `json:"-"`
	Config        ModelInstanceConfig    `json:"config"`
	Stats         InstanceStats          `json:"stats"`
	StartedAt     time.Time              `json:"started_at"`
	LastHealthy   time.Time              `json:"last_healthy"`
	RestartCount  int                    `json:"restart_count"`
	mutex         sync.RWMutex           `json:"-"`
}

// InstanceStatus represents the current status of a model instance
type InstanceStatus string

const (
	StatusStopped    InstanceStatus = "stopped"
	StatusStarting   InstanceStatus = "starting"
	StatusRunning    InstanceStatus = "running"
	StatusHealthy    InstanceStatus = "healthy"
	StatusUnhealthy  InstanceStatus = "unhealthy"
	StatusCrashed    InstanceStatus = "crashed"
	StatusDisabled   InstanceStatus = "disabled"
)

// ModelInstanceConfig contains configuration for a specific model instance
type ModelInstanceConfig struct {
	Enabled         bool          `json:"enabled"`
	AutoStart       bool          `json:"auto_start"`
	ContextSize     int           `json:"context_size"`
	MaxTokens       int           `json:"max_tokens"`
	Threads         int           `json:"threads"`
	MemoryLimit     int64         `json:"memory_limit_mb"`
	HealthTimeout   time.Duration `json:"health_timeout"`
	StartupTimeout  time.Duration `json:"startup_timeout"`
	MaxRestarts     int           `json:"max_restarts"`
	RestartDelay    time.Duration `json:"restart_delay"`
}

// InstanceStats tracks performance metrics for an instance
type InstanceStats struct {
	RequestCount     int64         `json:"request_count"`
	ErrorCount       int64         `json:"error_count"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	MemoryUsageMB    int64         `json:"memory_usage_mb"`
	CPUUsagePercent  float64       `json:"cpu_usage_percent"`
	LastRequestTime  time.Time     `json:"last_request_time"`
	UptimeSeconds    int64         `json:"uptime_seconds"`
}

// MultiInstanceManager manages multiple llama.cpp server instances
type MultiInstanceManager struct {
	instances       map[string]*ModelInstance
	config          *config.Config
	logger          *logrus.Logger
	resourceMonitor *ResourceMonitor
	portManager     *PortManager
	mutex           sync.RWMutex
	
	// Dynamic configuration
	maxInstances    int
	basePort        int
	modelsDir       string
	enabledModels   map[string]bool
	
	// Monitoring
	healthCheckInterval time.Duration
	monitoringEnabled   bool
	shutdownChan        chan struct{}
}

// ResourceMonitor tracks system resource usage
type ResourceMonitor struct {
	maxMemoryMB     int64
	maxCPUPercent   float64
	currentMemoryMB int64
	currentCPU      float64
	logger          *logrus.Logger
	mutex           sync.RWMutex
}

// PortManager manages port allocation for instances
type PortManager struct {
	basePort     int
	maxPorts     int
	allocatedPorts map[int]string
	mutex        sync.RWMutex
}

// NewMultiInstanceManager creates a new multi-instance model manager
func NewMultiInstanceManager(cfg *config.Config, logger *logrus.Logger) *MultiInstanceManager {
	basePort := 4000  // Use ports 4000-4010 as requested

	manager := &MultiInstanceManager{
		instances:           make(map[string]*ModelInstance),
		config:              cfg,
		logger:              logger,
		maxInstances:        4, // Dynamic, can be adjusted based on resources
		basePort:            basePort,
		modelsDir:           "/opt/pma/llama.cpp/models",
		enabledModels:       make(map[string]bool),
		healthCheckInterval: 30 * time.Second,
		monitoringEnabled:   true,
		shutdownChan:        make(chan struct{}),
	}

	// Initialize resource monitor
	manager.resourceMonitor = &ResourceMonitor{
		maxMemoryMB:   8192, // 8GB limit for RPi5
		maxCPUPercent: 90.0, // 90% CPU limit
		logger:        logger,
	}

	// Initialize port manager
	manager.portManager = &PortManager{
		basePort:       basePort,
		maxPorts:       10, // Support up to 10 instances
		allocatedPorts: make(map[int]string),
	}

	// Initialize default enabled models (can be configured dynamically)
	manager.initializeEnabledModels()

	logger.WithFields(logrus.Fields{
		"base_port":      basePort,
		"max_instances":  manager.maxInstances,
		"models_dir":     manager.modelsDir,
		"enabled_models": manager.enabledModels,
	}).Info("🚀 Multi-Instance Model Manager initialized")

	return manager
}

// initializeEnabledModels sets up the default enabled models
func (m *MultiInstanceManager) initializeEnabledModels() {
	// Default models to load (can be overridden by configuration)
	defaultModels := map[string]bool{
		"LFM2-1.2B-Q2":     true,  // Fastest for simple queries
		"LFM2-1.2B-Q4_K_M": true,  // Balanced performance
		"LFM2-1.2B-Q8":     true,  // High precision for complex queries
		"LFM2-1.2B":        false, // F16 disabled by default (too memory intensive)
	}

	for model, enabled := range defaultModels {
		m.enabledModels[model] = enabled
	}

	m.logger.WithField("enabled_models", m.enabledModels).Info("📋 Default model configuration loaded")
}

// Start initializes and starts all enabled model instances
func (m *MultiInstanceManager) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Info("🚀 Starting Multi-Instance Model Manager")

	// Start resource monitoring
	go m.startResourceMonitoring(ctx)

	// Create and start instances for enabled models
	for modelName, enabled := range m.enabledModels {
		if !enabled {
			m.logger.WithField("model", modelName).Info("📋 Model disabled, skipping")
			continue
		}

		if err := m.createAndStartInstance(ctx, modelName); err != nil {
			m.logger.WithError(err).WithField("model", modelName).Error("❌ Failed to start model instance")
			// Continue with other models instead of failing completely
			continue
		}
	}

	// Start health monitoring
	go m.startHealthMonitoring(ctx)

	runningInstances := len(m.getRunningInstances())
	m.logger.WithFields(logrus.Fields{
		"running_instances": runningInstances,
		"total_enabled":     m.countEnabledModels(),
	}).Info("✅ Multi-Instance Model Manager started successfully")

	return nil
}

// createAndStartInstance creates and starts a new model instance
func (m *MultiInstanceManager) createAndStartInstance(ctx context.Context, modelName string) error {
	// Find model file
	modelPath, err := m.findModelFile(modelName)
	if err != nil {
		return fmt.Errorf("model file not found for %s: %w", modelName, err)
	}

	// Allocate port
	port, err := m.portManager.AllocatePort(modelName)
	if err != nil {
		return fmt.Errorf("failed to allocate port for %s: %w", modelName, err)
	}

	// Create instance configuration
	instanceConfig := m.createInstanceConfig(modelName)

	// Create model instance
	instance := &ModelInstance{
		ID:           fmt.Sprintf("%s-%d", modelName, port),
		ModelPath:    modelPath,
		ModelName:    modelName,
		Quantization: m.extractQuantization(modelName),
		Port:         port,
		BaseURL:      fmt.Sprintf("http://localhost:%d", port),
		Status:       StatusStopped,
		Config:       instanceConfig,
		Stats:        InstanceStats{},
	}

	// Start the instance
	if err := m.startInstance(ctx, instance); err != nil {
		m.portManager.ReleasePort(port)
		return fmt.Errorf("failed to start instance for %s: %w", modelName, err)
	}

	// Register instance
	m.instances[modelName] = instance

	m.logger.WithFields(logrus.Fields{
		"model":        modelName,
		"instance_id":  instance.ID,
		"port":         port,
		"model_path":   modelPath,
		"quantization": instance.Quantization,
	}).Info("✅ Model instance started successfully")

	return nil
}

// startInstance starts a single model instance
func (m *MultiInstanceManager) startInstance(ctx context.Context, instance *ModelInstance) error {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	instance.Status = StatusStarting
	instance.StartedAt = time.Now()

	// Prepare command arguments
	args := []string{
		"--server",
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", instance.Port),
		"--model", instance.ModelPath,
		"--n-predict", "-1",
		"--ctx-size", fmt.Sprintf("%d", instance.Config.ContextSize),
		"--threads", fmt.Sprintf("%d", instance.Config.Threads),
		"--n-gpu-layers", "0", // CPU-only for RPi5
		"--chat-template", "chatml",
		"--verbose",
	}

	// Find llama.cpp binary
	binaryPath, err := m.findLlamaCppBinary()
	if err != nil {
		return fmt.Errorf("llama.cpp binary not found: %w", err)
	}

	// Create command
	instance.Process = exec.CommandContext(ctx, binaryPath, args...)

	// Start the process
	if err := instance.Process.Start(); err != nil {
		return fmt.Errorf("failed to start llama.cpp process: %w", err)
	}

	instance.Status = StatusStarting

	// Wait for instance to be ready
	if err := m.waitForInstanceReady(ctx, instance); err != nil {
		instance.Process.Process.Kill()
		return fmt.Errorf("instance failed to start: %w", err)
	}

	instance.Status = StatusHealthy
	instance.LastHealthy = time.Now()

	return nil
}

// Additional methods will be implemented in the next part...