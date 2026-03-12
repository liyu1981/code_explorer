ALTER TABLE skills RENAME TO agent_skills;
DROP INDEX IF EXISTS idx_skills_name;
CREATE INDEX IF NOT EXISTS idx_agent_skills_name ON agent_skills(name);
