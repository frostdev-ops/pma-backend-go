-- Drop initial schema tables and indexes

DROP INDEX IF EXISTS idx_system_config_key;
DROP INDEX IF EXISTS idx_automation_rules_enabled;
DROP INDEX IF EXISTS idx_entities_domain;
DROP INDEX IF EXISTS idx_entities_room_id;

DROP TABLE IF EXISTS automation_rules;
DROP TABLE IF EXISTS entities;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS system_config;
DROP TABLE IF EXISTS users; 