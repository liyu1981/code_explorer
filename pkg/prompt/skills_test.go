package prompt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func setupTestDB(t *testing.T) (*db.Store, func()) {
	dir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("open db: %v", err)
	}

	store := db.NewStore(database, dbPath)
	return store, func() {
		database.Close()
		os.RemoveAll(dir)
	}
}
