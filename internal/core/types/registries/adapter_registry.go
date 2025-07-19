package registries

import (
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// Custom errors for adapter registry
var (
	ErrAdapterNotFound          = fmt.Errorf("adapter not found")
	ErrAdapterAlreadyRegistered = fmt.Errorf("adapter already registered")
	ErrInvalidAdapter           = fmt.Errorf("invalid adapter")
)

// DefaultAdapterRegistry implements the AdapterRegistry interface
type DefaultAdapterRegistry struct {
	adapters         map[string]types.PMAAdapter
	adaptersBySource map[types.PMASourceType]types.PMAAdapter
	metrics          map[string]*types.AdapterMetrics
	mutex            sync.RWMutex
	logger           *logrus.Logger
}

// NewDefaultAdapterRegistry creates a new adapter registry
func NewDefaultAdapterRegistry(logger *logrus.Logger) *DefaultAdapterRegistry {
	return &DefaultAdapterRegistry{
		adapters:         make(map[string]types.PMAAdapter),
		adaptersBySource: make(map[types.PMASourceType]types.PMAAdapter),
		metrics:          make(map[string]*types.AdapterMetrics),
		logger:           logger,
	}
}

// RegisterAdapter registers a new adapter in the registry
func (r *DefaultAdapterRegistry) RegisterAdapter(adapter types.PMAAdapter) error {
	if adapter == nil {
		return ErrInvalidAdapter
	}

	adapterID := adapter.GetID()
	sourceType := adapter.GetSourceType()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if adapter ID already exists
	if _, exists := r.adapters[adapterID]; exists {
		return fmt.Errorf("%w: adapter ID '%s'", ErrAdapterAlreadyRegistered, adapterID)
	}

	// Check if source type already has an adapter
	if existingAdapter, exists := r.adaptersBySource[sourceType]; exists {
		r.logger.Warnf("Replacing existing adapter for source %s (old: %s, new: %s)",
			sourceType, existingAdapter.GetID(), adapterID)
		// Remove old adapter
		delete(r.adapters, existingAdapter.GetID())
		delete(r.metrics, existingAdapter.GetID())
	}

	// Register the adapter
	r.adapters[adapterID] = adapter
	r.adaptersBySource[sourceType] = adapter

	// Initialize metrics
	r.metrics[adapterID] = &types.AdapterMetrics{
		EntitiesManaged:     0,
		RoomsManaged:        0,
		ActionsExecuted:     0,
		SuccessfulActions:   0,
		FailedActions:       0,
		AverageResponseTime: 0,
		LastSync:            nil,
		SyncErrors:          0,
		Uptime:              0,
	}

	r.logger.Infof("Registered adapter: %s (source: %s, version: %s)",
		adapterID, sourceType, adapter.GetVersion())

	return nil
}

// UnregisterAdapter removes an adapter from the registry
func (r *DefaultAdapterRegistry) UnregisterAdapter(adapterID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	adapter, exists := r.adapters[adapterID]
	if !exists {
		return fmt.Errorf("%w: adapter ID '%s'", ErrAdapterNotFound, adapterID)
	}

	// Remove from maps
	delete(r.adapters, adapterID)
	delete(r.adaptersBySource, adapter.GetSourceType())
	delete(r.metrics, adapterID)

	r.logger.Infof("Unregistered adapter: %s", adapterID)

	return nil
}

// GetAdapter retrieves an adapter by its ID
func (r *DefaultAdapterRegistry) GetAdapter(adapterID string) (types.PMAAdapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.adapters[adapterID]
	if !exists {
		return nil, fmt.Errorf("%w: adapter ID '%s'", ErrAdapterNotFound, adapterID)
	}

	return adapter, nil
}

// GetAdapterBySource retrieves an adapter by its source type
func (r *DefaultAdapterRegistry) GetAdapterBySource(sourceType types.PMASourceType) (types.PMAAdapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.adaptersBySource[sourceType]
	if !exists {
		return nil, fmt.Errorf("%w: source type '%s'", ErrAdapterNotFound, sourceType)
	}

	return adapter, nil
}

// GetAllAdapters returns all registered adapters
func (r *DefaultAdapterRegistry) GetAllAdapters() []types.PMAAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapters := make([]types.PMAAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}

	return adapters
}

// GetConnectedAdapters returns only the connected adapters
func (r *DefaultAdapterRegistry) GetConnectedAdapters() []types.PMAAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapters := make([]types.PMAAdapter, 0)
	for _, adapter := range r.adapters {
		if adapter.IsConnected() {
			adapters = append(adapters, adapter)
		}
	}

	return adapters
}

// GetAdapterMetrics retrieves metrics for a specific adapter
func (r *DefaultAdapterRegistry) GetAdapterMetrics(adapterID string) (*types.AdapterMetrics, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check if adapter exists
	adapter, exists := r.adapters[adapterID]
	if !exists {
		return nil, fmt.Errorf("%w: adapter ID '%s'", ErrAdapterNotFound, adapterID)
	}

	// Try to get live metrics from adapter first
	if liveMetrics := adapter.GetMetrics(); liveMetrics != nil {
		// Update our stored metrics with live data
		r.mutex.RUnlock()
		r.mutex.Lock()
		r.metrics[adapterID] = liveMetrics
		r.mutex.Unlock()
		r.mutex.RLock()
		return liveMetrics, nil
	}

	// Fall back to stored metrics
	metrics, exists := r.metrics[adapterID]
	if !exists {
		// Initialize default metrics if not found
		r.mutex.RUnlock()
		r.mutex.Lock()
		r.metrics[adapterID] = &types.AdapterMetrics{
			EntitiesManaged:     0,
			RoomsManaged:        0,
			ActionsExecuted:     0,
			SuccessfulActions:   0,
			FailedActions:       0,
			AverageResponseTime: 0,
			LastSync:            nil,
			SyncErrors:          0,
			Uptime:              0,
		}
		metrics = r.metrics[adapterID]
		r.mutex.Unlock()
		r.mutex.RLock()
	}

	return metrics, nil
}

// UpdateAdapterMetrics updates the stored metrics for an adapter
func (r *DefaultAdapterRegistry) UpdateAdapterMetrics(adapterID string, metrics *types.AdapterMetrics) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.adapters[adapterID]; !exists {
		return fmt.Errorf("%w: adapter ID '%s'", ErrAdapterNotFound, adapterID)
	}

	r.metrics[adapterID] = metrics
	return nil
}

// GetRegistryStats returns statistics about the registry
func (r *DefaultAdapterRegistry) GetRegistryStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	connected := 0
	for _, adapter := range r.adapters {
		if adapter.IsConnected() {
			connected++
		}
	}

	return map[string]interface{}{
		"total_adapters":     len(r.adapters),
		"connected_adapters": connected,
		"source_types":       len(r.adaptersBySource),
		"last_updated":       time.Now(),
	}
}
