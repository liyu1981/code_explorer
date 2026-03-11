package codemogger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
)

func TestCodeIndex(t *testing.T) {
	dir, err := os.MkdirTemp("", "codemogger-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0755)

	// Create some test files
	goFile := filepath.Join(projectDir, "main.go")
	goContent := `package main
import "fmt"
func main() {
	fmt.Println("Hello, World!")
}
func Add(a, b int) int {
	return a + b
}
`
	os.WriteFile(goFile, []byte(goContent), 0644)

	pyFile := filepath.Join(projectDir, "utils.py")
	pyContent := `def subtract(a, b):
    return a - b

class Calculator:
    def multiply(self, a, b):
        return a * b
`
	os.WriteFile(pyFile, []byte(pyContent), 0644)

	idx, err := NewCodeIndex(dbPath)
	if err != nil {
		t.Fatalf("new code index: %v", err)
	}
	defer idx.Close()

	// Inject mock embedder
	idx.SetEmbedder(&embed.MockEmbedder{DimVal: 384})

	// Test Indexing
	opts := &IndexOptions{
		Languages: []string{"go", "python"},
		Verbose:   true,
	}
	res, err := idx.Index(projectDir, opts)
	if err != nil {
		t.Fatalf("index: %v", err)
	}

	if res.Files != 2 {
		t.Errorf("expected 2 files processed, got %d", res.Files)
	}
	if res.Chunks < 3 {
		t.Errorf("expected at least 3 chunks, got %d", res.Chunks)
	}

	// Test Listing
	files, err := idx.ListFiles()
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files in list, got %d", len(files))
	}

	// Test Search (Semantic)
	searchRes, err := idx.Search("main", &SearchOptions{Limit: 5, Mode: SearchModeSemantic})
	if err != nil {
		t.Fatalf("semantic search: %v", err)
	}
	if len(searchRes) == 0 {
		t.Error("expected search results, got none")
	}

	// Test Keyword Search
	searchRes, err = idx.Search("main", &SearchOptions{Limit: 5, Mode: SearchModeKeyword})
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
}
