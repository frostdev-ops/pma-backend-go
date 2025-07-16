package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pma/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// NewRouter creates and configures the main HTTP router
func NewRouter(cfg *config.Config, db *sql.DB, logger *logrus.Logger) *gin.Engine {
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

	// TODO: Add other route groups here
	// api/v1 routes will be added in subsequent implementations

	return router
}
