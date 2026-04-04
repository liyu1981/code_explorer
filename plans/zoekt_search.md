# Zoekt Implementation Plan - Updated Structure

## Overview
This plan outlines implementing search functionality for `pkg/zoekt`. Code is organized into:
- `pkg/zoekt/common/` - Shared types and utilities
- `pkg/zoekt/builder/` - Index building code
- `pkg/zoekt/builder.go` - Builder API
- `pkg/zoekt/search/` - Search/query code
- `pkg/zoekt/search.go` - Search API

---

## Directory Structure

```
pkg/zoekt/
├── common/           # Shared code between builder and search
│   ├── document.go  # Document, DocumentSection, SkipReason types
│   ├── bits.go      # Ngram utilities, delta encoding
│   ├── section.go   # simpleSection, compoundSection, writer
│   ├── zoekt.go     # Repository, RepositoryBranch, IndexMetadata, Symbol
│   └── options.go   # Common options
│
├── builder/         # Index building code
│   ├── shard_builder.go   # postingsBuilder, ShardBuilder
│   ├── builder.go         # Builder with parallel shard building
│   └── write.go           # ShardBuilder.Write() serialization
│
├── builder.go       # Builder API (wraps builder/)
│
├── search/          # Search/query code
│   ├── btree.go           # B+-tree for trigram lookup
│   ├── read.go            # IndexFile interface, reader
│   ├── indexdata.go       # indexData struct, loadIndex
│   ├── query.go           # Query types (Substring, Regexp, And, Or, Not)
│   ├── parse.go           # Query parsing
│   ├── searcher.go        # Searcher interface
│   ├── match.go          # Match tree iteration
│   ├── eval.go           # Query evaluation
│   └── shards.go         # Multi-shard search
│
└── search.go       # Search API (wraps search/)
```

---

## Component Details

### common/ - Shared Code
Files moved from root pkg/zoekt:
- `document.go` - Document, DocumentSection, SkipReason, IndexFS
- `bits.go` - ngram utilities, marshal/unmarshal functions
- `section.go` - simpleSection, compoundSection, writer
- `zoekt.go` - Repository, RepositoryBranch, IndexMetadata, Symbol types
- `options.go` - Common Options (retain IndexFS field)

### builder/ - Index Building
```go
// API in builder.go
func NewBuilder(opts Options) (*Builder, error)
func (b *Builder) Add(doc Document) error
func (b *Builder) AddFile(name string, content []byte) error
func (b *Builder) Finish() error
```

Files:
- `shard_builder.go` (moved) - postingsBuilder, ShardBuilder, DocChecker
- `builder.go` (moved) - Builder with parallel shard building
- `write.go` (moved) - ShardBuilder.Write() serialization

### search/ - Search/Query
```go
// API in search.go
type Searcher interface {
    Search(ctx context.Context, q Query, opts *SearchOptions) (*SearchResult, error)
    Close() error
}

func OpenShard(path string) (ShardSearcher, error)
func ParseQuery(s string) (Query, error)
func Search(ctx context.Context, searchers []Searcher, q Query, opts *SearchOptions) (*SearchResult, error)
```

Files:
- `btree.go` (new copy from ref/zoekt/index/btree.go)
- `read.go` (new) - IndexFile interface, reader, sqliteIndexFile
- `indexdata.go` (new) - indexData struct, loadIndex(), btree integration
- `query.go` (new) - Query interface, Substring, Regexp, And, Or, Not, Branch
- `parse.go` (new) - Parse() function for query strings
- `searcher.go` (new) - Searcher, ShardSearcher implementations
- `match.go` (new) - Match tree, trigram iteration (simplified from ref)
- `eval.go` (new) - Evaluate Query against ShardSearcher
- `shards.go` (new) - AggregateSearcher, parallel search

---

## Implementation Steps

### Step 1: Create common/
1. Move existing files to common/:
   - `document.go`, `bits.go`, `section.go`, `zoekt.go`, `options.go`

### Step 2: Create builder/
1. Move existing builder files:
   - `shard_builder.go` → `builder/shard_builder.go`
   - `builder.go` → `builder/builder.go`  
   - `write.go` → `builder/write.go`
2. Update imports in moved files

### Step 3: Create search/
1. Copy from ref code:
   - `btree.go` - adapt for standalone
   - Core logic from `indexdata.go`, `matchtree.go`, `eval.go`
2. Create new:
   - `read.go` - IndexFile with sqlitefs integration
   - `query.go`, `parse.go` - Query types and parsing
   - `searcher.go`, `shards.go` - Search implementations

### Step 4: Create API wrappers
1. Create `builder.go` - re-export builder API
2. Create `search.go` - re-export search API

### Step 5: Tests
- Update test file locations to match new structure
- Add search integration tests

---

## Reuse from Reference

### Direct Copy (with minimal changes)
- `search/btree.go` - Full copy from ref/zoekt/index/btree.go
- `search/match.go` - Core logic from ref/zoekt/index/matchtree.go

### Adapt (simplify for single-repo)
- `search/indexdata.go` - Reference indexdata.go, remove multi-repo
- `search/query.go` - Reference query/query.go, simplify
- `search/eval.go` - Reference eval.go, simplify

### Reference Only
- Complex optimizations (bloom filters, roaring bitmaps)
- Aggregated stats across repos
- gRPC/APIs for remote search

---

## Search API Example Usage

```go
// Open shard from sqlitefs
searcher, _ := zoekt.OpenShard("/repo_00000001_v16.00000.zoekt")
defer searcher.Close()

// Parse query
q, _ := zoekt.ParseQuery("func main")

// Search
result, _ := searcher.Search(ctx, q, nil)

for _, file := range result.Files {
    fmt.Println(file.FileName)
    for _, m := range file.Matches {
        fmt.Printf("  %s\n", m.Line)
    }
}
```