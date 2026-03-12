package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiHandler_Config(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("GetConfig", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/config", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if _, ok := resp["system"]; !ok {
			t.Error("expected 'system' in config response")
		}
		if _, ok := resp["research"]; !ok {
			t.Error("expected 'research' in config response")
		}
		if _, ok := resp["codemogger"]; !ok {
			t.Error("expected 'codemogger' in config response")
		}
	})
}
