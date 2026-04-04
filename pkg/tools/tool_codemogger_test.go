package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func TestCodemoggerTools(t *testing.T) {
	dir, err := os.MkdirTemp("", "tools-test-*")
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
	store := db.NewStore(sqlDB, dbPath)
	idx, err := codemogger.NewCodeIndex(cfg, store)
	if err != nil {
		t.Fatalf("NewCodeIndex: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	t.Run("ListFilesTool", func(t *testing.T) {
		tool := NewCodeMoggerListFilesTool(idx)

		if tool.Name() != "codemogger_list_files" {
			t.Errorf("unexpected name: %s", tool.Name())
		}

		got, err := tool.Execute(ctx, json.RawMessage(`{"codebaseID": "nonexistent"}`), nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		var files []string
		if err := json.Unmarshal([]byte(got), &files); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("ListFilesTool_MissingCodebaseID", func(t *testing.T) {
		tool := NewCodeMoggerListFilesTool(idx)

		_, err := tool.Execute(ctx, json.RawMessage("{}"), nil)
		if err == nil {
			t.Error("expected error for missing codebaseID")
		}
	})

	t.Run("SearchTool", func(t *testing.T) {
		tool := NewCodeMoggerSearchTool(idx)

		if tool.Name() != "codemogger_search" {
			t.Errorf("unexpected name: %s", tool.Name())
		}

		input := `{"codebaseID": "nonexistent", "query": "test", "limit": 5}`
		_, err := tool.Execute(ctx, json.RawMessage(input), nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	})

	t.Run("SearchTool_MissingCodebaseID", func(t *testing.T) {
		tool := NewCodeMoggerSearchTool(idx)

		input := `{"query": "test", "limit": 5}`
		_, err := tool.Execute(ctx, json.RawMessage(input), nil)
		if err == nil {
			t.Error("expected error for missing codebaseID")
		}
	})
}
