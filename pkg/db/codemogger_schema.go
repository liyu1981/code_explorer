package db

import (
	_ "github.com/tursodatabase/go-libsql"
)

// Codebase represents the system-wide definition of a codebase.
type Codebase struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RootPath  string `json:"rootPath"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	CreatedAt int64  `json:"createdAt"`
}

// CodemoggerMetadata represents the indexing-specific metadata for a codebase.
type CodemoggerMetadata struct {
	ID         string `json:"id"`
	CodebaseID string `json:"codebaseId"`
	IndexedAt  int64  `json:"indexedAt"`
}

type Chunk struct {
	ID             string
	CodebaseID     string
	FilePath       string
	ChunkKey       string
	Language       string
	Kind           string
	Name           string
	Signature      string
	Snippet        string
	StartLine      int
	EndLine        int
	FileHash       string
	IndexedAt      int
	Embedding      []byte
	EmbeddingModel string
}

type IndexedFile struct {
	ID         string
	CodebaseID string
	FilePath   string
	FileHash   string
	ChunkCount int
	IndexedAt  int64
}

type SearchResult struct {
	ChunkKey  string
	FilePath  string
	Name      string
	Kind      string
	Signature string
	Snippet   string
	StartLine int
	EndLine   int
	Score     float64
}

type FileInfo struct {
	FilePath   string
	FileHash   string
	ChunkCount int
	IndexedAt  int64
}

// CodebaseInfo remains for backward compatibility or joined views,
// but we prefer using Codebase and CodemoggerMetadata separately.
type CodebaseInfo struct {
	ID         string `json:"id"`
	RootPath   string `json:"rootPath"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Version    string `json:"version"`
	IndexedAt  int64  `json:"indexedAt"`
	FileCount  int    `json:"fileCount"`
	ChunkCount int    `json:"chunkCount"`
}

type CodeChunk struct {
	ChunkKey  string
	FilePath  string
	Language  string
	Kind      string
	Name      string
	Signature string
	Snippet   string
	StartLine int
	EndLine   int
	FileHash  string
}
