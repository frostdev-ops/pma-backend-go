package reports

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// reportEngine implements the ReportEngine interface
type reportEngine struct {
	db     *sql.DB
	config *analytics.AnalyticsConfig
	logger *logrus.Logger
}

// NewReportEngine creates a new report engine
func NewReportEngine(db *sql.DB, config *analytics.AnalyticsConfig, logger *logrus.Logger) (analytics.ReportEngine, error) {
	return &reportEngine{
		db:     db,
		config: config,
		logger: logger,
	}, nil
}

// CreateReport creates a report from a template and dataset
func (re *reportEngine) CreateReport(template *analytics.ReportTemplate, data *analytics.Dataset) (*analytics.Report, error) {
	if template == nil {
		return nil, fmt.Errorf("template cannot be nil")
	}
	if data == nil {
		return nil, fmt.Errorf("dataset cannot be nil")
	}

	reportID := uuid.New().String()

	report := &analytics.Report{
		ID:          reportID,
		Title:       template.Name,
		GeneratedAt: time.Now(),
		Format:      "html", // Default format
		Status:      "completed",
	}

	// Process each section
	var renderedSections []analytics.RenderedSection
	for _, section := range template.Sections {
		renderedSection, err := re.renderSection(section, data)
		if err != nil {
			re.logger.Warn("Failed to render section", map[string]interface{}{
				"section": section.ID,
				"error":   err.Error(),
			})
			continue
		}
		renderedSections = append(renderedSections, *renderedSection)
	}

	report.Sections = renderedSections

	// Generate summary
	report.Summary = re.generateSummary(data, template)

	// Calculate report size (simplified)
	reportData, _ := json.Marshal(report)
	report.Data = reportData
	report.SizeBytes = int64(len(reportData))

	return report, nil
}

// GetReportTemplates retrieves all available report templates
func (re *reportEngine) GetReportTemplates() ([]*analytics.ReportTemplate, error) {
	query := `
		SELECT id, name, description, category, type, sections, parameters, 
			   styling, created_by, created_at, updated_at, active
		FROM report_templates
		WHERE active = 1
		ORDER BY name`

	rows, err := re.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query report templates: %w", err)
	}
	defer rows.Close()

	var templates []*analytics.ReportTemplate
	for rows.Next() {
		template := &analytics.ReportTemplate{}
		var sectionsJSON, parametersJSON, stylingJSON []byte

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description,
			&template.Category, &template.Type, &sectionsJSON,
			&parametersJSON, &stylingJSON, &template.CreatedBy,
			&template.CreatedAt, &template.UpdatedAt, &template.Active,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(sectionsJSON, &template.Sections)
		json.Unmarshal(parametersJSON, &template.Parameters)
		json.Unmarshal(stylingJSON, &template.Styling)

		templates = append(templates, template)
	}

	return templates, nil
}

// GetReportTemplate retrieves a specific report template by ID
func (re *reportEngine) GetReportTemplate(templateID string) (*analytics.ReportTemplate, error) {
	query := `
		SELECT id, name, description, category, type, sections, parameters, 
			   styling, created_by, created_at, updated_at, active
		FROM report_templates
		WHERE id = ? AND active = 1`

	template := &analytics.ReportTemplate{}
	var sectionsJSON, parametersJSON, stylingJSON []byte

	err := re.db.QueryRow(query, templateID).Scan(
		&template.ID, &template.Name, &template.Description,
		&template.Category, &template.Type, &sectionsJSON,
		&parametersJSON, &stylingJSON, &template.CreatedBy,
		&template.CreatedAt, &template.UpdatedAt, &template.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get report template: %w", err)
	}

	json.Unmarshal(sectionsJSON, &template.Sections)
	json.Unmarshal(parametersJSON, &template.Parameters)
	json.Unmarshal(stylingJSON, &template.Styling)

	return template, nil
}

// CreateCustomTemplate creates a new custom report template
func (re *reportEngine) CreateCustomTemplate(template *analytics.ReportTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if template.ID == "" {
		template.ID = uuid.New().String()
	}
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}
	template.UpdatedAt = time.Now()

	sectionsJSON, _ := json.Marshal(template.Sections)
	parametersJSON, _ := json.Marshal(template.Parameters)
	stylingJSON, _ := json.Marshal(template.Styling)

	query := `
		INSERT OR REPLACE INTO report_templates (
			id, name, description, category, type, sections, parameters,
			styling, created_by, created_at, updated_at, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := re.db.Exec(query,
		template.ID, template.Name, template.Description,
		template.Category, template.Type, sectionsJSON,
		parametersJSON, stylingJSON, template.CreatedBy,
		template.CreatedAt, template.UpdatedAt, template.Active)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	re.logger.Info("Created report template", map[string]interface{}{
		"template_id": template.ID,
		"name":        template.Name,
	})

	return nil
}

// ScheduleReport schedules a report for automatic generation
func (re *reportEngine) ScheduleReport(schedule *analytics.ReportSchedule) error {
	if schedule == nil {
		return fmt.Errorf("schedule cannot be nil")
	}

	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = time.Now()
	}

	parametersJSON, _ := json.Marshal(schedule.Parameters)
	destinationsJSON, _ := json.Marshal(schedule.Destinations)

	query := `
		INSERT OR REPLACE INTO scheduled_reports (
			id, name, template_id, parameters, schedule_cron, format,
			destinations, created_by, created_at, last_run, next_run, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := re.db.Exec(query,
		schedule.ID, schedule.Name, schedule.TemplateID,
		parametersJSON, schedule.Schedule, schedule.Format,
		destinationsJSON, schedule.CreatedBy, schedule.CreatedAt,
		schedule.LastRun, schedule.NextRun, schedule.Active)

	if err != nil {
		return fmt.Errorf("failed to schedule report: %w", err)
	}

	re.logger.Info("Scheduled report", map[string]interface{}{
		"schedule_id": schedule.ID,
		"template_id": schedule.TemplateID,
		"schedule":    schedule.Schedule,
	})

	return nil
}

// GetScheduledReports retrieves all scheduled reports
func (re *reportEngine) GetScheduledReports() ([]*analytics.ScheduledReport, error) {
	query := `
		SELECT id, name, template_id, parameters, schedule_cron, format,
			   destinations, created_by, created_at, last_run, next_run, active
		FROM scheduled_reports
		WHERE active = 1
		ORDER BY created_at DESC`

	rows, err := re.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled reports: %w", err)
	}
	defer rows.Close()

	var scheduledReports []*analytics.ScheduledReport
	for rows.Next() {
		schedule := &analytics.ReportSchedule{}
		var parametersJSON, destinationsJSON []byte

		err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.TemplateID,
			&parametersJSON, &schedule.Schedule, &schedule.Format,
			&destinationsJSON, &schedule.CreatedBy, &schedule.CreatedAt,
			&schedule.LastRun, &schedule.NextRun, &schedule.Active,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(parametersJSON, &schedule.Parameters)
		json.Unmarshal(destinationsJSON, &schedule.Destinations)

		// Convert to ScheduledReport format
		scheduledReport := &analytics.ScheduledReport{
			ID:         schedule.ID,
			ScheduleID: schedule.ID,
			Status:     "scheduled",
			StartedAt:  schedule.CreatedAt,
		}
		if schedule.LastRun != nil {
			scheduledReport.CompletedAt = schedule.LastRun
		}

		scheduledReports = append(scheduledReports, scheduledReport)
	}

	return scheduledReports, nil
}

// DeleteScheduledReport deletes a scheduled report
func (re *reportEngine) DeleteScheduledReport(scheduleID string) error {
	query := `UPDATE scheduled_reports SET active = 0 WHERE id = ?`

	_, err := re.db.Exec(query, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled report: %w", err)
	}

	re.logger.Info("Deleted scheduled report", map[string]interface{}{
		"schedule_id": scheduleID,
	})

	return nil
}

// Private helper methods

// renderSection renders a report section with data
func (re *reportEngine) renderSection(section analytics.ReportSection, data *analytics.Dataset) (*analytics.RenderedSection, error) {
	rendered := &analytics.RenderedSection{
		ID:    section.ID,
		Title: section.Title,
		Type:  section.Type,
		Order: section.Order,
	}

	switch section.Type {
	case "metric":
		rendered.Content = re.renderMetricSection(section, data)
	case "chart":
		rendered.Content = re.renderChartSection(section, data)
	case "table":
		rendered.Content = re.renderTableSection(section, data)
	case "text":
		rendered.Content = re.renderTextSection(section, data)
	default:
		rendered.Content = fmt.Sprintf("Unknown section type: %s", section.Type)
	}

	return rendered, nil
}

// renderMetricSection renders a metric section
func (re *reportEngine) renderMetricSection(section analytics.ReportSection, data *analytics.Dataset) interface{} {
	if len(data.Data) == 0 {
		return map[string]interface{}{
			"value": 0,
			"label": "No Data",
		}
	}

	// Calculate metric based on aggregation
	var value float64
	switch section.Query.Aggregation {
	case analytics.AggregationSum:
		for _, point := range data.Data {
			value += point.Value
		}
	case analytics.AggregationAvg:
		sum := 0.0
		for _, point := range data.Data {
			sum += point.Value
		}
		value = sum / float64(len(data.Data))
	case analytics.AggregationCount:
		value = float64(len(data.Data))
	case analytics.AggregationMax:
		value = data.Data[0].Value
		for _, point := range data.Data {
			if point.Value > value {
				value = point.Value
			}
		}
	case analytics.AggregationMin:
		value = data.Data[0].Value
		for _, point := range data.Data {
			if point.Value < value {
				value = point.Value
			}
		}
	default:
		value = float64(len(data.Data))
	}

	return map[string]interface{}{
		"value":       value,
		"label":       section.Title,
		"aggregation": string(section.Query.Aggregation),
		"dataPoints":  len(data.Data),
	}
}

// renderChartSection renders a chart section
func (re *reportEngine) renderChartSection(section analytics.ReportSection, data *analytics.Dataset) interface{} {
	chartData := make([]map[string]interface{}, 0, len(data.Data))

	for _, point := range data.Data {
		chartData = append(chartData, map[string]interface{}{
			"timestamp": point.Timestamp,
			"value":     point.Value,
			"tags":      point.Tags,
		})
	}

	return map[string]interface{}{
		"type":   "line", // Default chart type
		"data":   chartData,
		"title":  section.Title,
		"xLabel": "Time",
		"yLabel": "Value",
	}
}

// renderTableSection renders a table section
func (re *reportEngine) renderTableSection(section analytics.ReportSection, data *analytics.Dataset) interface{} {
	headers := []string{"Timestamp", "Value"}
	rows := make([][]interface{}, 0, len(data.Data))

	for _, point := range data.Data {
		row := []interface{}{
			point.Timestamp.Format("2006-01-02 15:04:05"),
			point.Value,
		}
		rows = append(rows, row)
	}

	return map[string]interface{}{
		"headers": headers,
		"rows":    rows,
		"total":   len(rows),
	}
}

// renderTextSection renders a text section
func (re *reportEngine) renderTextSection(section analytics.ReportSection, data *analytics.Dataset) interface{} {
	// Simple text template processing
	text := section.Template
	if text == "" {
		text = fmt.Sprintf("Data summary for %s: %d data points collected.",
			data.Name, len(data.Data))
	}

	return map[string]interface{}{
		"text": text,
		"type": "paragraph",
	}
}

// generateSummary generates a summary for the report
func (re *reportEngine) generateSummary(data *analytics.Dataset, template *analytics.ReportTemplate) analytics.ReportSummary {
	summary := analytics.ReportSummary{
		TotalSections:   len(template.Sections),
		DataPoints:      len(data.Data),
		KeyMetrics:      make(map[string]float64),
		Insights:        []string{},
		Recommendations: []string{},
	}

	if len(data.Data) > 0 {
		summary.TimeRange = analytics.TimeRange{
			Start: data.Data[0].Timestamp,
			End:   data.Data[len(data.Data)-1].Timestamp,
		}

		// Calculate key metrics
		if data.Statistics != nil {
			summary.KeyMetrics["average"] = data.Statistics.Mean
			summary.KeyMetrics["minimum"] = data.Statistics.Min
			summary.KeyMetrics["maximum"] = data.Statistics.Max
			summary.KeyMetrics["total"] = data.Statistics.Sum
		}

		// Generate basic insights
		if data.Statistics != nil && data.Statistics.StdDev > 0 {
			summary.Insights = append(summary.Insights,
				fmt.Sprintf("Data shows variability with standard deviation of %.2f", data.Statistics.StdDev))
		}

		// Generate recommendations
		if len(data.Data) < 10 {
			summary.Recommendations = append(summary.Recommendations,
				"Consider collecting more data points for better analysis")
		}
	}

	return summary
}
