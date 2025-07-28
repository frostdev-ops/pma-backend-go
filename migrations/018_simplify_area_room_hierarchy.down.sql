-- 018_simplify_area_room_hierarchy.down.sql
-- Rollback Area → Room → Entity hierarchy simplification

-- Remove the trigger
DROP TRIGGER IF EXISTS update_rooms_area_timestamp;

-- Recreate room_area_assignments table if it was dropped
CREATE TABLE IF NOT EXISTS room_area_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    area_id INTEGER NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    assignment_type TEXT DEFAULT 'primary',
    confidence_score REAL DEFAULT 1.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(room_id, area_id, assignment_type)
);

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_room_area_assignments_room ON room_area_assignments(room_id);
CREATE INDEX IF NOT EXISTS idx_room_area_assignments_area ON room_area_assignments(area_id);

-- Migrate data back from rooms.area_id to room_area_assignments
INSERT OR IGNORE INTO room_area_assignments (room_id, area_id, assignment_type, confidence_score)
SELECT id, area_id, 'primary', 1.0
FROM rooms
WHERE area_id IS NOT NULL;

-- Remove area_id column from rooms
DROP INDEX IF EXISTS idx_rooms_area_id;
ALTER TABLE rooms DROP COLUMN area_id; 