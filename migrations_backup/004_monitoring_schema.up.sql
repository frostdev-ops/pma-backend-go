-- Create monitoring and metrics tables

-- Metrics table for storing time-series metrics data
CREATE TABLE IF NOT EXISTS metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_name TEXT NOT NULL,
    metric_type TEXT NOT NULL, -- counter, gauge, histogram
    value REAL NOT NULL,
    labels JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(metric_name, labels, timestamp)
);

-- Alerts table for storing system alerts
CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    source TEXT NOT NULL,
    message TEXT NOT NULL,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolved_by TEXT,
    resolved BOOLEAN DEFAULT FALSE
);

-- System snapshots table for periodic system state capture
CREATE TABLE IF NOT EXISTS system_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cpu_usage REAL,
    memory_usage REAL,
    disk_usage REAL,
    network_bytes_sent INTEGER,
    network_bytes_recv INTEGER,
    goroutines INTEGER,
    connections_active INTEGER,
    connections_idle INTEGER,
    snapshot_data JSON, -- Full system stats as JSON
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance metrics table for HTTP request metrics
CREATE TABLE IF NOT EXISTS performance_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    size_bytes INTEGER,
    user_agent TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Alert rules table for configurable alert rules
CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    metric TEXT NOT NULL,
    operator TEXT NOT NULL CHECK (operator IN ('>', '<', '>=', '<=', '==', '!=')),
    threshold REAL NOT NULL,
    duration_seconds INTEGER NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    message TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    conditions JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Metric thresholds table for threshold configurations
CREATE TABLE IF NOT EXISTS metric_thresholds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric TEXT NOT NULL UNIQUE,
    warning_threshold REAL,
    critical_threshold REAL,
    operator TEXT NOT NULL DEFAULT '>=' CHECK (operator IN ('>', '<', '>=', '<=', '==', '!=')),
    duration_seconds INTEGER DEFAULT 300,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Health check results table
CREATE TABLE IF NOT EXISTS health_checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    component TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    message TEXT,
    details JSON,
    duration_ms INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Service monitoring table for tracking service status over time
CREATE TABLE IF NOT EXISTS service_monitoring (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name TEXT NOT NULL,
    service_type TEXT NOT NULL, -- database, api, external, etc.
    status TEXT NOT NULL CHECK (status IN ('up', 'down', 'degraded')),
    response_time_ms INTEGER,
    error_message TEXT,
    metadata JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_name_timestamp ON metrics(metric_name, timestamp);

CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_source ON alerts(source);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);

CREATE INDEX IF NOT EXISTS idx_system_snapshots_timestamp ON system_snapshots(timestamp);

CREATE INDEX IF NOT EXISTS idx_performance_timestamp ON performance_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_performance_method_path ON performance_metrics(method, path);
CREATE INDEX IF NOT EXISTS idx_performance_status ON performance_metrics(status_code);

CREATE INDEX IF NOT EXISTS idx_health_checks_component ON health_checks(component);
CREATE INDEX IF NOT EXISTS idx_health_checks_timestamp ON health_checks(timestamp);
CREATE INDEX IF NOT EXISTS idx_health_checks_status ON health_checks(status);

CREATE INDEX IF NOT EXISTS idx_service_monitoring_name ON service_monitoring(service_name);
CREATE INDEX IF NOT EXISTS idx_service_monitoring_timestamp ON service_monitoring(timestamp);
CREATE INDEX IF NOT EXISTS idx_service_monitoring_status ON service_monitoring(status);

-- Create triggers to update the updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_alert_rules_timestamp 
    AFTER UPDATE ON alert_rules
    FOR EACH ROW
    BEGIN
        UPDATE alert_rules SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_metric_thresholds_timestamp 
    AFTER UPDATE ON metric_thresholds
    FOR EACH ROW
    BEGIN
        UPDATE metric_thresholds SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END; 