package db

import (
	"database/sql"
	"io"
	"sync"

	_ "github.com/tursodatabase/go-libsql"
)

type Codebase struct {
	ID        string
	RootPath  string
	Name      string
	IndexedAt int64
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

type Store struct {
	db     *sql.DB
	dbPath string
	mu     sync.Mutex
}

func NewStore(db *sql.DB, dbPath string) *Store {
	return &Store{
		db:     db,
		dbPath: dbPath,
	}
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
	ID         string
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

func (s *Store) reconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		if err := s.db.Ping(); err == nil {
			return nil
		}
		s.db.Close()
	}

	dsn := "file:" + s.dbPath + "?_busy_timeout=5000"
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return err
	}
	s.db = db
	return db.Ping()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}

var _ io.Closer = (*Store)(nil)
