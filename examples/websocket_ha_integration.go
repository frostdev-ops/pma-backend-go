package main

import (
	"log"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketHAIntegrationExample demonstrates how to set up and use
// the complete WebSocket Home Assistant integration
func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Parse batch window duration
	batchWindow, err := time.ParseDuration(cfg.WebSocket.HomeAssistant.BatchWindow)
	if err != nil {
		logger.Warnf("Invalid batch window duration, using default: %v", err)
		batchWindow = 100 * time.Millisecond
	}

	// Create WebSocket Hub
	wsHub := websocket.NewHub(logger)

	// Create HA Event Forwarder configuration
	forwarderConfig := &websocket.HAEventForwarderConfig{
		MaxEventsPerSecond:   cfg.WebSocket.HomeAssistant.MaxEventsPerSecond,
		BatchEvents:          cfg.WebSocket.HomeAssistant.BatchEvents,
		BatchWindow:          batchWindow,
		DefaultSubscriptions: cfg.WebSocket.HomeAssistant.DefaultSubscriptions,
		ForwardAllEntities:   cfg.WebSocket.HomeAssistant.ForwardAllEntities,
		MaxErrorsRetained:    cfg.WebSocket.HomeAssistant.MaxErrorsRetained,
	}

	// Create HA Event Forwarder
	haForwarder := websocket.NewHAEventForwarder(wsHub, logger, forwarderConfig)

	// Start the WebSocket Hub
	go wsHub.Run()
	logger.Info("WebSocket Hub started")

	// Start the HA Event Forwarder
	err = haForwarder.Start()
	if err != nil {
		log.Fatalf("Failed to start HA Event Forwarder: %v", err)
	}
	logger.Info("HA Event Forwarder started")

	// Example: Set up entity-to-room mapping
	entityRoomMap := map[string]string{
		"light.living_room_main":   "living_room",
		"light.living_room_accent": "living_room",
		"light.bedroom_ceiling":    "bedroom",
		"light.bedroom_bedside":    "bedroom",
		"switch.kitchen_main":      "kitchen",
		"sensor.living_room_temp":  "living_room",
		"binary_sensor.front_door": "entrance",
		"climate.living_room":      "living_room",
	}

	err = haForwarder.UpdateRoomFilters(entityRoomMap)
	if err != nil {
		logger.Errorf("Failed to update room filters: %v", err)
	} else {
		logger.Infof("Updated room filters for %d entities", len(entityRoomMap))
	}

	// Example: Configure which event types to forward
	haForwarder.SetEventTypeEnabled(websocket.MessageTypeHAStateChanged, true)
	haForwarder.SetEventTypeEnabled(websocket.MessageTypeHAEntityAdded, true)
	haForwarder.SetEventTypeEnabled(websocket.MessageTypeHAEntityRemoved, true)
	haForwarder.SetEventTypeEnabled(websocket.MessageTypeHASyncStatus, true)
	haForwarder.SetEventTypeEnabled(websocket.MessageTypeHAServiceCalled, false) // Disable service calls for this example

	logger.Info("Event type filtering configured")

	// Simulate some Home Assistant events to demonstrate the forwarding
	go simulateHAEvents(haForwarder, logger)

	// Display forwarding statistics periodically
	go displayStats(haForwarder, logger)

	// Run the example for 30 seconds
	logger.Info("WebSocket HA Integration example running for 30 seconds...")
	time.Sleep(30 * time.Second)

	// Cleanup
	logger.Info("Shutting down WebSocket HA Integration example...")
	err = haForwarder.Stop()
	if err != nil {
		logger.Errorf("Error stopping HA Event Forwarder: %v", err)
	}

	logger.Info("WebSocket HA Integration example completed")
}

// simulateHAEvents simulates various Home Assistant events to test the forwarding
func simulateHAEvents(forwarder *websocket.HAEventForwarder, logger *logrus.Logger) {
	// Wait a moment for setup
	time.Sleep(2 * time.Second)

	logger.Info("Starting to simulate HA events...")

	// Simulate connection status
	err := forwarder.ForwardSyncStatus("connected", "Successfully connected to Home Assistant", 42)
	if err != nil {
		logger.Errorf("Failed to forward sync status: %v", err)
	}

	// Simulate some state changes
	entities := []string{
		"light.living_room_main",
		"light.bedroom_ceiling",
		"sensor.living_room_temp",
		"switch.kitchen_main",
	}

	for i := 0; i < 10; i++ {
		for _, entityID := range entities {
			var oldState, newState string
			var attributes map[string]interface{}

			switch {
			case contains(entityID, "light"):
				oldState = "off"
				newState = "on"
				attributes = map[string]interface{}{
					"brightness":    255,
					"color_temp":    154,
					"friendly_name": "Living Room Light",
				}

			case contains(entityID, "sensor"):
				oldState = "20.5"
				newState = "21.2"
				attributes = map[string]interface{}{
					"unit_of_measurement": "°C",
					"device_class":        "temperature",
					"friendly_name":       "Living Room Temperature",
				}

			case contains(entityID, "switch"):
				oldState = "off"
				newState = "on"
				attributes = map[string]interface{}{
					"friendly_name": "Kitchen Main Switch",
				}
			}

			err := forwarder.ForwardStateChanged(entityID, oldState, newState, attributes)
			if err != nil {
				logger.Errorf("Failed to forward state change for %s: %v", entityID, err)
			}

			// Small delay between events
			time.Sleep(200 * time.Millisecond)
		}

		// Longer pause between rounds
		time.Sleep(2 * time.Second)
	}

	// Simulate entity being added
	newEntityData := map[string]interface{}{
		"entity_id":     "light.new_smart_bulb",
		"friendly_name": "New Smart Bulb",
		"state":         "off",
		"domain":        "light",
		"platform":      "hue",
	}

	err = forwarder.ForwardEntityAdded("light.new_smart_bulb", newEntityData)
	if err != nil {
		logger.Errorf("Failed to forward entity added: %v", err)
	}

	// Wait a bit, then simulate entity being removed
	time.Sleep(3 * time.Second)
	err = forwarder.ForwardEntityRemoved("light.old_bulb")
	if err != nil {
		logger.Errorf("Failed to forward entity removed: %v", err)
	}

	// Simulate sync status changes
	time.Sleep(2 * time.Second)
	err = forwarder.ForwardSyncStatus("syncing", "Performing full sync...", 0)
	if err != nil {
		logger.Errorf("Failed to forward sync status: %v", err)
	}

	time.Sleep(1 * time.Second)
	err = forwarder.ForwardSyncStatus("connected", "Sync completed successfully", 47)
	if err != nil {
		logger.Errorf("Failed to forward sync status: %v", err)
	}

	logger.Info("Finished simulating HA events")
}

// displayStats shows forwarding statistics periodically
func displayStats(forwarder *websocket.HAEventForwarder, logger *logrus.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := forwarder.GetForwardingStats()

		logger.WithFields(logrus.Fields{
			"events_forwarded":   stats.EventsForwarded,
			"events_dropped":     stats.EventsDropped,
			"connected_clients":  stats.ConnectedClients,
			"subscribed_clients": stats.SubscribedClients,
			"batched_events":     stats.BatchedEvents,
			"batches_processed":  stats.BatchesProcessed,
			"last_event_time":    stats.LastEventTime,
		}).Info("HA Event Forwarding Statistics")

		// Show event type breakdown
		for eventType, count := range stats.EventTypeStats {
			logger.WithFields(logrus.Fields{
				"event_type": eventType,
				"count":      count,
			}).Debug("Event type statistics")
		}

		// Show errors if any
		if len(stats.ForwardingErrors) > 0 {
			logger.WithField("error_count", len(stats.ForwardingErrors)).Warn("Forwarding errors detected")
			for _, err := range stats.ForwardingErrors {
				logger.WithFields(logrus.Fields{
					"event_type": err.EventType,
					"entity_id":  err.EntityID,
					"error":      err.Error,
					"timestamp":  err.Timestamp,
				}).Warn("Forwarding error details")
			}
		}
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] != substr &&
		(len(s) > len(substr) && s[:len(substr)] == substr ||
			len(s) == len(substr) && s == substr ||
			findInString(s, substr))
}

// findInString is a simple substring search
func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example client simulator (for testing purposes)
func simulateWebSocketClient(logger *logrus.Logger) {
	// This would typically be implemented using a WebSocket client library
	// For demonstration purposes, we'll just log what a client would do

	logger.Info("Simulated WebSocket client connecting...")

	// Client would send subscription messages like:
	subscribeMessage := map[string]interface{}{
		"type": "subscribe_ha_events",
		"data": map[string]interface{}{
			"event_types": []string{
				websocket.MessageTypeHAStateChanged,
				websocket.MessageTypeHASyncStatus,
			},
		},
	}

	logger.WithField("message", subscribeMessage).Info("Client would send subscription message")

	// Client would also set entity and room filters:
	entityFilterMessage := map[string]interface{}{
		"type": "subscribe_ha_entities",
		"data": map[string]interface{}{
			"entity_ids": []string{
				"light.living_room_main",
				"sensor.living_room_temp",
			},
		},
	}

	logger.WithField("message", entityFilterMessage).Info("Client would send entity filter message")

	roomFilterMessage := map[string]interface{}{
		"type": "subscribe_ha_rooms",
		"data": map[string]interface{}{
			"room_ids": []string{
				"living_room",
				"bedroom",
			},
		},
	}

	logger.WithField("message", roomFilterMessage).Info("Client would send room filter message")

	logger.Info("Simulated client setup complete - would now receive filtered HA events")
}
