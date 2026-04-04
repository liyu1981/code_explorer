package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func setupTestHandler(t *testing.T) (*ApiHandler, *codemogger.CodeIndex, func()) {
	tmpDir, err := os.MkdirTemp("", "api-test")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := config.DefaultConfig()
	config.Set(cfg)

	if err := db.Migrate(dbPath); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}
	store := db.NewStore(sqlDB, dbPath)

	index, err := codemogger.NewCodeIndex(cfg, store)
	if err != nil {
		sqlDB.Close()
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	h := NewHandler(&ApiConfig{CodemoggerIndex: index})

	cleanup := func() {
		h.Stop()
		index.Close()
		sqlDB.Close()
		os.RemoveAll(tmpDir)
	}

	return h, index, cleanup
}

func doRequest(h *ApiHandler, method, path string, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(method, path, nil)
	if body != "" {
		// Set body if needed (simplified for now)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}
