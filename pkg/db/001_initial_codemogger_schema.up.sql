CREATE TABLE IF NOT EXISTS codemogger_codebases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    root_path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    indexed_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS codemogger_chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    codebase_id INTEGER NOT NULL,
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

CREATE TABLE IF NOT EXISTS codemogger_indexed_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    codebase_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codebase_id, file_path),
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
);

CREATE INDEX IF NOT EXISTS idx_chunks_codebase_id ON codemogger_chunks(codebase_id);
CREATE INDEX IF NOT EXISTS idx_indexed_files_codebase_id ON codemogger_indexed_files(codebase_id);

CREATE VIRTUAL TABLE IF NOT EXISTS codemogger_chunks_fts USING fts5(
    name,
    signature,
    snippet,
    content='codemogger_chunks',
    content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS codemogger_chunks_ai AFTER INSERT ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet) VALUES (new.id, new.name, new.signature, new.snippet);
END;

CREATE TRIGGER IF NOT EXISTS codemogger_chunks_ad AFTER DELETE ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(codemogger_chunks_fts, rowid, name, signature, snippet) VALUES('delete', old.id, old.name, old.signature, old.snippet);
END;

CREATE TRIGGER IF NOT EXISTS codemogger_chunks_au AFTER UPDATE ON codemogger_chunks BEGIN
  INSERT INTO codemogger_chunks_fts(codemogger_chunks_fts, rowid, name, signature, snippet) VALUES('delete', old.id, old.name, old.signature, old.snippet);
  INSERT INTO codemogger_chunks_fts(rowid, name, signature, snippet) VALUES (new.id, new.name, new.signature, new.snippet);
END;
