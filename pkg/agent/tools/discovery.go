package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

// ReadFileTool allows reading content from local files
type ReadFileTool struct {
	baseDir string
}

func NewReadFileTool(baseDir string) *ReadFileTool {
	return &ReadFileTool{baseDir: baseDir}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Reads the content of a file. Supports optional start_line and end_line."
}

func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file relative to codebase root",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "Optional 1-based start line number",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": "Optional 1-based end line number",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Path      string `json:"path"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	fullPath := filepath.Join(t.baseDir, req.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if req.StartLine > 0 {
		start := req.StartLine - 1
		if start >= len(lines) {
			return "", fmt.Errorf("start_line %d is out of bounds", req.StartLine)
		}
		end := len(lines)
		if req.EndLine > 0 && req.EndLine <= len(lines) {
			end = req.EndLine
		}
		if end < start {
			return "", fmt.Errorf("end_line %d is before start_line %d", req.EndLine, req.StartLine)
		}
		lines = lines[start:end]
	}

	return strings.Join(lines, "\n"), nil
}

// GetTreeTool provides directory structure
type GetTreeTool struct {
	baseDir string
}

func NewGetTreeTool(baseDir string) *GetTreeTool {
	return &GetTreeTool{baseDir: baseDir}
}

func (t *GetTreeTool) Name() string {
	return "get_tree"
}

func (t *GetTreeTool) Description() string {
	return "Returns a tree-like directory structure of the codebase."
}

func (t *GetTreeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"depth": map[string]any{
				"type":        "integer",
				"description": "Max depth to traverse (default 3)",
			},
		},
	}
}

func (t *GetTreeTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Depth int `json:"depth"`
	}
	json.Unmarshal(input, &req)
	if req.Depth <= 0 {
		req.Depth = 3
	}

	var sb strings.Builder
	err := filepath.WalkDir(t.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(t.baseDir, path)
		if rel == "." {
			return nil
		}

		// Skip hidden dirs
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) > req.Depth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		indent := strings.Repeat("  ", len(parts)-1)
		if d.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

// GrepSearchTool allows fast pattern search
type GrepSearchTool struct {
	baseDir string
}

func NewGrepSearchTool(baseDir string) *GrepSearchTool {
	return &GrepSearchTool{baseDir: baseDir}
}

func (t *GrepSearchTool) Name() string {
	return "grep_search"
}

func (t *GrepSearchTool) Description() string {
	return "Performs a fast regex search across the codebase using ripgrep (if available) or grep."
}

func (t *GrepSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern to search for",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepSearchTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	// Try ripgrep first, then grep
	cmd := exec.CommandContext(ctx, "rg", "-n", "--max-count", "100", req.Pattern, t.baseDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to standard grep
		cmd = exec.CommandContext(ctx, "grep", "-rn", "--max-count=100", req.Pattern, t.baseDir)
		output, err = cmd.CombinedOutput()
	}

	// Clean up paths in output to be relative
	result := string(output)
	result = strings.ReplaceAll(result, t.baseDir, ".")

	if result == "" && err != nil {
		return "No matches found.", nil
	}

	return result, nil
}
