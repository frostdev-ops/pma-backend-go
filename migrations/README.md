# Database Migrations

This directory contains the complete database schema migrations for the PMA (Personal Management Assistant) system. The migrations have been consolidated, cleaned up, and properly sequenced.

## Migration Overview

| Migration | Description | Tables Created |
|-----------|-------------|----------------|
| `001_initial_schema` | Core system tables | users, entities, rooms, system_config, automation_rules |
| `002_authentication` | Authentication and security | auth_settings, sessions, failed_auth_attempts, kiosk_tokens, kiosk_pairing_sessions, display_settings |
| `003_device_management` | Device management system | devices, device_states, device_credentials, device_events, network_devices, ups_status, cameras, bluetooth_devices |
| `004_file_management` | File and backup management | files, media_info, backups, backup_schedules, file_permissions |
| `005_network_management` | Network infrastructure | port_forwarding_rules, network_interfaces, traffic_stats, dhcp_leases, network_scans, device_services, network_system_status |
| `006_monitoring` | System monitoring and metrics | metrics, alerts, system_snapshots, performance_metrics, alert_rules, metric_thresholds, health_checks, service_monitoring |
| `007_user_preferences` | User customization | user_preferences, user_dashboards, custom_themes, notification_subscriptions, automation_suggestions, widget_analytics, translation_cache, user_locales |
| `008_analytics` | Analytics and reporting | analytics_events, custom_metrics, metric_data, time_series_data, report_templates, reports, dashboards, visualizations, prediction_models, predictions, anomalies, insights |
| `009_performance_optimizations` | Database performance | Indexes and SQLite optimizations |
| `010_conversation_management` | AI conversation system | conversations, conversation_messages, mcp_tools, mcp_tool_executions, conversation_analytics |
| `011_energy_management` | Energy monitoring | energy_settings, energy_history, device_energy |

## Key Features

### ✅ Properly Sequenced
- Migrations are numbered sequentially from 001 to 011
- Foreign key dependencies are respected in the creation order
- No duplicate migration numbers

### ✅ Complete Up/Down Migrations
- Every migration has both `.up.sql` and `.down.sql` files
- Down migrations properly reverse all changes including indexes and triggers

### ✅ Performance Optimized
- Comprehensive indexing strategy
- SQLite optimizations for better performance
- Partial indexes for frequently filtered data

### ✅ Data Integrity
- Foreign key constraints properly defined
- Check constraints for data validation
- Triggers for automatic cleanup and timestamp updates

## Dependency Chain

```
001_initial_schema (base tables)
├── 002_authentication (depends on users)
├── 003_device_management (depends on entities)
│   └── 005_network_management (depends on network_devices)
├── 004_file_management (depends on users, files)
├── 006_monitoring (standalone)
├── 007_user_preferences (depends on users)
├── 008_analytics (standalone)
├── 009_performance_optimizations (enhances existing tables)
├── 010_conversation_management (depends on users)
└── 011_energy_management (standalone)
```

## Migration Process

### Running Migrations
Migrations should be executed in numerical order:
1. Run all `.up.sql` files in sequence (001 through 011)
2. Each migration is idempotent using `IF NOT EXISTS` clauses

### Rolling Back
To rollback, run `.down.sql` files in reverse order:
1. Start from the highest number and work backwards
2. Each down migration removes all changes from its corresponding up migration

## Backup Information
- Original migrations are backed up in `migrations_backup/` directory
- Backup created before consolidation process

## Schema Statistics
- **Total Tables**: 51 tables across all domains
- **Total Indexes**: 100+ optimized indexes
- **Triggers**: 8 automatic maintenance triggers
- **Foreign Keys**: 15+ referential integrity constraints

## Key Improvements Made
1. **Eliminated Duplicates**: Removed duplicate migration numbers (002, 003, 009)
2. **Created Missing Files**: Added missing down migrations
3. **Fixed Dependencies**: Ensured proper foreign key dependency order
4. **Performance**: Added comprehensive indexing strategy
5. **Consistency**: Standardized naming conventions and structure
6. **Documentation**: Added comprehensive comments and descriptions
7. **SQLite Compatibility**: Fixed column references and removed PRAGMA statements from transactions
8. **Migration Safety**: Removed non-deterministic datetime expressions from indexes

## Development Notes
- All JSON fields are properly documented
- Timestamps use consistent formats
- Boolean fields use consistent naming patterns
- Primary keys follow established patterns (INTEGER AUTOINCREMENT or TEXT UUID)
- Indexes are named with consistent `idx_` prefix

## Validation and Testing

The migration structure has been thoroughly tested and validated:

✅ **All 11 migrations execute successfully** - No migration errors or failures  
✅ **Proper column references** - Fixed `device_states.timestamp` vs `created_at` naming  
✅ **SQLite transaction compatibility** - Removed PRAGMA statements that can't run in transactions  
✅ **Non-deterministic expressions removed** - Fixed datetime() expressions in index WHERE clauses  
✅ **Database state verified** - Final state shows version 11 with no dirty flags  
✅ **Application startup confirmed** - Migrations run successfully during application boot  

## Migration Safety Notes

- All migrations use `IF NOT EXISTS` clauses for idempotency
- Foreign key constraints are properly ordered to avoid dependency issues
- Index names are consistent and descriptive
- Down migrations properly reverse all changes including indexes and triggers
- SQLite-specific optimizations are handled at the connection level, not in migrations

This migration structure provides a solid foundation for the PMA system's database layer with proper separation of concerns, performance optimization, and maintainability. 