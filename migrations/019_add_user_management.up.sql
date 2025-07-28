-- Add user management functionality
-- This migration ensures the users table exists and adds necessary indexes

-- Create users table if it doesn't exist (it should already exist from initial migration)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for user management
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- Insert default admin user if no users exist
INSERT OR IGNORE INTO users (id, username, password_hash, created_at, updated_at)
SELECT 1, 'admin', 'admin', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = 1); 