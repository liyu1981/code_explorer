package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) (*Store, func()) {
	dir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("open db: %v", err)
	}

	store := NewStore(db, dbPath)
	return store, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}
