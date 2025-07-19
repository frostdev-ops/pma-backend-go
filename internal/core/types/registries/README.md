# PMA Registry Infrastructure

This package implements the core registry infrastructure for the PMA (Personal Management Assistant) backend system. The registries manage adapters, entities, and conflict resolution between different data sources.

## Overview

The PMA system follows a strict architecture where all external data sources (HomeAssistant, Ring, Shelly, UPS, Network devices, etc.) must be converted to unified PMA types before any API interaction. The registry infrastructure ensures this conversion happens correctly and manages conflicts when multiple sources provide the same entity.

## Components

### 1. AdapterRegistry (`adapter_registry.go`)

Manages all registered adapters that convert external data sources to PMA types.

**Features:**
- Thread-safe registration/unregistration of adapters
- Lookup adapters by ID or source type
- Track adapter metrics and health
- Handle source type conflicts (one adapter per source type)

**Usage:**
```go
registry := registries.NewDefaultAdapterRegistry(logger)
err := registry.RegisterAdapter(myAdapter)
adapter, err := registry.GetAdapterBySource(types.SourceHomeAssistant)
```

### 2. EntityRegistry (`entity_registry.go`)

Manages all PMA entities across different sources with efficient indexing.

**Features:**
- Thread-safe entity storage with multiple indexes
- Lookup by ID, type, source, or room
- Search entities by name/ID
- Track entity availability and statistics

**Indexes:**
- By entity ID (primary)
- By entity type (for filtering)
- By source type (for source-specific operations)
- By room ID (for room-based queries)

**Usage:**
```go
registry := registries.NewDefaultEntityRegistry(logger)
err := registry.RegisterEntity(entity)
entities, err := registry.GetEntitiesByType(types.EntityTypeLight)
```

### 3. SourcePriorityManager (`source_priority_manager.go`)

Manages priority ordering between different data sources for conflict resolution.

**Default Priorities (lower number = higher priority):**
- HomeAssistant: 1 (highest - most comprehensive)
- Ring: 2 (security devices)
- Shelly: 3 (smart switches/devices)
- UPS: 4 (power management)
- Network: 5 (network devices)
- PMA: 10 (virtual/computed entities)

**Features:**
- Configurable priority values
- Thread-safe priority updates
- Priority-based override decisions
- Source comparison utilities

**Usage:**
```go
manager := registries.NewDefaultSourcePriorityManager(logger)
err := manager.SetSourcePriority(types.SourceRing, 1)
shouldOverride := manager.ShouldOverride(currentSource, newSource)
```

### 4. ConflictResolver (`conflict_resolver.go`)

Handles conflicts when multiple sources provide the same entity using sophisticated resolution strategies.

**Resolution Strategy:**
1. **Source Priority**: Use configured source priorities
2. **Availability**: Prefer available entities over unavailable ones
3. **Quality Score**: Use entity quality scores as tiebreakers
4. **Recency**: Prefer more recently updated entities
5. **Entity-Specific Rules**: Special handling for certain entity types

**Features:**
- Multi-entity conflict resolution
- Attribute merging from multiple sources
- Virtual entity creation for multi-source entities
- Entity-type specific resolution rules

**Usage:**
```go
resolver := registries.NewDefaultConflictResolver(priorityManager, logger)
resolvedEntity, err := resolver.ResolveEntityConflict(conflictingEntities)
mergedAttrs := resolver.MergeEntityAttributes(entities)
```

### 5. RegistryManager (`registry.go`)

Unified manager that coordinates all registries and provides high-level operations.

**Features:**
- Centralized registry management
- Entity registration with automatic conflict resolution
- Adapter synchronization with conflict handling
- Registry consistency validation
- Comprehensive statistics and monitoring

**Usage:**
```go
manager := registries.NewRegistryManager(logger)
err := manager.RegisterEntityWithConflictResolution(entity)
err := manager.SyncEntitiesFromAdapter("adapter-id")
stats := manager.GetAllRegistryStats()
```

## Architecture Principles

### Thread Safety
All registry implementations use `sync.RWMutex` for thread-safe operations:
- Read operations use `RLock()`
- Write operations use `Lock()`
- Consistent locking order prevents deadlocks

### Error Handling
- Custom error types for specific failure scenarios
- Descriptive error messages with context
- Non-blocking error handling where possible

### Performance
- Multiple indexes for O(1) lookups
- Efficient slice operations with pre-allocated capacity
- Minimal memory allocations in hot paths

### Extensibility
- Interface-based design for easy testing and mocking
- Factory functions for dependency injection
- Configurable behaviors through options

## Testing

The package includes comprehensive unit tests:

```bash
go test ./internal/core/types/registries/ -v
```

**Test Coverage:**
- All public methods tested
- Error conditions verified
- Thread safety confirmed
- Mock adapters and entities for isolation

## Integration

### With Core Services

```go
// Initialize registry manager
registryManager := registries.NewRegistryManager(logger)

// Register adapters
haAdapter := homeassistant.NewAdapter(config)
registryManager.GetAdapterRegistry().RegisterAdapter(haAdapter)

// Sync entities from all adapters
for _, adapter := range registryManager.GetAdapterRegistry().GetAllAdapters() {
    err := registryManager.SyncEntitiesFromAdapter(adapter.GetID())
    if err != nil {
        logger.Errorf("Sync failed for %s: %v", adapter.GetID(), err)
    }
}
```

### With API Handlers

```go
// Get entities for API response
entities, err := registryManager.GetEntityRegistry().GetEntitiesByRoom(roomID)
if err != nil {
    return http.StatusInternalServerError, err
}

// Search entities
results, err := registryManager.GetEntityRegistry().SearchEntities(query)
```

## Configuration

### Source Priorities

Source priorities can be configured at runtime:

```go
priorityManager := registryManager.GetPriorityManager()

// Set custom priorities
err := priorityManager.SetMultiplePriorities(map[types.PMASourceType]int{
    types.SourceHomeAssistant: 1,
    types.SourceRing:          2,
    types.SourceShelly:        3,
})

// Reset to defaults
priorityManager.ResetToDefaults()
```

### Conflict Resolution

Conflict resolution behavior can be customized by implementing custom `ConflictResolver`:

```go
type CustomConflictResolver struct {
    *registries.DefaultConflictResolver
    // Custom fields
}

func (r *CustomConflictResolver) ResolveEntityConflict(entities []types.PMAEntity) (types.PMAEntity, error) {
    // Custom resolution logic
    return r.DefaultConflictResolver.ResolveEntityConflict(entities)
}
```

## Monitoring

### Registry Statistics

```go
stats := registryManager.GetAllRegistryStats()
// Returns comprehensive metrics about all registries
```

### Consistency Validation

```go
issues := registryManager.ValidateRegistryConsistency()
if len(issues) > 0 {
    for _, issue := range issues {
        logger.Warn(issue)
    }
}
```

## Future Enhancements

1. **Persistence**: Save/restore registry state to/from database
2. **Events**: Entity change notifications and event streaming
3. **Caching**: LRU cache for frequently accessed entities
4. **Metrics**: Prometheus metrics for monitoring
5. **Backup**: Registry state backup and recovery
6. **Clustering**: Distributed registry for multi-node deployments

## Dependencies

- `github.com/sirupsen/logrus`: Structured logging
- `github.com/stretchr/testify`: Testing framework
- Standard library: `sync`, `time`, `fmt`, etc.

## License

This code is part of the PMA backend system and follows the project's licensing terms. 