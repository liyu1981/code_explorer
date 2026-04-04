package zoekt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
)

func TestZoektIndex(t *testing.T) {
	// Setup temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "zoekt-index-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup some test files
	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	testFiles := map[string]string{
		"main.go": `package main
import "fmt"
func main() {
	fmt.Println("Hello, World!")
}`,
		"util.go": `package main
func Add(a, b int) int {
	return a + b
}`,
	}

	for name, content := range testFiles {
		err = os.WriteFile(filepath.Join(srcDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	// Create db store
	store, closeStore := db.SetupTestDB(t)
	defer closeStore()

	fs := sqlitefs.OpenFS(store)

	// Create index
	idx := NewZoektIndex(store, fs)

	ctx := context.Background()

	// 1. Initial Indexing
	opts := &IndexOptions{}
	res, err := idx.Index(ctx, srcDir, opts)
	if err != nil {
		t.Fatalf("initial indexing failed: %v", err)
	}

	if res.Files != 2 {
		t.Errorf("expected 2 files indexed, got %d", res.Files)
	}
	if res.Skipped != 0 {
		t.Errorf("expected 0 files skipped on initial index, got %d", res.Skipped)
	}

	// 2. Re-indexing without changes
	res, err = idx.Index(ctx, srcDir, opts)
	if err != nil {
		t.Fatalf("re-indexing failed: %v", err)
	}

	if res.Skipped != 2 {
		t.Errorf("expected 2 files skipped on re-index without changes, got %d", res.Skipped)
	}

	// 3. Re-indexing with a change
	err = os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package main\nfunc Add(a, b int) int { return a + b + 1 }"), 0644)
	if err != nil {
		t.Fatalf("failed to update test file: %v", err)
	}

	res, err = idx.Index(ctx, srcDir, opts)
	if err != nil {
		t.Fatalf("re-indexing with change failed: %v", err)
	}

	if res.Skipped != 1 {
		t.Errorf("expected 1 file skipped, got %d", res.Skipped)
	}
	if res.Files != 2 {
		t.Errorf("expected 2 total files, got %d", res.Files)
	}

	// 4. Re-indexing with a removal
	err = os.Remove(filepath.Join(srcDir, "util.go"))
	if err != nil {
		t.Fatalf("failed to remove test file: %v", err)
	}

	res, err = idx.Index(ctx, srcDir, opts)
	if err != nil {
		t.Fatalf("re-indexing with removal failed: %v", err)
	}

	if res.Files != 1 {
		t.Errorf("expected 1 file, got %d", res.Files)
	}
	if res.Removed != 1 {
		t.Errorf("expected 1 file removed from tracking, got %d", res.Removed)
	}
}
