DROP TABLE IF EXISTS entity_metadata;

-- Note: SQLite doesn't support dropping columns directly
-- The columns will remain but can be ignored
-- ALTER TABLE entities DROP COLUMN pma_capabilities;
-- ALTER TABLE entities DROP COLUMN available; 