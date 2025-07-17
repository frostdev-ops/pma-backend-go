-- Reverse Performance Optimization Migration
-- This migration removes performance optimizations added in 009_performance_optimizations.up.sql

-- Drop performance monitoring triggers
DROP TRIGGER IF EXISTS update_performance_metrics_on_entity_change;

-- Drop performance monitoring views
DROP VIEW IF EXISTS performance_dashboard;

-- Drop performance monitoring tables
DROP TABLE IF EXISTS performance_cache_stats;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS performance_slow_queries;

-- Drop text search indexes
DROP INDEX IF EXISTS idx_entities_search_entity_id;
DROP INDEX IF EXISTS idx_entities_search_name;

-- Drop partial indexes for active/enabled records
DROP INDEX IF EXISTS idx_automations_enabled;
DROP INDEX IF EXISTS idx_entities_active;

-- Drop covering indexes
DROP INDEX IF EXISTS idx_device_states_covering_recent;
DROP INDEX IF EXISTS idx_entities_covering_basic;

-- Drop composite indexes for JOIN operations
DROP INDEX IF EXISTS idx_metrics_entities_join;
DROP INDEX IF EXISTS idx_device_states_entities_join;

-- Drop notification indexes
DROP INDEX IF EXISTS idx_notifications_type_timestamp;
DROP INDEX IF EXISTS idx_notifications_user_read;

-- Drop user session indexes
DROP INDEX IF EXISTS idx_user_sessions_expires_at;
DROP INDEX IF EXISTS idx_user_sessions_user_active;

-- Drop log indexes
DROP INDEX IF EXISTS idx_logs_source_timestamp;
DROP INDEX IF EXISTS idx_logs_level_timestamp;

-- Drop automation execution indexes
DROP INDEX IF EXISTS idx_automation_executions_duration;
DROP INDEX IF EXISTS idx_automation_executions_status_timestamp;
DROP INDEX IF EXISTS idx_automation_executions_automation_timestamp;

-- Drop metrics table indexes
DROP INDEX IF EXISTS idx_metrics_value_range;
DROP INDEX IF EXISTS idx_metrics_entity_timestamp;
DROP INDEX IF EXISTS idx_metrics_name_timestamp;

-- Drop device_states table indexes
DROP INDEX IF EXISTS idx_device_states_created_at;
DROP INDEX IF EXISTS idx_device_states_device_state;
DROP INDEX IF EXISTS idx_device_states_timestamp_device;

-- Drop entities table indexes
DROP INDEX IF EXISTS idx_entities_friendly_name;
DROP INDEX IF EXISTS idx_entities_last_updated;
DROP INDEX IF EXISTS idx_entities_domain_state;

-- Reset SQLite settings to defaults
PRAGMA journal_mode = DELETE;
PRAGMA synchronous = FULL;
PRAGMA cache_size = -2000; -- Default 2MB cache
PRAGMA temp_store = DEFAULT;
PRAGMA mmap_size = 0; -- Disable memory mapping 