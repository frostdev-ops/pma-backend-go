-- 019_controller_dashboards.down.sql
-- Rollback Controller Dashboard System Tables

-- Drop triggers first
DROP TRIGGER IF EXISTS update_controller_dashboards_timestamp;
DROP TRIGGER IF EXISTS update_controller_templates_timestamp;

-- Drop indexes
DROP INDEX IF EXISTS idx_controller_dashboards_user_id;
DROP INDEX IF EXISTS idx_controller_dashboards_category;
DROP INDEX IF EXISTS idx_controller_dashboards_is_favorite;
DROP INDEX IF EXISTS idx_controller_dashboards_created_at;
DROP INDEX IF EXISTS idx_controller_dashboards_updated_at;

DROP INDEX IF EXISTS idx_controller_templates_user_id;
DROP INDEX IF EXISTS idx_controller_templates_category;
DROP INDEX IF EXISTS idx_controller_templates_is_public;
DROP INDEX IF EXISTS idx_controller_templates_usage_count;

DROP INDEX IF EXISTS idx_controller_shares_dashboard_id;
DROP INDEX IF EXISTS idx_controller_shares_user_id;
DROP INDEX IF EXISTS idx_controller_shares_shared_by;

DROP INDEX IF EXISTS idx_controller_usage_logs_dashboard_id;
DROP INDEX IF EXISTS idx_controller_usage_logs_user_id;
DROP INDEX IF EXISTS idx_controller_usage_logs_action;
DROP INDEX IF EXISTS idx_controller_usage_logs_created_at;

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS controller_usage_logs;
DROP TABLE IF EXISTS controller_shares;
DROP TABLE IF EXISTS controller_templates;
DROP TABLE IF EXISTS controller_dashboards; 