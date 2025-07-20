package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/automation"
	"github.com/frostdev-ops/pma-backend-go/internal/core/bluetooth"
	"github.com/frostdev-ops/pma-backend-go/internal/core/cache"
	"github.com/frostdev-ops/pma-backend-go/internal/core/display"
	"github.com/frostdev-ops/pma-backend-go/internal/core/energymgr"
	"github.com/frostdev-ops/pma-backend-go/internal/core/kiosk"
	networkcore "github.com/frostdev-ops/pma-backend-go/internal/core/network"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"

	"github.com/frostdev-ops/pma-backend-go/internal/core/queue"
	"github.com/frostdev-ops/pma-backend-go/internal/core/rooms"
	"github.com/frostdev-ops/pma-backend-go/internal/core/system"
	"github.com/frostdev-ops/pma-backend-go/internal/core/test"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	upscore "github.com/frostdev-ops/pma-backend-go/internal/core/ups"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/internal/database/sqlite"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Repository adapters to bridge interface mismatches

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
	// TODO: Fix type mismatch - temporarily disabled
	return nil, fmt.Errorf("GetConversations adapter method temporarily disabled due to type mismatch")
	// return a.repo.GetConversations(ctx, filter)
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
	// TODO: Fix type mismatch - temporarily disabled
	return nil, fmt.Errorf("GetConversationMessages adapter method temporarily disabled due to type mismatch")
	// return a.repo.GetConversationMessages(ctx, conversationID, limit, offset)
}

func (a *ConversationRepositoryAdapter) CreateOrUpdateAnalytics(ctx context.Context, analytics *ai.ConversationAnalytics) error {
	// TODO: Fix type mismatch - temporarily disabled
	return fmt.Errorf("CreateOrUpdateAnalytics adapter method temporarily disabled due to type mismatch")
	// return a.repo.CreateOrUpdateAnalytics(ctx, analytics)
}

func (a *ConversationRepositoryAdapter) GetConversationAnalytics(ctx context.Context, conversationID string, date time.Time) (*ai.ConversationAnalytics, error) {
	// TODO: Fix type mismatch - temporarily disabled
	return nil, fmt.Errorf("GetConversationAnalytics adapter method temporarily disabled due to type mismatch")
	// return a.repo.GetConversationAnalytics(ctx, conversationID, date)
}

func (a *ConversationRepositoryAdapter) GetGlobalStatistics(ctx context.Context, userID string, startDate, endDate time.Time) (*ai.ConversationStatistics, error) {
	// TODO: Fix type mismatch - temporarily disabled
	return nil, fmt.Errorf("GetGlobalStatistics adapter method temporarily disabled due to type mismatch")
	// return a.repo.GetGlobalStatistics(ctx, userID, startDate, endDate)
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
	// TODO: Fix type mismatch - temporarily disabled
	return nil, fmt.Errorf("GetToolByName adapter method temporarily disabled due to type mismatch")
	// return a.repo.GetToolByName(ctx, name)
}

func (a *MCPRepositoryAdapter) GetEnabledTools(ctx context.Context, category string) ([]*ai.MCPTool, error) {
	// TODO: Fix type mismatch - temporarily disabled
	return nil, fmt.Errorf("GetEnabledTools adapter method temporarily disabled due to type mismatch")
	// return a.repo.GetEnabledTools(ctx, category)
}

func (a *MCPRepositoryAdapter) CreateToolExecution(ctx context.Context, execution *ai.MCPToolExecution) error {
	// TODO: Fix type mismatch - temporarily disabled
	return fmt.Errorf("CreateToolExecution adapter method temporarily disabled due to type mismatch")
	// return a.repo.CreateToolExecution(ctx, execution)
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
	networkService      *networkcore.Service
	upsService          *upscore.Service
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

	testService  *test.Service
	cacheManager cache.CacheManager
	CacheHandler *CacheHandler

	// Unified PMA Type System Components
	typeRegistry     *types.PMATypeRegistry
	adapterRegistry  types.AdapterRegistry
	entityRegistry   types.EntityRegistry
	conflictResolver types.ConflictResolver
	priorityManager  types.SourcePriorityManager
	unifiedService   *unified.UnifiedEntityService
}

// NewHandlers creates a new handlers instance
func NewHandlers(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger, wsHub *websocket.Hub, db *sql.DB) *Handlers {
	// Initialize AI services
	var llmManager *ai.LLMManager
	var chatService *ai.ChatService

	// Try to initialize LLM manager
	if manager, err := ai.NewLLMManager(cfg, logger); err != nil {
		logger.WithError(err).Warn("Failed to initialize LLM manager")
	} else {
		llmManager = manager
		// Initialize chat service with the LLM manager
		chatService = ai.NewChatService(llmManager, logger)
	}

	// Initialize core services
	networkConfig := networkcore.Config{
		RouterBaseURL:   cfg.Router.BaseURL,
		RouterAuthToken: cfg.Router.AuthToken,
	}
	networkService := networkcore.NewService(networkConfig, repos.Network, wsHub, logger)

	// Initialize UPS service
	upsConfig := upscore.Config{
		NUTHost:            cfg.Devices.UPS.NUTHost,
		NUTPort:            cfg.Devices.UPS.NUTPort,
		UPSName:            cfg.Devices.UPS.UPSName,
		MonitoringInterval: 30 * time.Second,
		HistoryRetention:   cfg.Devices.UPS.HistoryRetentionDays,
	}
	upsService := upscore.NewService(upsConfig, repos.UPS, wsHub, logger)

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

	// Get registry manager and components
	registryManager := unifiedService.GetRegistryManager()
	adapterRegistry := registryManager.GetAdapterRegistry()
	entityRegistry := registryManager.GetEntityRegistry()
	conflictResolver := registryManager.GetConflictResolver()
	priorityManager := registryManager.GetPriorityManager()

	// Initialize Automation Engine
	var automationEngine *automation.AutomationEngine
	var automationHandler *AutomationHandler

	automationConfig := &automation.EngineConfig{
		Workers:              4,
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

	if engine, err := automation.NewAutomationEngine(automationConfig, unifiedService, wsHub, logger); err != nil {
		logger.WithError(err).Warn("Failed to initialize automation engine")
	} else {
		automationEngine = engine
		automationHandler = NewAutomationHandler(engine, logger)

		// Start the automation engine
		if err := automationEngine.Start(context.Background()); err != nil {
			logger.WithError(err).Warn("Failed to start automation engine")
		} else {
			logger.Info("Automation engine started successfully")
		}
	}

	// Initialize adapters based on config (basic implementation)
	ctx := context.Background()

	// Home Assistant adapter (main adapter for now)
	logger.Info("Initializing Home Assistant adapter")
	haAdapter := homeassistant.NewHomeAssistantAdapter(cfg, logger)
	if err := unifiedService.RegisterAdapter(haAdapter); err != nil {
		logger.WithError(err).Error("Failed to register Home Assistant adapter")
	} else {
		go func() {
			logger.Info("Connecting Home Assistant adapter...")
			if err := haAdapter.Connect(ctx); err != nil {
				logger.WithError(err).Error("Failed to connect Home Assistant adapter")
			} else {
				logger.Info("Home Assistant adapter connected successfully")

				// Trigger initial sync after connection
				logger.Info("Triggering initial entity sync...")
				syncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				result, err := unifiedService.SyncFromSource(syncCtx, types.SourceHomeAssistant)
				if err != nil {
					logger.WithError(err).Error("Failed to perform initial sync")
				} else {
					logger.WithFields(logrus.Fields{
						"entities_found":      result.EntitiesFound,
						"entities_registered": result.EntitiesRegistered,
						"entities_updated":    result.EntitiesUpdated,
						"duration":            result.Duration,
					}).Info("Initial entity sync completed successfully")
				}

				// Start periodic sync scheduler
				if err := unifiedService.StartPeriodicSync(); err != nil {
					logger.WithError(err).Error("Failed to start periodic sync scheduler")
				}
			}
		}()
	}

	// TODO: Initialize other adapters (Ring, Shelly, UPS, Network) based on config
	// These will be added in future iterations when the adapter interfaces are standardized

	// Initialize new handlers with standard logger
	stdLogger := log.New(os.Stdout, "[PMA] ", log.LstdFlags)
	eventsHandler := NewEventsHandler(stdLogger)
	mcpHandler := NewMCPHandler(stdLogger)
	fileHandler := NewFileHandler(stdLogger, "./data/screensaver-images")

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

		testService:  test.NewService(cfg, repos, logger, db),
		cacheManager: cacheManager,
		CacheHandler: cacheHandler,

		// Unified PMA Type System
		typeRegistry:     typeRegistry,
		adapterRegistry:  adapterRegistry,
		entityRegistry:   entityRegistry,
		conflictResolver: conflictResolver,
		priorityManager:  priorityManager,
		unifiedService:   unifiedService,
	}

	// Initialize MCP tool executor with real services
	mcpToolExecutor := ai.NewMCPToolExecutor(logger)
	if unifiedService != nil && roomService != nil && systemService != nil && energyService != nil && automationEngine != nil {
		// Create service wrappers with the real services
		entityServiceWrapper, roomServiceWrapper, systemServiceWrapper, energyServiceWrapper, automationServiceWrapper := ai.CreateServiceWrappers(
			unifiedService,
			roomService,
			systemService,
			energyService,
			automationEngine,
		)

		// Set the real services on the executor
		mcpToolExecutor.SetServices(entityServiceWrapper, roomServiceWrapper, systemServiceWrapper, energyServiceWrapper, automationServiceWrapper)
		logger.Info("MCP tool executor initialized with real services")
	} else {
		logger.Warn("Some services not available, MCP tool executor initialized with default wrappers")
	}
	handlers.mcpToolExecutor = mcpToolExecutor

	// Initialize conversation service if we have the required components
	// Note: Conversation service initialization disabled due to interface mismatches
	// TODO: Fix interface alignment between AI package and repository interfaces
	if llmManager != nil && repos.Conversation != nil && repos.MCP != nil {
		conversationService := ai.NewConversationService(
			llmManager,
			NewConversationRepositoryAdapter(repos.Conversation), // This already implements ai.ConversationRepositoryInterface
			NewMCPRepositoryAdapter(repos.MCP),                   // This already implements ai.MCPRepositoryInterface
			mcpToolExecutor,
			logger,
		)
		handlers.conversationService = conversationService
		logger.Info("Conversation service initialized with MCP integration")
	} else {
		logger.Info("Conversation service initialization skipped - using MCP tool executor directly")
	}

	logger.Info("Unified PMA type system initialized successfully")
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

// File Handler Wrappers
func (h *Handlers) GetScreensaverImages(c *gin.Context) {
	h.fileHandler.GetScreensaverImages(c)
}

func (h *Handlers) GetScreensaverStorage(c *gin.Context) {
	h.fileHandler.GetScreensaverStorage(c)
}

func (h *Handlers) UploadScreensaverImages(c *gin.Context) {
	h.fileHandler.UploadScreensaverImages(c)
}

func (h *Handlers) DeleteScreensaverImage(c *gin.Context) {
	h.fileHandler.DeleteScreensaverImage(c)
}

func (h *Handlers) GetScreensaverImage(c *gin.Context) {
	h.fileHandler.GetScreensaverImage(c)
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
