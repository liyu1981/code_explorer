CREATE TABLE knowledge_pages (
    id TEXT PRIMARY KEY, -- nanoid
    codebase_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(codebase_id, slug),
    FOREIGN KEY (codebase_id) REFERENCES codebases(id) ON DELETE CASCADE
);

CREATE INDEX idx_knowledge_pages_codebase ON knowledge_pages(codebase_id);
CREATE INDEX idx_knowledge_pages_slug ON knowledge_pages(slug);
