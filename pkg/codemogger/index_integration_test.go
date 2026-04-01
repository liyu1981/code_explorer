//go:build integration

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

func TestCodeIndexIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codemogger-integration-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

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

	cfg := config.DefaultConfig()
	cfg.CodeMogger.InheritSystemLLM = false
	cfg.CodeMogger.Embedder = config.EmbedderConfig{
		Type:  "local",
		Model: "unsloth/Qwen3.5-9B-GGUF:Q4_K_M",
		OpenAI: config.OpenAIConfig{
			APIBase: "http://localhost:20003/v1",
			APIKey:  "",
		},
	}
	config.Set(cfg)

	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	store := db.NewStore(sqlDB, dbPath)

	idx, err := NewCodeIndex(cfg, store)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	t.Logf("Using embedder model: %s", idx.embedder.Model())

	ctx := context.Background()

	t.Run("Index Files", func(t *testing.T) {
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
		t.Logf("Indexed %d files", res.Files)
	})

	t.Run("List Files", func(t *testing.T) {
		files, err := idx.ListFiles(ctx, "")
		if err != nil {
			t.Fatalf("ListFiles failed: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 indexed files, got %d", len(files))
		}
		t.Logf("Listed %d files", len(files))
	})

	t.Run("Search Semantic", func(t *testing.T) {
		searchOpts := &SearchOptions{
			Limit: 5,
			Mode:  SearchModeSemantic,
		}
		results, err := idx.Search(ctx, "", "hello", searchOpts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Errorf("expected some results, got 0")
		}
		t.Logf("Found %d semantic search results", len(results))
		for _, r := range results {
			t.Logf("  - %s (score: %.3f)", r.FilePath, r.Score)
		}
	})

	t.Run("Search Keyword", func(t *testing.T) {
		searchOpts := &SearchOptions{
			Limit: 5,
			Mode:  SearchModeKeyword,
		}
		results, err := idx.Search(ctx, "", "main", searchOpts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Errorf("expected some results for keyword search, got 0")
		}
		t.Logf("Found %d keyword search results", len(results))
		for _, r := range results {
			t.Logf("  - %s (score: %.3f)", r.FilePath, r.Score)
		}
	})
}

func TestEmbedderDirectIntegration(t *testing.T) {
	emb := embed.NewOpenAIEmbedder(
		"http://localhost:20003/v1",
		"unsloth/Qwen3.5-9B-GGUF:Q4_K_M",
		"",
		384,
	)

	t.Logf("Testing embedder with model: %s, dimension: %d", emb.Model(), emb.Dimension())

	texts := []string{
		"Hello, World!",
		"How are you today?",
		"This is a test embedding.",
	}

	embeddings, err := emb.Embed(texts)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Fatalf("expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		t.Logf("Embedding %d: dimension=%d, sample=%.4f", i, len(emb), emb[0])
	}
}
