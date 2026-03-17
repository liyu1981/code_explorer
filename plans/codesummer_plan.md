# CodeSummer Feature Plan

## Overview
CodeSummer summarizes each node (source file, normal file, or directory) in a codebase using LLM and stores summaries with embeddings for semantic search.

## Scope

**This plan covers only the backend implementation.** No frontend changes are included.
- API endpoints, if needed, should be added separately
- Frontend integration is out of scope for this feature

## Architecture

### 1. Database Schema (`pkg/db/`)

New tables:
- `codesummer_codebases` - metadata linking to codebases
- `codesummer_summaries` - the main summary table with embeddings

```sql
-- codesummer metadata
CREATE TABLE codesummer_codebases (
    id TEXT PRIMARY KEY,
    codebase_id TEXT NOT NULL,
    indexed_at INTEGER NOT NULL DEFAULT 0
);

-- summaries for files/directories
CREATE TABLE codesummer_summaries (
    id TEXT PRIMARY KEY,
    codesummer_id TEXT NOT NULL,
    node_path TEXT NOT NULL,        -- absolute path to file/dir
    node_type TEXT NOT NULL,        -- 'source_file', 'normal_file', 'directory'
    language TEXT,                  -- e.g., 'go', 'python' (for source files)
    summary TEXT NOT NULL,          -- LLM-generated summary
    definitions TEXT NOT NULL,      -- JSON: constants, functions, classes defined
    dependencies TEXT NOT NULL,     -- JSON: imported/required modules
    data_manipulated TEXT NOT NULL, -- JSON: files/DBs/data structures
    data_flow TEXT NOT NULL,        -- JSON: how data flows in/out
    embedding BLOB,                 -- vector embedding
    embedding_model TEXT,
    indexed_at INTEGER NOT NULL,
    UNIQUE(codesummer_id, node_path)
);

-- track indexed paths to detect changes
CREATE TABLE codesummer_indexed_paths (
    id TEXT PRIMARY KEY,
    codesummer_id TEXT NOT NULL,
    node_path TEXT NOT NULL,
    node_type TEXT NOT NULL,
    file_hash TEXT,                 -- for files only
    indexed_at INTEGER NOT NULL,
    UNIQUE(codesummer_id, node_path)
);
```

### 2. Schema Types (`pkg/db/codesummer_schema.go`)

```go
type CodesummerCodebase struct {
    ID         string
    CodebaseID string
    IndexedAt  int64
}

type CodesummerSummary struct {
    ID              string
    CodesummerID    string
    NodePath        string
    NodeType        string  // "source_file", "normal_file", "directory"
    Language        string
    Summary         string
    Definitions     string  // JSON
    Dependencies    string  // JSON
    DataManipulated string  // JSON
    DataFlow        string  // JSON
    Embedding       []float32
    EmbeddingModel  string
    IndexedAt       int64
}

type IndexedPath struct {
    ID          string
    CodesummerID string
    NodePath    string
    NodeType    string
    FileHash    string
    IndexedAt   int64
}
```

### 3. Store Methods (`pkg/db/codesummer_store.go`)

- `GetOrCreateCodesummerCodebase(ctx, codebaseID)` - creates metadata record
- `CodesummerUpsertSummary(ctx, summary)` - insert/update summary
- `CodesummerUpsertBatchSummaries(ctx, summaries)` - batch insert
- `CodesummerUpsertEmbeddings(ctx, items[])` - update embeddings
- `CodesummerGetSummary(ctx, codesummerID, nodePath)` - get single summary
- `CodesummerListSummaries(ctx, codesummerID)` - list all for a codebase
- `CodesummerVectorSearch(ctx, queryEmbedding, limit)` - semantic search
- `CodesummerUpsertIndexedPath(ctx, path)` - track indexed paths

### 4. Core Package (`pkg/codesummer/`)

#### 4.1 Types (`types.go`)
```go
type SummaryRequest struct {
    CodebaseID string
}

type NodeInfo struct {
    Path         string
    Type         string  // "source_file", "normal_file", "directory"
    Language     string
    Content      string  // for files
    Hash         string
    Children     []string // for directories
    Definitions  []Definition
}

type Definition struct {
    Kind      string // function, class, struct, const, etc.
    Name      string
    Signature string
}

type NodeSummary struct {
    NodeInfo
    Summary         string
    Dependencies    []string
    DataManipulated []string
    DataFlow        DataFlowInfo
}

type DataFlowInfo struct {
    Inputs  []string
    Outputs []string
}
```

#### 4.2 Node Classifier (`classifier.go`)
- Determine if a path is source file, normal file, or directory
- Use `chunk.Languages` to detect if source file and its language
- For directories, collect child paths

#### 4.3 Definition Extractor (`extractor.go`)
- Use tree-sitter to extract top-level definitions
- Reuse `chunk.ChunkFile()` logic but extract definitions differently
- Return list of definitions with kind, name, signature

#### 4.4 LLM Prompts (`prompts.go`)

**System Prompt:**
```
You are a code analyst. For the given file or directory, provide a concise summary covering:
1. What is this file/directory for?
2. What constants, functions, classes, structs does it define?
3. What are its dependencies (imports, requires, includes)?
4. What data does it manipulate (files, databases, data structures)?
5. How does data flow in and out?

// TODO: try to use LLM structured result here, define go struct for the wanted struct and then provide it with JSON SChema to LLM
Respond in JSON format with keys: summary, dependencies, data_manipulated, data_flow
```

**Structured Output**: Use LLM's structured response feature with JSON Schema:

```go
type FileSummary struct {
    Summary         string   `json:"summary"`
    Dependencies    []string `json:"dependencies"`
    DataManipulated []string `json:"data_manipulated"`
    DataFlow        struct {
        Inputs  []string `json:"inputs"`
        Outputs []string `json:"outputs"`
    } `json:"data_flow"`
}
```

Pass this schema to `llm.Generate()` via `responseFormat` parameter.
```

**File Prompt:**
```
Analyze this {language} file:
```{language}
{content}
```

Extracted definitions:
{definitions}

Provide a detailed summary.
```

**Directory Prompt:**
```
Analyze this directory: {path}

Children:
{children_summaries}

Provide a summary of what this directory contains and how its components work together.
```

**Divide-and-Conquer for Large Directories:**

When children summaries exceed context window:
1. Group children into batches that fit in context window
2. Summarize each batch individually (intermediate summaries)
3. If multiple intermediate summaries exist, recursively summarize them
4. Final summary is stored; intermediates kept in memory only

Implementation in `summarizer.go`:
```go
func (s *Summarizer) SummarizeDirectoryBatch(ctx context.Context, dir string, childrenSummaries []NodeSummary) (NodeSummary, error) {
    const maxContextLength = 100000 // characters, tune based on LLM
    if totalLength(childrenSummaries) < maxContextLength {
        return s.summarizeDirectorySingle(ctx, dir, childrenSummaries)
    }
    // Divide into batches
    batches := partitionByLength(childrenSummaries, maxContextLength)
    var intermediate []NodeSummary
    for _, batch := range batches {
        summary, err := s.summarizeDirectorySingle(ctx, dir, batch)
        if err != nil { return nil, err }
        intermediate = append(intermediate, summary)
    }
    // If too many intermediates, recurse
    if len(intermediate) > 1 {
        return s.SummarizeDirectoryBatch(ctx, dir+"_merged", intermediate)
    }
    return intermediate[0], nil
}
```

#### 4.5 Summarizer (`summarizer.go`)
- Interface for LLM-based summarization
- Use `agent.LLM` interface
- Two modes:
  - `SummarizeFile(nodeInfo)` - for source/normal files
  - `SummarizeDirectory(nodeInfo, childrenSummaries)` - for directories
- Parse LLM response into `NodeSummary`

#### 4.6 Embedder Integration (`embedder.go`)
- Use `embed.Embedder` interface from codemogger
- Generate embedding of the summary text
- Store with summary

#### 4.7 Main API (`summer.go`)

```go
func Summary(ctx context.Context, db *db.Store, codebaseID string) error
```

Flow:
1. Get or create codesummer_codebases record
2. Walk codebase using `util.StartFileWalker()`
3. For each file/directory:
   - Classify node type
   - If file: extract definitions, call LLM, save summary with embedding
   - If directory: wait for children, call LLM with children summaries
4. Track indexed paths with hashes to detect changes
5. Process in batches for efficiency

### 5. Key Design Decisions

#### Node Processing Order
- Process all files first, then directories bottom-up
- Directory summary needs children's summaries as context
- Use post-order traversal: children before parent

#### Change Detection
- Track file hashes in `codesummer_indexed_paths`
- Re-process nodes where hash changed
- Delete summaries for deleted paths

#### Batch Processing
- Process files in batches (e.g., 10 at a time)
- Batch LLM calls when possible
- Batch database inserts

#### Embedding Strategy
- Embed the full summary text (not just the description)
- Use same embedder as codemogger for consistency

### 6. Implementation Order

1. **Database**: Add SQL migration and schema types
2. **Store**: Implement database operations
3. **Classifier**: Node type detection
4. **Extractor**: Tree-sitter definition extraction
5. **Prompts**: LLM prompt templates
6. **Summarizer**: LLM integration
7. **Embedder**: Embedding generation
8. **Main**: Wire everything together with Summary() function

### 7. Reused Components

| Component | Source | Usage |
|-----------|--------|-------|
| FileWalker | pkg/util/walker | Walk codebase respecting ignores |
| Tree-sitter | pkg/codemogger/chunk | Parse source files |
| LLM | pkg/agent | Generate summaries |
| Embedder | pkg/codemogger/embed | Generate embeddings |
| DB Store | pkg/db | Persist summaries |

### 8. Constants

Add to `pkg/constant/`:
```go
const (
    // Codesummer batch processing
    CodesummerBatchSize = 10
)
```

- Embedder: Reuse existing codemogger embedder config
- LLM: Reuse system-wide LLM model setting
- Batch size: Use constant `CodesummerBatchSize`
