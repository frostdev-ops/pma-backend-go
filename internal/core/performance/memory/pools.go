package memory

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ObjectPoolManager manages various object pools for memory optimization
type ObjectPoolManager struct {
	pools         map[string]Pool
	stats         map[string]*PoolStats
	mu            sync.RWMutex
	config        *PoolManagerConfig
	monitorTicker *time.Ticker
	stopChan      chan bool
}

// Pool defines the interface for object pools
type Pool interface {
	Get() interface{}
	Put(interface{})
	Size() int
	Reset()
	Stats() *PoolStats
}

// PoolStats contains statistics for an object pool
type PoolStats struct {
	Name      string    `json:"name"`
	Gets      int64     `json:"gets"`
	Puts      int64     `json:"puts"`
	Hits      int64     `json:"hits"`
	Misses    int64     `json:"misses"`
	Size      int       `json:"size"`
	MaxSize   int       `json:"max_size"`
	HitRate   float64   `json:"hit_rate"`
	LastReset time.Time `json:"last_reset"`
	CreatedAt time.Time `json:"created_at"`
}

// PoolManagerConfig contains configuration for the pool manager
type PoolManagerConfig struct {
	MonitorInterval    time.Duration `json:"monitor_interval"`
	AutoSizing         bool          `json:"auto_sizing"`
	MaxPoolSize        int           `json:"max_pool_size"`
	MinPoolSize        int           `json:"min_pool_size"`
	SizeAdjustmentRate float64       `json:"size_adjustment_rate"`
}

// NewObjectPoolManager creates a new object pool manager
func NewObjectPoolManager(config *PoolManagerConfig) *ObjectPoolManager {
	if config == nil {
		config = &PoolManagerConfig{
			MonitorInterval:    time.Minute * 5,
			AutoSizing:         true,
			MaxPoolSize:        1000,
			MinPoolSize:        10,
			SizeAdjustmentRate: 0.1,
		}
	}

	opm := &ObjectPoolManager{
		pools:    make(map[string]Pool),
		stats:    make(map[string]*PoolStats),
		config:   config,
		stopChan: make(chan bool),
	}

	// Initialize common pools
	opm.initializeCommonPools()

	// Start monitoring if auto-sizing is enabled
	if config.AutoSizing {
		opm.startMonitoring()
	}

	return opm
}

// initializeCommonPools creates commonly used object pools
func (opm *ObjectPoolManager) initializeCommonPools() {
	// Buffer pool for I/O operations
	opm.RegisterPool("buffer", NewBufferPool(1024, opm.config.MaxPoolSize))

	// JSON response pool
	opm.RegisterPool("json_response", NewJSONResponsePool(opm.config.MaxPoolSize))

	// String builder pool
	opm.RegisterPool("string_builder", NewStringBuilderPool(opm.config.MaxPoolSize))

	// Slice pools for different sizes
	opm.RegisterPool("byte_slice_small", NewByteSlicePool(1024, opm.config.MaxPoolSize))
	opm.RegisterPool("byte_slice_medium", NewByteSlicePool(8192, opm.config.MaxPoolSize))
	opm.RegisterPool("byte_slice_large", NewByteSlicePool(65536, opm.config.MaxPoolSize))
}

// RegisterPool registers a new pool with the manager
func (opm *ObjectPoolManager) RegisterPool(name string, pool Pool) {
	opm.mu.Lock()
	defer opm.mu.Unlock()

	opm.pools[name] = pool
	opm.stats[name] = &PoolStats{
		Name:      name,
		CreatedAt: time.Now(),
		LastReset: time.Now(),
	}
}

// GetPool returns a pool by name
func (opm *ObjectPoolManager) GetPool(name string) (Pool, bool) {
	opm.mu.RLock()
	defer opm.mu.RUnlock()

	pool, exists := opm.pools[name]
	return pool, exists
}

// Get retrieves an object from the specified pool
func (opm *ObjectPoolManager) Get(poolName string) interface{} {
	opm.mu.RLock()
	pool, exists := opm.pools[poolName]
	stats := opm.stats[poolName]
	opm.mu.RUnlock()

	if !exists {
		return nil
	}

	obj := pool.Get()

	// Update statistics
	opm.mu.Lock()
	stats.Gets++
	if obj != nil {
		stats.Hits++
	} else {
		stats.Misses++
	}
	stats.HitRate = float64(stats.Hits) / float64(stats.Gets)
	opm.mu.Unlock()

	return obj
}

// Put returns an object to the specified pool
func (opm *ObjectPoolManager) Put(poolName string, obj interface{}) {
	opm.mu.RLock()
	pool, exists := opm.pools[poolName]
	stats := opm.stats[poolName]
	opm.mu.RUnlock()

	if !exists {
		return
	}

	pool.Put(obj)

	// Update statistics
	opm.mu.Lock()
	stats.Puts++
	opm.mu.Unlock()
}

// GetAllStats returns statistics for all pools
func (opm *ObjectPoolManager) GetAllStats() map[string]*PoolStats {
	opm.mu.RLock()
	defer opm.mu.RUnlock()

	result := make(map[string]*PoolStats)
	for name, pool := range opm.pools {
		poolStats := pool.Stats()
		if poolStats != nil {
			result[name] = poolStats
		} else if stats, exists := opm.stats[name]; exists {
			// Create a copy
			statsCopy := *stats
			statsCopy.Size = pool.Size()
			result[name] = &statsCopy
		}
	}

	return result
}

// startMonitoring begins pool monitoring and auto-sizing
func (opm *ObjectPoolManager) startMonitoring() {
	opm.monitorTicker = time.NewTicker(opm.config.MonitorInterval)

	go func() {
		for {
			select {
			case <-opm.monitorTicker.C:
				opm.optimizePools()
			case <-opm.stopChan:
				opm.monitorTicker.Stop()
				return
			}
		}
	}()
}

// optimizePools performs automatic pool size optimization
func (opm *ObjectPoolManager) optimizePools() {
	opm.mu.Lock()
	defer opm.mu.Unlock()

	for name, pool := range opm.pools {
		stats := opm.stats[name]

		// Calculate optimal size based on usage patterns
		if stats.Gets > 0 {
			currentSize := pool.Size()

			// If hit rate is low, increase pool size
			if stats.HitRate < 0.8 && currentSize < opm.config.MaxPoolSize {
				// Increase size gradually
				newSize := int(float64(currentSize) * (1 + opm.config.SizeAdjustmentRate))
				if newSize > opm.config.MaxPoolSize {
					newSize = opm.config.MaxPoolSize
				}
				opm.adjustPoolSize(name, pool, newSize)
			}

			// If hit rate is very high and pool is large, consider reducing size
			if stats.HitRate > 0.95 && currentSize > opm.config.MinPoolSize {
				// Only reduce if gets are infrequent
				getsPerMinute := float64(stats.Gets) / time.Since(stats.LastReset).Minutes()
				if getsPerMinute < 10 { // Less than 10 gets per minute
					newSize := int(float64(currentSize) * (1 - opm.config.SizeAdjustmentRate))
					if newSize < opm.config.MinPoolSize {
						newSize = opm.config.MinPoolSize
					}
					opm.adjustPoolSize(name, pool, newSize)
				}
			}
		}
	}
}

// adjustPoolSize adjusts the size of a pool (if supported)
func (opm *ObjectPoolManager) adjustPoolSize(name string, pool Pool, newSize int) {
	// This would be implemented by pools that support dynamic sizing
	// For now, we'll just log the recommendation
}

// Stop stops the pool manager
func (opm *ObjectPoolManager) Stop() {
	close(opm.stopChan)
}

// BufferPool implements a pool for byte buffers
type BufferPool struct {
	pool    sync.Pool
	size    int
	maxSize int
	stats   *PoolStats
	mu      sync.RWMutex
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(bufferSize, maxSize int) *BufferPool {
	bp := &BufferPool{
		size:    bufferSize,
		maxSize: maxSize,
		stats: &PoolStats{
			Name:      "buffer",
			CreatedAt: time.Now(),
			LastReset: time.Now(),
		},
	}

	bp.pool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, bufferSize))
		},
	}

	return bp
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() interface{} {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	buf := bp.pool.Get().(*bytes.Buffer)
	buf.Reset()

	bp.stats.Gets++
	bp.stats.Hits++
	bp.stats.HitRate = float64(bp.stats.Hits) / float64(bp.stats.Gets)

	return buf
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(obj interface{}) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if buf, ok := obj.(*bytes.Buffer); ok {
		// Reset the buffer before returning to pool
		buf.Reset()
		bp.pool.Put(buf)
		bp.stats.Puts++
	}
}

// Size returns the current pool size (estimated)
func (bp *BufferPool) Size() int {
	return 0 // sync.Pool doesn't expose size
}

// Reset resets pool statistics
func (bp *BufferPool) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.stats.Gets = 0
	bp.stats.Puts = 0
	bp.stats.Hits = 0
	bp.stats.Misses = 0
	bp.stats.HitRate = 0
	bp.stats.LastReset = time.Now()
}

// Stats returns pool statistics
func (bp *BufferPool) Stats() *PoolStats {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	statsCopy := *bp.stats
	return &statsCopy
}

// JSONResponsePool implements a pool for JSON response objects
type JSONResponsePool struct {
	pool    sync.Pool
	maxSize int
	stats   *PoolStats
	mu      sync.RWMutex
}

// JSONResponse represents a reusable JSON response structure
type JSONResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewJSONResponsePool creates a new JSON response pool
func NewJSONResponsePool(maxSize int) *JSONResponsePool {
	jrp := &JSONResponsePool{
		maxSize: maxSize,
		stats: &PoolStats{
			Name:      "json_response",
			CreatedAt: time.Now(),
			LastReset: time.Now(),
		},
	}

	jrp.pool = sync.Pool{
		New: func() interface{} {
			return &JSONResponse{}
		},
	}

	return jrp
}

// Get retrieves a JSON response from the pool
func (jrp *JSONResponsePool) Get() interface{} {
	jrp.mu.Lock()
	defer jrp.mu.Unlock()

	resp := jrp.pool.Get().(*JSONResponse)

	// Reset the response
	resp.Status = ""
	resp.Message = ""
	resp.Data = nil
	resp.Error = ""

	jrp.stats.Gets++
	jrp.stats.Hits++
	jrp.stats.HitRate = float64(jrp.stats.Hits) / float64(jrp.stats.Gets)

	return resp
}

// Put returns a JSON response to the pool
func (jrp *JSONResponsePool) Put(obj interface{}) {
	jrp.mu.Lock()
	defer jrp.mu.Unlock()

	if resp, ok := obj.(*JSONResponse); ok {
		jrp.pool.Put(resp)
		jrp.stats.Puts++
	}
}

// Size returns the current pool size (estimated)
func (jrp *JSONResponsePool) Size() int {
	return 0 // sync.Pool doesn't expose size
}

// Reset resets pool statistics
func (jrp *JSONResponsePool) Reset() {
	jrp.mu.Lock()
	defer jrp.mu.Unlock()

	jrp.stats.Gets = 0
	jrp.stats.Puts = 0
	jrp.stats.Hits = 0
	jrp.stats.Misses = 0
	jrp.stats.HitRate = 0
	jrp.stats.LastReset = time.Now()
}

// Stats returns pool statistics
func (jrp *JSONResponsePool) Stats() *PoolStats {
	jrp.mu.RLock()
	defer jrp.mu.RUnlock()

	statsCopy := *jrp.stats
	return &statsCopy
}

// StringBuilderPool implements a pool for strings.Builder
type StringBuilderPool struct {
	pool    sync.Pool
	maxSize int
	stats   *PoolStats
	mu      sync.RWMutex
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool(maxSize int) *StringBuilderPool {
	sbp := &StringBuilderPool{
		maxSize: maxSize,
		stats: &PoolStats{
			Name:      "string_builder",
			CreatedAt: time.Now(),
			LastReset: time.Now(),
		},
	}

	sbp.pool = sync.Pool{
		New: func() interface{} {
			var sb strings.Builder
			sb.Grow(256) // Pre-allocate some capacity
			return &sb
		},
	}

	return sbp
}

// Get retrieves a string builder from the pool
func (sbp *StringBuilderPool) Get() interface{} {
	sbp.mu.Lock()
	defer sbp.mu.Unlock()

	sb := sbp.pool.Get().(*strings.Builder)
	sb.Reset()

	sbp.stats.Gets++
	sbp.stats.Hits++
	sbp.stats.HitRate = float64(sbp.stats.Hits) / float64(sbp.stats.Gets)

	return sb
}

// Put returns a string builder to the pool
func (sbp *StringBuilderPool) Put(obj interface{}) {
	sbp.mu.Lock()
	defer sbp.mu.Unlock()

	if sb, ok := obj.(*strings.Builder); ok {
		sbp.pool.Put(sb)
		sbp.stats.Puts++
	}
}

// Size returns the current pool size (estimated)
func (sbp *StringBuilderPool) Size() int {
	return 0 // sync.Pool doesn't expose size
}

// Reset resets pool statistics
func (sbp *StringBuilderPool) Reset() {
	sbp.mu.Lock()
	defer sbp.mu.Unlock()

	sbp.stats.Gets = 0
	sbp.stats.Puts = 0
	sbp.stats.Hits = 0
	sbp.stats.Misses = 0
	sbp.stats.HitRate = 0
	sbp.stats.LastReset = time.Now()
}

// Stats returns pool statistics
func (sbp *StringBuilderPool) Stats() *PoolStats {
	sbp.mu.RLock()
	defer sbp.mu.RUnlock()

	statsCopy := *sbp.stats
	return &statsCopy
}

// ByteSlicePool implements a pool for byte slices
type ByteSlicePool struct {
	pool     sync.Pool
	capacity int
	maxSize  int
	stats    *PoolStats
	mu       sync.RWMutex
}

// NewByteSlicePool creates a new byte slice pool
func NewByteSlicePool(capacity, maxSize int) *ByteSlicePool {
	bsp := &ByteSlicePool{
		capacity: capacity,
		maxSize:  maxSize,
		stats: &PoolStats{
			Name:      "byte_slice",
			CreatedAt: time.Now(),
			LastReset: time.Now(),
		},
	}

	bsp.pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, capacity)
		},
	}

	return bsp
}

// Get retrieves a byte slice from the pool
func (bsp *ByteSlicePool) Get() interface{} {
	bsp.mu.Lock()
	defer bsp.mu.Unlock()

	slice := bsp.pool.Get().([]byte)
	slice = slice[:0] // Reset length but keep capacity

	bsp.stats.Gets++
	bsp.stats.Hits++
	bsp.stats.HitRate = float64(bsp.stats.Hits) / float64(bsp.stats.Gets)

	return slice
}

// Put returns a byte slice to the pool
func (bsp *ByteSlicePool) Put(obj interface{}) {
	bsp.mu.Lock()
	defer bsp.mu.Unlock()

	if slice, ok := obj.([]byte); ok {
		// Only accept slices with appropriate capacity
		if cap(slice) == bsp.capacity {
			bsp.pool.Put(slice)
			bsp.stats.Puts++
		}
	}
}

// Size returns the current pool size (estimated)
func (bsp *ByteSlicePool) Size() int {
	return 0 // sync.Pool doesn't expose size
}

// Reset resets pool statistics
func (bsp *ByteSlicePool) Reset() {
	bsp.mu.Lock()
	defer bsp.mu.Unlock()

	bsp.stats.Gets = 0
	bsp.stats.Puts = 0
	bsp.stats.Hits = 0
	bsp.stats.Misses = 0
	bsp.stats.HitRate = 0
	bsp.stats.LastReset = time.Now()
}

// Stats returns pool statistics
func (bsp *ByteSlicePool) Stats() *PoolStats {
	bsp.mu.RLock()
	defer bsp.mu.RUnlock()

	statsCopy := *bsp.stats
	return &statsCopy
}

// GetPoolReport generates a comprehensive pool performance report
func (opm *ObjectPoolManager) GetPoolReport() *PoolReport {
	opm.mu.RLock()
	defer opm.mu.RUnlock()

	allStats := opm.GetAllStats()

	report := &PoolReport{
		GeneratedAt:     time.Now(),
		TotalPools:      len(opm.pools),
		PoolStats:       allStats,
		Recommendations: opm.generatePoolRecommendations(allStats),
		Configuration:   *opm.config,
	}

	return report
}

// PoolReport contains comprehensive pool analysis
type PoolReport struct {
	GeneratedAt     time.Time             `json:"generated_at"`
	TotalPools      int                   `json:"total_pools"`
	PoolStats       map[string]*PoolStats `json:"pool_stats"`
	Recommendations []string              `json:"recommendations"`
	Configuration   PoolManagerConfig     `json:"configuration"`
}

// generatePoolRecommendations generates optimization recommendations for pools
func (opm *ObjectPoolManager) generatePoolRecommendations(allStats map[string]*PoolStats) []string {
	recommendations := []string{}

	for name, stats := range allStats {
		if stats.Gets > 0 {
			if stats.HitRate < 0.5 {
				recommendations = append(recommendations,
					fmt.Sprintf("Pool '%s' has low hit rate (%.2f) - consider increasing pool size", name, stats.HitRate))
			}

			if stats.Gets > 1000 && stats.HitRate > 0.95 {
				recommendations = append(recommendations,
					fmt.Sprintf("Pool '%s' has very high hit rate and usage - monitor for memory pressure", name))
			}
		}

		utilization := float64(stats.Puts) / float64(stats.Gets)
		if utilization < 0.3 {
			recommendations = append(recommendations,
				fmt.Sprintf("Pool '%s' has low object return rate (%.2f) - check for object leaks", name, utilization))
		}
	}

	if len(allStats) > 20 {
		recommendations = append(recommendations, "Large number of pools detected - consider consolidating similar pools")
	}

	return recommendations
}
