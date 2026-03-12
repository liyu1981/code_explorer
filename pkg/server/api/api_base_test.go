package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/constant"
)

func TestApiHandler_Base(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("Health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["status"] != "healthy" {
			t.Errorf("expected healthy, got %v", resp["status"])
		}
	})

	t.Run("Version", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/version", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]string
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["version"] != constant.Version {
			t.Errorf("expected version %s, got %s", constant.Version, resp["version"])
		}
	})

	t.Run("ListCodebases", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/codebases", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var codebases []any
		if err := json.NewDecoder(w.Body).Decode(&codebases); err != nil {
			t.Fatal(err)
		}
		// Initially it might be empty, but it should be a JSON array
	})
}
