"use client";

import { Book, ChevronRight, Hash, History, Search } from "lucide-react";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { KnowledgeViewer } from "../knowledge/_components/knowledge-viewer";
import { cn } from "@/lib/utils";
import { useState } from "react";

const MOCK_CONTENT = `
# System Architecture Overviews

This document outlines the core architectural patterns and data flows within the **Code Explorer** system.

## 1. High-Level Workflow

The system operates as a distributed agentic platform for deep codebase analysis.

\`\`\`mermaid
graph TD
    User((User)) -->|Research Query| API[Go API Server]
    API -->|Queue Task| W[Task Worker Pool]
    W -->|Index| CM[Code Mogger]
    W -->|Iterative Analysis| RA[Research Agent]
    RA -->|Vector Search| CM
    RA -->|Read Files| FS[Local Filesystem]
    RA -->|Refine Answer| LLM[LLM Provider]
    RA -->|Store Report| DB[(SQLite/libsql)]
\`\`\`

## 2. Core Components

### Task Queue System
The task queue uses a persistent SQLite backend to ensure job reliability across restarts.

| Component | Responsibility | Persistence |
| :--- | :--- | :--- |
| **Manager** | Orchestrates workers and job claiming | In-memory + DB |
| **Worker** | Executes individual task handlers | Transient |
| **Store** | Atomic job status updates | SQLite (libsql) |

### Code Mogger (Indexing)
Indexes are generated using a sliding window chunking strategy.

\`\`\`go
func (idx *CodeIndex) IndexFile(ctx context.Context, path string) error {
    // 1. Read file
    content, _ := os.ReadFile(path)
    
    // 2. Generate chunks
    chunks := chunk.Split(content, 150)
    
    // 3. Embed and store
    for _, c := range chunks {
        vec := idx.embedder.Embed(c)
        idx.store.SaveChunk(path, c, vec)
    }
    return nil
}
\`\`\`

## 3. Data Relationships

> [!IMPORTANT]
> All primary keys use Nanoids for better URL safety and distributed generation without coordination.

1. **Codebase** has many **Sessions**
2. **Session** has many **Reports** (turns)
3. **Codebase** has many **Knowledge Pages**
4. **Knowledge Page** is uniquely identified by \`(codebase_id, slug)\`

### Database Schema Snippet
\`\`\`sql
CREATE TABLE knowledge_pages (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
\`\`\`

---

## 4. Key Performance Indicators

* **P95 Latency:** < 500ms for vector search.
* **Worker Throughput:** ~50 files/sec during initial indexing.
* **DB Contention:** Managed via WAL mode and 5s busy timeout.
`;

const SIDEBAR_ITEMS = [
  { slug: "arch-overview", title: "Architecture Overview", active: true },
  { slug: "api-reference", title: "API Reference", active: false },
  { slug: "deployment", title: "Deployment Guide", active: false },
  { slug: "security", title: "Security Best Practices", active: false },
];

export default function KnowledgeMockPage() {
  const [activeSlug, setActiveSlug] = useState("arch-overview");

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center gap-4">
            <Book className="h-5 w-5 text-primary" />
            <div className="flex items-center gap-2">
              <span className="text-xl font-bold tracking-tight text-primary">
                Knowledge
              </span>
              <span className="px-2 py-0.5 rounded-md bg-muted text-[10px] font-bold text-muted-foreground uppercase tracking-widest">
                code-explorer
              </span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search knowledge..."
                className="bg-muted/50 border border-border/50 rounded-xl pl-9 pr-4 py-1.5 text-xs font-medium focus:ring-2 focus:ring-primary/20 outline-none w-64"
              />
            </div>
          </div>
        </div>
      </AppHeader>

      <div className="flex-1 flex overflow-hidden">
        {/* Navigation Sidebar */}
        <div className="w-72 border-r border-border/40 bg-muted/5 flex flex-col">
          <div className="p-4 border-b border-border/20">
            <h3 className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground px-2 mb-4">
              Documentation
            </h3>
            <div className="space-y-1">
              {SIDEBAR_ITEMS.map((item) => (
                <button
                  key={item.slug}
                  onClick={() => setActiveSlug(item.slug)}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-semibold transition-all group",
                    activeSlug === item.slug
                      ? "bg-primary/10 text-primary shadow-sm"
                      : "hover:bg-muted text-muted-foreground hover:text-foreground",
                  )}
                >
                  <Hash
                    className={cn(
                      "h-3.5 w-3.5",
                      activeSlug === item.slug
                        ? "text-primary"
                        : "text-muted-foreground/40 group-hover:text-muted-foreground",
                    )}
                  />
                  {item.title}
                  {activeSlug === item.slug && (
                    <ChevronRight className="h-3.5 w-3.5 ml-auto opacity-50" />
                  )}
                </button>
              ))}
            </div>
          </div>

          <div className="p-4 mt-auto">
            <div className="bg-card border border-border/50 rounded-2xl p-4 shadow-sm">
              <div className="flex items-center gap-2 mb-2">
                <History className="h-3.5 w-3.5 text-primary" />
                <span className="text-[10px] font-bold uppercase tracking-widest text-foreground">
                  Recent Changes
                </span>
              </div>
              <p className="text-[10px] text-muted-foreground leading-relaxed">
                Last updated by **liyu1981** about 2 hours ago.
              </p>
            </div>
          </div>
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-auto bg-background/50">
          <div className="max-w-5xl mx-auto p-12 lg:p-16">
            <KnowledgeViewer content={MOCK_CONTENT} />

            <div className="mt-20 pt-12 border-t border-border/20 flex items-center justify-end text-muted-foreground">
              <p className="text-[10px] font-mono opacity-50">
                PAGE_ID: {activeSlug.toUpperCase()}_VERSION_1
              </p>
            </div>
          </div>
        </div>
      </div>
    </AppContainer>
  );
}
