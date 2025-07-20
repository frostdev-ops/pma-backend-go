package api

import (
	"database/sql"

	"github.com/frostdev-ops/pma-backend-go/internal/api/handlers"
	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/logger"
	"github.com/gin-gonic/gin"
)

// RouterWithHandlers contains the router and handlers for service access
type RouterWithHandlers struct {
	Router   *gin.Engine
	Handlers *handlers.Handlers
}

// NewRouter creates and configures the main HTTP router with enhanced error handling
//
// Enhanced Error Handling Features:
// - Custom 404 handler for non-existent endpoints with helpful suggestions
// - Custom 405 handler for unsupported HTTP methods
// - Enhanced error middleware with detailed logging and request context
// - Comprehensive error responses with troubleshooting information
// - Stack traces in development mode for debugging
// - Automatic endpoint suggestions for common 404 errors
//
// Error Response Format:
//
//	{
//	  "success": false,
//	  "error": "Error message",
//	  "code": 404,
//	  "timestamp": "2024-01-01T00:00:00Z",
//	  "path": "/invalid-endpoint",
//	  "method": "GET",
//	  "suggestions": ["Similar endpoints that might help"]
//	}
//	}
func NewRouter(cfg *config.Config, repos *database.Repositories, batchLogger *logger.BatchLogger, wsHub *websocket.Hub, db *sql.DB) *RouterWithHandlers {
	// Set gin mode based on config
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Configure router to not redirect trailing slashes
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// Global middleware - use the underlying logrus.Logger for error handling
	router.Use(middleware.ErrorHandlingMiddleware(batchLogger.Logger))
	router.Use(middleware.AppErrorMiddleware(batchLogger.Logger))
	// Use the BatchLogger for request logging with batching capabilities
	router.Use(middleware.LoggingMiddleware(batchLogger))
	router.Use(middleware.CORSMiddleware())

	// Rate limiting - temporarily disabled for debugging
	// rateLimiter := middleware.NewRateLimiter(100, 200) // 100 requests/sec, burst 200
	// router.Use(rateLimiter.RateLimitMiddleware())

	// Initialize handlers - pass the underlying logrus.Logger to handlers
	h := handlers.NewHandlers(cfg, repos, batchLogger.Logger, wsHub, db)

	// Handle non-existent routes
	router.NoRoute(h.HandleNotFound)
	router.NoMethod(h.HandleMethodNotAllowed)

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
			// Legacy endpoints (keep for backward compatibility)
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/validate", h.ValidateToken)

			// Frontend-compatible PIN authentication endpoints
			auth.POST("/verify-pin", h.VerifyPinV2)
			auth.POST("/set-pin", h.SetPinV2)
			auth.POST("/change-pin", h.ChangePinV2)
			auth.POST("/disable-pin", h.DisablePinV2)
			auth.GET("/pin-status", h.GetPinStatusV2)
			auth.GET("/session", h.GetSessionV2)
			auth.POST("/logout", h.LogoutV2)
		}

		// Public API routes (no auth required)
		public := api.Group("/")
		{
			public.GET("/status", h.Health)

			// SSE stream endpoint (public for real-time updates) - needs special CORS handling
			public.GET("/events/stream", middleware.CORSMiddlewareSSE(), h.GetEventStream)

			// Image serving endpoint (public for screensaver display)
			public.GET("/screensaver/images/:filename", h.GetScreensaverImage)
		}

		// Mobile upload page (public)
		router.GET("/upload", h.GetMobileUploadPage)

		// Protected API routes (auth removed - now public)
		formerly_protected := api.Group("/")
		// protected.Use(middleware.AuthMiddleware(cfg.Auth.JWTSecret)) // REMOVED: No auth middleware
		{
			// User profile routes
			profile := formerly_protected.Group("/profile")
			{
				profile.GET("/", h.GetProfile)
				profile.PUT("/password", h.UpdatePassword)
			}

			// User management routes (admin functionality, now public)
			users := formerly_protected.Group("/users")
			{
				users.GET("/", h.GetAllUsers)
				users.DELETE("/:id", h.DeleteUser)
			}
			// Configuration endpoints
			config := formerly_protected.Group("/config")
			{
				config.GET("/:key", h.GetConfig)
				config.PUT("/:key", h.SetConfig)
				config.GET("/", h.GetAllConfig)
			}

			// Entity endpoints
			entities := formerly_protected.Group("/entities")
			{
				entities.GET("/", h.GetEntities)
				entities.GET("/:id", h.GetEntity)
				entities.POST("/", h.CreateOrUpdateEntity)
				entities.PUT("/:id/state", h.UpdateEntityState)
				entities.PUT("/:id/room", h.AssignEntityToRoom)
				entities.DELETE("/:id", h.DeleteEntity)
				entities.POST("/sync", h.SyncEntities)
				entities.GET("/sync/status", h.GetSyncStatus)
			}

			// Room endpoints
			rooms := formerly_protected.Group("/rooms")
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
			scenes := formerly_protected.Group("/scenes")
			{
				scenes.GET("/", h.GetScenes)
				scenes.GET("/:id", h.GetScene)
				scenes.POST("/:id/activate", h.ActivateScene)
			}

			// WebSocket management endpoints (protected)
			ws := formerly_protected.Group("/websocket")
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
			ai := formerly_protected.Group("/ai")
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

				// AI Settings & Management
				ai.GET("/settings", h.GetAISettings)
				ai.POST("/settings", h.SaveAISettings)
				ai.POST("/test-connection", h.TestAIConnection)

				// Ollama Process Management
				ollama := ai.Group("/ollama")
				{
					ollama.GET("/status", h.GetOllamaStatus)
					ollama.GET("/metrics", h.GetOllamaMetrics)
					ollama.GET("/health", h.GetOllamaHealth)
					ollama.POST("/start", h.StartOllamaProcess)
					ollama.POST("/stop", h.StopOllamaProcess)
					ollama.POST("/restart", h.RestartOllamaProcess)
					ollama.GET("/monitoring", h.GetOllamaMonitoring)
				}

				// MCP Server Management
				mcp := ai.Group("/mcp")
				{
					mcp.GET("/servers", h.GetMCPServers)
					mcp.POST("/servers", h.AddMCPServer)
					mcp.DELETE("/servers/:id", h.RemoveMCPServer)
					mcp.POST("/servers/:id/restart", h.RestartMCPServer)
				}
			}

			// Automation endpoints
			automation := formerly_protected.Group("/automation")
			{
				automation.GET("/rules", h.GetAutomations)
				automation.GET("/rules/:id", h.GetAutomation)
				automation.POST("/rules", h.CreateAutomation)
				automation.PUT("/rules/:id", h.UpdateAutomation)
				automation.DELETE("/rules/:id", h.DeleteAutomation)
				automation.POST("/rules/:id/enable", h.EnableAutomation)
				automation.POST("/rules/:id/disable", h.DisableAutomation)
				automation.POST("/rules/:id/test", h.TestAutomation)
				automation.POST("/rules/:id/trigger", h.TriggerAutomationRule)
				automation.POST("/rules/import", h.ImportAutomations)
				automation.GET("/rules/export", h.ExportAutomations)
				automation.POST("/rules/validate", h.ValidateAutomation)
				automation.GET("/statistics", h.GetAutomationStatistics)
				automation.GET("/templates", h.GetAutomationTemplates)
				automation.GET("/history", h.GetAutomationHistory)
				automation.GET("/stats", h.GetAutomationStats)
			}

			// Area Management endpoints
			areas := formerly_protected.Group("/areas")
			{
				areas.GET("/", h.GetAreas)
				areas.POST("/", h.CreateArea)
				areas.GET("/:id", h.GetArea)
				areas.PUT("/:id", h.UpdateArea)
				areas.DELETE("/:id", h.DeleteArea)
				areas.GET("/:id/entities", h.GetAreaEntities)
				areas.POST("/:id/entities", h.AssignEntitiesToArea)
				areas.DELETE("/:id/entities/:entity_id", h.RemoveEntityFromArea)
			}

			// Network management endpoints
			network := formerly_protected.Group("/network")
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
			ups := formerly_protected.Group("/ups")
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
			system := formerly_protected.Group("/system")
			{
				// Basic information and health
				system.GET("/info", h.GetSystemInfo)
				system.GET("/status", h.GetSystemStatus)
				system.GET("/health", h.GetBasicSystemHealth)
				system.GET("/health/detailed", h.GetSystemHealth)
				system.GET("/device-info", h.GetDeviceInfo)
				system.GET("/metrics", h.GetSystemMetrics)

				// System logs
				system.GET("/logs", h.GetSystemLogs)

				// Power management
				system.POST("/reboot", h.RebootSystem)
				system.POST("/shutdown", h.ShutdownSystem)

				// Configuration
				system.GET("/config", h.GetSystemConfig)
				system.POST("/config", h.UpdateSystemConfig)

				// Error tracking
				system.GET("/errors", h.GetErrorHistory)
				system.DELETE("/errors", h.ClearErrorHistory)

				// Health reporting (for external monitoring)
				system.POST("/health-report", h.ReportHealth)
			}

			// Display settings endpoints
			display := formerly_protected.Group("/display-settings")
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
			bluetooth := formerly_protected.Group("/bluetooth")
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
			energy := formerly_protected.Group("/energy")
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
			ring := formerly_protected.Group("/ring")
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
			shelly := formerly_protected.Group("/shelly")
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
			conversations := formerly_protected.Group("/conversations")
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
			events := formerly_protected.Group("/events")
			{
				events.GET("/status", h.GetEventStatus)
			}

			// MCP (Model Context Protocol) endpoints
			mcp := formerly_protected.Group("/mcp")
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
			screensaver := formerly_protected.Group("/screensaver")
			{
				screensaver.GET("/images", h.GetScreensaverImages)
				screensaver.GET("/storage", h.GetScreensaverStorage)
				screensaver.POST("/images/upload", h.UploadScreensaverImages)
				screensaver.DELETE("/images/:id", h.DeleteScreensaverImage)
			}
		}

		// Legacy display settings endpoints for backward compatibility
		api.GET("/display-settings", h.GetDisplaySettingsLegacy)
		api.POST("/display-settings", h.UpdateDisplaySettingsLegacy)
		// Note: /display-settings/capabilities removed due to conflict with protected route
		// Note: /display-settings/wake removed due to conflict with protected route
		// Note: /display-settings/hardware removed due to conflict with protected route

		// Legacy scene endpoints for backward compatibility
		// Note: All scene endpoints removed due to conflicts with protected routes
	}

	// Legacy API routes without v1 prefix for frontend compatibility
	legacyAPI := router.Group("/api")
	{
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
		legacyAPI.GET("/events/stream", middleware.CORSMiddlewareSSE(), h.GetEventStream)
		legacyAPI.GET("/screensaver/images", h.GetScreensaverImages)
		legacyAPI.GET("/screensaver/images/:filename", h.GetScreensaverImage)

		// Settings endpoints that frontend expects (moved to public for compatibility)
		legacyAPI.GET("/settings/system", h.GetAllConfig)
		legacyAPI.GET("/settings/theme", h.GetConfig)

		// Advanced system configuration endpoints
		legacyAPI.GET("/v1/settings/system", h.GetAdvancedSystemSettings)
		legacyAPI.PUT("/v1/settings/theme", h.UpdateThemeSettings)
		legacyAPI.GET("/v1/health", h.GetComprehensiveSystemHealth)

		// Unified sync service endpoints
		legacyAPI.GET("/v1/sync/status", h.GetSyncStatus)
		legacyAPI.POST("/v1/sync/trigger", h.TriggerSync)
		legacyAPI.GET("/v1/sync/history", h.GetSyncHistory)

		// Additional public endpoints for frontend compatibility (no auth whatsoever)
		legacyAPI.GET("/frontend/settings/system", h.GetAllConfig)
		legacyAPI.GET("/frontend/settings/theme", h.GetConfig)

		// Network settings endpoints
		legacyAPI.GET("/settings/network", h.GetNetworkSettings)
		legacyAPI.PUT("/settings/network", h.UpdateNetworkSettings)
		legacyAPI.POST("/settings/network/reset", h.ResetNetworkConfiguration)
		legacyAPI.GET("/network/router/test", h.TestRouterConnectivity)

		// Kiosk management endpoints
		legacyAPI.GET("/kiosk/status", h.GetKioskStatus)
		legacyAPI.POST("/kiosk/screenshot", h.TakeKioskScreenshot)
		legacyAPI.POST("/kiosk/restart", h.RestartKioskSystem)
		legacyAPI.GET("/kiosk/logs", h.GetKioskLogs)
		legacyAPI.GET("/kiosk/display/status", h.GetKioskDisplayStatus)
		legacyAPI.POST("/kiosk/display/brightness", h.ControlKioskDisplayBrightness)
		legacyAPI.POST("/kiosk/display/sleep", h.PutKioskDisplayToSleep)
		legacyAPI.POST("/kiosk/display/wake", h.WakeKioskDisplay)
		legacyAPI.GET("/kiosk/config", h.GetKioskConfiguration)
		legacyAPI.PUT("/kiosk/config", h.UpdateKioskConfiguration)

		// Legacy display settings routes (auth removed)
		legacyAPI.GET("/display-settings", h.GetDisplaySettingsLegacy)
		legacyAPI.POST("/display-settings", h.UpdateDisplaySettingsLegacy) // REMOVED: auth middleware

		// Legacy protected routes (auth completely removed - now public)
		// legacyProtected := legacyAPI.Group("/")
		// legacyProtected.Use(middleware.OptionalAuthMiddleware(cfg, repos.Config, logger)) // REMOVED: No auth middleware
		{
			// System endpoints that were causing 401 errors (now public)
			legacyAPI.GET("/system/health/detailed", h.GetSystemHealth)
			legacyAPI.GET("/system/status", h.GetSystemStatus)
			legacyAPI.GET("/system/metrics", h.GetSystemMetrics)
			legacyAPI.GET("/system/config", h.GetAllConfig)
			legacyAPI.POST("/system/config", h.SetConfig)

			// Additional system config alias
			legacyAPI.GET("/system/settings", h.GetAllConfig)

			// Configuration endpoints
			legacyAPI.GET("/config", h.GetAllConfig)
			legacyAPI.GET("/config/:key", h.GetConfig)
			legacyAPI.PUT("/config/:key", h.SetConfig)

			// Note: /settings/system and /settings/theme moved to public section for frontend compatibility

			// Other commonly used endpoints (with and without trailing slashes to prevent redirects)
			legacyAPI.GET("/entities", h.GetEntities)
			legacyAPI.GET("/entities/", h.GetEntities)
			legacyAPI.GET("/scenes", h.GetScenes)
			legacyAPI.GET("/scenes/", h.GetScenes)
			legacyAPI.GET("/rooms", h.GetRooms)
			legacyAPI.GET("/rooms/", h.GetRooms)

			// Screensaver upload endpoints
			legacyAPI.POST("/screensaver/images/upload", h.UploadScreensaverImages)
			legacyAPI.DELETE("/screensaver/images/:id", h.DeleteScreensaverImage)
		}
	}

	return &RouterWithHandlers{
		Router:   router,
		Handlers: h,
	}
}
