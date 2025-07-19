-- PMA Enhancement Tables Migration
-- This migration adds tables to support the PMA unified entity system

-- Enhanced unified entities table for PMA-specific entity data
CREATE TABLE IF NOT EXISTS unified_entities (
    entity_id TEXT PRIMARY KEY,
    last_unified DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sync_status TEXT NOT NULL DEFAULT 'synced',
    availability_status TEXT NOT NULL DEFAULT 'available',
    response_time REAL,
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    last_error_time DATETIME,
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id) ON DELETE CASCADE
);

-- Entity source mappings for tracking where entities come from
CREATE TABLE IF NOT EXISTS entity_source_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'home_assistant',
    source_id TEXT NOT NULL,
    metadata TEXT DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_id, source_type),
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id) ON DELETE CASCADE
);

-- Enhanced unified rooms table for PMA-specific room data
CREATE TABLE IF NOT EXISTS unified_rooms (
    room_id INTEGER PRIMARY KEY,
    entity_count INTEGER NOT NULL DEFAULT 0,
    active_entity_count INTEGER NOT NULL DEFAULT 0,
    last_activity DATETIME,
    sync_status TEXT NOT NULL DEFAULT 'synced',
    metadata TEXT DEFAULT '{}',
    temperature_sensor TEXT,
    humidity_sensor TEXT,
    current_temp REAL,
    current_humidity REAL,
    power_consumption REAL,
    energy_today REAL,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- Drop and recreate devices table with new schema for PMA enhancements
DROP TABLE IF EXISTS devices;
CREATE TABLE devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    manufacturer TEXT,
    model TEXT,
    sw_version TEXT,
    hw_version TEXT,
    area_id TEXT,
    config_entries TEXT DEFAULT '[]',
    connections TEXT DEFAULT '[]',
    identifiers TEXT DEFAULT '[]',
    via_device_id TEXT,
    disabled_by TEXT,
    configuration_url TEXT,
    entry_type TEXT,
    last_updated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Device entities mapping table
CREATE TABLE IF NOT EXISTS device_entities (
    device_id TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    PRIMARY KEY (device_id, entity_id),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id) ON DELETE CASCADE
);

-- Categories table for entity organization
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    icon TEXT,
    color TEXT,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- PMA settings table
CREATE TABLE IF NOT EXISTS pma_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_sync_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sync_interval_minutes INTEGER NOT NULL DEFAULT 15,
    cache_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cache_ttl_seconds INTEGER NOT NULL DEFAULT 300,
    max_retry_attempts INTEGER NOT NULL DEFAULT 3,
    retry_delay_ms INTEGER NOT NULL DEFAULT 2000,
    health_check_interval INTEGER NOT NULL DEFAULT 60,
    metrics_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Entity categories mapping table
CREATE TABLE IF NOT EXISTS entity_categories (
    entity_id TEXT NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (entity_id, category_id),
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_unified_entities_sync_status ON unified_entities(sync_status);
CREATE INDEX IF NOT EXISTS idx_unified_entities_availability ON unified_entities(availability_status);
CREATE INDEX IF NOT EXISTS idx_unified_entities_last_unified ON unified_entities(last_unified);
CREATE INDEX IF NOT EXISTS idx_unified_entities_error_count ON unified_entities(error_count);

CREATE INDEX IF NOT EXISTS idx_entity_source_mappings_entity_id ON entity_source_mappings(entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_source_mappings_source_type ON entity_source_mappings(source_type);
CREATE INDEX IF NOT EXISTS idx_entity_source_mappings_updated_at ON entity_source_mappings(updated_at);

CREATE INDEX IF NOT EXISTS idx_unified_rooms_sync_status ON unified_rooms(sync_status);
CREATE INDEX IF NOT EXISTS idx_unified_rooms_entity_count ON unified_rooms(entity_count);
CREATE INDEX IF NOT EXISTS idx_unified_rooms_last_activity ON unified_rooms(last_activity);

CREATE INDEX IF NOT EXISTS idx_devices_area_id ON devices(area_id);
CREATE INDEX IF NOT EXISTS idx_devices_last_updated ON devices(last_updated);

CREATE INDEX IF NOT EXISTS idx_device_entities_device_id ON device_entities(device_id);
CREATE INDEX IF NOT EXISTS idx_device_entities_entity_id ON device_entities(entity_id);

-- Insert default categories
INSERT OR IGNORE INTO categories (name, icon, description) VALUES 
('Lights', 'lightbulb', 'Lighting devices'),
('Switches', 'switch', 'Switch devices'),
('Sensors', 'sensor', 'Sensor devices'),
('Climate', 'thermometer', 'Climate control devices'),
('Media', 'speaker', 'Media devices'),
('Security', 'shield', 'Security devices'),
('Energy', 'battery', 'Energy monitoring devices'),
('Covers', 'window-close', 'Window covers and blinds'),
('Fans', 'fan', 'Fan devices'),
('Locks', 'lock', 'Door and window locks'),
('Cameras', 'camera', 'Camera devices'),
('Vacuum', 'robot-vacuum', 'Vacuum cleaners'),
('Weather', 'weather-partly-cloudy', 'Weather stations'),
('Automation', 'automation', 'Automation entities'),
('Scripts', 'script', 'Script entities'),
('Scenes', 'palette', 'Scene entities'),
('Other', 'help-circle', 'Other uncategorized entities');

-- Insert default PMA settings
INSERT OR IGNORE INTO pma_settings (
    auto_sync_enabled, sync_interval_minutes, cache_enabled, cache_ttl_seconds,
    max_retry_attempts, retry_delay_ms, health_check_interval, metrics_enabled
) VALUES (
    TRUE, 15, TRUE, 300, 3, 2000, 60, TRUE
);

-- Create triggers to automatically update timestamps
CREATE TRIGGER IF NOT EXISTS update_entity_source_mappings_timestamp 
    AFTER UPDATE ON entity_source_mappings
BEGIN
    UPDATE entity_source_mappings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_pma_settings_timestamp 
    AFTER UPDATE ON pma_settings
BEGIN
    UPDATE pma_settings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Create trigger to update room entity counts when entities are added/removed/updated
CREATE TRIGGER IF NOT EXISTS update_room_entity_count_on_insert
    AFTER INSERT ON entities
    WHEN NEW.room_id IS NOT NULL
BEGIN
    INSERT OR REPLACE INTO unified_rooms (room_id, entity_count, active_entity_count)
    VALUES (
        NEW.room_id,
        (SELECT COUNT(*) FROM entities WHERE room_id = NEW.room_id),
        (SELECT COUNT(*) FROM entities e 
         LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id 
         WHERE e.room_id = NEW.room_id AND COALESCE(ue.availability_status, 'available') = 'available')
    );
END;

CREATE TRIGGER IF NOT EXISTS update_room_entity_count_on_update
    AFTER UPDATE ON entities
    WHEN OLD.room_id IS NOT NULL OR NEW.room_id IS NOT NULL
BEGIN
    -- Update old room if room_id changed
    INSERT OR REPLACE INTO unified_rooms (room_id, entity_count, active_entity_count)
    SELECT 
        OLD.room_id,
        (SELECT COUNT(*) FROM entities WHERE room_id = OLD.room_id),
        (SELECT COUNT(*) FROM entities e 
         LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id 
         WHERE e.room_id = OLD.room_id AND COALESCE(ue.availability_status, 'available') = 'available')
    WHERE OLD.room_id IS NOT NULL AND OLD.room_id != NEW.room_id;
    
    -- Update new room
    INSERT OR REPLACE INTO unified_rooms (room_id, entity_count, active_entity_count)
    SELECT 
        NEW.room_id,
        (SELECT COUNT(*) FROM entities WHERE room_id = NEW.room_id),
        (SELECT COUNT(*) FROM entities e 
         LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id 
         WHERE e.room_id = NEW.room_id AND COALESCE(ue.availability_status, 'available') = 'available')
    WHERE NEW.room_id IS NOT NULL;
END;

CREATE TRIGGER IF NOT EXISTS update_room_entity_count_on_delete
    AFTER DELETE ON entities
    WHEN OLD.room_id IS NOT NULL
BEGIN
    INSERT OR REPLACE INTO unified_rooms (room_id, entity_count, active_entity_count)
    VALUES (
        OLD.room_id,
        (SELECT COUNT(*) FROM entities WHERE room_id = OLD.room_id),
        (SELECT COUNT(*) FROM entities e 
         LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id 
         WHERE e.room_id = OLD.room_id AND COALESCE(ue.availability_status, 'available') = 'available')
    );
END;

-- Create trigger to update room active entity count when unified entity availability changes
CREATE TRIGGER IF NOT EXISTS update_room_active_count_on_unified_entity_update
    AFTER UPDATE ON unified_entities
    WHEN OLD.availability_status != NEW.availability_status
BEGIN
    UPDATE unified_rooms 
    SET active_entity_count = (
        SELECT COUNT(*) FROM entities e 
        LEFT JOIN unified_entities ue ON e.entity_id = ue.entity_id 
        WHERE e.room_id = unified_rooms.room_id 
        AND COALESCE(ue.availability_status, 'available') = 'available'
    )
    WHERE room_id IN (
        SELECT room_id FROM entities WHERE entity_id = NEW.entity_id AND room_id IS NOT NULL
    );
END; 