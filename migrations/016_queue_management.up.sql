-- Action Queue Management System Migration
-- This migration creates tables for managing action queues with retry mechanisms,
-- priority handling, and comprehensive status tracking

-- Enum types for action priorities
CREATE TABLE IF NOT EXISTS action_priorities (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    weight INTEGER NOT NULL DEFAULT 0,
    description TEXT
);

-- Insert default priorities
INSERT OR IGNORE INTO action_priorities (id, name, weight, description) VALUES
(1, 'low', 1, 'Low priority actions'),
(2, 'normal', 5, 'Normal priority actions'),
(3, 'high', 10, 'High priority actions'),
(4, 'urgent', 20, 'Urgent priority actions'),
(5, 'critical', 50, 'Critical priority actions');

-- Enum types for action statuses
CREATE TABLE IF NOT EXISTS action_statuses (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    is_terminal BOOLEAN DEFAULT FALSE
);

-- Insert default statuses
INSERT OR IGNORE INTO action_statuses (id, name, description, is_terminal) VALUES
(1, 'pending', 'Action is queued and waiting to be processed', FALSE),
(2, 'processing', 'Action is currently being executed', FALSE),
(3, 'completed', 'Action completed successfully', TRUE),
(4, 'failed', 'Action failed and will not be retried', TRUE),
(5, 'retrying', 'Action failed but will be retried', FALSE),
(6, 'cancelled', 'Action was cancelled before completion', TRUE),
(7, 'timeout', 'Action timed out during execution', TRUE),
(8, 'paused', 'Action execution is paused', FALSE);

-- Action types that can be queued
CREATE TABLE IF NOT EXISTS action_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    handler_name TEXT NOT NULL,
    default_timeout INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    retry_backoff_factor REAL DEFAULT 2.0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default action types
INSERT OR IGNORE INTO action_types (name, description, handler_name, default_timeout, max_retries) VALUES
('entity_state_change', 'Change state of Home Assistant entity', 'EntityStateHandler', 30, 3),
('service_call', 'Call Home Assistant service', 'ServiceCallHandler', 60, 3),
('scene_activation', 'Activate scene', 'SceneHandler', 30, 2),
('automation_trigger', 'Trigger automation rule', 'AutomationHandler', 120, 3),
('system_command', 'Execute system command', 'SystemCommandHandler', 300, 1),
('script_execution', 'Execute custom script', 'ScriptHandler', 600, 2),
('notification_send', 'Send notification', 'NotificationHandler', 15, 5),
('bulk_operation', 'Execute multiple actions as one unit', 'BulkOperationHandler', 600, 2);

-- Main queued actions table
CREATE TABLE IF NOT EXISTS queued_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type_id INTEGER NOT NULL REFERENCES action_types(id),
    priority_id INTEGER NOT NULL DEFAULT 2 REFERENCES action_priorities(id),
    status_id INTEGER NOT NULL DEFAULT 1 REFERENCES action_statuses(id),
    
    -- Action identification and metadata
    name TEXT NOT NULL,
    description TEXT,
    user_id INTEGER REFERENCES users(id),
    correlation_id TEXT, -- For grouping related actions
    parent_action_id INTEGER REFERENCES queued_actions(id), -- For dependent actions
    
    -- Action payload and configuration
    action_data TEXT NOT NULL, -- JSON serialized action parameters
    target_entity_id TEXT, -- Optional entity target for faster filtering
    
    -- Execution configuration
    timeout_seconds INTEGER,
    max_retries INTEGER,
    retry_count INTEGER DEFAULT 0,
    retry_backoff_factor REAL DEFAULT 2.0,
    
    -- Scheduling and timing
    scheduled_at TIMESTAMP, -- When to execute (NULL = execute immediately)
    execute_after TIMESTAMP, -- Don't execute before this time
    deadline TIMESTAMP, -- Must complete before this time
    
    -- Execution tracking
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    last_attempt_at TIMESTAMP,
    next_retry_at TIMESTAMP,
    
    -- Result and error tracking
    result_data TEXT, -- JSON serialized result
    error_message TEXT,
    error_details TEXT, -- JSON serialized error details
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT, -- Service or user that created the action
    
    -- Performance tracking
    execution_duration_ms INTEGER, -- How long the action took to execute
    
    CHECK (retry_count >= 0),
    CHECK (max_retries >= 0),
    CHECK (timeout_seconds > 0 OR timeout_seconds IS NULL),
    CHECK (retry_backoff_factor >= 1.0)
);

-- Action execution results and history
CREATE TABLE IF NOT EXISTS action_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_id INTEGER NOT NULL REFERENCES queued_actions(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    status_id INTEGER NOT NULL REFERENCES action_statuses(id),
    
    -- Execution details
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    
    -- Result data
    success BOOLEAN DEFAULT FALSE,
    result_data TEXT, -- JSON serialized result
    error_code TEXT,
    error_message TEXT,
    error_details TEXT, -- JSON serialized error details
    
    -- Context information
    worker_id TEXT, -- Which worker processed this
    execution_context TEXT, -- JSON with execution environment details
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (attempt_number > 0),
    CHECK (duration_ms >= 0 OR duration_ms IS NULL)
);

-- Queue configuration and settings
CREATE TABLE IF NOT EXISTS queue_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL,
    data_type TEXT NOT NULL DEFAULT 'string', -- string, integer, boolean, float, json
    description TEXT,
    category TEXT DEFAULT 'general',
    is_readonly BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default queue settings
INSERT OR IGNORE INTO queue_settings (key, value, data_type, description, category) VALUES
('max_concurrent_workers', '5', 'integer', 'Maximum number of concurrent queue workers', 'processing'),
('worker_poll_interval_ms', '1000', 'integer', 'How often workers check for new actions (milliseconds)', 'processing'),
('default_action_timeout', '30', 'integer', 'Default timeout for actions in seconds', 'processing'),
('max_retry_attempts', '3', 'integer', 'Default maximum retry attempts', 'retry'),
('retry_backoff_base_ms', '1000', 'integer', 'Base backoff time for retries in milliseconds', 'retry'),
('max_retry_backoff_ms', '300000', 'integer', 'Maximum backoff time for retries (5 minutes)', 'retry'),
('dead_letter_retention_days', '7', 'integer', 'How long to keep failed actions in days', 'cleanup'),
('completed_action_retention_days', '30', 'integer', 'How long to keep completed actions in days', 'cleanup'),
('enable_action_dependencies', 'true', 'boolean', 'Enable action dependency management', 'features'),
('enable_bulk_operations', 'true', 'boolean', 'Enable bulk operation support', 'features'),
('queue_health_check_interval_ms', '30000', 'integer', 'Health check interval in milliseconds', 'monitoring'),
('enable_websocket_notifications', 'true', 'boolean', 'Send queue updates via WebSocket', 'notifications');

-- Action dependencies (for actions that must wait for others)
CREATE TABLE IF NOT EXISTS action_dependencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_id INTEGER NOT NULL REFERENCES queued_actions(id) ON DELETE CASCADE,
    depends_on_action_id INTEGER NOT NULL REFERENCES queued_actions(id) ON DELETE CASCADE,
    dependency_type TEXT NOT NULL DEFAULT 'completion', -- completion, success, failure
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(action_id, depends_on_action_id),
    CHECK (action_id != depends_on_action_id)
);

-- Performance indexes for efficient queue operations
CREATE INDEX IF NOT EXISTS idx_queued_actions_status_priority ON queued_actions(status_id, priority_id, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_queued_actions_scheduled_at ON queued_actions(scheduled_at) WHERE scheduled_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_queued_actions_next_retry_at ON queued_actions(next_retry_at) WHERE next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_queued_actions_target_entity ON queued_actions(target_entity_id) WHERE target_entity_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_queued_actions_correlation ON queued_actions(correlation_id) WHERE correlation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_queued_actions_created_at ON queued_actions(created_at);
CREATE INDEX IF NOT EXISTS idx_queued_actions_user_id ON queued_actions(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_queued_actions_parent ON queued_actions(parent_action_id) WHERE parent_action_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_action_results_action_id ON action_results(action_id);
CREATE INDEX IF NOT EXISTS idx_action_results_attempt ON action_results(action_id, attempt_number);
CREATE INDEX IF NOT EXISTS idx_action_results_status ON action_results(status_id);
CREATE INDEX IF NOT EXISTS idx_action_results_created_at ON action_results(created_at);

CREATE INDEX IF NOT EXISTS idx_action_dependencies_action ON action_dependencies(action_id);
CREATE INDEX IF NOT EXISTS idx_action_dependencies_depends_on ON action_dependencies(depends_on_action_id);

CREATE INDEX IF NOT EXISTS idx_queue_settings_category ON queue_settings(category);

-- Create triggers for updating timestamps
CREATE TRIGGER IF NOT EXISTS update_queued_actions_updated_at
    AFTER UPDATE ON queued_actions
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
    BEGIN
        UPDATE queued_actions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_queue_settings_updated_at
    AFTER UPDATE ON queue_settings
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
    BEGIN
        UPDATE queue_settings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_action_types_updated_at
    AFTER UPDATE ON action_types
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
    BEGIN
        UPDATE action_types SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END; 