# Zoekt Index Implementation Plan

## Overview

This document outlines the implementation plan for `pkg/zoekt`, a source code indexing package based on the zoekt indexing system. The goal is to create a functional indexer that builds searchable indexes stored in `pkg/sqlitefs`.

## Zoekt Indexing Architecture

### High-Level Flow

1. **Entry Point** (`cmd/zoekt-index/main.go`):
   - Walks a directory tree, filtering out ignored directories (.git, .hg, .svn)
   - Reads files and passes them to the Builder
   - Optional: reads `.meta` file for repository metadata

2. **Builder** (`index/builder.go`):
   - Manages parallel shard building
   - Buffers documents until `ShardMax` (default 100MB) is reached
   - Creates `ShardBuilder` instances to build individual shards
   - Handles incremental/delta indexing
   - Sorts documents by rank before writing

3. **ShardBuilder** (`index/shard_builder.go`):
   - Processes individual documents
   - Extracts trigrams from content and names
   - Creates postings (inverted index entries)
   - Maintains language detection

4. **Write** (`index/write.go`):
   - Serializes the shard to disk (into sqlitefs)
   - Writes various index sections (content, names, postings, metadata, etc.)

### Key Data Structures

#### Document (`index/document.go`)
```go
type Document struct {
    Name              string  // relative path from repo root
    Content           []byte  // not stored in index - only metadata
    Branches          []string
    SubRepositoryPath string
    Language          string
    Category          FileCategory
    SkipReason        SkipReason
}
```

#### Options (`index/builder.go`)
- `IndexDir`: output directory for index files (mapped to sqlitefs)
- `SizeMax`: max file size to index (default 2MB)
- `ShardMax`: max corpus size per shard (default 100MB)
- `Parallelism`: concurrent shard builds
- `TrigramMax`: max distinct trigrams per doc
- `LargeFiles`: glob patterns for large file override

### Index File Format (`index/toc.go`)

The index consists of sections:
- `fileNames`: indexed filenames (relative paths)
- `postings`: inverted index for content (trigram -> document positions)
- `namePostings`: inverted index for filenames
- `ngramText`/`nameNgramText`: trigram values
- `branchMasks`: which branches contain each file
- `metadata`: repository metadata
- `languages`: language detection results
- `categories`: file categorization

Note: Raw file content is NOT stored. Only metadata (filenames, symbols, etc.) is indexed for search.

### Trigram Indexing

- Uses 3-character trigrams (n-grams)
- ASCII trigrams use direct array indexing (2M entries, zero hash cost)
- Non-ASCII trigrams use map
- Postings stored as delta-encoded varints

## Implementation Plan for pkg/zoekt

### Phase 1: Core Index Builder

1. **Package Structure**:
   ```
   pkg/zoekt/
   ├── builder.go      # Builder implementation
   ├── shard.go        # ShardBuilder implementation
   ├── document.go     # Document types
   ├── options.go       # Configuration options
   ├── write.go        # Index serialization
   └── index.go        # Public API
   ```

2. **Core Types**:
   - Implement `Document`, `DocumentSection`, `SkipReason`
   - Implement `Options` with configurable defaults
   - Implement `Builder` struct with parallel shard building
   - Implement `ShardBuilder` for individual shards

3. **Trigram Processing**:
   - Implement postingsBuilder with ASCII direct-index
   - Implement non-ASCII trigram handling with map
   - Implement delta-encoded posting lists

### Phase 2: Index Writing to sqlitefs

1. **Storage Integration**:
   - Modify output to write to sqlitefs instead of plain files
   - Store index shards as blobs in sqlitefs
   - Use repository ID as key for lookup
   - Only store filename (relative path), NOT raw content

2. **Serialization**:
   - Implement compoundSection and simpleSection types
   - Implement indexTOC with all required sections (no fileContents section)
   - Write JSON metadata
   - Write binary sections with proper formatting

3. **File Output**:
   - Store shard files with `.zoekt` extension in sqlitefs
   - Support compound shards for multiple repos
   - Handle atomic writes with temp blobs

### Phase 3: Language Detection

1. **File-based Detection**:
   - Use file extension matching
   - Map extensions to language codes

## Key Design Decisions

1. **Sharding Strategy**:
   - Default 100MB shard size
   - Parallel build with configurable concurrency
   - Document ranking before writing

2. **Memory Management**:
   - Reuse posting builder pools
   - Clear skipped document content
   - Check memory usage during build

3. **Compatibility**:
   - Start with index format version 17 (latest)
   - Support incremental indexing
   - Track index options hash

4. **Output**:
   - Index files stored in `pkg/sqlitefs` blob storage
   - Repository metadata indexed by repo ID

## API Design

```go
// pkg/zoekt/builder.go
type Options struct {
    IndexDir            string  // maps to sqlitefs mount point
    SizeMax             int
    ShardMax            int
    Parallelism         int
    TrigramMax          int
    LargeFiles          []string
    RepositoryDescription Repository
}

type Builder struct {
    // internal fields
}

func NewBuilder(opts Options) (*Builder, error)
func (b *Builder) Add(doc Document) error
func (b *Builder) AddFile(name string, content []byte) error
func (b *Builder) Finish() error

// pkg/zoekt/document.go
type Document struct {
    Name            string
    Content         []byte
    Branches        []string
    Language        string
}

type DocumentSection struct {
    Start, End uint32
}

// pkg/zoekt/index.go
type Repository struct {
    ID   uint32
    Name string
}
```

## Testing Strategy

1. Build indexes from test repositories
2. Verify index file structure in sqlitefs
3. Benchmark indexing performance

## Dependencies

- Roaring bitmap for repo ID bitmaps
- go-enry for language detection (via file extensions)