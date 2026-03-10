-- 006_system_codebases.down.sql

-- 1. Create original codemogger_codebases structure
CREATE TABLE IF NOT EXISTS codemogger_codebases_old (
    id TEXT PRIMARY KEY,
    root_path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    indexed_at INTEGER NOT NULL DEFAULT 0
);

-- 2. Migrate data back
INSERT INTO codemogger_codebases_old (id, root_path, name, indexed_at)
SELECT c.id, c.root_path, c.name, mc.indexed_at
FROM codebases c
JOIN codemogger_codebases mc ON mc.codebase_id = c.id;

-- 3. Cleanup new tables
DROP TABLE IF EXISTS codemogger_codebases;
DROP TABLE IF EXISTS codebases;

-- 4. Restore table name
ALTER TABLE codemogger_codebases_old RENAME TO codemogger_codebases;
