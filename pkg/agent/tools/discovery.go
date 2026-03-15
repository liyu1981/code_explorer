package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/util"
)

// ReadFileTool allows reading content from local files
type ReadFileTool struct {
	baseDir string
}

func NewReadFileBaseTool() *ReadFileTool {
	return &ReadFileTool{}
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

func (t *ReadFileTool) Bind(ctx context.Context, state map[string]any) {
	if baseDir := state["baseDir"].(string); baseDir != "" {
		t.baseDir = baseDir
	}
}

// GetTreeTool provides directory structure
type GetTreeTool struct {
	baseDir string
}

func NewGetTreeBaseTool(baseDir string) *GetTreeTool {
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

// Execute will return a string with the directory structure as follows
//   example: depth=1
//            GEMINI.md  bin/
//            (all names in one line, with space as separator, and dir is suffixed with trailing slash)

//	example: depth=2
//	         GEMINI.md  bin/ bin/index.js
//	         (all names in one line, with space as separator, list in depth-first order of all files within depth)
//
// also will respect the .gitignore rules (with help from gocodewalker, see ref in codemogger/scan/walker.go)
func (t *GetTreeTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Depth int `json:"depth"`
	}
	json.Unmarshal(input, &req)
	if req.Depth <= 0 {
		req.Depth = 3
	}

	fileListQueue := util.StartFileWalker(t.baseDir, true)

	pathMap := make(map[string]bool)
	for f := range fileListQueue {
		rel, err := filepath.Rel(t.baseDir, f.Location)
		if err != nil {
			continue
		}
		parts := strings.Split(rel, string(os.PathSeparator))

		for i := 1; i <= len(parts) && i <= req.Depth; i++ {
			subPath := filepath.Join(parts[:i]...)
			if i < len(parts) {
				// It's a directory
				pathMap[filepath.ToSlash(subPath)+"/"] = true
			} else {
				// It's a file
				pathMap[filepath.ToSlash(subPath)] = true
			}
		}
	}

	var allPaths []string
	for p := range pathMap {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	return strings.Join(allPaths, " "), nil
}

// GrepSearchTool allows fast pattern search
type GrepSearchTool struct {
	baseDir string
}

func NewGrepSearchBaseTool(baseDir string) *GrepSearchTool {
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
