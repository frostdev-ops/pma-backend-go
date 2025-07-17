-- Files table for file management
CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    size INTEGER NOT NULL,
    mime_type TEXT,
    checksum TEXT NOT NULL,
    category TEXT,
    metadata TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Media information table
CREATE TABLE IF NOT EXISTS media_info (
    file_id TEXT PRIMARY KEY,
    duration INTEGER,
    width INTEGER,
    height INTEGER,
    codec TEXT,
    bitrate INTEGER,
    metadata TEXT, -- JSON
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

-- Backups table
CREATE TABLE IF NOT EXISTS backups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    size INTEGER NOT NULL,
    options TEXT, -- JSON
    status TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

-- Backup schedules table
CREATE TABLE IF NOT EXISTS backup_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    options TEXT, -- JSON
    enabled BOOLEAN DEFAULT TRUE,
    last_run DATETIME,
    next_run DATETIME
);

-- File access control table
CREATE TABLE IF NOT EXISTS file_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id TEXT NOT NULL,
    user_id INTEGER,
    permission TEXT NOT NULL, -- read, write, delete
    granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX idx_files_category ON files(category);
CREATE INDEX idx_files_created ON files(created_at);
CREATE INDEX idx_files_mime_type ON files(mime_type);
CREATE INDEX idx_files_checksum ON files(checksum);
CREATE INDEX idx_backups_created ON backups(created_at);
CREATE INDEX idx_backups_status ON backups(status);
CREATE INDEX idx_backup_schedules_enabled ON backup_schedules(enabled);
CREATE INDEX idx_file_permissions_file_id ON file_permissions(file_id);
CREATE INDEX idx_file_permissions_user_id ON file_permissions(user_id); 