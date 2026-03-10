-- 006_system_codebases.up.sql

-- Disable foreign keys to allow table replacement
PRAGMA foreign_keys=OFF;

-- 1. Create temporary table to handle migration safely
CREATE TABLE IF NOT EXISTS codemogger_codebases_temp (
    id TEXT PRIMARY KEY,
    root_path TEXT,
    name TEXT,
    indexed_at INTEGER
);

-- 2. Try to copy data from original table if it exists
INSERT INTO codemogger_codebases_temp (id, root_path, name, indexed_at)
SELECT id, root_path, name, indexed_at FROM codemogger_codebases WHERE 1=1;

-- 3. Drop the old table
DROP TABLE IF EXISTS codemogger_codebases;

-- 4. Create new system codebase table
CREATE TABLE IF NOT EXISTS codebases (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'local',
    version TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- 5. Create new codemogger_codebases table
CREATE TABLE IF NOT EXISTS codemogger_codebases (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    indexed_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY(codebase_id) REFERENCES codebases(id) ON DELETE CASCADE
);

-- 6. Migrate data from temp to new tables
INSERT INTO codebases (id, name, root_path, type, version, created_at)
SELECT id, COALESCE(name, ''), COALESCE(root_path, ''), 'local', '', COALESCE(indexed_at, 0) 
FROM codemogger_codebases_temp 
WHERE root_path IS NOT NULL;

INSERT INTO codemogger_codebases (id, codebase_id, indexed_at)
SELECT id, id, COALESCE(indexed_at, 0) 
FROM codemogger_codebases_temp 
WHERE root_path IS NOT NULL;

-- 7. Drop temp table
DROP TABLE IF EXISTS codemogger_codebases_temp;

-- Re-enable foreign keys
PRAGMA foreign_keys=ON;
