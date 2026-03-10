CREATE TABLE IF NOT EXISTS saved_reports (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    codebase_id TEXT NOT NULL,
    title TEXT NOT NULL,
    query TEXT NOT NULL,
    content TEXT NOT NULL,
    codebase_name TEXT NOT NULL,
    codebase_path TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

-- FTS table for searching saved reports
CREATE VIRTUAL TABLE IF NOT EXISTS saved_reports_fts USING fts5(
    id UNINDEXED,
    title,
    query,
    content,
    tokenize='trigram'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS saved_reports_ai AFTER INSERT ON saved_reports BEGIN
    INSERT INTO saved_reports_fts(id, title, query, content)
    VALUES (new.id, new.title, new.query, new.content);
END;

CREATE TRIGGER IF NOT EXISTS saved_reports_ad AFTER DELETE ON saved_reports BEGIN
    DELETE FROM saved_reports_fts WHERE id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS saved_reports_au AFTER UPDATE ON saved_reports BEGIN
    DELETE FROM saved_reports_fts WHERE id = old.id;
    INSERT INTO saved_reports_fts(id, title, query, content)
    VALUES (new.id, new.title, new.query, new.content);
END;
