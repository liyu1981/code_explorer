package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

func setupTestHandler(t *testing.T) (*ApiHandler, *codemogger.CodeIndex, func()) {
	tmpDir, err := os.MkdirTemp("", "api-test")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	index, err := codemogger.NewCodeIndex(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(&ApiConfig{Index: index})

	cleanup := func() {
		h.Stop()
		index.Close()
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
