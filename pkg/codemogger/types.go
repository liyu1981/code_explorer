package codemogger

type SearchMode string

const (
	SearchModeSemantic SearchMode = "semantic"
	SearchModeKeyword  SearchMode = "keyword"
	SearchModeHybrid   SearchMode = "hybrid"
)

type SearchOptions struct {
	Limit          int        `json:"limit,omitempty"`
	Threshold      float64    `json:"threshold,omitempty"`
	IncludeSnippet bool       `json:"includeSnippet,omitempty"`
	Mode           SearchMode `json:"mode,omitempty"`
}

type IndexOptions struct {
	Languages []string                               `json:"languages,omitempty"`
	Verbose   bool                                   `json:"verbose,omitempty"`
	Progress  func(current, total int, stage string) `json:"-"`
}

type IndexResult struct {
	Files    int      `json:"files"`
	Chunks   int      `json:"chunks"`
	Embedded int      `json:"embedded"`
	Skipped  int      `json:"skipped"`
	Removed  int      `json:"removed"`
	Errors   []string `json:"errors"`
	Duration int      `json:"duration"`
}

type SearchResult struct {
	ChunkKey  string  `json:"chunkKey"`
	FilePath  string  `json:"filePath"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Signature string  `json:"signature"`
	Snippet   string  `json:"snippet"`
	StartLine int     `json:"startLine"`
	EndLine   int     `json:"endLine"`
	Score     float64 `json:"score"`
}

type IndexedFile struct {
	FilePath   string `json:"filePath"`
	FileHash   string `json:"fileHash"`
	ChunkCount int    `json:"chunkCount"`
	IndexedAt  int64  `json:"indexedAt"`
}

type Codebase struct {
	ID         string `json:"id"`
	RootPath   string `json:"rootPath"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Version    string `json:"version"`
	IndexedAt  int64  `json:"indexedAt"`
	FileCount  int    `json:"fileCount"`
	ChunkCount int    `json:"chunkCount"`
}
