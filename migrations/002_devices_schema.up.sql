-- Devices table for storing device configurations and metadata
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    adapter_type TEXT NOT NULL,
    device_type TEXT NOT NULL,
    name TEXT NOT NULL,
    metadata TEXT, -- JSON
    config TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Device states table for storing current and historical device states
CREATE TABLE IF NOT EXISTS device_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,
    state TEXT NOT NULL, -- JSON
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Device credentials table for storing encrypted credentials
CREATE TABLE IF NOT EXISTS device_credentials (
    device_id TEXT PRIMARY KEY,
    credentials TEXT NOT NULL, -- JSON (encrypted)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Device events table for storing device events and history
CREATE TABLE IF NOT EXISTS device_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT,
    adapter_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    data TEXT, -- JSON
    source TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_devices_adapter_type ON devices(adapter_type);
CREATE INDEX IF NOT EXISTS idx_devices_device_type ON devices(device_type);
CREATE INDEX IF NOT EXISTS idx_devices_updated_at ON devices(updated_at);

CREATE INDEX IF NOT EXISTS idx_device_states_device_id ON device_states(device_id);
CREATE INDEX IF NOT EXISTS idx_device_states_timestamp ON device_states(timestamp);
CREATE INDEX IF NOT EXISTS idx_device_states_device_timestamp ON device_states(device_id, timestamp);

CREATE INDEX IF NOT EXISTS idx_device_events_device_id ON device_events(device_id);
CREATE INDEX IF NOT EXISTS idx_device_events_adapter_type ON device_events(adapter_type);
CREATE INDEX IF NOT EXISTS idx_device_events_event_type ON device_events(event_type);
CREATE INDEX IF NOT EXISTS idx_device_events_timestamp ON device_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_device_events_device_timestamp ON device_events(device_id, timestamp); 