package homeassistant

import (
	"context"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHomeAssistantAdapter(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()

	adapter := NewHomeAssistantAdapter(cfg, logger)

	assert.NotNil(t, adapter)
	assert.Equal(t, "homeassistant_primary", adapter.GetID())
	assert.Equal(t, types.SourceHomeAssistant, adapter.GetSourceType())
	assert.Equal(t, "Home Assistant", adapter.GetName())
	assert.Equal(t, "1.0.0", adapter.GetVersion())
	assert.False(t, adapter.IsConnected())
	assert.Equal(t, "disconnected", adapter.GetStatus())
}

func TestAdapterInterfaceImplementation(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()

	var adapter types.PMAAdapter = NewHomeAssistantAdapter(cfg, logger)

	// Test that all interface methods are available
	assert.NotEmpty(t, adapter.GetID())
	assert.Equal(t, types.SourceHomeAssistant, adapter.GetSourceType())
	assert.NotEmpty(t, adapter.GetName())
	assert.NotEmpty(t, adapter.GetVersion())
	assert.True(t, adapter.SupportsRealtime())

	// Test capabilities
	entityTypes := adapter.GetSupportedEntityTypes()
	assert.Contains(t, entityTypes, types.EntityTypeLight)
	assert.Contains(t, entityTypes, types.EntityTypeSwitch)
	assert.Contains(t, entityTypes, types.EntityTypeSensor)

	capabilities := adapter.GetSupportedCapabilities()
	assert.Contains(t, capabilities, types.CapabilityDimmable)
	assert.Contains(t, capabilities, types.CapabilityColorable)
	assert.Contains(t, capabilities, types.CapabilityTemperature)
}

func TestEntityConversion(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	// Test light entity conversion
	haLight := &HAEntity{
		EntityID:     "light.living_room",
		State:        "on",
		FriendlyName: "Living Room Light",
		Domain:       "light",
		Attributes: map[string]interface{}{
			"brightness":   200,
			"color_mode":   "rgb",
			"rgb_color":    []int{255, 255, 255},
			"device_class": "light",
		},
		LastUpdated: time.Now(),
		LastChanged: time.Now(),
	}

	pmaEntity, err := adapter.ConvertEntity(haLight)
	require.NoError(t, err)
	require.NotNil(t, pmaEntity)

	assert.Equal(t, "ha_light.living_room", pmaEntity.GetID())
	assert.Equal(t, types.EntityTypeLight, pmaEntity.GetType())
	assert.Equal(t, "Living Room Light", pmaEntity.GetFriendlyName())
	assert.Equal(t, types.StateOn, pmaEntity.GetState())
	assert.True(t, pmaEntity.IsAvailable())
	assert.Equal(t, types.SourceHomeAssistant, pmaEntity.GetSource())

	// Check capabilities
	assert.True(t, pmaEntity.HasCapability(types.CapabilityBrightness))
	assert.True(t, pmaEntity.HasCapability(types.CapabilityColorable))
}

func TestSensorConversion(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	haSensor := &HAEntity{
		EntityID:     "sensor.temperature",
		State:        "23.5",
		FriendlyName: "Temperature Sensor",
		Domain:       "sensor",
		Attributes: map[string]interface{}{
			"unit_of_measurement": "°C",
			"device_class":        "temperature",
		},
		LastUpdated: time.Now(),
		LastChanged: time.Now(),
	}

	pmaEntity, err := adapter.ConvertEntity(haSensor)
	require.NoError(t, err)
	require.NotNil(t, pmaEntity)

	assert.Equal(t, "ha_sensor.temperature", pmaEntity.GetID())
	assert.Equal(t, types.EntityTypeSensor, pmaEntity.GetType())
	assert.Equal(t, "Temperature Sensor", pmaEntity.GetFriendlyName())
	assert.True(t, pmaEntity.HasCapability(types.CapabilityTemperature))

	// Check that numeric value was added
	attributes := pmaEntity.GetAttributes()
	assert.Equal(t, 23.5, attributes["numeric_value"])
	assert.Equal(t, "°C", attributes["unit_of_measurement"])
}

func TestRoomConversion(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	haArea := &HAArea{
		ID:      "living_room",
		Name:    "Living Room",
		Icon:    "mdi:sofa",
		Aliases: []string{"Main Room", "Front Room"},
	}

	pmaRoom, err := adapter.ConvertRoom(haArea)
	require.NoError(t, err)
	require.NotNil(t, pmaRoom)

	assert.Equal(t, "ha_room_living_room", pmaRoom.ID)
	assert.Equal(t, "Living Room", pmaRoom.Name)
	assert.Equal(t, "mdi:sofa", pmaRoom.Icon)
	assert.Equal(t, "Main Room", pmaRoom.Description)
	assert.Equal(t, types.SourceHomeAssistant, pmaRoom.GetSource())
}

func TestAreaConversion(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	haArea := &HAArea{
		ID:      "ground_floor",
		Name:    "Ground Floor",
		Icon:    "mdi:home-floor-g",
		Aliases: []string{"Main Floor"},
	}

	pmaArea, err := adapter.ConvertArea(haArea)
	require.NoError(t, err)
	require.NotNil(t, pmaArea)

	assert.Equal(t, "ha_area_ground_floor", pmaArea.ID)
	assert.Equal(t, "Ground Floor", pmaArea.Name)
	assert.Equal(t, "mdi:home-floor-g", pmaArea.Icon)
	assert.Equal(t, "Main Floor", pmaArea.Description)
	assert.Equal(t, types.SourceHomeAssistant, pmaArea.GetSource())
}

func TestInvalidEntityConversion(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	// Test with nil entity
	_, err := adapter.ConvertEntity(nil)
	assert.Error(t, err)

	// Test with wrong type
	_, err = adapter.ConvertEntity("invalid")
	assert.Error(t, err)
}

func TestActionValidation(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	// Test invalid action (adapter not connected)
	invalidAction := types.PMAControlAction{
		EntityID: "",
		Action:   "",
	}

	result, err := adapter.ExecuteAction(context.Background(), invalidAction)
	require.NoError(t, err) // Error should be in result, not returned
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
	assert.Equal(t, "INVALID_ACTION", result.Error.Code)
}

func TestHealthAndMetrics(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}
	logger := logrus.New()
	adapter := NewHomeAssistantAdapter(cfg, logger)

	// Test health
	health := adapter.GetHealth()
	assert.NotNil(t, health)
	assert.False(t, health.IsHealthy) // Should be false since not connected
	assert.NotZero(t, health.LastHealthCheck)

	// Test metrics
	metrics := adapter.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.ActionsExecuted)
	assert.Equal(t, int64(0), metrics.SuccessfulActions)
	assert.Equal(t, int64(0), metrics.FailedActions)
	assert.GreaterOrEqual(t, metrics.Uptime, time.Duration(0))
}

func TestGetFirstAlias(t *testing.T) {
	// Test with aliases
	aliases := []string{"First", "Second", "Third"}
	result := getFirstAlias(aliases)
	assert.Equal(t, "First", result)

	// Test with empty slice
	result = getFirstAlias([]string{})
	assert.Equal(t, "", result)

	// Test with nil slice
	result = getFirstAlias(nil)
	assert.Equal(t, "", result)
}
