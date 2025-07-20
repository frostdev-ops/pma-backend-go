package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/errors"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler handles error monitoring and recovery endpoints
type ErrorHandler struct {
	recoveryManager *errors.RecoveryManager
	logger          *logrus.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(recoveryManager *errors.RecoveryManager, logger *logrus.Logger) *ErrorHandler {
	return &ErrorHandler{
		recoveryManager: recoveryManager,
		logger:          logger,
	}
}

// RegisterRoutes registers error monitoring routes
func (eh *ErrorHandler) RegisterRoutes(router *gin.RouterGroup) {
	errorGroup := router.Group("/errors")
	{
		errorGroup.GET("/reports", eh.GetErrorReports)
		errorGroup.GET("/reports/:error_id", eh.GetErrorReport)
		errorGroup.POST("/reports/:error_id/resolve", eh.ResolveError)
		errorGroup.GET("/stats", eh.GetErrorStats)
		errorGroup.GET("/recovery/metrics", eh.GetRecoveryMetrics)
		errorGroup.GET("/recovery/circuit-breakers", eh.GetCircuitBreakerStatus)
		errorGroup.POST("/recovery/circuit-breakers/:name/reset", eh.ResetCircuitBreaker)
		errorGroup.GET("/health", eh.GetErrorHealthStatus)
		errorGroup.POST("/cleanup", eh.CleanupOldErrors)
		errorGroup.POST("/test", eh.TestErrorRecovery)
	}
}

// GetErrorReports returns paginated error reports
func (eh *ErrorHandler) GetErrorReports(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	category := c.Query("category")
	severity := c.Query("severity")
	resolved := c.Query("resolved")
	component := c.Query("component")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 50
	}

	// Get all error reports
	allReports := eh.recoveryManager.GetErrorReporter().GetErrorReports()

	// Filter reports
	var filteredReports []errors.ErrorReport
	for _, report := range allReports {
		// Apply filters
		if category != "" && string(report.Error.Category) != category {
			continue
		}
		if severity != "" && string(report.Error.Severity) != severity {
			continue
		}
		if resolved != "" {
			isResolved := report.Resolved
			if resolved == "true" && !isResolved {
				continue
			}
			if resolved == "false" && isResolved {
				continue
			}
		}
		if component != "" && report.Error.Context.Component != component {
			continue
		}

		filteredReports = append(filteredReports, report)
	}

	// Calculate pagination
	total := len(filteredReports)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		filteredReports = []errors.ErrorReport{}
	} else if end > total {
		filteredReports = filteredReports[start:]
	} else {
		filteredReports = filteredReports[start:end]
	}

	// Prepare response
	response := map[string]interface{}{
		"reports": filteredReports,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
		"filters": map[string]interface{}{
			"category":  category,
			"severity":  severity,
			"resolved":  resolved,
			"component": component,
		},
	}

	utils.SendSuccess(c, response)
}

// GetErrorReport returns a specific error report by ID
func (eh *ErrorHandler) GetErrorReport(c *gin.Context) {
	errorID := c.Param("error_id")
	if errorID == "" {
		utils.SendError(c, http.StatusBadRequest, "Error ID is required")
		return
	}

	// Find the error report
	reports := eh.recoveryManager.GetErrorReporter().GetErrorReports()
	for _, report := range reports {
		if report.Error.ErrorID == errorID {
			utils.SendSuccess(c, report)
			return
		}
	}

	utils.SendError(c, http.StatusNotFound, "Error report not found")
}

// ResolveError marks an error as resolved
func (eh *ErrorHandler) ResolveError(c *gin.Context) {
	errorID := c.Param("error_id")
	if errorID == "" {
		utils.SendError(c, http.StatusBadRequest, "Error ID is required")
		return
	}

	// Parse request body for resolution details
	var request struct {
		ResolvedBy string `json:"resolved_by"`
		Notes      string `json:"notes"`
	}
	c.ShouldBindJSON(&request)

	// Mark error as resolved
	success := eh.recoveryManager.GetErrorReporter().ResolveError(errorID)
	if !success {
		utils.SendError(c, http.StatusNotFound, "Error report not found")
		return
	}

	// Log the resolution
	eh.logger.WithFields(logrus.Fields{
		"error_id":    errorID,
		"resolved_by": request.ResolvedBy,
		"notes":       request.Notes,
	}).Info("Error report resolved")

	utils.SendSuccess(c, map[string]interface{}{
		"message":     "Error report resolved successfully",
		"error_id":    errorID,
		"resolved_at": time.Now(),
		"resolved_by": request.ResolvedBy,
	})
}

// GetErrorStats returns error statistics and trends
func (eh *ErrorHandler) GetErrorStats(c *gin.Context) {
	// Get basic error stats
	stats := eh.recoveryManager.GetErrorReporter().GetErrorStats()

	// Add additional analytics
	reports := eh.recoveryManager.GetErrorReporter().GetErrorReports()

	// Calculate time-based statistics
	now := time.Now()
	stats["time_analysis"] = map[string]interface{}{
		"last_hour":  eh.countErrorsInTimeRange(reports, now.Add(-time.Hour), now),
		"last_day":   eh.countErrorsInTimeRange(reports, now.Add(-24*time.Hour), now),
		"last_week":  eh.countErrorsInTimeRange(reports, now.Add(-7*24*time.Hour), now),
		"last_month": eh.countErrorsInTimeRange(reports, now.Add(-30*24*time.Hour), now),
	}

	// Calculate top error sources
	componentStats := make(map[string]int)
	operationStats := make(map[string]int)

	for _, report := range reports {
		if !report.Resolved {
			if report.Error.Context != nil {
				if report.Error.Context.Component != "" {
					componentStats[report.Error.Context.Component]++
				}
				if report.Error.Context.Operation != "" {
					operationStats[report.Error.Context.Operation]++
				}
			}
		}
	}

	stats["top_components"] = componentStats
	stats["top_operations"] = operationStats

	// Calculate error trends
	stats["trends"] = eh.calculateErrorTrends(reports)

	utils.SendSuccess(c, stats)
}

// GetRecoveryMetrics returns comprehensive recovery system metrics
func (eh *ErrorHandler) GetRecoveryMetrics(c *gin.Context) {
	metrics := eh.recoveryManager.GetMetrics()

	// Add additional recovery metrics
	reports := eh.recoveryManager.GetErrorReporter().GetErrorReports()

	recoveryMetrics := map[string]interface{}{
		"total_errors_tracked":  len(reports),
		"recovery_success_rate": eh.calculateRecoverySuccessRate(reports),
		"avg_resolution_time":   eh.calculateAverageResolutionTime(reports),
		"circuit_breaker_count": len(metrics["circuit_breakers"].(map[string]interface{})),
		"active_incidents":      eh.countActiveIncidents(reports),
	}

	// Merge with existing metrics
	for k, v := range recoveryMetrics {
		metrics[k] = v
	}

	utils.SendSuccess(c, metrics)
}

// GetCircuitBreakerStatus returns the status of all circuit breakers
func (eh *ErrorHandler) GetCircuitBreakerStatus(c *gin.Context) {
	metrics := eh.recoveryManager.GetMetrics()
	circuitBreakers := metrics["circuit_breakers"].(map[string]interface{})

	// Enhance circuit breaker data with additional analysis
	enhancedStatus := make(map[string]interface{})
	for name, cbMetrics := range circuitBreakers {
		cbData := cbMetrics.(map[string]interface{})

		// Add health assessment
		state := cbData["state"].(string)
		failures := cbData["failures"].(int64)
		maxFailures := cbData["max_failures"].(int)

		healthStatus := "healthy"
		if state == "open" {
			healthStatus = "unhealthy"
		} else if state == "half-open" {
			healthStatus = "recovering"
		} else if failures > int64(maxFailures/2) {
			healthStatus = "warning"
		}

		cbData["health_status"] = healthStatus
		cbData["failure_rate"] = float64(failures) / float64(maxFailures)

		enhancedStatus[name] = cbData
	}

	utils.SendSuccess(c, enhancedStatus)
}

// ResetCircuitBreaker manually resets a circuit breaker
func (eh *ErrorHandler) ResetCircuitBreaker(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.SendError(c, http.StatusBadRequest, "Circuit breaker name is required")
		return
	}

	// Note: This would require adding a Reset method to the CircuitBreaker
	// For now, we'll simulate the operation
	eh.logger.WithField("circuit_breaker", name).Info("Circuit breaker reset requested")

	utils.SendSuccess(c, map[string]interface{}{
		"message": "Circuit breaker reset request received",
		"name":    name,
		"note":    "Manual reset functionality would be implemented here",
	})
}

// GetErrorHealthStatus returns overall error system health
func (eh *ErrorHandler) GetErrorHealthStatus(c *gin.Context) {
	reports := eh.recoveryManager.GetErrorReporter().GetErrorReports()
	metrics := eh.recoveryManager.GetMetrics()

	// Calculate health indicators
	now := time.Now()
	recentErrors := eh.countErrorsInTimeRange(reports, now.Add(-time.Hour), now)
	criticalErrors := eh.countCriticalErrors(reports)
	unresolvedErrors := eh.countUnresolvedErrors(reports)

	// Determine overall health status
	healthStatus := "healthy"
	if criticalErrors > 0 {
		healthStatus = "critical"
	} else if recentErrors > 10 {
		healthStatus = "warning"
	} else if unresolvedErrors > 5 {
		healthStatus = "degraded"
	}

	// Check circuit breaker health
	circuitBreakers := metrics["circuit_breakers"].(map[string]interface{})
	openCircuitBreakers := 0
	for _, cbMetrics := range circuitBreakers {
		cbData := cbMetrics.(map[string]interface{})
		if cbData["state"].(string) == "open" {
			openCircuitBreakers++
		}
	}

	if openCircuitBreakers > 0 {
		if healthStatus == "healthy" {
			healthStatus = "degraded"
		}
	}

	health := map[string]interface{}{
		"status":                 healthStatus,
		"timestamp":              time.Now(),
		"recent_errors":          recentErrors,
		"critical_errors":        criticalErrors,
		"unresolved_errors":      unresolvedErrors,
		"open_circuit_breakers":  openCircuitBreakers,
		"total_circuit_breakers": len(circuitBreakers),
		"recommendations":        eh.generateHealthRecommendations(healthStatus, recentErrors, criticalErrors, unresolvedErrors, openCircuitBreakers),
	}

	utils.SendSuccess(c, health)
}

// CleanupOldErrors removes old resolved error reports
func (eh *ErrorHandler) CleanupOldErrors(c *gin.Context) {
	var request struct {
		MaxAgeDays int `json:"max_age_days" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if request.MaxAgeDays < 1 || request.MaxAgeDays > 365 {
		utils.SendError(c, http.StatusBadRequest, "Max age must be between 1 and 365 days")
		return
	}

	maxAge := time.Duration(request.MaxAgeDays) * 24 * time.Hour
	removed := eh.recoveryManager.GetErrorReporter().ClearOldErrors(maxAge)

	eh.logger.WithFields(logrus.Fields{
		"max_age_days":  request.MaxAgeDays,
		"removed_count": removed,
	}).Info("Old error reports cleaned up")

	utils.SendSuccess(c, map[string]interface{}{
		"message":       "Old error reports cleaned up successfully",
		"removed_count": removed,
		"max_age_days":  request.MaxAgeDays,
	})
}

// TestErrorRecovery tests the error recovery system
func (eh *ErrorHandler) TestErrorRecovery(c *gin.Context) {
	var request struct {
		ErrorType string `json:"error_type"`
		Component string `json:"component"`
		Simulate  bool   `json:"simulate"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if request.ErrorType == "" {
		request.ErrorType = "test"
	}
	if request.Component == "" {
		request.Component = "error_handler_test"
	}

	// Create a test error
	testErr := errors.NewEnhanced(
		http.StatusInternalServerError,
		"Test error for recovery system",
		errors.CategoryInternal,
		errors.SeverityMedium,
	).WithContext(&errors.ErrorContext{
		Component: request.Component,
		Operation: "test_error_recovery",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"test_type": request.ErrorType,
			"simulated": request.Simulate,
		},
	})

	// Report the error if not simulated
	if !request.Simulate {
		eh.recoveryManager.GetErrorReporter().ReportError(testErr)
	}

	eh.logger.WithFields(logrus.Fields{
		"error_type": request.ErrorType,
		"component":  request.Component,
		"simulated":  request.Simulate,
	}).Info("Error recovery test executed")

	utils.SendSuccess(c, map[string]interface{}{
		"message":    "Error recovery test completed",
		"error_type": request.ErrorType,
		"component":  request.Component,
		"simulated":  request.Simulate,
		"error_id":   testErr.ErrorID,
	})
}

// Helper methods

func (eh *ErrorHandler) countErrorsInTimeRange(reports []errors.ErrorReport, start, end time.Time) int {
	count := 0
	for _, report := range reports {
		if report.Timestamp.After(start) && report.Timestamp.Before(end) {
			count++
		}
	}
	return count
}

func (eh *ErrorHandler) countCriticalErrors(reports []errors.ErrorReport) int {
	count := 0
	for _, report := range reports {
		if report.Error.Severity == errors.SeverityCritical && !report.Resolved {
			count++
		}
	}
	return count
}

func (eh *ErrorHandler) countUnresolvedErrors(reports []errors.ErrorReport) int {
	count := 0
	for _, report := range reports {
		if !report.Resolved {
			count++
		}
	}
	return count
}

func (eh *ErrorHandler) countActiveIncidents(reports []errors.ErrorReport) int {
	count := 0
	for _, report := range reports {
		if !report.Resolved && (report.Error.Severity == errors.SeverityCritical || report.Error.Severity == errors.SeverityHigh) {
			count++
		}
	}
	return count
}

func (eh *ErrorHandler) calculateRecoverySuccessRate(reports []errors.ErrorReport) float64 {
	if len(reports) == 0 {
		return 100.0
	}

	resolved := 0
	for _, report := range reports {
		if report.Resolved {
			resolved++
		}
	}

	return float64(resolved) / float64(len(reports)) * 100.0
}

func (eh *ErrorHandler) calculateAverageResolutionTime(reports []errors.ErrorReport) float64 {
	var totalTime time.Duration
	count := 0

	for _, report := range reports {
		if report.Resolved && report.ResolutionTime != nil {
			resolutionTime := report.ResolutionTime.Sub(report.FirstSeen)
			totalTime += resolutionTime
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalTime.Seconds() / float64(count)
}

func (eh *ErrorHandler) calculateErrorTrends(reports []errors.ErrorReport) map[string]interface{} {
	now := time.Now()

	// Calculate hourly trends for the last 24 hours
	hourlyTrends := make([]int, 24)
	for _, report := range reports {
		if report.Timestamp.After(now.Add(-24 * time.Hour)) {
			hour := int(now.Sub(report.Timestamp).Hours())
			if hour >= 0 && hour < 24 {
				hourlyTrends[23-hour]++ // Reverse order for chronological display
			}
		}
	}

	return map[string]interface{}{
		"hourly_last_24h": hourlyTrends,
		"trend_direction": eh.calculateTrendDirection(hourlyTrends),
	}
}

func (eh *ErrorHandler) calculateTrendDirection(hourlyData []int) string {
	if len(hourlyData) < 6 {
		return "insufficient_data"
	}

	// Compare recent 6 hours with previous 6 hours
	recentSum := 0
	previousSum := 0

	for i := len(hourlyData) - 6; i < len(hourlyData); i++ {
		recentSum += hourlyData[i]
	}

	for i := len(hourlyData) - 12; i < len(hourlyData)-6; i++ {
		previousSum += hourlyData[i]
	}

	if float64(recentSum) > float64(previousSum)*1.2 {
		return "increasing"
	} else if float64(recentSum) < float64(previousSum)*0.8 {
		return "decreasing"
	}
	return "stable"
}

func (eh *ErrorHandler) generateHealthRecommendations(status string, recentErrors, criticalErrors, unresolvedErrors, openCircuitBreakers int) []string {
	var recommendations []string

	switch status {
	case "critical":
		recommendations = append(recommendations, "Immediate attention required: Critical errors detected")
		recommendations = append(recommendations, "Review and resolve critical errors immediately")
		recommendations = append(recommendations, "Consider enabling emergency protocols")

	case "warning":
		recommendations = append(recommendations, "Monitor system closely: High error rate detected")
		if recentErrors > 20 {
			recommendations = append(recommendations, "Consider implementing additional rate limiting")
		}

	case "degraded":
		if unresolvedErrors > 10 {
			recommendations = append(recommendations, "Review and resolve pending error reports")
		}
		if openCircuitBreakers > 0 {
			recommendations = append(recommendations, "Check external service dependencies")
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System error handling is operating normally")
		recommendations = append(recommendations, "Continue regular monitoring")
	}

	return recommendations
}
