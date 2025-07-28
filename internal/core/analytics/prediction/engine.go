package prediction

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/sirupsen/logrus"
)

// predictionEngine implements the PredictionEngine interface
type predictionEngine struct {
	db     *sql.DB
	config *analytics.AnalyticsConfig
	logger *logrus.Logger
	models map[string]*StoredModel
}

// StoredModel represents a trained prediction model
type StoredModel struct {
	ID        string
	Type      string
	Accuracy  float64
	CreatedAt time.Time
	UpdatedAt time.Time

	// Model parameters for linear regression
	Slope     float64
	Intercept float64

	// Statistical parameters
	Mean     float64
	StdDev   float64
	Variance float64

	// Data info
	DataPoints int
	LastValue  float64
	Trend      float64
}

// NewPredictionEngine creates a new prediction engine
func NewPredictionEngine(db *sql.DB, config *analytics.AnalyticsConfig, logger *logrus.Logger) (analytics.PredictionEngine, error) {
	return &predictionEngine{
		db:     db,
		config: config,
		logger: logger,
		models: make(map[string]*StoredModel),
	}, nil
}

// TrainModel trains a prediction model using statistical methods
func (pe *predictionEngine) TrainModel(modelType string, data []analytics.DataPoint) (*analytics.Model, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient data points for training (need at least 2, got %d)", len(data))
	}

	// Sort data by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	model := &StoredModel{
		ID:         fmt.Sprintf("%s_%d", modelType, time.Now().Unix()),
		Type:       modelType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		DataPoints: len(data),
		LastValue:  data[len(data)-1].Value,
	}

	switch modelType {
	case "linear_regression":
		pe.trainLinearRegression(model, data)
	case "moving_average":
		pe.trainMovingAverage(model, data)
	case "trend_analysis":
		pe.trainTrendAnalysis(model, data)
	default:
		// Default to linear regression
		pe.trainLinearRegression(model, data)
	}

	// Calculate model accuracy using coefficient of determination (R²)
	model.Accuracy = pe.calculateAccuracy(model, data)

	// Store model
	pe.models[model.ID] = model

	pe.logger.WithFields(logrus.Fields{
		"model_id":    model.ID,
		"model_type":  modelType,
		"data_points": len(data),
		"accuracy":    model.Accuracy,
	}).Info("Trained prediction model")

	return &analytics.Model{
		ID:       model.ID,
		Type:     modelType,
		Accuracy: model.Accuracy,
	}, nil
}

// PredictValue makes a prediction using a trained model
func (pe *predictionEngine) PredictValue(modelID string, input map[string]interface{}) (*analytics.Prediction, error) {
	model, exists := pe.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	// Extract prediction parameters
	timeHorizon, ok := input["time_horizon"].(float64)
	if !ok {
		timeHorizon = 1.0 // Default to 1 hour ahead
	}

	var predictedValue float64
	var confidence float64

	switch model.Type {
	case "linear_regression":
		predictedValue = model.Slope*timeHorizon + model.Intercept
		confidence = pe.calculateConfidence(model)
	case "moving_average":
		predictedValue = model.Mean
		confidence = 1.0 - (model.StdDev / math.Abs(model.Mean))
	case "trend_analysis":
		predictedValue = model.LastValue + (model.Trend * timeHorizon)
		confidence = pe.calculateConfidence(model)
	default:
		predictedValue = model.LastValue
		confidence = 0.5
	}

	// Ensure confidence is within valid range
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	return &analytics.Prediction{
		ModelID:        modelID,
		InputData:      input,
		PredictedValue: predictedValue,
		Confidence:     confidence,
		CreatedAt:      time.Now(),
	}, nil
}

// GetModelAccuracy retrieves model accuracy
func (pe *predictionEngine) GetModelAccuracy(modelID string) (float64, error) {
	model, exists := pe.models[modelID]
	if !exists {
		return 0, fmt.Errorf("model %s not found", modelID)
	}
	return model.Accuracy, nil
}

// UpdateModel updates a model with new data
func (pe *predictionEngine) UpdateModel(modelID string, newData []analytics.DataPoint) error {
	model, exists := pe.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	if len(newData) == 0 {
		return fmt.Errorf("no new data provided for model update")
	}

	// Sort new data
	sort.Slice(newData, func(i, j int) bool {
		return newData[i].Timestamp.Before(newData[j].Timestamp)
	})

	// Update model parameters based on new data
	switch model.Type {
	case "linear_regression":
		pe.trainLinearRegression(model, newData)
	case "moving_average":
		pe.trainMovingAverage(model, newData)
	case "trend_analysis":
		pe.trainTrendAnalysis(model, newData)
	}

	// Recalculate accuracy
	model.Accuracy = pe.calculateAccuracy(model, newData)
	model.UpdatedAt = time.Now()
	model.DataPoints += len(newData)
	model.LastValue = newData[len(newData)-1].Value

	pe.logger.WithFields(logrus.Fields{
		"model_id":        modelID,
		"new_accuracy":    model.Accuracy,
		"new_data_points": len(newData),
	}).Info("Updated prediction model")

	return nil
}

// GetAvailableModels retrieves available models
func (pe *predictionEngine) GetAvailableModels() ([]*analytics.ModelInfo, error) {
	models := make([]*analytics.ModelInfo, 0, len(pe.models))

	for _, model := range pe.models {
		models = append(models, &analytics.ModelInfo{
			ID:         model.ID,
			Type:       model.Type,
			Accuracy:   model.Accuracy,
			TrainedAt:  model.CreatedAt,
			Active:     true,
			DataPoints: model.DataPoints,
		})
	}

	return models, nil
}

// DeleteModel deletes a model
func (pe *predictionEngine) DeleteModel(modelID string) error {
	if _, exists := pe.models[modelID]; !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	delete(pe.models, modelID)
	pe.logger.WithField("model_id", modelID).Info("Deleted prediction model")
	return nil
}

// Helper methods for different prediction algorithms

func (pe *predictionEngine) trainLinearRegression(model *StoredModel, data []analytics.DataPoint) {
	n := float64(len(data))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, point := range data {
		x := float64(i) // Use index as X (time progression)
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate linear regression coefficients
	denominator := n*sumX2 - sumX*sumX
	if denominator != 0 {
		model.Slope = (n*sumXY - sumX*sumY) / denominator
		model.Intercept = (sumY - model.Slope*sumX) / n
	} else {
		model.Slope = 0
		model.Intercept = sumY / n
	}

	// Calculate mean and variance
	model.Mean = sumY / n
	sumSqDev := 0.0
	for _, point := range data {
		dev := point.Value - model.Mean
		sumSqDev += dev * dev
	}
	model.Variance = sumSqDev / n
	model.StdDev = math.Sqrt(model.Variance)
}

func (pe *predictionEngine) trainMovingAverage(model *StoredModel, data []analytics.DataPoint) {
	sum := 0.0
	for _, point := range data {
		sum += point.Value
	}

	model.Mean = sum / float64(len(data))

	// Calculate standard deviation
	sumSqDev := 0.0
	for _, point := range data {
		dev := point.Value - model.Mean
		sumSqDev += dev * dev
	}
	model.Variance = sumSqDev / float64(len(data))
	model.StdDev = math.Sqrt(model.Variance)

	// For moving average, intercept is the mean
	model.Intercept = model.Mean
	model.Slope = 0 // No trend in pure moving average
}

func (pe *predictionEngine) trainTrendAnalysis(model *StoredModel, data []analytics.DataPoint) {
	if len(data) < 2 {
		return
	}

	// Calculate trend as average change per time unit
	totalChange := 0.0
	totalTime := 0.0

	for i := 1; i < len(data); i++ {
		change := data[i].Value - data[i-1].Value
		duration := data[i].Timestamp.Sub(data[i-1].Timestamp).Hours()
		if duration > 0 {
			totalChange += change
			totalTime += duration
		}
	}

	if totalTime > 0 {
		model.Trend = totalChange / totalTime // Change per hour
	}

	// Calculate statistics
	pe.trainMovingAverage(model, data) // Get mean and std dev
}

func (pe *predictionEngine) calculateAccuracy(model *StoredModel, data []analytics.DataPoint) float64 {
	if len(data) < 2 {
		return 0.0
	}

	// Calculate R² (coefficient of determination)
	meanY := 0.0
	for _, point := range data {
		meanY += point.Value
	}
	meanY /= float64(len(data))

	ssRes := 0.0 // Sum of squares of residuals
	ssTot := 0.0 // Total sum of squares

	for i, point := range data {
		// Predict value using the model
		var predicted float64
		switch model.Type {
		case "linear_regression":
			predicted = model.Slope*float64(i) + model.Intercept
		case "moving_average":
			predicted = model.Mean
		case "trend_analysis":
			if i > 0 {
				timeDiff := data[i].Timestamp.Sub(data[0].Timestamp).Hours()
				predicted = data[0].Value + (model.Trend * timeDiff)
			} else {
				predicted = point.Value
			}
		default:
			predicted = model.Mean
		}

		residual := point.Value - predicted
		ssRes += residual * residual

		deviation := point.Value - meanY
		ssTot += deviation * deviation
	}

	if ssTot == 0 {
		return 1.0 // Perfect prediction if no variance
	}

	r2 := 1 - (ssRes / ssTot)

	// Ensure R² is within valid range [0, 1]
	if r2 < 0 {
		r2 = 0
	} else if r2 > 1 {
		r2 = 1
	}

	return r2
}

func (pe *predictionEngine) calculateConfidence(model *StoredModel) float64 {
	// Confidence based on accuracy and data quality
	confidence := model.Accuracy

	// Adjust confidence based on data quantity
	dataConfidence := math.Min(float64(model.DataPoints)/100.0, 1.0)
	confidence *= dataConfidence

	// Adjust confidence based on variability
	if model.Mean != 0 {
		variabilityPenalty := model.StdDev / math.Abs(model.Mean)
		confidence *= (1.0 - math.Min(variabilityPenalty, 0.5))
	}

	return math.Max(confidence, 0.1) // Minimum confidence of 10%
}
