package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

func TestCodemoggerTools(t *testing.T) {
	dir, err := os.MkdirTemp("", "tools-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")
	idx, err := codemogger.NewCodeIndex(dbPath)
	if err != nil {
		t.Fatalf("NewCodeIndex: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	t.Run("ListFilesTool", func(t *testing.T) {
		tool := NewListFilesTool()
		if err := tool.Bind(ctx, map[string]any{"index": idx}); err != nil {
			t.Fatalf("Bind failed: %v", err)
		}

		if tool.Name() != "codemogger_list_files" {
			t.Errorf("unexpected name: %s", tool.Name())
		}

		got, err := tool.Execute(ctx, json.RawMessage("{}"), nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		var files []string
		if err := json.Unmarshal([]byte(got), &files); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		// Expect empty list for new DB
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("SearchTool", func(t *testing.T) {
		tool := NewSearchTool()
		if err := tool.Bind(ctx, map[string]any{"index": idx}); err != nil {
			t.Fatalf("Bind failed: %v", err)
		}

		if tool.Name() != "codemogger_search" {
			t.Errorf("unexpected name: %s", tool.Name())
		}

		input := `{"query": "test", "limit": 5}`
		_, err := tool.Execute(ctx, json.RawMessage(input), nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		// Since DB is empty, markdown output should be empty
	})
}
