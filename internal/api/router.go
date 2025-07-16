package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/sirupsen/logrus"
)

// NewRouter creates and configures the main HTTP router
func NewRouter(cfg *config.Config, repos *database.Repositories, logger *logrus.Logger) *gin.Engine {
	// Set gin mode based on config
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "pma-backend",
		})
	})

	// Test endpoint to verify database connectivity
	router.GET("/api/test-db", func(c *gin.Context) {
		// Simple test to verify config repository works
		configs, err := repos.Config.GetAll(c.Request.Context())
		if err != nil {
			logger.WithError(err).Error("Failed to query database")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":       "ok",
			"message":      "Database connected successfully",
			"config_count": len(configs),
		})
	})

	// TODO: Add other route groups here
	// api/v1 routes will be added in subsequent implementations

	return router
}
