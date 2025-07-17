-- Analytics Migration
-- Comprehensive analytics including events, reports, predictions, and visualizations

-- Analytics events table for tracking all system events
CREATE TABLE IF NOT EXISTS analytics_events (
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
CREATE TABLE IF NOT EXISTS custom_metrics (
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
CREATE TABLE IF NOT EXISTS metric_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_id TEXT NOT NULL,
    value REAL NOT NULL,
    tags JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (metric_id) REFERENCES custom_metrics(id) ON DELETE CASCADE
);

-- Time series data for historical analysis
CREATE TABLE IF NOT EXISTS time_series_data (
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
CREATE TABLE IF NOT EXISTS report_templates (
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
CREATE TABLE IF NOT EXISTS reports (
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

-- Dashboards configuration
CREATE TABLE IF NOT EXISTS dashboards (
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
CREATE TABLE IF NOT EXISTS visualizations (
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

-- Prediction models
CREATE TABLE IF NOT EXISTS prediction_models (
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
CREATE TABLE IF NOT EXISTS predictions (
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

-- Anomaly detection results
CREATE TABLE IF NOT EXISTS anomalies (
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
CREATE TABLE IF NOT EXISTS insights (
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
CREATE INDEX IF NOT EXISTS idx_analytics_events_type_timestamp ON analytics_events(type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_entity_timestamp ON analytics_events(entity_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_user_timestamp ON analytics_events(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_source ON analytics_events(source);

CREATE INDEX IF NOT EXISTS idx_metric_data_metric_timestamp ON metric_data(metric_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metric_data_timestamp ON metric_data(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_time_series_name_timestamp ON time_series_data(series_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_time_series_resolution ON time_series_data(resolution, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_time_series_type ON time_series_data(data_type);

CREATE INDEX IF NOT EXISTS idx_reports_generated ON reports(generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_template ON reports(template_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);

CREATE INDEX IF NOT EXISTS idx_predictions_model_created ON predictions(model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_predictions_confidence ON predictions(confidence DESC);

CREATE INDEX IF NOT EXISTS idx_visualizations_dashboard ON visualizations(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_visualizations_type ON visualizations(type);

CREATE INDEX IF NOT EXISTS idx_anomalies_series_detected ON anomalies(series_name, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON anomalies(severity);
CREATE INDEX IF NOT EXISTS idx_anomalies_acknowledged ON anomalies(acknowledged);

CREATE INDEX IF NOT EXISTS idx_insights_entity ON insights(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_insights_type_generated ON insights(insight_type, generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_insights_viewed ON insights(viewed); 