package llm

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

func NewReadFileTool() Tool {
	return &ReadFileTool{}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Reads the content of a file. Supports optional start_line and end_line."
}

func (t *ReadFileTool) Clone() Tool {
	return &ReadFileTool{baseDir: t.baseDir}
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
	if t.baseDir == "" {
		return "", fmt.Errorf("baseDir is empty")
	}

	var req struct {
		Path      string `json:"path"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	var fullPath string
	if filepath.IsAbs(req.Path) {
		fullPath = req.Path
	} else {
		fullPath = filepath.Join(t.baseDir, req.Path)
	}
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

func (t *ReadFileTool) Bind(ctx context.Context, state *map[string]any) error {
	baseDir, err := util.SafeExtract[string](state, "baseDir")
	if err != nil {
		return fmt.Errorf("bind failed: %v", err)
	}
	if baseDir != "" {
		t.baseDir = baseDir
		return nil
	} else {
		return fmt.Errorf("bind failed: basedDir is nil")
	}
}

// GetTreeTool provides directory structure
type GetTreeTool struct {
	baseDir string
}

func NewGetTreeTool() Tool {
	return &GetTreeTool{}
}

func (t *GetTreeTool) Name() string {
	return "get_tree"
}

func (t *GetTreeTool) Description() string {
	return "Returns a tree-like directory structure of the codebase."
}

func (t *GetTreeTool) Clone() Tool {
	return &GetTreeTool{baseDir: t.baseDir}
}

func (t *GetTreeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"depth": map[string]any{
				"type":        "integer",
				"description": "Max depth to traverse (default 0, unlimited)",
			},
		},
	}
}

// Execute returns a string with the directory structure in unix `tree` style.
// If depth is not provided (or <= 0), it recurses infinitely.
//
// Example output (depth=2):
//
//	.
//	├── GEMINI.md
//	├── bin/
//	│   └── index.js
//	└── cmd/
//	    └── main.go
func (t *GetTreeTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	if t.baseDir == "" {
		return "", fmt.Errorf("baseDir is empty")
	}

	var req struct {
		Depth int `json:"depth"`
	}
	json.Unmarshal(input, &req)
	// depth <= 0 means unlimited
	maxDepth := req.Depth

	// Collect all file paths from the walker
	fileListQueue := util.StartFileWalker(t.baseDir, true)

	// Build a set of relative file paths
	filePaths := make(map[string]struct{})
	for f := range fileListQueue {
		rel, err := filepath.Rel(t.baseDir, f.Location)
		if err != nil {
			continue
		}
		filePaths[filepath.ToSlash(rel)] = struct{}{}
	}

	// Build a tree node structure
	type node struct {
		name     string
		children map[string]*node
		isDir    bool
	}

	root := &node{name: ".", children: make(map[string]*node), isDir: true}

	for rel := range filePaths {
		parts := strings.Split(rel, "/")
		cur := root
		for i, part := range parts {
			if _, ok := cur.children[part]; !ok {
				isDir := i < len(parts)-1
				cur.children[part] = &node{
					name:     part,
					children: make(map[string]*node),
					isDir:    isDir,
				}
			}
			if i < len(parts)-1 {
				cur.children[part].isDir = true
			}
			cur = cur.children[part]
		}
	}

	// Render the tree recursively
	var sb strings.Builder
	sb.WriteString(".\n")

	var render func(n *node, prefix string, depth int)
	render = func(n *node, prefix string, depth int) {
		if maxDepth > 0 && depth > maxDepth {
			return
		}

		// Sort children: dirs first, then files, both alphabetically
		var dirs, files []string
		for name, child := range n.children {
			if child.isDir {
				dirs = append(dirs, name)
			} else {
				files = append(files, name)
			}
		}
		sort.Strings(dirs)
		sort.Strings(files)
		sorted := append(dirs, files...)

		for i, name := range sorted {
			child := n.children[name]
			isLast := i == len(sorted)-1

			connector := "├── "
			if isLast {
				connector = "└── "
			}

			label := name
			if child.isDir {
				label = name + "/"
			}
			sb.WriteString(prefix + connector + label + "\n")

			if child.isDir {
				extension := "│   "
				if isLast {
					extension = "    "
				}
				render(child, prefix+extension, depth+1)
			}
		}
	}

	render(root, "", 1)

	return strings.TrimRight(sb.String(), "\n"), nil
}

func (t *GetTreeTool) Bind(ctx context.Context, state *map[string]any) error {
	baseDir, err := util.SafeExtract[string](state, "baseDir")
	if err != nil {
		return fmt.Errorf("bind failed: %v", err)
	}
	if baseDir != "" {
		t.baseDir = baseDir
		return nil
	} else {
		return fmt.Errorf("bind failed: basedDir is nil")
	}
}

// GrepSearchTool allows fast pattern search
type GrepSearchTool struct {
	baseDir string
}

func NewGrepSearchTool() Tool {
	return &GrepSearchTool{}
}

func (t *GrepSearchTool) Name() string {
	return "grep_search"
}

func (t *GrepSearchTool) Description() string {
	return "Performs a fast regex search across the codebase using ripgrep (if available) or grep."
}

func (t *GrepSearchTool) Clone() Tool {
	return &GrepSearchTool{baseDir: t.baseDir}
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
	if t.baseDir == "" {
		return "", fmt.Errorf("baseDir is empty")
	}

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

func (t *GrepSearchTool) Bind(ctx context.Context, state *map[string]any) error {
	baseDir, err := util.SafeExtract[string](state, "baseDir")
	if err != nil {
		return fmt.Errorf("bind failed: %v", err)
	}
	if baseDir != "" {
		t.baseDir = baseDir
		return nil
	} else {
		return fmt.Errorf("bind failed: basedDir is nil")
	}
}
