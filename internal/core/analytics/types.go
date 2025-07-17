package analytics

import (
	"io"
	"time"
)

// Core Analytics Interfaces

// AnalyticsManager is the main interface for the analytics system
type AnalyticsManager interface {
	ProcessEvent(event *AnalyticsEvent) error
	GenerateReport(request *ReportRequest) (*Report, error)
	GetHistoricalData(query *HistoricalQuery) (*Dataset, error)
	CreateCustomMetric(metric *CustomMetric) error
	GetInsights(entityType string, timeRange TimeRange) ([]*Insight, error)
	ExportData(request *ExportRequest) (io.Reader, error)
	GetDashboard(dashboardID string) (*Dashboard, error)
	CreateDashboard(config *DashboardConfig) (*Dashboard, error)
	GetVisualizationData(vizID string, params map[string]interface{}) (interface{}, error)
}

// DataAggregator handles data aggregation and statistics
type DataAggregator interface {
	AggregateByTime(data []DataPoint, interval time.Duration) []AggregatedPoint
	AggregateByEntity(data []DataPoint, groupBy string) map[string][]DataPoint
	CalculateStatistics(data []DataPoint) *Statistics
	DetectTrends(data []DataPoint) ([]*Trend, error)
	IdentifyAnomalies(data []DataPoint) ([]*Anomaly, error)
	ComputeCorrelations(datasets map[string][]DataPoint) map[string]float64
}

// TimeSeriesManager handles time series data operations
type TimeSeriesManager interface {
	StoreDataPoint(series string, point DataPoint) error
	QueryTimeSeries(query *TimeSeriesQuery) (*TimeSeriesResult, error)
	CreateRetentionPolicy(policy *RetentionPolicy) error
	DownsampleData(series string, resolution time.Duration) error
	GetSeriesMetadata(series string) (*SeriesMetadata, error)
	CreateSeries(name string, metadata SeriesMetadata) error
	DeleteSeries(name string) error
}

// ReportEngine handles report generation
type ReportEngine interface {
	CreateReport(template *ReportTemplate, data *Dataset) (*Report, error)
	GetReportTemplates() ([]*ReportTemplate, error)
	CreateCustomTemplate(template *ReportTemplate) error
	ScheduleReport(schedule *ReportSchedule) error
	GetScheduledReports() ([]*ScheduledReport, error)
	DeleteScheduledReport(scheduleID string) error
}

// VisualizationEngine handles data visualization
type VisualizationEngine interface {
	CreateDashboard(config *DashboardConfig) (*Dashboard, error)
	AddVisualization(dashboardID string, viz *Visualization) error
	GetVisualizationData(vizID string, params map[string]interface{}) (interface{}, error)
	ExportVisualization(vizID string, format string) ([]byte, error)
	GetAvailableChartTypes() []ChartType
	UpdateVisualization(vizID string, config VisualizationConfig) error
}

// PredictionEngine handles predictive analytics
type PredictionEngine interface {
	TrainModel(modelType string, data []DataPoint) (*Model, error)
	PredictValue(modelID string, input map[string]interface{}) (*Prediction, error)
	GetModelAccuracy(modelID string) (float64, error)
	UpdateModel(modelID string, newData []DataPoint) error
	GetAvailableModels() ([]*ModelInfo, error)
	DeleteModel(modelID string) error
}

// MetricsBuilder handles custom metrics
type MetricsBuilder interface {
	CreateMetric(definition *MetricDefinition) (*CustomMetric, error)
	UpdateMetric(metricID string, value float64, tags map[string]string) error
	GetMetricHistory(metricID string, timeRange TimeRange) ([]DataPoint, error)
	CreateComputedMetric(formula string, dependencies []string) (*ComputedMetric, error)
	GetAvailableMetrics() ([]*MetricInfo, error)
	DeleteMetric(metricID string) error
}

// ExportManager handles data export
type ExportManager interface {
	ExportToCSV(data *Dataset) ([]byte, error)
	ExportToJSON(data *Dataset) ([]byte, error)
	ExportToExcel(data *Dataset) ([]byte, error)
	ExportToPDF(report *Report) ([]byte, error)
	ScheduleExport(schedule *ExportSchedule) error
	SendToWebhook(data interface{}, webhookURL string) error
	GetExportHistory() ([]*ExportJob, error)
}

// Core Data Types

// AnalyticsEvent represents a system event to be analyzed
type AnalyticsEvent struct {
	ID         string                 `json:"id" db:"id"`
	Type       string                 `json:"type" db:"type"`
	EntityID   string                 `json:"entity_id" db:"entity_id"`
	EntityType string                 `json:"entity_type" db:"entity_type"`
	UserID     string                 `json:"user_id" db:"user_id"`
	Data       map[string]interface{} `json:"data" db:"data"`
	Context    EventContext           `json:"context" db:"context"`
	Source     string                 `json:"source" db:"source"`
	Tags       map[string]string      `json:"tags" db:"tags"`
	Timestamp  time.Time              `json:"timestamp" db:"timestamp"`
}

// EventContext provides additional context for events
type EventContext struct {
	RequestID   string            `json:"request_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	IPAddress   string            `json:"ip_address,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Version     string            `json:"version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AggregatedPoint represents an aggregated data point
type AggregatedPoint struct {
	Timestamp   time.Time       `json:"timestamp"`
	Count       int64           `json:"count"`
	Sum         float64         `json:"sum"`
	Average     float64         `json:"average"`
	Min         float64         `json:"min"`
	Max         float64         `json:"max"`
	Percentiles map[int]float64 `json:"percentiles,omitempty"`
}

// Statistics contains statistical analysis results
type Statistics struct {
	Count       int64              `json:"count"`
	Sum         float64            `json:"sum"`
	Mean        float64            `json:"mean"`
	Median      float64            `json:"median"`
	Mode        float64            `json:"mode"`
	StdDev      float64            `json:"std_dev"`
	Variance    float64            `json:"variance"`
	Min         float64            `json:"min"`
	Max         float64            `json:"max"`
	Range       float64            `json:"range"`
	Percentiles map[int]float64    `json:"percentiles"`
	Quartiles   map[string]float64 `json:"quartiles"` // Q1, Q2, Q3
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Dataset represents a collection of data points
type Dataset struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Data        []DataPoint            `json:"data"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Statistics  *Statistics            `json:"statistics,omitempty"`
	Generated   time.Time              `json:"generated"`
}

// Custom Metrics Types

// CustomMetric represents a user-defined metric
type CustomMetric struct {
	ID              string           `json:"id" db:"id"`
	Name            string           `json:"name" db:"name"`
	Description     string           `json:"description" db:"description"`
	Type            MetricType       `json:"type" db:"type"`
	Unit            string           `json:"unit" db:"unit"`
	Definition      MetricDefinition `json:"definition" db:"definition"`
	RetentionPeriod int              `json:"retention_period" db:"retention_period"` // days
	CreatedBy       string           `json:"created_by" db:"created_by"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
	Active          bool             `json:"active" db:"active"`
}

// MetricDefinition defines how a metric is calculated
type MetricDefinition struct {
	Formula      string            `json:"formula,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Aggregation  AggregationType   `json:"aggregation"`
	Filters      map[string]string `json:"filters,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
}

// ComputedMetric represents a metric calculated from other metrics
type ComputedMetric struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Formula      string    `json:"formula"`
	Dependencies []string  `json:"dependencies"`
	LastComputed time.Time `json:"last_computed"`
	Result       float64   `json:"result"`
}

// MetricInfo provides basic information about a metric
type MetricInfo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        MetricType `json:"type"`
	Unit        string     `json:"unit"`
	Active      bool       `json:"active"`
}

// Reporting Types

// ReportRequest represents a request to generate a report
type ReportRequest struct {
	Type        string                 `json:"type"`
	Template    string                 `json:"template"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	TimeRange   TimeRange              `json:"time_range"`
	Aggregation AggregationType        `json:"aggregation"`
	Format      string                 `json:"format"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Report represents a generated report
type Report struct {
	ID          string            `json:"id" db:"id"`
	Title       string            `json:"title"`
	GeneratedAt time.Time         `json:"generated_at" db:"generated_at"`
	GeneratedBy string            `json:"generated_by" db:"generated_by"`
	Sections    []RenderedSection `json:"sections"`
	Summary     ReportSummary     `json:"summary" db:"summary"`
	Format      string            `json:"format" db:"format"`
	Data        []byte            `json:"data,omitempty" db:"data"`
	FilePath    string            `json:"file_path" db:"file_path"`
	SizeBytes   int64             `json:"size_bytes" db:"size_bytes"`
	Status      string            `json:"status" db:"status"`
}

// ReportTemplate defines the structure of a report
type ReportTemplate struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Category    string            `json:"category" db:"category"`
	Type        string            `json:"type" db:"type"`
	Sections    []ReportSection   `json:"sections" db:"sections"`
	Parameters  []ReportParameter `json:"parameters" db:"parameters"`
	Styling     ReportStyling     `json:"styling" db:"styling"`
	CreatedBy   string            `json:"created_by" db:"created_by"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	Active      bool              `json:"active" db:"active"`
}

// ReportSection defines a section within a report template
type ReportSection struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Type     string                 `json:"type"` // text, chart, table, metric
	Query    DataQuery              `json:"query,omitempty"`
	Template string                 `json:"template,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Order    int                    `json:"order"`
}

// RenderedSection represents a rendered report section
type RenderedSection struct {
	ID      string      `json:"id"`
	Title   string      `json:"title"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
	Order   int         `json:"order"`
}

// ReportParameter defines a configurable parameter for reports
type ReportParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Required     bool        `json:"required"`
	Options      []string    `json:"options,omitempty"`
}

// ReportStyling defines the visual styling for reports
type ReportStyling struct {
	Theme    string            `json:"theme"`
	Colors   []string          `json:"colors"`
	Fonts    map[string]string `json:"fonts"`
	Layout   string            `json:"layout"`
	PageSize string            `json:"page_size"`
	Margins  map[string]int    `json:"margins"`
	Header   string            `json:"header"`
	Footer   string            `json:"footer"`
}

// ReportSummary provides a summary of report contents
type ReportSummary struct {
	TotalSections   int                `json:"total_sections"`
	DataPoints      int                `json:"data_points"`
	TimeRange       TimeRange          `json:"time_range"`
	KeyMetrics      map[string]float64 `json:"key_metrics"`
	Insights        []string           `json:"insights"`
	Recommendations []string           `json:"recommendations"`
}

// ReportSchedule defines a scheduled report
type ReportSchedule struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	TemplateID   string                 `json:"template_id" db:"template_id"`
	Parameters   map[string]interface{} `json:"parameters" db:"parameters"`
	Schedule     string                 `json:"schedule" db:"schedule_cron"` // cron expression
	Format       string                 `json:"format" db:"format"`
	Destinations []ExportDestination    `json:"destinations" db:"destinations"`
	CreatedBy    string                 `json:"created_by" db:"created_by"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	LastRun      *time.Time             `json:"last_run" db:"last_run"`
	NextRun      *time.Time             `json:"next_run" db:"next_run"`
	Active       bool                   `json:"active" db:"active"`
}

// ScheduledReport represents an instance of a scheduled report
type ScheduledReport struct {
	ID          string     `json:"id"`
	ScheduleID  string     `json:"schedule_id"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	FilePath    string     `json:"file_path,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// Visualization Types

// Dashboard represents a collection of visualizations
type Dashboard struct {
	ID             string          `json:"id" db:"id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	Layout         string          `json:"layout" db:"layout"`
	Config         DashboardConfig `json:"config" db:"config"`
	Visualizations []Visualization `json:"visualizations"`
	CreatedBy      string          `json:"created_by" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	Shared         bool            `json:"shared" db:"shared"`
	Active         bool            `json:"active" db:"active"`
}

// DashboardConfig defines dashboard configuration
type DashboardConfig struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Layout         string            `json:"layout"` // grid, masonry, flow
	Visualizations []Visualization   `json:"visualizations"`
	Filters        []DashboardFilter `json:"filters"`
	RefreshRate    time.Duration     `json:"refresh_rate"`
	Theme          string            `json:"theme"`
	AutoRefresh    bool              `json:"auto_refresh"`
}

// Visualization represents a single chart or visualization
type Visualization struct {
	ID                  string               `json:"id" db:"id"`
	DashboardID         string               `json:"dashboard_id" db:"dashboard_id"`
	Name                string               `json:"name" db:"name"`
	Type                string               `json:"type" db:"type"` // line, bar, pie, gauge, table, heatmap
	Query               DataQuery            `json:"query" db:"query_config"`
	Options             VisualizationOptions `json:"options" db:"visualization_config"`
	Position            Position             `json:"position" db:"position"`
	Size                Size                 `json:"size"`
	DataRefreshInterval int                  `json:"data_refresh_interval" db:"data_refresh_interval"`
	CreatedAt           time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at" db:"updated_at"`
	Active              bool                 `json:"active" db:"active"`
}

// VisualizationOptions defines visualization configuration
type VisualizationOptions struct {
	Title      string                 `json:"title"`
	Width      int                    `json:"width"`
	Height     int                    `json:"height"`
	XLabel     string                 `json:"x_label"`
	YLabel     string                 `json:"y_label"`
	Colors     []string               `json:"colors"`
	Theme      string                 `json:"theme"`
	ShowLegend bool                   `json:"show_legend"`
	ShowGrid   bool                   `json:"show_grid"`
	Animation  bool                   `json:"animation"`
	Tooltip    bool                   `json:"tooltip"`
	Zoom       bool                   `json:"zoom"`
	Export     bool                   `json:"export"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

// VisualizationConfig is an alias for VisualizationOptions
type VisualizationConfig = VisualizationOptions

// Position defines the position of a visualization
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size defines the size of a visualization
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardFilter defines filters that can be applied to dashboards
type DashboardFilter struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Options []string    `json:"options,omitempty"`
}

// ChartType defines available chart types
type ChartType struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	DataTypes   []string `json:"data_types"`
	Icon        string   `json:"icon"`
}

// Prediction Types

// Model represents a machine learning model
type Model struct {
	ID               string                 `json:"id" db:"id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	Type             string                 `json:"type" db:"type"` // linear_regression, time_series, classification
	Algorithm        string                 `json:"algorithm" db:"algorithm"`
	Accuracy         float64                `json:"accuracy" db:"accuracy"`
	Parameters       map[string]interface{} `json:"parameters" db:"parameters"`
	Features         []string               `json:"features" db:"features"`
	TrainingDataSize int                    `json:"training_data_size" db:"training_data_size"`
	TrainedAt        time.Time              `json:"trained_at" db:"trained_at"`
	TrainedBy        string                 `json:"trained_by" db:"trained_by"`
	ModelData        []byte                 `json:"model_data,omitempty" db:"model_data"`
	Active           bool                   `json:"active" db:"active"`
}

// Prediction represents a model prediction
type Prediction struct {
	ID                string                 `json:"id" db:"id"`
	ModelID           string                 `json:"model_id" db:"model_id"`
	InputData         map[string]interface{} `json:"input_data" db:"input_data"`
	PredictedValue    float64                `json:"predicted_value" db:"predicted_value"`
	Confidence        float64                `json:"confidence" db:"confidence"`
	PredictionHorizon int                    `json:"prediction_horizon" db:"prediction_horizon"` // hours
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	ActualValue       *float64               `json:"actual_value" db:"actual_value"`
	Factors           map[string]float64     `json:"factors,omitempty"`
}

// ModelInfo provides basic information about a model
type ModelInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Accuracy   float64   `json:"accuracy"`
	TrainedAt  time.Time `json:"trained_at"`
	Active     bool      `json:"active"`
	DataPoints int       `json:"data_points"`
}

// Historical Data Types

// TimeSeriesQuery represents a query for time series data
type TimeSeriesQuery struct {
	Series      string                 `json:"series"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Resolution  time.Duration          `json:"resolution"`
	Aggregation string                 `json:"aggregation"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

// TimeSeriesResult contains the result of a time series query
type TimeSeriesResult struct {
	Series     string          `json:"series"`
	Data       []DataPoint     `json:"data"`
	Metadata   SeriesMetadata  `json:"metadata"`
	Query      TimeSeriesQuery `json:"query"`
	Total      int             `json:"total"`
	Statistics *Statistics     `json:"statistics,omitempty"`
}

// SeriesMetadata contains metadata about a time series
type SeriesMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Unit        string            `json:"unit"`
	DataType    string            `json:"data_type"`
	Tags        map[string]string `json:"tags"`
	Created     time.Time         `json:"created"`
	LastUpdate  time.Time         `json:"last_update"`
	DataPoints  int64             `json:"data_points"`
	Resolution  time.Duration     `json:"resolution"`
	Retention   time.Duration     `json:"retention"`
}

// RetentionPolicy defines data retention rules
type RetentionPolicy struct {
	SeriesPattern string        `json:"series_pattern"`
	Duration      time.Duration `json:"duration"`
	Aggregation   string        `json:"aggregation"`
	Downsample    bool          `json:"downsample"`
	Archive       bool          `json:"archive"`
}

// Trend represents a detected trend in data
type Trend struct {
	Direction    string    `json:"direction"` // up, down, stable
	Slope        float64   `json:"slope"`
	Confidence   float64   `json:"confidence"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Magnitude    float64   `json:"magnitude"`
	Significance string    `json:"significance"` // low, medium, high
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID             string                 `json:"id" db:"id"`
	SeriesName     string                 `json:"series_name" db:"series_name"`
	Type           string                 `json:"type" db:"anomaly_type"` // spike, drop, trend_change, outlier
	Severity       string                 `json:"severity" db:"severity"` // low, medium, high, critical
	DetectedAt     time.Time              `json:"detected_at" db:"detected_at"`
	Value          float64                `json:"value" db:"value"`
	ExpectedValue  float64                `json:"expected_value" db:"expected_value"`
	Deviation      float64                `json:"deviation" db:"deviation"`
	Confidence     float64                `json:"confidence" db:"confidence"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Acknowledged   bool                   `json:"acknowledged" db:"acknowledged"`
	AcknowledgedBy string                 `json:"acknowledged_by" db:"acknowledged_by"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at" db:"acknowledged_at"`
}

// Insight represents an automatically generated insight
type Insight struct {
	ID              string      `json:"id" db:"id"`
	EntityType      string      `json:"entity_type" db:"entity_type"`
	EntityID        string      `json:"entity_id" db:"entity_id"`
	Type            string      `json:"type" db:"insight_type"` // trend, pattern, anomaly, recommendation
	Title           string      `json:"title" db:"title"`
	Description     string      `json:"description" db:"description"`
	ImpactLevel     string      `json:"impact_level" db:"impact_level"` // low, medium, high
	Confidence      float64     `json:"confidence" db:"confidence"`
	DataPoints      []DataPoint `json:"data_points" db:"data_points"`
	Recommendations []string    `json:"recommendations" db:"recommendations"`
	GeneratedAt     time.Time   `json:"generated_at" db:"generated_at"`
	ExpiresAt       *time.Time  `json:"expires_at" db:"expires_at"`
	Viewed          bool        `json:"viewed" db:"viewed"`
	ActedUpon       bool        `json:"acted_upon" db:"acted_upon"`
}

// Export Types

// ExportRequest represents a request to export data
type ExportRequest struct {
	Query       DataQuery              `json:"query"`
	Format      string                 `json:"format"` // csv, json, excel, pdf
	Compression bool                   `json:"compression"`
	Destination ExportDestination      `json:"destination"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// ExportDestination defines where exported data should be sent
type ExportDestination struct {
	Type    string                 `json:"type"` // file, webhook, email
	Target  string                 `json:"target"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// ExportSchedule defines a scheduled export
type ExportSchedule struct {
	ID           string              `json:"id" db:"id"`
	Name         string              `json:"name" db:"name"`
	Description  string              `json:"description" db:"description"`
	QueryConfig  DataQuery           `json:"query_config" db:"query_config"`
	Format       string              `json:"format" db:"format"`
	Schedule     string              `json:"schedule" db:"schedule_cron"`
	Destinations []ExportDestination `json:"destinations" db:"destinations"`
	Compression  bool                `json:"compression" db:"compression"`
	CreatedBy    string              `json:"created_by" db:"created_by"`
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
	LastRun      *time.Time          `json:"last_run" db:"last_run"`
	NextRun      *time.Time          `json:"next_run" db:"next_run"`
	Active       bool                `json:"active" db:"active"`
}

// ExportJob represents an export job execution
type ExportJob struct {
	ID           string     `json:"id" db:"id"`
	ScheduleID   string     `json:"schedule_id" db:"schedule_id"`
	Name         string     `json:"name" db:"name"`
	Format       string     `json:"format" db:"format"`
	Status       string     `json:"status" db:"status"` // pending, running, completed, failed
	StartedAt    time.Time  `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at" db:"completed_at"`
	FilePath     string     `json:"file_path" db:"file_path"`
	FileSize     int64      `json:"file_size" db:"file_size"`
	RecordsCount int        `json:"records_count" db:"records_count"`
	Error        string     `json:"error" db:"error_message"`
}

// Query Types

// DataQuery represents a query for data
type DataQuery struct {
	Series      string                 `json:"series"`
	TimeRange   TimeRange              `json:"time_range"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Aggregation AggregationType        `json:"aggregation"`
	GroupBy     []string               `json:"group_by,omitempty"`
	OrderBy     string                 `json:"order_by,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

// HistoricalQuery represents a query for historical data
type HistoricalQuery struct {
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id,omitempty"`
	TimeRange   TimeRange              `json:"time_range"`
	Metrics     []string               `json:"metrics,omitempty"`
	Aggregation AggregationType        `json:"aggregation"`
	Resolution  time.Duration          `json:"resolution,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

// Enums and Constants

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// AggregationType defines how data should be aggregated
type AggregationType string

const (
	AggregationSum    AggregationType = "sum"
	AggregationAvg    AggregationType = "avg"
	AggregationMin    AggregationType = "min"
	AggregationMax    AggregationType = "max"
	AggregationCount  AggregationType = "count"
	AggregationMedian AggregationType = "median"
	AggregationP95    AggregationType = "p95"
	AggregationP99    AggregationType = "p99"
	AggregationStdDev AggregationType = "stddev"
	AggregationFirst  AggregationType = "first"
	AggregationLast   AggregationType = "last"
)

// Chart format constants
const (
	FormatPNG  = "png"
	FormatJPEG = "jpeg"
	FormatSVG  = "svg"
	FormatPDF  = "pdf"
	FormatHTML = "html"
)

// Export format constants
const (
	ExportFormatCSV   = "csv"
	ExportFormatJSON  = "json"
	ExportFormatExcel = "excel"
	ExportFormatPDF   = "pdf"
)

// Report format constants
const (
	ReportFormatPDF  = "pdf"
	ReportFormatHTML = "html"
	ReportFormatJSON = "json"
)

// Chart type constants
const (
	ChartTypeLine    = "line"
	ChartTypeBar     = "bar"
	ChartTypePie     = "pie"
	ChartTypeGauge   = "gauge"
	ChartTypeTable   = "table"
	ChartTypeHeatmap = "heatmap"
	ChartTypeArea    = "area"
	ChartTypeScatter = "scatter"
)

// Analytics event types
const (
	EventTypeUserAction   = "user_action"
	EventTypeSystemEvent  = "system_event"
	EventTypeDeviceUpdate = "device_update"
	EventTypeError        = "error"
	EventTypePerformance  = "performance"
	EventTypeAutomation   = "automation"
	EventTypeSecurity     = "security"
)

// Anomaly types
const (
	AnomalyTypeSpike       = "spike"
	AnomalyTypeDrop        = "drop"
	AnomalyTypeTrendChange = "trend_change"
	AnomalyTypeOutlier     = "outlier"
)

// Severity levels
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Insight types
const (
	InsightTypeTrend          = "trend"
	InsightTypePattern        = "pattern"
	InsightTypeAnomaly        = "anomaly"
	InsightTypeRecommendation = "recommendation"
)

// Helper functions

// NewTimeRange creates a new TimeRange
func NewTimeRange(start, end time.Time) TimeRange {
	return TimeRange{Start: start, End: end}
}

// Duration returns the duration of the time range
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Contains checks if a time is within the range
func (tr TimeRange) Contains(t time.Time) bool {
	return t.After(tr.Start) && t.Before(tr.End)
}

// LastHour returns a TimeRange for the last hour
func LastHour() TimeRange {
	end := time.Now()
	start := end.Add(-time.Hour)
	return NewTimeRange(start, end)
}

// LastDay returns a TimeRange for the last 24 hours
func LastDay() TimeRange {
	end := time.Now()
	start := end.Add(-24 * time.Hour)
	return NewTimeRange(start, end)
}

// LastWeek returns a TimeRange for the last week
func LastWeek() TimeRange {
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)
	return NewTimeRange(start, end)
}

// LastMonth returns a TimeRange for the last 30 days
func LastMonth() TimeRange {
	end := time.Now()
	start := end.Add(-30 * 24 * time.Hour)
	return NewTimeRange(start, end)
}

// AnalyticsConfig contains configuration for the analytics system
type AnalyticsConfig struct {
	DataRetention struct {
		RawData    time.Duration `yaml:"raw_data"`
		Aggregated time.Duration `yaml:"aggregated"`
		Reports    time.Duration `yaml:"reports"`
	} `yaml:"data_retention"`

	Processing struct {
		BatchSize       int           `yaml:"batch_size"`
		ProcessInterval time.Duration `yaml:"process_interval"`
		MaxWorkers      int           `yaml:"max_workers"`
		QueueSize       int           `yaml:"queue_size"`
	} `yaml:"processing"`

	Prediction struct {
		Enabled          bool          `yaml:"enabled"`
		ModelRetention   time.Duration `yaml:"model_retention"`
		TrainingInterval time.Duration `yaml:"training_interval"`
	} `yaml:"prediction"`

	Export struct {
		MaxFileSize    int64         `yaml:"max_file_size"`
		TempDirectory  string        `yaml:"temp_directory"`
		WebhookTimeout time.Duration `yaml:"webhook_timeout"`
	} `yaml:"export"`

	Visualization struct {
		CacheTimeout  time.Duration `yaml:"cache_timeout"`
		MaxDataPoints int           `yaml:"max_data_points"`
		DefaultTheme  string        `yaml:"default_theme"`
	} `yaml:"visualization"`
}
