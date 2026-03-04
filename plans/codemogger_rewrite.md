# Plan: Rewrite codemogger from TypeScript to Go

## Overview

Rewrite `vendor/codemogger` (TypeScript) to Go in `pkg/codemogger`. codemogger is a code indexing library for AI coding agents that provides semantic and keyword search over source codebases using local embeddings and full-text search, stored in libSQL (Turso).

## Key Changes from Original Plan

1. **No CLI tool** - codemogger is a Go package in this project, not a standalone CLI
2. **Use main go.mod** - codemogger uses the project's existing go.mod
3. **Use Turso/libSQL** - SQLite-compatible with built-in FTS5 and vector search
4. **No MCP server** - codemogger is just a Go pkg
5. **Local embeddings** - Use `github.com/clems4ever/all-minilm-l6-v2-go` (Go port of huggingface all-MiniLM-L6-v2)

## Project Scope

### Core Features (Must Have)
1. **Code Indexing**: Parse source files using tree-sitter, extract code definitions (functions, structs, classes, etc.)
2. **Incremental Indexing**: Only re-index changed files based on SHA-256 hash
3. **Semantic Search**: Vector similarity search using embeddings
4. **Keyword Search**: FTS5 full-text search
5. **Hybrid Search**: Combine semantic and keyword using Reciprocal Rank Fusion
6. **Go Package**: Simple importable library
7. **libSQL Storage**: Single .db file per codebase

### Secondary Features (Nice to Have)
1. (none)

## Architecture

```
pkg/codemogger/
├── index.go                 # Main CodeIndex API
├── config.go                # Configuration
│
├── db/
│   ├── db.go               # Database connection & schema
│   ├── store.go           # CRUD operations
│   └── schema.go          # SQL definitions
│
├── chunk/
│   ├── chunker.go         # Code chunking logic
│   ├── treesitter.go      # AST parsing
│   ├── languages.go       # Language configs
│   └── types.go           # CodeChunk interface
│
├── embed/
│   ├── embedder.go        # Embedder interface
│   ├── local.go           # Local embedding (all-MiniLM-L6-v2)
│   └── openai.go          # OpenAI-compatible API embeddings
│
├── scan/
│   └── walker.go          # Directory scanning with gitignore
│
├── search/
│   ├── searcher.go        # Search logic
│   ├── query.go           # Query preprocessing
│   └── rank.go            # RRF ranking
│
└── format/
    ├── json.go            # JSON output
    └── text.go            # Text output
```

## Technology Choices

### tree-sitter
- Use `github.com/smacker/go-tree-sitter` for Go bindings
- Bundled grammars as Go packages: `github.com/smacker/go-tree-sitter-rust`, `go`, `python`, etc.
- Parse files to extract top-level definitions

### Turso/libSQL
- Use `turso.tech/database/tursogo` or `github.com/tursodatabase/go-libsql`
- Built-in FTS5 for keyword search
- Built-in vector search for semantic search (no need to compute in Go)
- Single .db file per codebase, SQLite-compatible

### Embeddings
- **Primary**: `github.com/clems4ever/all-minilm-l6-v2-go` - Native Go implementation of all-MiniLM-L6-v2
  - 384-dimensional embeddings
  - Runs locally on CPU
  - ONNX Runtime for inference
  - Model weights embedded in binary (~90MB)
- **Alternative**: OpenAI-compatible API

## Implementation Steps

### Phase 1: Foundation
1. Add dependencies to go.mod
2. Create database schema (chunks table, FTS5 table, vector column)
3. Implement basic DB connection and store
4. Implement file walker with gitignore support

### Phase 2: Code Chunking
1. Add tree-sitter dependencies for common languages
2. Implement language detection
3. Implement AST parsing for Rust, Go, Python, TypeScript, JavaScript
4. Extract code definitions (functions, structs, classes, etc.)
5. Create CodeChunk struct with metadata

### Phase 3: Indexing
1. Implement full indexing pipeline: scan → chunk → embed → store
2. Add SHA-256 hashing for incremental updates
3. Handle large files (split >150 lines)

### Phase 4: Search
1. Implement keyword search (FTS5)
2. Implement semantic search (libSQL vector search)
3. Implement hybrid search (RRF)
4. Add query preprocessing (stopwords)

### Phase 5: API Polish
1. Clean up public API
2. Add tests
3. Add examples

## Language Support (Priority Order)

1. **Tier 1** (Must have): Go, Rust, Python, TypeScript, JavaScript
2. **Tier 2** (Nice to have): Java, C, C++, Ruby, PHP, C#, Scala, Zig

## Dependencies (Go)

```go
require (
    turso.tech/database/tursogo
    
    github.com/smacker/go-tree-sitter v0.22.x
    github.com/smacker/go-tree-sitter-go    // grammar
    github.com/smacker/go-tree-sitter-rust  // grammar
    github.com/smacker/go-tree-sitter-python
    github.com/smacker/go-tree-sitter-typescript
    github.com/smacker/go-tree-sitter-javascript
    github.com/smacker/go-tree-sitter-c
    github.com/smacker/go-tree-sitter-cpp
    
    github.com/clems4ever/all-minilm-l6-v2-go  // Local embeddings
    
    github.com/go-playground/validator/v10  // config validation
    github.com/spf13/cobra                  // CLI (if needed)
    github.com/spf13/viper                  // config
    
    github.com/google/go-github/v68         // gitignore parsing
)
```

## Usage Example

```go
package main

import (
    "log"
    "github.com/liyu1981/code_explorer/pkg/codemogger"
)

func main() {
    db, err := codemogger.NewCodeIndex("./myproject.db", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Index a codebase
    err = db.Index("/path/to/project", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Search
    results, err := db.Search("authentication middleware", &codemogger.SearchOptions{
        Mode: codemogger.SearchModeHybrid,
        Limit: 10,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    for _, r := range results {
        println(r.Name, r.Path, r.Snippet)
    }
}
```

## Configuration

Config file (`configs/codemogger.json`):
```json
{
  "db_path": "./codemogger.db",
  "embedder": {
    "type": "local",  // or "openai"
    "model": "all-MiniLM-L6-v2"
  },
  "openai": {
    "api_base": "http://localhost:8080/v1",
    "model": "text-embedding-3-small",
    "api_key": ""
  },
  "languages": ["go", "rust", "python", "typescript", "javascript"],
  "chunk_lines": 150
}
```