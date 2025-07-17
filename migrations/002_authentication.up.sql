-- Authentication Enhancement Migration
-- Adds comprehensive authentication features including sessions, security, and display settings

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

-- Display settings
CREATE TABLE IF NOT EXISTS display_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    brightness INTEGER DEFAULT 100,
    sleep_timeout INTEGER DEFAULT 300,
    orientation TEXT DEFAULT 'landscape',
    screensaver_enabled BOOLEAN DEFAULT TRUE,
    screensaver_timeout INTEGER DEFAULT 600,
    screensaver_type TEXT DEFAULT 'clock',
    screensaver_settings TEXT, -- JSON object
    key TEXT,
    value TEXT,
    category TEXT DEFAULT 'general',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (id = 1) -- Ensure only one row for system settings
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_failed_auth_client_id ON failed_auth_attempts(client_id);
CREATE INDEX IF NOT EXISTS idx_failed_auth_attempt_at ON failed_auth_attempts(attempt_at);
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_token ON kiosk_tokens(token);
CREATE INDEX IF NOT EXISTS idx_kiosk_pairing_pin ON kiosk_pairing_sessions(pin);

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