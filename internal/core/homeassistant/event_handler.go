package homeassistant

import (
	"context"
	"fmt"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/sirupsen/logrus"
)

// EventProcessor handles processing of Home Assistant events
type EventProcessor struct {
	syncSvc *SyncService
	mapper  *EntityMapper
	logger  *logrus.Logger
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(syncSvc *SyncService, mapper *EntityMapper, logger *logrus.Logger) *EventProcessor {
	return &EventProcessor{
		syncSvc: syncSvc,
		mapper:  mapper,
		logger:  logger,
	}
}

// EventProcessorInterface defines the interface for event processing
type EventProcessorInterface interface {
	// Event processing
	HandleStateChanged(event homeassistant.Event) error
	HandleEntityRegistryUpdated(event homeassistant.Event) error
	HandleAreaRegistryUpdated(event homeassistant.Event) error

	// Event filtering
	ShouldProcessEvent(eventType string, entityID string) bool

	// Error handling
	HandleEventError(event interface{}, err error)
}

// HandleStateChanged handles Home Assistant state change events
func (p *EventProcessor) HandleStateChanged(event homeassistant.Event) error {
	data := event.Data
	entityID, ok := data["entity_id"].(string)
	if !ok {
		return fmt.Errorf("invalid entity_id in state change event")
	}

	newState, ok := data["new_state"].(*homeassistant.EntityState)
	if !ok || newState == nil {
		return fmt.Errorf("invalid new_state in state change event")
	}

	p.logger.Debugf("Processing state change for entity: %s", entityID)
	return p.syncSvc.syncSingleEntity(context.Background(), newState)
}

// HandleEntityRegistryUpdated handles entity registry update events
func (p *EventProcessor) HandleEntityRegistryUpdated(event homeassistant.Event) error {
	p.logger.Debug("Entity registry updated, triggering entity resync")

	// For now, we'll just log this. In a more sophisticated implementation,
	// we could parse the event data and only sync affected entities
	go func() {
		ctx := context.Background()
		if err := p.syncSvc.FullSync(ctx); err != nil {
			p.logger.WithError(err).Error("Failed to sync after entity registry update")
		}
	}()

	return nil
}

// HandleAreaRegistryUpdated handles area registry update events
func (p *EventProcessor) HandleAreaRegistryUpdated(event homeassistant.Event) error {
	p.logger.Debug("Area registry updated, triggering area resync")

	go func() {
		ctx := context.Background()
		if err := p.syncSvc.syncAreas(ctx); err != nil {
			p.logger.WithError(err).Error("Failed to sync areas after registry update")
		}
	}()

	return nil
}

// ShouldProcessEvent determines if an event should be processed
func (p *EventProcessor) ShouldProcessEvent(eventType string, entityID string) bool {
	// Check if this is a supported event type
	supportedEvents := map[string]bool{
		"state_changed":           true,
		"entity_registry_updated": true,
		"area_registry_updated":   true,
	}

	if !supportedEvents[eventType] {
		return false
	}

	// For state changes, check if the entity domain is supported
	if eventType == "state_changed" && entityID != "" {
		return p.syncSvc.shouldProcessEntity(entityID)
	}

	return true
}

// HandleEventError handles errors that occur during event processing
func (p *EventProcessor) HandleEventError(event interface{}, err error) {
	p.logger.WithError(err).Warnf("Failed to process event: %v", event)

	// Could implement retry logic or dead letter queue here
	// For now, just log the error
}
