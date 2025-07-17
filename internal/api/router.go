package api

import (
	"github.com/frostdev-ops/pma-backend-go/internal/api/handlers"
	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewRouter creates and configures the main HTTP router
func NewRouter(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger, wsHub *websocket.Hub, haForwarder *websocket.HAEventForwarder) *gin.Engine {
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
	h := handlers.NewHandlers(cfg, repos, logger, wsHub)

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
		}

		// Public API routes (no auth required)
		public := api.Group("/")
		{
			public.GET("/status", h.Health)
		}

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
					ha.GET("/stats", h.GetHAEventStats(haForwarder))
					ha.POST("/test", h.TestHAEventForwarding(haForwarder))
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

			// TODO: Add more protected endpoints here
			// automation := protected.Group("/automation")
		}
	}

	return router
}
