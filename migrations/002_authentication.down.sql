-- Drop authentication enhancement tables and triggers

-- Drop triggers first
DROP TRIGGER IF EXISTS clean_expired_sessions;
DROP TRIGGER IF EXISTS clean_old_failed_attempts;

-- Drop indexes
DROP INDEX IF EXISTS idx_sessions_token;
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_failed_auth_client_id;
DROP INDEX IF EXISTS idx_failed_auth_attempt_at;
DROP INDEX IF EXISTS idx_kiosk_tokens_token;
DROP INDEX IF EXISTS idx_kiosk_pairing_pin;

-- Drop tables
DROP TABLE IF EXISTS display_settings;
DROP TABLE IF EXISTS kiosk_pairing_sessions;
DROP TABLE IF EXISTS kiosk_tokens;
DROP TABLE IF EXISTS failed_auth_attempts;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS auth_settings; 