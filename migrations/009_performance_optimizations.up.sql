-- Performance Optimization Migration
-- Adds indexes for common query patterns and optimizes database performance

-- Add indexes for common query patterns on entities table
CREATE INDEX IF NOT EXISTS idx_entities_domain_state 
ON entities(domain, state) WHERE domain IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_entities_last_updated 
ON entities(last_updated DESC) WHERE last_updated IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_entities_friendly_name 
ON entities(friendly_name) WHERE friendly_name IS NOT NULL;

-- Add indexes for device_states table performance
CREATE INDEX IF NOT EXISTS idx_device_states_timestamp_device 
ON device_states(timestamp DESC, device_id);

CREATE INDEX IF NOT EXISTS idx_device_states_device_state 
ON device_states(device_id, state) WHERE state IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_device_states_timestamp_desc 
ON device_states(timestamp DESC);

-- Add composite indexes for JOIN operations
CREATE INDEX IF NOT EXISTS idx_device_states_entities_join 
ON device_states(device_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_entities_join 
ON metrics(metric_name, timestamp DESC);

-- Add covering indexes for frequently accessed combinations
CREATE INDEX IF NOT EXISTS idx_entities_covering_basic 
ON entities(entity_id, friendly_name, domain, state, last_updated);

CREATE INDEX IF NOT EXISTS idx_device_states_covering_recent 
ON device_states(device_id, timestamp DESC, state);

-- Add partial indexes for active/enabled records
CREATE INDEX IF NOT EXISTS idx_entities_recent 
ON entities(entity_id, last_updated DESC);

CREATE INDEX IF NOT EXISTS idx_automations_enabled 
ON automation_rules(id, name, enabled) 
WHERE enabled = TRUE;

-- Add text search indexes for better search performance
CREATE INDEX IF NOT EXISTS idx_entities_search_name 
ON entities(friendly_name COLLATE NOCASE);

CREATE INDEX IF NOT EXISTS idx_entities_search_entity_id 
ON entities(entity_id COLLATE NOCASE); 