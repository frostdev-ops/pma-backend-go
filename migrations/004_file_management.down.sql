-- Drop file management tables and indexes

-- Drop indexes
DROP INDEX IF EXISTS idx_file_permissions_user_id;
DROP INDEX IF EXISTS idx_file_permissions_file_id;
DROP INDEX IF EXISTS idx_backup_schedules_enabled;
DROP INDEX IF EXISTS idx_backups_status;
DROP INDEX IF EXISTS idx_backups_created;
DROP INDEX IF EXISTS idx_files_checksum;
DROP INDEX IF EXISTS idx_files_mime_type;
DROP INDEX IF EXISTS idx_files_created;
DROP INDEX IF EXISTS idx_files_category;

-- Drop tables
DROP TABLE IF EXISTS file_permissions;
DROP TABLE IF EXISTS backup_schedules;
DROP TABLE IF EXISTS backups;
DROP TABLE IF EXISTS media_info;
DROP TABLE IF EXISTS files; 