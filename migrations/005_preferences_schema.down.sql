-- Drop indexes first
DROP INDEX IF EXISTS idx_user_locales_locale;
DROP INDEX IF EXISTS idx_translation_cache_locale;
DROP INDEX IF EXISTS idx_widget_analytics_widget;
DROP INDEX IF EXISTS idx_widget_analytics_user;
DROP INDEX IF EXISTS idx_suggestions_user_status;
DROP INDEX IF EXISTS idx_notification_subscriptions_user;
DROP INDEX IF EXISTS idx_custom_themes_user;
DROP INDEX IF EXISTS idx_dashboards_updated;
DROP INDEX IF EXISTS idx_preferences_updated;

-- Drop tables
DROP TABLE IF EXISTS user_locales;
DROP TABLE IF EXISTS translation_cache;
DROP TABLE IF EXISTS widget_analytics;
DROP TABLE IF EXISTS automation_suggestions;
DROP TABLE IF EXISTS notification_subscriptions;
DROP TABLE IF EXISTS custom_themes;
DROP TABLE IF EXISTS user_dashboards;
DROP TABLE IF EXISTS user_preferences; 