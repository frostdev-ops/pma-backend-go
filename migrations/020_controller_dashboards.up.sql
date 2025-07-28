-- 019_controller_dashboards.up.sql
-- Controller Dashboard System Tables

-- Main dashboards table
CREATE TABLE IF NOT EXISTS controller_dashboards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    category TEXT DEFAULT 'custom',
    
    -- Layout configuration stored as JSON
    layout_config TEXT NOT NULL DEFAULT '{"columns":12,"rows":8,"grid_size":64,"gap":8,"responsive":true}',
    
    -- Dashboard elements stored as JSON array
    elements_json TEXT NOT NULL DEFAULT '[]',
    
    -- Style configuration stored as JSON
    style_config TEXT NOT NULL DEFAULT '{"theme":"auto","border_radius":8,"padding":16}',
    
    -- Access control configuration stored as JSON
    access_config TEXT NOT NULL DEFAULT '{"public":false,"shared_with":[],"requires_auth":false}',
    
    -- Metadata
    is_favorite BOOLEAN DEFAULT FALSE,
    tags TEXT DEFAULT '[]', -- JSON array of tags
    thumbnail_url TEXT DEFAULT NULL,
    version INTEGER DEFAULT 1,
    
    -- User ownership (nullable for system/default dashboards)
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP DEFAULT NULL
);

-- Dashboard templates table
CREATE TABLE IF NOT EXISTS controller_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    category TEXT DEFAULT 'custom',
    
    -- Template configuration stored as JSON
    template_json TEXT NOT NULL,
    
    -- Template variables for customization stored as JSON
    variables_json TEXT DEFAULT '[]',
    
    -- Preview/thumbnail
    thumbnail_url TEXT DEFAULT NULL,
    
    -- Usage statistics
    usage_count INTEGER DEFAULT 0,
    rating REAL DEFAULT 0.0,
    
    -- Visibility and access
    is_public BOOLEAN DEFAULT FALSE,
    
    -- User ownership (nullable for system templates)
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Dashboard sharing table
CREATE TABLE IF NOT EXISTS controller_shares (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dashboard_id INTEGER NOT NULL REFERENCES controller_dashboards(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Permission levels: view, edit, admin
    permissions TEXT NOT NULL DEFAULT 'view',
    
    -- Sharing metadata
    shared_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP DEFAULT NULL,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique sharing per user per dashboard
    UNIQUE(dashboard_id, user_id)
);

-- Dashboard usage logs for analytics
CREATE TABLE IF NOT EXISTS controller_usage_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dashboard_id INTEGER NOT NULL REFERENCES controller_dashboards(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    
    -- Action types: view, edit, element_action, create, update, delete
    action TEXT NOT NULL,
    
    -- Element-specific actions
    element_id TEXT DEFAULT NULL,
    element_type TEXT DEFAULT NULL,
    
    -- Session information
    session_id TEXT DEFAULT NULL,
    ip_address TEXT DEFAULT NULL,
    user_agent TEXT DEFAULT NULL,
    
    -- Performance metrics
    duration_ms INTEGER DEFAULT NULL,
    
    -- Additional metadata stored as JSON
    metadata TEXT DEFAULT '{}',
    
    -- Timestamp
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_controller_dashboards_user_id ON controller_dashboards(user_id);
CREATE INDEX IF NOT EXISTS idx_controller_dashboards_category ON controller_dashboards(category);
CREATE INDEX IF NOT EXISTS idx_controller_dashboards_is_favorite ON controller_dashboards(is_favorite);
CREATE INDEX IF NOT EXISTS idx_controller_dashboards_created_at ON controller_dashboards(created_at);
CREATE INDEX IF NOT EXISTS idx_controller_dashboards_updated_at ON controller_dashboards(updated_at);

CREATE INDEX IF NOT EXISTS idx_controller_templates_user_id ON controller_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_controller_templates_category ON controller_templates(category);
CREATE INDEX IF NOT EXISTS idx_controller_templates_is_public ON controller_templates(is_public);
CREATE INDEX IF NOT EXISTS idx_controller_templates_usage_count ON controller_templates(usage_count);

CREATE INDEX IF NOT EXISTS idx_controller_shares_dashboard_id ON controller_shares(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_controller_shares_user_id ON controller_shares(user_id);
CREATE INDEX IF NOT EXISTS idx_controller_shares_shared_by ON controller_shares(shared_by);

CREATE INDEX IF NOT EXISTS idx_controller_usage_logs_dashboard_id ON controller_usage_logs(dashboard_id);
CREATE INDEX IF NOT EXISTS idx_controller_usage_logs_user_id ON controller_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_controller_usage_logs_action ON controller_usage_logs(action);
CREATE INDEX IF NOT EXISTS idx_controller_usage_logs_created_at ON controller_usage_logs(created_at);

-- Triggers for updating timestamps
CREATE TRIGGER IF NOT EXISTS update_controller_dashboards_timestamp 
    AFTER UPDATE ON controller_dashboards
    BEGIN
        UPDATE controller_dashboards SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_controller_templates_timestamp 
    AFTER UPDATE ON controller_templates
    BEGIN
        UPDATE controller_templates SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

-- Insert default/system dashboard template
INSERT OR IGNORE INTO controller_templates (
    id, name, description, category, template_json, variables_json, is_public, user_id
) VALUES (
    1,
    'Basic Controls',
    'A simple dashboard template with common smart home controls',
    'system',
    '{"layout":{"columns":12,"rows":8,"grid_size":64,"gap":8,"responsive":true},"elements":[{"type":"button","position":{"x":0,"y":0,"width":3,"height":2},"config":{"label":"Light Toggle","button":{"text":"Toggle Light","action_type":"entity_action"}},"style":{"background":"#374151","text_color":"#ffffff"},"behavior":{"enabled":true,"visible":true,"interactive":true}},{"type":"slider","position":{"x":3,"y":0,"width":4,"height":2},"config":{"label":"Brightness","min":0,"max":100,"step":1,"unit":"%"},"style":{"background":"#374151","text_color":"#ffffff"},"behavior":{"enabled":true,"visible":true,"interactive":true}},{"type":"display_value","position":{"x":7,"y":0,"width":3,"height":1},"config":{"label":"Temperature","format":"number","unit":"°C"},"style":{"background":"#1f2937","text_color":"#ffffff"},"behavior":{"enabled":true,"visible":true,"interactive":false}}]}',
    '[{"name":"room_name","type":"text","default":"Living Room","description":"Name of the room for this dashboard"},{"name":"light_entity","type":"entity","entity_type":"light","description":"Primary light entity to control"}]',
    TRUE,
    NULL
); 