-- Create entity metadata table for PMA system
CREATE TABLE IF NOT EXISTS entity_metadata (
    entity_id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    source_entity_id TEXT NOT NULL,
    metadata TEXT, -- Changed from JSONB to TEXT for SQLite compatibility
    quality_score REAL DEFAULT 1.0,
    last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_virtual BOOLEAN DEFAULT FALSE,
    virtual_sources TEXT, -- Changed to TEXT for SQLite compatibility
    FOREIGN KEY (entity_id) REFERENCES entities(entity_id) ON DELETE CASCADE
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_entity_metadata_source ON entity_metadata(source);
CREATE INDEX IF NOT EXISTS idx_entity_metadata_quality ON entity_metadata(quality_score);
CREATE INDEX IF NOT EXISTS idx_entity_metadata_sync ON entity_metadata(last_synced);

-- Add PMA-specific columns to entities table if not exists
ALTER TABLE entities ADD COLUMN pma_capabilities TEXT; -- Changed to TEXT for SQLite
ALTER TABLE entities ADD COLUMN available BOOLEAN DEFAULT TRUE; 