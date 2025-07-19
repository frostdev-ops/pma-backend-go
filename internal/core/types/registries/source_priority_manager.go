package registries

import (
	"fmt"
	"sort"
	"sync"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// Custom errors for source priority manager
var (
	ErrInvalidPriority = fmt.Errorf("invalid priority value")
	ErrUnknownSource   = fmt.Errorf("unknown source type")
)

// SourcePriority represents a source and its priority
type SourcePriority struct {
	Source   types.PMASourceType
	Priority int
}

// DefaultSourcePriorityManager implements the SourcePriorityManager interface
type DefaultSourcePriorityManager struct {
	priorities map[types.PMASourceType]int // source -> priority (lower number = higher priority)
	mutex      sync.RWMutex
	logger     *logrus.Logger
}

// NewDefaultSourcePriorityManager creates a new source priority manager with default priorities
func NewDefaultSourcePriorityManager(logger *logrus.Logger) *DefaultSourcePriorityManager {
	manager := &DefaultSourcePriorityManager{
		priorities: make(map[types.PMASourceType]int),
		logger:     logger,
	}

	// Set default priorities (lower number = higher priority)
	defaultPriorities := map[types.PMASourceType]int{
		types.SourceHomeAssistant: 1,  // Highest priority - most comprehensive
		types.SourceRing:          2,  // Security devices
		types.SourceShelly:        3,  // Smart switches/devices
		types.SourceUPS:           4,  // Power management
		types.SourceNetwork:       5,  // Network devices
		types.SourcePMA:           10, // Virtual/computed entities
	}

	for source, priority := range defaultPriorities {
		manager.priorities[source] = priority
	}

	return manager
}

// SetSourcePriority sets the priority for a specific source
func (m *DefaultSourcePriorityManager) SetSourcePriority(source types.PMASourceType, priority int) error {
	if priority < 0 {
		return fmt.Errorf("%w: priority must be non-negative, got %d", ErrInvalidPriority, priority)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	oldPriority, existed := m.priorities[source]
	m.priorities[source] = priority

	if existed {
		m.logger.Infof("Updated priority for source %s: %d -> %d", source, oldPriority, priority)
	} else {
		m.logger.Infof("Set priority for source %s: %d", source, priority)
	}

	return nil
}

// GetSourcePriority returns the priority for a specific source
func (m *DefaultSourcePriorityManager) GetSourcePriority(source types.PMASourceType) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if priority, exists := m.priorities[source]; exists {
		return priority
	}

	// Return a default high priority number for unknown sources
	// This ensures they have lower priority than known sources
	return 1000
}

// GetPriorityOrder returns all sources ordered by priority (highest priority first)
func (m *DefaultSourcePriorityManager) GetPriorityOrder() []types.PMASourceType {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a slice of SourcePriority structs for sorting
	sourcePriorities := make([]SourcePriority, 0, len(m.priorities))
	for source, priority := range m.priorities {
		sourcePriorities = append(sourcePriorities, SourcePriority{
			Source:   source,
			Priority: priority,
		})
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(sourcePriorities, func(i, j int) bool {
		return sourcePriorities[i].Priority < sourcePriorities[j].Priority
	})

	// Extract just the sources in priority order
	sources := make([]types.PMASourceType, len(sourcePriorities))
	for i, sp := range sourcePriorities {
		sources[i] = sp.Source
	}

	return sources
}

// ShouldOverride determines if a new source should override the current source
func (m *DefaultSourcePriorityManager) ShouldOverride(currentSource, newSource types.PMASourceType) bool {
	currentPriority := m.GetSourcePriority(currentSource)
	newPriority := m.GetSourcePriority(newSource)

	// Lower priority number means higher priority
	// Override if new source has higher priority (lower number)
	return newPriority < currentPriority
}

// GetHighestPrioritySource returns the source with the highest priority from a list
func (m *DefaultSourcePriorityManager) GetHighestPrioritySource(sources []types.PMASourceType) types.PMASourceType {
	if len(sources) == 0 {
		return ""
	}

	if len(sources) == 1 {
		return sources[0]
	}

	highestPriority := m.GetSourcePriority(sources[0])
	highestSource := sources[0]

	for _, source := range sources[1:] {
		priority := m.GetSourcePriority(source)
		if priority < highestPriority { // Lower number = higher priority
			highestPriority = priority
			highestSource = source
		}
	}

	return highestSource
}

// CompareSources compares two sources and returns:
// -1 if source1 has higher priority than source2
//
//	0 if they have equal priority
//	1 if source2 has higher priority than source1
func (m *DefaultSourcePriorityManager) CompareSources(source1, source2 types.PMASourceType) int {
	priority1 := m.GetSourcePriority(source1)
	priority2 := m.GetSourcePriority(source2)

	if priority1 < priority2 {
		return -1 // source1 has higher priority
	} else if priority1 > priority2 {
		return 1 // source2 has higher priority
	}
	return 0 // equal priority
}

// ResetToDefaults resets all priorities to their default values
func (m *DefaultSourcePriorityManager) ResetToDefaults() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	defaultPriorities := map[types.PMASourceType]int{
		types.SourceHomeAssistant: 1,
		types.SourceRing:          2,
		types.SourceShelly:        3,
		types.SourceUPS:           4,
		types.SourceNetwork:       5,
		types.SourcePMA:           10,
	}

	m.priorities = make(map[types.PMASourceType]int)
	for source, priority := range defaultPriorities {
		m.priorities[source] = priority
	}

	m.logger.Info("Reset source priorities to default values")
}

// GetAllPriorities returns a copy of all current priority settings
func (m *DefaultSourcePriorityManager) GetAllPriorities() map[types.PMASourceType]int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	priorities := make(map[types.PMASourceType]int)
	for source, priority := range m.priorities {
		priorities[source] = priority
	}

	return priorities
}

// SetMultiplePriorities sets priorities for multiple sources at once
func (m *DefaultSourcePriorityManager) SetMultiplePriorities(priorities map[types.PMASourceType]int) error {
	// Validate all priorities first
	for source, priority := range priorities {
		if priority < 0 {
			return fmt.Errorf("%w: priority for source %s must be non-negative, got %d",
				ErrInvalidPriority, source, priority)
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Apply all changes
	for source, priority := range priorities {
		oldPriority, existed := m.priorities[source]
		m.priorities[source] = priority

		if existed {
			m.logger.Debugf("Updated priority for source %s: %d -> %d", source, oldPriority, priority)
		} else {
			m.logger.Debugf("Set priority for source %s: %d", source, priority)
		}
	}

	m.logger.Infof("Updated priorities for %d sources", len(priorities))

	return nil
}

// IsValidSource checks if a source type is recognized
func (m *DefaultSourcePriorityManager) IsValidSource(source types.PMASourceType) bool {
	validSources := map[types.PMASourceType]bool{
		types.SourceHomeAssistant: true,
		types.SourceRing:          true,
		types.SourceShelly:        true,
		types.SourceUPS:           true,
		types.SourceNetwork:       true,
		types.SourcePMA:           true,
	}

	return validSources[source]
}

// GetSourcesInPriorityRange returns sources within a priority range (inclusive)
func (m *DefaultSourcePriorityManager) GetSourcesInPriorityRange(minPriority, maxPriority int) []types.PMASourceType {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var sources []types.PMASourceType
	for source, priority := range m.priorities {
		if priority >= minPriority && priority <= maxPriority {
			sources = append(sources, source)
		}
	}

	return sources
}
