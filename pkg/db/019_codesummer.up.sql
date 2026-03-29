-- codesummer metadata
CREATE TABLE IF NOT EXISTS codesummer_codebases (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    indexed_at INTEGER NOT NULL DEFAULT 0
);

-- summaries for files/directories
CREATE TABLE IF NOT EXISTS codesummer_summaries (
    id TEXT PRIMARY KEY,
    codesummer_id TEXT NOT NULL,
    node_path TEXT NOT NULL,
    node_type TEXT NOT NULL,
    language TEXT,
    summary TEXT NOT NULL,
    definitions TEXT NOT NULL,
    dependencies TEXT NOT NULL,
    data_manipulated TEXT NOT NULL,
    data_flow TEXT NOT NULL,
    embedding BLOB,
    embedding_model TEXT,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codesummer_id, node_path)
);

-- track indexed paths to detect changes
CREATE TABLE IF NOT EXISTS codesummer_indexed_paths (
    id TEXT PRIMARY KEY,
    codesummer_id TEXT NOT NULL,
    node_path TEXT NOT NULL,
    node_type TEXT NOT NULL,
    file_hash TEXT,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codesummer_id, node_path)
);

-- indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_codesummer_summaries_codesummer_id ON codesummer_summaries(codesummer_id);
CREATE INDEX IF NOT EXISTS idx_codesummer_indexed_paths_codesummer_id ON codesummer_indexed_paths(codesummer_id);
