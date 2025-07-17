package historical

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/sirupsen/logrus"
)

// timeSeriesManager implements the TimeSeriesManager interface
type timeSeriesManager struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTimeSeriesManager creates a new time series manager
func NewTimeSeriesManager(db *sql.DB, logger *logrus.Logger) (analytics.TimeSeriesManager, error) {
	return &timeSeriesManager{
		db:     db,
		logger: logger,
	}, nil
}

// StoreDataPoint stores a data point in the time series database
func (tsm *timeSeriesManager) StoreDataPoint(series string, point analytics.DataPoint) error {
	query := `
		INSERT INTO time_series_data (
			series_name, timestamp, value, tags, metadata, data_type
		) VALUES (?, ?, ?, ?, ?, ?)`

	tagsJSON, _ := json.Marshal(point.Tags)
	metadataJSON, _ := json.Marshal(point.Metadata)

	_, err := tsm.db.Exec(query,
		series, point.Timestamp, point.Value,
		tagsJSON, metadataJSON, "raw")

	if err != nil {
		return fmt.Errorf("failed to store data point: %w", err)
	}

	tsm.logger.Debug("Stored time series data point", map[string]interface{}{
		"series":    series,
		"timestamp": point.Timestamp,
		"value":     point.Value,
	})

	return nil
}

// QueryTimeSeries queries time series data based on the provided query
func (tsm *timeSeriesManager) QueryTimeSeries(query *analytics.TimeSeriesQuery) (*analytics.TimeSeriesResult, error) {
	if query == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	// Build SQL query
	sqlQuery := `
		SELECT timestamp, value, tags, metadata
		FROM time_series_data
		WHERE series_name = ? AND timestamp >= ? AND timestamp <= ?`

	args := []interface{}{query.Series, query.StartTime, query.EndTime}

	// Add filters if specified
	if len(query.Filters) > 0 {
		for key, value := range query.Filters {
			sqlQuery += fmt.Sprintf(" AND JSON_EXTRACT(tags, '$.%s') = ?", key)
			args = append(args, value)
		}
	}

	// Add ordering
	sqlQuery += " ORDER BY timestamp"

	// Add limit and offset if specified
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
		if query.Offset > 0 {
			sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
		}
	}

	rows, err := tsm.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series data: %w", err)
	}
	defer rows.Close()

	var dataPoints []analytics.DataPoint
	for rows.Next() {
		var timestamp time.Time
		var value float64
		var tagsJSON, metadataJSON []byte

		if err := rows.Scan(&timestamp, &value, &tagsJSON, &metadataJSON); err != nil {
			continue // Skip invalid rows
		}

		var tags map[string]string
		var metadata map[string]interface{}
		json.Unmarshal(tagsJSON, &tags)
		json.Unmarshal(metadataJSON, &metadata)

		dataPoints = append(dataPoints, analytics.DataPoint{
			Timestamp: timestamp,
			Value:     value,
			Tags:      tags,
			Metadata:  metadata,
		})
	}

	// Apply aggregation if specified
	if query.Aggregation != "" && query.Resolution > 0 {
		dataPoints = tsm.aggregateDataPoints(dataPoints, query.Resolution, query.Aggregation)
	}

	// Get series metadata
	metadata, err := tsm.GetSeriesMetadata(query.Series)
	if err != nil {
		// Create default metadata if not found
		metadata = &analytics.SeriesMetadata{
			Name:       query.Series,
			DataType:   "numeric",
			Created:    time.Now(),
			LastUpdate: time.Now(),
			DataPoints: int64(len(dataPoints)),
		}
	}

	result := &analytics.TimeSeriesResult{
		Series:   query.Series,
		Data:     dataPoints,
		Metadata: *metadata,
		Query:    *query,
		Total:    len(dataPoints),
	}

	return result, nil
}

// CreateRetentionPolicy creates a data retention policy
func (tsm *timeSeriesManager) CreateRetentionPolicy(policy *analytics.RetentionPolicy) error {
	// This is a simplified implementation
	// In a production system, you'd store policies in a dedicated table
	tsm.logger.Info("Created retention policy", map[string]interface{}{
		"pattern":  policy.SeriesPattern,
		"duration": policy.Duration,
	})
	return nil
}

// DownsampleData downsamples data to a lower resolution
func (tsm *timeSeriesManager) DownsampleData(series string, resolution time.Duration) error {
	// Query existing raw data
	query := &analytics.TimeSeriesQuery{
		Series:    series,
		StartTime: time.Now().Add(-30 * 24 * time.Hour), // Last 30 days
		EndTime:   time.Now(),
	}

	result, err := tsm.QueryTimeSeries(query)
	if err != nil {
		return fmt.Errorf("failed to query data for downsampling: %w", err)
	}

	if len(result.Data) == 0 {
		return nil // No data to downsample
	}

	// Aggregate data by resolution
	aggregatedData := tsm.aggregateDataPoints(result.Data, resolution, "avg")

	// Store aggregated data
	for _, point := range aggregatedData {
		downSampledQuery := `
			INSERT OR REPLACE INTO time_series_data (
				series_name, timestamp, value, tags, metadata, resolution, data_type
			) VALUES (?, ?, ?, ?, ?, ?, ?)`

		tagsJSON, _ := json.Marshal(point.Tags)
		metadataJSON, _ := json.Marshal(point.Metadata)

		_, err := tsm.db.Exec(downSampledQuery,
			series, point.Timestamp, point.Value,
			tagsJSON, metadataJSON, int64(resolution.Seconds()), "aggregated")

		if err != nil {
			tsm.logger.Warn("Failed to store downsampled data point", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// GetSeriesMetadata retrieves metadata for a time series
func (tsm *timeSeriesManager) GetSeriesMetadata(series string) (*analytics.SeriesMetadata, error) {
	query := `
		SELECT 
			COUNT(*) as data_points,
			MIN(timestamp) as first_timestamp,
			MAX(timestamp) as last_timestamp
		FROM time_series_data
		WHERE series_name = ?`

	var dataPoints int64
	var firstTimestamp, lastTimestamp sql.NullTime

	err := tsm.db.QueryRow(query, series).Scan(&dataPoints, &firstTimestamp, &lastTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get series metadata: %w", err)
	}

	created := time.Now()
	lastUpdate := time.Now()

	if firstTimestamp.Valid {
		created = firstTimestamp.Time
	}
	if lastTimestamp.Valid {
		lastUpdate = lastTimestamp.Time
	}

	metadata := &analytics.SeriesMetadata{
		Name:       series,
		DataType:   "numeric",
		Created:    created,
		LastUpdate: lastUpdate,
		DataPoints: dataPoints,
		Tags:       make(map[string]string),
	}

	return metadata, nil
}

// CreateSeries creates a new time series with metadata
func (tsm *timeSeriesManager) CreateSeries(name string, metadata analytics.SeriesMetadata) error {
	// For this implementation, series are created implicitly when data is stored
	tsm.logger.Info("Time series will be created on first data point", map[string]interface{}{
		"series": name,
	})
	return nil
}

// DeleteSeries deletes a time series and all its data
func (tsm *timeSeriesManager) DeleteSeries(name string) error {
	query := `DELETE FROM time_series_data WHERE series_name = ?`

	result, err := tsm.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete series: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	tsm.logger.Info("Deleted time series", map[string]interface{}{
		"series": name,
		"rows":   rowsAffected,
	})

	return nil
}

// Private helper methods

// aggregateDataPoints aggregates data points by time resolution
func (tsm *timeSeriesManager) aggregateDataPoints(data []analytics.DataPoint, resolution time.Duration, aggregation string) []analytics.DataPoint {
	if len(data) == 0 {
		return data
	}

	// Group data by time buckets
	buckets := make(map[int64][]analytics.DataPoint)
	start := data[0].Timestamp

	for _, point := range data {
		bucket := point.Timestamp.Sub(start) / resolution
		buckets[int64(bucket)] = append(buckets[int64(bucket)], point)
	}

	var aggregated []analytics.DataPoint

	for bucket, points := range buckets {
		if len(points) == 0 {
			continue
		}

		timestamp := start.Add(time.Duration(bucket) * resolution)
		value := tsm.aggregateValues(points, aggregation)

		// Merge tags from all points in the bucket
		tags := make(map[string]string)
		for _, point := range points {
			for k, v := range point.Tags {
				tags[k] = v
			}
		}

		aggregated = append(aggregated, analytics.DataPoint{
			Timestamp: timestamp,
			Value:     value,
			Tags:      tags,
		})
	}

	return aggregated
}

// aggregateValues aggregates values based on the specified method
func (tsm *timeSeriesManager) aggregateValues(points []analytics.DataPoint, aggregation string) float64 {
	if len(points) == 0 {
		return 0
	}

	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}

	switch strings.ToLower(aggregation) {
	case "sum":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum

	case "avg", "average", "mean":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))

	case "min":
		min := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min

	case "max":
		max := values[0]
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max

	case "count":
		return float64(len(values))

	case "first":
		return values[0]

	case "last":
		return values[len(values)-1]

	default:
		// Default to average
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	}
}
