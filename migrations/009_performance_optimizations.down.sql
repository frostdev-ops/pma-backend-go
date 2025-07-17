-- Reverse Performance Optimization Migration
-- This migration removes performance optimizations

-- Drop text search indexes
DROP INDEX IF EXISTS idx_entities_search_entity_id;
DROP INDEX IF EXISTS idx_entities_search_name;

-- Drop partial indexes for active/enabled records
DROP INDEX IF EXISTS idx_automations_enabled;
DROP INDEX IF EXISTS idx_entities_recent;

-- Drop covering indexes
DROP INDEX IF EXISTS idx_device_states_covering_recent;
DROP INDEX IF EXISTS idx_entities_covering_basic;

-- Drop composite indexes for JOIN operations
DROP INDEX IF EXISTS idx_metrics_entities_join;
DROP INDEX IF EXISTS idx_device_states_entities_join;

-- Drop device_states table indexes
DROP INDEX IF EXISTS idx_device_states_timestamp_desc;
DROP INDEX IF EXISTS idx_device_states_device_state;
DROP INDEX IF EXISTS idx_device_states_timestamp_device;

-- Drop entities table indexes
DROP INDEX IF EXISTS idx_entities_friendly_name;
DROP INDEX IF EXISTS idx_entities_last_updated;
DROP INDEX IF EXISTS idx_entities_domain_state; 