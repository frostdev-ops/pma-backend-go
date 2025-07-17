package analytics

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

// dataAggregator implements the DataAggregator interface
type dataAggregator struct {
	logger *logrus.Logger
}

// NewDataAggregator creates a new data aggregator
func NewDataAggregator(logger *logrus.Logger) DataAggregator {
	return &dataAggregator{
		logger: logger,
	}
}

// AggregateByTime aggregates data points by time intervals
func (da *dataAggregator) AggregateByTime(data []DataPoint, interval time.Duration) []AggregatedPoint {
	if len(data) == 0 {
		return []AggregatedPoint{}
	}

	// Sort data by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	var aggregated []AggregatedPoint
	if len(data) == 0 {
		return aggregated
	}

	// Group data by intervals
	start := data[0].Timestamp
	buckets := make(map[int64][]DataPoint)

	for _, point := range data {
		bucket := point.Timestamp.Sub(start) / interval
		buckets[int64(bucket)] = append(buckets[int64(bucket)], point)
	}

	// Create aggregated points
	for bucket, points := range buckets {
		if len(points) == 0 {
			continue
		}

		timestamp := start.Add(time.Duration(bucket) * interval)

		var sum, min, max float64
		min = points[0].Value
		max = points[0].Value

		for _, point := range points {
			sum += point.Value
			if point.Value < min {
				min = point.Value
			}
			if point.Value > max {
				max = point.Value
			}
		}

		avg := sum / float64(len(points))

		// Calculate percentiles
		values := make([]float64, len(points))
		for i, point := range points {
			values[i] = point.Value
		}
		sort.Float64s(values)

		percentiles := make(map[int]float64)
		percentiles[50] = da.calculatePercentile(values, 0.5)
		percentiles[95] = da.calculatePercentile(values, 0.95)
		percentiles[99] = da.calculatePercentile(values, 0.99)

		aggregated = append(aggregated, AggregatedPoint{
			Timestamp:   timestamp,
			Count:       int64(len(points)),
			Sum:         sum,
			Average:     avg,
			Min:         min,
			Max:         max,
			Percentiles: percentiles,
		})
	}

	// Sort by timestamp
	sort.Slice(aggregated, func(i, j int) bool {
		return aggregated[i].Timestamp.Before(aggregated[j].Timestamp)
	})

	return aggregated
}

// AggregateByEntity aggregates data points by entity
func (da *dataAggregator) AggregateByEntity(data []DataPoint, groupBy string) map[string][]DataPoint {
	grouped := make(map[string][]DataPoint)

	for _, point := range data {
		key := "unknown"
		if point.Tags != nil {
			if value, exists := point.Tags[groupBy]; exists {
				key = value
			}
		}
		grouped[key] = append(grouped[key], point)
	}

	return grouped
}

// CalculateStatistics calculates comprehensive statistics for data points
func (da *dataAggregator) CalculateStatistics(data []DataPoint) *Statistics {
	if len(data) == 0 {
		return &Statistics{}
	}

	values := make([]float64, len(data))
	sum := 0.0

	for i, point := range data {
		values[i] = point.Value
		sum += point.Value
	}

	sort.Float64s(values)

	count := int64(len(values))
	mean := sum / float64(count)
	median := da.calculateMedian(values)
	mode := da.calculateMode(values)
	min := values[0]
	max := values[len(values)-1]
	variance := da.calculateVariance(values, mean)
	stdDev := math.Sqrt(variance)

	// Calculate percentiles
	percentiles := make(map[int]float64)
	percentiles[25] = da.calculatePercentile(values, 0.25)
	percentiles[50] = da.calculatePercentile(values, 0.5)
	percentiles[75] = da.calculatePercentile(values, 0.75)
	percentiles[90] = da.calculatePercentile(values, 0.9)
	percentiles[95] = da.calculatePercentile(values, 0.95)
	percentiles[99] = da.calculatePercentile(values, 0.99)

	// Calculate quartiles
	quartiles := make(map[string]float64)
	quartiles["Q1"] = percentiles[25]
	quartiles["Q2"] = percentiles[50] // Median
	quartiles["Q3"] = percentiles[75]

	return &Statistics{
		Count:       count,
		Sum:         sum,
		Mean:        mean,
		Median:      median,
		Mode:        mode,
		StdDev:      stdDev,
		Variance:    variance,
		Min:         min,
		Max:         max,
		Range:       max - min,
		Percentiles: percentiles,
		Quartiles:   quartiles,
	}
}

// DetectTrends identifies trends in the data
func (da *dataAggregator) DetectTrends(data []DataPoint) ([]*Trend, error) {
	if len(data) < 3 {
		return []*Trend{}, nil
	}

	// Sort data by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	var trends []*Trend

	// Simple linear regression to detect overall trend
	trend := da.detectLinearTrend(data)
	if trend != nil {
		trends = append(trends, trend)
	}

	// Detect change points
	changePointTrends := da.detectChangePoints(data)
	trends = append(trends, changePointTrends...)

	return trends, nil
}

// IdentifyAnomalies detects anomalies in the data
func (da *dataAggregator) IdentifyAnomalies(data []DataPoint) ([]*Anomaly, error) {
	if len(data) < 5 {
		return []*Anomaly{}, nil
	}

	var anomalies []*Anomaly

	// Statistical outlier detection using IQR method
	stats := da.CalculateStatistics(data)
	iqr := stats.Quartiles["Q3"] - stats.Quartiles["Q1"]
	lowerBound := stats.Quartiles["Q1"] - 1.5*iqr
	upperBound := stats.Quartiles["Q3"] + 1.5*iqr

	for _, point := range data {
		if point.Value < lowerBound || point.Value > upperBound {
			severity := da.calculateAnomalySeverity(point.Value, stats.Mean, stats.StdDev)

			anomaly := &Anomaly{
				SeriesName:    "data_series",
				Type:          da.determineAnomalyType(point.Value, stats.Mean),
				Severity:      severity,
				DetectedAt:    point.Timestamp,
				Value:         point.Value,
				ExpectedValue: stats.Mean,
				Deviation:     math.Abs(point.Value - stats.Mean),
				Confidence:    da.calculateAnomalyConfidence(point.Value, lowerBound, upperBound),
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	// Z-score based anomaly detection
	zScoreAnomalies := da.detectZScoreAnomalies(data, stats)
	anomalies = append(anomalies, zScoreAnomalies...)

	return anomalies, nil
}

// ComputeCorrelations computes correlations between different datasets
func (da *dataAggregator) ComputeCorrelations(datasets map[string][]DataPoint) map[string]float64 {
	correlations := make(map[string]float64)

	// Convert to slice of dataset names for easier iteration
	names := make([]string, 0, len(datasets))
	for name := range datasets {
		names = append(names, name)
	}

	// Compute pairwise correlations
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			name1, name2 := names[i], names[j]
			correlation := da.computePearsonCorrelation(datasets[name1], datasets[name2])
			correlations[fmt.Sprintf("%s_%s", name1, name2)] = correlation
		}
	}

	return correlations
}

// Private helper methods

func (da *dataAggregator) calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := percentile * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedValues[lower]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

func (da *dataAggregator) calculateMedian(sortedValues []float64) float64 {
	n := len(sortedValues)
	if n%2 == 0 {
		return (sortedValues[n/2-1] + sortedValues[n/2]) / 2
	}
	return sortedValues[n/2]
}

func (da *dataAggregator) calculateMode(values []float64) float64 {
	frequency := make(map[float64]int)
	for _, value := range values {
		frequency[value]++
	}

	var mode float64
	maxFreq := 0
	for value, freq := range frequency {
		if freq > maxFreq {
			maxFreq = freq
			mode = value
		}
	}

	return mode
}

func (da *dataAggregator) calculateVariance(values []float64, mean float64) float64 {
	sum := 0.0
	for _, value := range values {
		diff := value - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (da *dataAggregator) detectLinearTrend(data []DataPoint) *Trend {
	n := float64(len(data))
	if n < 2 {
		return nil
	}

	// Convert timestamps to numeric values (seconds since first timestamp)
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range data {
		x := float64(i) // Use index as x for simplicity
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (linear regression)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend direction
	direction := "stable"
	if slope > 0.01 {
		direction = "up"
	} else if slope < -0.01 {
		direction = "down"
	}

	// Calculate confidence (R-squared)
	meanY := sumY / n
	var ssRes, ssTot float64
	for i, point := range data {
		x := float64(i)
		predicted := (sumY-slope*sumX)/n + slope*x
		ssRes += math.Pow(point.Value-predicted, 2)
		ssTot += math.Pow(point.Value-meanY, 2)
	}

	confidence := 1 - (ssRes / ssTot)
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
		confidence = 0
	}

	return &Trend{
		Direction:    direction,
		Slope:        slope,
		Confidence:   math.Max(0, math.Min(1, confidence)),
		StartTime:    data[0].Timestamp,
		EndTime:      data[len(data)-1].Timestamp,
		Magnitude:    math.Abs(slope),
		Significance: da.determineTrendSignificance(slope, confidence),
	}
}

func (da *dataAggregator) detectChangePoints(data []DataPoint) []*Trend {
	// Simple change point detection using sliding window
	var trends []*Trend
	windowSize := len(data) / 4
	if windowSize < 3 {
		return trends
	}

	for i := windowSize; i < len(data)-windowSize; i++ {
		before := data[i-windowSize : i]
		after := data[i : i+windowSize]

		beforeStats := da.CalculateStatistics(before)
		afterStats := da.CalculateStatistics(after)

		// Detect significant change in mean
		changeMagnitude := math.Abs(afterStats.Mean - beforeStats.Mean)
		if changeMagnitude > beforeStats.StdDev {
			direction := "up"
			if afterStats.Mean < beforeStats.Mean {
				direction = "down"
			}

			trend := &Trend{
				Direction:    direction,
				Slope:        (afterStats.Mean - beforeStats.Mean) / float64(windowSize),
				Confidence:   0.7, // Fixed confidence for change points
				StartTime:    data[i-windowSize].Timestamp,
				EndTime:      data[i+windowSize-1].Timestamp,
				Magnitude:    changeMagnitude,
				Significance: da.determineTrendSignificance(changeMagnitude, 0.7),
			}
			trends = append(trends, trend)
		}
	}

	return trends
}

func (da *dataAggregator) determineTrendSignificance(magnitude, confidence float64) string {
	if confidence > 0.8 && magnitude > 0.5 {
		return "high"
	} else if confidence > 0.6 && magnitude > 0.3 {
		return "medium"
	}
	return "low"
}

func (da *dataAggregator) calculateAnomalySeverity(value, mean, stdDev float64) string {
	zScore := math.Abs(value-mean) / stdDev

	if zScore > 3 {
		return SeverityCritical
	} else if zScore > 2.5 {
		return SeverityHigh
	} else if zScore > 2 {
		return SeverityMedium
	}
	return SeverityLow
}

func (da *dataAggregator) determineAnomalyType(value, mean float64) string {
	if value > mean*1.5 {
		return AnomalyTypeSpike
	} else if value < mean*0.5 {
		return AnomalyTypeDrop
	}
	return AnomalyTypeOutlier
}

func (da *dataAggregator) calculateAnomalyConfidence(value, lowerBound, upperBound float64) float64 {
	if value < lowerBound {
		distance := lowerBound - value
		return math.Min(1.0, distance/(lowerBound*0.1))
	} else if value > upperBound {
		distance := value - upperBound
		return math.Min(1.0, distance/(upperBound*0.1))
	}
	return 0.5
}

func (da *dataAggregator) detectZScoreAnomalies(data []DataPoint, stats *Statistics) []*Anomaly {
	var anomalies []*Anomaly
	threshold := 2.5 // Z-score threshold

	for _, point := range data {
		zScore := math.Abs(point.Value-stats.Mean) / stats.StdDev
		if zScore > threshold {
			severity := da.calculateAnomalySeverity(point.Value, stats.Mean, stats.StdDev)

			anomaly := &Anomaly{
				SeriesName:    "data_series",
				Type:          da.determineAnomalyType(point.Value, stats.Mean),
				Severity:      severity,
				DetectedAt:    point.Timestamp,
				Value:         point.Value,
				ExpectedValue: stats.Mean,
				Deviation:     math.Abs(point.Value - stats.Mean),
				Confidence:    math.Min(1.0, zScore/5.0), // Normalize to 0-1
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (da *dataAggregator) computePearsonCorrelation(data1, data2 []DataPoint) float64 {
	// Align data points by timestamp (simplified - assumes sorted data)
	minLen := len(data1)
	if len(data2) < minLen {
		minLen = len(data2)
	}

	if minLen < 2 {
		return 0.0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	n := float64(minLen)

	for i := 0; i < minLen; i++ {
		x := data1[i].Value
		y := data2[i].Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0.0
	}

	correlation := numerator / denominator
	if math.IsNaN(correlation) || math.IsInf(correlation, 0) {
		return 0.0
	}

	return correlation
}
