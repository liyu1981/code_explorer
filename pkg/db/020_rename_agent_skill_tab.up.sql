-- 1. Create the new table with updated schema
CREATE TABLE IF NOT EXISTS "agent_prompts" (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    system_prompt TEXT NOT NULL,
    user_prompt_tpl TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    tags TEXT NOT NULL DEFAULT '',
    tools TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_agent_prompts_name ON agent_prompts(name);

-- 2. Migrate existing data (user_prompt_tpl defaults to '')
INSERT INTO agent_prompts (id, name, system_prompt, created_at, updated_at, tags, tools)
SELECT id, name, system_prompt, created_at, updated_at, tags, tools
FROM agent_skills;

-- 3. Drop old table and index
DROP INDEX IF EXISTS idx_agent_skills_name;
DROP TABLE IF EXISTS agent_skills;