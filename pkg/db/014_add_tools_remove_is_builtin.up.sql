-- Add tools column
ALTER TABLE agent_skills ADD COLUMN tools TEXT NOT NULL DEFAULT '';

-- Remove is_builtin column
ALTER TABLE agent_skills DROP COLUMN is_builtin;
