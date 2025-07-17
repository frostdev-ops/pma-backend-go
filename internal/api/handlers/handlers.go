package handlers

import (
	"database/sql"
	"log"
	"os"
	"time"

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
	haSyncHandler    *HomeAssistantSyncHandler
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
		logger.WithError(err).Warn("Failed to initialize LLM manager")
	} else {
		llmManager = manager
		// Initialize chat service with the LLM manager
		chatService = ai.NewChatService(llmManager, logger)
	}

	// Initialize core services
	networkConfig := network.Config{
		RouterBaseURL:   "http://localhost:8080", // TODO: Get from config
		RouterAuthToken: "",                      // TODO: Get from config
	}
	networkService := network.NewService(networkConfig, repos.Network, wsHub, logger)

	// Initialize UPS service
	upsConfig := ups.Config{
		NUTHost:            "localhost", // TODO: Get from config
		NUTPort:            3493,        // TODO: Get from config
		UPSName:            "ups",       // TODO: Get from config
		MonitoringInterval: 30 * time.Second,
		HistoryRetention:   30, // days
	}
	upsService := ups.NewService(upsConfig, repos.UPS, wsHub, logger)

	// Initialize System service
	systemService := system.NewService(logger, 2000) // 2000 max log entries

	// Initialize Display service
	displayService := display.NewService(db, logger)

	// Initialize Bluetooth service
	bluetoothService := bluetooth.NewService(repos.Bluetooth, logger)

	// Initialize Energy service
	energyService := energymgr.NewService(repos.Energy, repos.Entity, repos.UPS, logger)

	// Initialize Home Assistant Sync Handler
	haSyncHandler := NewHomeAssistantSyncHandler(repos.HomeAssistant, logger)

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
		haSyncHandler:    haSyncHandler,
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
