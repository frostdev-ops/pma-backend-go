-- Analytics Schema Migration
-- Create tables for comprehensive analytics and reporting system

-- Analytics events table for tracking all system events
CREATE TABLE analytics_events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    entity_id TEXT,
    entity_type TEXT,
    user_id TEXT,
    data JSON NOT NULL,
    context JSON,
    source TEXT,
    tags JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

-- Custom metrics definitions
CREATE TABLE custom_metrics (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    type TEXT NOT NULL, -- counter, gauge, histogram, summary
    unit TEXT,
    definition JSON NOT NULL,
    retention_period INTEGER, -- in days
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT TRUE
);

-- Metric data points
CREATE TABLE metric_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_id TEXT NOT NULL,
    value REAL NOT NULL,
    tags JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (metric_id) REFERENCES custom_metrics(id) ON DELETE CASCADE
);

-- Time series data for historical analysis
CREATE TABLE time_series_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_name TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    value REAL NOT NULL,
    tags JSON,
    metadata JSON,
    resolution INTEGER, -- aggregation interval in seconds
    data_type TEXT DEFAULT 'raw' -- raw, hourly, daily, weekly, monthly
);

-- Report templates
CREATE TABLE report_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT,
    type TEXT NOT NULL, -- dashboard, summary, detailed, custom
    sections JSON NOT NULL,
    parameters JSON,
    styling JSON,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT TRUE
);

-- Generated reports
CREATE TABLE reports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT,
    parameters JSON,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    generated_by TEXT,
    format TEXT, -- pdf, html, json, csv
    data BLOB,
    summary JSON,
    file_path TEXT,
    size_bytes INTEGER,
    status TEXT DEFAULT 'completed', -- generating, completed, failed
    error_message TEXT,
    FOREIGN KEY (template_id) REFERENCES report_templates(id)
);

-- Scheduled reports
CREATE TABLE scheduled_reports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL,
    parameters JSON,
    schedule_cron TEXT NOT NULL, -- cron expression
    format TEXT NOT NULL,
    destinations JSON, -- email, webhook, file
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (template_id) REFERENCES report_templates(id)
);

-- Prediction models
CREATE TABLE prediction_models (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL, -- linear_regression, time_series, classification
    algorithm TEXT,
    accuracy REAL,
    parameters JSON,
    features JSON,
    training_data_size INTEGER,
    trained_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    trained_by TEXT,
    model_data BLOB,
    active BOOLEAN DEFAULT TRUE
);

-- Predictions made by models
CREATE TABLE predictions (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    input_data JSON NOT NULL,
    predicted_value REAL,
    confidence REAL,
    prediction_horizon INTEGER, -- hours into future
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    actual_value REAL, -- filled in later for accuracy tracking
    FOREIGN KEY (model_id) REFERENCES prediction_models(id)
);

-- Dashboards configuration
CREATE TABLE dashboards (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    layout TEXT, -- grid, masonry, flow
    config JSON NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    shared BOOLEAN DEFAULT FALSE,
    active BOOLEAN DEFAULT TRUE
);

-- Visualizations within dashboards
CREATE TABLE visualizations (
    id TEXT PRIMARY KEY,
    dashboard_id TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- line, bar, pie, gauge, table, heatmap
    query_config JSON NOT NULL,
    visualization_config JSON NOT NULL,
    position JSON, -- x, y, width, height
    data_refresh_interval INTEGER DEFAULT 300, -- seconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (dashboard_id) REFERENCES dashboards(id) ON DELETE CASCADE
);

-- Export schedules
CREATE TABLE export_schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    query_config JSON NOT NULL,
    format TEXT NOT NULL, -- csv, json, excel, pdf
    schedule_cron TEXT NOT NULL,
    destinations JSON, -- file, webhook, email
    compression BOOLEAN DEFAULT FALSE,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    active BOOLEAN DEFAULT TRUE
);

-- Export jobs history
CREATE TABLE export_jobs (
    id TEXT PRIMARY KEY,
    schedule_id TEXT,
    name TEXT NOT NULL,
    format TEXT NOT NULL,
    status TEXT DEFAULT 'pending', -- pending, running, completed, failed
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    file_path TEXT,
    file_size INTEGER,
    records_count INTEGER,
    error_message TEXT,
    FOREIGN KEY (schedule_id) REFERENCES export_schedules(id)
);

-- Data aggregation cache
CREATE TABLE aggregation_cache (
    id TEXT PRIMARY KEY,
    cache_key TEXT UNIQUE NOT NULL,
    data JSON NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Anomaly detection results
CREATE TABLE anomalies (
    id TEXT PRIMARY KEY,
    series_name TEXT NOT NULL,
    anomaly_type TEXT, -- spike, drop, trend_change, outlier
    severity TEXT, -- low, medium, high, critical
    detected_at TIMESTAMP NOT NULL,
    value REAL,
    expected_value REAL,
    deviation REAL,
    confidence REAL,
    metadata JSON,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by TEXT,
    acknowledged_at TIMESTAMP
);

-- Insights generated from data analysis
CREATE TABLE insights (
    id TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id TEXT,
    insight_type TEXT NOT NULL, -- trend, pattern, anomaly, recommendation
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    impact_level TEXT, -- low, medium, high
    confidence REAL,
    data_points JSON,
    recommendations JSON,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    viewed BOOLEAN DEFAULT FALSE,
    acted_upon BOOLEAN DEFAULT FALSE
);

-- Performance indexes for efficient querying
CREATE INDEX idx_analytics_events_type_timestamp ON analytics_events(type, timestamp DESC);
CREATE INDEX idx_analytics_events_entity_timestamp ON analytics_events(entity_id, timestamp DESC);
CREATE INDEX idx_analytics_events_user_timestamp ON analytics_events(user_id, timestamp DESC);
CREATE INDEX idx_analytics_events_source ON analytics_events(source);

CREATE INDEX idx_metric_data_metric_timestamp ON metric_data(metric_id, timestamp DESC);
CREATE INDEX idx_metric_data_timestamp ON metric_data(timestamp DESC);

CREATE INDEX idx_time_series_name_timestamp ON time_series_data(series_name, timestamp DESC);
CREATE INDEX idx_time_series_resolution ON time_series_data(resolution, timestamp DESC);
CREATE INDEX idx_time_series_type ON time_series_data(data_type);

CREATE INDEX idx_reports_generated ON reports(generated_at DESC);
CREATE INDEX idx_reports_template ON reports(template_id);
CREATE INDEX idx_reports_status ON reports(status);

CREATE INDEX idx_predictions_model_created ON predictions(model_id, created_at DESC);
CREATE INDEX idx_predictions_confidence ON predictions(confidence DESC);

CREATE INDEX idx_visualizations_dashboard ON visualizations(dashboard_id);
CREATE INDEX idx_visualizations_type ON visualizations(type);

CREATE INDEX idx_export_jobs_schedule ON export_jobs(schedule_id);
CREATE INDEX idx_export_jobs_status ON export_jobs(status);
CREATE INDEX idx_export_jobs_started ON export_jobs(started_at DESC);

CREATE INDEX idx_aggregation_cache_key ON aggregation_cache(cache_key);
CREATE INDEX idx_aggregation_cache_expires ON aggregation_cache(expires_at);

CREATE INDEX idx_anomalies_series_detected ON anomalies(series_name, detected_at DESC);
CREATE INDEX idx_anomalies_severity ON anomalies(severity);
CREATE INDEX idx_anomalies_acknowledged ON anomalies(acknowledged);

CREATE INDEX idx_insights_entity ON insights(entity_type, entity_id);
CREATE INDEX idx_insights_type_generated ON insights(insight_type, generated_at DESC);
CREATE INDEX idx_insights_viewed ON insights(viewed); 