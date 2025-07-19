package homeassistant

import (
	"context"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// ExampleIntegration demonstrates how to integrate the HomeAssistant adapter
// with the PMA system. This file serves as documentation and example code.

// IntegrateHomeAssistantAdapter shows how to register and use the HA adapter
func IntegrateHomeAssistantAdapter(registry types.AdapterRegistry, config *config.Config, logger *logrus.Logger) error {
	// Create the HomeAssistant adapter
	adapter := NewHomeAssistantAdapter(config, logger)

	// Register the adapter with the registry
	if err := registry.RegisterAdapter(adapter); err != nil {
		return fmt.Errorf("failed to register HomeAssistant adapter: %w", err)
	}

	logger.Info("HomeAssistant adapter successfully registered")
	return nil
}

// ExampleUsage demonstrates typical usage patterns
func ExampleUsage(registry types.AdapterRegistry, logger *logrus.Logger) error {
	ctx := context.Background()

	// Get the HomeAssistant adapter
	adapter, err := registry.GetAdapterBySource(types.SourceHomeAssistant)
	if err != nil {
		return fmt.Errorf("HomeAssistant adapter not found: %w", err)
	}

	// Connect to HomeAssistant
	if err := adapter.Connect(ctx); err != nil {
		logger.WithError(err).Error("Failed to connect to HomeAssistant")
		return err
	}
	defer adapter.Disconnect(ctx)

	logger.Info("Connected to HomeAssistant successfully")

	// Sync entities
	entities, err := adapter.SyncEntities(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to sync entities")
		return err
	}

	logger.WithField("count", len(entities)).Info("Synced entities from HomeAssistant")

	// Sync rooms
	rooms, err := adapter.SyncRooms(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to sync rooms")
		return err
	}

	logger.WithField("count", len(rooms)).Info("Synced rooms from HomeAssistant")

	// Example: Execute an action on a light
	lightAction := types.PMAControlAction{
		EntityID: "ha_light.living_room",
		Action:   "turn_on",
		Parameters: map[string]interface{}{
			"brightness": 0.8, // 80% brightness
			"color": map[string]interface{}{
				"r": 255.0,
				"g": 200.0,
				"b": 100.0,
			},
		},
		Context: &types.PMAContext{
			Source:      "example_integration",
			Description: "Example light control",
		},
	}

	result, err := adapter.ExecuteAction(ctx, lightAction)
	if err != nil {
		logger.WithError(err).Error("Failed to execute action")
		return err
	}

	if result.Success {
		logger.Info("Successfully turned on living room light")
	} else {
		logger.WithField("error", result.Error).Warn("Action failed")
	}

	// Get adapter health and metrics
	health := adapter.GetHealth()
	metrics := adapter.GetMetrics()

	logger.WithFields(logrus.Fields{
		"healthy":          health.IsHealthy,
		"entities_managed": metrics.EntitiesManaged,
		"rooms_managed":    metrics.RoomsManaged,
		"actions_executed": metrics.ActionsExecuted,
		"success_rate":     float64(metrics.SuccessfulActions) / float64(metrics.ActionsExecuted),
	}).Info("HomeAssistant adapter status")

	return nil
}

// ExampleEntityProcessing shows how to process converted entities
func ExampleEntityProcessing(entities []types.PMAEntity, logger *logrus.Logger) {
	lightCount := 0
	sensorCount := 0
	unavailableCount := 0

	for _, entity := range entities {
		// Process by type
		switch entity.GetType() {
		case types.EntityTypeLight:
			lightCount++

			// Example: Check if light supports dimming
			if entity.HasCapability(types.CapabilityDimmable) {
				logger.WithField("entity_id", entity.GetID()).Debug("Light supports dimming")
			}

		case types.EntityTypeSensor:
			sensorCount++

			// Example: Process temperature sensors
			if entity.HasCapability(types.CapabilityTemperature) {
				attributes := entity.GetAttributes()
				if temp, ok := attributes["numeric_value"].(float64); ok {
					logger.WithFields(logrus.Fields{
						"entity_id":   entity.GetID(),
						"temperature": temp,
						"unit":        attributes["unit_of_measurement"],
					}).Debug("Temperature sensor reading")
				}
			}
		}

		// Check availability
		if !entity.IsAvailable() {
			unavailableCount++
		}

		// Check quality score
		if entity.GetQualityScore() < 0.5 {
			logger.WithFields(logrus.Fields{
				"entity_id":     entity.GetID(),
				"quality_score": entity.GetQualityScore(),
			}).Warn("Low quality entity detected")
		}
	}

	logger.WithFields(logrus.Fields{
		"total":       len(entities),
		"lights":      lightCount,
		"sensors":     sensorCount,
		"unavailable": unavailableCount,
	}).Info("Entity processing summary")
}

// ExampleRealtimeUpdates demonstrates how to handle real-time updates
// Note: This is a conceptual example - actual WebSocket implementation would be more complex
func ExampleRealtimeUpdates(adapter types.PMAAdapter, logger *logrus.Logger) {
	// Since HomeAssistant supports real-time updates
	if !adapter.SupportsRealtime() {
		logger.Warn("Adapter doesn't support real-time updates")
		return
	}

	logger.Info("Setting up real-time update handling (conceptual)")

	// In a real implementation, you would:
	// 1. Establish WebSocket connection to HomeAssistant
	// 2. Subscribe to state changes
	// 3. Convert incoming state changes to PMA format
	// 4. Update the entity registry
	// 5. Notify subscribers of changes

	// Example event handling loop (pseudo-code)
	/*
		for {
			select {
			case event := <-websocketEventChannel:
				// Convert HA event to PMA entity
				pmaEntity, err := convertHAEventToPMAEntity(event)
				if err != nil {
					logger.WithError(err).Error("Failed to convert HA event")
					continue
				}

				// Update entity registry
				entityRegistry.UpdateEntity(pmaEntity)

				// Notify subscribers
				notifyEntityChanged(pmaEntity)

			case <-ctx.Done():
				return
			}
		}
	*/
}

// ExampleErrorHandling shows robust error handling patterns
func ExampleErrorHandling(adapter types.PMAAdapter, logger *logrus.Logger) {
	ctx := context.Background()

	// Retry connection with backoff
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := adapter.Connect(ctx)
		if err == nil {
			logger.Info("Successfully connected to HomeAssistant")
			break
		}

		logger.WithFields(logrus.Fields{
			"attempt": attempt,
			"error":   err,
		}).Warn("Connection attempt failed")

		if attempt < maxRetries {
			delay := baseDelay * time.Duration(attempt)
			logger.WithField("delay", delay).Info("Retrying connection after delay")
			time.Sleep(delay)
		} else {
			logger.Error("All connection attempts failed")
			return
		}
	}

	// Graceful error handling for actions
	action := types.PMAControlAction{
		EntityID: "ha_light.nonexistent",
		Action:   "turn_on",
	}

	result, err := adapter.ExecuteAction(ctx, action)
	if err != nil {
		logger.WithError(err).Error("Action execution failed with error")
		return
	}

	if !result.Success {
		if result.Error != nil {
			switch result.Error.Code {
			case "ENTITY_NOT_FOUND":
				logger.Warn("Entity not found - may have been removed")
			case "EXECUTION_ERROR":
				if result.Error.Retryable {
					logger.Info("Retryable error - could retry action")
				} else {
					logger.Error("Non-retryable execution error")
				}
			default:
				logger.WithField("error_code", result.Error.Code).Error("Unknown error")
			}
		}
	}
}

// ConfigurationExample shows how to configure the HomeAssistant adapter
func ConfigurationExample() *config.Config {
	return &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://homeassistant.local:8123", // HomeAssistant instance URL
			Token: "your_long_lived_access_token",    // Long-lived access token
			Sync: config.HomeAssistantSync{
				Enabled:              true,
				FullSyncInterval:     "5m", // Full sync every 5 minutes
				SupportedDomains:     []string{"light", "switch", "sensor", "climate", "cover"},
				ConflictResolution:   "homeassistant_priority", // Prefer HA data in conflicts
				BatchSize:            100,                      // Process 100 entities at a time
				RetryAttempts:        3,                        // Retry failed operations 3 times
				RetryDelay:           "1s",                     // Wait 1 second between retries
				EventBufferSize:      1000,                     // Buffer up to 1000 events
				EventProcessingDelay: "100ms",                  // Process events every 100ms
			},
		},
	}
}
