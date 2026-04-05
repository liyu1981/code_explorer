package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type mockStreamWriter struct{}

func (w *mockStreamWriter) WriteOpenAIChunk(id, model, content string, finishReason *string) error {
	return nil
}
func (w *mockStreamWriter) WriteCEEvent(event protocol.CEEvent) error { return nil }
func (w *mockStreamWriter) WriteDone() error                          { return nil }
func (w *mockStreamWriter) SendReasoning(content string) error        { return nil }
func (w *mockStreamWriter) SendTurnStarted(query string, timestamp int64) error {
	return nil
}
func (w *mockStreamWriter) SendStepUpdate(stepID string, label string, status protocol.StepStatus) error {
	return nil
}
func (w *mockStreamWriter) SendSourceAdded(source protocol.SourceMaterial) error {
	return nil
}
func (w *mockStreamWriter) SendResourceMaterial(resource protocol.SourceMaterial) error {
	return nil
}
func (w *mockStreamWriter) SendToolCall(tool string, params any) error { return nil }
func (w *mockStreamWriter) SendToolResponse(tool string, response any) error {
	return nil
}
func (w *mockStreamWriter) SendTryRunStart(try int64) error  { return nil }
func (w *mockStreamWriter) SendTryRunEnd(try int64) error    { return nil }
func (w *mockStreamWriter) SendTryRunFailed(try int64) error { return nil }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	stream := &mockStreamWriter{}

	parseTreeLines := func(res string) []string {
		var names []string
		for _, line := range strings.Split(res, "\n") {
			if line == "." {
				continue
			}
			idx := strings.LastIndex(line, "── ")
			if idx == -1 {
				continue
			}
			names = append(names, line[idx+len("── "):])
		}
		return names
	}

	containsAll := func(t *testing.T, got []string, want []string) {
		t.Helper()
		set := make(map[string]bool, len(got))
		for _, g := range got {
			set[g] = true
		}
		for _, w := range want {
			if !set[w] {
				t.Errorf("missing expected entry %q in output", w)
			}
		}
	}

	containsNone := func(t *testing.T, got []string, banned []string) {
		t.Helper()
		for _, g := range got {
			for _, b := range banned {
				if g == b || strings.HasPrefix(g, b) {
					t.Errorf("found unexpected/ignored entry %q in output", g)
				}
			}
		}
	}

	t.Run("Root line", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "depth": 1}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if !strings.HasPrefix(res, ".\n") {
			t.Errorf("expected output to start with '.\n', got: %q", res[:minInt(len(res), 10)])
		}
	})

	t.Run("Depth 1", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "depth": 1}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		entries := parseTreeLines(res)

		containsAll(t, entries, []string{".gitignore", "bin/", "main.go", "src/"})
		containsNone(t, entries, []string{"index.js", "lib/", "utils.go"})
		containsNone(t, entries, []string{"ignored.txt", "node_modules/", ".git/"})
	})

	t.Run("Depth 2", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "depth": 2}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		entries := parseTreeLines(res)

		containsAll(t, entries, []string{".gitignore", "bin/", "index.js", "main.go", "src/", "lib/"})
		containsNone(t, entries, []string{"utils.go"})
		containsNone(t, entries, []string{"ignored.txt", "node_modules/", ".git/"})
	})

	t.Run("Unlimited depth", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		entries := parseTreeLines(res)

		containsAll(t, entries, []string{".gitignore", "bin/", "index.js", "main.go", "src/", "lib/", "utils.go"})
		containsNone(t, entries, []string{"ignored.txt", "node_modules/", ".git/"})
	})

	t.Run("Tree connectors", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "depth": 2}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !strings.Contains(res, "├── ") && !strings.Contains(res, "└── ") {
			t.Errorf("expected tree connectors (├── or └── ) in output, got:\n%s", res)
		}
		lines := strings.Split(res, "\n")
		for i, line := range lines {
			if i == len(lines)-1 {
				continue
			}
			nextIndent := len(lines[i+1]) - len(strings.TrimLeft(lines[i+1], "│ "))
			curIndent := len(line) - len(strings.TrimLeft(line, "│ "))
			if nextIndent <= curIndent && strings.Contains(line, "── ") {
				prefix := line[:strings.Index(line, "── ")]
				if strings.HasSuffix(prefix, "├") {
					if nextIndent < curIndent {
						t.Errorf("line %d should use └── (last sibling) but uses ├──: %q", i, line)
					}
				}
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
	stream := &mockStreamWriter{}

	t.Run("Read whole file", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "path": "test.txt"}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if res != content {
			t.Errorf("Expected %q, got %q", content, res)
		}
	})

	t.Run("Read partial file", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "path": "test.txt", "start_line": 2, "end_line": 4}`, tempDir))
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
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "path": "test.txt", "start_line": 10}`, tempDir))
		_, err := tool.Execute(context.Background(), input, stream)
		if err == nil {
			t.Fatal("Expected error for out of bounds start_line")
		}
	})
}

func TestGrepSearchTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("hello world\nfoo bar"), 0644)
	os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte("hello again"), 0644)

	tool := NewGrepSearchTool()
	stream := &mockStreamWriter{}

	t.Run("Search existing pattern", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "pattern": "hello"}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if !strings.Contains(res, "a.txt") || !strings.Contains(res, "b.txt") {
			t.Errorf("Expected results from a.txt and b.txt, got %q", res)
		}
	})

	t.Run("Search non-existing pattern", func(t *testing.T) {
		input := json.RawMessage(fmt.Sprintf(`{"base_dir": %q, "pattern": "nonexistent_pattern_12345"}`, tempDir))
		res, err := tool.Execute(context.Background(), input, stream)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if res != "No matches found." {
			t.Errorf("Expected 'No matches found.', got %q", res)
		}
	})
}
