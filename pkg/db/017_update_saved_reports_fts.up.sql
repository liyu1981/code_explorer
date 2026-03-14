-- Update FTS table to use stream_data instead of content
DROP TRIGGER IF EXISTS saved_reports_ai;
DROP TRIGGER IF EXISTS saved_reports_ad;
DROP TRIGGER IF EXISTS saved_reports_au;
DROP TABLE IF EXISTS saved_reports_fts;

CREATE VIRTUAL TABLE IF NOT EXISTS saved_reports_fts USING fts5(
    id,
    title,
    query,
    stream_data,
    content='saved_reports'
);

CREATE TRIGGER IF NOT EXISTS saved_reports_ai AFTER INSERT ON saved_reports BEGIN
    INSERT INTO saved_reports_fts(id, title, query, stream_data)
    VALUES (new.id, new.title, new.query, new.stream_data);
END;

CREATE TRIGGER IF NOT EXISTS saved_reports_ad AFTER DELETE ON saved_reports BEGIN
    DELETE FROM saved_reports_fts WHERE id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS saved_reports_au AFTER UPDATE ON saved_reports BEGIN
    DELETE FROM saved_reports_fts WHERE id = old.id;
    INSERT INTO saved_reports_fts(id, title, query, stream_data)
    VALUES (new.id, new.title, new.query, new.stream_data);
END;
