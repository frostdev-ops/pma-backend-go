-- Migration 012 Down: Revert display_settings table refactor

-- Drop new table
DROP TABLE IF EXISTS display_settings;

-- Recreate old table structure
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

-- Insert default values
INSERT OR IGNORE INTO display_settings (id) VALUES (1); 