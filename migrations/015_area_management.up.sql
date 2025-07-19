-- 015_area_management.up.sql
-- Area Management System Migration

-- Areas table: Enhanced area information beyond basic HA areas
CREATE TABLE IF NOT EXISTS areas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    area_id TEXT UNIQUE, -- Optional external area ID (e.g., HA area ID)
    description TEXT,
    icon TEXT,
    floor_level INTEGER DEFAULT 0,
    parent_area_id INTEGER REFERENCES areas(id) ON DELETE SET NULL,
    color TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    area_type TEXT DEFAULT 'room', -- room, zone, building, floor, etc.
    metadata TEXT, -- JSON metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Area mappings: Relationships between different area systems
CREATE TABLE IF NOT EXISTS area_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pma_area_id INTEGER NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    external_area_id TEXT NOT NULL, -- HA area ID, etc.
    external_system TEXT NOT NULL DEFAULT 'homeassistant',
    mapping_type TEXT DEFAULT 'direct', -- direct, derived, manual
    auto_sync BOOLEAN DEFAULT TRUE,
    sync_priority INTEGER DEFAULT 1,
    last_synced DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(external_area_id, external_system)
);

-- Area settings: Configuration and preferences
CREATE TABLE IF NOT EXISTS area_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    setting_key TEXT NOT NULL,
    setting_value TEXT,
    area_id INTEGER REFERENCES areas(id) ON DELETE CASCADE,
    is_global BOOLEAN DEFAULT FALSE, -- Global settings when area_id is NULL
    data_type TEXT DEFAULT 'string', -- string, integer, boolean, json
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(setting_key, area_id)
);

-- Area analytics: Metrics and statistics
CREATE TABLE IF NOT EXISTS area_analytics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    area_id INTEGER NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    metric_unit TEXT,
    aggregation_type TEXT DEFAULT 'snapshot', -- snapshot, sum, avg, max, min
    time_period TEXT, -- hourly, daily, weekly, monthly
    recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Area sync log: Synchronization history and status
CREATE TABLE IF NOT EXISTS area_sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_type TEXT NOT NULL, -- full, incremental, manual
    external_system TEXT NOT NULL DEFAULT 'homeassistant',
    status TEXT NOT NULL, -- pending, running, success, failed, partial
    areas_processed INTEGER DEFAULT 0,
    areas_updated INTEGER DEFAULT 0,
    areas_created INTEGER DEFAULT 0,
    areas_deleted INTEGER DEFAULT 0,
    error_message TEXT,
    sync_details TEXT, -- JSON details
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Room-area relationships: Link rooms to areas
CREATE TABLE IF NOT EXISTS room_area_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    area_id INTEGER NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    assignment_type TEXT DEFAULT 'primary', -- primary, secondary, inherited
    confidence_score REAL DEFAULT 1.0, -- 0.0 to 1.0
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(room_id, area_id, assignment_type)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_areas_area_id ON areas(area_id);
CREATE INDEX IF NOT EXISTS idx_areas_parent ON areas(parent_area_id);
CREATE INDEX IF NOT EXISTS idx_areas_type ON areas(area_type);
CREATE INDEX IF NOT EXISTS idx_areas_active ON areas(is_active);

CREATE INDEX IF NOT EXISTS idx_area_mappings_pma_area ON area_mappings(pma_area_id);
CREATE INDEX IF NOT EXISTS idx_area_mappings_external ON area_mappings(external_area_id, external_system);
CREATE INDEX IF NOT EXISTS idx_area_mappings_sync ON area_mappings(auto_sync, last_synced);

CREATE INDEX IF NOT EXISTS idx_area_settings_key ON area_settings(setting_key);
CREATE INDEX IF NOT EXISTS idx_area_settings_area ON area_settings(area_id);
CREATE INDEX IF NOT EXISTS idx_area_settings_global ON area_settings(is_global);

CREATE INDEX IF NOT EXISTS idx_area_analytics_area ON area_analytics(area_id);
CREATE INDEX IF NOT EXISTS idx_area_analytics_metric ON area_analytics(metric_name);
CREATE INDEX IF NOT EXISTS idx_area_analytics_time ON area_analytics(recorded_at);

CREATE INDEX IF NOT EXISTS idx_area_sync_log_system ON area_sync_log(external_system);
CREATE INDEX IF NOT EXISTS idx_area_sync_log_status ON area_sync_log(status);
CREATE INDEX IF NOT EXISTS idx_area_sync_log_started ON area_sync_log(started_at);

CREATE INDEX IF NOT EXISTS idx_room_area_assignments_room ON room_area_assignments(room_id);
CREATE INDEX IF NOT EXISTS idx_room_area_assignments_area ON room_area_assignments(area_id);

-- Insert default global settings
INSERT OR IGNORE INTO area_settings (setting_key, setting_value, is_global, data_type) VALUES
('sync_enabled', 'true', TRUE, 'boolean'),
('sync_interval_minutes', '60', TRUE, 'integer'),
('auto_create_areas', 'true', TRUE, 'boolean'),
('default_area_type', 'room', TRUE, 'string'),
('max_hierarchy_depth', '3', TRUE, 'integer'),
('analytics_retention_days', '365', TRUE, 'integer'),
('sync_on_startup', 'true', TRUE, 'boolean'),
('conflict_resolution', 'homeassistant_priority', TRUE, 'string');

-- Create triggers for updated_at timestamps
CREATE TRIGGER IF NOT EXISTS update_areas_timestamp 
    AFTER UPDATE ON areas
    BEGIN
        UPDATE areas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_area_mappings_timestamp 
    AFTER UPDATE ON area_mappings
    BEGIN
        UPDATE area_mappings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_area_settings_timestamp 
    AFTER UPDATE ON area_settings
    BEGIN
        UPDATE area_settings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_room_area_assignments_timestamp 
    AFTER UPDATE ON room_area_assignments
    BEGIN
        UPDATE room_area_assignments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END; 