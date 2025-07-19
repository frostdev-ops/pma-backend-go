# PMA Migration Scripts

This directory contains scripts to help migrate data to the PMA format.

## migrate_to_pma.go

This script migrates existing entities to the PMA format by:

1. Adding metadata entries for all existing entities
2. Determining the source adapter based on entity ID patterns
3. Setting default values for new PMA columns

### Usage

```bash
# Build the migration script
cd migrations/scripts
go build -o migrate_to_pma migrate_to_pma.go

# Run with default database path (./pma.db)
./migrate_to_pma

# Run with custom database path
./migrate_to_pma /path/to/your/database.db
```

### Prerequisites

1. Run migration 017 first to create the entity_metadata table:
   ```sql
   -- Apply migration 017_pma_entity_metadata.up.sql
   ```

2. Ensure you have a backup of your database before running the migration

### What it does

- Scans all entities that don't have metadata entries
- Determines source adapter based on entity ID patterns:
  - `ring*` → ring adapter
  - `shel*` → shelly adapter  
  - `ups_*` or contains "ups" → ups adapter
  - `net_*` or contains "network" → network adapter
  - contains "bluetooth" → bluetooth adapter
  - contains "camera" → camera adapter
  - default → homeassistant adapter
- Creates metadata entries with migration flags
- Sets default values for new PMA columns (available=true, capabilities=[])

### Output

The script will log progress and report:
- Number of entities migrated
- Any errors encountered
- Final summary of migration results 