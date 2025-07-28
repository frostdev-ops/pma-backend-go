-- 018_simplify_area_room_hierarchy.up.sql
-- Simplify Area → Room → Entity hierarchy

-- Add area_id directly to rooms table for simple one-to-many relationship
ALTER TABLE rooms ADD COLUMN area_id INTEGER REFERENCES areas(id) ON DELETE SET NULL;

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_rooms_area_id ON rooms(area_id);

-- Migrate existing room-area assignments to the new structure
-- Take the primary assignment for each room
UPDATE rooms 
SET area_id = (
    SELECT raa.area_id 
    FROM room_area_assignments raa 
    WHERE raa.room_id = rooms.id 
    AND raa.assignment_type = 'primary' 
    LIMIT 1
);

-- For rooms without primary assignments, use any assignment
UPDATE rooms 
SET area_id = (
    SELECT raa.area_id 
    FROM room_area_assignments raa 
    WHERE raa.room_id = rooms.id 
    AND rooms.area_id IS NULL
    ORDER BY raa.confidence_score DESC 
    LIMIT 1
)
WHERE area_id IS NULL;

-- Remove the complex many-to-many assignment table (backup data first in case rollback needed)
-- DROP TABLE room_area_assignments; -- Commented out for safety - can be removed later

-- Add trigger to update room's updated_at when area changes
CREATE TRIGGER IF NOT EXISTS update_rooms_area_timestamp 
    AFTER UPDATE OF area_id ON rooms
    BEGIN
        UPDATE rooms SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END; 