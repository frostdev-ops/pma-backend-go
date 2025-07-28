-- Revert user management functionality

-- Drop indexes
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_created_at;
 
-- Note: We don't drop the users table as it might be used by other parts of the system
-- The users table was created in the initial migration 