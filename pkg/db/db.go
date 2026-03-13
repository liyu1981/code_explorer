package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"

	"github.com/liyu1981/code_explorer/pkg/libsql"
)

var (
	instance *Store
	once     sync.Once
)

type Store struct {
	db      *sql.DB
	dbPath  string
	mu      sync.Mutex
	writeMu sync.Mutex
}

func NewStore(db *sql.DB, dbPath string) *Store {
	once.Do(func() {
		instance = &Store{}
	})
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.db = db
	instance.dbPath = dbPath
	return instance
}

func GetStore() *Store {
	return instance
}

func ResetStoreForTest() {
	instance = nil
	once = sync.Once{}
}

// ExecWrite executes a write query under a global write lock.
func (s *Store) ExecWrite(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.db.ExecContext(ctx, query, args...)
}

// Transaction executes a function within a database transaction and a global write lock.
func (s *Store) Transaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) reconnect() error {
	if s.db != nil {
		if err := s.db.Ping(); err == nil {
			return nil
		}
		_ = s.db.Close()
	}

	db, err := libsql.OpenLibsqlDb(s.dbPath)
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
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

var _ io.Closer = (*Store)(nil)

func Open(dbPath string) (*sql.DB, error) {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(db, dbPath); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}
