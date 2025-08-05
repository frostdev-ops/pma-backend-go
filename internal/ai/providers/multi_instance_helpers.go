package providers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Helper methods for MultiInstanceManager

// findModelFile locates the model file for a given model name
func (m *MultiInstanceManager) findModelFile(modelName string) (string, error) {
	// Model file mappings
	modelFiles := map[string][]string{
		"LFM2-1.2B-Q2": {
			"lfm2/LFM2-1.2B-Q2_K.gguf",
			"LFM2-1.2B-Q2_K.gguf",
			"LFM2-1.2B-Q2.gguf",
		},
		"LFM2-1.2B-Q4_K_M": {
			"lfm2/LFM2-1.2B-Q4_K_M.gguf",
			"LFM2-1.2B-Q4_K_M.gguf",
		},
		"LFM2-1.2B-Q4": {
			"lfm2/LFM2-1.2B-Q4_K_M.gguf",
			"LFM2-1.2B-Q4_K_M.gguf",
			"LFM2-1.2B-Q4.gguf",
		},
		"LFM2-1.2B-Q8": {
			"lfm2/LFM2-1.2B-Q8_0.gguf",
			"LFM2-1.2B-Q8_0.gguf",
			"LFM2-1.2B-Q8.gguf",
		},
		"LFM2-1.2B": {
			"lfm2-full/LFM2-1.2B-F16.gguf",
			"lfm2-full/LFM2-1.2B.gguf",
			"LFM2-1.2B-F16.gguf",
			"LFM2-1.2B.gguf",
		},
	}

	// Search for model files
	possibleFiles, exists := modelFiles[modelName]
	if !exists {
		return "", fmt.Errorf("unknown model: %s", modelName)
	}

	modelDirs := []string{
		m.modelsDir,
		"./models",
		"/opt/pma/models",
		"/home/pma/models",
		"/usr/local/share/models",
	}

	for _, dir := range modelDirs {
		for _, file := range possibleFiles {
			fullPath := filepath.Join(dir, file)
			if _, err := os.Stat(fullPath); err == nil {
				m.logger.WithFields(logrus.Fields{
					"model": modelName,
					"path":  fullPath,
				}).Debug("📁 Found model file")
				return fullPath, nil
			}
		}
	}

	return "", fmt.Errorf("model file not found for %s in any search directory", modelName)
}

// findLlamaCppBinary locates the llama.cpp server binary
func (m *MultiInstanceManager) findLlamaCppBinary() (string, error) {
	possiblePaths := []string{
		"/opt/pma/llama.cpp/build/bin/llama-server",
		"/usr/local/bin/llama-server",
		"/usr/bin/llama-server",
		"./llama-server",
		"/opt/llama.cpp/llama-server",
		"/home/pma/llama.cpp/llama-server",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("llama.cpp binary not found in any expected location")
}

// extractQuantization extracts quantization type from model name
func (m *MultiInstanceManager) extractQuantization(modelName string) string {
	if strings.Contains(modelName, "Q2") {
		return "Q2"
	}
	if strings.Contains(modelName, "Q4") {
		return "Q4"
	}
	if strings.Contains(modelName, "Q8") {
		return "Q8"
	}
	if strings.Contains(modelName, "F16") || (!strings.Contains(modelName, "Q")) {
		return "F16"
	}
	return "Unknown"
}

// createInstanceConfig creates configuration for a model instance
func (m *MultiInstanceManager) createInstanceConfig(modelName string) ModelInstanceConfig {
	// Base configuration
	config := ModelInstanceConfig{
		Enabled:        true,
		AutoStart:      true,
		ContextSize:    32768, // Full 32K context for all models
		MaxTokens:      -1,    // No limit
		Threads:        4,     // Optimized for RPi5
		HealthTimeout:  10 * time.Second,
		StartupTimeout: 60 * time.Second,
		MaxRestarts:    3,
		RestartDelay:   10 * time.Second,
	}

	// Model-specific optimizations
	switch {
	case strings.Contains(modelName, "Q2"):
		config.MemoryLimit = 1024 // 1GB for Q2
		config.StartupTimeout = 30 * time.Second
	case strings.Contains(modelName, "Q4"):
		config.MemoryLimit = 2048 // 2GB for Q4
		config.StartupTimeout = 45 * time.Second
	case strings.Contains(modelName, "Q8"):
		config.MemoryLimit = 3072 // 3GB for Q8
		config.StartupTimeout = 60 * time.Second
	default: // F16
		config.MemoryLimit = 4096 // 4GB for F16
		config.StartupTimeout = 90 * time.Second
	}

	return config
}

// waitForInstanceReady waits for an instance to be ready
func (m *MultiInstanceManager) waitForInstanceReady(ctx context.Context, instance *ModelInstance) error {
	timeout := instance.Config.StartupTimeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	healthURL := fmt.Sprintf("%s/health", instance.BaseURL)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	m.logger.WithFields(logrus.Fields{
		"instance_id": instance.ID,
		"port":        instance.Port,
		"timeout":     timeout,
	}).Info("⏳ Waiting for instance to be ready")

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for instance %s to be ready", instance.ID)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
			if err != nil {
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				m.logger.WithField("instance_id", instance.ID).Info("✅ Instance is ready")
				return nil
			}
		}
	}
}

// GetInstanceForModel returns the best available instance for a model
func (m *MultiInstanceManager) GetInstanceForModel(modelName string) (*ModelInstance, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Direct match first
	if instance, exists := m.instances[modelName]; exists {
		if instance.Status == StatusHealthy || instance.Status == StatusRunning {
			return instance, nil
		}
	}

	// Smart fallback based on quantization hierarchy
	fallbackOrder := m.getFallbackOrder(modelName)

	for _, fallback := range fallbackOrder {
		if instance, exists := m.instances[fallback]; exists {
			if instance.Status == StatusHealthy || instance.Status == StatusRunning {
				m.logger.WithFields(logrus.Fields{
					"requested": modelName,
					"fallback":  fallback,
					"reason":    "intelligent_fallback",
				}).Info("🔄 Using fallback model instance")
				return instance, nil
			}
		}
	}

	return nil, fmt.Errorf("no healthy instance available for model %s or its fallbacks", modelName)
}

// getFallbackOrder returns intelligent fallback order for models
func (m *MultiInstanceManager) getFallbackOrder(modelName string) []string {
	fallbacks := map[string][]string{
		"LFM2-1.2B-Q2":     {"LFM2-1.2B-Q4_K_M", "LFM2-1.2B-Q4", "LFM2-1.2B-Q8"},
		"LFM2-1.2B-Q4":     {"LFM2-1.2B-Q4_K_M", "LFM2-1.2B-Q8", "LFM2-1.2B-Q2"},
		"LFM2-1.2B-Q4_K_M": {"LFM2-1.2B-Q4", "LFM2-1.2B-Q8", "LFM2-1.2B-Q2"},
		"LFM2-1.2B-Q8":     {"LFM2-1.2B-Q4_K_M", "LFM2-1.2B-Q4", "LFM2-1.2B"},
		"LFM2-1.2B":        {"LFM2-1.2B-Q8", "LFM2-1.2B-Q4_K_M", "LFM2-1.2B-Q4"},
	}

	if order, exists := fallbacks[modelName]; exists {
		return order
	}

	// Default fallback to Q4
	return []string{"LFM2-1.2B-Q4_K_M", "LFM2-1.2B-Q4", "LFM2-1.2B-Q2"}
}

// startResourceMonitoring monitors system resources
func (m *MultiInstanceManager) startResourceMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			m.updateResourceUsage()
		}
	}
}

// updateResourceUsage updates current resource usage statistics
func (m *MultiInstanceManager) updateResourceUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.resourceMonitor.mutex.Lock()
	m.resourceMonitor.currentMemoryMB = int64(memStats.Alloc / 1024 / 1024)
	m.resourceMonitor.mutex.Unlock()

	// Log resource usage
	if m.resourceMonitor.currentMemoryMB > m.resourceMonitor.maxMemoryMB*80/100 {
		m.logger.WithFields(logrus.Fields{
			"current_memory_mb": m.resourceMonitor.currentMemoryMB,
			"max_memory_mb":     m.resourceMonitor.maxMemoryMB,
			"usage_percent":     (m.resourceMonitor.currentMemoryMB * 100) / m.resourceMonitor.maxMemoryMB,
		}).Warn("⚠️ High memory usage detected")
	}
}

// startHealthMonitoring monitors instance health
func (m *MultiInstanceManager) startHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.shutdownChan:
			return
		case <-ticker.C:
			m.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks checks health of all instances
func (m *MultiInstanceManager) performHealthChecks(ctx context.Context) {
	m.mutex.RLock()
	instances := make([]*ModelInstance, 0, len(m.instances))
	for _, instance := range m.instances {
		instances = append(instances, instance)
	}
	m.mutex.RUnlock()

	for _, instance := range instances {
		go m.checkInstanceHealth(ctx, instance)
	}
}

// checkInstanceHealth checks health of a single instance
func (m *MultiInstanceManager) checkInstanceHealth(ctx context.Context, instance *ModelInstance) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status == StatusDisabled || instance.Status == StatusStopped {
		return
	}

	client := &http.Client{Timeout: instance.Config.HealthTimeout}
	healthURL := fmt.Sprintf("%s/health", instance.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		m.markInstanceUnhealthy(instance, fmt.Sprintf("failed to create health check request: %v", err))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		m.markInstanceUnhealthy(instance, fmt.Sprintf("health check failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		instance.Status = StatusHealthy
		instance.LastHealthy = time.Now()
	} else {
		m.markInstanceUnhealthy(instance, fmt.Sprintf("health check returned status %d", resp.StatusCode))
	}
}

// markInstanceUnhealthy marks an instance as unhealthy and potentially restarts it
func (m *MultiInstanceManager) markInstanceUnhealthy(instance *ModelInstance, reason string) {
	instance.Status = StatusUnhealthy
	instance.Stats.ErrorCount++

	m.logger.WithFields(logrus.Fields{
		"instance_id": instance.ID,
		"reason":      reason,
		"error_count": instance.Stats.ErrorCount,
	}).Warn("⚠️ Instance marked as unhealthy")

	// Auto-restart logic
	if instance.RestartCount < instance.Config.MaxRestarts {
		m.logger.WithField("instance_id", instance.ID).Info("🔄 Attempting to restart unhealthy instance")
		go m.restartInstance(context.Background(), instance)
	} else {
		m.logger.WithField("instance_id", instance.ID).Error("❌ Max restarts exceeded, disabling instance")
		instance.Status = StatusDisabled
	}
}

// restartInstance restarts a failed instance
func (m *MultiInstanceManager) restartInstance(ctx context.Context, instance *ModelInstance) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	// Stop existing process
	if instance.Process != nil {
		instance.Process.Process.Kill()
		instance.Process.Wait()
	}

	// Wait for restart delay
	time.Sleep(instance.Config.RestartDelay)

	// Increment restart count
	instance.RestartCount++

	// Restart the instance
	if err := m.startInstance(ctx, instance); err != nil {
		m.logger.WithError(err).WithField("instance_id", instance.ID).Error("❌ Failed to restart instance")
		instance.Status = StatusCrashed
		return
	}

	m.logger.WithFields(logrus.Fields{
		"instance_id":   instance.ID,
		"restart_count": instance.RestartCount,
	}).Info("✅ Instance restarted successfully")
}

// Port Manager methods

// AllocatePort allocates a port for a model instance
func (pm *PortManager) AllocatePort(modelName string) (int, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for i := 0; i < pm.maxPorts; i++ {
		port := pm.basePort + i
		if _, allocated := pm.allocatedPorts[port]; !allocated {
			pm.allocatedPorts[port] = modelName
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports (base: %d, max: %d)", pm.basePort, pm.maxPorts)
}

// ReleasePort releases a port allocation
func (pm *PortManager) ReleasePort(port int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	delete(pm.allocatedPorts, port)
}

// Helper methods for manager

// getRunningInstances returns all running instances
func (m *MultiInstanceManager) getRunningInstances() []*ModelInstance {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var running []*ModelInstance
	for _, instance := range m.instances {
		if instance.Status == StatusRunning || instance.Status == StatusHealthy {
			running = append(running, instance)
		}
	}
	return running
}

// countEnabledModels counts enabled models
func (m *MultiInstanceManager) countEnabledModels() int {
	count := 0
	for _, enabled := range m.enabledModels {
		if enabled {
			count++
		}
	}
	return count
}

// Shutdown gracefully shuts down all instances
func (m *MultiInstanceManager) Shutdown(ctx context.Context) error {
	m.logger.Info("🛑 Shutting down Multi-Instance Model Manager")

	close(m.shutdownChan)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, instance := range m.instances {
		if instance.Process != nil {
			instance.Process.Process.Kill()
			instance.Process.Wait()
		}
		m.portManager.ReleasePort(instance.Port)
	}

	m.logger.Info("✅ Multi-Instance Model Manager shutdown complete")
	return nil
}

// GetInstanceStats returns statistics for all instances
func (m *MultiInstanceManager) GetInstanceStats() map[string]InstanceStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]InstanceStats)
	for name, instance := range m.instances {
		instance.mutex.RLock()
		stats[name] = instance.Stats
		instance.mutex.RUnlock()
	}
	return stats
}

// GetManagerStatus returns overall manager status
func (m *MultiInstanceManager) GetManagerStatus() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	runningInstances := m.getRunningInstances()

	return map[string]interface{}{
		"total_instances":   len(m.instances),
		"running_instances": len(runningInstances),
		"enabled_models":    m.enabledModels,
		"max_instances":     m.maxInstances,
		"base_port":         m.basePort,
		"resource_usage": map[string]interface{}{
			"memory_mb":      m.resourceMonitor.currentMemoryMB,
			"memory_limit":   m.resourceMonitor.maxMemoryMB,
			"memory_percent": (m.resourceMonitor.currentMemoryMB * 100) / m.resourceMonitor.maxMemoryMB,
		},
		"port_allocations": m.portManager.allocatedPorts,
	}
}

// RestartModel restarts a specific model instance by name
func (m *MultiInstanceManager) RestartModel(ctx context.Context, modelName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Find the instance for this model
	instance, exists := m.instances[modelName]
	if !exists {
		return fmt.Errorf("model instance not found: %s", modelName)
	}

	m.logger.WithFields(logrus.Fields{
		"model_name":     modelName,
		"instance_id":    instance.ID,
		"current_status": instance.Status,
	}).Info("🔄 Restarting model instance")

	// Restart the instance
	m.restartInstance(ctx, instance)

	return nil
}
