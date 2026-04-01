package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func setupTestCodemoggerIndex(t *testing.T) (*codemogger.CodeIndex, func()) {
	tmpDir, err := os.MkdirTemp("", "api-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := config.DefaultConfig()
	config.Set(cfg)
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("Failed to migrate db: %v", err)
	}
	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	store := db.NewStore(sqlDB, dbPath)
	idx, err := codemogger.NewCodeIndex(cfg, store)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Use MockEmbedder to avoid network calls
	idx.SetEmbedder(&embed.MockEmbedder{DimVal: 384})

	cleanup := func() {
		idx.Close()
		os.RemoveAll(tmpDir)
	}

	return idx, cleanup
}

func TestApiListCodebases(t *testing.T) {
	idx, cleanup := setupTestCodemoggerIndex(t)
	defer cleanup()

	h := NewHandler(&ApiConfig{CodemoggerIndex: idx})
	defer h.Stop()
	req := httptest.NewRequest("GET", "/api/codemogger/codebases", nil)
	rr := httptest.NewRecorder()

	h.handleListCodebases(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var codebases []codemogger.Codebase
	if err := json.NewDecoder(rr.Body).Decode(&codebases); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// Initially should be empty
	if len(codebases) != 0 {
		t.Errorf("Expected 0 codebases, got %v", len(codebases))
	}
}

func TestApiCodemoggerIndex(t *testing.T) {
	idx, cleanup := setupTestCodemoggerIndex(t)
	defer cleanup()

	// Create a dummy file to index
	tmpDir, _ := os.MkdirTemp("", "api-index-test-*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(testFile, []byte("package main\nfunc main() {}"), 0644)

	h := NewHandler(&ApiConfig{CodemoggerIndex: idx})
	defer h.Stop()

	body, _ := json.Marshal(map[string]any{
		"dir":   tmpDir,
		"langs": []string{"go"},
	})
	req := httptest.NewRequest("POST", "/api/codemogger/index", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.handleIndex(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "indexing_queued" {
		t.Errorf("Expected indexing_queued, got %v", resp["status"])
	}
}

func TestApiSearch(t *testing.T) {
	idx, cleanup := setupTestCodemoggerIndex(t)
	defer cleanup()

	h := NewHandler(&ApiConfig{CodemoggerIndex: idx})
	defer h.Stop()

	body, _ := json.Marshal(map[string]any{
		"query": "test query",
		"limit": 5,
		"mode":  "hybrid",
	})
	req := httptest.NewRequest("POST", "/api/codemogger/search", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.handleSearch(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var results []codemogger.SearchResult
	if err := json.NewDecoder(rr.Body).Decode(&results); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}
