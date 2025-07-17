-- Network Management Migration
-- Adds comprehensive network management capabilities including port forwarding, traffic monitoring, and DHCP

-- Port forwarding rules
CREATE TABLE IF NOT EXISTS port_forwarding_rules (
    id TEXT PRIMARY KEY, -- UUID string
    name TEXT NOT NULL,
    description TEXT,
    external_port INTEGER NOT NULL,
    internal_ip TEXT NOT NULL,
    internal_mac TEXT NOT NULL,
    internal_port INTEGER NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('tcp', 'udp', 'both')),
    hostname TEXT,
    active BOOLEAN DEFAULT TRUE,
    auto_suggested BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Network interfaces
CREATE TABLE IF NOT EXISTS network_interfaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL, -- ethernet, wifi, bridge, etc.
    mac_address TEXT,
    ip_address TEXT,
    netmask TEXT,
    state TEXT DEFAULT 'unknown', -- up, down, unknown
    mtu INTEGER,
    speed INTEGER, -- Mbps
    duplex TEXT, -- full, half, unknown
    metadata TEXT, -- JSON object
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Traffic statistics
CREATE TABLE IF NOT EXISTS traffic_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    interface_name TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    bytes_sent INTEGER DEFAULT 0,
    bytes_received INTEGER DEFAULT 0,
    packets_sent INTEGER DEFAULT 0,
    packets_received INTEGER DEFAULT 0,
    errors_sent INTEGER DEFAULT 0,
    errors_received INTEGER DEFAULT 0,
    drops_sent INTEGER DEFAULT 0,
    drops_received INTEGER DEFAULT 0
);

-- DHCP leases
CREATE TABLE IF NOT EXISTS dhcp_leases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address TEXT NOT NULL,
    mac_address TEXT NOT NULL,
    hostname TEXT,
    lease_start DATETIME NOT NULL,
    lease_end DATETIME NOT NULL,
    state TEXT DEFAULT 'active', -- active, expired, released
    client_id TEXT,
    vendor_class_id TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Network scans and discovery history
CREATE TABLE IF NOT EXISTS network_scans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scan_type TEXT NOT NULL, -- ping, port, service, full
    target TEXT NOT NULL, -- IP, range, or 'auto'
    status TEXT DEFAULT 'running', -- running, completed, failed
    devices_found INTEGER DEFAULT 0,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    error_message TEXT,
    metadata TEXT -- JSON object with scan details
);

-- Device services (open ports, running services)
CREATE TABLE IF NOT EXISTS device_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL, -- tcp, udp
    service_name TEXT,
    banner TEXT,
    version TEXT,
    state TEXT DEFAULT 'open', -- open, closed, filtered
    response_time INTEGER, -- milliseconds
    discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_verified DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_ip) REFERENCES network_devices(ip_address) ON DELETE CASCADE
);

-- Network system status (bridge, router, etc.)
CREATE TABLE IF NOT EXISTS network_system_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    component TEXT NOT NULL UNIQUE, -- bridge, dhcp_server, router, firewall
    status TEXT NOT NULL, -- running, stopped, error, unknown
    enabled BOOLEAN DEFAULT FALSE,
    pid INTEGER,
    uptime INTEGER, -- seconds
    memory_usage INTEGER, -- KB
    cpu_usage REAL, -- percentage
    metadata TEXT, -- JSON object
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Enhance existing network_devices table with additional fields
-- (using ALTER TABLE since we can't modify existing table structure)
ALTER TABLE network_devices ADD COLUMN response_time INTEGER; -- milliseconds
ALTER TABLE network_devices ADD COLUMN open_ports TEXT; -- JSON array
ALTER TABLE network_devices ADD COLUMN os_fingerprint TEXT;
ALTER TABLE network_devices ADD COLUMN vendor_info TEXT;
ALTER TABLE network_devices ADD COLUMN discovery_method TEXT; -- ping, arp, mdns, etc.
ALTER TABLE network_devices ADD COLUMN user_label TEXT;
ALTER TABLE network_devices ADD COLUMN notes TEXT;
ALTER TABLE network_devices ADD COLUMN tags TEXT; -- JSON array

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_port_forwarding_external_port ON port_forwarding_rules(external_port);
CREATE INDEX IF NOT EXISTS idx_port_forwarding_internal_ip ON port_forwarding_rules(internal_ip);
CREATE INDEX IF NOT EXISTS idx_port_forwarding_active ON port_forwarding_rules(active);
CREATE INDEX IF NOT EXISTS idx_network_interfaces_name ON network_interfaces(name);
CREATE INDEX IF NOT EXISTS idx_network_interfaces_type ON network_interfaces(type);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_interface ON traffic_stats(interface_name);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_timestamp ON traffic_stats(timestamp);
CREATE INDEX IF NOT EXISTS idx_dhcp_leases_ip ON dhcp_leases(ip_address);
CREATE INDEX IF NOT EXISTS idx_dhcp_leases_mac ON dhcp_leases(mac_address);
CREATE INDEX IF NOT EXISTS idx_dhcp_leases_state ON dhcp_leases(state);
CREATE INDEX IF NOT EXISTS idx_network_scans_type ON network_scans(scan_type);
CREATE INDEX IF NOT EXISTS idx_network_scans_status ON network_scans(status);
CREATE INDEX IF NOT EXISTS idx_device_services_device_ip ON device_services(device_ip);
CREATE INDEX IF NOT EXISTS idx_device_services_port ON device_services(port);
CREATE INDEX IF NOT EXISTS idx_network_system_component ON network_system_status(component);

-- Create triggers for automatic timestamp updates
CREATE TRIGGER IF NOT EXISTS update_port_forwarding_timestamp 
AFTER UPDATE ON port_forwarding_rules
BEGIN
    UPDATE port_forwarding_rules SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_network_interfaces_timestamp 
AFTER UPDATE ON network_interfaces
BEGIN
    UPDATE network_interfaces SET updated_at = datetime('now') WHERE id = NEW.id;
END;

-- Clean up old traffic stats (keep 30 days)
CREATE TRIGGER IF NOT EXISTS cleanup_old_traffic_stats 
AFTER INSERT ON traffic_stats
BEGIN
    DELETE FROM traffic_stats 
    WHERE timestamp < datetime('now', '-30 days');
END;

-- Clean up expired DHCP leases
CREATE TRIGGER IF NOT EXISTS cleanup_expired_dhcp_leases 
AFTER INSERT ON dhcp_leases
BEGIN
    UPDATE dhcp_leases 
    SET state = 'expired' 
    WHERE lease_end < datetime('now') AND state = 'active';
END; 