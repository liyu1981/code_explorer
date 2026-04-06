package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
	"github.com/liyu1981/code_explorer/pkg/server/api"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	zindex "github.com/liyu1981/code_explorer/pkg/zoekt/index"
)

func TestUIServer_SetupRoutes(t *testing.T) {
	// Setup a temporary UI directory
	tmpDir, err := os.MkdirTemp("", "ui-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some dummy files
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("index content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test.html"), []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "static"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "static", "app.js"), []byte("js content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Setup test DB so NewHandler doesn't panic
	db.ResetStoreForTest()
	sqlitefs.ResetFS()
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := config.DefaultConfig()
	config.Set(cfg)
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("OpenLibsqlDb: %v", err)
	}
	store := db.NewStore(sqlDB, dbPath)
	idx, err := codemogger.NewCodeIndex(cfg, store)
	if err != nil {
		t.Fatalf("NewCodeIndex: %v", err)
	}
	defer idx.Close()
	zFs := sqlitefs.OpenFS(store)
	zIdx := zindex.NewZoektIndex(store, zFs)

	apiHandler := api.NewHandler(&api.ApiConfig{CodemoggerIndex: idx, ZoektIndex: zIdx})
	defer apiHandler.Stop()

	s := &UIServer{
		staticFS:   os.DirFS(tmpDir),
		ApiHandler: apiHandler,
	}

	handler := s.SetupRoutes()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"Root", "/", http.StatusOK, "index content"},
		{"Exact HTML", "/test.html", http.StatusOK, "test content"},
		{"Clean Path to HTML", "/test", http.StatusOK, "test content"},
		{"Static File", "/static/app.js", http.StatusOK, "js content"},
		{"API Health", "/health", http.StatusOK, `{"status":"healthy"}`},
		{"Not Found", "/non-existent", http.StatusNotFound, "404 page not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			body := strings.TrimSpace(w.Body.String())
			if tt.expectedBody != "" && !strings.Contains(body, tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, body)
			}
		})
	}
}

func TestCorsMiddleware(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS headers for OPTIONS")
	}

	// Test GET request with Origin
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin: http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
