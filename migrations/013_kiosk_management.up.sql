-- Kiosk Management System Enhancement Migration
-- This migration adds comprehensive kiosk management capabilities

-- Kiosk device configurations
CREATE TABLE IF NOT EXISTS kiosk_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id TEXT NOT NULL,
    theme TEXT DEFAULT 'auto' CHECK (theme IN ('light', 'dark', 'auto')),
    layout TEXT DEFAULT 'grid' CHECK (layout IN ('grid', 'list')),
    quick_actions TEXT, -- JSON array of device IDs for quick access
    update_interval INTEGER DEFAULT 1000, -- milliseconds
    display_timeout INTEGER DEFAULT 300, -- seconds before dimming/sleep
    brightness INTEGER DEFAULT 80, -- 0-100
    screensaver_enabled BOOLEAN DEFAULT 1,
    screensaver_type TEXT DEFAULT 'clock' CHECK (screensaver_type IN ('clock', 'slideshow', 'blank')),
    screensaver_timeout INTEGER DEFAULT 900, -- seconds
    auto_hide_navigation BOOLEAN DEFAULT 0,
    fullscreen_mode BOOLEAN DEFAULT 1,
    voice_control_enabled BOOLEAN DEFAULT 0,
    gesture_control_enabled BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(room_id)
);

-- Kiosk device groups for bulk management
CREATE TABLE IF NOT EXISTS kiosk_device_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    color TEXT DEFAULT '#3b82f6', -- hex color for UI
    icon TEXT DEFAULT 'devices', -- icon name
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Many-to-many relationship between kiosks and groups
CREATE TABLE IF NOT EXISTS kiosk_group_memberships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kiosk_token_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kiosk_token_id) REFERENCES kiosk_tokens(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES kiosk_device_groups(id) ON DELETE CASCADE,
    UNIQUE(kiosk_token_id, group_id)
);

-- Kiosk activity and error logs
CREATE TABLE IF NOT EXISTS kiosk_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kiosk_token_id TEXT NOT NULL,
    level TEXT NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error', 'critical')),
    category TEXT NOT NULL, -- 'system', 'user_action', 'device_interaction', 'error', 'security'
    message TEXT NOT NULL,
    details TEXT, -- JSON object with additional context
    device_id TEXT, -- if related to a specific device
    user_action TEXT, -- if related to user interaction
    error_code TEXT, -- for system errors
    stack_trace TEXT, -- for debugging errors
    ip_address TEXT,
    user_agent TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kiosk_token_id) REFERENCES kiosk_tokens(id) ON DELETE CASCADE
);

-- Kiosk device status and health monitoring
CREATE TABLE IF NOT EXISTS kiosk_device_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kiosk_token_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('online', 'offline', 'pairing', 'error', 'maintenance')),
    last_heartbeat DATETIME DEFAULT CURRENT_TIMESTAMP,
    device_info TEXT, -- JSON object with device details (OS, browser, screen resolution, etc.)
    performance_metrics TEXT, -- JSON object with performance data
    error_count_24h INTEGER DEFAULT 0,
    uptime_seconds INTEGER DEFAULT 0,
    network_quality TEXT, -- 'excellent', 'good', 'fair', 'poor'
    battery_level INTEGER, -- for battery-powered devices
    temperature INTEGER, -- device temperature if available
    memory_usage_percent INTEGER,
    cpu_usage_percent INTEGER,
    storage_usage_percent INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kiosk_token_id) REFERENCES kiosk_tokens(id) ON DELETE CASCADE,
    UNIQUE(kiosk_token_id)
);

-- Kiosk command queue for remote management
CREATE TABLE IF NOT EXISTS kiosk_commands (
    id TEXT PRIMARY KEY,
    kiosk_token_id TEXT NOT NULL,
    command_type TEXT NOT NULL, -- 'restart', 'update_config', 'refresh', 'screenshot', 'log_level', etc.
    command_data TEXT, -- JSON object with command parameters
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'acknowledged', 'completed', 'failed', 'expired')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME,
    acknowledged_at DATETIME,
    completed_at DATETIME,
    expires_at DATETIME NOT NULL,
    result_data TEXT, -- JSON object with command result
    error_message TEXT,
    FOREIGN KEY (kiosk_token_id) REFERENCES kiosk_tokens(id) ON DELETE CASCADE
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_kiosk_configs_room_id ON kiosk_configs(room_id);
CREATE INDEX IF NOT EXISTS idx_kiosk_logs_kiosk_token_id ON kiosk_logs(kiosk_token_id);
CREATE INDEX IF NOT EXISTS idx_kiosk_logs_timestamp ON kiosk_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_kiosk_logs_level ON kiosk_logs(level);
CREATE INDEX IF NOT EXISTS idx_kiosk_logs_category ON kiosk_logs(category);
CREATE INDEX IF NOT EXISTS idx_kiosk_device_status_kiosk_token_id ON kiosk_device_status(kiosk_token_id);
CREATE INDEX IF NOT EXISTS idx_kiosk_device_status_status ON kiosk_device_status(status);
CREATE INDEX IF NOT EXISTS idx_kiosk_device_status_last_heartbeat ON kiosk_device_status(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_kiosk_commands_kiosk_token_id ON kiosk_commands(kiosk_token_id);
CREATE INDEX IF NOT EXISTS idx_kiosk_commands_status ON kiosk_commands(status);
CREATE INDEX IF NOT EXISTS idx_kiosk_commands_expires_at ON kiosk_commands(expires_at);
CREATE INDEX IF NOT EXISTS idx_kiosk_group_memberships_kiosk_token_id ON kiosk_group_memberships(kiosk_token_id);
CREATE INDEX IF NOT EXISTS idx_kiosk_group_memberships_group_id ON kiosk_group_memberships(group_id);

-- Insert default device groups
INSERT OR IGNORE INTO kiosk_device_groups (id, name, description, color, icon) VALUES
('default', 'Default Group', 'Default group for all kiosk devices', '#6b7280', 'devices'),
('living-room', 'Living Room', 'Kiosks in living room areas', '#3b82f6', 'sofa'),
('kitchen', 'Kitchen', 'Kitchen area kiosks', '#10b981', 'chef-hat'),
('bedroom', 'Bedrooms', 'Bedroom kiosk devices', '#8b5cf6', 'bed'),
('office', 'Office/Study', 'Work and study area kiosks', '#f59e0b', 'briefcase'),
('guest', 'Guest Areas', 'Guest room and temporary kiosks', '#ef4444', 'user-plus');

-- Update existing kiosk_tokens table if it doesn't have the active column
-- Note: SQLite doesn't support ALTER TABLE ADD COLUMN IF NOT EXISTS
-- We need to handle this gracefully
CREATE TABLE IF NOT EXISTS kiosk_tokens_new (
    id TEXT PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    room_id TEXT NOT NULL,
    allowed_devices TEXT, -- JSON array
    active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME,
    expires_at DATETIME
);

-- Copy data from old table if it exists and doesn't have active column
INSERT OR IGNORE INTO kiosk_tokens_new (id, token, name, room_id, allowed_devices, created_at, last_used, expires_at)
SELECT id, token, name, room_id, allowed_devices, created_at, last_used, expires_at
FROM kiosk_tokens;

-- Drop old table and rename new one
DROP TABLE IF EXISTS kiosk_tokens;
ALTER TABLE kiosk_tokens_new RENAME TO kiosk_tokens;

-- Recreate index for kiosk tokens
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_token ON kiosk_tokens(token);
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_room_id ON kiosk_tokens(room_id);
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_active ON kiosk_tokens(active);
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_last_used ON kiosk_tokens(last_used); 