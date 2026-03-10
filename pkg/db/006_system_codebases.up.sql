-- 006_system_codebases.up.sql

-- Robust migration: Rename everything, create new, migrate, drop old.
-- This avoids "FOREIGN KEY constraint failed" when dropping parent tables.

PRAGMA foreign_keys=OFF;

-- 1. Rename existing tables to _old
-- We wrap in BEGIN/COMMIT or just run sequentially.
-- Check if tables exist before renaming is hard in SQL, so we rely on Migrator's execution.

ALTER TABLE codemogger_codebases RENAME TO old_codemogger_codebases;
ALTER TABLE codemogger_chunks RENAME TO old_codemogger_chunks;
ALTER TABLE codemogger_indexed_files RENAME TO old_codemogger_indexed_files;
ALTER TABLE research_sessions RENAME TO old_research_sessions;
ALTER TABLE research_reports RENAME TO old_research_reports;

-- 2. Create NEW System codebase table
CREATE TABLE IF NOT EXISTS codebases (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'local',
    version TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- 3. Create NEW Module-specific tables with updated FKs
CREATE TABLE IF NOT EXISTS codemogger_codebases (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    indexed_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY(codebase_id) REFERENCES codebases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS codemogger_chunks (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL, -- References codemogger_codebases(id)
    file_path TEXT NOT NULL,
    chunk_key TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL,
    kind TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    signature TEXT NOT NULL DEFAULT '',
    snippet TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    file_hash TEXT NOT NULL,
    indexed_at INTEGER NOT NULL,
    embedding BLOB,
    embedding_model TEXT DEFAULT '',
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS codemogger_indexed_files (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL, -- References codemogger_codebases(id)
    file_path TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codebase_id, file_path),
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS research_sessions (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL, -- References codebases(id)
    title TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    archived_at INTEGER,
    FOREIGN KEY(codebase_id) REFERENCES codebases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS research_reports (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL UNIQUE,
    stream_data TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY(session_id) REFERENCES research_sessions(id) ON DELETE CASCADE
);

-- 4. Re-create Indexes
CREATE INDEX IF NOT EXISTS idx_chunks_codebase_id ON codemogger_chunks(codebase_id);
CREATE INDEX IF NOT EXISTS idx_indexed_files_codebase_id ON codemogger_indexed_files(codebase_id);
CREATE INDEX IF NOT EXISTS idx_research_sessions_codebase_id ON research_sessions(codebase_id);
CREATE INDEX IF NOT EXISTS idx_research_reports_session_id ON research_reports(session_id);

-- 5. Migrate Data
-- System Codebases
INSERT INTO codebases (id, name, root_path, type, version, created_at)
SELECT id, name, root_path, 'local', '', indexed_at FROM old_codemogger_codebases;

-- Codemogger Metadata (Preserve ID to keep chunks/files linked)
INSERT INTO codemogger_codebases (id, codebase_id, indexed_at)
SELECT id, id, indexed_at FROM old_codemogger_codebases;

-- Chunks
INSERT INTO codemogger_chunks SELECT * FROM old_codemogger_chunks;

-- Indexed Files
INSERT INTO codemogger_indexed_files SELECT * FROM old_codemogger_indexed_files;

-- Research Sessions
INSERT INTO research_sessions SELECT * FROM old_research_sessions;

-- Research Reports
INSERT INTO research_reports SELECT * FROM old_research_reports;

-- 6. Re-create FTS table and triggers (as they reference codemogger_chunks)
DROP TABLE IF EXISTS codemogger_chunks_fts;
CREATE VIRTUAL TABLE codemogger_chunks_fts USING fts5(
    name,
    signature,
    snippet,
    content='codemogger_chunks'
);

DROP TRIGGER IF EXISTS codemogger_chunks_ai;
DROP TRIGGER IF EXISTS codemogger_chunks_ad;
DROP TRIGGER IF EXISTS codemogger_chunks_au;

DROP TRIGGER IF EXISTS codemogger_chunks_after_insert;
CREATE TRIGGER codemogger_chunks_after_insert AFTER INSERT ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet) VALUES (new.rowid, new.name, new.signature, new.snippet);
END;

DROP TRIGGER IF EXISTS codemogger_chunks_after_delete;
CREATE TRIGGER codemogger_chunks_after_delete AFTER DELETE ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(codemogger_chunks_fts, rowid, name, signature, snippet) VALUES('delete', old.rowid, old.name, old.signature, old.snippet);
END;

DROP TRIGGER IF EXISTS codemogger_chunks_after_update;
CREATE TRIGGER codemogger_chunks_after_update AFTER UPDATE ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(codemogger_chunks_fts, rowid, name, signature, snippet) VALUES('delete', old.rowid, old.name, old.signature, old.snippet);
  INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet) VALUES (new.rowid, new.name, new.signature, new.snippet);
END;

-- Populate FTS
INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet)
SELECT rowid, name, signature, snippet FROM codemogger_chunks;

-- 7. Drop OLD tables
DROP TABLE old_codemogger_chunks;
DROP TABLE old_codemogger_indexed_files;
DROP TABLE old_research_reports;
DROP TABLE old_research_sessions;
DROP TABLE old_codemogger_codebases;

PRAGMA foreign_keys=ON;
