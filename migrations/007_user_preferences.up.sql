-- User Preferences Migration
-- User customization including preferences, dashboards, themes, and localization

-- User preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id TEXT PRIMARY KEY,
    preferences JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- User dashboards table
CREATE TABLE IF NOT EXISTS user_dashboards (
    user_id TEXT PRIMARY KEY,
    dashboard JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Custom themes table
CREATE TABLE IF NOT EXISTS custom_themes (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    definition JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Notification subscriptions table
CREATE TABLE IF NOT EXISTS notification_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL,
    config JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Automation suggestions table
CREATE TABLE IF NOT EXISTS automation_suggestions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    suggestion JSON NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acted_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Widget analytics table
CREATE TABLE IF NOT EXISTS widget_analytics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    widget_id TEXT NOT NULL,
    widget_type TEXT NOT NULL,
    user_id TEXT NOT NULL,
    view_count INTEGER DEFAULT 0,
    interact_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    avg_load_time REAL DEFAULT 0.0,
    last_accessed TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Translation cache table (for dynamic translations)
CREATE TABLE IF NOT EXISTS translation_cache (
    locale TEXT NOT NULL,
    namespace TEXT DEFAULT 'default',
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (locale, namespace, key)
);

-- User locales table (to track user locale preferences)
CREATE TABLE IF NOT EXISTS user_locales (
    user_id TEXT PRIMARY KEY,
    locale_code TEXT NOT NULL DEFAULT 'en-US',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_preferences_updated ON user_preferences(updated_at);
CREATE INDEX IF NOT EXISTS idx_dashboards_updated ON user_dashboards(updated_at);
CREATE INDEX IF NOT EXISTS idx_custom_themes_user ON custom_themes(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_subscriptions_user ON notification_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_user_status ON automation_suggestions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_widget_analytics_user ON widget_analytics(user_id);
CREATE INDEX IF NOT EXISTS idx_widget_analytics_widget ON widget_analytics(widget_id);
CREATE INDEX IF NOT EXISTS idx_translation_cache_locale ON translation_cache(locale);
CREATE INDEX IF NOT EXISTS idx_user_locales_locale ON user_locales(locale_code); 