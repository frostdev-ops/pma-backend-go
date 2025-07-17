package analytics

import (
	"math"
	"sort"
	"sync"
	"time"
)

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	RequestCount    int64                       `json:"request_count"`
	AverageLatency  time.Duration               `json:"average_latency"`
	P50Latency      time.Duration               `json:"p50_latency"`
	P95Latency      time.Duration               `json:"p95_latency"`
	P99Latency      time.Duration               `json:"p99_latency"`
	MinLatency      time.Duration               `json:"min_latency"`
	MaxLatency      time.Duration               `json:"max_latency"`
	Throughput      float64                     `json:"throughput"` // requests per second
	ErrorRate       float64                     `json:"error_rate"`
	ErrorCount      int64                       `json:"error_count"`
	TotalRequests   int64                       `json:"total_requests"`
	EndpointMetrics map[string]*EndpointMetrics `json:"endpoint_metrics"`
	TimeSeriesData  []TimeSeriesDataPoint       `json:"time_series_data"`
	ResourceMetrics *ResourcePerformanceMetrics `json:"resource_metrics"`
	DatabaseMetrics *DatabasePerformanceMetrics `json:"database_metrics"`
	CacheMetrics    *CachePerformanceMetrics    `json:"cache_metrics"`
	Period          time.Duration               `json:"period"`
	Timestamp       time.Time                   `json:"timestamp"`
}

// EndpointMetrics represents metrics for a specific endpoint
type EndpointMetrics struct {
	Path           string        `json:"path"`
	Method         string        `json:"method"`
	RequestCount   int64         `json:"request_count"`
	ErrorCount     int64         `json:"error_count"`
	AverageLatency time.Duration `json:"average_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	StatusCodes    map[int]int64 `json:"status_codes"`
	LastAccessed   time.Time     `json:"last_accessed"`
}

// TimeSeriesDataPoint represents a data point in time series
type TimeSeriesDataPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Metric    string            `json:"metric"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ResourcePerformanceMetrics represents resource-related performance metrics
type ResourcePerformanceMetrics struct {
	CPUUsage    float64         `json:"cpu_usage"`
	MemoryUsage float64         `json:"memory_usage"`
	DiskUsage   float64         `json:"disk_usage"`
	Goroutines  int             `json:"goroutines"`
	HeapSize    uint64          `json:"heap_size"`
	GCPauses    []time.Duration `json:"gc_pauses"`
	GCCount     uint32          `json:"gc_count"`
}

// DatabasePerformanceMetrics represents database performance metrics
type DatabasePerformanceMetrics struct {
	QueryCount        int64         `json:"query_count"`
	AverageQueryTime  time.Duration `json:"average_query_time"`
	SlowQueries       int64         `json:"slow_queries"`
	ConnectionsActive int           `json:"connections_active"`
	ConnectionsIdle   int           `json:"connections_idle"`
	ErrorCount        int64         `json:"error_count"`
}

// CachePerformanceMetrics represents cache performance metrics
type CachePerformanceMetrics struct {
	HitCount  int64   `json:"hit_count"`
	MissCount int64   `json:"miss_count"`
	HitRate   float64 `json:"hit_rate"`
	Size      int64   `json:"size"`
	Evictions int64   `json:"evictions"`
}

// RequestData represents individual request data for analysis
type RequestData struct {
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	StatusCode int               `json:"status_code"`
	Duration   time.Duration     `json:"duration"`
	Timestamp  time.Time         `json:"timestamp"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Size       int64             `json:"size,omitempty"`
}

// PerformanceTracker tracks performance metrics
type PerformanceTracker struct {
	mu              sync.RWMutex
	requests        []RequestData
	endpointMetrics map[string]*EndpointMetrics
	maxDataPoints   int
	timeWindow      time.Duration

	// Configuration
	trackUserAgents bool
	calculateP99    bool
	calculateP95    bool
	calculateP50    bool
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(maxDataPoints int, timeWindow time.Duration) *PerformanceTracker {
	return &PerformanceTracker{
		requests:        make([]RequestData, 0, maxDataPoints),
		endpointMetrics: make(map[string]*EndpointMetrics),
		maxDataPoints:   maxDataPoints,
		timeWindow:      timeWindow,
		trackUserAgents: false,
		calculateP99:    true,
		calculateP95:    true,
		calculateP50:    true,
	}
}

// SetOptions sets tracking options
func (pt *PerformanceTracker) SetOptions(trackUserAgents, p99, p95, p50 bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.trackUserAgents = trackUserAgents
	pt.calculateP99 = p99
	pt.calculateP95 = p95
	pt.calculateP50 = p50
}

// RecordRequest records a request for performance analysis
func (pt *PerformanceTracker) RecordRequest(method, path string, statusCode int, duration time.Duration, userAgent string, size int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	request := RequestData{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   duration,
		Timestamp:  time.Now(),
		Size:       size,
	}

	if pt.trackUserAgents {
		request.UserAgent = userAgent
	}

	// Add request to slice
	pt.requests = append(pt.requests, request)

	// Remove old requests if we exceed max data points
	if len(pt.requests) > pt.maxDataPoints {
		pt.requests = pt.requests[1:]
	}

	// Update endpoint metrics
	endpointKey := method + ":" + path
	if pt.endpointMetrics[endpointKey] == nil {
		pt.endpointMetrics[endpointKey] = &EndpointMetrics{
			Path:        path,
			Method:      method,
			StatusCodes: make(map[int]int64),
			MinLatency:  duration,
			MaxLatency:  duration,
		}
	}

	endpoint := pt.endpointMetrics[endpointKey]
	endpoint.RequestCount++
	endpoint.LastAccessed = request.Timestamp
	endpoint.StatusCodes[statusCode]++

	if statusCode >= 400 {
		endpoint.ErrorCount++
	}

	if duration < endpoint.MinLatency {
		endpoint.MinLatency = duration
	}

	if duration > endpoint.MaxLatency {
		endpoint.MaxLatency = duration
	}

	// Calculate rolling average
	if endpoint.AverageLatency == 0 {
		endpoint.AverageLatency = duration
	} else {
		// Simple exponential moving average
		alpha := 0.1
		endpoint.AverageLatency = time.Duration(float64(endpoint.AverageLatency)*(1-alpha) + float64(duration)*alpha)
	}
}

// GetPerformanceMetrics calculates and returns performance metrics for the specified period
func (pt *PerformanceTracker) GetPerformanceMetrics(period time.Duration) *PerformanceMetrics {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	var relevantRequests []RequestData

	// Filter requests within the time period
	for _, req := range pt.requests {
		if req.Timestamp.After(cutoff) {
			relevantRequests = append(relevantRequests, req)
		}
	}

	if len(relevantRequests) == 0 {
		return &PerformanceMetrics{
			Period:          period,
			Timestamp:       time.Now(),
			EndpointMetrics: pt.copyEndpointMetrics(),
		}
	}

	// Calculate metrics
	metrics := &PerformanceMetrics{
		Period:          period,
		Timestamp:       time.Now(),
		EndpointMetrics: pt.copyEndpointMetrics(),
		TimeSeriesData:  pt.generateTimeSeriesData(relevantRequests, period),
	}

	// Calculate basic metrics
	var totalDuration time.Duration
	var errorCount int64
	durations := make([]time.Duration, len(relevantRequests))

	for i, req := range relevantRequests {
		totalDuration += req.Duration
		durations[i] = req.Duration

		if req.StatusCode >= 400 {
			errorCount++
		}
	}

	metrics.RequestCount = int64(len(relevantRequests))
	metrics.ErrorCount = errorCount
	metrics.TotalRequests = int64(len(pt.requests))
	metrics.AverageLatency = totalDuration / time.Duration(len(relevantRequests))
	metrics.Throughput = float64(len(relevantRequests)) / period.Seconds()

	if len(relevantRequests) > 0 {
		metrics.ErrorRate = float64(errorCount) / float64(len(relevantRequests))
	}

	// Calculate percentiles
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	metrics.MinLatency = durations[0]
	metrics.MaxLatency = durations[len(durations)-1]

	if pt.calculateP50 {
		metrics.P50Latency = pt.calculatePercentile(durations, 0.5)
	}

	if pt.calculateP95 {
		metrics.P95Latency = pt.calculatePercentile(durations, 0.95)
	}

	if pt.calculateP99 {
		metrics.P99Latency = pt.calculatePercentile(durations, 0.99)
	}

	return metrics
}

// GetLatencyPercentiles calculates specific percentiles for latency
func (pt *PerformanceTracker) GetLatencyPercentiles(period time.Duration, percentiles []float64) map[float64]time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	var durations []time.Duration

	for _, req := range pt.requests {
		if req.Timestamp.After(cutoff) {
			durations = append(durations, req.Duration)
		}
	}

	if len(durations) == 0 {
		return make(map[float64]time.Duration)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	result := make(map[float64]time.Duration)
	for _, p := range percentiles {
		result[p] = pt.calculatePercentile(durations, p)
	}

	return result
}

// GetErrorRateByEndpoint returns error rates for each endpoint
func (pt *PerformanceTracker) GetErrorRateByEndpoint(period time.Duration) map[string]float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	endpointStats := make(map[string]struct {
		total  int64
		errors int64
	})

	for _, req := range pt.requests {
		if req.Timestamp.After(cutoff) {
			key := req.Method + ":" + req.Path
			stats := endpointStats[key]
			stats.total++

			if req.StatusCode >= 400 {
				stats.errors++
			}

			endpointStats[key] = stats
		}
	}

	errorRates := make(map[string]float64)
	for endpoint, stats := range endpointStats {
		if stats.total > 0 {
			errorRates[endpoint] = float64(stats.errors) / float64(stats.total)
		}
	}

	return errorRates
}

// GetThroughputTimeSeries returns throughput data over time
func (pt *PerformanceTracker) GetThroughputTimeSeries(period time.Duration, bucketSize time.Duration) []TimeSeriesDataPoint {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	buckets := make(map[int64]int64)

	for _, req := range pt.requests {
		if req.Timestamp.After(cutoff) {
			bucket := req.Timestamp.Unix() / int64(bucketSize.Seconds())
			buckets[bucket]++
		}
	}

	var dataPoints []TimeSeriesDataPoint
	for bucket, count := range buckets {
		timestamp := time.Unix(bucket*int64(bucketSize.Seconds()), 0)
		throughput := float64(count) / bucketSize.Seconds()

		dataPoints = append(dataPoints, TimeSeriesDataPoint{
			Timestamp: timestamp,
			Value:     throughput,
			Metric:    "throughput",
		})
	}

	// Sort by timestamp
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	return dataPoints
}

// GetSlowRequests returns requests that exceed the specified threshold
func (pt *PerformanceTracker) GetSlowRequests(threshold time.Duration, limit int) []RequestData {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var slowRequests []RequestData

	for _, req := range pt.requests {
		if req.Duration > threshold {
			slowRequests = append(slowRequests, req)
		}
	}

	// Sort by duration (slowest first)
	sort.Slice(slowRequests, func(i, j int) bool {
		return slowRequests[i].Duration > slowRequests[j].Duration
	})

	if limit > 0 && len(slowRequests) > limit {
		slowRequests = slowRequests[:limit]
	}

	return slowRequests
}

// ClearOldData removes data older than the retention period
func (pt *PerformanceTracker) ClearOldData(retentionPeriod time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	var newRequests []RequestData

	for _, req := range pt.requests {
		if req.Timestamp.After(cutoff) {
			newRequests = append(newRequests, req)
		}
	}

	pt.requests = newRequests
}

// calculatePercentile calculates the specified percentile from sorted durations
func (pt *PerformanceTracker) calculatePercentile(sortedDurations []time.Duration, percentile float64) time.Duration {
	if len(sortedDurations) == 0 {
		return 0
	}

	if percentile <= 0 {
		return sortedDurations[0]
	}

	if percentile >= 1 {
		return sortedDurations[len(sortedDurations)-1]
	}

	index := percentile * float64(len(sortedDurations)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedDurations[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	lowerValue := float64(sortedDurations[lower])
	upperValue := float64(sortedDurations[upper])

	result := lowerValue + weight*(upperValue-lowerValue)
	return time.Duration(result)
}

// copyEndpointMetrics creates a copy of endpoint metrics
func (pt *PerformanceTracker) copyEndpointMetrics() map[string]*EndpointMetrics {
	copy := make(map[string]*EndpointMetrics)

	for key, metric := range pt.endpointMetrics {
		statusCodes := make(map[int]int64)
		for code, count := range metric.StatusCodes {
			statusCodes[code] = count
		}

		copy[key] = &EndpointMetrics{
			Path:           metric.Path,
			Method:         metric.Method,
			RequestCount:   metric.RequestCount,
			ErrorCount:     metric.ErrorCount,
			AverageLatency: metric.AverageLatency,
			P95Latency:     metric.P95Latency,
			MinLatency:     metric.MinLatency,
			MaxLatency:     metric.MaxLatency,
			StatusCodes:    statusCodes,
			LastAccessed:   metric.LastAccessed,
		}
	}

	return copy
}

// generateTimeSeriesData generates time series data from requests
func (pt *PerformanceTracker) generateTimeSeriesData(requests []RequestData, period time.Duration) []TimeSeriesDataPoint {
	if len(requests) == 0 {
		return nil
	}

	bucketSize := period / 20 // 20 data points
	if bucketSize < time.Minute {
		bucketSize = time.Minute
	}

	buckets := make(map[int64][]time.Duration)

	for _, req := range requests {
		bucket := req.Timestamp.Unix() / int64(bucketSize.Seconds())
		buckets[bucket] = append(buckets[bucket], req.Duration)
	}

	var dataPoints []TimeSeriesDataPoint

	for bucket, durations := range buckets {
		timestamp := time.Unix(bucket*int64(bucketSize.Seconds()), 0)

		// Calculate average latency for this bucket
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avgLatency := total / time.Duration(len(durations))

		dataPoints = append(dataPoints, TimeSeriesDataPoint{
			Timestamp: timestamp,
			Value:     float64(avgLatency.Milliseconds()),
			Metric:    "average_latency_ms",
		})

		// Add throughput data point
		throughput := float64(len(durations)) / bucketSize.Seconds()
		dataPoints = append(dataPoints, TimeSeriesDataPoint{
			Timestamp: timestamp,
			Value:     throughput,
			Metric:    "throughput_rps",
		})
	}

	// Sort by timestamp
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	return dataPoints
}
