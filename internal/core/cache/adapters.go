package cache

import (
	"fmt"
	"time"
)

// MockCache provides a simple mock cache for testing and examples
type MockCache struct {
	*BaseCache
	refreshData map[string]interface{}
}

// NewMockCache creates a new mock cache
func NewMockCache(name string, cacheType CacheType) *MockCache {
	return &MockCache{
		BaseCache:   NewBaseCache(name, cacheType, 10*time.Minute),
		refreshData: make(map[string]interface{}),
	}
}

// Refresh simulates refreshing cache data
func (m *MockCache) Refresh() error {
	// Simulate loading fresh data
	m.refreshData["last_refresh"] = time.Now()
	m.refreshData["data_version"] = time.Now().Unix()

	// Clear existing data and reload
	err := m.Clear()
	if err != nil {
		return err
	}

	// Add some mock data
	m.Set("test_key_1", "test_value_1", 0)
	m.Set("test_key_2", "test_value_2", 0)
	m.Set("last_refresh", m.refreshData["last_refresh"], 0)

	return nil
}

// ConfigurationCache provides a cache for configuration data
type ConfigurationCache struct {
	*BaseCache
	configData map[string]interface{}
}

// NewConfigurationCache creates a new configuration cache
func NewConfigurationCache() *ConfigurationCache {
	return &ConfigurationCache{
		BaseCache:  NewBaseCache("configuration", CacheTypeConfiguration, 30*time.Minute),
		configData: make(map[string]interface{}),
	}
}

// Refresh reloads configuration data
func (c *ConfigurationCache) Refresh() error {
	// Clear existing cache
	err := c.Clear()
	if err != nil {
		return err
	}

	// Reload configuration data (this would come from actual config sources)
	configs := map[string]interface{}{
		"app_name":    "PMA Backend",
		"version":     "1.0.0",
		"cache_ttl":   "5m",
		"max_size":    1000,
		"last_loaded": time.Now(),
	}

	for key, value := range configs {
		c.Set(key, value, 0)
	}

	return nil
}

// SessionCache provides a cache for session data
type SessionCache struct {
	*BaseCache
}

// NewSessionCache creates a new session cache
func NewSessionCache() *SessionCache {
	return &SessionCache{
		BaseCache: NewBaseCache("sessions", CacheTypeSession, 1*time.Hour),
	}
}

// Refresh clears expired sessions
func (s *SessionCache) Refresh() error {
	// Remove expired sessions
	expired := s.GetExpiredKeys()
	for _, key := range expired {
		s.Delete(key)
	}
	return nil
}

// WebSocketCache provides a cache for WebSocket connection data
type WebSocketCache struct {
	*BaseCache
}

// NewWebSocketCache creates a new WebSocket cache
func NewWebSocketCache() *WebSocketCache {
	return &WebSocketCache{
		BaseCache: NewBaseCache("websocket_connections", CacheTypeWebSocket, 2*time.Hour),
	}
}

// Refresh cleans up disconnected WebSocket data
func (w *WebSocketCache) Refresh() error {
	// Clean up stale connection data
	// This would typically involve checking connection status and removing stale entries
	w.CleanupExpired()
	return nil
}

// AnalyticsCache provides a cache for analytics data
type AnalyticsCache struct {
	*BaseCache
}

// NewAnalyticsCache creates a new analytics cache
func NewAnalyticsCache() *AnalyticsCache {
	return &AnalyticsCache{
		BaseCache: NewBaseCache("analytics", CacheTypeAnalytics, 15*time.Minute),
	}
}

// Refresh recomputes analytics data
func (a *AnalyticsCache) Refresh() error {
	// Clear existing analytics
	err := a.Clear()
	if err != nil {
		return err
	}

	// Recompute analytics (this would involve actual calculations)
	analytics := map[string]interface{}{
		"total_requests":   1000,
		"average_response": "125ms",
		"error_rate":       0.02,
		"cache_hit_rate":   0.85,
		"computed_at":      time.Now(),
	}

	for key, value := range analytics {
		a.Set(key, value, 0)
	}

	return nil
}

// SystemCache provides a cache for system-level data
type SystemCache struct {
	*BaseCache
}

// NewSystemCache creates a new system cache
func NewSystemCache() *SystemCache {
	return &SystemCache{
		BaseCache: NewBaseCache("system", CacheTypeSystem, 1*time.Minute),
	}
}

// Refresh updates system metrics
func (s *SystemCache) Refresh() error {
	// Clear existing system data
	err := s.Clear()
	if err != nil {
		return err
	}

	// Refresh system metrics (this would come from actual system monitoring)
	metrics := map[string]interface{}{
		"cpu_usage":    45.2,
		"memory_usage": 67.8,
		"disk_usage":   34.1,
		"uptime":       "5d 12h 30m",
		"last_updated": time.Now(),
	}

	for key, value := range metrics {
		s.Set(key, value, 0)
	}

	return nil
}

// CreateDefaultCaches creates a set of default caches for the system
func CreateDefaultCaches() []Cache {
	return []Cache{
		NewMockCache("query_results", CacheTypeQuery),
		NewMockCache("api_responses", CacheTypeResponse),
		NewConfigurationCache(),
		NewSessionCache(),
		NewWebSocketCache(),
		NewAnalyticsCache(),
		NewSystemCache(),
	}
}

// CreateTestCaches creates caches for testing purposes
func CreateTestCaches() []Cache {
	caches := []Cache{
		NewMockCache("test_cache_1", CacheTypeEntity),
		NewMockCache("test_cache_2", CacheTypeDisplay),
		NewMockCache("test_cache_3", CacheTypeNetwork),
		NewMockCache("test_cache_4", CacheTypeOther),
	}

	// Add some test data
	for i, cache := range caches {
		mockCache := cache.(*MockCache)
		mockCache.Set(fmt.Sprintf("key_%d_1", i), fmt.Sprintf("value_%d_1", i), 0)
		mockCache.Set(fmt.Sprintf("key_%d_2", i), fmt.Sprintf("value_%d_2", i), 0)
		mockCache.Set(fmt.Sprintf("key_%d_3", i), fmt.Sprintf("value_%d_3", i), 0)
	}

	return caches
}
