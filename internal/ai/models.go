package ai

import (
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"time"
)

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Messages     []ChatMessage        `json:"messages" binding:"required"`
	Model        string               `json:"model,omitempty"`
	MaxTokens    int                  `json:"max_tokens,omitempty"`
	Temperature  float64              `json:"temperature,omitempty"`
	TopP         float64              `json:"top_p,omitempty"`
	Stream       bool                 `json:"stream,omitempty"`
	SystemPrompt string               `json:"system_prompt,omitempty"`
	Context      *ConversationContext `json:"context,omitempty"`
	Provider     string               `json:"provider,omitempty"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
}

// CompletionRequest represents an incoming completion request
type CompletionRequest struct {
	Prompt      string            `json:"prompt" binding:"required"`
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Stop        []string          `json:"stop,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ConversationContext provides context for AI responses
type ConversationContext struct {
	UserID          string                 `json:"user_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	HomeAssistant   *HAContext             `json:"home_assistant,omitempty"`
	Entities        []EntityContext        `json:"entities,omitempty"`
	Rooms           []RoomContext          `json:"rooms,omitempty"`
	RecentActions   []ActionContext        `json:"recent_actions,omitempty"`
	SystemStatus    *SystemStatusContext   `json:"system_status,omitempty"`
	UserPreferences map[string]interface{} `json:"user_preferences,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
}

// HAContext represents Home Assistant context
type HAContext struct {
	Connected    bool                   `json:"connected"`
	EntityCount  int                    `json:"entity_count"`
	LastSync     time.Time              `json:"last_sync"`
	RecentEvents []HAEventContext       `json:"recent_events,omitempty"`
	ActiveScenes []string               `json:"active_scenes,omitempty"`
	SystemInfo   map[string]interface{} `json:"system_info,omitempty"`
}

// EntityContext represents entity information for AI context
type EntityContext struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	State        string                 `json:"state"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Room         string                 `json:"room,omitempty"`
	LastChanged  time.Time              `json:"last_changed"`
	Availability string                 `json:"availability"`
}

// RoomContext represents room information for AI context
type RoomContext struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	EntityCount int             `json:"entity_count"`
	Entities    []EntityContext `json:"entities,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Humidity    float64         `json:"humidity,omitempty"`
	Occupied    bool            `json:"occupied,omitempty"`
}

// ActionContext represents recent user actions
type ActionContext struct {
	Type       string                 `json:"type"`
	Target     string                 `json:"target"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Result     string                 `json:"result,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Success    bool                   `json:"success"`
}

// HAEventContext represents Home Assistant events
type HAEventContext struct {
	EventType string                 `json:"event_type"`
	EntityID  string                 `json:"entity_id,omitempty"`
	NewState  string                 `json:"new_state,omitempty"`
	OldState  string                 `json:"old_state,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// SystemStatusContext represents overall system status
type SystemStatusContext struct {
	CPUUsage          float64           `json:"cpu_usage"`
	MemoryUsage       float64           `json:"memory_usage"`
	DiskUsage         float64           `json:"disk_usage"`
	NetworkStatus     string            `json:"network_status"`
	DatabaseStatus    string            `json:"database_status"`
	ServiceStatuses   map[string]string `json:"service_statuses,omitempty"`
	ActiveConnections int               `json:"active_connections"`
	Uptime            time.Duration     `json:"uptime"`
	LastBackup        time.Time         `json:"last_backup"`
	Alerts            []SystemAlert     `json:"alerts,omitempty"`
}

// SystemAlert represents system alerts
type SystemAlert struct {
	Level     string                 `json:"level"` // "info", "warning", "error", "critical"
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Resolved  bool                   `json:"resolved"`
}

// EntityAnalysisRequest represents a request to analyze entity patterns
type EntityAnalysisRequest struct {
	EntityIDs    []string               `json:"entity_ids" binding:"required"`
	TimeRange    TimeRange              `json:"time_range"`
	AnalysisType string                 `json:"analysis_type"` // "patterns", "anomalies", "trends", "suggestions"
	Options      map[string]interface{} `json:"options,omitempty"`
}

// TimeRange represents a time range for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// EntityAnalysisResponse represents the response from entity analysis
type EntityAnalysisResponse struct {
	EntityID     string                 `json:"entity_id"`
	AnalysisType string                 `json:"analysis_type"`
	Insights     []AnalysisInsight      `json:"insights"`
	Patterns     []PatternInsight       `json:"patterns,omitempty"`
	Anomalies    []AnomalyInsight       `json:"anomalies,omitempty"`
	Suggestions  []AutomationSuggestion `json:"suggestions,omitempty"`
	Confidence   float64                `json:"confidence"`
	ProcessedAt  time.Time              `json:"processed_at"`
}

// AnalysisInsight represents a general insight from analysis
type AnalysisInsight struct {
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Actionable  bool                   `json:"actionable"`
}

// PatternInsight represents discovered patterns
type PatternInsight struct {
	Type        string       `json:"type"` // "daily", "weekly", "seasonal", "event_based"
	Description string       `json:"description"`
	Frequency   string       `json:"frequency"`
	Confidence  float64      `json:"confidence"`
	Examples    []string     `json:"examples,omitempty"`
	TimePattern *TimePattern `json:"time_pattern,omitempty"`
	Triggers    []string     `json:"triggers,omitempty"`
}

// TimePattern represents time-based patterns
type TimePattern struct {
	DaysOfWeek []int  `json:"days_of_week,omitempty"` // 0=Sunday, 1=Monday, etc.
	Hours      []int  `json:"hours,omitempty"`        // 0-23
	Duration   string `json:"duration,omitempty"`     // "5m", "1h", etc.
	Recurrence string `json:"recurrence,omitempty"`   // "daily", "weekly", etc.
}

// AnomalyInsight represents detected anomalies
type AnomalyInsight struct {
	Type        string                 `json:"type"` // "value", "timing", "frequency", "duration"
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"` // "low", "medium", "high", "critical"
	Confidence  float64                `json:"confidence"`
	DetectedAt  time.Time              `json:"detected_at"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Suggested   bool                   `json:"suggested"` // Whether action is suggested
}

// AutomationSuggestion represents suggested automations
type AutomationSuggestion struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Type            string                `json:"type"` // "schedule", "trigger", "condition", "scene"
	Confidence      float64               `json:"confidence"`
	Benefits        []string              `json:"benefits"`
	Triggers        []TriggerSuggestion   `json:"triggers"`
	Actions         []ActionSuggestion    `json:"actions"`
	Conditions      []ConditionSuggestion `json:"conditions,omitempty"`
	Schedule        *ScheduleSuggestion   `json:"schedule,omitempty"`
	EstimatedImpact string                `json:"estimated_impact"`
	Complexity      string                `json:"complexity"` // "simple", "moderate", "complex"
}

// TriggerSuggestion represents automation trigger suggestions
type TriggerSuggestion struct {
	Type      string                 `json:"type"` // "state", "time", "event", "numeric_state"
	EntityID  string                 `json:"entity_id,omitempty"`
	FromState string                 `json:"from_state,omitempty"`
	ToState   string                 `json:"to_state,omitempty"`
	Attribute string                 `json:"attribute,omitempty"`
	Above     *float64               `json:"above,omitempty"`
	Below     *float64               `json:"below,omitempty"`
	Platform  string                 `json:"platform,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Time      string                 `json:"time,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ActionSuggestion represents automation action suggestions
type ActionSuggestion struct {
	Type     string                 `json:"type"` // "call_service", "set_state", "scene", "script"
	Service  string                 `json:"service,omitempty"`
	EntityID string                 `json:"entity_id,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Target   map[string]interface{} `json:"target,omitempty"`
	Delay    string                 `json:"delay,omitempty"`
	WaitFor  string                 `json:"wait_for,omitempty"`
}

// ConditionSuggestion represents automation condition suggestions
type ConditionSuggestion struct {
	Type      string   `json:"type"` // "state", "numeric_state", "time", "template"
	EntityID  string   `json:"entity_id,omitempty"`
	State     string   `json:"state,omitempty"`
	Attribute string   `json:"attribute,omitempty"`
	Above     *float64 `json:"above,omitempty"`
	Below     *float64 `json:"below,omitempty"`
	After     string   `json:"after,omitempty"`
	Before    string   `json:"before,omitempty"`
	Weekday   []string `json:"weekday,omitempty"`
	Template  string   `json:"template,omitempty"`
}

// ScheduleSuggestion represents schedule-based automation suggestions
type ScheduleSuggestion struct {
	Time       string   `json:"time,omitempty"`
	DaysOfWeek []string `json:"days_of_week,omitempty"`
	Months     []string `json:"months,omitempty"`
	Cron       string   `json:"cron,omitempty"`
}

// AutomationGenerationRequest represents a request to generate automation rules
type AutomationGenerationRequest struct {
	Description string                 `json:"description" binding:"required"`
	EntityIDs   []string               `json:"entity_ids,omitempty"`
	RoomIDs     []string               `json:"room_ids,omitempty"`
	Complexity  string                 `json:"complexity,omitempty"` // "simple", "moderate", "complex"
	Options     map[string]interface{} `json:"options,omitempty"`
	Context     *ConversationContext   `json:"context,omitempty"`
}

// AutomationGenerationResponse represents generated automation rules
type AutomationGenerationResponse struct {
	Automations []GeneratedAutomation `json:"automations"`
	Summary     string                `json:"summary"`
	Warnings    []string              `json:"warnings,omitempty"`
	Suggestions []string              `json:"suggestions,omitempty"`
	Confidence  float64               `json:"confidence"`
	GeneratedAt time.Time             `json:"generated_at"`
}

// GeneratedAutomation represents a single generated automation
type GeneratedAutomation struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Triggers    []TriggerSuggestion    `json:"triggers"`
	Actions     []ActionSuggestion     `json:"actions"`
	Conditions  []ConditionSuggestion  `json:"conditions,omitempty"`
	HAConfig    map[string]interface{} `json:"ha_config"` // Home Assistant YAML config
	Complexity  string                 `json:"complexity"`
	Benefits    []string               `json:"benefits"`
	Risks       []string               `json:"risks,omitempty"`
	TestSteps   []string               `json:"test_steps,omitempty"`
}

// SystemSummaryRequest represents a request for system status summary
type SystemSummaryRequest struct {
	IncludeEntities   bool     `json:"include_entities,omitempty"`
	IncludeRooms      bool     `json:"include_rooms,omitempty"`
	IncludeAutomation bool     `json:"include_automation,omitempty"`
	IncludeAlerts     bool     `json:"include_alerts,omitempty"`
	EntityTypes       []string `json:"entity_types,omitempty"`
	DetailLevel       string   `json:"detail_level,omitempty"` // "brief", "normal", "detailed"
}

// SystemSummaryResponse represents system status summary
type SystemSummaryResponse struct {
	Summary      string               `json:"summary"`
	Highlights   []string             `json:"highlights"`
	Concerns     []string             `json:"concerns,omitempty"`
	Suggestions  []string             `json:"suggestions,omitempty"`
	EntityStats  *EntityStats         `json:"entity_stats,omitempty"`
	RoomStats    *RoomStats           `json:"room_stats,omitempty"`
	SystemHealth *SystemHealthSummary `json:"system_health,omitempty"`
	GeneratedAt  time.Time            `json:"generated_at"`
}

// EntityStats represents entity statistics
type EntityStats struct {
	Total         int                  `json:"total"`
	Available     int                  `json:"available"`
	Unavailable   int                  `json:"unavailable"`
	ByType        map[string]int       `json:"by_type"`
	ByRoom        map[string]int       `json:"by_room"`
	RecentChanges []EntityChangeRecord `json:"recent_changes,omitempty"`
}

// RoomStats represents room statistics
type RoomStats struct {
	Total       int                `json:"total"`
	Occupied    int                `json:"occupied"`
	ByType      map[string]int     `json:"by_type,omitempty"`
	Temperature map[string]float64 `json:"temperature,omitempty"`
	Humidity    map[string]float64 `json:"humidity,omitempty"`
}

// SystemHealthSummary represents overall system health
type SystemHealthSummary struct {
	OverallStatus string             `json:"overall_status"` // "healthy", "warning", "critical"
	Services      map[string]string  `json:"services"`
	Resources     ResourceUsage      `json:"resources"`
	Connectivity  ConnectivityStatus `json:"connectivity"`
	LastBackup    time.Time          `json:"last_backup"`
	Uptime        time.Duration      `json:"uptime"`
}

// ResourceUsage represents system resource usage
type ResourceUsage struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Disk   float64 `json:"disk"`
	Status string  `json:"status"` // "normal", "high", "critical"
}

// ConnectivityStatus represents connectivity status
type ConnectivityStatus struct {
	HomeAssistant bool            `json:"home_assistant"`
	Internet      bool            `json:"internet"`
	LocalNetwork  bool            `json:"local_network"`
	Services      map[string]bool `json:"services,omitempty"`
	LastChecked   time.Time       `json:"last_checked"`
}

// EntityChangeRecord represents recent entity changes
type EntityChangeRecord struct {
	EntityID   string    `json:"entity_id"`
	EntityName string    `json:"entity_name"`
	OldState   string    `json:"old_state"`
	NewState   string    `json:"new_state"`
	ChangedAt  time.Time `json:"changed_at"`
	ChangeType string    `json:"change_type"` // "state", "attribute", "availability"
}

// AI Settings & Management Models

// AISettingsResponse represents AI configuration settings
type AISettingsResponse struct {
	Providers       []AIProviderInfo `json:"providers"`
	DefaultProvider string           `json:"default_provider"`
	FallbackEnabled bool             `json:"fallback_enabled"`
	MaxRetries      int              `json:"max_retries"`
	Timeout         string           `json:"timeout"`
	LastUpdated     time.Time        `json:"last_updated"`
}

// AIProviderInfo represents AI provider information
type AIProviderInfo struct {
	Type         string                 `json:"type"`
	Enabled      bool                   `json:"enabled"`
	URL          string                 `json:"url,omitempty"`
	DefaultModel string                 `json:"default_model"`
	Models       []string               `json:"models,omitempty"`
	Priority     int                    `json:"priority"`
	Status       string                 `json:"status"` // "connected", "disconnected", "error"
	LastChecked  time.Time              `json:"last_checked"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// AISettingsRequest represents AI settings update request
type AISettingsRequest struct {
	Providers       []config.AIProviderConfig `json:"providers"`
	DefaultProvider string                    `json:"default_provider"`
	FallbackEnabled bool                      `json:"fallback_enabled"`
	MaxRetries      int                       `json:"max_retries"`
	Timeout         string                    `json:"timeout"`
}

// AIConnectionTestRequest represents connection test request
type AIConnectionTestRequest struct {
	ProviderType string                 `json:"provider_type"`
	URL          string                 `json:"url,omitempty"`
	APIKey       string                 `json:"api_key,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// AIConnectionTestResponse represents connection test response
type AIConnectionTestResponse struct {
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
	Models      []string  `json:"models,omitempty"`
	Latency     string    `json:"latency,omitempty"`
	TestedAt    time.Time `json:"tested_at"`
	ErrorDetail string    `json:"error_detail,omitempty"`
}

// Ollama Management Models

// OllamaStatusResponse represents Ollama process status
type OllamaStatusResponse struct {
	Running       bool               `json:"running"`
	ProcessID     int                `json:"process_id,omitempty"`
	StartTime     time.Time          `json:"start_time,omitempty"`
	Version       string             `json:"version,omitempty"`
	Models        []OllamaModelInfo  `json:"models,omitempty"`
	SystemInfo    OllamaSystemInfo   `json:"system_info"`
	ResourceUsage OllamaResourceInfo `json:"resource_usage"`
	LastChecked   time.Time          `json:"last_checked"`
}

// OllamaModelInfo represents Ollama model information
type OllamaModelInfo struct {
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	Digest        string    `json:"digest"`
	ModifiedAt    time.Time `json:"modified_at"`
	Details       string    `json:"details,omitempty"`
	Family        string    `json:"family,omitempty"`
	Format        string    `json:"format,omitempty"`
	ParameterSize string    `json:"parameter_size,omitempty"`
}

// OllamaSystemInfo represents Ollama system information
type OllamaSystemInfo struct {
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	GPU          bool   `json:"gpu"`
	GPUInfo      string `json:"gpu_info,omitempty"`
	Memory       int64  `json:"memory"`
}

// OllamaResourceInfo represents Ollama resource usage
type OllamaResourceInfo struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage int64     `json:"memory_usage"`
	GPUUsage    float64   `json:"gpu_usage,omitempty"`
	ActiveModel string    `json:"active_model,omitempty"`
	LastRequest time.Time `json:"last_request,omitempty"`
}

// OllamaMetricsResponse represents Ollama metrics
type OllamaMetricsResponse struct {
	Status         OllamaStatusResponse `json:"status"`
	RequestCount   int64                `json:"request_count"`
	ErrorCount     int64                `json:"error_count"`
	AverageLatency float64              `json:"average_latency"`
	TotalUptime    string               `json:"total_uptime"`
	Health         string               `json:"health"` // "healthy", "degraded", "down"
}

// OllamaProcessResponse represents Ollama process control response
type OllamaProcessResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	ProcessID int       `json:"process_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
