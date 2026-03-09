CREATE TABLE IF NOT EXISTS research_sessions (
    id TEXT PRIMARY KEY,
    codebase_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    archived_at INTEGER,
    FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
);

CREATE TABLE IF NOT EXISTS research_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    type TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    data TEXT NOT NULL,
    FOREIGN KEY(session_id) REFERENCES research_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_research_events_session_id ON research_events(session_id);
CREATE INDEX IF NOT EXISTS idx_research_sessions_codebase_id ON research_sessions(codebase_id);
