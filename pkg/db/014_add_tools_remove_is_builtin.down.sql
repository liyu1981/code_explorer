-- Remove tools column
ALTER TABLE agent_skills DROP COLUMN tools;

-- Add is_builtin column
ALTER TABLE agent_skills ADD COLUMN is_builtin BOOLEAN NOT NULL DEFAULT 0;
