package codemogger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func TestCodeIndex(t *testing.T) {
	// Setup temporary directory for test DB and files
	tmpDir, err := os.MkdirTemp("", "codemogger-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

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

	// Initialize config
	cfg := config.DefaultConfig()
	config.Set(cfg)

	// Create db store with migrations
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	store := db.NewStore(sqlDB, dbPath)

	// Create index
	idx, err := NewCodeIndex(cfg, store)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	// Use MockEmbedder to avoid network calls
	idx.SetEmbedder(&embed.MockEmbedder{DimVal: 384})

	ctx := context.Background()

	// Test Indexing
	opts := &IndexOptions{
		Languages: []string{"go"},
	}
	res, err := idx.Index(ctx, srcDir, opts)
	if err != nil {
		t.Fatalf("indexing failed: %v", err)
	}

	if res.Files != 2 {
		t.Errorf("expected 2 files indexed, got %d", res.Files)
	}

	// Test ListFiles
	files, err := idx.ListFiles(ctx)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 indexed files, got %d", len(files))
	}

	// Test Search (semantic mock)
	searchOpts := &SearchOptions{
		Limit: 5,
		Mode:  SearchModeSemantic,
	}
	results, err := idx.Search(ctx, "hello", searchOpts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("expected some results, got 0")
	}

	// Test Search (keyword)
	searchOpts.Mode = SearchModeKeyword
	results, err = idx.Search(ctx, "main", searchOpts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("expected some results for keyword search, got 0")
	}
}
