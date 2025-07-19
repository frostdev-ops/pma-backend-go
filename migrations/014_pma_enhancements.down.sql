-- Drop PMA Enhancement Tables Migration (Rollback)

-- Drop triggers first
DROP TRIGGER IF EXISTS update_room_active_count_on_unified_entity_update;
DROP TRIGGER IF EXISTS update_room_entity_count_on_delete;
DROP TRIGGER IF EXISTS update_room_entity_count_on_update;
DROP TRIGGER IF EXISTS update_room_entity_count_on_insert;
DROP TRIGGER IF EXISTS update_pma_settings_timestamp;
DROP TRIGGER IF EXISTS update_entity_source_mappings_timestamp;

-- Drop indexes
DROP INDEX IF EXISTS idx_device_entities_entity_id;
DROP INDEX IF EXISTS idx_device_entities_device_id;
DROP INDEX IF EXISTS idx_devices_last_updated;
DROP INDEX IF EXISTS idx_devices_area_id;
DROP INDEX IF EXISTS idx_unified_rooms_last_activity;
DROP INDEX IF EXISTS idx_unified_rooms_entity_count;
DROP INDEX IF EXISTS idx_unified_rooms_sync_status;
DROP INDEX IF EXISTS idx_entity_source_mappings_updated_at;
DROP INDEX IF EXISTS idx_entity_source_mappings_source_type;
DROP INDEX IF EXISTS idx_entity_source_mappings_entity_id;
DROP INDEX IF EXISTS idx_unified_entities_error_count;
DROP INDEX IF EXISTS idx_unified_entities_last_unified;
DROP INDEX IF EXISTS idx_unified_entities_availability;
DROP INDEX IF EXISTS idx_unified_entities_sync_status;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS entity_categories;
DROP TABLE IF EXISTS device_entities;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS unified_rooms;
DROP TABLE IF EXISTS entity_source_mappings;
DROP TABLE IF EXISTS unified_entities;
DROP TABLE IF EXISTS pma_settings; 