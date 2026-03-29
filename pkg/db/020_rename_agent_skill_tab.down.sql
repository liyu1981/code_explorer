-- 1. Recreate the original agent_skills table
CREATE TABLE IF NOT EXISTS "agent_skills" (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    system_prompt TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    tags TEXT NOT NULL DEFAULT '',
    tools TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_agent_skills_name ON agent_skills(name);

-- 2. Migrate data back (dropping user_prompt_tpl)
INSERT INTO agent_skills (id, name, system_prompt, created_at, updated_at, tags, tools)
SELECT id, name, system_prompt, created_at, updated_at, tags, tools
FROM agent_prompts;

-- 3. Drop the new table and index
DROP INDEX IF EXISTS idx_agent_prompts_name;
DROP TABLE IF EXISTS agent_prompts;
