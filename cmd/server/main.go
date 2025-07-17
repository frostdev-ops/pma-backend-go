package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/api"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/entities"
	haSync "github.com/frostdev-ops/pma-backend-go/internal/core/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/core/rooms"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.New()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db, cfg.Database.MigrationsPath); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Create repositories
	repos := database.NewRepositories(db)

	// Create WebSocket hub
	wsHub := websocket.NewHub(log)
	go wsHub.Run()

	// Initialize core services
	entityService := entities.NewService(repos.Entity, repos.Room, log)
	roomService := rooms.NewService(repos.Room, repos.Entity, log)

	// Initialize Home Assistant services if enabled
	var haClient *homeassistant.Client
	var syncService *haSync.SyncService

	if cfg.HomeAssistant.URL != "" && cfg.HomeAssistant.Token != "" {
		log.Info("Initializing Home Assistant integration")

		// Create Home Assistant client
		var err error
		haClient, err = homeassistant.NewClient(cfg, repos.Config, log)
		if err != nil {
			log.WithError(err).Warn("Failed to create Home Assistant client")
		} else {
			// Initialize the client
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := haClient.Initialize(ctx); err != nil {
				log.WithError(err).Warn("Failed to initialize Home Assistant client")
			} else {
				log.Info("Home Assistant client initialized successfully")

				// Create sync service if enabled
				if cfg.HomeAssistant.Sync.Enabled {
					// Convert config to sync config
					syncConfig := &haSync.SyncConfig{
						Enabled:            cfg.HomeAssistant.Sync.Enabled,
						SupportedDomains:   cfg.HomeAssistant.Sync.SupportedDomains,
						ConflictResolution: cfg.HomeAssistant.Sync.ConflictResolution,
						BatchSize:          cfg.HomeAssistant.Sync.BatchSize,
						RetryAttempts:      cfg.HomeAssistant.Sync.RetryAttempts,
						EventBufferSize:    cfg.HomeAssistant.Sync.EventBufferSize,
					}

					// Parse duration strings
					if interval, err := time.ParseDuration(cfg.HomeAssistant.Sync.FullSyncInterval); err == nil {
						syncConfig.FullSyncInterval = interval
					} else {
						syncConfig.FullSyncInterval = time.Hour // default
					}

					if delay, err := time.ParseDuration(cfg.HomeAssistant.Sync.RetryDelay); err == nil {
						syncConfig.RetryDelay = delay
					} else {
						syncConfig.RetryDelay = 5 * time.Second // default
					}

					if procDelay, err := time.ParseDuration(cfg.HomeAssistant.Sync.EventProcessingDelay); err == nil {
						syncConfig.EventProcessingDelay = procDelay
					} else {
						syncConfig.EventProcessingDelay = 100 * time.Millisecond // default
					}

					// Create sync service
					syncService = haSync.NewSyncService(haClient, entityService, roomService, repos.Config, wsHub, log, syncConfig)

					// Start sync service
					go func() {
						if err := syncService.Start(context.Background()); err != nil {
							log.WithError(err).Error("Failed to start Home Assistant sync service")
						}
					}()

					log.Info("Home Assistant sync service started")
				}
			}
			cancel()
		}
	}

	// Create HA Event Forwarder
	haForwarder := websocket.NewHAEventForwarder(wsHub, log, nil)

	// Initialize router
	router := api.NewRouter(cfg, repos, log, wsHub, haForwarder)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Infof("Starting PMA Backend on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop Home Assistant sync service if running
	if syncService != nil {
		log.Info("Stopping Home Assistant sync service...")
		if err := syncService.Stop(ctx); err != nil {
			log.WithError(err).Warn("Failed to stop sync service gracefully")
		}
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Info("Server exited")
}
