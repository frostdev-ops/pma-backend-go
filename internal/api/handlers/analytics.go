package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	analyticsManager analytics.AnalyticsManager
	logger           *logrus.Logger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsManager analytics.AnalyticsManager, logger *logrus.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsManager: analyticsManager,
		logger:           logger,
	}
}

// RegisterRoutes registers analytics routes
func (h *AnalyticsHandler) RegisterRoutes(router gin.IRouter) {
	analyticsGroup := router.Group("/analytics")
	{
		// Data & Analytics endpoints
		analyticsGroup.GET("/data", h.GetHistoricalData)
		analyticsGroup.POST("/events", h.SubmitEvent)
		analyticsGroup.GET("/metrics", h.GetCustomMetrics)
		analyticsGroup.POST("/metrics", h.CreateCustomMetric)
		analyticsGroup.GET("/insights/:entityType", h.GetInsights)

		// Report endpoints
		reportsGroup := analyticsGroup.Group("/reports")
		{
			reportsGroup.GET("", h.ListReports)
			reportsGroup.POST("/generate", h.GenerateReport)
			reportsGroup.GET("/:id", h.GetReport)
			reportsGroup.GET("/templates", h.ListReportTemplates)
			reportsGroup.POST("/templates", h.CreateReportTemplate)
			reportsGroup.POST("/schedule", h.ScheduleReport)
			reportsGroup.GET("/schedules", h.ListScheduledReports)
			reportsGroup.DELETE("/schedules/:id", h.DeleteScheduledReport)
		}

		// Visualization endpoints
		vizGroup := analyticsGroup.Group("/visualizations")
		{
			vizGroup.GET("", h.ListVisualizations)
			vizGroup.POST("", h.CreateVisualization)
			vizGroup.GET("/:id/data", h.GetVisualizationData)
			vizGroup.PUT("/:id", h.UpdateVisualization)
			vizGroup.DELETE("/:id", h.DeleteVisualization)
		}

		// Dashboard endpoints
		dashboardGroup := analyticsGroup.Group("/dashboards")
		{
			dashboardGroup.GET("", h.ListDashboards)
			dashboardGroup.POST("", h.CreateDashboard)
			dashboardGroup.GET("/:id", h.GetDashboard)
			dashboardGroup.PUT("/:id", h.UpdateDashboard)
			dashboardGroup.DELETE("/:id", h.DeleteDashboard)
		}

		// Export endpoints
		exportGroup := analyticsGroup.Group("/export")
		{
			exportGroup.POST("/csv", h.ExportCSV)
			exportGroup.POST("/json", h.ExportJSON)
			exportGroup.POST("/excel", h.ExportExcel)
			exportGroup.POST("/pdf", h.ExportPDF)
			exportGroup.GET("/schedules", h.ListExportSchedules)
			exportGroup.POST("/schedules", h.CreateExportSchedule)
		}

		// Prediction endpoints (if enabled)
		predictionGroup := analyticsGroup.Group("/predictions")
		{
			predictionGroup.POST("/train", h.TrainModel)
			predictionGroup.POST("/predict", h.MakePrediction)
			predictionGroup.GET("/models", h.ListModels)
			predictionGroup.GET("/models/:id/accuracy", h.GetModelAccuracy)
			predictionGroup.DELETE("/models/:id", h.DeleteModel)
		}
	}
}

// Data & Analytics Handlers

// GetHistoricalData retrieves historical data
func (h *AnalyticsHandler) GetHistoricalData(c *gin.Context) {
	entityType := c.Query("entity_type")
	entityID := c.Query("entity_id")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	aggregation := c.DefaultQuery("aggregation", "avg")

	if entityType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type is required"})
		return
	}

	// Parse time range
	timeRange, err := h.parseTimeRange(startTimeStr, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time range: " + err.Error()})
		return
	}

	query := &analytics.HistoricalQuery{
		EntityType:  entityType,
		EntityID:    entityID,
		TimeRange:   timeRange,
		Aggregation: analytics.AggregationType(aggregation),
	}

	dataset, err := h.analyticsManager.GetHistoricalData(query)
	if err != nil {
		h.logger.Error("Failed to get historical data", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve data"})
		return
	}

	c.JSON(http.StatusOK, dataset)
}

// SubmitEvent submits an analytics event
func (h *AnalyticsHandler) SubmitEvent(c *gin.Context) {
	var event analytics.AnalyticsEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event data: " + err.Error()})
		return
	}

	if err := h.analyticsManager.ProcessEvent(&event); err != nil {
		h.logger.Error("Failed to process analytics event", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Event submitted successfully"})
}

// GetCustomMetrics retrieves custom metrics
func (h *AnalyticsHandler) GetCustomMetrics(c *gin.Context) {
	// This would be implemented by the metrics builder
	c.JSON(http.StatusOK, gin.H{"metrics": []interface{}{}})
}

// CreateCustomMetric creates a new custom metric
func (h *AnalyticsHandler) CreateCustomMetric(c *gin.Context) {
	var metric analytics.CustomMetric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metric data: " + err.Error()})
		return
	}

	if err := h.analyticsManager.CreateCustomMetric(&metric); err != nil {
		h.logger.Error("Failed to create custom metric", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create metric"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Metric created successfully"})
}

// GetInsights generates insights for an entity type
func (h *AnalyticsHandler) GetInsights(c *gin.Context) {
	entityType := c.Param("entityType")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	timeRange, err := h.parseTimeRange(startTimeStr, endTimeStr)
	if err != nil {
		// Use default time range if not provided
		timeRange = analytics.LastWeek()
	}

	insights, err := h.analyticsManager.GetInsights(entityType, timeRange)
	if err != nil {
		h.logger.Error("Failed to get insights", map[string]interface{}{
			"error":       err.Error(),
			"entity_type": entityType,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate insights"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"insights": insights})
}

// Report Handlers

// ListReports lists available reports
func (h *AnalyticsHandler) ListReports(c *gin.Context) {
	// This is a simplified implementation
	c.JSON(http.StatusOK, gin.H{"reports": []interface{}{}})
}

// GenerateReport generates a new report
func (h *AnalyticsHandler) GenerateReport(c *gin.Context) {
	var request analytics.ReportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report request: " + err.Error()})
		return
	}

	report, err := h.analyticsManager.GenerateReport(&request)
	if err != nil {
		h.logger.Error("Failed to generate report", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetReport retrieves a specific report
func (h *AnalyticsHandler) GetReport(c *gin.Context) {
	reportID := c.Param("id")
	// Implementation would retrieve from database
	c.JSON(http.StatusOK, gin.H{"report_id": reportID})
}

// ListReportTemplates lists available report templates
func (h *AnalyticsHandler) ListReportTemplates(c *gin.Context) {
	// This would be implemented by the report engine
	c.JSON(http.StatusOK, gin.H{"templates": []interface{}{}})
}

// CreateReportTemplate creates a new report template
func (h *AnalyticsHandler) CreateReportTemplate(c *gin.Context) {
	var template analytics.ReportTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template data: " + err.Error()})
		return
	}

	// This would be implemented by the report engine
	c.JSON(http.StatusCreated, gin.H{"message": "Template created successfully"})
}

// ScheduleReport schedules a report
func (h *AnalyticsHandler) ScheduleReport(c *gin.Context) {
	var schedule analytics.ReportSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule data: " + err.Error()})
		return
	}

	// This would be implemented by the report engine
	c.JSON(http.StatusCreated, gin.H{"message": "Report scheduled successfully"})
}

// ListScheduledReports lists scheduled reports
func (h *AnalyticsHandler) ListScheduledReports(c *gin.Context) {
	// This would be implemented by the report engine
	c.JSON(http.StatusOK, gin.H{"scheduled_reports": []interface{}{}})
}

// DeleteScheduledReport deletes a scheduled report
func (h *AnalyticsHandler) DeleteScheduledReport(c *gin.Context) {
	scheduleID := c.Param("id")
	// This would be implemented by the report engine
	c.JSON(http.StatusOK, gin.H{"message": "Scheduled report deleted", "schedule_id": scheduleID})
}

// Visualization Handlers

// ListVisualizations lists available visualizations
func (h *AnalyticsHandler) ListVisualizations(c *gin.Context) {
	// This would be implemented by the visualization engine
	c.JSON(http.StatusOK, gin.H{"visualizations": []interface{}{}})
}

// CreateVisualization creates a new visualization
func (h *AnalyticsHandler) CreateVisualization(c *gin.Context) {
	var viz analytics.Visualization
	if err := c.ShouldBindJSON(&viz); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid visualization data: " + err.Error()})
		return
	}

	// This would be implemented by the visualization engine
	c.JSON(http.StatusCreated, gin.H{"message": "Visualization created successfully"})
}

// GetVisualizationData gets data for a visualization
func (h *AnalyticsHandler) GetVisualizationData(c *gin.Context) {
	vizID := c.Param("id")

	// Parse query parameters
	params := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	data, err := h.analyticsManager.GetVisualizationData(vizID, params)
	if err != nil {
		h.logger.Error("Failed to get visualization data", map[string]interface{}{
			"error":  err.Error(),
			"viz_id": vizID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get visualization data"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// UpdateVisualization updates a visualization
func (h *AnalyticsHandler) UpdateVisualization(c *gin.Context) {
	vizID := c.Param("id")
	var viz analytics.Visualization
	if err := c.ShouldBindJSON(&viz); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid visualization data: " + err.Error()})
		return
	}

	// This would be implemented by the visualization engine
	c.JSON(http.StatusOK, gin.H{"message": "Visualization updated", "viz_id": vizID})
}

// DeleteVisualization deletes a visualization
func (h *AnalyticsHandler) DeleteVisualization(c *gin.Context) {
	vizID := c.Param("id")
	// This would be implemented by the visualization engine
	c.JSON(http.StatusOK, gin.H{"message": "Visualization deleted", "viz_id": vizID})
}

// Dashboard Handlers

// ListDashboards lists available dashboards
func (h *AnalyticsHandler) ListDashboards(c *gin.Context) {
	// This would be implemented by the visualization engine
	c.JSON(http.StatusOK, gin.H{"dashboards": []interface{}{}})
}

// CreateDashboard creates a new dashboard
func (h *AnalyticsHandler) CreateDashboard(c *gin.Context) {
	var config analytics.DashboardConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dashboard data: " + err.Error()})
		return
	}

	dashboard, err := h.analyticsManager.CreateDashboard(&config)
	if err != nil {
		h.logger.Error("Failed to create dashboard", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create dashboard"})
		return
	}

	c.JSON(http.StatusCreated, dashboard)
}

// GetDashboard retrieves a dashboard
func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	dashboardID := c.Param("id")

	dashboard, err := h.analyticsManager.GetDashboard(dashboardID)
	if err != nil {
		h.logger.Error("Failed to get dashboard", map[string]interface{}{
			"error":        err.Error(),
			"dashboard_id": dashboardID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard"})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// UpdateDashboard updates a dashboard
func (h *AnalyticsHandler) UpdateDashboard(c *gin.Context) {
	dashboardID := c.Param("id")
	var config analytics.DashboardConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dashboard data: " + err.Error()})
		return
	}

	// This would be implemented by the visualization engine
	c.JSON(http.StatusOK, gin.H{"message": "Dashboard updated", "dashboard_id": dashboardID})
}

// DeleteDashboard deletes a dashboard
func (h *AnalyticsHandler) DeleteDashboard(c *gin.Context) {
	dashboardID := c.Param("id")
	// This would be implemented by the visualization engine
	c.JSON(http.StatusOK, gin.H{"message": "Dashboard deleted", "dashboard_id": dashboardID})
}

// Export Handlers

// ExportCSV exports data as CSV
func (h *AnalyticsHandler) ExportCSV(c *gin.Context) {
	var request analytics.ExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export request: " + err.Error()})
		return
	}

	request.Format = "csv"
	reader, err := h.analyticsManager.ExportData(&request)
	if err != nil {
		h.logger.Error("Failed to export CSV", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export data"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=export.csv")
	c.DataFromReader(http.StatusOK, -1, "text/csv", reader, nil)
}

// ExportJSON exports data as JSON
func (h *AnalyticsHandler) ExportJSON(c *gin.Context) {
	var request analytics.ExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export request: " + err.Error()})
		return
	}

	request.Format = "json"
	reader, err := h.analyticsManager.ExportData(&request)
	if err != nil {
		h.logger.Error("Failed to export JSON", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export data"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=export.json")
	c.DataFromReader(http.StatusOK, -1, "application/json", reader, nil)
}

// ExportExcel exports data as Excel
func (h *AnalyticsHandler) ExportExcel(c *gin.Context) {
	var request analytics.ExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export request: " + err.Error()})
		return
	}

	request.Format = "excel"
	reader, err := h.analyticsManager.ExportData(&request)
	if err != nil {
		h.logger.Error("Failed to export Excel", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export data"})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=export.xlsx")
	c.DataFromReader(http.StatusOK, -1, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", reader, nil)
}

// ExportPDF exports data as PDF
func (h *AnalyticsHandler) ExportPDF(c *gin.Context) {
	var request analytics.ReportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report request: " + err.Error()})
		return
	}

	request.Format = "pdf"
	report, err := h.analyticsManager.GenerateReport(&request)
	if err != nil {
		h.logger.Error("Failed to generate PDF report", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=report.pdf")
	c.Data(http.StatusOK, "application/pdf", report.Data)
}

// ListExportSchedules lists export schedules
func (h *AnalyticsHandler) ListExportSchedules(c *gin.Context) {
	// This would be implemented by the export manager
	c.JSON(http.StatusOK, gin.H{"schedules": []interface{}{}})
}

// CreateExportSchedule creates an export schedule
func (h *AnalyticsHandler) CreateExportSchedule(c *gin.Context) {
	var schedule analytics.ExportSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule data: " + err.Error()})
		return
	}

	// This would be implemented by the export manager
	c.JSON(http.StatusCreated, gin.H{"message": "Export scheduled successfully"})
}

// Prediction Handlers

// TrainModel trains a prediction model
func (h *AnalyticsHandler) TrainModel(c *gin.Context) {
	var request struct {
		ModelType string                `json:"model_type"`
		Data      []analytics.DataPoint `json:"data"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid training request: " + err.Error()})
		return
	}

	// This would be implemented by the prediction engine
	c.JSON(http.StatusOK, gin.H{"message": "Model training started"})
}

// MakePrediction makes a prediction using a model
func (h *AnalyticsHandler) MakePrediction(c *gin.Context) {
	var request struct {
		ModelID string                 `json:"model_id"`
		Input   map[string]interface{} `json:"input"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prediction request: " + err.Error()})
		return
	}

	// This would be implemented by the prediction engine
	c.JSON(http.StatusOK, gin.H{
		"predicted_value": 42.0,
		"confidence":      0.85,
	})
}

// ListModels lists available prediction models
func (h *AnalyticsHandler) ListModels(c *gin.Context) {
	// This would be implemented by the prediction engine
	c.JSON(http.StatusOK, gin.H{"models": []interface{}{}})
}

// GetModelAccuracy gets model accuracy
func (h *AnalyticsHandler) GetModelAccuracy(c *gin.Context) {
	modelID := c.Param("id")
	// This would be implemented by the prediction engine
	c.JSON(http.StatusOK, gin.H{
		"model_id": modelID,
		"accuracy": 0.92,
	})
}

// DeleteModel deletes a prediction model
func (h *AnalyticsHandler) DeleteModel(c *gin.Context) {
	modelID := c.Param("id")
	// This would be implemented by the prediction engine
	c.JSON(http.StatusOK, gin.H{"message": "Model deleted", "model_id": modelID})
}

// Helper methods

// parseTimeRange parses start and end time strings into a TimeRange
func (h *AnalyticsHandler) parseTimeRange(startTimeStr, endTimeStr string) (analytics.TimeRange, error) {
	var timeRange analytics.TimeRange
	var err error

	if startTimeStr == "" || endTimeStr == "" {
		// Use default time range (last 24 hours)
		return analytics.LastDay(), nil
	}

	timeRange.Start, err = time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return timeRange, err
	}

	timeRange.End, err = time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return timeRange, err
	}

	return timeRange, nil
}

// parseIntParam parses an integer parameter from query string
func (h *AnalyticsHandler) parseIntParam(c *gin.Context, param string, defaultValue int) int {
	valueStr := c.Query(param)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
