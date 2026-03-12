-- Add tags column
ALTER TABLE skills ADD COLUMN tags TEXT NOT NULL DEFAULT '';

-- Remove user_prompt if we want to be clean, but SQLite doesn't support DROP COLUMN in older versions easily.
-- However, newer SQLite (3.35.0+) supports it. Since we are using modern tools, let's try it.
-- If it fails, we might need a more complex migration or just ignore the column.
-- Let's check the current Go library version support.
-- User said: "we do not need a user prompt(goal) thing ... nor need the user field in db now"
-- There is no "user" field in skills table in 010, so maybe they meant user_prompt.

ALTER TABLE skills DROP COLUMN user_prompt;
