-- Rollback migration
ALTER TABLE codemogger_codebases ALTER COLUMN id INTEGER;
ALTER TABLE codemogger_chunks ALTER COLUMN id INTEGER;
ALTER TABLE codemogger_chunks ALTER COLUMN codebase_id INTEGER;
ALTER TABLE codemogger_indexed_files ALTER COLUMN id INTEGER;
ALTER TABLE codemogger_indexed_files ALTER COLUMN codebase_id INTEGER;
ALTER TABLE research_sessions ALTER COLUMN codebase_id INTEGER;
ALTER TABLE research_reports ALTER COLUMN id INTEGER;
