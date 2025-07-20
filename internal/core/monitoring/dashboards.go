package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DashboardEngine manages performance dashboards and real-time visualizations
type DashboardEngine struct {
	config           *DashboardConfig
	logger           *logrus.Logger
	dashboards       map[string]*Dashboard
	widgets          map[string]*Widget
	dataSources      map[string]DataSource
	liveStreams      map[string]*LiveStream
	metricCache      *MetricCache
	visualizationMgr *VisualizationManager
	alertingEngine   *AlertingEngine
	mu               sync.RWMutex
	stopChan         chan bool
}

// DashboardConfig contains dashboard engine configuration
type DashboardConfig struct {
	Enabled              bool              `json:"enabled"`
	RefreshInterval      time.Duration     `json:"refresh_interval"`
	MaxDataPoints        int               `json:"max_data_points"`
	CacheSize            int               `json:"cache_size"`
	CacheTTL             time.Duration     `json:"cache_ttl"`
	DefaultTimeRange     time.Duration     `json:"default_time_range"`
	LiveStreamBufferSize int               `json:"live_stream_buffer_size"`
	MaxConcurrentQueries int               `json:"max_concurrent_queries"`
	CustomThemes         map[string]*Theme `json:"custom_themes"`
	DefaultTheme         string            `json:"default_theme"`
	ExportFormats        []string          `json:"export_formats"`
	WebSocketEndpoint    string            `json:"websocket_endpoint"`
	EnableRealTime       bool              `json:"enable_real_time"`
	EnableExports        bool              `json:"enable_exports"`
}

// Dashboard represents a monitoring dashboard
type Dashboard struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Tags            []string               `json:"tags"`
	Widgets         []*Widget              `json:"widgets"`
	Layout          *DashboardLayout       `json:"layout"`
	TimeRange       *TimeRange             `json:"time_range"`
	RefreshInterval time.Duration          `json:"refresh_interval"`
	Variables       map[string]*Variable   `json:"variables"`
	Annotations     []*Annotation          `json:"annotations"`
	Permissions     *DashboardPermissions  `json:"permissions"`
	Theme           string                 `json:"theme"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by"`
	Shared          bool                   `json:"shared"`
	Starred         bool                   `json:"starred"`
	Version         int                    `json:"version"`
	Settings        map[string]interface{} `json:"settings"`
	LiveUpdates     bool                   `json:"live_updates"`
}

// Widget represents a dashboard widget
type Widget struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Type            WidgetType             `json:"type"`
	Position        *WidgetPosition        `json:"position"`
	Size            *WidgetSize            `json:"size"`
	DataSource      string                 `json:"data_source"`
	Query           *MetricQuery           `json:"query"`
	Visualization   *VisualizationConfig   `json:"visualization"`
	Thresholds      []*Threshold           `json:"thresholds"`
	Options         map[string]interface{} `json:"options"`
	Targets         []*QueryTarget         `json:"targets"`
	Transformations []*DataTransformation  `json:"transformations"`
	FieldConfig     *FieldConfig           `json:"field_config"`
	AlertRule       *WidgetAlertRule       `json:"alert_rule"`
	RefreshRate     time.Duration          `json:"refresh_rate"`
	LastUpdated     time.Time              `json:"last_updated"`
	CacheKey        string                 `json:"cache_key"`
	Conditional     *ConditionalDisplay    `json:"conditional"`
}

// DashboardLayout defines the layout structure
type DashboardLayout struct {
	Type        LayoutType             `json:"type"`
	Columns     int                    `json:"columns"`
	RowHeight   int                    `json:"row_height"`
	Responsive  bool                   `json:"responsive"`
	GridSize    *GridSize              `json:"grid_size"`
	Breakpoints map[string]*Breakpoint `json:"breakpoints"`
}

// WidgetPosition defines widget position
type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WidgetSize defines widget dimensions
type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GridSize defines grid dimensions
type GridSize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// Breakpoint defines responsive breakpoints
type Breakpoint struct {
	Width   int `json:"width"`
	Columns int `json:"columns"`
}

// Variable represents a dashboard variable
type Variable struct {
	Name         string          `json:"name"`
	Type         VariableType    `json:"type"`
	Label        string          `json:"label"`
	Description  string          `json:"description"`
	Query        string          `json:"query"`
	Options      []string        `json:"options"`
	DefaultValue interface{}     `json:"default_value"`
	MultiSelect  bool            `json:"multi_select"`
	IncludeAll   bool            `json:"include_all"`
	Refresh      VariableRefresh `json:"refresh"`
}

// Annotation represents a dashboard annotation
type Annotation struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Query     string                 `json:"query"`
	Enabled   bool                   `json:"enabled"`
	IconColor string                 `json:"icon_color"`
	Tags      []string               `json:"tags"`
	Type      AnnotationType         `json:"type"`
	TimeField string                 `json:"time_field"`
	TextField string                 `json:"text_field"`
	TagsField string                 `json:"tags_field"`
	Options   map[string]interface{} `json:"options"`
}

// DashboardPermissions defines access permissions
type DashboardPermissions struct {
	Viewers []string       `json:"viewers"`
	Editors []string       `json:"editors"`
	Admins  []string       `json:"admins"`
	Public  bool           `json:"public"`
	Role    PermissionRole `json:"role"`
}

// VisualizationConfig defines visualization settings
type VisualizationConfig struct {
	Type       ChartType              `json:"type"`
	Options    map[string]interface{} `json:"options"`
	Series     []*SeriesConfig        `json:"series"`
	Axes       *AxesConfig            `json:"axes"`
	Legend     *LegendConfig          `json:"legend"`
	Colors     []string               `json:"colors"`
	Tooltip    *TooltipConfig         `json:"tooltip"`
	Animation  *AnimationConfig       `json:"animation"`
	Responsive bool                   `json:"responsive"`
}

// SeriesConfig defines series configuration
type SeriesConfig struct {
	Name        string                 `json:"name"`
	Type        SeriesType             `json:"type"`
	Color       string                 `json:"color"`
	YAxis       int                    `json:"y_axis"`
	Smooth      bool                   `json:"smooth"`
	Stack       string                 `json:"stack"`
	Area        bool                   `json:"area"`
	LineWidth   int                    `json:"line_width"`
	PointSize   int                    `json:"point_size"`
	ShowPoints  bool                   `json:"show_points"`
	Aggregation AggregationType        `json:"aggregation"`
	Transform   *SeriesTransform       `json:"transform"`
	Options     map[string]interface{} `json:"options"`
}

// AxesConfig defines axes configuration
type AxesConfig struct {
	XAxis *AxisConfig `json:"x_axis"`
	YAxis *AxisConfig `json:"y_axis"`
}

// AxisConfig defines individual axis configuration
type AxisConfig struct {
	Show         bool         `json:"show"`
	Label        string       `json:"label"`
	Min          *float64     `json:"min"`
	Max          *float64     `json:"max"`
	Unit         string       `json:"unit"`
	Scale        ScaleType    `json:"scale"`
	Position     AxisPosition `json:"position"`
	GridLines    bool         `json:"grid_lines"`
	TickInterval *float64     `json:"tick_interval"`
	Format       string       `json:"format"`
}

// LegendConfig defines legend configuration
type LegendConfig struct {
	Show      bool            `json:"show"`
	Position  LegendPosition  `json:"position"`
	Alignment LegendAlignment `json:"alignment"`
	MaxWidth  int             `json:"max_width"`
	Values    []string        `json:"values"`
}

// TooltipConfig defines tooltip configuration
type TooltipConfig struct {
	Show      bool        `json:"show"`
	Shared    bool        `json:"shared"`
	Sort      TooltipSort `json:"sort"`
	ValueType ValueType   `json:"value_type"`
	Decimals  int         `json:"decimals"`
}

// AnimationConfig defines animation settings
type AnimationConfig struct {
	Enabled  bool          `json:"enabled"`
	Duration time.Duration `json:"duration"`
	Easing   EasingType    `json:"easing"`
}

// Threshold defines alert thresholds for widgets
type Threshold struct {
	Value     float64            `json:"value"`
	Color     string             `json:"color"`
	Condition ThresholdCondition `json:"condition"`
	Label     string             `json:"label"`
}

// QueryTarget defines a query target
type QueryTarget struct {
	RefID      string                 `json:"ref_id"`
	Query      string                 `json:"query"`
	Legend     string                 `json:"legend"`
	DataSource string                 `json:"data_source"`
	Hide       bool                   `json:"hide"`
	Options    map[string]interface{} `json:"options"`
}

// DataTransformation defines data transformations
type DataTransformation struct {
	Type    TransformationType     `json:"type"`
	Options map[string]interface{} `json:"options"`
}

// FieldConfig defines field-specific configuration
type FieldConfig struct {
	Defaults  *FieldDefaults   `json:"defaults"`
	Overrides []*FieldOverride `json:"overrides"`
}

// FieldDefaults defines default field settings
type FieldDefaults struct {
	Unit        string                 `json:"unit"`
	Min         *float64               `json:"min"`
	Max         *float64               `json:"max"`
	Decimals    int                    `json:"decimals"`
	DisplayName string                 `json:"display_name"`
	Color       *ColorConfig           `json:"color"`
	Thresholds  []*Threshold           `json:"thresholds"`
	Mappings    []*ValueMapping        `json:"mappings"`
	NoValue     string                 `json:"no_value"`
	Options     map[string]interface{} `json:"options"`
}

// FieldOverride defines field override rules
type FieldOverride struct {
	Matcher    *FieldMatcher          `json:"matcher"`
	Properties map[string]interface{} `json:"properties"`
}

// FieldMatcher defines field matching criteria
type FieldMatcher struct {
	ID      string `json:"id"`
	Options string `json:"options"`
}

// ColorConfig defines color configuration
type ColorConfig struct {
	Mode       ColorMode `json:"mode"`
	FixedColor string    `json:"fixed_color"`
	SeriesBy   SeriesBy  `json:"series_by"`
}

// ValueMapping defines value mappings
type ValueMapping struct {
	Type    MappingType            `json:"type"`
	Options map[string]interface{} `json:"options"`
}

// WidgetAlertRule defines widget-specific alert rules
type WidgetAlertRule struct {
	Enabled      bool              `json:"enabled"`
	Conditions   []*AlertCondition `json:"conditions"`
	Frequency    time.Duration     `json:"frequency"`
	Handler      string            `json:"handler"`
	Message      string            `json:"message"`
	Severity     AlertSeverity     `json:"severity"`
	NoDataState  NoDataState       `json:"no_data_state"`
	ExecErrState ExecErrState      `json:"exec_err_state"`
}

// ConditionalDisplay defines conditional widget display
type ConditionalDisplay struct {
	Conditions []*DisplayCondition `json:"conditions"`
	ShowMode   ShowMode            `json:"show_mode"`
}

// DisplayCondition defines when to show/hide widgets
type DisplayCondition struct {
	Variable string            `json:"variable"`
	Operator ConditionOperator `json:"operator"`
	Value    interface{}       `json:"value"`
}

// LiveStream handles real-time data streaming
type LiveStream struct {
	ID          string                    `json:"id"`
	DashboardID string                    `json:"dashboard_id"`
	WidgetID    string                    `json:"widget_id"`
	Query       string                    `json:"query"`
	Interval    time.Duration             `json:"interval"`
	Buffer      []DataPoint               `json:"buffer"`
	Subscribers map[string]chan DataPoint `json:"-"`
	LastUpdate  time.Time                 `json:"last_update"`
	Active      bool                      `json:"active"`
	mu          sync.RWMutex
}

// MetricCache caches metric query results
type MetricCache struct {
	data    map[string]*CacheEntry
	ttl     time.Duration
	maxSize int
	mu      sync.RWMutex
}

// CacheEntry represents a cached metric result
type CacheEntry struct {
	Data      interface{}   `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
	QueryHash string        `json:"query_hash"`
	TTL       time.Duration `json:"ttl"`
}

// VisualizationManager handles visualization rendering
type VisualizationManager struct {
	renderers map[ChartType]ChartRenderer
	logger    *logrus.Logger
}

// ChartRenderer interface for different chart types
type ChartRenderer interface {
	Render(data interface{}, config *VisualizationConfig) (interface{}, error)
	GetSupportedOptions() []string
	ValidateConfig(config *VisualizationConfig) error
}

// DataSource interface for metric data sources
type DataSource interface {
	Query(ctx context.Context, query string, timeRange *TimeRange) (interface{}, error)
	TestConnection() error
	GetMetrics() ([]string, error)
	GetLabels(metric string) ([]string, error)
	GetLabelValues(label string) ([]string, error)
}

// Theme defines dashboard theming
type Theme struct {
	Name       string                 `json:"name"`
	Colors     *ThemeColors           `json:"colors"`
	Typography *ThemeTypography       `json:"typography"`
	Spacing    *ThemeSpacing          `json:"spacing"`
	Borders    *ThemeBorders          `json:"borders"`
	Shadows    *ThemeShadows          `json:"shadows"`
	Animations *ThemeAnimations       `json:"animations"`
	Custom     map[string]interface{} `json:"custom"`
}

// ThemeColors defines color palette
type ThemeColors struct {
	Primary    string   `json:"primary"`
	Secondary  string   `json:"secondary"`
	Background string   `json:"background"`
	Surface    string   `json:"surface"`
	Text       string   `json:"text"`
	Success    string   `json:"success"`
	Warning    string   `json:"warning"`
	Error      string   `json:"error"`
	Info       string   `json:"info"`
	Chart      []string `json:"chart"`
	Gradient   []string `json:"gradient"`
}

// ThemeTypography defines typography settings
type ThemeTypography struct {
	FontFamily    string  `json:"font_family"`
	FontSize      int     `json:"font_size"`
	FontWeight    int     `json:"font_weight"`
	LineHeight    float64 `json:"line_height"`
	LetterSpacing float64 `json:"letter_spacing"`
}

// ThemeSpacing defines spacing settings
type ThemeSpacing struct {
	Unit   int `json:"unit"`
	Small  int `json:"small"`
	Medium int `json:"medium"`
	Large  int `json:"large"`
}

// ThemeBorders defines border settings
type ThemeBorders struct {
	Width  int    `json:"width"`
	Radius int    `json:"radius"`
	Style  string `json:"style"`
	Color  string `json:"color"`
}

// ThemeShadows defines shadow settings
type ThemeShadows struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Large  string `json:"large"`
}

// ThemeAnimations defines animation settings
type ThemeAnimations struct {
	Duration string `json:"duration"`
	Easing   string `json:"easing"`
}

// SeriesTransform defines series data transformations
type SeriesTransform struct {
	Type    TransformationType     `json:"type"`
	Options map[string]interface{} `json:"options"`
}

// Enums
type WidgetType string

const (
	WidgetTypeGraph      WidgetType = "graph"
	WidgetTypeSingleStat WidgetType = "singlestat"
	WidgetTypeTable      WidgetType = "table"
	WidgetTypeHeatmap    WidgetType = "heatmap"
	WidgetTypeGauge      WidgetType = "gauge"
	WidgetTypeBar        WidgetType = "bar"
	WidgetTypePie        WidgetType = "pie"
	WidgetTypeText       WidgetType = "text"
	WidgetTypeAlert      WidgetType = "alert"
	WidgetTypeLog        WidgetType = "log"
	WidgetTypeTrace      WidgetType = "trace"
	WidgetTypeMap        WidgetType = "map"
	WidgetTypeCustom     WidgetType = "custom"
)

type LayoutType string

const (
	LayoutGrid    LayoutType = "grid"
	LayoutMasonry LayoutType = "masonry"
	LayoutFlow    LayoutType = "flow"
	LayoutCustom  LayoutType = "custom"
)

type VariableType string

const (
	VarTypeQuery      VariableType = "query"
	VarTypeCustom     VariableType = "custom"
	VarTypeConstant   VariableType = "constant"
	VarTypeInterval   VariableType = "interval"
	VarTypeDataSource VariableType = "datasource"
)

type VariableRefresh string

const (
	RefreshNever   VariableRefresh = "never"
	RefreshOnLoad  VariableRefresh = "on_load"
	RefreshOnRange VariableRefresh = "on_range"
)

type AnnotationType string

const (
	AnnotationBuiltIn AnnotationType = "builtin"
	AnnotationQuery   AnnotationType = "query"
	AnnotationTags    AnnotationType = "tags"
)

type PermissionRole string

const (
	RoleViewer PermissionRole = "viewer"
	RoleEditor PermissionRole = "editor"
	RoleAdmin  PermissionRole = "admin"
)

type ChartType string

const (
	ChartLine        ChartType = "line"
	ChartArea        ChartType = "area"
	ChartBar         ChartType = "bar"
	ChartColumn      ChartType = "column"
	ChartPie         ChartType = "pie"
	ChartDonut       ChartType = "donut"
	ChartScatter     ChartType = "scatter"
	ChartHeatmap     ChartType = "heatmap"
	ChartGauge       ChartType = "gauge"
	ChartSingleStat  ChartType = "singlestat"
	ChartTable       ChartType = "table"
	ChartHistogram   ChartType = "histogram"
	ChartCandlestick ChartType = "candlestick"
)

type SeriesType string

const (
	SeriesLine    SeriesType = "line"
	SeriesBar     SeriesType = "bar"
	SeriesArea    SeriesType = "area"
	SeriesScatter SeriesType = "scatter"
)

type ScaleType string

const (
	ScaleLinear ScaleType = "linear"
	ScaleLog    ScaleType = "log"
	ScaleTime   ScaleType = "time"
)

type AxisPosition string

const (
	PositionLeft   AxisPosition = "left"
	PositionRight  AxisPosition = "right"
	PositionTop    AxisPosition = "top"
	PositionBottom AxisPosition = "bottom"
)

type LegendPosition string

const (
	LegendTop    LegendPosition = "top"
	LegendBottom LegendPosition = "bottom"
	LegendLeft   LegendPosition = "left"
	LegendRight  LegendPosition = "right"
)

type LegendAlignment string

const (
	AlignCenter LegendAlignment = "center"
	AlignLeft   LegendAlignment = "left"
	AlignRight  LegendAlignment = "right"
)

type TooltipSort string

const (
	SortNone TooltipSort = "none"
	SortAsc  TooltipSort = "asc"
	SortDesc TooltipSort = "desc"
)

type ValueType string

const (
	ValueIndividual ValueType = "individual"
	ValueCumulative ValueType = "cumulative"
)

type EasingType string

const (
	EaseLinear EasingType = "linear"
	EaseInOut  EasingType = "ease-in-out"
	EaseIn     EasingType = "ease-in"
	EaseOut    EasingType = "ease-out"
)

type ThresholdCondition string

const (
	ThresholdGT  ThresholdCondition = "gt"
	ThresholdLT  ThresholdCondition = "lt"
	ThresholdEQ  ThresholdCondition = "eq"
	ThresholdNEQ ThresholdCondition = "neq"
)

type TransformationType string

const (
	TransformReduce    TransformationType = "reduce"
	TransformGroupBy   TransformationType = "group_by"
	TransformFilter    TransformationType = "filter"
	TransformCalculate TransformationType = "calculate"
	TransformRename    TransformationType = "rename"
	TransformMerge     TransformationType = "merge"
	TransformSort      TransformationType = "sort"
)

type ColorMode string

const (
	ColorFixed    ColorMode = "fixed"
	ColorPalette  ColorMode = "palette"
	ColorValue    ColorMode = "value"
	ColorGradient ColorMode = "gradient"
)

type SeriesBy string

const (
	SeriesByLast SeriesBy = "last"
	SeriesByMin  SeriesBy = "min"
	SeriesByMax  SeriesBy = "max"
)

type MappingType string

const (
	MappingValue   MappingType = "value"
	MappingRange   MappingType = "range"
	MappingRegex   MappingType = "regex"
	MappingSpecial MappingType = "special"
)

type NoDataState string

const (
	NoDataNoData   NoDataState = "no_data"
	NoDataAlerting NoDataState = "alerting"
	NoDataOK       NoDataState = "ok"
)

type ExecErrState string

const (
	ExecErrAlerting ExecErrState = "alerting"
	ExecErrOK       ExecErrState = "ok"
)

type ShowMode string

const (
	ShowModeShow ShowMode = "show"
	ShowModeHide ShowMode = "hide"
)

type AggregationType string

const (
	AggSum    AggregationType = "sum"
	AggAvg    AggregationType = "avg"
	AggMin    AggregationType = "min"
	AggMax    AggregationType = "max"
	AggCount  AggregationType = "count"
	AggMedian AggregationType = "median"
	AggP95    AggregationType = "p95"
	AggP99    AggregationType = "p99"
	AggStdDev AggregationType = "stddev"
	AggFirst  AggregationType = "first"
	AggLast   AggregationType = "last"
)

// NewDashboardEngine creates a new dashboard engine
func NewDashboardEngine(config *DashboardConfig, logger *logrus.Logger) *DashboardEngine {
	if config == nil {
		config = DefaultDashboardConfig()
	}

	engine := &DashboardEngine{
		config:      config,
		logger:      logger,
		dashboards:  make(map[string]*Dashboard),
		widgets:     make(map[string]*Widget),
		dataSources: make(map[string]DataSource),
		liveStreams: make(map[string]*LiveStream),
		stopChan:    make(chan bool),
	}

	// Initialize components
	engine.metricCache = NewMetricCache(config.CacheSize, config.CacheTTL)
	engine.visualizationMgr = NewVisualizationManager(logger)

	return engine
}

// DefaultDashboardConfig returns default dashboard configuration
func DefaultDashboardConfig() *DashboardConfig {
	return &DashboardConfig{
		Enabled:              true,
		RefreshInterval:      time.Second * 30,
		MaxDataPoints:        1000,
		CacheSize:            100,
		CacheTTL:             time.Minute * 5,
		DefaultTimeRange:     time.Hour,
		LiveStreamBufferSize: 100,
		MaxConcurrentQueries: 10,
		DefaultTheme:         "dark",
		ExportFormats:        []string{"png", "pdf", "csv", "json"},
		WebSocketEndpoint:    "/ws/dashboard",
		EnableRealTime:       true,
		EnableExports:        true,
		CustomThemes: map[string]*Theme{
			"dark": {
				Name: "Dark Theme",
				Colors: &ThemeColors{
					Primary:    "#1f77b4",
					Secondary:  "#ff7f0e",
					Background: "#1e1e1e",
					Surface:    "#2d2d2d",
					Text:       "#ffffff",
					Success:    "#2ca02c",
					Warning:    "#ff7f0e",
					Error:      "#d62728",
					Info:       "#17a2b8",
					Chart:      []string{"#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd", "#8c564b", "#e377c2", "#7f7f7f", "#bcbd22", "#17becf"},
				},
			},
		},
	}
}

// Start starts the dashboard engine
func (de *DashboardEngine) Start(ctx context.Context) error {
	if !de.config.Enabled {
		de.logger.Info("Dashboard engine is disabled")
		return nil
	}

	de.logger.Info("Starting dashboard engine")

	// Start live stream manager
	if de.config.EnableRealTime {
		go de.manageLiveStreams(ctx)
	}

	// Start cache cleanup routine
	go de.cacheCleanupRoutine(ctx)

	return nil
}

// Stop stops the dashboard engine
func (de *DashboardEngine) Stop() {
	de.logger.Info("Stopping dashboard engine")
	close(de.stopChan)
}

// CreateDashboard creates a new dashboard
func (de *DashboardEngine) CreateDashboard(dashboard *Dashboard) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	if dashboard.ID == "" {
		dashboard.ID = generateDashboardID()
	}

	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()
	dashboard.Version = 1

	de.dashboards[dashboard.ID] = dashboard

	de.logger.Infof("Created dashboard: %s", dashboard.Name)
	return nil
}

// GetDashboard retrieves a dashboard by ID
func (de *DashboardEngine) GetDashboard(id string) (*Dashboard, error) {
	de.mu.RLock()
	defer de.mu.RUnlock()

	dashboard, exists := de.dashboards[id]
	if !exists {
		return nil, fmt.Errorf("dashboard not found: %s", id)
	}

	return dashboard, nil
}

// UpdateDashboard updates an existing dashboard
func (de *DashboardEngine) UpdateDashboard(dashboard *Dashboard) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	existing, exists := de.dashboards[dashboard.ID]
	if !exists {
		return fmt.Errorf("dashboard not found: %s", dashboard.ID)
	}

	dashboard.Version = existing.Version + 1
	dashboard.UpdatedAt = time.Now()
	dashboard.CreatedAt = existing.CreatedAt

	de.dashboards[dashboard.ID] = dashboard

	de.logger.Infof("Updated dashboard: %s", dashboard.Name)
	return nil
}

// DeleteDashboard deletes a dashboard
func (de *DashboardEngine) DeleteDashboard(id string) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	_, exists := de.dashboards[id]
	if !exists {
		return fmt.Errorf("dashboard not found: %s", id)
	}

	delete(de.dashboards, id)

	// Stop any live streams for this dashboard
	for streamID, stream := range de.liveStreams {
		if stream.DashboardID == id {
			stream.Active = false
			delete(de.liveStreams, streamID)
		}
	}

	de.logger.Infof("Deleted dashboard: %s", id)
	return nil
}

// GetDashboardData retrieves data for a dashboard
func (de *DashboardEngine) GetDashboardData(dashboardID string, timeRange *TimeRange) (map[string]interface{}, error) {
	dashboard, err := de.GetDashboard(dashboardID)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})

	// Get data for each widget
	for _, widget := range dashboard.Widgets {
		widgetData, err := de.getWidgetData(widget, timeRange)
		if err != nil {
			de.logger.WithError(err).Errorf("Failed to get data for widget %s", widget.ID)
			continue
		}
		data[widget.ID] = widgetData
	}

	return data, nil
}

// getWidgetData retrieves data for a specific widget
func (de *DashboardEngine) getWidgetData(widget *Widget, timeRange *TimeRange) (interface{}, error) {
	// Check cache first
	cacheKey := generateCacheKey(widget, timeRange)
	if cachedData := de.metricCache.Get(cacheKey); cachedData != nil {
		return cachedData, nil
	}

	// Query data source
	dataSource, exists := de.dataSources[widget.DataSource]
	if !exists {
		return nil, fmt.Errorf("data source not found: %s", widget.DataSource)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	rawData, err := dataSource.Query(ctx, widget.Query.Query, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to query data source: %w", err)
	}

	// Apply transformations
	transformedData := de.applyTransformations(rawData, widget.Transformations)

	// Generate visualization
	visualizationData, err := de.visualizationMgr.RenderVisualization(widget.Type, transformedData, widget.Visualization)
	if err != nil {
		return nil, fmt.Errorf("failed to render visualization: %w", err)
	}

	// Cache the result
	de.metricCache.Set(cacheKey, visualizationData)

	return visualizationData, nil
}

// RegisterDataSource registers a data source
func (de *DashboardEngine) RegisterDataSource(name string, dataSource DataSource) {
	de.mu.Lock()
	defer de.mu.Unlock()

	de.dataSources[name] = dataSource
	de.logger.Infof("Registered data source: %s", name)
}

// StartLiveStream starts a live stream for a widget
func (de *DashboardEngine) StartLiveStream(dashboardID, widgetID string) (*LiveStream, error) {
	de.mu.Lock()
	defer de.mu.Unlock()

	dashboard, exists := de.dashboards[dashboardID]
	if !exists {
		return nil, fmt.Errorf("dashboard not found: %s", dashboardID)
	}

	var widget *Widget
	for _, w := range dashboard.Widgets {
		if w.ID == widgetID {
			widget = w
			break
		}
	}

	if widget == nil {
		return nil, fmt.Errorf("widget not found: %s", widgetID)
	}

	streamID := fmt.Sprintf("%s_%s", dashboardID, widgetID)

	stream := &LiveStream{
		ID:          streamID,
		DashboardID: dashboardID,
		WidgetID:    widgetID,
		Query:       widget.Query.Query,
		Interval:    widget.RefreshRate,
		Buffer:      make([]DataPoint, 0, de.config.LiveStreamBufferSize),
		Subscribers: make(map[string]chan DataPoint),
		Active:      true,
	}

	de.liveStreams[streamID] = stream

	// Start stream goroutine
	go de.runLiveStream(stream)

	de.logger.Infof("Started live stream: %s", streamID)
	return stream, nil
}

// manageLiveStreams manages all live streams
func (de *DashboardEngine) manageLiveStreams(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-de.stopChan:
			return
		case <-ticker.C:
			de.updateLiveStreams()
		}
	}
}

// updateLiveStreams updates all active live streams
func (de *DashboardEngine) updateLiveStreams() {
	de.mu.RLock()
	streams := make([]*LiveStream, 0, len(de.liveStreams))
	for _, stream := range de.liveStreams {
		if stream.Active {
			streams = append(streams, stream)
		}
	}
	de.mu.RUnlock()

	for _, stream := range streams {
		if time.Since(stream.LastUpdate) >= stream.Interval {
			go de.updateLiveStream(stream)
		}
	}
}

// updateLiveStream updates a single live stream
func (de *DashboardEngine) updateLiveStream(stream *LiveStream) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Get data source
	de.mu.RLock()
	dashboard := de.dashboards[stream.DashboardID]
	de.mu.RUnlock()

	if dashboard == nil {
		return
	}

	var widget *Widget
	for _, w := range dashboard.Widgets {
		if w.ID == stream.WidgetID {
			widget = w
			break
		}
	}

	if widget == nil {
		return
	}

	dataSource := de.dataSources[widget.DataSource]
	if dataSource == nil {
		return
	}

	// Query latest data
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	timeRange := &TimeRange{
		Start: time.Now().Add(-time.Minute),
		End:   time.Now(),
	}

	rawData, err := dataSource.Query(ctx, stream.Query, timeRange)
	if err != nil {
		de.logger.WithError(err).Errorf("Failed to query live stream data: %s", stream.ID)
		return
	}

	// Convert to data points (simplified)
	dataPoint := DataPoint{
		Timestamp: time.Now(),
		Value:     extractValue(rawData), // Helper function to extract numeric value
	}

	// Add to buffer
	stream.Buffer = append(stream.Buffer, dataPoint)
	if len(stream.Buffer) > de.config.LiveStreamBufferSize {
		stream.Buffer = stream.Buffer[1:]
	}

	// Notify subscribers
	for _, subscriber := range stream.Subscribers {
		select {
		case subscriber <- dataPoint:
		default:
			// Subscriber channel is full, skip
		}
	}

	stream.LastUpdate = time.Now()
}

// Helper functions
func generateDashboardID() string {
	return fmt.Sprintf("dashboard_%d", time.Now().UnixNano())
}

func generateCacheKey(widget *Widget, timeRange *TimeRange) string {
	data, _ := json.Marshal(map[string]interface{}{
		"widget_id": widget.ID,
		"query":     widget.Query.Query,
		"start":     timeRange.Start.Unix(),
		"end":       timeRange.End.Unix(),
	})
	return fmt.Sprintf("%x", data)[:16]
}

func extractValue(data interface{}) float64 {
	// Simplified value extraction - in real implementation would handle various data types
	return float64(time.Now().Unix() % 100)
}

func (de *DashboardEngine) applyTransformations(data interface{}, transformations []*DataTransformation) interface{} {
	// Apply each transformation in sequence
	result := data
	for _, transform := range transformations {
		result = de.applyTransformation(result, transform)
	}
	return result
}

func (de *DashboardEngine) applyTransformation(data interface{}, transform *DataTransformation) interface{} {
	// Implementation would handle different transformation types
	return data
}

func (de *DashboardEngine) runLiveStream(stream *LiveStream) {
	// Implementation for running individual live stream
}

func (de *DashboardEngine) cacheCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-de.stopChan:
			return
		case <-ticker.C:
			de.metricCache.Cleanup()
		}
	}
}

// Component implementations
func NewMetricCache(size int, ttl time.Duration) *MetricCache {
	return &MetricCache{
		data:    make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: size,
	}
}

func (mc *MetricCache) Get(key string) interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists {
		return nil
	}

	if time.Since(entry.Timestamp) > mc.ttl {
		delete(mc.data, key)
		return nil
	}

	return entry.Data
}

func (mc *MetricCache) Set(key string, data interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Remove oldest entry if cache is full
	if len(mc.data) >= mc.maxSize {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range mc.data {
			if oldestKey == "" || v.Timestamp.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.Timestamp
			}
		}
		delete(mc.data, oldestKey)
	}

	mc.data[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
}

func (mc *MetricCache) Cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.ttl)
	for key, entry := range mc.data {
		if entry.Timestamp.Before(cutoff) {
			delete(mc.data, key)
		}
	}
}

func NewVisualizationManager(logger *logrus.Logger) *VisualizationManager {
	return &VisualizationManager{
		renderers: make(map[ChartType]ChartRenderer),
		logger:    logger,
	}
}

func (vm *VisualizationManager) RenderVisualization(widgetType WidgetType, data interface{}, config *VisualizationConfig) (interface{}, error) {
	// Implementation would render different visualization types
	return map[string]interface{}{
		"type":   widgetType,
		"data":   data,
		"config": config,
	}, nil
}
