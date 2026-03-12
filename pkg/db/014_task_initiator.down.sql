DROP INDEX IF EXISTS idx_tasks_initiator_id;
-- SQLite does not support DROP COLUMN easily, but we can leave it or recreate the table if needed.
-- For simple migrations we often just leave it.
