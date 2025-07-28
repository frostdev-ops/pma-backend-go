package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func BenchmarkEntitySync(b *testing.B) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise during benchmarks
	cfg := &config.Config{}           // Minimal config for testing
	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	// Create adapter with many entities
	adapter := NewLargeMockAdapter(1000) // 1000 entities
	err := unifiedService.RegisterAdapter(adapter)
	if err != nil {
		b.Fatal(err)
	}

	err = adapter.Connect(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark sync operation
	for i := 0; i < b.N; i++ {
		_, err := unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConflictResolution(b *testing.B) {
	// Setup with multiple adapters having overlapping entities
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing
	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	// Create multiple adapters with same entities
	adapter1 := NewLargeMockAdapter(500)
	adapter1.id = "adapter1"
	adapter1.sourceType = types.SourceHomeAssistant

	adapter2 := NewLargeMockAdapter(500)
	adapter2.id = "adapter2"
	adapter2.sourceType = types.SourceShelly

	err := unifiedService.RegisterAdapter(adapter1)
	if err != nil {
		b.Fatal(err)
	}

	err = unifiedService.RegisterAdapter(adapter2)
	if err != nil {
		b.Fatal(err)
	}

	err = adapter1.Connect(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	err = adapter2.Connect(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark conflict resolution during sync
	for i := 0; i < b.N; i++ {
		_, err := unifiedService.SyncFromAllSources(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEntityRetrieval(b *testing.B) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing
	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	adapter := NewLargeMockAdapter(10000) // Large number of entities
	err := unifiedService.RegisterAdapter(adapter)
	if err != nil {
		b.Fatal(err)
	}

	err = adapter.Connect(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	// Initial sync
	_, err = unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark entity retrieval
	for i := 0; i < b.N; i++ {
		options := unified.GetAllOptions{
			IncludeRoom: true,
			IncludeArea: true,
		}
		_, err := unifiedService.GetAll(context.Background(), options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkActionExecution(b *testing.B) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing
	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	adapter := NewMockPMAAdapter()
	err := unifiedService.RegisterAdapter(adapter)
	if err != nil {
		b.Fatal(err)
	}

	err = adapter.Connect(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	// Sync entities
	_, err = unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
	if err != nil {
		b.Fatal(err)
	}

	action := types.PMAControlAction{
		EntityID: "ha_light.test_light",
		Action:   "turn_on",
		Parameters: map[string]interface{}{
			"brightness": 100,
		},
	}

	b.ResetTimer()

	// Benchmark action execution
	for i := 0; i < b.N; i++ {
		_, err := unifiedService.ExecuteAction(context.Background(), action)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing
	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	adapter := NewMockPMAAdapter()
	err := unifiedService.RegisterAdapter(adapter)
	if err != nil {
		b.Fatal(err)
	}

	err = adapter.Connect(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	// Sync entities
	_, err = unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// Benchmark concurrent operations
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of operations
			switch b.N % 3 {
			case 0:
				// Get all entities
				options := unified.GetAllOptions{}
				_, err := unifiedService.GetAll(context.Background(), options)
				if err != nil {
					b.Error(err)
				}
			case 1:
				// Get specific entity
				_, err := unifiedService.GetByID(context.Background(), "ha_light.test_light", unified.GetEntityOptions{})
				if err != nil {
					b.Error(err)
				}
			case 2:
				// Execute action
				action := types.PMAControlAction{
					EntityID: "ha_light.test_light",
					Action:   "turn_on",
				}
				_, err := unifiedService.ExecuteAction(context.Background(), action)
				if err != nil {
					b.Error(err)
				}
			}
		}
	})
}

func TestPerformanceScaling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing

	testCases := []struct {
		name        string
		entityCount int
		maxDuration time.Duration
	}{
		{"Small scale (100 entities)", 100, 100 * time.Millisecond},
		{"Medium scale (1000 entities)", 1000, 500 * time.Millisecond},
		{"Large scale (5000 entities)", 5000, 2 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			typeRegistry := types.NewPMATypeRegistry(logger)
			unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

			adapter := NewLargeMockAdapter(tc.entityCount)
			err := unifiedService.RegisterAdapter(adapter)
			assert.NoError(t, err)

			err = adapter.Connect(context.Background())
			assert.NoError(t, err)

			// Measure sync time
			start := time.Now()
			_, err = unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
			syncDuration := time.Since(start)

			assert.NoError(t, err)
			assert.Less(t, syncDuration, tc.maxDuration,
				"Sync took too long for %d entities: %v", tc.entityCount, syncDuration)

			// Measure retrieval time
			start = time.Now()
			options := unified.GetAllOptions{}
			entities, err := unifiedService.GetAll(context.Background(), options)
			retrievalDuration := time.Since(start)

			assert.NoError(t, err)
			assert.Equal(t, tc.entityCount, len(entities))
			assert.Less(t, retrievalDuration, tc.maxDuration/2,
				"Retrieval took too long for %d entities: %v", tc.entityCount, retrievalDuration)

			t.Logf("Scale test %s: sync=%v, retrieval=%v", tc.name, syncDuration, retrievalDuration)
		})
	}
}

func TestMemoryUsage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing

	// Test with increasing entity counts to check for memory leaks
	entityCounts := []int{100, 500, 1000}

	for _, count := range entityCounts {
		t.Run(fmt.Sprintf("Memory test with %d entities", count), func(t *testing.T) {
			typeRegistry := types.NewPMATypeRegistry(logger)
			unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

			adapter := NewLargeMockAdapter(count)
			err := unifiedService.RegisterAdapter(adapter)
			assert.NoError(t, err)

			err = adapter.Connect(context.Background())
			assert.NoError(t, err)

			// Sync multiple times to check for memory leaks
			for i := 0; i < 10; i++ {
				_, err = unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
				assert.NoError(t, err)
			}

			// Get entities multiple times
			for i := 0; i < 10; i++ {
				options := unified.GetAllOptions{}
				entities, err := unifiedService.GetAll(context.Background(), options)
				assert.NoError(t, err)
				assert.Equal(t, count, len(entities))
			}

			// Clean up
			err = adapter.Disconnect(context.Background())
			assert.NoError(t, err)
		})
	}
}

func TestAdapterHealthUnderLoad(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	cfg := &config.Config{} // Minimal config for testing
	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	adapter := NewMockPMAAdapter()
	err := unifiedService.RegisterAdapter(adapter)
	assert.NoError(t, err)

	err = adapter.Connect(context.Background())
	assert.NoError(t, err)

	// Sync entities
	_, err = unifiedService.SyncFromSource(context.Background(), types.SourceHomeAssistant)
	assert.NoError(t, err)

	// Execute many actions concurrently
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(actionNum int) {
			defer func() { done <- true }()

			action := types.PMAControlAction{
				EntityID: "ha_light.test_light",
				Action:   "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 50 + actionNum%50,
				},
			}

			_, err := unifiedService.ExecuteAction(context.Background(), action)
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all actions to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Check adapter health after load
	health := adapter.GetHealth()
	assert.True(t, health.IsHealthy)
	assert.Empty(t, health.Issues)

	// Check metrics
	metrics := adapter.GetMetrics()
	assert.Equal(t, int64(100), metrics.ActionsExecuted)
	assert.Equal(t, int64(100), metrics.SuccessfulActions)
	assert.Equal(t, int64(0), metrics.FailedActions)
}
