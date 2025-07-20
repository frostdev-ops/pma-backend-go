package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/api"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/git"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/logger"
)

var (
	versionTemp, _ = git.GetVersion()
	version        = func() string {
		if versionTemp == "" {
			return "1.0"
		}
		return versionTemp
	}()
	buildTime = time.Now().Format("January 2, 2006 at 3:04:05 PM MST")
	gitCommit = func() string {
		commit, err := git.GetCurrentCommitMessage()
		if err != nil || commit == "" {
			return "Not Found"
		}
		return commit
	}()
)

func main() {
	// Define flags
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
		configFile  = flag.String("config", "", "Path to configuration file")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("PMA Backend Go\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		fmt.Printf("Status: Authentication middleware completely disabled\n")
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		fmt.Printf("PMA Backend Go - Personal Management Assistant Backend\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nAuthentication Status: DISABLED - All auth middleware removed\n")
		os.Exit(0)
	}

	// Initialize logger
	log := logger.New()

	// Log version information at startup
	log.Infof("Starting PMA Backend Go v%s (build: %s, commit: %s)", version, buildTime, gitCommit)
	log.Info("STATUS: Authentication middleware completely disabled")

	// Load configuration
	var cfg *config.Config
	var err error

	if *configFile != "" {
		log.Infof("Loading configuration from: %s", *configFile)
		// Note: config.Load() will use the file specified in VIPER_CONFIG_FILE or default
		cfg, err = config.Load()
	} else {
		cfg, err = config.Load()
	}
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
	if cfg.Database.Migration.Enabled {
		if err := database.Migrate(db, cfg.Database.MigrationsPath); err != nil {
			log.Fatal("Failed to run migrations:", err)
		}
	} else {
		log.Info("Database migrations disabled in configuration")
	}

	// Create repositories
	repos := database.NewRepositories(db)

	// Initialize authentication service if enabled
	if cfg.Auth.Enabled {
		authConfig := auth.AuthConfig{
			SessionTimeout:    cfg.Auth.TokenExpiry,
			MaxFailedAttempts: 3,
			LockoutDuration:   300,
			JWTSecret:         cfg.Auth.JWTSecret,
		}
		authService := auth.NewService(repos.Auth, authConfig, log.Logger)

		// Initialize auth service (creates default settings if none exist)
		if err := authService.Initialize(context.Background()); err != nil {
			log.WithError(err).Warn("Failed to initialize auth service")
		} else {
			log.Info("Authentication service initialized successfully")
		}
	}

	// Create WebSocket hub
	wsHub := websocket.NewHub(log.Logger)
	go wsHub.Run()

	// Initialize router with handlers
	routerWithHandlers := api.NewRouter(cfg, repos, log, wsHub, db)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      routerWithHandlers.Router,
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

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Stop periodic sync scheduler if running
	if routerWithHandlers.Handlers != nil && routerWithHandlers.Handlers.GetUnifiedService() != nil {
		log.Info("Stopping periodic sync scheduler...")
		routerWithHandlers.Handlers.GetUnifiedService().StopPeriodicSync()
	}

	// Stop automation engine if running
	if routerWithHandlers.Handlers != nil && routerWithHandlers.Handlers.GetAutomationEngine() != nil {
		log.Info("Stopping automation engine...")
		routerWithHandlers.Handlers.GetAutomationEngine().Stop()
	}

	log.Info("Stopping services...")

	// Create a deadline to wait for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}
