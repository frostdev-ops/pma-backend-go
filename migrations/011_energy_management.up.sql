-- Energy Management Migration
-- Energy monitoring and consumption tracking system

-- Energy settings table
CREATE TABLE IF NOT EXISTS energy_settings (
    id INTEGER PRIMARY KEY,
    energy_rate REAL NOT NULL DEFAULT 0.12,
    currency TEXT NOT NULL DEFAULT 'USD',
    tracking_enabled INTEGER NOT NULL DEFAULT 1,
    update_interval INTEGER NOT NULL DEFAULT 30,
    historical_period INTEGER NOT NULL DEFAULT 30,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Energy history table for overall consumption snapshots
CREATE TABLE IF NOT EXISTS energy_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    power_consumption REAL NOT NULL,
    energy_usage REAL NOT NULL,
    cost REAL NOT NULL,
    device_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Device energy table for individual device consumption
CREATE TABLE IF NOT EXISTS device_energy (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id TEXT NOT NULL,
    device_name TEXT NOT NULL,
    room TEXT,
    power_consumption REAL NOT NULL,
    energy_usage REAL NOT NULL,
    cost REAL NOT NULL,
    state TEXT NOT NULL,
    is_on INTEGER NOT NULL,
    percentage REAL NOT NULL,
    timestamp TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_energy_history_timestamp ON energy_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_energy_history_created_at ON energy_history(created_at);
CREATE INDEX IF NOT EXISTS idx_device_energy_entity_id ON device_energy(entity_id);
CREATE INDEX IF NOT EXISTS idx_device_energy_room ON device_energy(room);
CREATE INDEX IF NOT EXISTS idx_device_energy_timestamp ON device_energy(timestamp);
CREATE INDEX IF NOT EXISTS idx_device_energy_created_at ON device_energy(created_at);

-- Insert default energy settings
INSERT OR IGNORE INTO energy_settings (id, energy_rate, currency, tracking_enabled, update_interval, historical_period)
VALUES (1, 0.12, 'USD', 1, 30, 30); 