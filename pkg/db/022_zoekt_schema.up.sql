CREATE TABLE IF NOT EXISTS zoekt_codebases (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codebase_id),
    FOREIGN KEY (codebase_id) REFERENCES codebases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS zoekt_indexed_files (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codebase_id, file_path),
    FOREIGN KEY (codebase_id) REFERENCES zoekt_codebases(id) ON DELETE CASCADE
);
