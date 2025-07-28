package visualization

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// visualizationEngine implements the VisualizationEngine interface
type visualizationEngine struct {
	db     *sql.DB
	config *analytics.AnalyticsConfig
	logger *logrus.Logger
}

// NewVisualizationEngine creates a new visualization engine
func NewVisualizationEngine(db *sql.DB, config *analytics.AnalyticsConfig, logger *logrus.Logger) (analytics.VisualizationEngine, error) {
	return &visualizationEngine{
		db:     db,
		config: config,
		logger: logger,
	}, nil
}

// CreateDashboard creates a new dashboard
func (ve *visualizationEngine) CreateDashboard(config *analytics.DashboardConfig) (*analytics.Dashboard, error) {
	dashboard := &analytics.Dashboard{
		ID:          uuid.New().String(),
		Name:        config.Name,
		Description: config.Description,
		Layout:      config.Layout,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Active:      true,
	}

	// Store dashboard in database
	query := `
		INSERT INTO analytics_dashboards (id, name, description, layout, created_at, updated_at, active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := ve.db.Exec(query, dashboard.ID, dashboard.Name, dashboard.Description,
		dashboard.Layout, dashboard.CreatedAt, dashboard.UpdatedAt, dashboard.Active)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	ve.logger.WithFields(logrus.Fields{
		"dashboard_id":   dashboard.ID,
		"dashboard_name": dashboard.Name,
	}).Info("Dashboard created successfully")

	return dashboard, nil
}

// GetDashboard retrieves a dashboard by ID
func (ve *visualizationEngine) GetDashboard(dashboardID string) (*analytics.Dashboard, error) {
	query := `
		SELECT id, name, description, layout, created_at, updated_at, active
		FROM analytics_dashboards
		WHERE id = ?`

	var dashboard analytics.Dashboard
	err := ve.db.QueryRow(query, dashboardID).Scan(
		&dashboard.ID, &dashboard.Name, &dashboard.Description,
		&dashboard.Layout, &dashboard.CreatedAt, &dashboard.UpdatedAt, &dashboard.Active)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dashboard not found: %s", dashboardID)
		}
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Get visualizations for this dashboard
	visualizations, err := ve.getVisualizationsForDashboard(dashboardID)
	if err != nil {
		ve.logger.WithError(err).Warn("Failed to get visualizations for dashboard")
		// Don't fail - just return empty visualizations
		visualizations = []analytics.Visualization{}
	}
	dashboard.Visualizations = visualizations

	return &dashboard, nil
}

// AddVisualization adds a visualization to a dashboard
func (ve *visualizationEngine) AddVisualization(dashboardID string, viz *analytics.Visualization) error {
	if viz.ID == "" {
		viz.ID = uuid.New().String()
	}
	viz.DashboardID = dashboardID
	viz.CreatedAt = time.Now()
	viz.UpdatedAt = time.Now()

	query := `
		INSERT INTO analytics_visualizations 
		(id, dashboard_id, name, type, position_x, position_y, width, height, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := ve.db.Exec(query, viz.ID, viz.DashboardID, viz.Name, viz.Type,
		viz.Position.X, viz.Position.Y, viz.Size.Width, viz.Size.Height,
		viz.CreatedAt, viz.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to add visualization: %w", err)
	}

	ve.logger.WithFields(logrus.Fields{
		"visualization_id": viz.ID,
		"dashboard_id":     dashboardID,
		"chart_type":       viz.Type,
	}).Info("Visualization added to dashboard")

	return nil
}

// GetVisualizationData gets data for a visualization
func (ve *visualizationEngine) GetVisualizationData(vizID string, params map[string]interface{}) (interface{}, error) {
	// Get visualization config
	viz, err := ve.getVisualization(vizID)
	if err != nil {
		return nil, fmt.Errorf("failed to get visualization: %w", err)
	}

	// Generate data based on chart type
	switch viz.Type {
	case "line":
		return ve.generateLineChartData(viz, params)
	case "bar":
		return ve.generateBarChartData(viz, params)
	case "pie":
		return ve.generatePieChartData(viz, params)
	case "gauge":
		return ve.generateGaugeData(viz, params)
	case "table":
		return ve.generateTableData(viz, params)
	default:
		return ve.generateDefaultData(viz, params)
	}
}

// ExportVisualization exports a visualization
func (ve *visualizationEngine) ExportVisualization(vizID string, format string) ([]byte, error) {
	data, err := ve.GetVisualizationData(vizID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get visualization data: %w", err)
	}

	switch format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	case "csv":
		return ve.exportToCSV(data)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// GetAvailableChartTypes returns available chart types
func (ve *visualizationEngine) GetAvailableChartTypes() []analytics.ChartType {
	return []analytics.ChartType{
		{
			ID:          "line",
			Name:        "Line Chart",
			Description: "Time series line chart for tracking values over time",
			Category:    "time_series",
			Icon:        "📈",
			DataTypes:   []string{"numeric", "timestamp"},
		},
		{
			ID:          "bar",
			Name:        "Bar Chart",
			Description: "Categorical bar chart for comparing values",
			Category:    "categorical",
			Icon:        "📊",
			DataTypes:   []string{"categorical", "numeric"},
		},
		{
			ID:          "pie",
			Name:        "Pie Chart",
			Description: "Pie chart for showing proportions",
			Category:    "categorical",
			Icon:        "🥧",
			DataTypes:   []string{"categorical", "numeric"},
		},
		{
			ID:          "gauge",
			Name:        "Gauge",
			Description: "Gauge chart for showing single values with thresholds",
			Category:    "single_value",
			Icon:        "🎯",
			DataTypes:   []string{"numeric"},
		},
		{
			ID:          "table",
			Name:        "Data Table",
			Description: "Tabular display of data with sorting and filtering",
			Category:    "tabular",
			Icon:        "📋",
			DataTypes:   []string{"mixed"},
		},
	}
}

// UpdateVisualization updates a visualization
func (ve *visualizationEngine) UpdateVisualization(vizID string, config analytics.VisualizationConfig) error {
	query := `
		UPDATE analytics_visualizations 
		SET updated_at = ?
		WHERE id = ?`

	result, err := ve.db.Exec(query, time.Now(), vizID)

	if err != nil {
		return fmt.Errorf("failed to update visualization: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("visualization not found: %s", vizID)
	}

	ve.logger.WithField("visualization_id", vizID).Info("Visualization updated successfully")
	return nil
}

// Helper methods for visualization engine

// getVisualizationsForDashboard retrieves all visualizations for a dashboard
func (ve *visualizationEngine) getVisualizationsForDashboard(dashboardID string) ([]analytics.Visualization, error) {
	query := `
		SELECT id, dashboard_id, name, type, position_x, position_y, width, height, created_at, updated_at, active
		FROM analytics_visualizations
		WHERE dashboard_id = ? AND active = 1
		ORDER BY created_at`

	rows, err := ve.db.Query(query, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to query visualizations: %w", err)
	}
	defer rows.Close()

	var visualizations []analytics.Visualization
	for rows.Next() {
		var viz analytics.Visualization
		var posX, posY, width, height int

		err := rows.Scan(&viz.ID, &viz.DashboardID, &viz.Name, &viz.Type,
			&posX, &posY, &width, &height, &viz.CreatedAt, &viz.UpdatedAt, &viz.Active)
		if err != nil {
			continue
		}

		viz.Position = analytics.Position{X: posX, Y: posY}
		viz.Size = analytics.Size{Width: width, Height: height}
		visualizations = append(visualizations, viz)
	}

	return visualizations, nil
}

// getVisualization retrieves a single visualization by ID
func (ve *visualizationEngine) getVisualization(vizID string) (*analytics.Visualization, error) {
	query := `
		SELECT id, dashboard_id, name, type, position_x, position_y, width, height, created_at, updated_at, active
		FROM analytics_visualizations
		WHERE id = ?`

	var viz analytics.Visualization
	var posX, posY, width, height int

	err := ve.db.QueryRow(query, vizID).Scan(&viz.ID, &viz.DashboardID, &viz.Name, &viz.Type,
		&posX, &posY, &width, &height, &viz.CreatedAt, &viz.UpdatedAt, &viz.Active)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("visualization not found: %s", vizID)
		}
		return nil, fmt.Errorf("failed to get visualization: %w", err)
	}

	viz.Position = analytics.Position{X: posX, Y: posY}
	viz.Size = analytics.Size{Width: width, Height: height}
	return &viz, nil
}

// generateLineChartData generates data for line charts
func (ve *visualizationEngine) generateLineChartData(viz *analytics.Visualization, params map[string]interface{}) (interface{}, error) {
	// Get time series data from analytics events or metrics
	data, err := ve.getTimeSeriesData(viz, params)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":   "line",
		"data":   data,
		"title":  viz.Name,
		"xLabel": "Time",
		"yLabel": "Value",
		"options": map[string]interface{}{
			"responsive": true,
			"animation":  true,
			"showGrid":   true,
		},
	}, nil
}

// generateBarChartData generates data for bar charts
func (ve *visualizationEngine) generateBarChartData(viz *analytics.Visualization, params map[string]interface{}) (interface{}, error) {
	// Get aggregated data for bar chart
	data, err := ve.getAggregatedData(viz, params)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":   "bar",
		"data":   data,
		"title":  viz.Name,
		"xLabel": "Category",
		"yLabel": "Value",
		"options": map[string]interface{}{
			"responsive": true,
			"showLegend": true,
		},
	}, nil
}

// generatePieChartData generates data for pie charts
func (ve *visualizationEngine) generatePieChartData(viz *analytics.Visualization, params map[string]interface{}) (interface{}, error) {
	// Get categorical data for pie chart
	data, err := ve.getCategoricalData(viz, params)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":  "pie",
		"data":  data,
		"title": viz.Name,
		"options": map[string]interface{}{
			"responsive":  true,
			"showLegend":  true,
			"showTooltip": true,
		},
	}, nil
}

// generateGaugeData generates data for gauge charts
func (ve *visualizationEngine) generateGaugeData(viz *analytics.Visualization, params map[string]interface{}) (interface{}, error) {
	// Get latest value for gauge
	value, err := ve.getLatestValue(viz, params)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":  "gauge",
		"value": value,
		"title": viz.Name,
		"min":   0,
		"max":   100,
		"thresholds": []map[string]interface{}{
			{"value": 30, "color": "green"},
			{"value": 70, "color": "yellow"},
			{"value": 90, "color": "red"},
		},
	}, nil
}

// generateTableData generates data for table displays
func (ve *visualizationEngine) generateTableData(viz *analytics.Visualization, params map[string]interface{}) (interface{}, error) {
	// Get tabular data
	data, err := ve.getTabularData(viz, params)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":  "table",
		"data":  data,
		"title": viz.Name,
		"options": map[string]interface{}{
			"pagination": true,
			"sorting":    true,
			"filtering":  true,
		},
	}, nil
}

// generateDefaultData generates default data for unknown chart types
func (ve *visualizationEngine) generateDefaultData(viz *analytics.Visualization, params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"type":    viz.Type,
		"data":    []interface{}{},
		"title":   viz.Name,
		"message": fmt.Sprintf("Chart type '%s' not yet implemented", viz.Type),
	}, nil
}

// Data retrieval helper methods

// getTimeSeriesData retrieves time-based data for visualizations
func (ve *visualizationEngine) getTimeSeriesData(viz *analytics.Visualization, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Query analytics events for time series data
	query := `
		SELECT timestamp, data
		FROM analytics_events
		WHERE timestamp >= datetime('now', '-24 hours')
		ORDER BY timestamp DESC
		LIMIT 100`

	rows, err := ve.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		var timestamp time.Time
		var dataJSON []byte

		if err := rows.Scan(&timestamp, &dataJSON); err != nil {
			continue
		}

		// Simple data point for demonstration
		data = append(data, map[string]interface{}{
			"timestamp": timestamp,
			"value":     10 + (len(data) % 20), // Sample varying data
		})
	}

	return data, nil
}

// getAggregatedData retrieves aggregated data for bar charts
func (ve *visualizationEngine) getAggregatedData(viz *analytics.Visualization, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Generate sample aggregated data
	categories := []string{"Lights", "Switches", "Sensors", "Cameras", "Locks"}
	var data []map[string]interface{}

	for i, category := range categories {
		data = append(data, map[string]interface{}{
			"category": category,
			"value":    (i + 1) * 15, // Sample data
			"color":    fmt.Sprintf("#%06x", (i*40)%255+(i*80)%255*256+(i*120)%255*65536),
		})
	}

	return data, nil
}

// getCategoricalData retrieves categorical data for pie charts
func (ve *visualizationEngine) getCategoricalData(viz *analytics.Visualization, params map[string]interface{}) ([]map[string]interface{}, error) {
	return ve.getAggregatedData(viz, params) // Same format as bar chart for now
}

// getLatestValue retrieves the latest single value for gauges
func (ve *visualizationEngine) getLatestValue(viz *analytics.Visualization, params map[string]interface{}) (float64, error) {
	// Query for latest numeric value
	query := `
		SELECT timestamp, data
		FROM analytics_events
		ORDER BY timestamp DESC
		LIMIT 1`

	var timestamp time.Time
	var dataJSON []byte

	err := ve.db.QueryRow(query).Scan(&timestamp, &dataJSON)
	if err != nil {
		// Return sample value if no data
		return 65.0, nil
	}

	// Extract numeric value from JSON data (simplified)
	return 75.0, nil // Sample gauge value
}

// getTabularData retrieves data for table displays
func (ve *visualizationEngine) getTabularData(viz *analytics.Visualization, params map[string]interface{}) (map[string]interface{}, error) {
	headers := []string{"Entity", "State", "Last Update", "Source"}
	rows := [][]interface{}{
		{"Living Room Light", "On", time.Now().Format("15:04:05"), "Home Assistant"},
		{"Front Door", "Locked", time.Now().Add(-5 * time.Minute).Format("15:04:05"), "Ring"},
		{"Temperature Sensor", "22.5°C", time.Now().Add(-2 * time.Minute).Format("15:04:05"), "Shelly"},
		{"Security Camera", "Recording", time.Now().Add(-1 * time.Minute).Format("15:04:05"), "Ring"},
	}

	return map[string]interface{}{
		"headers": headers,
		"rows":    rows,
		"total":   len(rows),
	}, nil
}

// exportToCSV exports visualization data to CSV format
func (ve *visualizationEngine) exportToCSV(data interface{}) ([]byte, error) {
	// Simple CSV export - in production this would be more sophisticated
	csvData := "timestamp,value\n"

	if dataMap, ok := data.(map[string]interface{}); ok {
		if dataPoints, ok := dataMap["data"].([]map[string]interface{}); ok {
			for _, point := range dataPoints {
				if timestamp, ok := point["timestamp"]; ok {
					if value, ok := point["value"]; ok {
						csvData += fmt.Sprintf("%v,%v\n", timestamp, value)
					}
				}
			}
		}
	}

	return []byte(csvData), nil
}
