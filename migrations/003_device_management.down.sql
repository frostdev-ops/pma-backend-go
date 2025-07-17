-- Drop device management tables and indexes

-- Drop indexes
DROP INDEX IF EXISTS idx_bluetooth_address;
DROP INDEX IF EXISTS idx_cameras_entity_id;
DROP INDEX IF EXISTS idx_network_devices_online;
DROP INDEX IF EXISTS idx_network_devices_ip;
DROP INDEX IF EXISTS idx_device_events_device_timestamp;
DROP INDEX IF EXISTS idx_device_events_timestamp;
DROP INDEX IF EXISTS idx_device_events_event_type;
DROP INDEX IF EXISTS idx_device_events_adapter_type;
DROP INDEX IF EXISTS idx_device_events_device_id;
DROP INDEX IF EXISTS idx_device_states_device_timestamp;
DROP INDEX IF EXISTS idx_device_states_timestamp;
DROP INDEX IF EXISTS idx_device_states_device_id;
DROP INDEX IF EXISTS idx_devices_updated_at;
DROP INDEX IF EXISTS idx_devices_device_type;
DROP INDEX IF EXISTS idx_devices_adapter_type;

-- Drop tables
DROP TABLE IF EXISTS bluetooth_devices;
DROP TABLE IF EXISTS cameras;
DROP TABLE IF EXISTS ups_status;
DROP TABLE IF EXISTS network_devices;
DROP TABLE IF EXISTS device_events;
DROP TABLE IF EXISTS device_credentials;
DROP TABLE IF EXISTS device_states;
DROP TABLE IF EXISTS devices; 