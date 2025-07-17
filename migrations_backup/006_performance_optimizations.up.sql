-- Performance Optimization Migration
-- This migration adds indexes for common query patterns and optimizes database performance

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

CREATE INDEX IF NOT EXISTS idx_device_states_timestamp 
ON device_states(timestamp DESC);

-- Add indexes for system_config table optimization
CREATE INDEX IF NOT EXISTS idx_system_config_key 
ON system_config(key); 