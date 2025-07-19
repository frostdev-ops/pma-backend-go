package registries

import (
	"testing"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultSourcePriorityManager(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	assert.NotNil(t, manager)

	// Test default priorities
	assert.Equal(t, 1, manager.GetSourcePriority(types.SourceHomeAssistant))
	assert.Equal(t, 2, manager.GetSourcePriority(types.SourceRing))
	assert.Equal(t, 3, manager.GetSourcePriority(types.SourceShelly))
	assert.Equal(t, 4, manager.GetSourcePriority(types.SourceUPS))
	assert.Equal(t, 5, manager.GetSourcePriority(types.SourceNetwork))
	assert.Equal(t, 10, manager.GetSourcePriority(types.SourcePMA))
}

func TestSourcePriorityManager_SetSourcePriority(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// Test setting valid priority
	err := manager.SetSourcePriority(types.SourceHomeAssistant, 100)
	assert.NoError(t, err)
	assert.Equal(t, 100, manager.GetSourcePriority(types.SourceHomeAssistant))

	// Test setting invalid priority
	err = manager.SetSourcePriority(types.SourceHomeAssistant, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "priority must be non-negative")
}

func TestSourcePriorityManager_GetPriorityOrder(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	order := manager.GetPriorityOrder()
	assert.NotEmpty(t, order)

	// Check that HomeAssistant (priority 1) comes first
	assert.Equal(t, types.SourceHomeAssistant, order[0])

	// Check that PMA (priority 10) comes last
	assert.Equal(t, types.SourcePMA, order[len(order)-1])

	// Verify order is correct (ascending priority)
	for i := 1; i < len(order); i++ {
		priority1 := manager.GetSourcePriority(order[i-1])
		priority2 := manager.GetSourcePriority(order[i])
		assert.True(t, priority1 <= priority2, "Priority order should be ascending")
	}
}

func TestSourcePriorityManager_ShouldOverride(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// HomeAssistant (priority 1) should override Ring (priority 2)
	assert.True(t, manager.ShouldOverride(types.SourceRing, types.SourceHomeAssistant))

	// Ring (priority 2) should not override HomeAssistant (priority 1)
	assert.False(t, manager.ShouldOverride(types.SourceHomeAssistant, types.SourceRing))

	// Same source should not override itself
	assert.False(t, manager.ShouldOverride(types.SourceHomeAssistant, types.SourceHomeAssistant))
}

func TestSourcePriorityManager_GetHighestPrioritySource(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// Test empty slice
	result := manager.GetHighestPrioritySource([]types.PMASourceType{})
	assert.Equal(t, types.PMASourceType(""), result)

	// Test single source
	result = manager.GetHighestPrioritySource([]types.PMASourceType{types.SourceRing})
	assert.Equal(t, types.SourceRing, result)

	// Test multiple sources
	sources := []types.PMASourceType{
		types.SourcePMA,           // priority 10
		types.SourceRing,          // priority 2
		types.SourceShelly,        // priority 3
		types.SourceHomeAssistant, // priority 1 (highest)
	}
	result = manager.GetHighestPrioritySource(sources)
	assert.Equal(t, types.SourceHomeAssistant, result)
}

func TestSourcePriorityManager_CompareSources(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// HomeAssistant (priority 1) should have higher priority than Ring (priority 2)
	result := manager.CompareSources(types.SourceHomeAssistant, types.SourceRing)
	assert.Equal(t, -1, result)

	// Ring (priority 2) should have lower priority than HomeAssistant (priority 1)
	result = manager.CompareSources(types.SourceRing, types.SourceHomeAssistant)
	assert.Equal(t, 1, result)

	// Same source should have equal priority
	result = manager.CompareSources(types.SourceHomeAssistant, types.SourceHomeAssistant)
	assert.Equal(t, 0, result)
}

func TestSourcePriorityManager_ResetToDefaults(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// Change a priority
	err := manager.SetSourcePriority(types.SourceHomeAssistant, 100)
	assert.NoError(t, err)
	assert.Equal(t, 100, manager.GetSourcePriority(types.SourceHomeAssistant))

	// Reset to defaults
	manager.ResetToDefaults()
	assert.Equal(t, 1, manager.GetSourcePriority(types.SourceHomeAssistant))
}

func TestSourcePriorityManager_SetMultiplePriorities(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	priorities := map[types.PMASourceType]int{
		types.SourceHomeAssistant: 50,
		types.SourceRing:          60,
		types.SourceShelly:        70,
	}

	err := manager.SetMultiplePriorities(priorities)
	assert.NoError(t, err)

	assert.Equal(t, 50, manager.GetSourcePriority(types.SourceHomeAssistant))
	assert.Equal(t, 60, manager.GetSourcePriority(types.SourceRing))
	assert.Equal(t, 70, manager.GetSourcePriority(types.SourceShelly))

	// Test with invalid priority
	invalidPriorities := map[types.PMASourceType]int{
		types.SourceHomeAssistant: -1,
	}
	err = manager.SetMultiplePriorities(invalidPriorities)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "priority")
}

func TestSourcePriorityManager_IsValidSource(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// Test valid sources
	assert.True(t, manager.IsValidSource(types.SourceHomeAssistant))
	assert.True(t, manager.IsValidSource(types.SourceRing))
	assert.True(t, manager.IsValidSource(types.SourceShelly))
	assert.True(t, manager.IsValidSource(types.SourceUPS))
	assert.True(t, manager.IsValidSource(types.SourceNetwork))
	assert.True(t, manager.IsValidSource(types.SourcePMA))

	// Test invalid source
	assert.False(t, manager.IsValidSource(types.PMASourceType("invalid")))
}

func TestSourcePriorityManager_GetSourcesInPriorityRange(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// Get sources with priority 1-5
	sources := manager.GetSourcesInPriorityRange(1, 5)
	assert.Contains(t, sources, types.SourceHomeAssistant) // priority 1
	assert.Contains(t, sources, types.SourceRing)          // priority 2
	assert.Contains(t, sources, types.SourceShelly)        // priority 3
	assert.Contains(t, sources, types.SourceUPS)           // priority 4
	assert.Contains(t, sources, types.SourceNetwork)       // priority 5
	assert.NotContains(t, sources, types.SourcePMA)        // priority 10 (out of range)

	// Get sources with priority 10-20
	sources = manager.GetSourcesInPriorityRange(10, 20)
	assert.Contains(t, sources, types.SourcePMA)              // priority 10
	assert.NotContains(t, sources, types.SourceHomeAssistant) // priority 1 (out of range)
}

func TestSourcePriorityManager_UnknownSource(t *testing.T) {
	logger := logrus.New()
	manager := NewDefaultSourcePriorityManager(logger)

	// Unknown source should get default high priority
	unknownSource := types.PMASourceType("unknown")
	priority := manager.GetSourcePriority(unknownSource)
	assert.Equal(t, 1000, priority)

	// Known source should override unknown source
	assert.True(t, manager.ShouldOverride(unknownSource, types.SourceHomeAssistant))
}
