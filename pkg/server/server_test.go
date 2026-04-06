package server

import (
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
	"github.com/liyu1981/code_explorer/pkg/server/api"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	zindex "github.com/liyu1981/code_explorer/pkg/zoekt/index"
)

func TestServerSetup(t *testing.T) {
	dir, err := os.MkdirTemp("", "server-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")
	cfg := config.DefaultConfig()
	config.Set(cfg)
	if err := db.Migrate(dbPath); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	sqlDB, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("OpenLibsqlDb: %v", err)
	}
	db.ResetStoreForTest()
	store := db.NewStore(sqlDB, dbPath)

	idx, err := codemogger.NewCodeIndex(cfg, store)
	if err != nil {
		t.Fatalf("NewCodeIndex: %v", err)
	}
	defer idx.Close()
	idx.SetEmbedder(&embed.MockEmbedder{DimVal: 384})

	zFs := sqlitefs.OpenFS(store)
	zIdx := zindex.NewZoektIndex(store, zFs)

	apiHandler := api.NewHandler(&api.ApiConfig{CodemoggerIndex: idx, ZoektIndex: zIdx})
	defer apiHandler.Stop()

	uiServer := NewUIServer(&Config{ApiHandler: apiHandler})
	handler := uiServer.SetupRoutes()

	if handler == nil {
		t.Fatalf("New() returned nil handler")
	}

	tests := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/api/version", http.StatusOK},
		{"GET", "/health", http.StatusOK},
		{"GET", "/non-existent", http.StatusNotFound},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("%s %s got status %d; want %d", tt.method, tt.path, w.Code, tt.status)
		}
	}
}
