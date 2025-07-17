package handlers

import (
	"database/sql"
	"log"
	"os"

	"github.com/frostdev-ops/pma-backend-go/internal/ai"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/bluetooth"
	"github.com/frostdev-ops/pma-backend-go/internal/core/display"
	"github.com/frostdev-ops/pma-backend-go/internal/core/energymgr"
	"github.com/frostdev-ops/pma-backend-go/internal/core/network"
	"github.com/frostdev-ops/pma-backend-go/internal/core/system"
	"github.com/frostdev-ops/pma-backend-go/internal/core/ups"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handlers holds all HTTP handlers and their dependencies
type Handlers struct {
	cfg              *config.Config
	repos            *database.Repositories
	log              *logrus.Logger
	wsHub            *websocket.Hub
	automation       *SimpleAutomationHandler
	llmManager       *ai.LLMManager
	chatService      *ai.ChatService
	networkService   *network.Service
	upsService       *ups.Service
	systemService    *system.Service
	displayService   *display.Service
	bluetoothService *bluetooth.Service
	energyService    *energymgr.Service
	eventsHandler    *EventsHandler
	mcpHandler       *MCPHandler
	fileHandler      *FileHandler
}

// NewHandlers creates a new handlers instance
func NewHandlers(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger, wsHub *websocket.Hub, db *sql.DB) *Handlers {
	// Create a simple automation handler (without engine for now)
	automationHandler := &SimpleAutomationHandler{
		logger: logger,
	}

	// Initialize AI services
	var llmManager *ai.LLMManager
	var chatService *ai.ChatService

	// Try to initialize LLM manager
	if manager, err := ai.NewLLMManager(cfg, logger); err != nil {
		logger.Warnf("Failed to initialize LLM manager: %v", err)
	} else {
		llmManager = manager
	}

	// Try to initialize chat service
	if service, err := ai.NewChatService(cfg, llmManager, logger); err != nil {
		logger.Warnf("Failed to initialize chat service: %v", err)
	} else {
		chatService = service
	}

	// Initialize core services
	networkService := network.NewService(cfg, logger)
	upsService := ups.NewService(cfg, logger)
	systemService := system.NewService(cfg, logger)
	displayService := display.NewService(cfg, logger)
	bluetoothService := bluetooth.NewService(cfg, logger)
	energyService := energymgr.NewService(cfg, repos, logger, db)

	// Initialize new handlers with standard logger
	stdLogger := log.New(os.Stdout, "[PMA] ", log.LstdFlags)
	eventsHandler := NewEventsHandler(stdLogger)
	mcpHandler := NewMCPHandler(stdLogger)
	fileHandler := NewFileHandler(stdLogger, "./data/screensaver-images")

	return &Handlers{
		cfg:              cfg,
		repos:            repos,
		log:              logger,
		wsHub:            wsHub,
		automation:       automationHandler,
		llmManager:       llmManager,
		chatService:      chatService,
		networkService:   networkService,
		upsService:       upsService,
		systemService:    systemService,
		displayService:   displayService,
		bluetoothService: bluetoothService,
		energyService:    energyService,
		eventsHandler:    eventsHandler,
		mcpHandler:       mcpHandler,
		fileHandler:      fileHandler,
	}
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

// Automation handler methods that return not implemented for now
func (h *Handlers) GetAutomations(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) GetAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) CreateAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) UpdateAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) DeleteAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) EnableAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) DisableAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) TestAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) ImportAutomations(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) ExportAutomations(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) ValidateAutomation(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) GetAutomationStatistics(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) GetAutomationTemplates(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}

func (h *Handlers) GetAutomationHistory(c *gin.Context) {
	c.JSON(501, gin.H{"error": "automation engine not yet integrated"})
}
