package types

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PMATypeRegistry manages all PMA types and provides type discovery
type PMATypeRegistry struct {
	entityTypes         map[PMAEntityType]reflect.Type
	adapterRegistry     AdapterRegistry
	entityRegistry      EntityRegistry
	roomRegistry        map[string]*PMARoom
	areaRegistry        map[string]*PMAArea
	sceneRegistry       map[string]*PMAScene
	automationRegistry  map[string]*PMAAutomation
	integrationRegistry map[string]*PMAIntegration

	// Type factories
	entityFactories map[PMAEntityType]EntityFactory

	// Configuration and services
	conflictResolver  ConflictResolver
	priorityManager   SourcePriorityManager
	qualityAssessment QualityAssessment
	idGenerator       EntityIDGenerator

	logger *logrus.Logger
	mutex  sync.RWMutex
}

// EntityFactory creates new instances of specific entity types
type EntityFactory func() PMAEntity

// NewPMATypeRegistry creates a new type registry
func NewPMATypeRegistry(logger *logrus.Logger) *PMATypeRegistry {
	registry := &PMATypeRegistry{
		entityTypes:         make(map[PMAEntityType]reflect.Type),
		roomRegistry:        make(map[string]*PMARoom),
		areaRegistry:        make(map[string]*PMAArea),
		sceneRegistry:       make(map[string]*PMAScene),
		automationRegistry:  make(map[string]*PMAAutomation),
		integrationRegistry: make(map[string]*PMAIntegration),
		entityFactories:     make(map[PMAEntityType]EntityFactory),
		logger:              logger,
	}

	// Register default entity types
	registry.registerDefaultTypes()

	return registry
}

// registerDefaultTypes registers all the built-in PMA types
func (r *PMATypeRegistry) registerDefaultTypes() {
	// Register entity types
	r.entityTypes[EntityTypeLight] = reflect.TypeOf((*PMALightEntity)(nil)).Elem()
	r.entityTypes[EntityTypeSwitch] = reflect.TypeOf((*PMASwitchEntity)(nil)).Elem()
	r.entityTypes[EntityTypeSensor] = reflect.TypeOf((*PMASensorEntity)(nil)).Elem()
	r.entityTypes[EntityTypeGeneric] = reflect.TypeOf((*PMABaseEntity)(nil)).Elem()

	// Register entity factories
	r.entityFactories[EntityTypeLight] = func() PMAEntity {
		return &PMALightEntity{PMABaseEntity: &PMABaseEntity{Type: EntityTypeLight}}
	}
	r.entityFactories[EntityTypeSwitch] = func() PMAEntity {
		return &PMASwitchEntity{PMABaseEntity: &PMABaseEntity{Type: EntityTypeSwitch}}
	}
	r.entityFactories[EntityTypeSensor] = func() PMAEntity {
		return &PMASensorEntity{PMABaseEntity: &PMABaseEntity{Type: EntityTypeSensor}}
	}
	r.entityFactories[EntityTypeGeneric] = func() PMAEntity {
		return &PMABaseEntity{Type: EntityTypeGeneric}
	}
}

// Type Discovery Methods

// GetSupportedEntityTypes returns all supported entity types
func (r *PMATypeRegistry) GetSupportedEntityTypes() []PMAEntityType {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	types := make([]PMAEntityType, 0, len(r.entityTypes))
	for entityType := range r.entityTypes {
		types = append(types, entityType)
	}
	return types
}

// GetEntityTypeInfo returns information about a specific entity type
func (r *PMATypeRegistry) GetEntityTypeInfo(entityType PMAEntityType) (*EntityTypeInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	reflectType, exists := r.entityTypes[entityType]
	if !exists {
		return nil, fmt.Errorf("entity type %s not found", entityType)
	}

	return &EntityTypeInfo{
		Type:         entityType,
		ReflectType:  reflectType,
		Name:         string(entityType),
		Description:  r.getEntityTypeDescription(entityType),
		Capabilities: r.getDefaultCapabilities(entityType),
		Actions:      r.getDefaultActions(entityType),
	}, nil
}

// EntityTypeInfo contains information about a registered entity type
type EntityTypeInfo struct {
	Type         PMAEntityType   `json:"type"`
	ReflectType  reflect.Type    `json:"-"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Capabilities []PMACapability `json:"capabilities"`
	Actions      []string        `json:"actions"`
}

// Factory Methods

// CreateEntity creates a new entity of the specified type
func (r *PMATypeRegistry) CreateEntity(entityType PMAEntityType) (PMAEntity, error) {
	r.mutex.RLock()
	factory, exists := r.entityFactories[entityType]
	r.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no factory found for entity type %s", entityType)
	}

	entity := factory()

	// Set default metadata
	entity.(*PMABaseEntity).Metadata = &PMAMetadata{
		Source:       SourcePMA,
		LastSynced:   time.Now(),
		QualityScore: 1.0,
	}

	return entity, nil
}

// CreateRoom creates a new PMA room
func (r *PMATypeRegistry) CreateRoom(id, name string) *PMARoom {
	now := time.Now()
	return &PMARoom{
		ID:        id,
		Name:      name,
		EntityIDs: make([]string, 0),
		Children:  make([]string, 0),
		Metadata: &PMAMetadata{
			Source:       SourcePMA,
			LastSynced:   now,
			QualityScore: 1.0,
		},
		Attributes: make(map[string]interface{}),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateArea creates a new PMA area
func (r *PMATypeRegistry) CreateArea(id, name string) *PMAArea {
	now := time.Now()
	return &PMAArea{
		ID:        id,
		Name:      name,
		RoomIDs:   make([]string, 0),
		EntityIDs: make([]string, 0),
		Metadata: &PMAMetadata{
			Source:       SourcePMA,
			LastSynced:   now,
			QualityScore: 1.0,
		},
		Attributes: make(map[string]interface{}),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Registration Methods

// RegisterEntityType registers a new entity type
func (r *PMATypeRegistry) RegisterEntityType(entityType PMAEntityType, reflectType reflect.Type, factory EntityFactory) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.entityTypes[entityType]; exists {
		return fmt.Errorf("entity type %s already registered", entityType)
	}

	r.entityTypes[entityType] = reflectType
	r.entityFactories[entityType] = factory

	r.logger.WithFields(logrus.Fields{
		"entity_type":  entityType,
		"reflect_type": reflectType.Name(),
	}).Info("Registered new entity type")

	return nil
}

// SetAdapterRegistry sets the adapter registry
func (r *PMATypeRegistry) SetAdapterRegistry(registry AdapterRegistry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.adapterRegistry = registry
}

// SetEntityRegistry sets the entity registry
func (r *PMATypeRegistry) SetEntityRegistry(registry EntityRegistry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entityRegistry = registry
}

// SetConflictResolver sets the conflict resolver
func (r *PMATypeRegistry) SetConflictResolver(resolver ConflictResolver) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.conflictResolver = resolver
}

// Validation Methods

// ValidateEntity validates an entity against its type constraints
func (r *PMATypeRegistry) ValidateEntity(entity PMAEntity) []ValidationError {
	var errors []ValidationError

	// Basic validation
	if entity.GetID() == "" {
		errors = append(errors, ValidationError{
			Field:   "id",
			Code:    "REQUIRED",
			Message: "Entity ID is required",
		})
	}

	if entity.GetFriendlyName() == "" {
		errors = append(errors, ValidationError{
			Field:   "friendly_name",
			Code:    "REQUIRED",
			Message: "Entity friendly name is required",
		})
	}

	// Type-specific validation
	entityType := entity.GetType()
	if _, exists := r.entityTypes[entityType]; !exists {
		errors = append(errors, ValidationError{
			Field:   "type",
			Code:    "INVALID",
			Message: fmt.Sprintf("Unknown entity type: %s", entityType),
			Value:   entityType,
		})
	}

	// Capability validation
	capabilities := entity.GetCapabilities()
	validCapabilities := r.getValidCapabilities()
	for _, cap := range capabilities {
		if !r.isValidCapability(cap, validCapabilities) {
			errors = append(errors, ValidationError{
				Field:   "capabilities",
				Code:    "INVALID",
				Message: fmt.Sprintf("Invalid capability: %s", cap),
				Value:   cap,
			})
		}
	}

	return errors
}

// Conversion Helpers

// ConvertToType attempts to convert an entity to a specific type
func (r *PMATypeRegistry) ConvertToType(entity PMAEntity, targetType PMAEntityType) (PMAEntity, error) {
	if entity.GetType() == targetType {
		return entity, nil
	}

	// Create new entity of target type
	newEntity, err := r.CreateEntity(targetType)
	if err != nil {
		return nil, err
	}

	// Copy base properties
	base := newEntity.(*PMABaseEntity)
	base.ID = entity.GetID()
	base.FriendlyName = entity.GetFriendlyName()
	base.Icon = entity.GetIcon()
	base.State = entity.GetState()
	base.Attributes = entity.GetAttributes()
	base.LastUpdated = entity.GetLastUpdated()
	base.RoomID = entity.GetRoomID()
	base.AreaID = entity.GetAreaID()
	base.DeviceID = entity.GetDeviceID()
	base.Metadata = entity.GetMetadata()
	base.Available = entity.IsAvailable()

	// Update type and recalculate capabilities
	base.Type = targetType
	base.Capabilities = r.getDefaultCapabilities(targetType)

	return newEntity, nil
}

// Utility Methods

// getEntityTypeDescription returns a description for an entity type
func (r *PMATypeRegistry) getEntityTypeDescription(entityType PMAEntityType) string {
	descriptions := map[PMAEntityType]string{
		EntityTypeLight:        "Controllable light source with dimming and color capabilities",
		EntityTypeSwitch:       "Simple on/off controllable device",
		EntityTypeSensor:       "Read-only sensor providing measurements or state information",
		EntityTypeClimate:      "Climate control device with temperature and HVAC management",
		EntityTypeCover:        "Motorized cover with position control (blinds, curtains, garage doors)",
		EntityTypeCamera:       "Video camera with streaming and recording capabilities",
		EntityTypeLock:         "Electronic lock with secure control capabilities",
		EntityTypeFan:          "Controllable fan with speed and direction settings",
		EntityTypeMediaPlayer:  "Media playback device with volume and content control",
		EntityTypeBinarySensor: "Binary sensor providing true/false state information",
		EntityTypeDevice:       "Generic device with basic control capabilities",
		EntityTypeGeneric:      "Generic entity with basic PMA functionality",
	}

	if desc, exists := descriptions[entityType]; exists {
		return desc
	}
	return "Custom entity type"
}

// getDefaultCapabilities returns default capabilities for an entity type
func (r *PMATypeRegistry) getDefaultCapabilities(entityType PMAEntityType) []PMACapability {
	capabilities := map[PMAEntityType][]PMACapability{
		EntityTypeLight:        {CapabilityDimmable, CapabilityBrightness},
		EntityTypeSwitch:       {},
		EntityTypeSensor:       {CapabilityConnectivity},
		EntityTypeClimate:      {CapabilityTemperature, CapabilityHumidity},
		EntityTypeCover:        {CapabilityPosition},
		EntityTypeCamera:       {CapabilityRecording, CapabilityStreaming},
		EntityTypeLock:         {},
		EntityTypeFan:          {CapabilityDimmable},
		EntityTypeMediaPlayer:  {CapabilityVolume},
		EntityTypeBinarySensor: {CapabilityConnectivity},
		EntityTypeDevice:       {CapabilityConnectivity},
		EntityTypeGeneric:      {},
	}

	if caps, exists := capabilities[entityType]; exists {
		return caps
	}
	return []PMACapability{}
}

// getDefaultActions returns default actions for an entity type
func (r *PMATypeRegistry) getDefaultActions(entityType PMAEntityType) []string {
	actions := map[PMAEntityType][]string{
		EntityTypeLight:        {"turn_on", "turn_off", "toggle", "set_brightness"},
		EntityTypeSwitch:       {"turn_on", "turn_off", "toggle"},
		EntityTypeSensor:       {},
		EntityTypeClimate:      {"set_temperature", "set_hvac_mode"},
		EntityTypeCover:        {"open", "close", "set_position"},
		EntityTypeCamera:       {"start_recording", "stop_recording", "take_snapshot"},
		EntityTypeLock:         {"lock", "unlock"},
		EntityTypeFan:          {"turn_on", "turn_off", "set_speed"},
		EntityTypeMediaPlayer:  {"play", "pause", "stop", "set_volume"},
		EntityTypeBinarySensor: {},
		EntityTypeDevice:       {"turn_on", "turn_off"},
		EntityTypeGeneric:      {"turn_on", "turn_off"},
	}

	if acts, exists := actions[entityType]; exists {
		return acts
	}
	return []string{}
}

// getValidCapabilities returns all valid capabilities
func (r *PMATypeRegistry) getValidCapabilities() []PMACapability {
	return []PMACapability{
		CapabilityDimmable, CapabilityColorable, CapabilityTemperature,
		CapabilityHumidity, CapabilityPosition, CapabilityVolume,
		CapabilityBrightness, CapabilityMotion, CapabilityRecording,
		CapabilityStreaming, CapabilityNotification, CapabilityBattery,
		CapabilityConnectivity,
	}
}

// isValidCapability checks if a capability is valid
func (r *PMATypeRegistry) isValidCapability(capability PMACapability, validCapabilities []PMACapability) bool {
	for _, valid := range validCapabilities {
		if capability == valid {
			return true
		}
	}
	return false
}

// GetRegistryStats returns statistics about the registry
func (r *PMATypeRegistry) GetRegistryStats() *RegistryStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return &RegistryStats{
		RegisteredEntityTypes:  len(r.entityTypes),
		RegisteredRooms:        len(r.roomRegistry),
		RegisteredAreas:        len(r.areaRegistry),
		RegisteredScenes:       len(r.sceneRegistry),
		RegisteredAutomations:  len(r.automationRegistry),
		RegisteredIntegrations: len(r.integrationRegistry),
	}
}

// RegistryStats contains statistics about the type registry
type RegistryStats struct {
	RegisteredEntityTypes  int `json:"registered_entity_types"`
	RegisteredRooms        int `json:"registered_rooms"`
	RegisteredAreas        int `json:"registered_areas"`
	RegisteredScenes       int `json:"registered_scenes"`
	RegisteredAutomations  int `json:"registered_automations"`
	RegisteredIntegrations int `json:"registered_integrations"`
}
