package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type mockStreamWriter struct {
	protocol.IStreamWriter
}

func (m *mockStreamWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestGetTreeTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	files := map[string]string{
		"main.go":           "package main",
		"bin/index.js":      "console.log('hi')",
		"src/lib/utils.go":  "package lib",
		".gitignore":        "ignored.txt\nnode_modules/",
		"ignored.txt":       "ignore me",
		"node_modules/a.js": "ignored",
		".git/config":       "git config",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	tool := NewGetTreeTool()
	state := map[string]any{"baseDir": tempDir}
	if err := tool.Bind(context.Background(), &state); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	stream := &mockStreamWriter{}

	t.Run("Depth 1", func(t *testing.T) {
		input := json.RawMessage(`{"depth": 1}`)
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Expected: .gitignore GEMINI.md (not here) main.go bin/ src/
		// (wait, .gitignore itself should be shown? Yes, usually)
		// The example says: GEMINI.md bin/
		// So we expect: .gitignore bin/ main.go src/ (alphabetical? depth-first? instruction says depth-first)

		// Let's see what the current implementation gives (indented tree)
		// But the instruction says: (all names in one line, with space as separator, and dir is suffixed with trailing slash)

		// For depth=1, depth-first doesn't matter much for order, just files in root and immediate dirs.

		parts := strings.Fields(res)
		sort.Strings(parts)
		expected := []string{".gitignore", "bin/", "main.go", "src/"}
		sort.Strings(expected)

		if strings.Join(parts, " ") != strings.Join(expected, " ") {
			t.Errorf("Expected %v, got %v", expected, parts)
		}
	})

	t.Run("Depth 2", func(t *testing.T) {
		input := json.RawMessage(`{"depth": 2}`)
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Expected: .gitignore bin/ bin/index.js main.go src/ src/lib/
		parts := strings.Fields(res)
		// Instruction says: list in depth-first order
		// bin/ comes before bin/index.js
		// src/ comes before src/lib/

		expected := []string{".gitignore", "bin/", "bin/index.js", "main.go", "src/", "src/lib/"}
		// Check if they are all present
		for _, e := range expected {
			found := false
			for _, p := range parts {
				if p == e {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Missing expected item: %s", e)
			}
		}

		// Also check that ignored.txt and node_modules/ are NOT present
		for _, p := range parts {
			if p == "ignored.txt" || strings.HasPrefix(p, "node_modules") || strings.HasPrefix(p, ".git/") {
				t.Errorf("Found ignored item: %s", p)
			}
		}
	})
}

func TestReadFileTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "readfile-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	content := "line1\nline2\nline3\nline4\nline5"
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)

	tool := NewReadFileTool()
	state := map[string]any{"baseDir": tempDir}
	if err := tool.Bind(context.Background(), &state); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	stream := &mockStreamWriter{}

	t.Run("Read whole file", func(t *testing.T) {
		input := json.RawMessage(`{"path": "test.txt"}`)
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if res != content {
			t.Errorf("Expected %q, got %q", content, res)
		}
	})

	t.Run("Read partial file", func(t *testing.T) {
		input := json.RawMessage(`{"path": "test.txt", "start_line": 2, "end_line": 4}`)
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		expected := "line2\nline3\nline4"
		if res != expected {
			t.Errorf("Expected %q, got %q", expected, res)
		}
	})

	t.Run("Out of bounds", func(t *testing.T) {
		input := json.RawMessage(`{"path": "test.txt", "start_line": 10}`)
		_, err := tool.Execute(context.Background(), input, stream)
		if err == nil {
			t.Fatal("Expected error for out of bounds start_line")
		}
	})
}

func TestGrepSearchTool(t *testing.T) {
	// Skip if rg or grep not available (not easy to check here, but we assume at least grep is there)
	tempDir, err := os.MkdirTemp("", "grep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("hello world\nfoo bar"), 0644)
	os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte("hello again"), 0644)

	tool := NewGrepSearchTool()
	state := map[string]any{"baseDir": tempDir}
	if err := tool.Bind(context.Background(), &state); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	stream := &mockStreamWriter{}

	t.Run("Search existing pattern", func(t *testing.T) {
		input := json.RawMessage(`{"pattern": "hello"}`)
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if !strings.Contains(res, "a.txt") || !strings.Contains(res, "b.txt") {
			t.Errorf("Expected results from a.txt and b.txt, got %q", res)
		}
	})

	t.Run("Search non-existing pattern", func(t *testing.T) {
		input := json.RawMessage(`{"pattern": "nonexistent_pattern_12345"}`)
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if res != "No matches found." {
			t.Errorf("Expected 'No matches found.', got %q", res)
		}
	})
}
