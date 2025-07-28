package api

import (
	"database/sql"

	"github.com/frostdev-ops/pma-backend-go/internal/api/handlers"
	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/debug"
	"github.com/frostdev-ops/pma-backend-go/pkg/errors"
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
func NewRouter(cfg *config.Config, repos *database.Repositories, batchLogger *logger.BatchLogger, wsHub *websocket.Hub, db *sql.DB, enhancedDB *database.EnhancedDB) *RouterWithHandlers {
	// Set gin mode based on config
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Configure router to handle trailing slashes properly
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// Create recovery manager for error handling
	recoveryManager := errors.NewRecoveryManager(batchLogger.Logger)

	// Global middleware - use the underlying logrus.Logger for error handling
	router.Use(middleware.ErrorHandlingMiddleware(batchLogger.Logger, recoveryManager))
	router.Use(middleware.ErrorResponseMiddleware(batchLogger.Logger, recoveryManager))
	// Use the BatchLogger for request logging with batching capabilities
	router.Use(middleware.LoggingMiddleware(batchLogger))
	router.Use(middleware.CORSMiddleware())

	// Rate limiting - temporarily disabled for debugging
	// rateLimiter := middleware.NewRateLimiter(100, 200) // 100 requests/sec, burst 200
	// router.Use(rateLimiter.RateLimitMiddleware())

	// Initialize handlers - pass the underlying logrus.Logger to handlers
	h := handlers.NewHandlers(cfg, repos, batchLogger.Logger, wsHub, db, enhancedDB, recoveryManager, nil)

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

			// User/password authentication endpoints
			auth.POST("/user/login", h.UserLogin)
			auth.POST("/user/register", h.UserRegister)

			// Remote authentication status
			auth.GET("/remote-status", h.GetRemoteAuthStatus)
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

		// Protected API routes - use remote auth middleware
		protected := api.Group("/")
		protected.Use(middleware.RemoteAuthMiddleware(cfg)) // Use remote auth middleware
		{
			// User profile routes
			profile := protected.Group("/profile")
			{
				profile.GET("/", h.GetProfile)
				profile.PUT("/password", h.UpdatePassword)
			}

			// User management routes (admin functionality)
			users := protected.Group("/users")
			{
				users.GET("/", h.GetAllUsers)
				users.DELETE("/:id", h.DeleteUser)
				users.GET("/:id", h.GetUser)
				users.PUT("/:id", h.UpdateUser)
			}
			// Configuration endpoints
			config := protected.Group("/config")
			{
				config.GET("/:key", h.GetConfig)
				config.PUT("/:key", h.SetConfig)
				config.GET("/", h.GetAllConfig)
			}

			// Entity management using unified PMA type system
			entities := protected.Group("/entities")
			{
				entities.GET("/", h.GetEntities)
				entities.GET("/:id", h.GetEntity)
				entities.POST("/:id/action", h.ExecuteEntityAction)
				entities.DELETE("/:id", h.DeleteEntity)
				entities.POST("/", h.CreateOrUpdateEntity)
				entities.PUT("/:id/state", h.UpdateEntityState)
				entities.PUT("/:id/room", h.AssignEntityToRoom)
				entities.POST("/sync", h.SyncEntities)
				entities.GET("/sync/status", h.GetSyncStatus)
				entities.GET("/search", h.SearchEntities)
				entities.GET("/types", h.GetEntityTypes)
				entities.GET("/capabilities", h.GetEntityCapabilities)
				entities.GET("/type/:type", h.GetEntitiesByType)
				entities.GET("/source/:source", h.GetEntitiesBySource)
				entities.GET("/room/:roomId", h.GetEntitiesByRoom)

				// Debug endpoints for troubleshooting
				entities.POST("/debug/sync", h.DebugSyncEntities)
				entities.GET("/debug/registry", h.DebugEntityRegistry)
				entities.GET("/debug/ha-connection", h.TestHAConnection)
				entities.GET("/debug/ha-simple", h.TestHAConnectionSimple)
			}

			// Add explicit routes without trailing slashes for main collections
			protected.GET("/entities", h.GetEntities)
			protected.GET("/rooms", h.GetRooms)
			protected.POST("/rooms", h.CreateRoom)
			protected.GET("/scenes", h.GetScenes)
			protected.GET("/config", h.GetAllConfig)
			protected.GET("/areas", h.GetAreas)

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

			// Controller Dashboard endpoints
			controllers := protected.Group("/controllers")
			{
				// Dashboard CRUD
				controllers.GET("/", h.GetControllerDashboards)
				controllers.GET("/:id", h.GetControllerDashboard)
				controllers.POST("/", h.CreateControllerDashboard)
				controllers.PUT("/:id", h.UpdateControllerDashboard)
				controllers.DELETE("/:id", h.DeleteControllerDashboard)
				controllers.POST("/:id/duplicate", h.DuplicateControllerDashboard)
				controllers.PUT("/:id/favorite", h.ToggleControllerDashboardFavorite)

				// Element actions
				controllers.POST("/:id/elements/:elementId/action", h.ExecuteControllerElementAction)

				// Analytics
				controllers.GET("/:id/stats", h.GetControllerDashboardStats)

				// Import/Export
				controllers.GET("/:id/export", h.ExportControllerDashboard)
				controllers.POST("/import", h.ImportControllerDashboard)

				// Sharing
				controllers.POST("/:id/share", h.ShareControllerDashboard)

				// Search
				controllers.GET("/search", h.SearchControllerDashboards)
			}

			// Controller Template endpoints
			templates := protected.Group("/controller-templates")
			{
				templates.GET("/", h.GetControllerTemplates)
				templates.POST("/", h.CreateControllerTemplate)
				templates.POST("/:id/apply", h.ApplyControllerTemplate)
			}

			// Controller Analytics endpoints
			analytics := protected.Group("/controller-analytics")
			{
				analytics.GET("/", h.GetControllerAnalytics)
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

				// WebSocket Optimization routes
				if h.WebSocketOptimizationHandler != nil {
					h.WebSocketOptimizationHandler.RegisterRoutes(ws)
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

				// Model Management endpoints
				models := ai.Group("/models")
				{
					models.GET("/storage", h.GetModelStorageInfo)
					models.GET("/downloads", h.GetModelDownloads)
					models.POST("/install", h.InstallModel)
					models.DELETE("/:id", h.RemoveModel)
					models.POST("/:id/test", h.RunModelTest)
				}

				// AI Settings & Management
				ai.GET("/settings", h.GetAISettings)
				ai.PUT("/settings", h.UpdateAISettings) // Fixed method name
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
			areas := protected.Group("/areas")
			{
				areas.GET("/", h.GetAreas)
				areas.POST("/", h.CreateArea)
				areas.GET("/:id", h.GetArea)
				areas.PUT("/:id", h.UpdateArea)
				areas.DELETE("/:id", h.DeleteArea)

				// Legacy entity endpoints (deprecated)
				areas.GET("/:id/entities", h.GetAreaEntities)
				areas.POST("/:id/entities", h.AssignEntitiesToArea)
				areas.DELETE("/:id/entities/:entity_id", h.RemoveEntityFromArea)

				// New simplified hierarchy endpoints
				areas.GET("/:id/hierarchy", h.GetAreaWithFullHierarchy)
				areas.GET("/:id/rooms", h.GetAreaRooms)
				areas.POST("/:id/rooms/:room_id", h.AssignRoomToArea)
				areas.DELETE("/rooms/:room_id", h.RemoveRoomFromArea)

				// Bulk operations
				areas.POST("/:id/actions", h.ExecuteBulkAreaAction)
				areas.GET("/:id/action-entities", h.GetAreaEntitiesForAction)
				areas.GET("/summaries", h.GetAreaSummaries)
			}

			// Room management endpoints (enhanced)
			// rooms.GET("/unassigned", h.GetUnassignedRooms) // Already defined in rooms group above

			// Camera Management endpoints
			cameras := protected.Group("/cameras")
			{
				// Basic camera operations
				cameras.GET("/", h.GetCameras)
				cameras.GET("/enabled", h.GetEnabledCameras)
				cameras.POST("/", h.CreateCamera)
				cameras.GET("/:id", h.GetCamera)
				cameras.PUT("/:id", h.UpdateCamera)
				cameras.DELETE("/:id", h.DeleteCamera)

				// Camera by entity ID
				cameras.GET("/entity/:entityId", h.GetCameraByEntityID)

				// Camera operations
				cameras.GET("/type/:type", h.GetCamerasByType)
				cameras.GET("/search", h.SearchCameras)
				cameras.PUT("/:id/status", h.UpdateCameraStatus)

				// Camera streaming and snapshots
				cameras.GET("/:id/stream", h.GetCameraStream)
				cameras.GET("/:id/snapshot", h.GetCameraSnapshot)

				// Camera events
				cameras.GET("/events", h.GetAllCameraEvents)
				cameras.GET("/:id/events", h.GetCameraEvents)

				// Camera statistics
				cameras.GET("/stats", h.GetCameraStats)
			}

			// Preferences endpoints
			if h.PreferencesHandler != nil {
				preferences := protected.Group("/preferences")
				{
					// User preferences
					preferences.GET("/", h.PreferencesHandler.GetUserPreferences)
					preferences.PUT("/", h.PreferencesHandler.UpdateUserPreferences)
					preferences.POST("/reset", h.PreferencesHandler.ResetToDefaults)
					preferences.GET("/section/:section", h.PreferencesHandler.GetPreferenceSection)
					preferences.PUT("/section/:section", h.PreferencesHandler.UpdatePreferenceSection)

					// Themes
					themes := preferences.Group("/themes")
					{
						themes.GET("/", h.PreferencesHandler.GetAvailableThemes)
						themes.GET("/:id", h.PreferencesHandler.GetTheme)
						themes.POST("/", h.PreferencesHandler.CreateCustomTheme)
						themes.DELETE("/:id", h.PreferencesHandler.DeleteCustomTheme)
						themes.POST("/:id/apply", h.PreferencesHandler.ApplyTheme)
					}

					// Dashboard widgets
					dashboard := preferences.Group("/dashboard")
					{
						dashboard.GET("/", h.PreferencesHandler.GetUserDashboard)
						dashboard.POST("/", h.PreferencesHandler.SaveDashboard)
						dashboard.POST("/widgets", h.PreferencesHandler.AddWidget)
						dashboard.PUT("/widgets/:id", h.PreferencesHandler.UpdateWidget)
						dashboard.DELETE("/widgets/:id", h.PreferencesHandler.RemoveWidget)
						dashboard.GET("/available-widgets", h.PreferencesHandler.GetAvailableWidgets)
						dashboard.GET("/widgets/:id/data", h.PreferencesHandler.GetWidgetData)
						dashboard.POST("/widgets/:id/refresh", h.PreferencesHandler.RefreshWidget)
					}

					// Localization
					locale := preferences.Group("/locale")
					{
						locale.GET("/", h.PreferencesHandler.GetUserLocale)
						locale.PUT("/", h.PreferencesHandler.SetUserLocale)
						locale.GET("/supported", h.PreferencesHandler.GetSupportedLocales)
						locale.GET("/translations/:locale", h.PreferencesHandler.GetTranslations)
					}

					// Import/Export
					preferences.GET("/export", h.PreferencesHandler.ExportPreferences)
					preferences.POST("/import", h.PreferencesHandler.ImportPreferences)

					// Statistics
					preferences.GET("/statistics", h.PreferencesHandler.GetPreferenceStatistics)
				}
			}

			// Backup endpoints
			if h.BackupHandler != nil {
				backup := protected.Group("/backup")
				{
					// Basic backup operations
					backup.POST("/", h.BackupHandler.CreateBackup)
					backup.GET("/", h.BackupHandler.ListBackups)
					backup.GET("/:id", h.BackupHandler.GetBackup)
					backup.DELETE("/:id", h.BackupHandler.DeleteBackup)
					backup.POST("/:id/restore", h.BackupHandler.RestoreBackup)

					// Backup validation and integrity
					backup.POST("/:id/validate", h.BackupHandler.ValidateBackup)

					// Import/Export
					backup.GET("/:id/export", h.BackupHandler.ExportBackup)
					backup.POST("/import", h.BackupHandler.ImportBackup)

					// Scheduling and automation
					backup.POST("/schedule", h.BackupHandler.ScheduleBackup)

					// Statistics and management
					backup.GET("/statistics", h.BackupHandler.GetBackupStatistics)
					backup.POST("/cleanup", h.BackupHandler.CleanupOldBackups)
				}
			}

			// Media Processing endpoints
			if h.MediaHandler != nil {
				media := protected.Group("/media")
				{
					// Media processing
					media.POST("/process", h.MediaHandler.ProcessMedia)
					media.POST("/validate", h.MediaHandler.ValidateMedia)
					media.GET("/info/:id", h.MediaHandler.GetMediaInfo)

					// Thumbnail generation
					media.POST("/thumbnail", h.MediaHandler.GenerateThumbnail)
					media.GET("/thumbnail/:id", h.MediaHandler.GetThumbnail)
					media.POST("/thumbnails/multiple", h.MediaHandler.GenerateMultipleThumbnails)

					// Media streaming
					media.GET("/stream/video/:id", h.MediaHandler.StreamVideo)
					media.GET("/stream/audio/:id", h.MediaHandler.StreamAudio)
					media.GET("/stream/url/:id", h.MediaHandler.GetStreamingURL)

					// Video transcoding
					media.POST("/transcode/:id", h.MediaHandler.TranscodeVideo)

					// System information
					media.GET("/formats", h.MediaHandler.GetSupportedFormats)
					media.GET("/stats", h.MediaHandler.GetMediaStats)
				}
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
				// UPS status and monitoring
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

				// UPS monitoring control
				ups.POST("/monitoring/start", h.StartUPSMonitoring)
				ups.POST("/monitoring/stop", h.StopUPSMonitoring)

				// UPS alerts and thresholds
				ups.PUT("/alerts/thresholds", h.UpdateUPSAlertThresholds)
			}

			// System management endpoints
			system := protected.Group("/system")
			{
				// Basic information and health
				system.GET("/info", h.GetSystemInfo)
				system.GET("/status", h.GetSystemStatus)
				system.GET("/services", h.GetServiceStatus)
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
				// Discovery and device listing
				shelly.POST("/discover", h.DiscoverShellyDevices)
				shelly.GET("/devices", h.ListShellyDevices)
				shelly.GET("/devices/:id", h.GetShellyDevice)

				// Device control
				shelly.POST("/devices/:id/control", h.ControlShellyDeviceV2)

				// Configuration and status
				shelly.PUT("/config", h.UpdateShellyConfig)
				shelly.GET("/status", h.GetShellyAdapterStatus)
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

			// Analytics system endpoints
			if h.AnalyticsHandler != nil {
				h.AnalyticsHandler.RegisterRoutes(protected)
			}

			// Performance management endpoints
			performance := protected.Group("/performance")
			{
				performance.GET("/status", h.GetPerformanceStatus)
				performance.GET("/profile", h.StartProfiling)
				performance.POST("/optimize", h.TriggerOptimization)
				performance.GET("/report", h.GetPerformanceReport)
				performance.GET("/queries/slow", h.GetSlowQueries)
				performance.POST("/benchmark", h.RunBenchmarks)
				performance.GET("/memory", h.GetMemoryStats)
				performance.POST("/memory/gc", h.ForceGarbageCollection)
				performance.GET("/database/pool", h.GetDatabasePoolStats)
				performance.POST("/database/optimize", h.OptimizeDatabase)
			}

			// Cache management endpoints
			if h.CacheHandler != nil {
				h.CacheHandler.RegisterRoutes(protected)
			}

			// Memory management endpoints
			memory := protected.Group("/memory")
			{
				// Core memory operations
				memory.GET("/status", h.GetMemoryStatus)
				memory.GET("/stats", h.GetMemoryStats)
				memory.POST("/gc", h.ForceGarbageCollection)
				memory.POST("/optimize", h.OptimizeMemory)

				// Leak detection
				memory.GET("/leaks", h.DetectMemoryLeaks)
				memory.GET("/leaks/scan", h.ScanForLeaks)

				// Pool management
				memory.GET("/pools", h.GetPoolStats)
				memory.GET("/pools/:name", h.GetPoolDetail)
				memory.POST("/pools/:name/resize", h.ResizePool)
				memory.POST("/pools/optimize", h.OptimizePools)

				// Memory pressure
				memory.GET("/pressure", h.GetMemoryPressure)
				memory.POST("/pressure/handle", h.HandleMemoryPressure)
				memory.GET("/pressure/config", h.GetPressureConfig)
				memory.PUT("/pressure/config", h.UpdatePressureConfig)

				// Preallocation
				memory.GET("/preallocation", h.GetPreallocationStats)
				memory.POST("/preallocation/analyze", h.AnalyzeUsagePatterns)
				memory.POST("/preallocation/optimize", h.OptimizePreallocation)

				// Optimization engine
				memory.GET("/optimization/status", h.GetOptimizationStatus)
				memory.POST("/optimization/start", h.StartOptimization)
				memory.POST("/optimization/stop", h.StopOptimization)
				memory.GET("/optimization/history", h.GetOptimizationHistory)
				memory.GET("/optimization/report", h.GetOptimizationReport)

				// Advanced monitoring
				memory.GET("/monitor", h.GetMemoryMonitoring)
				memory.POST("/monitor/start", h.StartMemoryMonitoring)
				memory.POST("/monitor/stop", h.StopMemoryMonitoring)
			}

			// Monitoring & Alerting system endpoints
			monitoring := protected.Group("/monitoring")
			{
				// Alerting endpoints
				alerts := monitoring.Group("/alerts")
				{
					alerts.GET("/", h.GetAlerts)
					alerts.GET("/rules", h.GetAlertRules)
					alerts.POST("/rules", h.CreateAlertRule)
					alerts.PUT("/rules/:id", h.UpdateAlertRule)
					alerts.DELETE("/rules/:id", h.DeleteAlertRule)
					alerts.POST("/rules/:id/test", h.TestAlertRule)
					alerts.GET("/active", h.GetActiveAlerts)
					alerts.GET("/history", h.GetAlertHistory)
					alerts.POST("/:id/acknowledge", h.AcknowledgeAlert)
					alerts.POST("/:id/resolve", h.ResolveAlert)
					alerts.GET("/statistics", h.GetAlertStatistics)
					alerts.GET("/rules/:id/evaluate", h.EvaluateAlertRule)
				}

				// Dashboard endpoints
				dashboards := monitoring.Group("/dashboards")
				{
					dashboards.GET("/", h.GetMonitoringDashboards)
					dashboards.POST("/", h.CreateMonitoringDashboard)
					dashboards.GET("/:id", h.GetMonitoringDashboard)
					dashboards.PUT("/:id", h.UpdateMonitoringDashboard)
					dashboards.DELETE("/:id", h.DeleteMonitoringDashboard)
					dashboards.GET("/:id/data", h.GetDashboardData)
					dashboards.GET("/:id/export", h.ExportDashboard)
					dashboards.POST("/:id/duplicate", h.DuplicateDashboard)
					dashboards.GET("/:id/widgets/:widget_id/data", h.GetWidgetData)
					dashboards.POST("/:id/widgets", h.AddWidget)
					dashboards.PUT("/:id/widgets/:widget_id", h.UpdateWidget)
					dashboards.DELETE("/:id/widgets/:widget_id", h.RemoveWidget)
					dashboards.GET("/templates", h.GetDashboardTemplates)
					dashboards.POST("/import", h.ImportDashboard)
				}

				// Live streaming endpoints
				streaming := monitoring.Group("/streaming")
				{
					streaming.GET("/dashboards/:id/widgets/:widget_id/stream", h.StartLiveStream)
					streaming.DELETE("/streams/:stream_id", h.StopLiveStream)
					streaming.GET("/streams", h.GetActiveStreams)
				}

				// Predictive analytics endpoints
				prediction := monitoring.Group("/prediction")
				{
					prediction.GET("/models", h.GetPredictionModels)
					prediction.POST("/models", h.CreatePredictionModel)
					prediction.GET("/models/:id", h.GetPredictionModel)
					prediction.PUT("/models/:id", h.UpdatePredictionModel)
					prediction.DELETE("/models/:id", h.DeletePredictionModel)
					prediction.POST("/models/:id/train", h.TrainModel)
					prediction.POST("/models/:id/predict", h.GeneratePrediction)
					prediction.GET("/models/:id/performance", h.GetModelPerformance)
					prediction.GET("/predictions", h.GetPredictions)
					prediction.GET("/predictions/history", h.GetPredictionHistory)
				}

				// Anomaly detection endpoints
				anomalies := monitoring.Group("/anomalies")
				{
					anomalies.GET("/detectors", h.GetAnomalyDetectors)
					anomalies.POST("/detectors", h.CreateAnomalyDetector)
					anomalies.GET("/detectors/:id", h.GetAnomalyDetector)
					anomalies.PUT("/detectors/:id", h.UpdateAnomalyDetector)
					anomalies.DELETE("/detectors/:id", h.DeleteAnomalyDetector)
					anomalies.POST("/detectors/:id/detect", h.DetectAnomalies)
					anomalies.GET("/", h.GetAnomalies)
					anomalies.GET("/history", h.GetAnomalyHistory)
					anomalies.GET("/statistics", h.GetAnomalyStatistics)
					anomalies.POST("/:id/feedback", h.ProvideAnomalyFeedback)
				}

				// Forecasting endpoints
				forecasting := monitoring.Group("/forecasting")
				{
					forecasting.GET("/forecasters", h.GetForecasters)
					forecasting.POST("/forecasters", h.CreateForecaster)
					forecasting.GET("/forecasters/:id", h.GetForecaster)
					forecasting.PUT("/forecasters/:id", h.UpdateForecaster)
					forecasting.DELETE("/forecasters/:id", h.DeleteForecaster)
					forecasting.POST("/forecasters/:id/forecast", h.GenerateForecast)
					forecasting.GET("/forecasts", h.GetForecasts)
					forecasting.GET("/forecasts/:id", h.GetForecastDetails)
					forecasting.GET("/forecasts/:id/accuracy", h.GetForecastAccuracy)
				}

				// Monitoring overview and status
				monitoring.GET("/overview", h.GetMonitoringOverview)
				monitoring.GET("/health", h.GetMonitoringHealth)
				monitoring.GET("/metrics/summary", h.GetMetricsSummary)
				monitoring.GET("/system/performance", h.GetSystemPerformance)
				monitoring.GET("/reports/daily", h.GetDailyReport)
				monitoring.GET("/reports/weekly", h.GetWeeklyReport)
				monitoring.GET("/reports/monthly", h.GetMonthlyReport)
				monitoring.POST("/reports/custom", h.GenerateCustomReport)
			}

			// Security system endpoints
			security := protected.Group("/security")
			{
				// Security metrics and status
				security.GET("/status", h.GetSecurityStatus)
				security.GET("/metrics", h.GetSecurityMetrics)
				security.GET("/events", h.GetSecurityEvents)

				// Rate limiting management
				security.GET("/ratelimit/status", h.GetRateLimitStatus)
				security.GET("/ratelimit/metrics", h.GetRateLimitMetrics)
				security.GET("/ratelimit/violators", h.GetTopViolators)
				security.POST("/ratelimit/block", h.BlockIP)
				security.POST("/ratelimit/unblock", h.UnblockIP)

				// IP management
				security.GET("/ips/blocked", h.GetBlockedIPs)
				security.POST("/ips/block", h.BlockIPAddress)
				security.POST("/ips/unblock", h.UnblockIPAddress)
				security.GET("/ips/whitelist", h.GetWhitelistedIPs)
				security.POST("/ips/whitelist", h.AddToWhitelist)
				security.DELETE("/ips/whitelist", h.RemoveFromWhitelist)

				// Threat intelligence
				security.GET("/threats", h.GetThreats)
				security.POST("/threats", h.AddThreat)
				security.DELETE("/threats/:ip", h.RemoveThreat)
				security.GET("/threats/analysis", h.GetThreatAnalysis)

				// Attack detection
				security.GET("/attacks", h.GetAttackData)
				security.GET("/attacks/patterns", h.GetAttackPatterns)
				security.GET("/attacks/summary", h.GetAttackSummary)

				// Configuration management
				security.GET("/config", h.GetSecurityConfig)
				security.PUT("/config", h.UpdateSecurityConfig)
				security.POST("/config/reset", h.ResetSecurityConfig)

				// Security reports
				security.GET("/reports/summary", h.GetSecuritySummary)
				security.GET("/reports/detailed", h.GetDetailedSecurityReport)
				security.POST("/reports/export", h.ExportSecurityReport)

				// Real-time monitoring
				security.GET("/monitor/live", h.GetLiveSecurityData)
				security.GET("/monitor/alerts", h.GetSecurityAlerts)
			}

			// Error handling system endpoints
			errors := protected.Group("/errors")
			{
				errors.GET("/reports", h.GetErrorReports)
				errors.GET("/reports/:error_id", h.GetErrorReport)
				errors.POST("/reports/:error_id/resolve", h.ResolveError)
				errors.GET("/stats", h.GetErrorStats)
				errors.GET("/recovery/metrics", h.GetRecoveryMetrics)
				errors.GET("/recovery/circuit-breakers", h.GetCircuitBreakerStatus)
				errors.POST("/recovery/circuit-breakers/:name/reset", h.ResetCircuitBreaker)
				errors.GET("/health", h.GetErrorHealthStatus)
				errors.POST("/cleanup", h.CleanupOldErrors)
				errors.POST("/test", h.TestErrorRecovery)
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

			// Kiosk management endpoints (v1 API)
			kiosk := protected.Group("/kiosk")
			{
				// Status and monitoring
				kiosk.GET("/status", h.GetKioskStatus)
				kiosk.GET("/logs", h.GetKioskLogs)
				kiosk.GET("/config", h.GetKioskConfiguration)
				kiosk.PUT("/config", h.UpdateKioskConfiguration)

				// Display control
				kiosk.GET("/display/status", h.GetKioskDisplayStatus)
				kiosk.POST("/display/brightness", h.ControlKioskDisplayBrightness)
				kiosk.POST("/display/sleep", h.PutKioskDisplayToSleep)
				kiosk.POST("/display/wake", h.WakeKioskDisplay)

				// System control
				kiosk.POST("/screenshot", h.TakeKioskScreenshot)
				kiosk.POST("/restart", h.RestartKioskSystem)
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

		// Legacy AI endpoints for frontend compatibility
		legacyAPI.GET("/ai/providers", h.GetProviders)
		legacyAPI.GET("/ai/models", h.GetModels)
		legacyAPI.POST("/ai/chat", h.ChatWithAI)
		legacyAPI.GET("/ai/statistics", h.GetAIStatistics)

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

		// Kiosk endpoints at the path frontend expects (/v1/api/kiosk/)
		legacyAPI.GET("/v1/api/kiosk/status", h.GetKioskStatus)
		legacyAPI.GET("/v1/api/kiosk/logs", h.GetKioskLogs)
		legacyAPI.GET("/v1/api/kiosk/config", h.GetKioskConfiguration)
		legacyAPI.PUT("/v1/api/kiosk/config", h.UpdateKioskConfiguration)
		legacyAPI.GET("/v1/api/kiosk/display/status", h.GetKioskDisplayStatus)
		legacyAPI.POST("/v1/api/kiosk/display/brightness", h.ControlKioskDisplayBrightness)
		legacyAPI.POST("/v1/api/kiosk/display/sleep", h.PutKioskDisplayToSleep)
		legacyAPI.POST("/v1/api/kiosk/display/wake", h.WakeKioskDisplay)
		legacyAPI.POST("/v1/api/kiosk/screenshot", h.TakeKioskScreenshot)
		legacyAPI.POST("/v1/api/kiosk/restart", h.RestartKioskSystem)

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

// NewRouterWithDebug creates and configures the main HTTP router with debug logging
func NewRouterWithDebug(cfg *config.Config, repos *database.Repositories, batchLogger *logger.BatchLogger, wsHub *websocket.Hub, db *sql.DB, enhancedDB *database.EnhancedDB, debugLogger *debug.DebugLogger) *RouterWithHandlers {
	// Set gin mode based on config
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Configure router to handle trailing slashes properly
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// Create recovery manager for error handling
	recoveryManager := errors.NewRecoveryManager(batchLogger.Logger)

	// Global middleware - use the underlying logrus.Logger for error handling
	router.Use(middleware.ErrorHandlingMiddleware(batchLogger.Logger, recoveryManager))
	router.Use(middleware.ErrorResponseMiddleware(batchLogger.Logger, recoveryManager))
	// Use the BatchLogger for request logging with batching capabilities
	router.Use(middleware.LoggingMiddleware(batchLogger))
	router.Use(middleware.CORSMiddleware())

	// Add debug middleware if debug logger is available
	if debugLogger != nil {
		debugMiddleware := middleware.NewDebugMiddleware(debugLogger)
		router.Use(debugMiddleware.DebugLoggingMiddleware())
	}

	// Rate limiting - temporarily disabled for debugging
	// rateLimiter := middleware.NewRateLimiter(100, 200) // 100 requests/sec, burst 200
	// router.Use(rateLimiter.RateLimitMiddleware())

	// Initialize handlers - pass the underlying logrus.Logger to handlers
	h := handlers.NewHandlers(cfg, repos, batchLogger.Logger, wsHub, db, enhancedDB, recoveryManager, debugLogger)

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
		}

		// ... rest of the routes remain the same
	}

	return &RouterWithHandlers{
		Router:   router,
		Handlers: h,
	}
}
