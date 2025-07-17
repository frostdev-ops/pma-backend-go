-- Drop indexes
DROP INDEX IF EXISTS idx_device_energy_created_at;
DROP INDEX IF EXISTS idx_device_energy_timestamp;
DROP INDEX IF EXISTS idx_device_energy_room;
DROP INDEX IF EXISTS idx_device_energy_entity_id;
DROP INDEX IF EXISTS idx_energy_history_created_at;
DROP INDEX IF EXISTS idx_energy_history_timestamp;

-- Drop tables
DROP TABLE IF EXISTS device_energy;
DROP TABLE IF EXISTS energy_history;
DROP TABLE IF EXISTS energy_settings; 