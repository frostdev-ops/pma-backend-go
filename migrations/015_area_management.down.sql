-- 015_area_management.down.sql
-- Area Management System Migration Rollback

-- Drop triggers first
DROP TRIGGER IF EXISTS update_room_area_assignments_timestamp;
DROP TRIGGER IF EXISTS update_area_settings_timestamp;
DROP TRIGGER IF EXISTS update_area_mappings_timestamp;
DROP TRIGGER IF EXISTS update_areas_timestamp;

-- Drop indexes
DROP INDEX IF EXISTS idx_room_area_assignments_area;
DROP INDEX IF EXISTS idx_room_area_assignments_room;
DROP INDEX IF EXISTS idx_area_sync_log_started;
DROP INDEX IF EXISTS idx_area_sync_log_status;
DROP INDEX IF EXISTS idx_area_sync_log_system;
DROP INDEX IF EXISTS idx_area_analytics_time;
DROP INDEX IF EXISTS idx_area_analytics_metric;
DROP INDEX IF EXISTS idx_area_analytics_area;
DROP INDEX IF EXISTS idx_area_settings_global;
DROP INDEX IF EXISTS idx_area_settings_area;
DROP INDEX IF EXISTS idx_area_settings_key;
DROP INDEX IF EXISTS idx_area_mappings_sync;
DROP INDEX IF EXISTS idx_area_mappings_external;
DROP INDEX IF EXISTS idx_area_mappings_pma_area;
DROP INDEX IF EXISTS idx_areas_active;
DROP INDEX IF EXISTS idx_areas_type;
DROP INDEX IF EXISTS idx_areas_parent;
DROP INDEX IF EXISTS idx_areas_area_id;

-- Drop tables in reverse order of creation (considering foreign key dependencies)
DROP TABLE IF EXISTS room_area_assignments;
DROP TABLE IF EXISTS area_sync_log;
DROP TABLE IF EXISTS area_analytics;
DROP TABLE IF EXISTS area_settings;
DROP TABLE IF EXISTS area_mappings;
DROP TABLE IF EXISTS areas; 