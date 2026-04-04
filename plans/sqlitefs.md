# SQLite Filesystem (pkg/sqlitefs) - Implementation Plan

## Goal

Create a SQLite-backed virtual filesystem optimized for read-heavy workloads, following the project's conventions (db, libsql, config packages).

---

## Architecture

### Package Structure

```
pkg/sqlitefs/
├── fs.go              # Main FS interface and implementation
├── store.go           # SQLite operations (embedded in fs.go for simplicity)
├── cache.go           # LRU cache for hot chunks
├── config.go          # Configuration (optional, extends global config)
└── fs_test.go         # Unit tests
```

### Dependencies

- `pkg/db` - For Store struct pattern and SQL utilities
- `pkg/libsql` - For opening SQLite databases
- `pkg/config` - Optional configuration (or use environment-based config)
- `github.com/hashicorp/golang-lru` - For LRU cache (or implement simple one)

---

## Schema Design

### 1. Filesystem Nodes Table

```sql
CREATE TABLE fs_nodes (
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

CREATE INDEX idx_fs_nodes_parent ON fs_nodes(parent_id);
CREATE INDEX idx_fs_nodes_parent_name ON fs_nodes(parent_id, name);
```

### 2. File Chunks Table

```sql
CREATE TABLE fs_file_chunks (
    file_id INTEGER NOT NULL,
    chunk_index INTEGER NOT NULL,
    data BLOB NOT NULL,
    
    PRIMARY KEY (file_id, chunk_index),
    FOREIGN KEY (file_id) REFERENCES fs_nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_fs_file_chunks_file ON fs_file_chunks(file_id);
```

---

## Core Design Decisions

### 1. Chunk Size: 4KB

- Follows the plan's recommendation (aligned to modern CPU cache line size)
- Good balance between row count and random read efficiency

### 2. SQLite PRAGMA Settings

```go
// Applied on database open
PRAGMA journal_mode = WAL
PRAGMA synchronous = NORMAL
PRAGMA cache_size = -200000  // 200MB
PRAGMA mmap_size = 1073741824  // 1GB
```

### 3. Path Resolution

- Traverse path component by component in Go code
- Single query per component: `SELECT id FROM fs_nodes WHERE parent_id = ? AND name = ?`

---

## API Design

Following the project's interface patterns:

```go
type FileSystem interface {
    // File operations
    Read(path string, offset int64, size int) ([]byte, error)
    Write(path string, offset int64, data []byte) error
    Create(path string, data []byte) error
    Delete(path string) error
    
    // Directory operations
    Mkdir(path string) error
    List(path string) ([]FileInfo, error)
    Exists(path string) (bool, error)
}

type FileInfo struct {
    Name    string
    Path    string
    IsDir   bool
    Size    int64
    Modified int64
}
```

---

## Implementation Pattern

### Singleton Pattern (following pkg/db)

```go
var (
    instance *SQLiteFS
    once     sync.Once
)

func GetFS() *SQLiteFS {
    once.Do(func() {
        instance = newSQLiteFS()
    })
    return instance
}

func OpenFS(db *sql.DB) *SQLiteFS {
    once.Do(func() {
        instance = &SQLiteFS{db: db}
    })
    return instance
}
```

### Configuration

```go
type SQLiteFSConfig struct {
    ChunkSize      int  // default 16384 (16KB)
    CacheSize      int  // number of chunks to cache
    EnableCache    bool // default true
}
```

Or simpler - use constants with environment override:
- `SQLITEFS_CHUNK_SIZE` (default 4096)
- `SQLITEFS_CACHE_SIZE` (default 1000)

---

## Read Path (Optimized for reads)

```go
func (fs *SQLiteFS) Read(path string, offset int64, size int) ([]byte, error) {
    // 1. Resolve path to file_id
    fileID, err := fs.resolvePath(path)
    if err != nil {
        return nil, err
    }
    
    // 2. Calculate chunk range
    startChunk := offset / fs.chunkSize
    endChunk := (offset + int64(size) - 1) / fs.chunkSize
    
    // 3. Read chunks (with cache)
    chunks, err := fs.readChunks(fileID, startChunk, endChunk)
    if err != nil {
        return nil, err
    }
    
    // 4. Combine and slice
    return fs.combineAndSlice(chunks, offset, size), nil
}
```

### Optimizations:
1. **LRU Cache**: Check cache before DB read
2. **Read-ahead**: Fetch next 1-2 chunks proactively  
3. **mmap**: Already enabled via PRAGMA

---

## Write Path (Batched)

```go
func (fs *SQLiteFS) Write(path string, offset int64, data []byte) error {
    // 1. Resolve or create file
    fileID, err := fs.resolveOrCreateFile(path)
    if err != nil {
        return err
    }
    
    // 2. Split into chunks
    chunks := fs.splitIntoChunks(fileID, offset, data)
    
    // 3. Batch write in transaction
    return fs.db.Transaction(func(tx *sql.Tx) error {
        // Write chunks
        for _, c := range chunks {
            _, err := tx.Exec(`INSERT OR REPLACE INTO fs_file_chunks...`, ...)
        }
        // Update file metadata
        _, err := tx.Exec(`UPDATE fs_nodes SET size = ?, updated_at = ? WHERE id = ?`, ...)
        return err
    })
}
```

### Optimizations:
1. Batch multiple chunks per transaction
2. Optional write buffer (accumulate writes, flush periodically)

---

## Cache Implementation

Simple in-memory LRU following patterns in the codebase:

```go
type ChunkCache struct {
    mu    sync.RWMutex
    data  map[ChunkKey][]byte
    order []ChunkKey
    size  int
    max   int
}

type ChunkKey struct {
    FileID     int64
    ChunkIndex int64
}
```

---

## Testing Strategy

### Unit Tests (fs_test.go)

- `TestRead` - basic read
- `TestReadPartial` - partial/random read
- `TestWrite` - basic write
- `TestWriteOffset` - write at offset
- `TestMkdir` - directory creation
- `TestList` - directory listing
- `TestDelete` - file/dir deletion

### Integration Tests (fs_integration_test.go)

- Use temp database with `go:build integration` tag
- Test WAL mode, concurrent access
- Benchmark random read latency

```go
//go:build integration

package sqlitefs

import (
    "testing"
    "github.com/liyu1981/code_explorer/pkg/db"
    "github.com/liyu1981/code_explorer/pkg/libsql"
)

func TestReadWriteIntegration(t *testing.T) {
    // Setup temp DB
    // Run tests
}
```

---

## Follow-up: Potential Uses in This Project

1. **Knowledge base storage** - Store knowledge entries with versioning
2. **Research session storage** - Store research results in virtual FS
3. **Cache layer** - For codemogger chunk storage (replacing current approach)

---

## Summary

- **Pattern**: Follow existing `pkg/db`, `pkg/config` patterns
- **Storage**: SQLite with WAL + mmap
- **Chunk size**: 16KB
- **Cache**: Optional LRU for hot chunks
- **API**: Standard filesystem operations (Read, Write, Mkdir, List, Delete)