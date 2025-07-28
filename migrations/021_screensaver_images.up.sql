-- Screensaver Images Migration
-- Table for storing screensaver image metadata and management

-- Screensaver images table
CREATE TABLE IF NOT EXISTS screensaver_images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    width INTEGER,
    height INTEGER,
    checksum TEXT NOT NULL UNIQUE,
    uploaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uploaded_by TEXT NOT NULL DEFAULT 'system',
    tags TEXT, -- JSON array of tags
    active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_screensaver_images_active ON screensaver_images(active);
CREATE INDEX IF NOT EXISTS idx_screensaver_images_uploaded_at ON screensaver_images(uploaded_at);
CREATE INDEX IF NOT EXISTS idx_screensaver_images_checksum ON screensaver_images(checksum);
CREATE INDEX IF NOT EXISTS idx_screensaver_images_content_type ON screensaver_images(content_type);
CREATE INDEX IF NOT EXISTS idx_screensaver_images_file_size ON screensaver_images(file_size);

-- Update trigger for updated_at
CREATE TRIGGER IF NOT EXISTS update_screensaver_images_updated_at
    AFTER UPDATE ON screensaver_images
    FOR EACH ROW
BEGIN
    UPDATE screensaver_images 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END; 