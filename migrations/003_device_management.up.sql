-- Device Management Migration
-- Comprehensive device management including adapters, states, credentials, and network devices

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

-- Network devices discovery
CREATE TABLE IF NOT EXISTS network_devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address TEXT NOT NULL UNIQUE,
    mac_address TEXT,
    hostname TEXT,
    manufacturer TEXT,
    device_type TEXT,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_online BOOLEAN DEFAULT TRUE,
    services TEXT, -- JSON array
    metadata TEXT, -- JSON object
    response_time INTEGER, -- milliseconds
    open_ports TEXT, -- JSON array
    os_fingerprint TEXT,
    vendor_info TEXT,
    discovery_method TEXT, -- ping, arp, mdns, etc.
    user_label TEXT,
    notes TEXT,
    tags TEXT -- JSON array
);

-- UPS monitoring
CREATE TABLE IF NOT EXISTS ups_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    battery_charge REAL,
    battery_runtime INTEGER,
    input_voltage REAL,
    output_voltage REAL,
    load REAL,
    status TEXT,
    temperature REAL,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Camera devices
CREATE TABLE IF NOT EXISTS cameras (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    type TEXT DEFAULT 'generic',
    stream_url TEXT,
    snapshot_url TEXT,
    capabilities TEXT, -- JSON object
    settings TEXT,     -- JSON object
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Bluetooth devices
CREATE TABLE IF NOT EXISTS bluetooth_devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    address TEXT NOT NULL UNIQUE,
    name TEXT,
    device_class TEXT,
    is_paired BOOLEAN DEFAULT FALSE,
    is_connected BOOLEAN DEFAULT FALSE,
    services TEXT, -- JSON array
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    paired_at DATETIME
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

CREATE INDEX IF NOT EXISTS idx_network_devices_ip ON network_devices(ip_address);
CREATE INDEX IF NOT EXISTS idx_network_devices_online ON network_devices(is_online);
CREATE INDEX IF NOT EXISTS idx_cameras_entity_id ON cameras(entity_id);
CREATE INDEX IF NOT EXISTS idx_bluetooth_address ON bluetooth_devices(address); 