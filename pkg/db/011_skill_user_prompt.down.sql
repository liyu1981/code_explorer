-- SQLite doesn't support DROP COLUMN easily before 3.35.0, 
-- but for simplicity let's just leave it or use a temp table if we really need to.
-- Since it's a new field, we can usually just ignore it.
ALTER TABLE skills DROP COLUMN user_prompt;
