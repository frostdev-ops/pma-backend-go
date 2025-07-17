package websocket

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ForwardingError represents an error that occurred during event forwarding
type ForwardingError struct {
	EventType   string    `json:"event_type"`
	EntityID    string    `json:"entity_id,omitempty"`
	ClientCount int       `json:"client_count"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
}

// ForwardingStats contains statistics about event forwarding
type ForwardingStats struct {
	EventsForwarded   int64             `json:"events_forwarded"`
	EventsDropped     int64             `json:"events_dropped"`
	ConnectedClients  int               `json:"connected_clients"`
	SubscribedClients int               `json:"subscribed_clients"`
	LastEventTime     time.Time         `json:"last_event_time"`
	ForwardingErrors  []ForwardingError `json:"forwarding_errors"`
	EventTypeStats    map[string]int64  `json:"event_type_stats"`
	BatchedEvents     int64             `json:"batched_events"`
	BatchesProcessed  int64             `json:"batches_processed"`
}

// HAEventForwarderConfig contains configuration for the HA event forwarder
type HAEventForwarderConfig struct {
	MaxEventsPerSecond   int           `yaml:"max_events_per_second"`
	BatchEvents          bool          `yaml:"batch_events"`
	BatchWindow          time.Duration `yaml:"batch_window"`
	DefaultSubscriptions []string      `yaml:"default_subscriptions"`
	ForwardAllEntities   bool          `yaml:"forward_all_entities"`
	MaxErrorsRetained    int           `yaml:"max_errors_retained"`
}

// DefaultHAEventForwarderConfig returns default configuration
func DefaultHAEventForwarderConfig() *HAEventForwarderConfig {
	return &HAEventForwarderConfig{
		MaxEventsPerSecond: 50,
		BatchEvents:        true,
		BatchWindow:        100 * time.Millisecond,
		DefaultSubscriptions: []string{
			MessageTypeHAStateChanged,
			MessageTypeHASyncStatus,
		},
		ForwardAllEntities: false,
		MaxErrorsRetained:  100,
	}
}

// HAEvent represents a Home Assistant event to be forwarded
type HAEvent struct {
	Type      string
	EntityID  string
	Data      interface{}
	RoomID    *string
	Timestamp time.Time
}

// EventBatcher handles batching of rapid events
type EventBatcher struct {
	events     []HAEvent
	timer      *time.Timer
	batchDelay time.Duration
	forwarder  *HAEventForwarder
	mu         sync.Mutex
}

// HAEventForwarder forwards Home Assistant events to WebSocket clients
type HAEventForwarder struct {
	hub    *Hub
	logger *logrus.Logger
	config *HAEventForwarderConfig

	// Event filtering
	enabledEventTypes map[string]bool
	roomFilterMap     map[string][]string // entity_id -> room_ids

	// Rate limiting
	eventCounter  int64
	lastResetTime time.Time
	rateLimitMu   sync.RWMutex

	// Statistics
	stats   *ForwardingStats
	statsMu sync.RWMutex

	// Event batching
	batcher *EventBatcher

	// Errors tracking
	errors   []ForwardingError
	errorsMu sync.RWMutex
}

// HAEventForwarderInterface defines the interface for the HA event forwarder
type HAEventForwarderInterface interface {
	// Event forwarding
	ForwardStateChanged(entityID string, oldState, newState interface{}, attributes map[string]interface{}) error
	ForwardEntityAdded(entityID string, entityData interface{}) error
	ForwardEntityRemoved(entityID string) error
	ForwardAreaUpdated(areaID string, areaData interface{}) error
	ForwardSyncStatus(status string, message string, entityCount int) error
	ForwardServiceCalled(service string, serviceData map[string]interface{}, entityID *string) error

	// Configuration
	SetEventTypeEnabled(eventType string, enabled bool)
	UpdateRoomFilters(entityRoomMap map[string]string) error

	// Client filtering
	ShouldForwardToClient(client *Client, entityID string, eventType string) bool

	// Statistics
	GetForwardingStats() ForwardingStats

	// Lifecycle
	Start() error
	Stop() error
}

// NewHAEventForwarder creates a new HA event forwarder
func NewHAEventForwarder(hub *Hub, logger *logrus.Logger, config *HAEventForwarderConfig) *HAEventForwarder {
	if config == nil {
		config = DefaultHAEventForwarderConfig()
	}

	forwarder := &HAEventForwarder{
		hub:               hub,
		logger:            logger,
		config:            config,
		enabledEventTypes: make(map[string]bool),
		roomFilterMap:     make(map[string][]string),
		lastResetTime:     time.Now(),
		stats: &ForwardingStats{
			EventTypeStats: make(map[string]int64),
		},
		errors: make([]ForwardingError, 0, config.MaxErrorsRetained),
	}

	// Initialize default subscriptions
	for _, eventType := range config.DefaultSubscriptions {
		forwarder.enabledEventTypes[eventType] = true
	}

	// Initialize event batcher if enabled
	if config.BatchEvents {
		forwarder.batcher = &EventBatcher{
			events:     make([]HAEvent, 0, 10),
			batchDelay: config.BatchWindow,
			forwarder:  forwarder,
		}
	}

	return forwarder
}

// Start starts the event forwarder
func (f *HAEventForwarder) Start() error {
	f.logger.Info("Starting HA Event Forwarder")

	// Start rate limit reset ticker
	go f.startRateLimitReset()

	return nil
}

// Stop stops the event forwarder
func (f *HAEventForwarder) Stop() error {
	f.logger.Info("Stopping HA Event Forwarder")

	// Flush any pending batched events
	if f.batcher != nil {
		f.batcher.FlushBatch()
	}

	return nil
}

// ForwardStateChanged forwards a state change event
func (f *HAEventForwarder) ForwardStateChanged(entityID string, oldState, newState interface{}, attributes map[string]interface{}) error {
	if !f.isEventTypeEnabled(MessageTypeHAStateChanged) {
		return nil
	}

	if !f.checkRateLimit() {
		f.incrementDroppedEvents()
		return nil
	}

	// Convert state values to strings
	oldStateStr := ""
	if oldState != nil {
		oldStateStr = oldState.(string)
	}
	newStateStr := newState.(string)

	// Get room ID if available
	roomID := f.getRoomIDForEntity(entityID)

	// Create message
	message := NewHAStateChangedMessage(entityID, oldStateStr, newStateStr, attributes, roomID)

	// Forward to clients
	return f.forwardEventToClients(message.Type, entityID, message.ToMessage(), roomID)
}

// ForwardEntityAdded forwards an entity added event
func (f *HAEventForwarder) ForwardEntityAdded(entityID string, entityData interface{}) error {
	if !f.isEventTypeEnabled(MessageTypeHAEntityAdded) {
		return nil
	}

	if !f.checkRateLimit() {
		f.incrementDroppedEvents()
		return nil
	}

	roomID := f.getRoomIDForEntity(entityID)

	// Convert entity data to map
	entityDataMap, ok := entityData.(map[string]interface{})
	if !ok {
		entityDataMap = map[string]interface{}{"data": entityData}
	}

	message := NewHAEntityAddedMessage(entityID, entityDataMap, roomID)
	return f.forwardEventToClients(message.Type, entityID, message.ToMessage(), roomID)
}

// ForwardEntityRemoved forwards an entity removed event
func (f *HAEventForwarder) ForwardEntityRemoved(entityID string) error {
	if !f.isEventTypeEnabled(MessageTypeHAEntityRemoved) {
		return nil
	}

	if !f.checkRateLimit() {
		f.incrementDroppedEvents()
		return nil
	}

	roomID := f.getRoomIDForEntity(entityID)
	message := NewHAEntityRemovedMessage(entityID, roomID)
	return f.forwardEventToClients(message.Type, entityID, message.ToMessage(), roomID)
}

// ForwardAreaUpdated forwards an area update event
func (f *HAEventForwarder) ForwardAreaUpdated(areaID string, areaData interface{}) error {
	if !f.isEventTypeEnabled(MessageTypeHAAreaUpdated) {
		return nil
	}

	if !f.checkRateLimit() {
		f.incrementDroppedEvents()
		return nil
	}

	// For area updates, we don't have a specific entity, so use area ID
	message := Message{
		Type: MessageTypeHAAreaUpdated,
		Data: map[string]interface{}{
			"area_id":   areaID,
			"area_data": areaData,
		},
		Timestamp: time.Now().UTC(),
	}

	return f.forwardEventToClients(MessageTypeHAAreaUpdated, "", message, nil)
}

// ForwardSyncStatus forwards a sync status event
func (f *HAEventForwarder) ForwardSyncStatus(status string, message string, entityCount int) error {
	if !f.isEventTypeEnabled(MessageTypeHASyncStatus) {
		return nil
	}

	statusMessage := NewHASyncStatusMessage(status, message, entityCount)
	return f.forwardEventToClients(statusMessage.Type, "", statusMessage.ToMessage(), nil)
}

// ForwardServiceCalled forwards a service call event
func (f *HAEventForwarder) ForwardServiceCalled(service string, serviceData map[string]interface{}, entityID *string) error {
	if !f.isEventTypeEnabled(MessageTypeHAServiceCalled) {
		return nil
	}

	if !f.checkRateLimit() {
		f.incrementDroppedEvents()
		return nil
	}

	var roomID *string
	if entityID != nil {
		roomID = f.getRoomIDForEntity(*entityID)
	}

	message := NewHAServiceCalledMessage(service, serviceData, entityID, roomID)
	entityIDStr := ""
	if entityID != nil {
		entityIDStr = *entityID
	}

	return f.forwardEventToClients(message.Type, entityIDStr, message.ToMessage(), roomID)
}

// forwardEventToClients forwards an event to appropriate clients
func (f *HAEventForwarder) forwardEventToClients(eventType, entityID string, message Message, roomID *string) error {
	if f.config.BatchEvents && f.batcher != nil {
		return f.batcher.AddEvent(HAEvent{
			Type:      eventType,
			EntityID:  entityID,
			Data:      message,
			RoomID:    roomID,
			Timestamp: time.Now(),
		})
	}

	return f.sendToClients(eventType, entityID, message, roomID)
}

// sendToClients sends a message to appropriate clients
func (f *HAEventForwarder) sendToClients(eventType, entityID string, message Message, roomID *string) error {
	f.hub.mu.RLock()
	clients := make([]*Client, 0, len(f.hub.clients))
	for client := range f.hub.clients {
		clients = append(clients, client)
	}
	f.hub.mu.RUnlock()

	var sentCount int
	var errors []error

	for _, client := range clients {
		if f.ShouldForwardToClient(client, entityID, eventType) {
			select {
			case client.send <- message.ToJSON():
				sentCount++
			default:
				// Client's send channel is full, skip
				f.addError(ForwardingError{
					EventType:   eventType,
					EntityID:    entityID,
					ClientCount: 1,
					Error:       "client send channel full",
					Timestamp:   time.Now(),
				})
			}
		}
	}

	f.updateStats(eventType, sentCount, len(errors))

	if len(errors) > 0 {
		return errors[0] // Return first error
	}

	return nil
}

// ShouldForwardToClient determines if an event should be forwarded to a specific client
func (f *HAEventForwarder) ShouldForwardToClient(client *Client, entityID string, eventType string) bool {
	// Check if client is subscribed to this event type
	if !client.IsSubscribedToHAEvent(eventType) {
		return false
	}

	// If entityID is provided, check entity and room filters
	if entityID != "" {
		// Check entity filter
		if !client.IsSubscribedToHAEntity(entityID) {
			return false
		}

		// Check room filter if entity has room association
		if roomIDs, exists := f.roomFilterMap[entityID]; exists && len(roomIDs) > 0 {
			roomSubscribed := false
			for _, roomID := range roomIDs {
				if client.IsSubscribedToHARoom(roomID) {
					roomSubscribed = true
					break
				}
			}
			if !roomSubscribed {
				return false
			}
		}
	}

	return true
}

// Configuration methods

// SetEventTypeEnabled enables or disables forwarding for an event type
func (f *HAEventForwarder) SetEventTypeEnabled(eventType string, enabled bool) {
	f.enabledEventTypes[eventType] = enabled
	f.logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"enabled":    enabled,
	}).Info("Event type forwarding updated")
}

// UpdateRoomFilters updates the entity-to-room mapping
func (f *HAEventForwarder) UpdateRoomFilters(entityRoomMap map[string]string) error {
	// Convert single room per entity to list format
	newRoomFilters := make(map[string][]string)
	for entityID, roomID := range entityRoomMap {
		newRoomFilters[entityID] = []string{roomID}
	}

	f.roomFilterMap = newRoomFilters
	f.logger.WithField("entity_count", len(entityRoomMap)).Info("Room filters updated")

	return nil
}

// Helper methods

// isEventTypeEnabled checks if an event type is enabled for forwarding
func (f *HAEventForwarder) isEventTypeEnabled(eventType string) bool {
	enabled, exists := f.enabledEventTypes[eventType]
	return exists && enabled
}

// getRoomIDForEntity gets the room ID for an entity
func (f *HAEventForwarder) getRoomIDForEntity(entityID string) *string {
	if rooms, exists := f.roomFilterMap[entityID]; exists && len(rooms) > 0 {
		return &rooms[0] // Return first room for now
	}
	return nil
}

// checkRateLimit checks if we're within rate limits
func (f *HAEventForwarder) checkRateLimit() bool {
	f.rateLimitMu.Lock()
	defer f.rateLimitMu.Unlock()

	now := time.Now()
	if now.Sub(f.lastResetTime) >= time.Second {
		f.eventCounter = 0
		f.lastResetTime = now
	}

	if f.eventCounter >= int64(f.config.MaxEventsPerSecond) {
		return false
	}

	f.eventCounter++
	return true
}

// startRateLimitReset starts the rate limit reset ticker
func (f *HAEventForwarder) startRateLimitReset() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		f.rateLimitMu.Lock()
		f.eventCounter = 0
		f.lastResetTime = time.Now()
		f.rateLimitMu.Unlock()
	}
}

// Statistics methods

// updateStats updates forwarding statistics
func (f *HAEventForwarder) updateStats(eventType string, sentCount, errorCount int) {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()

	f.stats.EventsForwarded += int64(sentCount)
	f.stats.EventsDropped += int64(errorCount)
	f.stats.LastEventTime = time.Now()
	f.stats.EventTypeStats[eventType]++
}

// incrementDroppedEvents increments the dropped events counter
func (f *HAEventForwarder) incrementDroppedEvents() {
	f.statsMu.Lock()
	defer f.statsMu.Unlock()
	f.stats.EventsDropped++
}

// addError adds a forwarding error to the error list
func (f *HAEventForwarder) addError(err ForwardingError) {
	f.errorsMu.Lock()
	defer f.errorsMu.Unlock()

	f.errors = append(f.errors, err)

	// Keep only the most recent errors
	if len(f.errors) > f.config.MaxErrorsRetained {
		f.errors = f.errors[1:]
	}
}

// GetForwardingStats returns current forwarding statistics
func (f *HAEventForwarder) GetForwardingStats() ForwardingStats {
	f.statsMu.RLock()
	defer f.statsMu.RUnlock()

	f.errorsMu.RLock()
	defer f.errorsMu.RUnlock()

	// Update client counts
	f.hub.mu.RLock()
	connectedClients := len(f.hub.clients)
	subscribedClients := 0
	for client := range f.hub.clients {
		if len(client.GetHASubscriptions()) > 0 {
			subscribedClients++
		}
	}
	f.hub.mu.RUnlock()

	// Create a copy of the stats
	stats := *f.stats
	stats.ConnectedClients = connectedClients
	stats.SubscribedClients = subscribedClients

	// Copy errors to avoid race conditions
	stats.ForwardingErrors = make([]ForwardingError, len(f.errors))
	copy(stats.ForwardingErrors, f.errors)

	// Copy event type stats
	stats.EventTypeStats = make(map[string]int64)
	for eventType, count := range f.stats.EventTypeStats {
		stats.EventTypeStats[eventType] = count
	}

	return stats
}

// Event batching implementation

// AddEvent adds an event to the batch
func (b *EventBatcher) AddEvent(event HAEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events = append(b.events, event)

	// Start timer if this is the first event in the batch
	if len(b.events) == 1 {
		b.timer = time.AfterFunc(b.batchDelay, b.FlushBatch)
	}

	return nil
}

// FlushBatch processes all events in the current batch
func (b *EventBatcher) FlushBatch() {
	b.mu.Lock()
	events := make([]HAEvent, len(b.events))
	copy(events, b.events)
	b.events = b.events[:0] // Clear the slice

	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.mu.Unlock()

	if len(events) == 0 {
		return
	}

	// Update batch statistics
	b.forwarder.statsMu.Lock()
	b.forwarder.stats.BatchedEvents += int64(len(events))
	b.forwarder.stats.BatchesProcessed++
	b.forwarder.statsMu.Unlock()

	// Process each event
	for _, event := range events {
		if message, ok := event.Data.(Message); ok {
			b.forwarder.sendToClients(event.Type, event.EntityID, message, event.RoomID)
		}
	}

	b.forwarder.logger.WithField("batch_size", len(events)).Debug("Processed event batch")
}
