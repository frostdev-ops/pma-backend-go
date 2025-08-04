# Unified Entity Service Architecture

## Overview

The Unified Entity Service is the core component of the PMA Backend Go system, providing a centralized abstraction layer for managing smart home devices across multiple platforms and protocols. It implements a sophisticated type system that unifies entities from different sources (Home Assistant, Ring, Shelly, UPS, etc.) into a consistent interface.

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Unified Entity Service                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Type Registry│  │ Adapter     │  │ Entity      │       │
│  │             │  │ Registry    │  │ Registry    │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Conflict    │  │ Source      │  │ Priority    │       │
│  │ Resolver    │  │ Priority    │  │ Manager     │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Device Adapters                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Home        │  │ Ring        │  │ Shelly      │       │
│  │ Assistant   │  │ Adapter     │  │ Adapter     │       │
│  │ Adapter     │  │             │  │             │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ UPS         │  │ Network     │  │ Bluetooth   │       │
│  │ Adapter     │  │ Adapter     │  │ Adapter     │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

### Type System

The service implements a comprehensive type system that defines:

#### Entity Types
```go
type EntityType string

const (
    EntityTypeLight    EntityType = "light"
    EntityTypeSwitch   EntityType = "switch"
    EntityTypeSensor   EntityType = "sensor"
    EntityTypeClimate  EntityType = "climate"
    EntityTypeCover    EntityType = "cover"
    EntityTypeLock     EntityType = "lock"
    EntityTypeCamera   EntityType = "camera"
    EntityTypeMedia    EntityType = "media"
    EntityTypeVacuum   EntityType = "vacuum"
    EntityTypeFan      EntityType = "fan"
    EntityTypeBinary   EntityType = "binary_sensor"
    EntityTypeNumber   EntityType = "number"
    EntityTypeSelect   EntityType = "select"
    EntityTypeText     EntityType = "text"
    EntityTypeButton   EntityType = "button"
    EntityTypeScene    EntityType = "scene"
    EntityTypeScript   EntityType = "script"
    EntityTypeAutomation EntityType = "automation"
)
```

#### Capabilities
```go
type Capability string

const (
    CapabilityOnOff        Capability = "on_off"
    CapabilityBrightness   Capability = "brightness"
    CapabilityColor        Capability = "color"
    CapabilityColorTemp    Capability = "color_temp"
    CapabilityEffect       Capability = "effect"
    CapabilityPosition     Capability = "position"
    CapabilityTilt        Capability = "tilt"
    CapabilityTemperature  Capability = "temperature"
    CapabilityHumidity     Capability = "humidity"
    CapabilityPressure     Capability = "pressure"
    CapabilityBattery      Capability = "battery"
    CapabilitySignal       Capability = "signal"
    CapabilityMotion       Capability = "motion"
    CapabilityPresence     Capability = "presence"
    CapabilityOccupancy    Capability = "occupancy"
    CapabilitySmoke        Capability = "smoke"
    CapabilityCarbonMonoxide Capability = "carbon_monoxide"
    CapabilityWater        Capability = "water"
    CapabilityGas          Capability = "gas"
    CapabilityVibration    Capability = "vibration"
    CapabilityLight        Capability = "light"
    CapabilitySound        Capability = "sound"
    CapabilityTamper       Capability = "tamper"
    CapabilityLock         Capability = "lock"
    CapabilityUnlock       Capability = "unlock"
    CapabilityOpen         Capability = "open"
    CapabilityClose        Capability = "close"
    CapabilityStop         Capability = "stop"
    CapabilitySetPosition  Capability = "set_position"
    CapabilitySetTilt      Capability = "set_tilt"
    CapabilitySetTemperature Capability = "set_temperature"
    CapabilitySetMode      Capability = "set_mode"
    CapabilitySetFanSpeed  Capability = "set_fan_speed"
    CapabilitySetHumidity  Capability = "set_humidity"
    CapabilitySetPreset    Capability = "set_preset"
    CapabilityStart        Capability = "start"
    CapabilityPause        Capability = "pause"
    CapabilityResume       Capability = "resume"
    CapabilityReturn       Capability = "return"
    CapabilityLocate       Capability = "locate"
    CapabilityCleanSpot    Capability = "clean_spot"
    CapabilityCleanArea    Capability = "clean_area"
    CapabilitySetFanMode   Capability = "set_fan_mode"
    CapabilitySetSwingMode Capability = "set_swing_mode"
    CapabilitySetAuxHeat   Capability = "set_aux_heat"
    CapabilitySetAwayMode  Capability = "set_away_mode"
    CapabilitySetHoldMode  Capability = "set_hold_mode"
    CapabilitySetPresetMode Capability = "set_preset_mode"
)
```

#### Unified Entity Structure
```go
type UnifiedEntity struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Type        EntityType             `json:"type"`
    State       string                 `json:"state"`
    Attributes  map[string]interface{} `json:"attributes"`
    Capabilities []Capability          `json:"capabilities"`
    RoomID      *string               `json:"room_id,omitempty"`
    AreaID      *string               `json:"area_id,omitempty"`
    Source      string                `json:"source"`
    LastUpdated time.Time             `json:"last_updated"`
    Metadata    EntityMetadata        `json:"metadata"`
}

type EntityMetadata struct {
    Model           string            `json:"model,omitempty"`
    Manufacturer    string            `json:"manufacturer,omitempty"`
    FirmwareVersion string            `json:"firmware_version,omitempty"`
    HardwareVersion string            `json:"hardware_version,omitempty"`
    SerialNumber    string            `json:"serial_number,omitempty"`
    MACAddress      string            `json:"mac_address,omitempty"`
    IPAddress       string            `json:"ip_address,omitempty"`
    LastSeen        time.Time         `json:"last_seen"`
    Platform        PlatformMetadata  `json:"platform"`
}

type PlatformMetadata struct {
    OriginalID    string                 `json:"original_id"`
    OriginalType  string                 `json:"original_type"`
    Platform      string                 `json:"platform"`
    ExtraData     map[string]interface{} `json:"extra_data,omitempty"`
}
```

## Service Components

### 1. Type Registry

The Type Registry manages entity type definitions and their relationships:

```go
type TypeRegistry struct {
    types map[EntityType]*EntityTypeDefinition
    capabilities map[Capability]*CapabilityDefinition
    mutex sync.RWMutex
}

type EntityTypeDefinition struct {
    Type         EntityType
    Name         string
    Description  string
    Capabilities []Capability
    Attributes   []string
    Actions      []string
    Category     string
}

type CapabilityDefinition struct {
    Capability   Capability
    Name         string
    Description  string
    Type         string
    Unit         string
    MinValue     interface{}
    MaxValue     interface{}
    Step         interface{}
    Options      []string
}
```

**Key Features:**
- Type validation and normalization
- Capability mapping and validation
- Attribute schema validation
- Action availability checking

### 2. Adapter Registry

The Adapter Registry manages device adapters for different platforms:

```go
type AdapterRegistry struct {
    adapters map[string]DeviceAdapter
    mutex    sync.RWMutex
}

type DeviceAdapter interface {
    // Lifecycle
    Start() error
    Stop() error
    IsRunning() bool
    
    // Device Management
    GetDevices() ([]UnifiedEntity, error)
    GetDevice(id string) (*UnifiedEntity, error)
    DiscoverDevices() error
    
    // Action Execution
    ExecuteAction(deviceID string, action string, parameters map[string]interface{}) error
    
    // Event Handling
    SubscribeToEvents(callback func(EntityEvent)) error
    UnsubscribeFromEvents() error
    
    // Health Monitoring
    GetHealth() (*AdapterHealth, error)
    GetStatistics() (*AdapterStatistics, error)
}

type AdapterHealth struct {
    Status        string    `json:"status"`
    LastCheck     time.Time `json:"last_check"`
    ErrorCount    int       `json:"error_count"`
    ResponseTime  time.Duration `json:"response_time"`
    Connected     bool      `json:"connected"`
    Message       string    `json:"message,omitempty"`
}

type AdapterStatistics struct {
    TotalDevices    int     `json:"total_devices"`
    OnlineDevices   int     `json:"online_devices"`
    OfflineDevices  int     `json:"offline_devices"`
    ErrorRate       float64 `json:"error_rate"`
    AvgResponseTime time.Duration `json:"avg_response_time"`
}
```

### 3. Entity Registry

The Entity Registry maintains the unified entity collection:

```go
type EntityRegistry struct {
    entities map[string]*UnifiedEntity
    mutex    sync.RWMutex
    cache    *EntityCache
}

type EntityCache struct {
    entities    map[string]*UnifiedEntity
    byRoom      map[string][]*UnifiedEntity
    byType      map[EntityType][]*UnifiedEntity
    byCapability map[Capability][]*UnifiedEntity
    mutex       sync.RWMutex
    ttl         time.Duration
    lastUpdate  time.Time
}
```

**Key Features:**
- In-memory entity storage with caching
- Indexed lookups by room, type, and capability
- Automatic cache invalidation
- Thread-safe concurrent access

### 4. Conflict Resolver

The Conflict Resolver handles entity conflicts when the same device is discovered by multiple adapters:

```go
type ConflictResolver struct {
    strategies map[string]ConflictResolutionStrategy
    rules      []ConflictResolutionRule
}

type ConflictResolutionStrategy string

const (
    StrategyFirstWins    ConflictResolutionStrategy = "first_wins"
    StrategyLastWins     ConflictResolutionStrategy = "last_wins"
    StrategyPriority     ConflictResolutionStrategy = "priority"
    StrategyMerge        ConflictResolutionStrategy = "merge"
    StrategyUserChoice   ConflictResolutionStrategy = "user_choice"
)

type ConflictResolutionRule struct {
    ID          string
    Name        string
    Description string
    Conditions  []ConflictCondition
    Strategy    ConflictResolutionStrategy
    Priority    int
    Enabled     bool
}

type ConflictCondition struct {
    Field    string
    Operator  string
    Value     interface{}
}
```

**Resolution Strategies:**
- **First Wins**: Keep the first discovered entity
- **Last Wins**: Replace with the most recent entity
- **Priority**: Use priority-based selection
- **Merge**: Combine attributes from multiple sources
- **User Choice**: Allow manual resolution

### 5. Source Priority Manager

The Source Priority Manager determines which adapter takes precedence:

```go
type SourcePriorityManager struct {
    priorities map[string]int
    rules      []PriorityRule
    mutex      sync.RWMutex
}

type PriorityRule struct {
    ID          string
    Name        string
    Description string
    Source      string
    Priority    int
    Conditions  []PriorityCondition
    Enabled     bool
}

type PriorityCondition struct {
    EntityType  EntityType
    Capability  Capability
    RoomID      string
    TimeRange   TimeRange
}
```

**Default Priorities:**
1. **Home Assistant** (Priority: 100) - Primary smart home platform
2. **Ring** (Priority: 90) - Security and camera devices
3. **Shelly** (Priority: 80) - IoT devices
4. **UPS** (Priority: 70) - Power monitoring
5. **Network** (Priority: 60) - Network devices
6. **Bluetooth** (Priority: 50) - Bluetooth devices

## API Interface

### Core Methods

```go
type UnifiedEntityService struct {
    typeRegistry     *TypeRegistry
    adapterRegistry  *AdapterRegistry
    entityRegistry   *EntityRegistry
    conflictResolver *ConflictResolver
    priorityManager  *SourcePriorityManager
    logger          *logrus.Logger
}

// Entity Management
func (s *UnifiedEntityService) GetEntities(filters EntityFilters) ([]UnifiedEntity, error)
func (s *UnifiedEntityService) GetEntity(id string) (*UnifiedEntity, error)
func (s *UnifiedEntityService) UpdateEntity(id string, updates EntityUpdates) error
func (s *UnifiedEntityService) DeleteEntity(id string) error

// Action Execution
func (s *UnifiedEntityService) ExecuteAction(entityID string, action string, parameters map[string]interface{}) error
func (s *UnifiedEntityService) ExecuteActions(actions []EntityAction) error

// Synchronization
func (s *UnifiedEntityService) SyncFromAllSources() error
func (s *UnifiedEntityService) SyncFromSource(source string) error
func (s *UnifiedEntityService) GetSyncStatus() (*SyncStatus, error)

// Discovery
func (s *UnifiedEntityService) DiscoverDevices() error
func (s *UnifiedEntityService) GetDiscoveryStatus() (*DiscoveryStatus, error)

// Health Monitoring
func (s *UnifiedEntityService) GetHealth() (*ServiceHealth, error)
func (s *UnifiedEntityService) GetStatistics() (*ServiceStatistics, error)
```

### Filtering and Querying

```go
type EntityFilters struct {
    IDs           []string
    Types         []EntityType
    States        []string
    Capabilities  []Capability
    RoomIDs       []string
    AreaIDs       []string
    Sources       []string
    Search        string
    Limit         int
    Offset        int
    SortBy        string
    SortOrder     string
}

type EntityUpdates struct {
    Name          *string
    RoomID        *string
    AreaID        *string
    Attributes    map[string]interface{}
    Metadata      *EntityMetadata
}

type EntityAction struct {
    EntityID   string                 `json:"entity_id"`
    Action     string                 `json:"action"`
    Parameters map[string]interface{} `json:"parameters,omitempty"`
    Timeout    time.Duration         `json:"timeout,omitempty"`
    Retries    int                   `json:"retries,omitempty"`
}
```

## Event System

### Entity Events

```go
type EntityEvent struct {
    Type      EntityEventType `json:"type"`
    EntityID  string         `json:"entity_id"`
    OldState  *UnifiedEntity `json:"old_state,omitempty"`
    NewState  *UnifiedEntity `json:"new_state,omitempty"`
    Source    string         `json:"source"`
    Timestamp time.Time      `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type EntityEventType string

const (
    EventTypeEntityAdded    EntityEventType = "entity_added"
    EventTypeEntityUpdated  EntityEventType = "entity_updated"
    EventTypeEntityRemoved  EntityEventType = "entity_removed"
    EventTypeStateChanged   EntityEventType = "state_changed"
    EventTypeActionExecuted EntityEventType = "action_executed"
    EventTypeActionFailed   EntityEventType = "action_failed"
    EventTypeSyncStarted    EntityEventType = "sync_started"
    EventTypeSyncCompleted  EntityEventType = "sync_completed"
    EventTypeSyncFailed     EntityEventType = "sync_failed"
)
```

### Event Subscription

```go
type EventSubscription struct {
    ID       string
    Filters  EventFilters
    Callback func(EntityEvent)
    Active   bool
}

type EventFilters struct {
    EntityIDs []string
    EventTypes []EntityEventType
    Sources   []string
    RoomIDs   []string
    AreaIDs   []string
}
```

## Performance Optimizations

### 1. Caching Strategy

```go
type EntityCache struct {
    entities    map[string]*UnifiedEntity
    byRoom      map[string][]*UnifiedEntity
    byType      map[EntityType][]*UnifiedEntity
    byCapability map[Capability][]*UnifiedEntity
    mutex       sync.RWMutex
    ttl         time.Duration
    lastUpdate  time.Time
    maxSize     int
    evictionPolicy string
}
```

**Cache Features:**
- LRU eviction policy
- Configurable TTL
- Indexed lookups
- Memory usage monitoring
- Automatic cleanup

### 2. Batch Operations

```go
func (s *UnifiedEntityService) BatchUpdate(updates []EntityUpdate) error
func (s *UnifiedEntityService) BatchExecuteActions(actions []EntityAction) error
func (s *UnifiedEntityService) BatchSync(sources []string) error
```

### 3. Connection Pooling

```go
type AdapterConnectionPool struct {
    adapters    map[string]*AdapterConnection
    maxConnections int
    timeout     time.Duration
    mutex       sync.RWMutex
}

type AdapterConnection struct {
    adapter     DeviceAdapter
    lastUsed    time.Time
    inUse       bool
    errorCount  int
    health      *AdapterHealth
}
```

## Error Handling

### Error Types

```go
type EntityError struct {
    Code        string
    Message     string
    EntityID    string
    Action      string
    Source      string
    Timestamp   time.Time
    Details     map[string]interface{}
}

const (
    ErrorCodeEntityNotFound     = "ENTITY_NOT_FOUND"
    ErrorCodeActionNotSupported = "ACTION_NOT_SUPPORTED"
    ErrorCodeAdapterUnavailable = "ADAPTER_UNAVAILABLE"
    ErrorCodeTimeout           = "TIMEOUT"
    ErrorCodeInvalidParameters = "INVALID_PARAMETERS"
    ErrorCodeConflict          = "CONFLICT"
    ErrorCodeSyncFailed        = "SYNC_FAILED"
)
```

### Retry Logic

```go
type RetryConfig struct {
    MaxAttempts     int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffFactor   float64
    RetryableErrors []string
}

func (s *UnifiedEntityService) executeWithRetry(action func() error, config RetryConfig) error
```

## Monitoring and Metrics

### Service Metrics

```go
type ServiceMetrics struct {
    TotalEntities     int64
    OnlineEntities    int64
    OfflineEntities   int64
    SyncDuration      time.Duration
    LastSyncTime      time.Time
    ActionSuccessRate float64
    AvgResponseTime   time.Duration
    ErrorCount        int64
    CacheHitRate      float64
    MemoryUsage       int64
}

type AdapterMetrics struct {
    AdapterName       string
    TotalDevices      int64
    OnlineDevices     int64
    OfflineDevices    int64
    ErrorRate         float64
    AvgResponseTime   time.Duration
    LastSeen          time.Time
    SyncStatus        string
}
```

### Health Checks

```go
type ServiceHealth struct {
    Status        string
    Timestamp     time.Time
    Adapters      map[string]*AdapterHealth
    CacheStatus   string
    MemoryUsage   float64
    ErrorCount    int64
    LastSync      time.Time
    Message       string
}
```

## Configuration

### Service Configuration

```yaml
unified_entity_service:
  enabled: true
  cache:
    ttl: "5m"
    max_size: 10000
    eviction_policy: "lru"
  sync:
    interval: "1m"
    batch_size: 100
    timeout: "30s"
    retry_attempts: 3
  conflict_resolution:
    default_strategy: "priority"
    rules:
      - name: "Home Assistant Priority"
        conditions:
          - field: "source"
            operator: "equals"
            value: "home_assistant"
        strategy: "priority"
        priority: 100
  adapters:
    home_assistant:
      enabled: true
      priority: 100
      timeout: "10s"
    ring:
      enabled: true
      priority: 90
      timeout: "15s"
    shelly:
      enabled: true
      priority: 80
      timeout: "5s"
```

## Usage Examples

### Basic Entity Operations

```go
// Get all entities
entities, err := service.GetEntities(EntityFilters{})

// Get entities by room
entities, err := service.GetEntities(EntityFilters{
    RoomIDs: []string{"living_room"},
})

// Get entities by capability
entities, err := service.GetEntities(EntityFilters{
    Capabilities: []Capability{CapabilityBrightness},
})

// Execute action
err := service.ExecuteAction("light.living_room", "turn_on", map[string]interface{}{
    "brightness": 128,
})

// Batch execute actions
actions := []EntityAction{
    {EntityID: "light.living_room", Action: "turn_on"},
    {EntityID: "light.kitchen", Action: "turn_off"},
}
err := service.ExecuteActions(actions)
```

### Event Handling

```go
// Subscribe to entity events
subscription := &EventSubscription{
    Filters: EventFilters{
        EntityIDs: []string{"light.living_room"},
        EventTypes: []EntityEventType{EventTypeStateChanged},
    },
    Callback: func(event EntityEvent) {
        log.Printf("Entity %s state changed: %s", event.EntityID, event.NewState.State)
    },
}

service.SubscribeToEvents(subscription)
```

### Health Monitoring

```go
// Get service health
health, err := service.GetHealth()
if err != nil {
    log.Printf("Service health check failed: %v", err)
}

// Get adapter statistics
stats, err := service.GetStatistics()
for adapter, metrics := range stats.Adapters {
    log.Printf("Adapter %s: %d devices, %.2f%% error rate", 
        adapter, metrics.TotalDevices, metrics.ErrorRate*100)
}
```

## Integration Points

### 1. API Layer Integration

The Unified Entity Service integrates with the API layer through handlers:

```go
// API Handler integration
func (h *Handlers) GetEntities(c *gin.Context) {
    filters := parseEntityFilters(c)
    entities, err := h.unifiedService.GetEntities(filters)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": entities})
}
```

### 2. WebSocket Integration

Real-time entity updates are broadcast through WebSocket:

```go
// WebSocket event broadcasting
func (s *UnifiedEntityService) broadcastEntityEvent(event EntityEvent) {
    s.wsHub.Broadcast("entity_event", event)
}
```

### 3. Automation Integration

The service provides entity data to the automation engine:

```go
// Automation engine integration
func (s *UnifiedEntityService) GetEntitiesForAutomation(rule *AutomationRule) ([]UnifiedEntity, error) {
    filters := buildFiltersFromRule(rule)
    return s.GetEntities(filters)
}
```

## Testing

### Unit Tests

```go
func TestUnifiedEntityService_GetEntities(t *testing.T) {
    service := NewUnifiedEntityService(config)
    
    // Test basic entity retrieval
    entities, err := service.GetEntities(EntityFilters{})
    assert.NoError(t, err)
    assert.NotEmpty(t, entities)
    
    // Test filtering
    entities, err = service.GetEntities(EntityFilters{
        Types: []EntityType{EntityTypeLight},
    })
    assert.NoError(t, err)
    for _, entity := range entities {
        assert.Equal(t, EntityTypeLight, entity.Type)
    }
}
```

### Integration Tests

```go
func TestUnifiedEntityService_Integration(t *testing.T) {
    // Test with real adapters
    service := NewUnifiedEntityService(config)
    
    // Start service
    err := service.Start()
    assert.NoError(t, err)
    defer service.Stop()
    
    // Test entity discovery
    err = service.DiscoverDevices()
    assert.NoError(t, err)
    
    // Test action execution
    err = service.ExecuteAction("test.light", "turn_on", nil)
    assert.NoError(t, err)
}
```

## Performance Benchmarks

### Entity Retrieval Performance

```
BenchmarkGetEntities_NoFilter-8         1000000    1200 ns/op
BenchmarkGetEntities_WithFilter-8        500000     2500 ns/op
BenchmarkGetEntities_ComplexFilter-8     200000     8000 ns/op
```

### Action Execution Performance

```
BenchmarkExecuteAction_Single-8          100000     15000 ns/op
BenchmarkExecuteAction_Batch-8           50000      25000 ns/op
BenchmarkExecuteAction_Parallel-8        200000     8000 ns/op
```

### Memory Usage

```
BenchmarkMemoryUsage_1000Entities-8     1000       2.5 MB
BenchmarkMemoryUsage_10000Entities-8    100        25 MB
BenchmarkMemoryUsage_100000Entities-8   10         250 MB
```

## Troubleshooting

### Common Issues

1. **Entity Not Found**
   - Check if adapter is running
   - Verify entity ID format
   - Check sync status

2. **Action Execution Failed**
   - Verify entity capabilities
   - Check adapter connectivity
   - Review action parameters

3. **Sync Issues**
   - Check adapter health
   - Verify network connectivity
   - Review sync configuration

4. **Performance Issues**
   - Monitor cache hit rates
   - Check memory usage
   - Review batch sizes

### Debug Commands

```bash
# Check service health
curl http://localhost:3001/api/v1/system/health

# Get entity statistics
curl http://localhost:3001/api/v1/entities/stats

# Check adapter status
curl http://localhost:3001/api/v1/adapters/status

# Force sync
curl -X POST http://localhost:3001/api/v1/entities/sync
```

## Future Enhancements

### Planned Features

1. **Advanced Conflict Resolution**
   - Machine learning-based conflict resolution
   - User preference learning
   - Automatic conflict detection

2. **Enhanced Caching**
   - Redis-based distributed caching
   - Predictive caching
   - Cache warming strategies

3. **Performance Optimizations**
   - Connection pooling improvements
   - Batch operation enhancements
   - Memory usage optimization

4. **Monitoring Enhancements**
   - Prometheus metrics integration
   - Advanced health checks
   - Performance dashboards

---

**Unified Entity Service** - Centralized smart home device management with multi-platform support. 