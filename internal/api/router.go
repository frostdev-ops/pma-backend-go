package api

import (
	"database/sql"

	"github.com/frostdev-ops/pma-backend-go/internal/api/handlers"
	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewRouter creates and configures the main HTTP router
func NewRouter(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger, wsHub *websocket.Hub, db *sql.DB) *gin.Engine {
	// Set gin mode based on config
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.ErrorHandlingMiddleware(logger))
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.CORSMiddleware())

	// Rate limiting
	rateLimiter := middleware.NewRateLimiter(100, 200) // 100 requests/sec, burst 200
	router.Use(rateLimiter.RateLimitMiddleware())

	// Initialize handlers
	h := handlers.NewHandlers(cfg, repos, logger, wsHub, db)

	// Public routes
	router.GET("/health", h.Health)

	// WebSocket endpoint (no auth required for connection)
	router.GET("/ws", h.WebSocketHandler(wsHub))

	// API v1 routes
	api := router.Group("/api/v1")
	{
		// Authentication routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/validate", h.ValidateToken)

			// PIN-based authentication (compatible with Node backend)
			auth.POST("/verify-pin", h.VerifyPin)
			auth.POST("/set-pin", h.SetPin)
			auth.GET("/pin-status", h.GetPinStatus)
			auth.GET("/session", h.GetSession)
		}

		// Public API routes (no auth required)
		public := api.Group("/")
		{
			public.GET("/status", h.Health)

			// SSE stream endpoint (public for real-time updates)
			public.GET("/events/stream", h.GetEventStream)

			// Image serving endpoint (public for screensaver display)
			public.GET("/screensaver/images/:filename", h.GetScreensaverImage)
		}

		// Mobile upload page (public)
		router.GET("/upload", h.GetMobileUploadPage)

		// Protected API routes (auth required)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg.Auth.JWTSecret))
		{
			// User profile routes
			profile := protected.Group("/profile")
			{
				profile.GET("/", h.GetProfile)
				profile.PUT("/password", h.UpdatePassword)
			}

			// Protected PIN authentication routes
			authProtected := protected.Group("/auth")
			{
				authProtected.POST("/change-pin", h.ChangePin)
				authProtected.POST("/disable-pin", h.DisablePin)
				authProtected.POST("/logout", h.Logout)
			}

			// User management routes (admin functionality)
			users := protected.Group("/users")
			{
				users.GET("/", h.GetAllUsers)
				users.DELETE("/:id", h.DeleteUser)
			}
			// Configuration endpoints
			config := protected.Group("/config")
			{
				config.GET("/:key", h.GetConfig)
				config.PUT("/:key", h.SetConfig)
				config.GET("/", h.GetAllConfig)
			}

			// Entity endpoints
			entities := protected.Group("/entities")
			{
				entities.GET("/", h.GetEntities)
				entities.GET("/:id", h.GetEntity)
				entities.POST("/", h.CreateOrUpdateEntity)
				entities.PUT("/:id/state", h.UpdateEntityState)
				entities.PUT("/:id/room", h.AssignEntityToRoom)
				entities.DELETE("/:id", h.DeleteEntity)
			}

			// Room endpoints
			rooms := protected.Group("/rooms")
			{
				rooms.GET("/", h.GetRooms)
				rooms.GET("/:id", h.GetRoom)
				rooms.POST("/", h.CreateRoom)
				rooms.PUT("/:id", h.UpdateRoom)
				rooms.DELETE("/:id", h.DeleteRoom)
				rooms.GET("/stats", h.GetRoomStats)
				rooms.POST("/sync-ha", h.SyncRoomsWithHA)
			}

			// Scene endpoints
			scenes := protected.Group("/scenes")
			{
				scenes.GET("/", h.GetScenes)
				scenes.GET("/:id", h.GetScene)
				scenes.POST("/:id/activate", h.ActivateScene)
			}

			// WebSocket management endpoints (protected)
			ws := protected.Group("/websocket")
			{
				ws.GET("/stats", h.GetWebSocketStats(wsHub))
				ws.POST("/broadcast", h.BroadcastMessage(wsHub))

				// Home Assistant WebSocket event subscriptions
				ha := ws.Group("/ha")
				{
					ha.POST("/subscribe", h.SubscribeToHAEvents(wsHub))
					ha.POST("/unsubscribe", h.UnsubscribeFromHAEvents(wsHub))
					ha.GET("/subscriptions", h.GetHASubscriptions(wsHub))
				}
			}

			// AI endpoints
			ai := protected.Group("/ai")
			{
				ai.POST("/chat", h.ChatWithAI)
				ai.POST("/complete", h.CompleteText)
				ai.POST("/chat/context", h.ChatWithContext)
				ai.GET("/providers", h.GetProviders)
				ai.GET("/models", h.GetModels)
				ai.GET("/statistics", h.GetAIStatistics)
				ai.POST("/test/:provider", h.TestAIProvider)
				ai.GET("/summary", h.GetSystemSummary)
				ai.POST("/analyze/entity/:id", h.AnalyzeEntity)
				ai.POST("/generate/automation", h.GenerateAutomation)
			}

			// Automation endpoints
			automation := protected.Group("/automation")
			{
				automation.GET("/rules", h.GetAutomations)
				automation.GET("/rules/:id", h.GetAutomation)
				automation.POST("/rules", h.CreateAutomation)
				automation.PUT("/rules/:id", h.UpdateAutomation)
				automation.DELETE("/rules/:id", h.DeleteAutomation)
				automation.POST("/rules/:id/enable", h.EnableAutomation)
				automation.POST("/rules/:id/disable", h.DisableAutomation)
				automation.POST("/rules/:id/test", h.TestAutomation)
				automation.POST("/rules/import", h.ImportAutomations)
				automation.GET("/rules/export", h.ExportAutomations)
				automation.POST("/rules/validate", h.ValidateAutomation)
				automation.GET("/statistics", h.GetAutomationStatistics)
				automation.GET("/templates", h.GetAutomationTemplates)
				automation.GET("/history", h.GetAutomationHistory)
			}

			// Network management endpoints
			network := protected.Group("/network")
			{
				// Status and monitoring
				network.GET("/status", h.GetNetworkStatus)
				network.GET("/interfaces", h.GetNetworkInterfaces)
				network.GET("/traffic", h.GetTrafficStatistics)
				network.GET("/metrics", h.GetNetworkMetrics)
				network.GET("/config", h.GetNetworkConfiguration)
				network.POST("/test-connection", h.TestRouterConnection)

				// Device discovery and management
				network.GET("/devices", h.GetDiscoveredDevices)
				network.POST("/devices/scan", h.ScanNetworkDevices)
				network.GET("/devices/suggestions", h.GetDevicesWithPortSuggestions)

				// Port forwarding management
				network.GET("/port-forwarding", h.GetPortForwardingRules)
				network.POST("/port-forwarding", h.CreatePortForwardingRule)
				network.PUT("/port-forwarding/:ruleId", h.UpdatePortForwardingRule)
				network.DELETE("/port-forwarding/:ruleId", h.DeletePortForwardingRule)
			}

			// UPS monitoring endpoints
			ups := protected.Group("/ups")
			{
				// Status and monitoring
				ups.GET("/status", h.GetUPSStatus)
				ups.GET("/history", h.GetUPSHistory)
				ups.GET("/battery-trends", h.GetUPSBatteryTrends)
				ups.GET("/metrics", h.GetUPSMetrics)
				ups.GET("/config", h.GetUPSConfiguration)

				// UPS information and variables
				ups.GET("/info", h.GetUPSInfo)
				ups.GET("/variables", h.GetUPSVariables)
				ups.GET("/connection", h.GetUPSConnectionInfo)
				ups.POST("/test-connection", h.TestUPSConnection)

				// Monitoring control
				ups.POST("/monitoring/start", h.StartUPSMonitoring)
				ups.POST("/monitoring/stop", h.StopUPSMonitoring)

				// Alert configuration
				ups.PUT("/alerts/thresholds", h.UpdateUPSAlertThresholds)
			}

			// System management endpoints
			system := protected.Group("/system")
			{
				// Basic information and health
				system.GET("/info", h.GetSystemInfo)
				system.GET("/status", h.GetSystemStatus)
				system.GET("/health", h.GetBasicSystemHealth)
				system.GET("/health/detailed", h.GetSystemHealth)
				system.GET("/device-info", h.GetDeviceInfo)

				// System logs
				system.GET("/logs", h.GetSystemLogs)

				// Power management
				system.POST("/reboot", h.RebootSystem)
				system.POST("/shutdown", h.ShutdownSystem)

				// Configuration
				system.GET("/config", h.GetSystemConfig)
				system.POST("/config", h.UpdateSystemConfig)

				// Health reporting (for external monitoring)
				system.POST("/health-report", h.ReportHealth)
			}

			// Display settings endpoints
			display := protected.Group("/display-settings")
			{
				// Main display settings
				display.GET("/", h.GetDisplaySettings)
				display.POST("/", h.UpdateDisplaySettings)
				display.PUT("/", h.PutDisplaySettings)

				// Hardware capabilities and control
				display.GET("/capabilities", h.GetDisplayCapabilities)
				display.GET("/hardware", h.GetDisplayHardwareInfo)
				display.POST("/wake", h.WakeScreen)
			}

			// Bluetooth management endpoints
			bluetooth := protected.Group("/bluetooth")
			{
				// Status and adapter management
				bluetooth.GET("/status", h.GetBluetoothStatus)
				bluetooth.GET("/capabilities", h.GetBluetoothCapabilities)
				bluetooth.POST("/power", h.SetBluetoothPower)
				bluetooth.POST("/discoverable", h.SetBluetoothDiscoverable)

				// Device scanning and discovery
				bluetooth.POST("/scan", h.ScanForDevices)
				bluetooth.GET("/devices", h.GetAllBluetoothDevices)
				bluetooth.GET("/devices/paired", h.GetPairedDevices)
				bluetooth.GET("/devices/connected", h.GetConnectedDevices)
				bluetooth.GET("/devices/:address", h.GetBluetoothDevice)

				// Device pairing and connection
				bluetooth.POST("/devices/:address/pair", h.PairBluetoothDevice)
				bluetooth.POST("/devices/:address/connect", h.ConnectBluetoothDevice)
				bluetooth.POST("/devices/:address/disconnect", h.DisconnectBluetoothDevice)
				bluetooth.DELETE("/devices/:address", h.RemoveBluetoothDevice)

				// Statistics and monitoring
				bluetooth.GET("/stats", h.GetBluetoothStats)
			}

			// Energy management endpoints
			energy := protected.Group("/energy")
			{
				// Settings management
				energy.GET("/settings", h.GetEnergySettings)
				energy.PUT("/settings", h.UpdateEnergySettings)

				// Current energy data
				energy.GET("/data", h.GetEnergyData)
				energy.GET("/metrics", h.GetEnergyMetrics)

				// Energy history and statistics
				energy.GET("/history", h.GetEnergyHistory)
				energy.GET("/statistics", h.GetEnergyStatistics)

				// Device-specific energy data
				energy.GET("/devices/breakdown", h.GetEnergyDeviceBreakdown)
				energy.GET("/devices/:entityId/history", h.GetDeviceEnergyHistory)
				energy.GET("/devices/:entityId/data", h.GetDeviceEnergyData)

				// Tracking control
				energy.POST("/tracking/start", h.StartEnergyTracking)
				energy.POST("/tracking/stop", h.StopEnergyTracking)

				// Service management
				energy.GET("/service/status", h.GetEnergyServiceStatus)
				energy.POST("/cleanup", h.CleanupOldEnergyData)
			}

			// Ring camera integration endpoints
			ring := protected.Group("/ring")
			{
				// Configuration endpoints
				ring.GET("/config/status", h.GetRingConfigStatus)
				ring.POST("/config/setup", h.SetupRingConfig)
				ring.POST("/config/test", h.TestRingConnection)
				ring.DELETE("/config", h.DeleteRingConfig)
				ring.POST("/config/restart", h.RestartRingService)

				// Authentication endpoints
				ring.POST("/auth/start", h.StartRingAuthentication)
				ring.POST("/auth/verify", h.Complete2FA)

				// Camera endpoints
				ring.GET("/cameras", h.GetRingCameras)
				ring.GET("/cameras/:cameraId", h.GetRingCamera)
				ring.GET("/cameras/:cameraId/snapshot", h.GetRingCameraSnapshot)
				ring.POST("/cameras/:cameraId/light", h.ControlRingLight)
				ring.POST("/cameras/:cameraId/siren", h.ControlRingSiren)
				ring.GET("/cameras/:cameraId/events", h.GetRingCameraEvents)

				// Service status
				ring.GET("/status", h.GetRingStatus)
			}

			// Shelly device integration endpoints
			shelly := protected.Group("/shelly")
			{
				// Device management
				shelly.POST("/devices", h.AddShellyDevice)
				shelly.DELETE("/devices/:id", h.RemoveShellyDevice)
				shelly.GET("/devices", h.GetShellyDevices)
				shelly.GET("/devices/:id/status", h.GetShellyDeviceStatus)
				shelly.POST("/devices/:id/control", h.ControlShellyDevice)
				shelly.GET("/devices/:id/energy", h.GetShellyDeviceEnergy)

				// Discovery endpoints
				shelly.GET("/discovery/devices", h.GetDiscoveredShellyDevices)
				shelly.POST("/discovery/start", h.StartShellyDiscovery)
				shelly.POST("/discovery/stop", h.StopShellyDiscovery)
			}

			// Enhanced conversation management endpoints
			conversations := protected.Group("/conversations")
			{
				// Conversation CRUD operations
				conversations.POST("", h.CreateConversation)
				conversations.GET("", h.GetConversations)
				conversations.GET("/:id", h.GetConversation)
				conversations.PUT("/:id", h.UpdateConversation)
				conversations.DELETE("/:id", h.DeleteConversation)

				// Message management
				conversations.GET("/:id/messages", h.GetConversationMessages)
				conversations.POST("/:id/messages", h.SendMessage)

				// Conversation actions
				conversations.POST("/:id/archive", h.ArchiveConversation)
				conversations.POST("/:id/unarchive", h.UnarchiveConversation)
				conversations.POST("/:id/generate-title", h.GenerateConversationTitle)

				// Analytics and management
				conversations.GET("/statistics", h.GetConversationStatistics)
				conversations.POST("/cleanup", h.CleanupConversations)
			}

			// Events and Server-Sent Events (SSE) endpoints
			events := protected.Group("/events")
			{
				events.GET("/status", h.GetEventStatus)
			}

			// MCP (Model Context Protocol) endpoints
			mcp := protected.Group("/mcp")
			{
				mcp.GET("/status", h.GetMCPStatus)
				mcp.GET("/servers", h.GetMCPServers)
				mcp.POST("/servers", h.AddMCPServer)
				mcp.DELETE("/servers/:serverId", h.RemoveMCPServer)
				mcp.GET("/servers/:serverId/connect", h.ConnectMCPServer)
				mcp.GET("/servers/:serverId/disconnect", h.DisconnectMCPServer)
				mcp.GET("/tools", h.GetMCPTools)
				mcp.POST("/tools/execute", h.ExecuteMCPTools)
			}

			// File upload and screensaver management endpoints
			screensaver := protected.Group("/screensaver")
			{
				screensaver.GET("/images", h.GetScreensaverImages)
				screensaver.GET("/storage", h.GetScreensaverStorage)
				screensaver.POST("/images/upload", h.UploadScreensaverImages)
				screensaver.DELETE("/images/:id", h.DeleteScreensaverImage)
			}
		}

		// Legacy display settings endpoints for backward compatibility
		api.GET("/display-settings", h.GetDisplaySettingsLegacy)
		api.POST("/display-settings", middleware.AuthMiddleware(cfg.Auth.JWTSecret), h.UpdateDisplaySettingsLegacy)
		// Note: /display-settings/capabilities removed due to conflict with protected route
		// Note: /display-settings/wake removed due to conflict with protected route
		// Note: /display-settings/hardware removed due to conflict with protected route

		// Legacy scene endpoints for backward compatibility
		// Note: All scene endpoints removed due to conflicts with protected routes
	}

	// Legacy API routes without v1 prefix for frontend compatibility
	legacyAPI := router.Group("/api")
	{
		logger.Info("DEBUG: Setting up legacy API routes")

		// Legacy auth routes (public)
		legacyAuth := legacyAPI.Group("/auth")
		{
			legacyAuth.POST("/register", h.Register)
			legacyAuth.POST("/login", h.Login)
			legacyAuth.POST("/validate", h.ValidateToken)
			legacyAuth.POST("/verify-pin", h.VerifyPin)
			legacyAuth.POST("/set-pin", h.SetPin)
			legacyAuth.GET("/pin-status", h.GetPinStatus)
			legacyAuth.GET("/session", h.GetSession)
		}

		// Legacy public routes (no auth required)
		legacyAPI.GET("/status", h.Health)
		legacyAPI.GET("/health", h.Health) // Health endpoint alias
		legacyAPI.GET("/events/stream", h.GetEventStream)
		legacyAPI.GET("/screensaver/images", h.GetScreensaverImages)
		legacyAPI.GET("/screensaver/images/:filename", h.GetScreensaverImage)

		// Legacy display settings routes
		legacyAPI.GET("/display-settings", h.GetDisplaySettingsLegacy)
		legacyAPI.POST("/display-settings", middleware.AuthMiddleware(cfg.Auth.JWTSecret), h.UpdateDisplaySettingsLegacy)

		// Legacy protected routes (optional auth based on configuration)
		legacyProtected := legacyAPI.Group("/")
		legacyProtected.Use(middleware.OptionalAuthMiddleware(cfg, repos.Config, logger))
		{
			logger.Info("DEBUG: Registering legacy protected routes with OptionalAuthMiddleware")

			// System endpoints that were causing 401 errors
			legacyProtected.GET("/system/health/detailed", h.GetSystemHealth)
			legacyProtected.GET("/system/status", h.GetSystemStatus)
			legacyProtected.GET("/system/config", h.GetAllConfig)
			legacyProtected.POST("/system/config", h.SetConfig)

			// Additional system config alias
			legacyProtected.GET("/system/settings", h.GetAllConfig)

			// Configuration endpoints
			legacyProtected.GET("/config", h.GetAllConfig)
			legacyProtected.GET("/config/:key", h.GetConfig)
			legacyProtected.PUT("/config/:key", h.SetConfig)

			// Settings endpoints that frontend expects
			legacyProtected.GET("/settings/system", h.GetAllConfig)
			legacyProtected.GET("/settings/theme", h.GetConfig)

			logger.Info("DEBUG: Registered /api/settings/system and /api/settings/theme with OptionalAuthMiddleware")

			// Other commonly used endpoints (with and without trailing slashes to prevent redirects)
			legacyProtected.GET("/entities", h.GetEntities)
			legacyProtected.GET("/entities/", h.GetEntities)
			legacyProtected.GET("/scenes", h.GetScenes)
			legacyProtected.GET("/scenes/", h.GetScenes)
			legacyProtected.GET("/rooms", h.GetRooms)
			legacyProtected.GET("/rooms/", h.GetRooms)

			// Screensaver upload endpoints
			legacyProtected.POST("/screensaver/images/upload", h.UploadScreensaverImages)
			legacyProtected.DELETE("/screensaver/images/:id", h.DeleteScreensaverImage)
		}
	}

	return router
}
