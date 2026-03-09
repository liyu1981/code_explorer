-- Revert FTS triggers (might still be using rowid as it's safer)
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
