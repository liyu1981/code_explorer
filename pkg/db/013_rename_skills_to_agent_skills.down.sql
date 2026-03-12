ALTER TABLE agent_skills RENAME TO skills;
DROP INDEX IF EXISTS idx_agent_skills_name;
CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
