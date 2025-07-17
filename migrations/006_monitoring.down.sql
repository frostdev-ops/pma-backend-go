-- Drop monitoring and metrics tables (reverse order)

-- Drop triggers
DROP TRIGGER IF EXISTS update_metric_thresholds_timestamp;
DROP TRIGGER IF EXISTS update_alert_rules_timestamp;

-- Drop indexes
DROP INDEX IF EXISTS idx_service_monitoring_status;
DROP INDEX IF EXISTS idx_service_monitoring_timestamp;
DROP INDEX IF EXISTS idx_service_monitoring_name;

DROP INDEX IF EXISTS idx_health_checks_status;
DROP INDEX IF EXISTS idx_health_checks_timestamp;
DROP INDEX IF EXISTS idx_health_checks_component;

DROP INDEX IF EXISTS idx_performance_status;
DROP INDEX IF EXISTS idx_performance_method_path;
DROP INDEX IF EXISTS idx_performance_timestamp;

DROP INDEX IF EXISTS idx_system_snapshots_timestamp;

DROP INDEX IF EXISTS idx_alerts_created_at;
DROP INDEX IF EXISTS idx_alerts_source;
DROP INDEX IF EXISTS idx_alerts_severity;
DROP INDEX IF EXISTS idx_alerts_resolved;

DROP INDEX IF EXISTS idx_metrics_name_timestamp;
DROP INDEX IF EXISTS idx_metrics_name;
DROP INDEX IF EXISTS idx_metrics_timestamp;

-- Drop tables
DROP TABLE IF EXISTS service_monitoring;
DROP TABLE IF EXISTS health_checks;
DROP TABLE IF EXISTS metric_thresholds;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS system_snapshots;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS metrics; 