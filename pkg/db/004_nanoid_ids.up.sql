-- Migration to change all IDs from INTEGER to TEXT (NanoID)
-- Use ALTER TABLE ... ALTER COLUMN for primary keys and foreign keys

-- 1. codemogger_codebases
ALTER TABLE codemogger_codebases ALTER COLUMN id TEXT;

-- 2. codemogger_chunks
ALTER TABLE codemogger_chunks ALTER COLUMN id TEXT;
ALTER TABLE codemogger_chunks ALTER COLUMN codebase_id TEXT;

-- 3. codemogger_indexed_files
ALTER TABLE codemogger_indexed_files ALTER COLUMN id TEXT;
ALTER TABLE codemogger_indexed_files ALTER COLUMN codebase_id TEXT;

-- 4. research_sessions
ALTER TABLE research_sessions ALTER COLUMN codebase_id TEXT;

-- 5. research_reports
ALTER TABLE research_reports ALTER COLUMN id TEXT;
