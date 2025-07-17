-- Drop analytics schema migration
-- Remove all analytics and reporting tables

-- Drop indexes first
DROP INDEX IF EXISTS idx_insights_viewed;
DROP INDEX IF EXISTS idx_insights_type_generated;
DROP INDEX IF EXISTS idx_insights_entity;
DROP INDEX IF EXISTS idx_anomalies_acknowledged;
DROP INDEX IF EXISTS idx_anomalies_severity;
DROP INDEX IF EXISTS idx_anomalies_series_detected;
DROP INDEX IF EXISTS idx_visualizations_type;
DROP INDEX IF EXISTS idx_visualizations_dashboard;
DROP INDEX IF EXISTS idx_predictions_confidence;
DROP INDEX IF EXISTS idx_predictions_model_created;
DROP INDEX IF EXISTS idx_reports_status;
DROP INDEX IF EXISTS idx_reports_template;
DROP INDEX IF EXISTS idx_reports_generated;
DROP INDEX IF EXISTS idx_time_series_type;
DROP INDEX IF EXISTS idx_time_series_resolution;
DROP INDEX IF EXISTS idx_time_series_name_timestamp;
DROP INDEX IF EXISTS idx_metric_data_timestamp;
DROP INDEX IF EXISTS idx_metric_data_metric_timestamp;
DROP INDEX IF EXISTS idx_analytics_events_source;
DROP INDEX IF EXISTS idx_analytics_events_user_timestamp;
DROP INDEX IF EXISTS idx_analytics_events_entity_timestamp;
DROP INDEX IF EXISTS idx_analytics_events_type_timestamp;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS insights;
DROP TABLE IF EXISTS anomalies;
DROP TABLE IF EXISTS predictions;
DROP TABLE IF EXISTS prediction_models;
DROP TABLE IF EXISTS visualizations;
DROP TABLE IF EXISTS dashboards;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS report_templates;
DROP TABLE IF EXISTS time_series_data;
DROP TABLE IF EXISTS metric_data;
DROP TABLE IF EXISTS custom_metrics;
DROP TABLE IF EXISTS analytics_events; 