# Plan: System-Wide Codebase Refactor (Decoupled Architecture)

## 1. Goal
Decouple the core definition of a "Codebase" from the specific indexing metadata of the `codemogger` module. This allows other future modules (e.g., test runners, deployers) to reference a system-level codebase without being tied to indexing attributes.

## 2. Database Schema Evolution

### Core System Table: `codebases`
This table serves as the system-wide source of truth for all codebase entities.
```sql
CREATE TABLE IF NOT EXISTS codebases (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'local', -- 'local', 'github', etc.
    version TEXT NOT NULL DEFAULT '',    -- commit hash or version string
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
```

### Module Table: `codemogger_codebases`
This table stores metadata specific to the `codemogger` indexing module.
```sql
CREATE TABLE IF NOT EXISTS codemogger_codebases (
    id TEXT PRIMARY KEY,                  -- Specific ID for this codemogger profile
    codebase_id TEXT NOT NULL,           -- Reference to system codebase
    indexed_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY(codebase_id) REFERENCES codebases(id) ON DELETE CASCADE
);
```

## 3. Go Backend Refactoring (Separated Concerns)

### Core System Store (`pkg/db/codebase_store.go`) - **NEW**
Dedicated file for system-level codebase operations:
- `GetOrCreateCodebase(params)`: Creates or retrieves a record in the `codebases` table.
- `ListCodebases()`: Returns a list of all defined `Codebase` system entities.
- `GetCodebaseByID(id)`: Retrieves a single system codebase definition.
- `UpdateCodebaseVersion(id, version)`: Updates the version/commit hash.

### Codemogger Module Store (`pkg/db/codemogger_store.go`) - **REFACTOR**
Focus only on indexing-specific metadata:
- `CodemoggerGetMetadataByCodebase(codebaseID)`: Retrieves `indexed_at`, file counts, and chunk counts.
- `CodemoggerEnsureMetadata(codebaseID)`: Ensures a `codemogger_codebases` record exists for a given system codebase.
- `CodemoggerTouch(codebaseID)`: Updates `indexed_at` in the module table.

### Schema Models (`pkg/db/codemogger_schema.go`)
- Update `Codebase` struct to match the system definition.
- Introduce `CodemoggerMetadata` for the indexing-specific attributes.

## 4. Frontend Architecture Update

### UI Strategy (Two-Stage Loading)
Shift from a single "God-object" query to a compositional approach in `frontend/src/app/new/_components/codebase-list.tsx`:
1. **Primary Fetch:** Call `GET /api/codebases` to retrieve the system-level list (Name, Path, Type, Version).
2. **Enrichment Fetch:** For each codebase, fetch indexing status from `GET /api/codemogger/status?codebase_id=...` to populate "Files", "Chunks", and "Last Indexed".
3. **State Management:** Update `research-store.ts` to separate `SystemCodebase` from `IndexingMetadata`.

## 5. Integration & Logic
- **Indexer Update:** `pkg/codemogger` will now accept a `CodebaseID`. It will update `version` in the system store and `indexed_at` in the codemogger store upon completion.
- **Version Detection:** Add logic to detect `.git` for versioning (extracting the HEAD commit hash) during indexing.

## 6. Migration Strategy (`006_system_codebases.up.sql`)
1. Rename current `codemogger_codebases` to `codemogger_codebases_old`.
2. Create the new `codebases` and `codemogger_codebases` tables.
3. Migrate existing records:
   - Insert `root_path`, `name`, `id` into `codebases`.
   - Insert `id`, `codebase_id` (using the same original ID), and `indexed_at` into `codemogger_codebases`.
4. This preserves foreign key integrity for `codemogger_chunks` and `research_sessions` which already reference the original ID.
5. Drop `codemogger_codebases_old`.
