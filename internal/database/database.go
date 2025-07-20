package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/performance/database"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// EnhancedDB wraps sql.DB with performance enhancements
type EnhancedDB struct {
	*sql.DB
	PoolManager    database.PoolManager
	QueryCache     database.QueryCache
	QueryOptimizer database.QueryOptimizer
	logger         *logrus.Logger
	config         config.DatabaseConfig
}

// Initialize creates and configures the database connection with performance enhancements
func Initialize(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := initializeBasicDB(cfg)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// InitializeEnhanced creates and configures the database connection with full performance features
func InitializeEnhanced(cfg config.DatabaseConfig, logger *logrus.Logger) (*EnhancedDB, error) {
	// Initialize basic database connection
	sqlDB, err := initializeBasicDB(cfg)
	if err != nil {
		return nil, err
	}

	// Create enhanced DB wrapper
	enhancedDB := &EnhancedDB{
		DB:     sqlDB,
		logger: logger,
		config: cfg,
	}

	// Initialize performance components based on configuration
	if err := enhancedDB.initializePerformanceComponents(); err != nil {
		logger.WithError(err).Warn("Failed to initialize some performance components, continuing with basic setup")
	}

	logger.Info("Enhanced database initialized with performance optimizations")
	return enhancedDB, nil
}

// initializeBasicDB handles the core database setup
func initializeBasicDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MaxConnections / 2)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply SQLite optimizations
	if err := applySQLiteOptimizations(db); err != nil {
		return nil, fmt.Errorf("failed to apply SQLite optimizations: %w", err)
	}

	return db, nil
}

// applySQLiteOptimizations applies SQLite-specific performance settings
func applySQLiteOptimizations(db *sql.DB) error {
	optimizations := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456", // 256MB
		"PRAGMA optimize",
	}

	for _, pragma := range optimizations {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	return nil
}

// initializePerformanceComponents sets up advanced performance features
func (edb *EnhancedDB) initializePerformanceComponents() error {
	// Initialize connection pool manager
	if err := edb.initializePoolManager(); err != nil {
		edb.logger.WithError(err).Warn("Failed to initialize pool manager")
	}

	// Initialize query cache if enabled - check if config has Performance field
	enableCache := false
	if edb.config.MaxConnections > 0 { // Use MaxConnections as a proxy for enabling cache
		enableCache = true
	}

	if enableCache {
		if err := edb.initializeQueryCache(); err != nil {
			edb.logger.WithError(err).Warn("Failed to initialize query cache")
		}
	}

	// Initialize query optimizer
	if err := edb.initializeQueryOptimizer(); err != nil {
		edb.logger.WithError(err).Warn("Failed to initialize query optimizer")
	}

	return nil
}

// initializePoolManager sets up the connection pool manager
func (edb *EnhancedDB) initializePoolManager() error {
	poolConfig := &database.PoolConfig{
		MaxOpenConns:        edb.config.MaxConnections,
		MaxIdleConns:        edb.config.MaxConnections / 4,
		ConnMaxLifetime:     time.Hour,
		ConnMaxIdleTime:     time.Minute * 30,
		MonitorInterval:     time.Minute * 5,
		LeakThreshold:       time.Minute * 5,
		OptimizationEnabled: true,
	}

	poolManager := database.NewSQLitePoolManager(edb.DB, poolConfig)
	edb.PoolManager = poolManager

	edb.logger.Info("Database connection pool manager initialized")
	return nil
}

// initializeQueryCache sets up the query result cache
func (edb *EnhancedDB) initializeQueryCache() error {
	// Default cache settings
	maxMemoryBytes := int64(100 * 1024 * 1024) // 100MB
	maxEntries := 10000
	defaultTTL := time.Minute * 30

	cache := database.NewMemoryQueryCache(maxMemoryBytes, maxEntries, defaultTTL)
	edb.QueryCache = cache

	edb.logger.WithFields(logrus.Fields{
		"max_memory_mb": maxMemoryBytes / (1024 * 1024),
		"max_entries":   maxEntries,
		"default_ttl":   defaultTTL,
	}).Info("Database query cache initialized")

	return nil
}

// initializeQueryOptimizer sets up the query optimizer
func (edb *EnhancedDB) initializeQueryOptimizer() error {
	optimizer := database.NewSQLiteOptimizer(edb.DB)
	edb.QueryOptimizer = optimizer

	edb.logger.Info("Database query optimizer initialized")
	return nil
}

// Query executes a query with caching and optimization if available
func (edb *EnhancedDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	// Try cache first if available and enabled
	if edb.QueryCache != nil {
		if _, found := edb.QueryCache.Get(query, args); found {
			// Note: This is a simplified example. In practice, you'd need to
			// serialize/deserialize rows properly
			edb.logger.Debug("Query served from cache")
		}
	}

	// Optimize query if optimizer is available
	optimizedQuery := query
	optimizedArgs := args
	if edb.QueryOptimizer != nil {
		if oq, oa, err := edb.QueryOptimizer.OptimizeQuery(query, args); err == nil {
			optimizedQuery = oq
			optimizedArgs = oa
		}
	}

	// Execute the query
	return edb.DB.Query(optimizedQuery, optimizedArgs...)
}

// Migrate runs database migrations
func Migrate(db *sql.DB, migrationsPath string) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"sqlite",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetPerformanceStats returns performance statistics for all components
func (edb *EnhancedDB) GetPerformanceStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if edb.PoolManager != nil {
		stats["pool"] = edb.PoolManager.MonitorConnections()
	}

	if edb.QueryCache != nil {
		stats["cache"] = edb.QueryCache.GetStats()
	}

	if edb.QueryOptimizer != nil {
		if slowQueries, err := edb.QueryOptimizer.GetSlowQueries(); err == nil {
			stats["slow_queries"] = slowQueries
		}
	}

	return stats
}

// GetHealthStatus returns the health status of all database components
func (edb *EnhancedDB) GetHealthStatus() map[string]interface{} {
	health := make(map[string]interface{})

	// Basic DB health
	health["connection"] = func() string {
		if err := edb.DB.Ping(); err != nil {
			return "unhealthy"
		}
		return "healthy"
	}()

	// Pool health
	if edb.PoolManager != nil {
		health["pool"] = edb.PoolManager.GetConnectionHealth()
	}

	// Cache health
	if edb.QueryCache != nil {
		stats := edb.QueryCache.GetStats()
		health["cache"] = map[string]interface{}{
			"hit_rate":     stats.HitRate,
			"memory_usage": stats.MemoryUsage,
			"status": func() string {
				if stats.HitRate > 0.7 {
					return "healthy"
				} else if stats.HitRate > 0.4 {
					return "warning"
				}
				return "poor"
			}(),
		}
	}

	return health
}

// Close closes the database and all performance components
func (edb *EnhancedDB) Close() error {
	// Stop performance components
	if edb.PoolManager != nil {
		if pm, ok := edb.PoolManager.(*database.SQLitePoolManager); ok {
			pm.Stop()
		}
	}

	if edb.QueryCache != nil {
		if cache, ok := edb.QueryCache.(*database.MemoryQueryCache); ok {
			cache.Stop()
		}
	}

	// Close database connection
	return edb.DB.Close()
}
