-- Initial Schema Migration
-- Core tables for PMA system: users, entities, rooms, system configuration

-- Users table for authentication
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- System configuration table (for dynamic config)
CREATE TABLE IF NOT EXISTS system_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    encrypted BOOLEAN DEFAULT FALSE,
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Rooms table
CREATE TABLE IF NOT EXISTS rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    home_assistant_area_id TEXT,
    icon TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Entities table (Home Assistant entities)
CREATE TABLE IF NOT EXISTS entities (
    entity_id TEXT PRIMARY KEY,
    friendly_name TEXT,
    domain TEXT NOT NULL,
    state TEXT,
    attributes TEXT, -- JSON
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    room_id INTEGER,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE SET NULL
);

-- Automation rules
CREATE TABLE IF NOT EXISTS automation_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    trigger_type TEXT NOT NULL,
    trigger_config TEXT, -- JSON
    conditions TEXT, -- JSON
    actions TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_entities_room_id ON entities(room_id);
CREATE INDEX IF NOT EXISTS idx_entities_domain ON entities(domain);
CREATE INDEX IF NOT EXISTS idx_automation_rules_enabled ON automation_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_system_config_key ON system_config(key); 