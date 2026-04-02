CREATE TABLE IF NOT EXISTS fs_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    parent_id INTEGER,
    type TEXT CHECK(type IN ('file', 'dir')) NOT NULL,
    size INTEGER DEFAULT 0,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),
    UNIQUE(parent_id, name),
    FOREIGN KEY (parent_id) REFERENCES fs_nodes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fs_nodes_parent ON fs_nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_fs_nodes_parent_name ON fs_nodes(parent_id, name);

CREATE TABLE IF NOT EXISTS fs_file_chunks (
    file_id INTEGER NOT NULL,
    chunk_index INTEGER NOT NULL,
    data BLOB NOT NULL,
    PRIMARY KEY (file_id, chunk_index),
    FOREIGN KEY (file_id) REFERENCES fs_nodes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fs_file_chunks_file ON fs_file_chunks(file_id);