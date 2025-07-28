package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisEntityCache provides Redis-backed entity caching
type RedisEntityCache struct {
	client    *redis.Client
	config    *config.RedisConfig
	logger    *logrus.Logger
	keyPrefix string
	ttl       time.Duration
}

// NewRedisEntityCache creates a new Redis entity cache
func NewRedisEntityCache(cfg *config.Config, logger *logrus.Logger) (*RedisEntityCache, error) {
	if !cfg.Redis.Enabled {
		return nil, fmt.Errorf("Redis is not enabled in configuration")
	}

	// Parse TTL
	ttl, err := time.ParseDuration(cfg.Redis.EntityCacheTTL)
	if err != nil {
		ttl = 24 * time.Hour // Default to 24 hours
		logger.WithError(err).Warn("Invalid entity_cache_ttl, using default 24h")
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host":       cfg.Redis.Host,
		"port":       cfg.Redis.Port,
		"db":         cfg.Redis.DB,
		"key_prefix": cfg.Redis.KeyPrefix,
		"cache_ttl":  ttl,
	}).Info("Redis entity cache initialized successfully")

	return &RedisEntityCache{
		client:    rdb,
		config:    &cfg.Redis,
		logger:    logger,
		keyPrefix: cfg.Redis.KeyPrefix + "entity:",
		ttl:       ttl,
	}, nil
}

// SetEntity stores an entity in Redis cache
func (r *RedisEntityCache) SetEntity(ctx context.Context, entityID string, entity types.PMAEntity) error {
	key := r.keyPrefix + entityID

	// Serialize entity to JSON
	data, err := json.Marshal(entity)
	if err != nil {
		r.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to serialize entity for Redis")
		return fmt.Errorf("failed to serialize entity: %w", err)
	}

	// Store in Redis with TTL
	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		r.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to store entity in Redis")
		return fmt.Errorf("failed to store entity in Redis: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"key":       key,
		"ttl":       r.ttl,
	}).Debug("Entity stored in Redis cache")

	return nil
}

// GetEntity retrieves an entity from Redis cache
func (r *RedisEntityCache) GetEntity(ctx context.Context, entityID string) (types.PMAEntity, error) {
	key := r.keyPrefix + entityID

	// Get data from Redis
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("entity not found in cache: %s", entityID)
		}
		r.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to retrieve entity from Redis")
		return nil, fmt.Errorf("failed to retrieve entity from Redis: %w", err)
	}

	// Deserialize JSON to entity
	var rawEntity map[string]interface{}
	if err := json.Unmarshal([]byte(data), &rawEntity); err != nil {
		r.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to deserialize raw entity from Redis")
		return nil, fmt.Errorf("failed to deserialize entity: %w", err)
	}

	// Determine entity type and deserialize accordingly
	var entity types.PMAEntity
	if entityType, ok := rawEntity["type"].(string); ok {
		switch types.PMAEntityType(entityType) {
		case types.EntityTypeSwitch:
			var switchEntity types.PMASwitchEntity
			if err := json.Unmarshal([]byte(data), &switchEntity); err == nil {
				entity = &switchEntity
			}
		case types.EntityTypeLight:
			var lightEntity types.PMALightEntity
			if err := json.Unmarshal([]byte(data), &lightEntity); err == nil {
				entity = &lightEntity
			}
		case types.EntityTypeSensor:
			var sensorEntity types.PMASensorEntity
			if err := json.Unmarshal([]byte(data), &sensorEntity); err == nil {
				entity = &sensorEntity
			}
		default:
			// Fall back to base entity for unknown types
			var baseEntity types.PMABaseEntity
			if err := json.Unmarshal([]byte(data), &baseEntity); err == nil {
				entity = &baseEntity
			}
		}
	}

	// If type-specific deserialization failed, use base entity as fallback
	if entity == nil {
		var baseEntity types.PMABaseEntity
		if err := json.Unmarshal([]byte(data), &baseEntity); err != nil {
			r.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to deserialize entity from Redis")
			return nil, fmt.Errorf("failed to deserialize entity: %w", err)
		}
		entity = &baseEntity
	}

	r.logger.WithFields(logrus.Fields{
		"entity_id":   entityID,
		"entity_type": entity.GetType(),
	}).Debug("Entity retrieved from Redis cache with correct type")

	return entity, nil
}

// DeleteEntity removes an entity from Redis cache
func (r *RedisEntityCache) DeleteEntity(ctx context.Context, entityID string) error {
	key := r.keyPrefix + entityID

	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to delete entity from Redis")
		return fmt.Errorf("failed to delete entity from Redis: %w", err)
	}

	r.logger.WithField("entity_id", entityID).Debug("Entity deleted from Redis cache")
	return nil
}

// GetAllEntityIDs returns all entity IDs in the cache
func (r *RedisEntityCache) GetAllEntityIDs(ctx context.Context) ([]string, error) {
	pattern := r.keyPrefix + "*"

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		r.logger.WithError(err).Error("Failed to get entity keys from Redis")
		return nil, fmt.Errorf("failed to get entity keys: %w", err)
	}

	// Strip the prefix from keys to get entity IDs
	entityIDs := make([]string, len(keys))
	for i, key := range keys {
		entityIDs[i] = key[len(r.keyPrefix):]
	}

	r.logger.WithField("count", len(entityIDs)).Debug("Retrieved all entity IDs from Redis cache")
	return entityIDs, nil
}

// GetCacheSize returns the number of entities in cache
func (r *RedisEntityCache) GetCacheSize(ctx context.Context) (int, error) {
	pattern := r.keyPrefix + "*"

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		r.logger.WithError(err).Error("Failed to get cache size from Redis")
		return 0, fmt.Errorf("failed to get cache size: %w", err)
	}

	return len(keys), nil
}

// ClearCache removes all entities from cache
func (r *RedisEntityCache) ClearCache(ctx context.Context) error {
	pattern := r.keyPrefix + "*"

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for clearing cache: %w", err)
	}

	if len(keys) > 0 {
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
	}

	r.logger.WithField("cleared_count", len(keys)).Info("Redis entity cache cleared")
	return nil
}

// Close closes the Redis connection
func (r *RedisEntityCache) Close() error {
	return r.client.Close()
}

// Health checks Redis connection health
func (r *RedisEntityCache) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
