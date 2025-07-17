package websocket

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHub is a mock implementation of the Hub for testing
type MockHub struct {
	mock.Mock
	clients map[*Client]bool
	mu      struct{}
}

func (m *MockHub) GetClientByID(clientID string) *Client {
	args := m.Called(clientID)
	if client := args.Get(0); client != nil {
		return client.(*Client)
	}
	return nil
}

func (m *MockHub) GetAllClients() []*Client {
	args := m.Called()
	return args.Get(0).([]*Client)
}

func (m *MockHub) BroadcastToAll(message Message) {
	m.Called(message)
}

// MockClient is a mock implementation of the Client for testing
type MockClient struct {
	ID              string
	haSubscriptions map[string]bool
	entityFilters   map[string]bool
	roomFilters     map[string]bool
	sendChannel     chan []byte
}

func NewMockClient(id string) *MockClient {
	return &MockClient{
		ID:              id,
		haSubscriptions: make(map[string]bool),
		entityFilters:   make(map[string]bool),
		roomFilters:     make(map[string]bool),
		sendChannel:     make(chan []byte, 10),
	}
}

func (c *MockClient) IsSubscribedToHAEvent(eventType string) bool {
	return c.haSubscriptions[eventType]
}

func (c *MockClient) IsSubscribedToHAEntity(entityID string) bool {
	if len(c.entityFilters) == 0 {
		return true
	}
	return c.entityFilters[entityID]
}

func (c *MockClient) IsSubscribedToHARoom(roomID string) bool {
	if len(c.roomFilters) == 0 {
		return true
	}
	return c.roomFilters[roomID]
}

func (c *MockClient) GetHASubscriptions() map[string]bool {
	result := make(map[string]bool)
	for k, v := range c.haSubscriptions {
		result[k] = v
	}
	return result
}

func TestNewHAEventForwarder(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	hub := &MockHub{clients: make(map[*Client]bool)}
	config := DefaultHAEventForwarderConfig()

	forwarder := NewHAEventForwarder(hub, logger, config)

	assert.NotNil(t, forwarder)
	assert.Equal(t, hub, forwarder.hub)
	assert.Equal(t, logger, forwarder.logger)
	assert.Equal(t, config, forwarder.config)
	assert.NotNil(t, forwarder.enabledEventTypes)
	assert.NotNil(t, forwarder.roomFilterMap)
	assert.NotNil(t, forwarder.stats)

	// Check default subscriptions are enabled
	assert.True(t, forwarder.enabledEventTypes[MessageTypeHAStateChanged])
	assert.True(t, forwarder.enabledEventTypes[MessageTypeHASyncStatus])
}

func TestHAEventForwarder_ForwardStateChanged(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	hub := &MockHub{clients: make(map[*Client]bool)}
	config := DefaultHAEventForwarderConfig()
	config.BatchEvents = false // Disable batching for direct testing

	forwarder := NewHAEventForwarder(hub, logger, config)

	// Create mock clients
	client1 := NewMockClient("client1")
	client1.haSubscriptions[MessageTypeHAStateChanged] = true

	client2 := NewMockClient("client2")
	client2.haSubscriptions[MessageTypeHAStateChanged] = false

	clients := []*Client{(*Client)(client1), (*Client)(client2)}
	hub.On("GetAllClients").Return(clients)

	// Test forwarding
	entityID := "light.living_room"
	oldState := "off"
	newState := "on"
	attributes := map[string]interface{}{
		"brightness": 255,
		"color_temp": 154,
	}

	err := forwarder.ForwardStateChanged(entityID, oldState, newState, attributes)
	assert.NoError(t, err)

	// Check statistics
	stats := forwarder.GetForwardingStats()
	assert.Greater(t, stats.EventsForwarded, int64(0))
	assert.Equal(t, int64(1), stats.EventTypeStats[MessageTypeHAStateChanged])
}

func TestHAEventForwarder_ShouldForwardToClient(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}
	forwarder := NewHAEventForwarder(hub, logger, nil)

	client := NewMockClient("test-client")

	tests := []struct {
		name                string
		eventType           string
		entityID            string
		clientSubscriptions map[string]bool
		clientEntityFilters map[string]bool
		clientRoomFilters   map[string]bool
		roomFilters         map[string][]string
		expected            bool
	}{
		{
			name:                "not subscribed to event type",
			eventType:           MessageTypeHAStateChanged,
			entityID:            "light.test",
			clientSubscriptions: map[string]bool{},
			expected:            false,
		},
		{
			name:                "subscribed to event type",
			eventType:           MessageTypeHAStateChanged,
			entityID:            "light.test",
			clientSubscriptions: map[string]bool{MessageTypeHAStateChanged: true},
			expected:            true,
		},
		{
			name:                "subscribed to event type, entity filter blocks",
			eventType:           MessageTypeHAStateChanged,
			entityID:            "light.test",
			clientSubscriptions: map[string]bool{MessageTypeHAStateChanged: true},
			clientEntityFilters: map[string]bool{"light.other": true},
			expected:            false,
		},
		{
			name:                "subscribed to event type, entity filter allows",
			eventType:           MessageTypeHAStateChanged,
			entityID:            "light.test",
			clientSubscriptions: map[string]bool{MessageTypeHAStateChanged: true},
			clientEntityFilters: map[string]bool{"light.test": true},
			expected:            true,
		},
		{
			name:                "room filter blocks",
			eventType:           MessageTypeHAStateChanged,
			entityID:            "light.test",
			clientSubscriptions: map[string]bool{MessageTypeHAStateChanged: true},
			clientRoomFilters:   map[string]bool{"bedroom": true},
			roomFilters:         map[string][]string{"light.test": {"living_room"}},
			expected:            false,
		},
		{
			name:                "room filter allows",
			eventType:           MessageTypeHAStateChanged,
			entityID:            "light.test",
			clientSubscriptions: map[string]bool{MessageTypeHAStateChanged: true},
			clientRoomFilters:   map[string]bool{"living_room": true},
			roomFilters:         map[string][]string{"light.test": {"living_room"}},
			expected:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset client state
			client.haSubscriptions = tt.clientSubscriptions
			client.entityFilters = tt.clientEntityFilters
			client.roomFilters = tt.clientRoomFilters

			// Set forwarder room filters
			forwarder.roomFilterMap = tt.roomFilters

			result := forwarder.ShouldForwardToClient((*Client)(client), tt.entityID, tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHAEventForwarder_RateLimit(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}

	config := DefaultHAEventForwarderConfig()
	config.MaxEventsPerSecond = 2
	config.BatchEvents = false

	forwarder := NewHAEventForwarder(hub, logger, config)

	hub.On("GetAllClients").Return([]*Client{})

	// Send events up to the limit
	for i := 0; i < 2; i++ {
		err := forwarder.ForwardStateChanged("test.entity", "off", "on", nil)
		assert.NoError(t, err)
	}

	// This should be rate limited
	err := forwarder.ForwardStateChanged("test.entity", "off", "on", nil)
	assert.NoError(t, err) // No error, but event should be dropped

	stats := forwarder.GetForwardingStats()
	assert.Greater(t, stats.EventsDropped, int64(0))
}

func TestHAEventForwarder_EventBatching(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}

	config := DefaultHAEventForwarderConfig()
	config.BatchEvents = true
	config.BatchWindow = 50 * time.Millisecond

	forwarder := NewHAEventForwarder(hub, logger, config)

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHAStateChanged] = true

	hub.On("GetAllClients").Return([]*Client{(*Client)(client)})

	// Start the forwarder
	err := forwarder.Start()
	require.NoError(t, err)

	// Send multiple events quickly
	for i := 0; i < 5; i++ {
		err := forwarder.ForwardStateChanged("test.entity", "off", "on", nil)
		assert.NoError(t, err)
	}

	// Wait for batch to be processed
	time.Sleep(100 * time.Millisecond)

	stats := forwarder.GetForwardingStats()
	assert.Greater(t, stats.BatchedEvents, int64(0))
	assert.Greater(t, stats.BatchesProcessed, int64(0))

	forwarder.Stop()
}

func TestHAEventForwarder_UpdateRoomFilters(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}
	forwarder := NewHAEventForwarder(hub, logger, nil)

	entityRoomMap := map[string]string{
		"light.living_room": "living_room",
		"light.bedroom":     "bedroom",
		"switch.kitchen":    "kitchen",
	}

	err := forwarder.UpdateRoomFilters(entityRoomMap)
	assert.NoError(t, err)

	// Verify room filters were updated
	assert.Equal(t, []string{"living_room"}, forwarder.roomFilterMap["light.living_room"])
	assert.Equal(t, []string{"bedroom"}, forwarder.roomFilterMap["light.bedroom"])
	assert.Equal(t, []string{"kitchen"}, forwarder.roomFilterMap["switch.kitchen"])
}

func TestHAEventForwarder_ForwardSyncStatus(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}

	config := DefaultHAEventForwarderConfig()
	config.BatchEvents = false

	forwarder := NewHAEventForwarder(hub, logger, config)

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHASyncStatus] = true

	hub.On("GetAllClients").Return([]*Client{(*Client)(client)})

	err := forwarder.ForwardSyncStatus("connected", "Successfully connected to Home Assistant", 150)
	assert.NoError(t, err)

	stats := forwarder.GetForwardingStats()
	assert.Equal(t, int64(1), stats.EventTypeStats[MessageTypeHASyncStatus])
}

func TestHAEventForwarder_ForwardEntityAdded(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}

	config := DefaultHAEventForwarderConfig()
	config.BatchEvents = false

	forwarder := NewHAEventForwarder(hub, logger, config)

	// Enable entity added events
	forwarder.SetEventTypeEnabled(MessageTypeHAEntityAdded, true)

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHAEntityAdded] = true

	hub.On("GetAllClients").Return([]*Client{(*Client)(client)})

	entityData := map[string]interface{}{
		"entity_id":     "light.new_light",
		"friendly_name": "New Light",
		"state":         "off",
		"domain":        "light",
	}

	err := forwarder.ForwardEntityAdded("light.new_light", entityData)
	assert.NoError(t, err)

	stats := forwarder.GetForwardingStats()
	assert.Equal(t, int64(1), stats.EventTypeStats[MessageTypeHAEntityAdded])
}

func TestHAEventForwarder_SetEventTypeEnabled(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}
	forwarder := NewHAEventForwarder(hub, logger, nil)

	// Initially not enabled
	assert.False(t, forwarder.isEventTypeEnabled("test_event"))

	// Enable it
	forwarder.SetEventTypeEnabled("test_event", true)
	assert.True(t, forwarder.isEventTypeEnabled("test_event"))

	// Disable it
	forwarder.SetEventTypeEnabled("test_event", false)
	assert.False(t, forwarder.isEventTypeEnabled("test_event"))
}

func TestHAEventForwarder_GetForwardingStats(t *testing.T) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}
	forwarder := NewHAEventForwarder(hub, logger, nil)

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHAStateChanged] = true

	hub.On("GetAllClients").Return([]*Client{(*Client)(client)})

	stats := forwarder.GetForwardingStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.ConnectedClients)
	assert.Equal(t, 1, stats.SubscribedClients)
	assert.NotNil(t, stats.EventTypeStats)
	assert.NotNil(t, stats.ForwardingErrors)
}

// Benchmark tests

func BenchmarkEventForwarding(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise

	hub := &MockHub{clients: make(map[*Client]bool)}
	config := DefaultHAEventForwarderConfig()
	config.BatchEvents = false

	forwarder := NewHAEventForwarder(hub, logger, config)

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHAStateChanged] = true

	hub.On("GetAllClients").Return([]*Client{(*Client)(client)})

	attributes := map[string]interface{}{
		"brightness": 255,
		"color_temp": 154,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		forwarder.ForwardStateChanged("light.test", "off", "on", attributes)
	}
}

func BenchmarkEventBatching(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	hub := &MockHub{clients: make(map[*Client]bool)}
	config := DefaultHAEventForwarderConfig()
	config.BatchEvents = true
	config.BatchWindow = 10 * time.Millisecond

	forwarder := NewHAEventForwarder(hub, logger, config)
	forwarder.Start()
	defer forwarder.Stop()

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHAStateChanged] = true

	hub.On("GetAllClients").Return([]*Client{(*Client)(client)})

	attributes := map[string]interface{}{
		"brightness": 255,
		"color_temp": 154,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		forwarder.ForwardStateChanged("light.test", "off", "on", attributes)
	}
}

func BenchmarkClientFiltering(b *testing.B) {
	logger := logrus.New()
	hub := &MockHub{clients: make(map[*Client]bool)}
	forwarder := NewHAEventForwarder(hub, logger, nil)

	client := NewMockClient("test-client")
	client.haSubscriptions[MessageTypeHAStateChanged] = true
	client.entityFilters["light.test"] = true
	client.roomFilters["living_room"] = true

	forwarder.roomFilterMap = map[string][]string{
		"light.test": {"living_room"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		forwarder.ShouldForwardToClient((*Client)(client), "light.test", MessageTypeHAStateChanged)
	}
}
