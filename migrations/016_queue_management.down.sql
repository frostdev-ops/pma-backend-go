-- Rollback Action Queue Management System Migration
-- This migration removes all queue management tables and indexes

-- Drop triggers first
DROP TRIGGER IF EXISTS update_action_types_updated_at;
DROP TRIGGER IF EXISTS update_queue_settings_updated_at;
DROP TRIGGER IF EXISTS update_queued_actions_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_queue_settings_category;
DROP INDEX IF EXISTS idx_action_dependencies_depends_on;
DROP INDEX IF EXISTS idx_action_dependencies_action;
DROP INDEX IF EXISTS idx_action_results_created_at;
DROP INDEX IF EXISTS idx_action_results_status;
DROP INDEX IF EXISTS idx_action_results_attempt;
DROP INDEX IF EXISTS idx_action_results_action_id;
DROP INDEX IF EXISTS idx_queued_actions_parent;
DROP INDEX IF EXISTS idx_queued_actions_user_id;
DROP INDEX IF EXISTS idx_queued_actions_created_at;
DROP INDEX IF EXISTS idx_queued_actions_correlation;
DROP INDEX IF EXISTS idx_queued_actions_target_entity;
DROP INDEX IF EXISTS idx_queued_actions_next_retry_at;
DROP INDEX IF EXISTS idx_queued_actions_scheduled_at;
DROP INDEX IF EXISTS idx_queued_actions_status_priority;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS action_dependencies;
DROP TABLE IF EXISTS queue_settings;
DROP TABLE IF EXISTS action_results;
DROP TABLE IF EXISTS queued_actions;
DROP TABLE IF EXISTS action_types;
DROP TABLE IF EXISTS action_statuses;
DROP TABLE IF EXISTS action_priorities; 