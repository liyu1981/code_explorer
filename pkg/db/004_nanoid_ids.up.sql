-- Migration to change all IDs from INTEGER to TEXT (NanoID)
-- SQLite standard procedure: create new table, copy data, drop old table, rename.

PRAGMA foreign_keys=OFF;

-- 1. codemogger_codebases
CREATE TABLE new_codemogger_codebases (
    id TEXT PRIMARY KEY,
    root_path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    indexed_at INTEGER NOT NULL DEFAULT 0
);
INSERT INTO new_codemogger_codebases SELECT CAST(id AS TEXT), root_path, name, indexed_at FROM codemogger_codebases;
DROP TABLE codemogger_codebases;
ALTER TABLE new_codemogger_codebases RENAME TO codemogger_codebases;

-- 2. codemogger_chunks
CREATE TABLE new_codemogger_chunks (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
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
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
);
INSERT INTO new_codemogger_chunks SELECT CAST(id AS TEXT), CAST(codebase_id AS TEXT), file_path, chunk_key, language, kind, name, signature, snippet, start_line, end_line, file_hash, indexed_at, embedding, embedding_model FROM codemogger_chunks;
DROP TABLE codemogger_chunks;
ALTER TABLE new_codemogger_chunks RENAME TO codemogger_chunks;
CREATE INDEX IF NOT EXISTS idx_chunks_codebase_id ON codemogger_chunks(codebase_id);

-- Re-create FTS table and triggers for FTS using rowid
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

CREATE TRIGGER codemogger_chunks_ai AFTER INSERT ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet) VALUES (new.rowid, new.name, new.signature, new.snippet);
END;

CREATE TRIGGER codemogger_chunks_ad AFTER DELETE ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(codemogger_chunks_fts, rowid, name, signature, snippet) VALUES('delete', old.rowid, old.name, old.signature, old.snippet);
END;

CREATE TRIGGER codemogger_chunks_au AFTER UPDATE ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(codemogger_chunks_fts, rowid, name, signature, snippet) VALUES('delete', old.rowid, old.name, old.signature, old.snippet);
  INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet) VALUES (new.rowid, new.name, new.signature, new.snippet);
END;

-- Populate FTS with existing data
INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet)
SELECT rowid, name, signature, snippet FROM codemogger_chunks;

-- 3. codemogger_indexed_files
CREATE TABLE new_codemogger_indexed_files (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codebase_id, file_path),
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
);
INSERT INTO new_codemogger_indexed_files SELECT CAST(id AS TEXT), CAST(codebase_id AS TEXT), file_path, file_hash, chunk_count, indexed_at FROM codemogger_indexed_files;
DROP TABLE codemogger_indexed_files;
ALTER TABLE new_codemogger_indexed_files RENAME TO codemogger_indexed_files;
CREATE INDEX IF NOT EXISTS idx_indexed_files_codebase_id ON codemogger_indexed_files(codebase_id);

-- 4. research_sessions
CREATE TABLE new_research_sessions (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    title TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    archived_at INTEGER,
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
);
INSERT INTO new_research_sessions SELECT id, CAST(codebase_id AS TEXT), title, state, created_at, archived_at FROM research_sessions;
DROP TABLE research_sessions;
ALTER TABLE new_research_sessions RENAME TO research_sessions;
CREATE INDEX IF NOT EXISTS idx_research_sessions_codebase_id ON research_sessions(codebase_id);

-- 5. research_reports
CREATE TABLE new_research_reports (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    turn_id TEXT NOT NULL UNIQUE,
    stream_data TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY(session_id) REFERENCES research_sessions(id) ON DELETE CASCADE
);
INSERT INTO new_research_reports SELECT CAST(id AS TEXT), session_id, turn_id, stream_data, created_at, updated_at FROM research_reports;
DROP TABLE research_reports;
ALTER TABLE new_research_reports RENAME TO research_reports;
CREATE INDEX IF NOT EXISTS idx_research_reports_session_id ON research_reports(session_id);

PRAGMA foreign_keys=ON;
