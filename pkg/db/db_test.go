package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndStore(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Test Open (this runs migrations)
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Verify migrations ran by checking for a table
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&name)
	if err != nil {
		t.Fatalf("failed to verify migrations: %v", err)
	}
	if name != "tasks" {
		t.Errorf("expected table tasks, got %s", name)
	}

	// Test NewStore
	store := NewStore(db, dbPath)
	if store == nil {
		t.Fatal("NewStore returned nil")
	}

	// Test singleton
	store2 := NewStore(db, dbPath)
	if store != store2 {
		t.Error("NewStore did not return the same singleton instance")
	}

	// Test DB()
	if store.DB() != db {
		t.Error("store.DB() did not return the expected *sql.DB")
	}

	ctx := context.Background()

	// Test basic functionality using the store
	cb, err := store.GetOrCreateCodebase(ctx, "/test/path", "test", "local")
	if err != nil {
		t.Fatalf("failed to create codebase: %v", err)
	}

	// Test codemogger metadata
	metadataID, err := store.CodemoggerEnsureMetadata(ctx, cb.ID)
	if err != nil {
		t.Fatalf("failed to ensure metadata: %v", err)
	}
	if metadataID == "" {
		t.Error("expected non-empty metadata ID")
	}

	// Test listing
	codebases, err := store.CodemoggerListCodebases(ctx)
	if err != nil {
		t.Fatalf("failed to list codebases: %v", err)
	}
	if len(codebases) == 0 {
		t.Error("expected at least one codebase in list")
	}
}
