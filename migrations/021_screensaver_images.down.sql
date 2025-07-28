-- Rollback Screensaver Images Migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_screensaver_images_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_screensaver_images_file_size;
DROP INDEX IF EXISTS idx_screensaver_images_content_type;
DROP INDEX IF EXISTS idx_screensaver_images_checksum;
DROP INDEX IF EXISTS idx_screensaver_images_uploaded_at;
DROP INDEX IF EXISTS idx_screensaver_images_active;

-- Drop table
DROP TABLE IF EXISTS screensaver_images; 