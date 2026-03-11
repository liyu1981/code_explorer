package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndStore(t *testing.T) {
	dir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Check if tables exist
	var tables []string
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}

	expected := []string{"codebases", "codemogger_chunks", "codemogger_codebases", "codemogger_indexed_files", "schema_migrations"}
	for _, e := range expected {
		found := false
		for _, t := range tables {
			if t == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q not found, got %v", e, tables)
		}
	}

	// Test NewStore
	store := NewStore(db, dbPath)
	if store == nil {
		t.Fatal("expected store to be non-nil")
	}

	// Test a store method
	cb, err := store.GetOrCreateCodebase("/test/path", "test", "local")
	if err != nil {
		t.Fatalf("get or create system codebase: %v", err)
	}
	if cb.ID == "" {
		t.Errorf("expected valid id, got empty string")
	}

	id, err := store.CodemoggerEnsureMetadata(cb.ID)
	if err != nil {
		t.Fatalf("ensure codemogger metadata: %v", err)
	}
	if id == "" {
		t.Errorf("expected valid metadata id, got empty string")
	}

	codebases, err := store.CodemoggerListCodebases()
	if err != nil {
		t.Fatalf("list codebases: %v", err)
	}
	if len(codebases) != 1 {
		t.Errorf("expected 1 codebase, got %d", len(codebases))
	}
}
