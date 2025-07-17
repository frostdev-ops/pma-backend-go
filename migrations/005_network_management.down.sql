-- Drop network management tables, indexes, and triggers

-- Drop triggers first
DROP TRIGGER IF EXISTS cleanup_expired_dhcp_leases;
DROP TRIGGER IF EXISTS cleanup_old_traffic_stats;
DROP TRIGGER IF EXISTS update_network_interfaces_timestamp;
DROP TRIGGER IF EXISTS update_port_forwarding_timestamp;

-- Drop indexes
DROP INDEX IF EXISTS idx_network_system_component;
DROP INDEX IF EXISTS idx_device_services_port;
DROP INDEX IF EXISTS idx_device_services_device_ip;
DROP INDEX IF EXISTS idx_network_scans_status;
DROP INDEX IF EXISTS idx_network_scans_type;
DROP INDEX IF EXISTS idx_dhcp_leases_state;
DROP INDEX IF EXISTS idx_dhcp_leases_mac;
DROP INDEX IF EXISTS idx_dhcp_leases_ip;
DROP INDEX IF EXISTS idx_traffic_stats_timestamp;
DROP INDEX IF EXISTS idx_traffic_stats_interface;
DROP INDEX IF EXISTS idx_network_interfaces_type;
DROP INDEX IF EXISTS idx_network_interfaces_name;
DROP INDEX IF EXISTS idx_port_forwarding_active;
DROP INDEX IF EXISTS idx_port_forwarding_internal_ip;
DROP INDEX IF EXISTS idx_port_forwarding_external_port;

-- Drop tables
DROP TABLE IF EXISTS network_system_status;
DROP TABLE IF EXISTS device_services;
DROP TABLE IF EXISTS network_scans;
DROP TABLE IF EXISTS dhcp_leases;
DROP TABLE IF EXISTS traffic_stats;
DROP TABLE IF EXISTS network_interfaces;
DROP TABLE IF EXISTS port_forwarding_rules; 