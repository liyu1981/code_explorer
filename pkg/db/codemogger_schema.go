package db

import (
	"database/sql"
	"fmt"
	"io"
	"sync"

	_ "github.com/tursodatabase/go-libsql"
)

type Codebase struct {
	ID        int64
	RootPath  string
	Name      string
	IndexedAt int64
}

type Chunk struct {
	ID             int64
	CodebaseID     int64
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
	ID         int64
	CodebaseID int64
	FilePath   string
	FileHash   string
	ChunkCount int
	IndexedAt  int64
}

type Store struct {
	db     *sql.DB
	dbPath string
	mu     sync.Mutex
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

type CodebaseInfo struct {
	ID         int64
	RootPath   string
	Name       string
	IndexedAt  int64
	FileCount  int
	ChunkCount int
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

func (s *Store) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS codemogger_codebases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			root_path TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL DEFAULT '',
			indexed_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS codemogger_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codebase_id INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			chunk_key TEXT NOT NULL UNIQUE,
			language TEXT NOT NULL,
			kind TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			signature TEXT NOT NULL DEFAULT '',
			snippet TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			file_hash TEXT NOT NULL,
			indexed_at INTEGER NOT NULL,
			embedding BLOB,
			embedding_model TEXT DEFAULT '',
			FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
		)`,
		`CREATE TABLE IF NOT EXISTS codemogger_indexed_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codebase_id INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			file_hash TEXT NOT NULL,
			chunk_count INTEGER NOT NULL DEFAULT 0,
			indexed_at INTEGER NOT NULL,
			UNIQUE(codebase_id, file_path),
			FOREIGN KEY(codebase_id) REFERENCES codemogger_codebases(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_codebase_id ON codemogger_chunks(codebase_id)`,
		`CREATE INDEX IF NOT EXISTS idx_indexed_files_codebase_id ON codemogger_indexed_files(codebase_id)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) reconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Ping(); err != nil {
		s.db.Close()
		db, err := sql.Open("libsql", "file:"+s.dbPath)
		if err != nil {
			return err
		}
		s.db = db
		return db.Ping()
	}
	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}

var _ io.Closer = (*Store)(nil)
