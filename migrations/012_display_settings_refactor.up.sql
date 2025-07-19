-- Migration 012: Refactor display_settings table to match service expectations
-- Current table is a key-value store, replace with structured table

-- Drop the old key-value table
DROP TABLE IF EXISTS display_settings;

-- Create new table with correct schema for structured display settings
CREATE TABLE display_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    brightness INTEGER DEFAULT 100 NOT NULL,
    timeout INTEGER DEFAULT 300 NOT NULL,  -- seconds, 0 = never
    orientation TEXT DEFAULT 'landscape' NOT NULL,
    darkMode TEXT DEFAULT 'auto' NOT NULL,
    screensaver BOOLEAN DEFAULT TRUE NOT NULL,
    screensaverType TEXT DEFAULT 'clock',
    screensaverShowClock BOOLEAN DEFAULT TRUE,
    screensaverRotationSpeed INTEGER DEFAULT 5,
    screensaverPictureFrameImage TEXT,
    screensaverUploadEnabled BOOLEAN DEFAULT TRUE,
    dimBeforeSleep BOOLEAN DEFAULT TRUE,
    dimLevel INTEGER DEFAULT 30,
    dimTimeout INTEGER DEFAULT 60,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (id = 1), -- Ensure only one row for system settings
    CHECK (brightness >= 0 AND brightness <= 100),
    CHECK (timeout >= 0),
    CHECK (orientation IN ('portrait', 'landscape', 'portrait_flipped', 'landscape_flipped')),
    CHECK (darkMode IN ('light', 'dark', 'auto')),
    CHECK (screensaverType IN ('none', 'clock', 'slideshow', 'pictureframe')),
    CHECK (screensaverRotationSpeed >= 1 AND screensaverRotationSpeed <= 60),
    CHECK (dimLevel >= 1 AND dimLevel <= 99),
    CHECK (dimTimeout >= 1)
);

-- Insert default settings
INSERT INTO display_settings (id) VALUES (1);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_display_settings_updated_at ON display_settings(updated_at); 