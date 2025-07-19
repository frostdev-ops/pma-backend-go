-- Rollback kiosk management system enhancement

-- Drop indexes first
DROP INDEX IF EXISTS idx_kiosk_configs_room_id;
DROP INDEX IF EXISTS idx_kiosk_logs_kiosk_token_id;
DROP INDEX IF EXISTS idx_kiosk_logs_timestamp;
DROP INDEX IF EXISTS idx_kiosk_logs_level;
DROP INDEX IF EXISTS idx_kiosk_logs_category;
DROP INDEX IF EXISTS idx_kiosk_device_status_kiosk_token_id;
DROP INDEX IF EXISTS idx_kiosk_device_status_status;
DROP INDEX IF EXISTS idx_kiosk_device_status_last_heartbeat;
DROP INDEX IF EXISTS idx_kiosk_commands_kiosk_token_id;
DROP INDEX IF EXISTS idx_kiosk_commands_status;
DROP INDEX IF EXISTS idx_kiosk_commands_expires_at;
DROP INDEX IF EXISTS idx_kiosk_group_memberships_kiosk_token_id;
DROP INDEX IF EXISTS idx_kiosk_group_memberships_group_id;
DROP INDEX IF EXISTS idx_kiosk_tokens_room_id;
DROP INDEX IF EXISTS idx_kiosk_tokens_active;
DROP INDEX IF EXISTS idx_kiosk_tokens_last_used;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS kiosk_commands;
DROP TABLE IF EXISTS kiosk_device_status;
DROP TABLE IF EXISTS kiosk_logs;
DROP TABLE IF EXISTS kiosk_group_memberships;
DROP TABLE IF EXISTS kiosk_device_groups;
DROP TABLE IF EXISTS kiosk_configs;

-- Restore original kiosk_tokens table structure
CREATE TABLE IF NOT EXISTS kiosk_tokens_original (
    id TEXT PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    room_id TEXT NOT NULL,
    allowed_devices TEXT, -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME,
    expires_at DATETIME
);

-- Copy data back (excluding active column)
INSERT OR IGNORE INTO kiosk_tokens_original (id, token, name, room_id, allowed_devices, created_at, last_used, expires_at)
SELECT id, token, name, room_id, allowed_devices, created_at, last_used, expires_at
FROM kiosk_tokens;

-- Replace current table
DROP TABLE IF EXISTS kiosk_tokens;
ALTER TABLE kiosk_tokens_original RENAME TO kiosk_tokens;

-- Recreate original index
CREATE INDEX IF NOT EXISTS idx_kiosk_tokens_token ON kiosk_tokens(token); 