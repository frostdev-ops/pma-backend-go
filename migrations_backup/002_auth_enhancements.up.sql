-- Enhanced authentication tables

-- Authentication settings
CREATE TABLE IF NOT EXISTS auth_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    pin_code TEXT,
    session_timeout INTEGER DEFAULT 300,
    max_failed_attempts INTEGER DEFAULT 3,
    lockout_duration INTEGER DEFAULT 300,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (id = 1) -- Ensure only one row
);

-- Sessions for authentication
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Failed authentication attempts for rate limiting
CREATE TABLE IF NOT EXISTS failed_auth_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id TEXT NOT NULL,
    ip_address TEXT,
    attempt_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    attempt_type TEXT DEFAULT 'pin'
);

-- Kiosk device tokens
CREATE TABLE IF NOT EXISTS kiosk_tokens (
    id TEXT PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    room_id TEXT NOT NULL,
    allowed_devices TEXT, -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME,
    expires_at DATETIME
);

-- Kiosk pairing sessions
CREATE TABLE IF NOT EXISTS kiosk_pairing_sessions (
    id TEXT PRIMARY KEY,
    pin TEXT NOT NULL,
    room_id TEXT NOT NULL,
    device_info TEXT, -- JSON object
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending' -- pending, confirmed, expired
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
    metadata TEXT  -- JSON object
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

-- Display settings (enhanced)
CREATE TABLE IF NOT EXISTS display_settings_new (
    id INTEGER PRIMARY KEY DEFAULT 1,
    brightness INTEGER DEFAULT 100,
    sleep_timeout INTEGER DEFAULT 300,
    orientation TEXT DEFAULT 'landscape',
    screensaver_enabled BOOLEAN DEFAULT TRUE,
    screensaver_timeout INTEGER DEFAULT 600,
    screensaver_type TEXT DEFAULT 'clock',
    screensaver_settings TEXT, -- JSON object
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (id = 1) -- Ensure only one row
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
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_failed_auth_client_id ON failed_auth_attempts(client_id);
CREATE INDEX IF NOT EXISTS idx_failed_auth_attempt_at ON failed_auth_attempts(attempt_at);
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_token ON kiosk_tokens(token);
CREATE INDEX IF NOT EXISTS idx_kiosk_pairing_pin ON kiosk_pairing_sessions(pin);
CREATE INDEX IF NOT EXISTS idx_network_devices_ip ON network_devices(ip_address);
CREATE INDEX IF NOT EXISTS idx_network_devices_online ON network_devices(is_online);
CREATE INDEX IF NOT EXISTS idx_cameras_entity_id ON cameras(entity_id);
CREATE INDEX IF NOT EXISTS idx_bluetooth_address ON bluetooth_devices(address);

-- Clean up expired sessions trigger
CREATE TRIGGER IF NOT EXISTS clean_expired_sessions 
AFTER INSERT ON sessions
BEGIN
    DELETE FROM sessions WHERE expires_at < datetime('now');
END;

-- Clean up old failed attempts trigger (keep last 24 hours)
CREATE TRIGGER IF NOT EXISTS clean_old_failed_attempts 
AFTER INSERT ON failed_auth_attempts
BEGIN
    DELETE FROM failed_auth_attempts 
    WHERE attempt_at < datetime('now', '-24 hours');
END; 