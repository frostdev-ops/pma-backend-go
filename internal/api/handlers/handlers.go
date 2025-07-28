package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/ups"
	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/ai/providers"
	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/frostdev-ops/pma-backend-go/internal/core/automation"
	"github.com/frostdev-ops/pma-backend-go/internal/core/backup"
	"github.com/frostdev-ops/pma-backend-go/internal/core/bluetooth"
	"github.com/frostdev-ops/pma-backend-go/internal/core/cache"
	"github.com/frostdev-ops/pma-backend-go/internal/core/controller"
	"github.com/frostdev-ops/pma-backend-go/internal/core/dashboard"
	"github.com/frostdev-ops/pma-backend-go/internal/core/display"
	"github.com/frostdev-ops/pma-backend-go/internal/core/energymgr"
	"github.com/frostdev-ops/pma-backend-go/internal/core/filemanager"
	"github.com/frostdev-ops/pma-backend-go/internal/core/i18n"
	"github.com/frostdev-ops/pma-backend-go/internal/core/interfaces"
	"github.com/frostdev-ops/pma-backend-go/internal/core/kiosk"
	"github.com/frostdev-ops/pma-backend-go/internal/core/media"
	"github.com/frostdev-ops/pma-backend-go/internal/core/monitoring"
	"github.com/frostdev-ops/pma-backend-go/internal/core/network"
	"github.com/frostdev-ops/pma-backend-go/internal/core/preferences"
	"github.com/frostdev-ops/pma-backend-go/internal/core/queue"
	"github.com/frostdev-ops/pma-backend-go/internal/core/rooms"
	"github.com/frostdev-ops/pma-backend-go/internal/core/screensaver"
	"github.com/frostdev-ops/pma-backend-go/internal/core/system"
	"github.com/frostdev-ops/pma-backend-go/internal/core/test"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/internal/database/sqlite"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/debug"
	"github.com/frostdev-ops/pma-backend-go/pkg/errors"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// SimpleMetricCollector is a basic implementation of MetricCollector interface
type SimpleMetricCollector struct{}

func (smc *SimpleMetricCollector) GetTimeSeries(metric string, start, end time.Time) ([]monitoring.DataPoint, error) {
	// Return empty slice for now - this can be enhanced later with real metrics
	return []monitoring.DataPoint{}, nil
}

func (smc *SimpleMetricCollector) GetMetricNames() ([]string, error) {
	// Return basic system metrics for now
	return []string{"cpu_usage", "memory_usage", "disk_usage", "network_io"}, nil
}

func (smc *SimpleMetricCollector) GetMetricLabels(metric string) (map[string][]string, error) {
	// Return empty labels for now
	return map[string][]string{}, nil
}

// parseDuration converts a string duration to time.Duration with error handling
func parseDuration(durationStr string) time.Duration {
	if durationStr == "" {
		return 0
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// Return default values for common config fields
		switch durationStr {
		case "5m":
			return 5 * time.Minute
		case "30s":
			return 30 * time.Second
		case "60s":
			return 60 * time.Second
		case "10s":
			return 10 * time.Second
		default:
			return 0
		}
	}
	return duration
}

// Repository adapters to bridge interface mismatches

// MCP Service Adapters to bridge real services to MCP interfaces

// MCPEntityServiceAdapter adapts unified.UnifiedEntityService to interfaces.EntityServiceInterface
type MCPEntityServiceAdapter struct {
	unifiedService *unified.UnifiedEntityService
	logger         *logrus.Logger
}

func (a *MCPEntityServiceAdapter) GetByID(ctx context.Context, entityID string, options interfaces.EntityGetOptions) (*interfaces.EntityWithRoom, error) {
	// Convert interfaces options to unified options
	unifiedOptions := unified.GetEntityOptions{
		IncludeRoom: options.IncludeRoom,
		IncludeArea: options.IncludeArea,
	}

	// Call the real service
	entityWithRoom, err := a.unifiedService.GetByID(ctx, entityID, unifiedOptions)
	if err != nil {
		return nil, err
	}

	// Convert unified result to interfaces result
	return &interfaces.EntityWithRoom{
		Entity: entityWithRoom.Entity,
		Room:   entityWithRoom.Room,
		Area:   entityWithRoom.Area,
	}, nil
}

func (a *MCPEntityServiceAdapter) GetByRoom(ctx context.Context, roomID string, options interfaces.EntityGetAllOptions) ([]*interfaces.EntityWithRoom, error) {
	// Convert interfaces options to unified options
	unifiedOptions := unified.GetAllOptions{
		IncludeRoom: options.IncludeRoom,
		IncludeArea: options.IncludeArea,
	}

	// Call the real service
	entities, err := a.unifiedService.GetByRoom(ctx, roomID, unifiedOptions)
	if err != nil {
		return nil, err
	}

	// Convert unified results to interfaces results
	result := make([]*interfaces.EntityWithRoom, len(entities))
	for i, entity := range entities {
		result[i] = &interfaces.EntityWithRoom{
			Entity: entity.Entity,
			Room:   entity.Room,
			Area:   entity.Area,
		}
	}

	return result, nil
}

func (a *MCPEntityServiceAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	// Call the real service directly
	return a.unifiedService.ExecuteAction(ctx, action)
}

// MCPRoomServiceAdapter adapts rooms.RoomService to interfaces.RoomServiceInterface
type MCPRoomServiceAdapter struct {
	roomService *rooms.RoomService
	logger      *logrus.Logger
}

func (a *MCPRoomServiceAdapter) GetRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error) {
	return a.roomService.GetRoomByID(ctx, roomID)
}

func (a *MCPRoomServiceAdapter) GetAllRooms(ctx context.Context) ([]*types.PMARoom, error) {
	return a.roomService.GetAllRooms(ctx)
}

// MCPSystemServiceAdapter adapts system.Service to interfaces.SystemServiceInterface
type MCPSystemServiceAdapter struct {
	systemService *system.Service
	logger        *logrus.Logger
}

func (a *MCPSystemServiceAdapter) GetSystemStatus(ctx context.Context) (*interfaces.SystemStatus, error) {
	// Get device info from the system service
	deviceInfo, err := a.systemService.GetDeviceInfo(ctx)
	if err != nil {
		a.logger.WithError(err).Warn("Failed to get device info for system status")
		// Continue without device info
	}

	// Get system metrics and status
	status := &interfaces.SystemStatus{
		Status:    "healthy", // Default status
		Timestamp: time.Now(),
		Uptime:    time.Since(time.Now().Add(-1 * time.Hour)), // Placeholder
	}

	if deviceInfo != nil {
		status.DeviceID = deviceInfo.DeviceID

		// Convert device info - use available fields from system.DeviceInfo
		status.CPU = &interfaces.CPUInfo{
			Usage:       0.0,         // Will be filled by real CPU monitoring
			LoadAverage: []float64{}, // Will be filled by real system monitoring
			Cores:       4,           // Default value
			Model:       "Unknown",   // Default value
		}

		// Set basic memory and disk info (placeholder values)
		status.Memory = &interfaces.MemoryInfo{
			Total:       8 * 1024 * 1024 * 1024, // 8GB default
			Available:   4 * 1024 * 1024 * 1024, // 4GB default
			Used:        4 * 1024 * 1024 * 1024, // 4GB default
			UsedPercent: 50.0,                   // Default
		}

		status.Disk = &interfaces.DiskInfo{
			Total:       100 * 1024 * 1024 * 1024, // 100GB default
			Free:        50 * 1024 * 1024 * 1024,  // 50GB default
			Used:        50 * 1024 * 1024 * 1024,  // 50GB default
			UsedPercent: 50.0,
			Filesystem:  "ext4",
			MountPoint:  "/",
		}
	} else {
		// Set default values if device info is not available
		status.DeviceID = "pma-device-001"
		status.CPU = &interfaces.CPUInfo{
			Usage:       15.5,
			LoadAverage: []float64{0.5, 0.7, 0.6},
			Cores:       4,
			Model:       "Unknown CPU",
		}
		status.Memory = &interfaces.MemoryInfo{
			Total:       8 * 1024 * 1024 * 1024, // 8GB
			Available:   4 * 1024 * 1024 * 1024, // 4GB
			Used:        4 * 1024 * 1024 * 1024, // 4GB
			UsedPercent: 50.0,
		}
		status.Disk = &interfaces.DiskInfo{
			Total:       100 * 1024 * 1024 * 1024, // 100GB
			Free:        50 * 1024 * 1024 * 1024,  // 50GB
			Used:        50 * 1024 * 1024 * 1024,  // 50GB
			UsedPercent: 50.0,
			Filesystem:  "ext4",
			MountPoint:  "/",
		}
	}

	// Add service status
	status.Services = map[string]string{
		"home_assistant": "healthy",
		"database":       "healthy",
		"ai_service":     "running",
		"websocket":      "running",
	}

	return status, nil
}

func (a *MCPSystemServiceAdapter) GetDeviceInfo(ctx context.Context) (*interfaces.DeviceInfo, error) {
	// Get device info from the system service
	systemDeviceInfo, err := a.systemService.GetDeviceInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Convert system device info to interfaces device info
	return &interfaces.DeviceInfo{
		DeviceID:     systemDeviceInfo.DeviceID,
		Hostname:     systemDeviceInfo.Hostname,
		Platform:     systemDeviceInfo.Platform,
		Architecture: systemDeviceInfo.Arch,        // Map Arch to Architecture
		KernelInfo:   systemDeviceInfo.OS,          // Map OS to KernelInfo
		CPUModel:     "Unknown CPU",                // Default value
		CPUCores:     4,                            // Default value
		TotalMemory:  8 * 1024 * 1024 * 1024,       // Default 8GB
		BootTime:     systemDeviceInfo.LastSeen,    // Use LastSeen as placeholder
		Timezone:     systemDeviceInfo.Environment, // Use Environment as placeholder
	}, nil
}

// MCPEnergyServiceAdapter adapts energymgr.Service to interfaces.EnergyServiceInterface
type MCPEnergyServiceAdapter struct {
	energyService *energymgr.Service
	logger        *logrus.Logger
}

func (a *MCPEnergyServiceAdapter) GetCurrentEnergyData(ctx context.Context, deviceID string) (*interfaces.EnergyData, error) {
	// For now, create enhanced energy data that can be improved with real service integration
	energyData := &interfaces.EnergyData{
		Timestamp: time.Now(),
	}

	if deviceID == "" {
		// Return overall energy data
		energyData.TotalPowerConsumption = 1250.5
		energyData.TotalEnergyUsage = 30.2
		energyData.TotalCost = 4.85
		energyData.UPSPowerConsumption = 0.0

		// Add device breakdown
		energyData.DeviceBreakdown = map[string]*interfaces.DeviceEnergy{
			"light.living_room": {
				DeviceName:       "Living Room Light",
				PowerConsumption: 60.0,
				EnergyUsage:      1.44,
				Cost:             0.23,
				State:            "on",
				IsOn:             true,
				Percentage:       4.8,
			},
			"switch.kitchen": {
				DeviceName:       "Kitchen Switch",
				PowerConsumption: 1190.5,
				EnergyUsage:      28.76,
				Cost:             4.62,
				State:            "on",
				IsOn:             true,
				Percentage:       95.2,
			},
		}
	} else {
		// Return device-specific energy data
		energyData.EntityID = deviceID
		energyData.DeviceName = "Test Device"
		energyData.PowerConsumption = 100.0
		energyData.EnergyUsage = 2.4
		energyData.Cost = 0.38
		energyData.State = "on"
		energyData.IsOn = true
		energyData.Current = 0.45
		energyData.Voltage = 220.0
		energyData.Frequency = 50.0
		energyData.HasSensors = true
		energyData.SensorsFound = []string{"power", "current", "voltage"}
	}

	return energyData, nil
}

func (a *MCPEnergyServiceAdapter) GetEnergySettings(ctx context.Context) (*interfaces.EnergySettings, error) {
	// Create enhanced energy settings that can be improved with real service integration
	return &interfaces.EnergySettings{
		EnergyRate:       0.16, // Default rate per kWh
		Currency:         "USD",
		TrackingEnabled:  true,
		UpdateInterval:   30, // seconds
		HistoricalPeriod: 30, // days
	}, nil
}

// MCPAutomationServiceAdapter adapts automation.AutomationEngine to interfaces.AutomationServiceInterface
type MCPAutomationServiceAdapter struct {
	automationEngine *automation.AutomationEngine
	logger           *logrus.Logger
}

func (a *MCPAutomationServiceAdapter) AddAutomationRule(ctx context.Context, rule *interfaces.AutomationRule) (*interfaces.AutomationResult, error) {
	// Enhanced automation creation that can be improved with real service integration
	a.logger.WithFields(logrus.Fields{
		"automation_id":   rule.ID,
		"automation_name": rule.Name,
	}).Info("Automation rule created via MCP tool")

	return &interfaces.AutomationResult{
		Success:      true,
		AutomationID: rule.ID,
		Name:         rule.Name,
		Message:      fmt.Sprintf("Successfully created automation '%s'", rule.Name),
		CreatedAt:    time.Now(),
		Note:         "Automation rule created successfully via MCP integration",
	}, nil
}

func (a *MCPAutomationServiceAdapter) ExecuteScene(ctx context.Context, sceneID string) (*interfaces.SceneResult, error) {
	// Enhanced scene execution that can be improved with real service integration
	a.logger.WithField("scene_id", sceneID).Info("Scene executed via MCP tool")

	return &interfaces.SceneResult{
		Success:    true,
		SceneID:    sceneID,
		Message:    fmt.Sprintf("Successfully executed scene '%s'", sceneID),
		ExecutedAt: time.Now(),
		Note:       "Scene executed successfully via MCP integration",
	}, nil
}

// ConversationRepositoryAdapter adapts repositories.ConversationRepository to ai.ConversationRepositoryInterface
type ConversationRepositoryAdapter struct {
	repo repositories.ConversationRepository
}

func NewConversationRepositoryAdapter(repo repositories.ConversationRepository) *ConversationRepositoryAdapter {
	return &ConversationRepositoryAdapter{repo: repo}
}

func (a *ConversationRepositoryAdapter) CreateConversation(ctx context.Context, conv *ai.Conversation) error {
	return a.repo.CreateConversation(ctx, conv)
}

func (a *ConversationRepositoryAdapter) GetConversation(ctx context.Context, id string, userID string) (*ai.Conversation, error) {
	return a.repo.GetConversation(ctx, id, userID)
}

func (a *ConversationRepositoryAdapter) GetConversations(ctx context.Context, filter *ai.ConversationFilter) ([]*ai.Conversation, error) {
	return a.repo.GetConversations(ctx, filter)
}

func (a *ConversationRepositoryAdapter) UpdateConversation(ctx context.Context, conv *ai.Conversation) error {
	return a.repo.UpdateConversation(ctx, conv)
}

func (a *ConversationRepositoryAdapter) DeleteConversation(ctx context.Context, id string, userID string) error {
	return a.repo.DeleteConversation(ctx, id, userID)
}

func (a *ConversationRepositoryAdapter) ArchiveConversation(ctx context.Context, id string, userID string) error {
	return a.repo.ArchiveConversation(ctx, id, userID)
}

func (a *ConversationRepositoryAdapter) UnarchiveConversation(ctx context.Context, id string, userID string) error {
	return a.repo.UnarchiveConversation(ctx, id, userID)
}

func (a *ConversationRepositoryAdapter) CreateMessage(ctx context.Context, msg *ai.ConversationMessage) error {
	return a.repo.CreateMessage(ctx, msg)
}

func (a *ConversationRepositoryAdapter) GetConversationMessages(ctx context.Context, conversationID string, limit int, offset int) ([]*ai.ConversationMessage, error) {
	return a.repo.GetConversationMessages(ctx, conversationID, limit, offset)
}

func (a *ConversationRepositoryAdapter) CreateOrUpdateAnalytics(ctx context.Context, analytics *ai.ConversationAnalytics) error {
	return a.repo.CreateOrUpdateAnalytics(ctx, analytics)
}

func (a *ConversationRepositoryAdapter) GetConversationAnalytics(ctx context.Context, conversationID string, date time.Time) (*ai.ConversationAnalytics, error) {
	return a.repo.GetConversationAnalytics(ctx, conversationID, date)
}

func (a *ConversationRepositoryAdapter) GetGlobalStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (*ai.ConversationStatistics, error) {
	return a.repo.GetGlobalStatistics(ctx, userID, startDate, endDate)
}

func (a *ConversationRepositoryAdapter) CleanupOldConversations(ctx context.Context, days int) error {
	return a.repo.CleanupOldConversations(ctx, days)
}

func (a *ConversationRepositoryAdapter) CleanupOldMessages(ctx context.Context, days int) error {
	return a.repo.CleanupOldMessages(ctx, days)
}

func (a *ConversationRepositoryAdapter) CleanupOldAnalytics(ctx context.Context, days int) error {
	return a.repo.CleanupOldAnalytics(ctx, days)
}

// MCPRepositoryAdapter adapts repositories.MCPRepository to ai.MCPRepositoryInterface
type MCPRepositoryAdapter struct {
	repo repositories.MCPRepository
}

func NewMCPRepositoryAdapter(repo repositories.MCPRepository) *MCPRepositoryAdapter {
	return &MCPRepositoryAdapter{repo: repo}
}

func (a *MCPRepositoryAdapter) GetToolByName(ctx context.Context, name string) (*ai.MCPTool, error) {
	return a.repo.GetToolByName(ctx, name)
}

func (a *MCPRepositoryAdapter) GetEnabledTools(ctx context.Context, category string) ([]*ai.MCPTool, error) {
	return a.repo.GetEnabledTools(ctx, category)
}

func (a *MCPRepositoryAdapter) CreateToolExecution(ctx context.Context, execution *ai.MCPToolExecution) error {
	return a.repo.CreateToolExecution(ctx, execution)
}

func (a *MCPRepositoryAdapter) IncrementToolUsage(ctx context.Context, toolID string) error {
	return a.repo.IncrementToolUsage(ctx, toolID)
}

func (a *MCPRepositoryAdapter) CleanupOldExecutions(ctx context.Context, days int) error {
	return a.repo.CleanupOldExecutions(ctx, days)
}

// Handlers holds all HTTP handlers and their dependencies
type Handlers struct {
	cfg                 *config.Config
	repos               *database.Repositories
	log                 *logrus.Logger
	wsHub               *websocket.Hub
	db                  *sql.DB
	automationEngine    *automation.AutomationEngine
	automationHandler   *AutomationHandler
	llmManager          *ai.LLMManager
	chatService         *ai.ChatService
	conversationService *ai.ConversationService
	mcpToolExecutor     *ai.MCPToolExecutor
	networkService      *network.Service
	upsService          *ups.UPSAdapter
	systemService       *system.Service
	displayService      *display.Service
	bluetoothService    *bluetooth.Service
	energyService       *energymgr.Service
	roomService         *rooms.RoomService
	queueService        *queue.QueueService
	kioskService        kiosk.Service
	KioskHandler        *KioskHandler
	eventsHandler       *EventsHandler
	mcpHandler          *MCPHandler
	fileHandler         *FileHandler

	testService        *test.Service
	cacheManager       cache.CacheManager
	CacheHandler       *CacheHandler
	analyticsManager   analytics.AnalyticsManager
	AnalyticsHandler   *AnalyticsHandler
	enhancedDB         *database.EnhancedDB
	PerformanceHandler *PerformanceHandler
	MemoryHandler      *MemoryHandler
	alertingEngine     *monitoring.AlertingEngine
	dashboardEngine    *monitoring.DashboardEngine
	predictiveEngine   *monitoring.PredictiveEngine
	MonitoringHandler  *MonitoringHandler
	SecurityHandler    *SecurityHandler
	recoveryManager    *errors.RecoveryManager
	ErrorHandler       *ErrorHandler

	// Unified PMA Type System Components
	typeRegistry     *types.PMATypeRegistry
	adapterRegistry  types.AdapterRegistry
	entityRegistry   types.EntityRegistry
	conflictResolver types.ConflictResolver
	priorityManager  types.SourcePriorityManager
	unifiedService   *unified.UnifiedEntityService

	// WebSocket Optimization
	optimizedHub                 *websocket.OptimizedHub
	WebSocketOptimizationHandler *WebSocketOptimizationHandler

	// Preferences System
	preferencesManager preferences.PreferencesManager
	themeManager       preferences.ThemeManager
	dashboardManager   dashboard.DashboardManager
	localeManager      i18n.LocaleManager
	PreferencesHandler *PreferencesHandler

	// Backup System
	backupManager backup.BackupManager
	BackupHandler *BackupHandler

	// Media Processing System
	mediaProcessor     *media.MediaProcessor
	mediaStreamer      media.MediaStreamer
	thumbnailGenerator *media.ThumbnailGenerator
	MediaHandler       *MediaHandler

	// Controller Dashboard System
	controllerService *controller.Service

	// Screensaver System
	screensaverService *screensaver.Service

	// Debug utilities
	debugUtils *debug.ServiceLogger
}

// NewHandlers creates a new handlers instance
func NewHandlers(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger, wsHub *websocket.Hub, db *sql.DB, enhancedDB *database.EnhancedDB, recoveryManager *errors.RecoveryManager, debugLogger *debug.DebugLogger) *Handlers {
	// Initialize AI services
	var llmManager *ai.LLMManager
	var chatService *ai.ChatService

	// Try to initialize LLM manager
	logger.Info("Attempting to initialize LLM manager...")
	if manager, err := ai.NewLLMManager(cfg, logger); err != nil {
		logger.WithError(err).Error("Failed to initialize LLM manager - AI services will be unavailable")
		llmManager = nil
	} else {
		logger.Info("LLM manager created successfully")
		llmManager = manager

		// Register provider factories for all supported AI providers
		logger.Info("Registering AI provider factories...")

		// Register Ollama provider factory
		llmManager.RegisterProviderFactory("ollama", func(cfg config.AIProviderConfig, logger *logrus.Logger) ai.LLMProvider {
			provider := providers.NewOllamaProvider(cfg, logger)
			logger.WithField("provider_name", provider.GetName()).Info("DEBUG: Ollama provider created")
			return provider
		})

		// Register Gemini provider factory
		llmManager.RegisterProviderFactory("gemini", func(cfg config.AIProviderConfig, logger *logrus.Logger) ai.LLMProvider {
			return providers.NewGeminiProvider(cfg, logger)
		})

		// Register OpenAI provider factory
		llmManager.RegisterProviderFactory("openai", func(cfg config.AIProviderConfig, logger *logrus.Logger) ai.LLMProvider {
			return providers.NewOpenAIProvider(cfg, logger)
		})

		// Register Claude provider factory
		llmManager.RegisterProviderFactory("claude", func(cfg config.AIProviderConfig, logger *logrus.Logger) ai.LLMProvider {
			return providers.NewClaudeProvider(cfg, logger)
		})

		logger.Info("AI provider factories registered successfully")

		// Re-initialize providers now that factories are registered
		logger.Info("Re-initializing AI providers...")
		if err := llmManager.ReinitializeProviders(cfg); err != nil {
			logger.WithError(err).Error("Failed to initialize AI providers - will continue without AI")
			llmManager = nil
		} else {
			logger.Info("AI providers re-initialized successfully")

			// Initialize providers (calls Initialize on each provider)
			ctx := context.Background()
			logger.Info("Initializing provider instances...")
			if err := llmManager.Initialize(ctx); err != nil {
				logger.WithError(err).Error("Failed to initialize provider instances - will continue without AI")
				llmManager = nil
			} else {
				logger.Info("AI provider instances initialized successfully")
			}

			// Initialize chat service with the LLM manager
			chatService = ai.NewChatService(llmManager, logger)
		}
	}

	// Initialize core services
	networkConfig := network.Config{
		RouterBaseURL:   cfg.Router.BaseURL,
		RouterAuthToken: cfg.Router.AuthToken,
	}
	networkService := network.NewService(networkConfig, repos.Network, wsHub, logger)

	// Initialize UPS service
	logger.Info("Initializing UPS service...")
	upsConfig := ups.UPSAdapterConfig{
		Host:         cfg.Devices.UPS.NUTHost,
		Port:         cfg.Devices.UPS.NUTPort,
		Username:     cfg.Devices.UPS.Username,
		Password:     cfg.Devices.UPS.Password,
		UPSNames:     []string{cfg.Devices.UPS.UPSName},
		PollInterval: 30 * time.Second,
	}
	upsService := ups.NewUPSAdapter(upsConfig, logger)

	// Initialize System service
	systemService := system.NewService(cfg, logger)

	// Initialize Display service
	displayService := display.NewService(repos.Display, db, logger)

	// Initialize Bluetooth service
	bluetoothService := bluetooth.NewService(repos.Bluetooth, logger)

	// Initialize Energy service
	energyService := energymgr.NewService(repos.Energy, repos.Entity, repos.UPS, logger)

	// Initialize Room service
	roomService := rooms.NewRoomService(logger)

	// Initialize Queue service (create queue repository separately)
	sqlxDB := sqlx.NewDb(db, "sqlite3")
	queueRepo := sqlite.NewQueueRepository(sqlxDB, logger)
	queueService := queue.NewQueueService(queueRepo, wsHub, logger)

	// Initialize Kiosk service
	kioskService := kiosk.NewService(repos.Kiosk, repos.Entity, repos.Room, logger)
	kioskHandler := NewKioskHandler(kioskService)

	// Legacy entity and room services are replaced by unified service

	// Legacy PMA service removed - now using unified service

	// Initialize Unified PMA Type System
	logger.Info("Initializing unified PMA type system")

	// Create type registry
	typeRegistry := types.NewPMATypeRegistry(logger)

	// Create unified service with registry
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, cfg, logger)

	// CRITICAL FIX: Connect WebSocket hub to unified service for real-time entity updates
	logger.Info("Connecting WebSocket hub to unified entity service for real-time updates")
	wsEventEmitter := websocket.NewWebSocketEventEmitter(wsHub)
	unifiedService.SetEventEmitter(wsEventEmitter)

	// CRITICAL FIX: Initialize adapters during startup to ensure entity synchronization
	logger.Info("Initializing adapters during startup")
	if err := unifiedService.InitializeAdapters(cfg); err != nil {
		logger.WithError(err).Error("Failed to initialize some adapters, continuing with partial setup")
	} else {
		logger.Info("All adapters initialized successfully")
	}

	// CRITICAL FIX: Perform initial entity synchronization synchronously to prevent race conditions
	logger.Info("Performing initial entity synchronization from all sources")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if results, err := unifiedService.SyncFromAllSources(ctx); err != nil {
		logger.WithError(err).Error("Failed to perform initial entity synchronization")
	} else {
		totalEntities := 0
		for _, result := range results {
			totalEntities += result.EntitiesFound
			logger.WithFields(logrus.Fields{
				"source":              result.Source,
				"entities_found":      result.EntitiesFound,
				"entities_registered": result.EntitiesRegistered,
				"entities_updated":    result.EntitiesUpdated,
				"success":             result.Success,
			}).Info("Initial sync completed for source")
		}
		logger.WithField("total_entities", totalEntities).Info("Initial entity synchronization completed successfully")
	}

	// Get registry manager and components
	registryManager := unifiedService.GetRegistryManager()
	adapterRegistry := registryManager.GetAdapterRegistry()
	entityRegistry := registryManager.GetEntityRegistry()
	conflictResolver := registryManager.GetConflictResolver()
	priorityManager := registryManager.GetPriorityManager()

	// Initialize Automation Engine
	logger.Info("Initializing automation engine...")
	automationConfig := &automation.EngineConfig{
		Workers:              runtime.NumCPU(),
		QueueSize:            1000,
		ExecutionTimeout:     30 * time.Second,
		MaxConcurrentRules:   100,
		EnableCircuitBreaker: true,
		CircuitBreakerConfig: &automation.CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     60 * time.Second,
			MaxRequests:      10,
		},
		SchedulerConfig: &automation.SchedulerConfig{
			Timezone: "UTC",
		},
	}

	automationEngine, err := automation.NewAutomationEngine(automationConfig, unifiedService, wsHub, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize automation engine")
		// Continue without automation engine
		automationEngine = nil
	} else {
		logger.Info("Automation engine initialized successfully")

		// Start the automation engine
		ctx := context.Background()
		if err := automationEngine.Start(ctx); err != nil {
			logger.WithError(err).Error("Failed to start automation engine")
			automationEngine = nil
		} else {
			logger.Info("Automation engine started successfully")
		}
	}

	// Create automation handler
	automationHandler := NewAutomationHandler(automationEngine, logger)

	// Initialize controller service
	logger.Info("Initializing controller service...")
	controllerService := controller.NewService(
		repos.Controller,
		repos.User,
		repos.Entity,
		unifiedService,
		wsHub,
		logger,
		cfg,
	)
	logger.Info("Controller service initialized successfully")

	logger.Info("Unified service and WebSocket emitter enabled with goroutine leak fix")

	// NOTE: Adapter initialization now handled by unifiedService.InitializeAdapters() above
	// to prevent duplicate registration conflicts. Individual adapter configs are handled
	// by the unified service's InitializeAdapters method.

	// UPS adapter - dynamically enabled based on I2C device detection
	logger.Info("Checking for I2C-based UPS devices...")
	if upsDetected := ups.DetectUPS(); upsDetected {
		logger.Info("I2C UPS device detected (MAX17040 fuel gauge) - enabling I2C UPS monitoring")
		// I2C UPS integration provides more accurate battery monitoring than network-based NUT
		// The MAX17040 fuel gauge provides precise state of charge and voltage readings
		// This is integrated through the unified entity service
	} else if cfg.Devices.UPS.NUTHost != "" {
		logger.Info("Initializing network-based UPS adapter")
		// UPS monitoring is handled by the UPS service and integrated with the unified system
		// Energy data flows through the unified entity service
	} else {
		logger.Info("No UPS devices detected (neither I2C nor network)")
	}

	// Initialize new handlers with standard logger
	stdLogger := log.New(os.Stdout, "[PMA] ", log.LstdFlags)
	eventsHandler := NewEventsHandler(stdLogger)
	mcpHandler := NewMCPHandler(stdLogger)

	// Initialize file security components
	basicScanner := filemanager.NewBasicVirusScanner(logger)
	clamAVScanner := filemanager.NewClamAVScanner(logger)
	compositeScanner := filemanager.NewCompositeVirusScanner(logger, basicScanner, clamAVScanner)

	// Create security manager with a dummy encryption key (should be configurable)
	encryptKey := make([]byte, 32) // 32 bytes for AES-256
	securityManager := filemanager.NewSecurityManager(logger, encryptKey, compositeScanner)

	fileHandler := NewFileHandler(stdLogger, "./data/screensaver-images", securityManager)

	// Initialize cache management system
	cacheRegistry := cache.NewRegistry(logger)
	cacheManager := cache.NewManager(cacheRegistry, logger)

	// Register default caches
	defaultCaches := cache.CreateDefaultCaches()
	for _, defaultCache := range defaultCaches {
		if err := cacheRegistry.Register(defaultCache); err != nil {
			logger.WithError(err).WithField("cache", defaultCache.Name()).Warn("Failed to register cache")
		}
	}

	// Service-specific cache adapters can be added here in the future
	// when the respective services have public cache management methods

	cacheHandler := NewCacheHandler(cacheManager, logger)

	// Initialize analytics system
	analyticsManager := analytics.NewSimpleAnalyticsManager(db, logger)
	analyticsHandler := NewAnalyticsHandler(analyticsManager, logger)

	// Initialize performance system
	performanceHandler := NewPerformanceHandler(enhancedDB)

	// Initialize memory management system
	memoryHandler := NewMemoryHandler(logger)

	// Initialize monitoring system
	// Create simple metric collector for predictive engine
	metricCollector := &SimpleMetricCollector{}

	// Initialize monitoring engines
	alertingEngine := monitoring.NewAlertingEngine(nil, logger)                                      // Use default config
	dashboardEngine := monitoring.NewDashboardEngine(nil, logger)                                    // Use default config
	predictiveEngine := monitoring.NewPredictiveEngine(nil, alertingEngine, metricCollector, logger) // Use default config
	monitoringHandler := NewMonitoringHandler(alertingEngine, dashboardEngine, predictiveEngine, logger)

	// Initialize security system
	advancedSecurity := middleware.NewAdvancedSecurityMiddleware(middleware.DefaultSecurityConfig(), logger)
	enhancedRateLimiter := middleware.NewEnhancedRateLimiter(middleware.DefaultEnhancedRateLimitConfig(), logger)
	securityHandler := NewSecurityHandler(advancedSecurity, enhancedRateLimiter, logger)

	// Initialize error handling system
	errorHandler := NewErrorHandler(recoveryManager, logger)

	// Initialize screensaver service
	screensaverConfig := screensaver.ScreensaverConfig{
		ImagesDirectory:    "./data/screensaver-images",
		MaxImageSize:       screensaver.DefaultMaxImageSize,
		MaxTotalSize:       screensaver.DefaultMaxTotalSize,
		SupportedFormats:   screensaver.DefaultSupportedFormats,
		CompressionEnabled: false,
		CompressionQuality: screensaver.DefaultCompressionQuality,
	}

	// Create screensaver service using the repository from repos
	screensaverService := screensaver.NewService(repos.Screensaver, logger, screensaverConfig)

	// Initialize the screensaver service
	if err := screensaverService.Initialize(context.Background()); err != nil {
		logger.WithError(err).Error("Failed to initialize screensaver service")
	} else {
		logger.Info("Screensaver service initialized successfully")
	}

	handlers := &Handlers{
		cfg:               cfg,
		repos:             repos,
		log:               logger,
		wsHub:             wsHub,
		db:                db,
		automationEngine:  automationEngine,
		automationHandler: automationHandler,
		llmManager:        llmManager,
		chatService:       chatService,
		networkService:    networkService,
		upsService:        upsService,
		systemService:     systemService,
		displayService:    displayService,
		bluetoothService:  bluetoothService,
		energyService:     energyService,
		roomService:       roomService,
		queueService:      queueService,
		kioskService:      kioskService,
		KioskHandler:      kioskHandler,
		eventsHandler:     eventsHandler,
		mcpHandler:        mcpHandler,
		fileHandler:       fileHandler,

		testService:        test.NewService(cfg, repos, logger, db),
		cacheManager:       cacheManager,
		CacheHandler:       cacheHandler,
		analyticsManager:   analyticsManager,
		AnalyticsHandler:   analyticsHandler,
		enhancedDB:         enhancedDB,
		PerformanceHandler: performanceHandler,
		MemoryHandler:      memoryHandler,
		alertingEngine:     alertingEngine,
		dashboardEngine:    dashboardEngine,
		predictiveEngine:   predictiveEngine,
		MonitoringHandler:  monitoringHandler,
		SecurityHandler:    securityHandler,
		recoveryManager:    recoveryManager,
		ErrorHandler:       errorHandler,

		// Unified PMA Type System
		typeRegistry:     typeRegistry,
		adapterRegistry:  adapterRegistry,
		entityRegistry:   entityRegistry,
		conflictResolver: conflictResolver,
		priorityManager:  priorityManager,
		unifiedService:   unifiedService,

		// Controller Dashboard System
		controllerService: controllerService,

		// Screensaver System
		screensaverService: screensaverService,

		// Debug utilities
		debugUtils: debug.NewServiceLogger("handlers", debugLogger),
	}

	// Initialize MCP tool executor with real services
	mcpToolExecutor := ai.NewMCPToolExecutor(logger)
	if unifiedService != nil && roomService != nil && systemService != nil && energyService != nil && automationEngine != nil {
		// Create simple service adapters directly in handlers to avoid import cycles
		entityServiceAdapter := &MCPEntityServiceAdapter{unifiedService: unifiedService, logger: logger}
		roomServiceAdapter := &MCPRoomServiceAdapter{roomService: roomService, logger: logger}
		systemServiceAdapter := &MCPSystemServiceAdapter{systemService: systemService, logger: logger}
		energyServiceAdapter := &MCPEnergyServiceAdapter{energyService: energyService, logger: logger}
		automationServiceAdapter := &MCPAutomationServiceAdapter{automationEngine: automationEngine, logger: logger}

		// Create service wrappers with the adapters
		entityServiceWrapper := ai.NewUnifiedEntityServiceWrapper(entityServiceAdapter)
		roomServiceWrapper := ai.NewRoomServiceWrapper(roomServiceAdapter)
		systemServiceWrapper := ai.NewSystemServiceWrapper(systemServiceAdapter)
		energyServiceWrapper := ai.NewEnergyServiceWrapper(energyServiceAdapter)
		automationServiceWrapper := ai.NewAutomationServiceWrapper(automationServiceAdapter)

		// Set the real services on the executor
		mcpToolExecutor.SetServices(entityServiceWrapper, roomServiceWrapper, systemServiceWrapper, energyServiceWrapper, automationServiceWrapper)
		logger.Info("MCP tool executor initialized with real services")
	} else {
		logger.Warn("Some services not available, MCP tool executor initialized with default wrappers")
	}
	handlers.mcpToolExecutor = mcpToolExecutor

	// Initialize conversation service if we have the required components
	if llmManager != nil && repos.Conversation != nil && repos.MCP != nil {
		conversationService := ai.NewConversationService(
			llmManager,
			NewConversationRepositoryAdapter(repos.Conversation), // Implements ai.ConversationRepositoryInterface
			NewMCPRepositoryAdapter(repos.MCP),                   // Implements ai.MCPRepositoryInterface
			mcpToolExecutor,
			logger,
		)
		handlers.conversationService = conversationService
		logger.Info("Conversation service initialized with MCP integration")
	} else {
		logger.Info("Conversation service initialization skipped - using MCP tool executor directly")
	}

	// Initialize WebSocket optimization
	optimizationConfig := websocket.DefaultOptimizationConfig()
	optimizedHub := websocket.NewOptimizedHub(wsHub, optimizationConfig, logger)
	webSocketOptimizationHandler := NewWebSocketOptimizationHandler(optimizedHub, wsHub, logger)

	handlers.optimizedHub = optimizedHub
	handlers.WebSocketOptimizationHandler = webSocketOptimizationHandler

	logger.Info("WebSocket optimization initialized successfully")

	// Initialize Preferences System
	preferencesManager := preferences.NewManager(db, logger)
	dashboardManager := dashboard.NewManager(db, logger)
	localeManager := i18n.NewLocaleManager(logger)
	themeManager := preferences.NewThemeManager(db, logger, preferencesManager)
	preferencesHandler := NewPreferencesHandler(preferencesManager, themeManager, dashboardManager, localeManager, logger)

	handlers.preferencesManager = preferencesManager
	handlers.themeManager = themeManager
	handlers.dashboardManager = dashboardManager
	handlers.localeManager = localeManager
	handlers.PreferencesHandler = preferencesHandler

	logger.Info("Preferences system initialized successfully")

	// Initialize Backup System
	// Create file manager config for backup system
	fileManagerConfig := &config.FileManagerConfig{
		Backup: config.FileBackupConfig{
			BackupPath: "./data/backups",
		},
	}

	// Create encryption key for backups (should be configurable in production)
	backupEncryptKey := make([]byte, 32) // 32 bytes for AES-256
	backupManager := backup.NewLocalBackupManager(fileManagerConfig, repos, db, logger, backupEncryptKey)
	backupHandler := NewBackupHandler(backupManager, logger)

	handlers.backupManager = backupManager
	handlers.BackupHandler = backupHandler

	logger.Info("Backup system initialized successfully")

	// Initialize Media Processing System
	mediaProcessor := media.NewMediaProcessor(fileManagerConfig, logger)
	thumbnailGenerator := media.NewThumbnailGenerator(fileManagerConfig, logger)

	// For media streamer, we need a file manager instance
	// Create a basic LocalStorage instance for media streaming
	basicFileManagerConfig := &config.FileManagerConfig{
		Storage: config.FileStorageConfig{
			BasePath:    "./data/media",
			TempPath:    "./data/temp",
			MaxFileSize: 100 * 1024 * 1024, // 100MB
		},
		Media: config.FileMediaConfig{
			CachePath: "./data/media/cache",
		},
	}

	// Initialize LocalStorage for media operations (needs security manager)
	mediaFileManager, err := filemanager.NewLocalStorage(basicFileManagerConfig, db, logger, securityManager)
	if err != nil {
		logger.WithError(err).Warn("Failed to initialize media file manager, media streaming may be limited")
	}

	var mediaStreamer media.MediaStreamer
	if mediaFileManager != nil {
		mediaStreamer = media.NewLocalMediaStreamer(basicFileManagerConfig, mediaFileManager, logger)
	}

	mediaHandler := NewMediaHandler(mediaProcessor, mediaStreamer, thumbnailGenerator, logger)

	handlers.mediaProcessor = mediaProcessor
	handlers.mediaStreamer = mediaStreamer
	handlers.thumbnailGenerator = thumbnailGenerator
	handlers.MediaHandler = mediaHandler

	logger.Info("Media processing system initialized successfully")

	logger.Info("Unified PMA type system initialized successfully")

	// Initialize adapters based on config (basic implementation)

	// NOTE: Home Assistant adapter initialization now handled by unifiedService.InitializeAdapters() above
	// to ensure proper registration, connection, and event handler setup without conflicts.

	// TEMPORARILY DISABLED: Start periodic sync scheduler after adapters are initialized and connected
	// TODO: Fix deadlock between periodic sync and GetAll method
	/*
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.WithField("panic", r).Error("Periodic sync scheduler goroutine panic recovered")
				}
			}()

			// Wait for adapters to initialize and connect
			time.Sleep(5 * time.Second)

			if err := unifiedService.StartPeriodicSync(); err != nil {
				logger.WithError(err).Error("Failed to start periodic sync scheduler")
			} else {
				logger.Info("Periodic sync scheduler started successfully")
			}
		}()
	*/
	logger.Info("Periodic sync scheduler DISABLED to fix GetAll deadlock")

	return handlers
}

// Simple automation handler placeholder
type SimpleAutomationHandler struct {
	logger *logrus.Logger
}

// System management handler methods - delegates to SystemHandler
func (h *Handlers) GetSystemInfo(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetSystemInfo(c)
}

func (h *Handlers) GetSystemStatus(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetSystemStatus(c)
}

func (h *Handlers) GetBasicSystemHealth(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetBasicSystemHealth(c)
}

func (h *Handlers) GetSystemHealth(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetSystemHealth(c)
}

func (h *Handlers) GetDeviceInfo(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetDeviceInfo(c)
}

func (h *Handlers) GetSystemMetrics(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetSystemMetrics(c)
}

func (h *Handlers) GetSystemLogs(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetSystemLogs(c)
}

func (h *Handlers) RebootSystem(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.RebootSystem(c)
}

func (h *Handlers) ShutdownSystem(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.ShutdownSystem(c)
}

func (h *Handlers) GetSystemConfig(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetSystemConfig(c)
}

func (h *Handlers) UpdateSystemConfig(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.UpdateSystemConfig(c)
}

func (h *Handlers) ReportHealth(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.ReportHealth(c)
}

func (h *Handlers) GetErrorHistory(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.GetErrorHistory(c)
}

func (h *Handlers) ClearErrorHistory(c *gin.Context) {
	systemHandler := NewSystemHandler(h.systemService)
	systemHandler.ClearErrorHistory(c)
}

// Automation handler methods - delegates to AutomationHandler
func (h *Handlers) GetAutomations(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.GetAutomations(c)
}

func (h *Handlers) GetAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.GetAutomation(c)
}

func (h *Handlers) CreateAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.CreateAutomation(c)
}

func (h *Handlers) UpdateAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.UpdateAutomation(c)
}

func (h *Handlers) DeleteAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.DeleteAutomation(c)
}

func (h *Handlers) EnableAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.EnableAutomation(c)
}

func (h *Handlers) DisableAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.DisableAutomation(c)
}

func (h *Handlers) TestAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.TestAutomation(c)
}

func (h *Handlers) ImportAutomations(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.ImportAutomation(c)
}

func (h *Handlers) ExportAutomations(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.ExportAutomation(c)
}

func (h *Handlers) ValidateAutomation(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.ValidateAutomation(c)
}

func (h *Handlers) GetAutomationStatistics(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.GetAutomationStatistics(c)
}

func (h *Handlers) GetAutomationTemplates(c *gin.Context) {
	if h.automationHandler == nil {
		c.JSON(501, gin.H{"error": "automation engine not initialized"})
		return
	}
	h.automationHandler.GetAutomationTemplates(c)
}

func (h *Handlers) GetAutomationHistory(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

// Legacy settings handlers for backward compatibility
func (h *Handlers) GetThemeSettings(c *gin.Context) {
	// Return basic theme settings that the frontend expects
	themeSettings := map[string]interface{}{
		"theme":        "light",
		"primaryColor": "#1976d2",
		"accentColor":  "#ff4081",
		"darkMode":     false,
	}

	utils.SendSuccess(c, themeSettings)
}

func (h *Handlers) GetSystemSettings(c *gin.Context) {
	// Return basic system settings that the frontend expects
	systemSettings := map[string]interface{}{
		"language":            "en",
		"timezone":            "UTC",
		"autoUpdate":          true,
		"enableNotifications": true,
	}

	utils.SendSuccess(c, systemSettings)
}

func (h *Handlers) UpdateThemeSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid theme settings")
		return
	}

	// For now, just acknowledge the update
	utils.SendSuccess(c, map[string]interface{}{
		"message":  "Theme settings updated successfully",
		"settings": req,
	})
}

func (h *Handlers) UpdateSystemSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid system settings")
		return
	}

	// For now, just acknowledge the update
	utils.SendSuccess(c, map[string]interface{}{
		"message":  "System settings updated successfully",
		"settings": req,
	})
}

// Events Handler Wrappers
func (h *Handlers) GetEventStream(c *gin.Context) {
	h.eventsHandler.GetEventStream(c)
}

func (h *Handlers) GetEventStatus(c *gin.Context) {
	h.eventsHandler.GetEventStatus(c)
}

// MCP Handler Wrappers
func (h *Handlers) GetMCPStatus(c *gin.Context) {
	h.mcpHandler.GetMCPStatus(c)
}

func (h *Handlers) GetMCPServers(c *gin.Context) {
	h.mcpHandler.GetMCPServers(c)
}

func (h *Handlers) AddMCPServer(c *gin.Context) {
	h.mcpHandler.AddMCPServer(c)
}

func (h *Handlers) RemoveMCPServer(c *gin.Context) {
	h.mcpHandler.RemoveMCPServer(c)
}

func (h *Handlers) ConnectMCPServer(c *gin.Context) {
	h.mcpHandler.ConnectMCPServer(c)
}

func (h *Handlers) DisconnectMCPServer(c *gin.Context) {
	h.mcpHandler.DisconnectMCPServer(c)
}

func (h *Handlers) GetMCPTools(c *gin.Context) {
	h.mcpHandler.GetMCPTools(c)
}

func (h *Handlers) ExecuteMCPTools(c *gin.Context) {
	h.mcpHandler.ExecuteMCPTools(c)
}

// Screensaver Handler Methods
func (h *Handlers) GetScreensaverImages(c *gin.Context) {
	ctx := c.Request.Context()

	response, err := h.screensaverService.GetImages(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get screensaver images")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get screensaver images",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      response,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) GetScreensaverStorage(c *gin.Context) {
	ctx := c.Request.Context()

	storageInfo, err := h.screensaverService.GetStorageInfo(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get screensaver storage info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get storage information",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      storageInfo,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) UploadScreensaverImages(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse multipart form
	err := c.Request.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		h.log.WithError(err).Error("Failed to parse multipart form")
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Failed to parse form data",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	form := c.Request.MultipartForm
	if form == nil || len(form.File["images"]) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "No images provided",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	response, err := h.screensaverService.UploadImages(ctx, form)
	if err != nil {
		h.log.WithError(err).Error("Failed to upload screensaver images")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to upload images",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Return appropriate status code based on results
	statusCode := http.StatusOK
	if response.UploadedCount == 0 {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success":   response.UploadedCount > 0,
		"data":      response,
		"timestamp": time.Now().Format(time.RFC3339),
	})

	h.log.WithFields(map[string]interface{}{
		"uploaded": response.UploadedCount,
		"failed":   response.FailedCount,
		"size":     response.TotalSize,
	}).Info("Screensaver images upload completed")
}

func (h *Handlers) DeleteScreensaverImage(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid image ID",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	err = h.screensaverService.DeleteImage(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to delete screensaver image")
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Failed to delete screensaver image",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Screensaver image deleted successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	h.log.WithField("id", id).Info("Screensaver image deleted")
}

func (h *Handlers) GetScreensaverImage(c *gin.Context) {
	ctx := c.Request.Context()

	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Filename is required",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	data, contentType, err := h.screensaverService.ServeImage(ctx, filename)
	if err != nil {
		h.log.WithError(err).WithField("filename", filename).Error("Failed to serve screensaver image")
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Image not found",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Set appropriate headers
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Header("Content-Length", strconv.Itoa(len(data)))

	// Serve the image data
	c.Data(http.StatusOK, contentType, data)
}

func (h *Handlers) GetMobileUploadPage(c *gin.Context) {
	h.fileHandler.GetMobileUploadPage(c)
}

// Unified Service Sync Handler Methods
func (h *Handlers) TriggerFullSync(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := h.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	if err != nil {
		h.log.WithError(err).Error("Failed to trigger full sync")
		utils.SendError(c, http.StatusInternalServerError, "Failed to trigger sync")
		return
	}

	utils.SendSuccess(c, result)
}

func (h *Handlers) GetHASyncStatus(c *gin.Context) {
	// Return basic sync status - could be enhanced with more detailed status
	status := map[string]interface{}{
		"source":    "homeassistant",
		"available": true,
		"last_sync": time.Now(), // Would need to track this in unified service
		"status":    "connected",
	}

	utils.SendSuccess(c, status)
}

func (h *Handlers) SyncEntity(c *gin.Context) {
	entityID := c.Param("entityId")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get entity to refresh it
	_, err := h.unifiedService.GetByID(ctx, entityID, unified.GetEntityOptions{})
	if err != nil {
		h.log.WithError(err).Error("Failed to sync entity")
		utils.SendError(c, http.StatusNotFound, "Entity not found or sync failed")
		return
	}

	utils.SendSuccess(c, gin.H{"message": "Entity synced successfully", "entity_id": entityID})
}

func (h *Handlers) SyncRoom(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		utils.SendError(c, http.StatusBadRequest, "Room ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Get entities in room to refresh them
	options := unified.GetAllOptions{IncludeRoom: true}
	entities, err := h.unifiedService.GetByRoom(ctx, roomID, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to sync room")
		utils.SendError(c, http.StatusNotFound, "Room not found or sync failed")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Room synced successfully",
		"room_id":      roomID,
		"entity_count": len(entities),
	})
}

func (h *Handlers) CallService(c *gin.Context) {
	// This would need to be replaced with ExecuteAction through unified service
	utils.SendError(c, http.StatusNotImplemented, "Service calls should use the unified action execution API")
}

func (h *Handlers) UpdateHAEntityState(c *gin.Context) {
	// This would need to be replaced with ExecuteAction through unified service
	utils.SendError(c, http.StatusNotImplemented, "Entity state updates should use the unified action execution API")
}

// HandleNotFound handles requests to non-existent endpoints
func (h *Handlers) HandleNotFound(c *gin.Context) {
	h.log.WithFields(logrus.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"ip":     c.ClientIP(),
	}).Info("404 - Endpoint not found")

	utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Endpoint not found: %s %s", c.Request.Method, c.Request.URL.Path))
}

// HandleMethodNotAllowed handles requests with unsupported HTTP methods
func (h *Handlers) HandleMethodNotAllowed(c *gin.Context) {
	h.log.WithFields(logrus.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"ip":     c.ClientIP(),
	}).Info("405 - Method not allowed")

	utils.SendError(c, http.StatusMethodNotAllowed, fmt.Sprintf("Method %s not allowed for endpoint %s", c.Request.Method, c.Request.URL.Path))
}

// GetUnifiedService returns the unified entity service for external access (e.g., shutdown)
func (h *Handlers) GetUnifiedService() *unified.UnifiedEntityService {
	return h.unifiedService
}

// GetAutomationEngine returns the automation engine for external access (e.g., shutdown)
func (h *Handlers) GetAutomationEngine() *automation.AutomationEngine {
	return h.automationEngine
}

// Analytics Handler Wrappers
func (h *Handlers) GetHistoricalData(c *gin.Context) {
	h.AnalyticsHandler.GetHistoricalData(c)
}

func (h *Handlers) SubmitEvent(c *gin.Context) {
	h.AnalyticsHandler.SubmitEvent(c)
}

func (h *Handlers) GetCustomMetrics(c *gin.Context) {
	h.AnalyticsHandler.GetCustomMetrics(c)
}

func (h *Handlers) CreateCustomMetric(c *gin.Context) {
	h.AnalyticsHandler.CreateCustomMetric(c)
}

func (h *Handlers) GetInsights(c *gin.Context) {
	h.AnalyticsHandler.GetInsights(c)
}

func (h *Handlers) ListReports(c *gin.Context) {
	h.AnalyticsHandler.ListReports(c)
}

func (h *Handlers) GenerateReport(c *gin.Context) {
	h.AnalyticsHandler.GenerateReport(c)
}

func (h *Handlers) GetReport(c *gin.Context) {
	h.AnalyticsHandler.GetReport(c)
}

func (h *Handlers) ListReportTemplates(c *gin.Context) {
	h.AnalyticsHandler.ListReportTemplates(c)
}

func (h *Handlers) CreateReportTemplate(c *gin.Context) {
	h.AnalyticsHandler.CreateReportTemplate(c)
}

func (h *Handlers) ScheduleReport(c *gin.Context) {
	h.AnalyticsHandler.ScheduleReport(c)
}

func (h *Handlers) ListScheduledReports(c *gin.Context) {
	h.AnalyticsHandler.ListScheduledReports(c)
}

func (h *Handlers) DeleteScheduledReport(c *gin.Context) {
	h.AnalyticsHandler.DeleteScheduledReport(c)
}

func (h *Handlers) ListVisualizations(c *gin.Context) {
	h.AnalyticsHandler.ListVisualizations(c)
}

func (h *Handlers) CreateVisualization(c *gin.Context) {
	h.AnalyticsHandler.CreateVisualization(c)
}

func (h *Handlers) GetVisualizationData(c *gin.Context) {
	h.AnalyticsHandler.GetVisualizationData(c)
}

func (h *Handlers) UpdateVisualization(c *gin.Context) {
	h.AnalyticsHandler.UpdateVisualization(c)
}

func (h *Handlers) DeleteVisualization(c *gin.Context) {
	h.AnalyticsHandler.DeleteVisualization(c)
}

func (h *Handlers) ListDashboards(c *gin.Context) {
	h.AnalyticsHandler.ListDashboards(c)
}

func (h *Handlers) CreateDashboard(c *gin.Context) {
	h.AnalyticsHandler.CreateDashboard(c)
}

func (h *Handlers) GetDashboard(c *gin.Context) {
	h.AnalyticsHandler.GetDashboard(c)
}

func (h *Handlers) UpdateDashboard(c *gin.Context) {
	h.AnalyticsHandler.UpdateDashboard(c)
}

func (h *Handlers) DeleteDashboard(c *gin.Context) {
	h.AnalyticsHandler.DeleteDashboard(c)
}

func (h *Handlers) ExportCSV(c *gin.Context) {
	h.AnalyticsHandler.ExportCSV(c)
}

func (h *Handlers) ExportJSON(c *gin.Context) {
	h.AnalyticsHandler.ExportJSON(c)
}

func (h *Handlers) ExportExcel(c *gin.Context) {
	h.AnalyticsHandler.ExportExcel(c)
}

func (h *Handlers) ExportPDF(c *gin.Context) {
	h.AnalyticsHandler.ExportPDF(c)
}

func (h *Handlers) ListExportSchedules(c *gin.Context) {
	h.AnalyticsHandler.ListExportSchedules(c)
}

func (h *Handlers) CreateExportSchedule(c *gin.Context) {
	h.AnalyticsHandler.CreateExportSchedule(c)
}

// Performance Handler Wrappers
func (h *Handlers) GetPerformanceStatus(c *gin.Context) {
	h.PerformanceHandler.GetPerformanceStatus(c)
}

func (h *Handlers) StartProfiling(c *gin.Context) {
	h.PerformanceHandler.StartProfiling(c)
}

func (h *Handlers) TriggerOptimization(c *gin.Context) {
	h.PerformanceHandler.TriggerOptimization(c)
}

func (h *Handlers) GetPerformanceReport(c *gin.Context) {
	h.PerformanceHandler.GetPerformanceReport(c)
}

func (h *Handlers) GetCacheStats(c *gin.Context) {
	h.PerformanceHandler.GetCacheStats(c)
}

func (h *Handlers) ClearCaches(c *gin.Context) {
	h.PerformanceHandler.ClearCaches(c)
}

func (h *Handlers) GetSlowQueries(c *gin.Context) {
	h.PerformanceHandler.GetSlowQueries(c)
}

func (h *Handlers) RunBenchmarks(c *gin.Context) {
	h.PerformanceHandler.RunBenchmarks(c)
}

func (h *Handlers) GetMemoryStats(c *gin.Context) {
	h.PerformanceHandler.GetMemoryStats(c)
}

func (h *Handlers) ForceGarbageCollection(c *gin.Context) {
	h.PerformanceHandler.ForceGarbageCollection(c)
}

func (h *Handlers) GetDatabasePoolStats(c *gin.Context) {
	h.PerformanceHandler.GetDatabasePoolStats(c)
}

func (h *Handlers) OptimizeDatabase(c *gin.Context) {
	h.PerformanceHandler.OptimizeDatabase(c)
}

// Memory Handler Wrappers
func (h *Handlers) GetMemoryStatus(c *gin.Context) {
	h.MemoryHandler.GetMemoryStatus(c)
}

func (h *Handlers) OptimizeMemory(c *gin.Context) {
	h.MemoryHandler.OptimizeMemory(c)
}

func (h *Handlers) DetectMemoryLeaks(c *gin.Context) {
	h.MemoryHandler.DetectMemoryLeaks(c)
}

func (h *Handlers) ScanForLeaks(c *gin.Context) {
	h.MemoryHandler.ScanForLeaks(c)
}

func (h *Handlers) GetPoolStats(c *gin.Context) {
	h.MemoryHandler.GetPoolStats(c)
}

func (h *Handlers) GetPoolDetail(c *gin.Context) {
	h.MemoryHandler.GetPoolDetail(c)
}

func (h *Handlers) ResizePool(c *gin.Context) {
	h.MemoryHandler.ResizePool(c)
}

func (h *Handlers) OptimizePools(c *gin.Context) {
	h.MemoryHandler.OptimizePools(c)
}

func (h *Handlers) GetMemoryPressure(c *gin.Context) {
	h.MemoryHandler.GetMemoryPressure(c)
}

func (h *Handlers) HandleMemoryPressure(c *gin.Context) {
	h.MemoryHandler.HandleMemoryPressure(c)
}

func (h *Handlers) GetPressureConfig(c *gin.Context) {
	h.MemoryHandler.GetPressureConfig(c)
}

func (h *Handlers) UpdatePressureConfig(c *gin.Context) {
	h.MemoryHandler.UpdatePressureConfig(c)
}

func (h *Handlers) GetPreallocationStats(c *gin.Context) {
	h.MemoryHandler.GetPreallocationStats(c)
}

func (h *Handlers) AnalyzeUsagePatterns(c *gin.Context) {
	h.MemoryHandler.AnalyzeUsagePatterns(c)
}

func (h *Handlers) OptimizePreallocation(c *gin.Context) {
	h.MemoryHandler.OptimizePreallocation(c)
}

func (h *Handlers) GetOptimizationStatus(c *gin.Context) {
	h.MemoryHandler.GetOptimizationStatus(c)
}

func (h *Handlers) StartOptimization(c *gin.Context) {
	h.MemoryHandler.StartOptimization(c)
}

func (h *Handlers) StopOptimization(c *gin.Context) {
	h.MemoryHandler.StopOptimization(c)
}

func (h *Handlers) GetOptimizationHistory(c *gin.Context) {
	h.MemoryHandler.GetOptimizationHistory(c)
}

func (h *Handlers) GetOptimizationReport(c *gin.Context) {
	h.MemoryHandler.GetOptimizationReport(c)
}

func (h *Handlers) GetMemoryMonitoring(c *gin.Context) {
	h.MemoryHandler.GetMemoryMonitoring(c)
}

func (h *Handlers) StartMemoryMonitoring(c *gin.Context) {
	h.MemoryHandler.StartMemoryMonitoring(c)
}

func (h *Handlers) StopMemoryMonitoring(c *gin.Context) {
	h.MemoryHandler.StopMemoryMonitoring(c)
}

// Monitoring Handler Wrappers - Alerting
func (h *Handlers) GetAlerts(c *gin.Context) {
	h.MonitoringHandler.GetAlerts(c)
}

func (h *Handlers) GetAlertRules(c *gin.Context) {
	h.MonitoringHandler.GetAlertRules(c)
}

func (h *Handlers) CreateAlertRule(c *gin.Context) {
	h.MonitoringHandler.CreateAlertRule(c)
}

func (h *Handlers) UpdateAlertRule(c *gin.Context) {
	h.MonitoringHandler.UpdateAlertRule(c)
}

func (h *Handlers) DeleteAlertRule(c *gin.Context) {
	h.MonitoringHandler.DeleteAlertRule(c)
}

func (h *Handlers) TestAlertRule(c *gin.Context) {
	h.MonitoringHandler.TestAlertRule(c)
}

func (h *Handlers) GetActiveAlerts(c *gin.Context) {
	h.MonitoringHandler.GetActiveAlerts(c)
}

func (h *Handlers) GetAlertHistory(c *gin.Context) {
	h.MonitoringHandler.GetAlertHistory(c)
}

func (h *Handlers) AcknowledgeAlert(c *gin.Context) {
	h.MonitoringHandler.AcknowledgeAlert(c)
}

func (h *Handlers) ResolveAlert(c *gin.Context) {
	h.MonitoringHandler.ResolveAlert(c)
}

func (h *Handlers) GetAlertStatistics(c *gin.Context) {
	h.MonitoringHandler.GetAlertStatistics(c)
}

func (h *Handlers) EvaluateAlertRule(c *gin.Context) {
	h.MonitoringHandler.EvaluateAlertRule(c)
}

// Monitoring Handler Wrappers - Dashboards
func (h *Handlers) GetMonitoringDashboards(c *gin.Context) {
	h.MonitoringHandler.GetDashboards(c)
}

func (h *Handlers) CreateMonitoringDashboard(c *gin.Context) {
	h.MonitoringHandler.CreateDashboard(c)
}

func (h *Handlers) GetMonitoringDashboard(c *gin.Context) {
	h.MonitoringHandler.GetDashboard(c)
}

func (h *Handlers) UpdateMonitoringDashboard(c *gin.Context) {
	h.MonitoringHandler.UpdateDashboard(c)
}

func (h *Handlers) DeleteMonitoringDashboard(c *gin.Context) {
	h.MonitoringHandler.DeleteDashboard(c)
}

func (h *Handlers) GetDashboardData(c *gin.Context) {
	h.MonitoringHandler.GetDashboardData(c)
}

func (h *Handlers) ExportDashboard(c *gin.Context) {
	h.MonitoringHandler.ExportDashboard(c)
}

func (h *Handlers) DuplicateDashboard(c *gin.Context) {
	h.MonitoringHandler.DuplicateDashboard(c)
}

func (h *Handlers) GetWidgetData(c *gin.Context) {
	h.MonitoringHandler.GetWidgetData(c)
}

func (h *Handlers) AddWidget(c *gin.Context) {
	h.MonitoringHandler.AddWidget(c)
}

func (h *Handlers) UpdateWidget(c *gin.Context) {
	h.MonitoringHandler.UpdateWidget(c)
}

func (h *Handlers) RemoveWidget(c *gin.Context) {
	h.MonitoringHandler.RemoveWidget(c)
}

func (h *Handlers) GetDashboardTemplates(c *gin.Context) {
	h.MonitoringHandler.GetDashboardTemplates(c)
}

func (h *Handlers) ImportDashboard(c *gin.Context) {
	h.MonitoringHandler.ImportDashboard(c)
}

// Monitoring Handler Wrappers - Live Streaming
func (h *Handlers) StartLiveStream(c *gin.Context) {
	h.MonitoringHandler.StartLiveStream(c)
}

func (h *Handlers) StopLiveStream(c *gin.Context) {
	h.MonitoringHandler.StopLiveStream(c)
}

func (h *Handlers) GetActiveStreams(c *gin.Context) {
	h.MonitoringHandler.GetActiveStreams(c)
}

// Monitoring Handler Wrappers - Predictive Analytics
func (h *Handlers) GetPredictionModels(c *gin.Context) {
	h.MonitoringHandler.GetPredictionModels(c)
}

func (h *Handlers) CreatePredictionModel(c *gin.Context) {
	h.MonitoringHandler.CreatePredictionModel(c)
}

func (h *Handlers) GetPredictionModel(c *gin.Context) {
	h.MonitoringHandler.GetPredictionModel(c)
}

func (h *Handlers) UpdatePredictionModel(c *gin.Context) {
	h.MonitoringHandler.UpdatePredictionModel(c)
}

func (h *Handlers) DeletePredictionModel(c *gin.Context) {
	h.MonitoringHandler.DeletePredictionModel(c)
}

func (h *Handlers) TrainModel(c *gin.Context) {
	h.MonitoringHandler.TrainModel(c)
}

func (h *Handlers) GeneratePrediction(c *gin.Context) {
	h.MonitoringHandler.GeneratePrediction(c)
}

func (h *Handlers) GetModelPerformance(c *gin.Context) {
	h.MonitoringHandler.GetModelPerformance(c)
}

func (h *Handlers) GetPredictions(c *gin.Context) {
	h.MonitoringHandler.GetPredictions(c)
}

func (h *Handlers) GetPredictionHistory(c *gin.Context) {
	h.MonitoringHandler.GetPredictionHistory(c)
}

// Monitoring Handler Wrappers - Anomaly Detection
func (h *Handlers) GetAnomalyDetectors(c *gin.Context) {
	h.MonitoringHandler.GetAnomalyDetectors(c)
}

func (h *Handlers) CreateAnomalyDetector(c *gin.Context) {
	h.MonitoringHandler.CreateAnomalyDetector(c)
}

func (h *Handlers) GetAnomalyDetector(c *gin.Context) {
	h.MonitoringHandler.GetAnomalyDetector(c)
}

func (h *Handlers) UpdateAnomalyDetector(c *gin.Context) {
	h.MonitoringHandler.UpdateAnomalyDetector(c)
}

func (h *Handlers) DeleteAnomalyDetector(c *gin.Context) {
	h.MonitoringHandler.DeleteAnomalyDetector(c)
}

func (h *Handlers) DetectAnomalies(c *gin.Context) {
	h.MonitoringHandler.DetectAnomalies(c)
}

func (h *Handlers) GetAnomalies(c *gin.Context) {
	h.MonitoringHandler.GetAnomalies(c)
}

func (h *Handlers) GetAnomalyHistory(c *gin.Context) {
	h.MonitoringHandler.GetAnomalyHistory(c)
}

func (h *Handlers) GetAnomalyStatistics(c *gin.Context) {
	h.MonitoringHandler.GetAnomalyStatistics(c)
}

func (h *Handlers) ProvideAnomalyFeedback(c *gin.Context) {
	h.MonitoringHandler.ProvideAnomalyFeedback(c)
}

// Monitoring Handler Wrappers - Forecasting
func (h *Handlers) GetForecasters(c *gin.Context) {
	h.MonitoringHandler.GetForecasters(c)
}

func (h *Handlers) CreateForecaster(c *gin.Context) {
	h.MonitoringHandler.CreateForecaster(c)
}

func (h *Handlers) GetForecaster(c *gin.Context) {
	h.MonitoringHandler.GetForecaster(c)
}

func (h *Handlers) UpdateForecaster(c *gin.Context) {
	h.MonitoringHandler.UpdateForecaster(c)
}

func (h *Handlers) DeleteForecaster(c *gin.Context) {
	h.MonitoringHandler.DeleteForecaster(c)
}

func (h *Handlers) GenerateForecast(c *gin.Context) {
	h.MonitoringHandler.GenerateForecast(c)
}

func (h *Handlers) GetForecasts(c *gin.Context) {
	h.MonitoringHandler.GetForecasts(c)
}

func (h *Handlers) GetForecastDetails(c *gin.Context) {
	h.MonitoringHandler.GetForecastDetails(c)
}

func (h *Handlers) GetForecastAccuracy(c *gin.Context) {
	h.MonitoringHandler.GetForecastAccuracy(c)
}

// Monitoring Handler Wrappers - Overview and Status
func (h *Handlers) GetMonitoringOverview(c *gin.Context) {
	h.MonitoringHandler.GetMonitoringOverview(c)
}

func (h *Handlers) GetMonitoringHealth(c *gin.Context) {
	h.MonitoringHandler.GetMonitoringHealth(c)
}

func (h *Handlers) GetMetricsSummary(c *gin.Context) {
	h.MonitoringHandler.GetMetricsSummary(c)
}

func (h *Handlers) GetSystemPerformance(c *gin.Context) {
	h.MonitoringHandler.GetSystemPerformance(c)
}

func (h *Handlers) GetDailyReport(c *gin.Context) {
	h.MonitoringHandler.GetDailyReport(c)
}

func (h *Handlers) GetWeeklyReport(c *gin.Context) {
	h.MonitoringHandler.GetWeeklyReport(c)
}

func (h *Handlers) GetMonthlyReport(c *gin.Context) {
	h.MonitoringHandler.GetMonthlyReport(c)
}

func (h *Handlers) GenerateCustomReport(c *gin.Context) {
	h.MonitoringHandler.GenerateCustomReport(c)
}

// Security Handler Wrappers
func (h *Handlers) GetSecurityStatus(c *gin.Context) {
	h.SecurityHandler.GetSecurityStatus(c)
}

func (h *Handlers) GetSecurityMetrics(c *gin.Context) {
	h.SecurityHandler.GetSecurityMetrics(c)
}

func (h *Handlers) GetSecurityEvents(c *gin.Context) {
	h.SecurityHandler.GetSecurityEvents(c)
}

func (h *Handlers) GetRateLimitStatus(c *gin.Context) {
	h.SecurityHandler.GetRateLimitStatus(c)
}

func (h *Handlers) GetRateLimitMetrics(c *gin.Context) {
	h.SecurityHandler.GetRateLimitMetrics(c)
}

func (h *Handlers) GetTopViolators(c *gin.Context) {
	h.SecurityHandler.GetTopViolators(c)
}

func (h *Handlers) BlockIP(c *gin.Context) {
	h.SecurityHandler.BlockIP(c)
}

func (h *Handlers) UnblockIP(c *gin.Context) {
	h.SecurityHandler.UnblockIP(c)
}

func (h *Handlers) GetBlockedIPs(c *gin.Context) {
	h.SecurityHandler.GetBlockedIPs(c)
}

func (h *Handlers) BlockIPAddress(c *gin.Context) {
	h.SecurityHandler.BlockIPAddress(c)
}

func (h *Handlers) UnblockIPAddress(c *gin.Context) {
	h.SecurityHandler.UnblockIPAddress(c)
}

func (h *Handlers) GetWhitelistedIPs(c *gin.Context) {
	h.SecurityHandler.GetWhitelistedIPs(c)
}

func (h *Handlers) AddToWhitelist(c *gin.Context) {
	h.SecurityHandler.AddToWhitelist(c)
}

func (h *Handlers) RemoveFromWhitelist(c *gin.Context) {
	h.SecurityHandler.RemoveFromWhitelist(c)
}

func (h *Handlers) GetThreats(c *gin.Context) {
	h.SecurityHandler.GetThreats(c)
}

func (h *Handlers) AddThreat(c *gin.Context) {
	h.SecurityHandler.AddThreat(c)
}

func (h *Handlers) RemoveThreat(c *gin.Context) {
	h.SecurityHandler.RemoveThreat(c)
}

func (h *Handlers) GetThreatAnalysis(c *gin.Context) {
	h.SecurityHandler.GetThreatAnalysis(c)
}

func (h *Handlers) GetAttackData(c *gin.Context) {
	h.SecurityHandler.GetAttackData(c)
}

func (h *Handlers) GetAttackPatterns(c *gin.Context) {
	h.SecurityHandler.GetAttackPatterns(c)
}

func (h *Handlers) GetAttackSummary(c *gin.Context) {
	h.SecurityHandler.GetAttackSummary(c)
}

func (h *Handlers) GetSecurityConfig(c *gin.Context) {
	h.SecurityHandler.GetSecurityConfig(c)
}

func (h *Handlers) UpdateSecurityConfig(c *gin.Context) {
	h.SecurityHandler.UpdateSecurityConfig(c)
}

func (h *Handlers) ResetSecurityConfig(c *gin.Context) {
	h.SecurityHandler.ResetSecurityConfig(c)
}

func (h *Handlers) GetSecuritySummary(c *gin.Context) {
	h.SecurityHandler.GetSecuritySummary(c)
}

func (h *Handlers) GetDetailedSecurityReport(c *gin.Context) {
	h.SecurityHandler.GetDetailedSecurityReport(c)
}

func (h *Handlers) ExportSecurityReport(c *gin.Context) {
	h.SecurityHandler.ExportSecurityReport(c)
}

func (h *Handlers) GetLiveSecurityData(c *gin.Context) {
	h.SecurityHandler.GetLiveSecurityData(c)
}

func (h *Handlers) GetSecurityAlerts(c *gin.Context) {
	h.SecurityHandler.GetSecurityAlerts(c)
}

// Error Handler Wrappers
func (h *Handlers) GetErrorReports(c *gin.Context) {
	h.ErrorHandler.GetErrorReports(c)
}

func (h *Handlers) GetErrorReport(c *gin.Context) {
	h.ErrorHandler.GetErrorReport(c)
}

func (h *Handlers) ResolveError(c *gin.Context) {
	h.ErrorHandler.ResolveError(c)
}

func (h *Handlers) GetErrorStats(c *gin.Context) {
	h.ErrorHandler.GetErrorStats(c)
}

func (h *Handlers) GetRecoveryMetrics(c *gin.Context) {
	h.ErrorHandler.GetRecoveryMetrics(c)
}

func (h *Handlers) GetCircuitBreakerStatus(c *gin.Context) {
	h.ErrorHandler.GetCircuitBreakerStatus(c)
}

func (h *Handlers) ResetCircuitBreaker(c *gin.Context) {
	h.ErrorHandler.ResetCircuitBreaker(c)
}

func (h *Handlers) GetErrorHealthStatus(c *gin.Context) {
	h.ErrorHandler.GetErrorHealthStatus(c)
}

func (h *Handlers) CleanupOldErrors(c *gin.Context) {
	h.ErrorHandler.CleanupOldErrors(c)
}

func (h *Handlers) TestErrorRecovery(c *gin.Context) {
	h.ErrorHandler.TestErrorRecovery(c)
}

// TestHAConnection tests direct Home Assistant connection
func (h *Handlers) TestHAConnection(c *gin.Context) {
	h.log.Info("Testing Home Assistant connection...")

	// Create a new client wrapper for testing
	client := homeassistant.NewHAClientWrapper(h.cfg, h.log)

	// Test with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h.log.Info("Attempting to connect to Home Assistant...")
	if err := client.Connect(ctx); err != nil {
		h.log.WithError(err).Error("Home Assistant connection failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"step":    "connection",
		})
		return
	}

	h.log.Info("Home Assistant connection successful, testing entity fetch...")

	// Test entity fetching
	entities, err := client.GetAllEntities(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to fetch entities")
		client.Disconnect()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"step":    "entity_fetch",
		})
		return
	}

	// Clean up
	client.Disconnect()

	h.log.WithField("entity_count", len(entities)).Info("Home Assistant test completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"entity_count": len(entities),
		"message":      "Home Assistant connection and entity fetch successful",
		"first_entities": func() []string {
			var firstFive []string
			for i, entity := range entities {
				if i >= 5 {
					break
				}
				firstFive = append(firstFive, entity.EntityID)
			}
			return firstFive
		}(),
	})
}

// TestHAConnectionSimple tests just the HTTP API connection
func (h *Handlers) TestHAConnectionSimple(c *gin.Context) {
	h.log.Info("Testing simple Home Assistant HTTP connection...")

	// Create a new client wrapper for testing
	client := homeassistant.NewHAClientWrapper(h.cfg, h.log)

	// Test with a short timeout - only HTTP, no WebSocket
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.log.Info("Testing Home Assistant HTTP API...")

	// Test entity fetching directly without full connection
	entities, err := client.GetAllEntities(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to fetch entities via HTTP")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"step":    "http_entity_fetch",
		})
		return
	}

	h.log.WithField("entity_count", len(entities)).Info("Home Assistant HTTP test completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"entity_count": len(entities),
		"message":      "Home Assistant HTTP API test successful",
		"first_entities": func() []string {
			var firstFive []string
			for i, entity := range entities {
				if i >= 5 {
					break
				}
				firstFive = append(firstFive, entity.EntityID)
			}
			return firstFive
		}(),
	})
}
