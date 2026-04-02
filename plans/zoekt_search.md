# Zoekt Search Implementation Plan

## Overview
This plan outlines implementing search functionality for `pkg/zoekt` to query the indexed shards. The implementation will reuse code from `ref/zoekt/index/` and `ref/zoekt/query/` as much as possible.

## Architecture

### Components to Implement
1. **IndexFile interface** - Abstraction for reading index data (backed by sqlitefs)
2. **ShardReader** - Loads and manages a single shard for searching
3. **IndexData** - In-memory index data structure with btree for trigram lookup
4. **Searcher** - Main interface for executing searches
5. **Query parser** - Parse search queries (regex, substring, AND/OR/NOT)

### Key Files from Reference
- `ref/zoekt/index/read.go` - IndexFile, reader, loading logic
- `ref/zoekt/index/indexdata.go` - indexData struct with btree
- `ref/zoekt/index/btree.go` - B+-tree for trigram lookup
- `ref/zoekt/index/matchtree.go` - Match tree iteration
- `ref/zoekt/index/eval.go` - Query evaluation
- `ref/zoekt/query/query.go` - Query types
- `ref/zoekt/query/parse.go` - Query parsing

## Implementation Steps

### Phase 1: Index File Reading
1. **Create `pkg/zoekt/read.go`**
   - Define `IndexFile` interface with Read(), Size(), Close()
   - Implement `sqliteIndexFile` wrapping pkg/sqlitefs
   - Implement `reader` struct for reading sections

2. **Create `pkg/zoekt/indexdata.go`**
   - Define `indexData` struct holding in-memory index
   - Implement `loadIndex()` to parse TOC and load sections
   - Add btree index for content and filename trigrams
   - Implement file boundary, branch mask, checksum lookups

### Phase 2: B+tree Index
3. **Copy/modify `pkg/zoekt/btree.go`**
   - Reuse from `ref/zoekt/index/btree.go`
   - Adapt for standalone use (remove zoekt dependencies)
   - Implement Get(), getPostingList(), getBucket()

### Phase 3: Query Types
4. **Create `pkg/zoekt/query.go`**
   - Define Query interface with Search()
   - Implement: Substring, Regexp, And, Or, Not, Branch
   - Copy from `ref/zoekt/query/query.go`

5. **Create `pkg/zoekt/parse.go`**
   - Implement query parsing from string
   - Parse syntax: `"foo bar"`, `regex:.*\.go$`, `file:*.go`, `branch:main`

### Phase 4: Search Execution
6. **Create `pkg/zoekt/searcher.go`**
   - Define `Searcher` interface
   - Implement `ShardSearcher` for single shard
   - Implement Search() method returning Results

7. **Create `pkg/zoekt/match.go`**
   - Implement match tree iteration (from ref/zoekt/index/matchtree.go)
   - Trigram matching logic
   - Candidate evaluation

8. **Create `pkg/zoekt/eval.go`**
   - Query evaluation logic
   - Boolean combinations (And, Or, Not)
   - Score calculation

### Phase 5: Multi-Shard Search
9. **Create `pkg/zoekt/shards.go`**
   - AggregateSearcher for multiple shards
   - Result merging
   - Parallel shard search

### Phase 6: Integration with sqlitefs
10. **Update searcher to use pkg/sqlitefs**
    - Read index files from sqlitefs paths
    - Cache loaded shards
    - Handle incremental updates

## File Structure
```
pkg/zoekt/
├── read.go           # IndexFile interface, reader
├── indexdata.go      # indexData struct, loadIndex
├── btree.go          # B+-tree for trigram lookup
├── query.go          # Query types
├── parse.go          # Query parsing
├── searcher.go       # Searcher interface
├── match.go          # Match tree iteration
├── eval.go           # Query evaluation
├── shards.go         # Multi-shard search
└── search.go         # Main search API
```

## Reuse Strategy
1. **Direct copy** - btree.go, matchtree.go core logic
2. **Adapt** - indexData, query types (simplify for single-repo)
3. **Reference only** - complex optimizations, aggregations

## Key Interfaces

```go
// IndexFile reads index data from storage
type IndexFile interface {
    Read(off uint32, sz uint32) ([]byte, error)
    Size() (uint32, error)
    Close()
}

// Searcher searches indexed shards
type Searcher interface {
    Search(ctx context.Context, q Query, opts *SearchOptions) (*SearchResult, error)
    Close() error
}

// Query represents a search query
type Query interface {
    Search(ctx context.Context, s *ShardSearcher) (Matches, error)
}
```

## Search Result Structure
```go
type SearchResult struct {
    Files   []FileMatch
    Stats   Stats
}

type FileMatch struct {
    FileName   string
    Content    string
    Matches    []Match
    Score      float64
    Branch     string
    Repository string
}
```

## Testing Strategy
- Unit tests for btree, query parsing
- Integration test with sqlitefs (write then search)
- Performance benchmarks for large shards