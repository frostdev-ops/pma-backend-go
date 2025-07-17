-- Drop device tables and indexes
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

DROP TABLE IF EXISTS device_events;
DROP TABLE IF EXISTS device_credentials;
DROP TABLE IF EXISTS device_states;
DROP TABLE IF EXISTS devices; 