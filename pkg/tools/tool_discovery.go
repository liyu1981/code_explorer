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

// TODO: remove baseDir from the struct, as it will be passed in as param when execute
// ReadFileTool allows reading content from local files
type ReadFileTool struct{}

func NewReadFileTool() Tool {
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
			"base_dir": map[string]any{
				"type":        "string",
				"description": "The codebase root directory",
			},
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
		"required": []string{"base_dir", "path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		BaseDir   string `json:"base_dir"`
		Path      string `json:"path"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	if req.BaseDir == "" {
		return "", fmt.Errorf("base_dir is required")
	}

	var fullPath string
	if filepath.IsAbs(req.Path) {
		fullPath = req.Path
	} else {
		fullPath = filepath.Join(req.BaseDir, req.Path)
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

// GetTreeTool provides directory structure
type GetTreeTool struct{}

func NewGetTreeTool() Tool {
	return &GetTreeTool{}
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
			"base_dir": map[string]any{
				"type":        "string",
				"description": "The codebase root directory",
			},
			"depth": map[string]any{
				"type":        "integer",
				"description": "Max depth to traverse (default 0, unlimited)",
			},
		},
		"required": []string{"base_dir"},
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
	var req struct {
		BaseDir string `json:"base_dir"`
		Depth   int    `json:"depth"`
	}
	json.Unmarshal(input, &req)

	if req.BaseDir == "" {
		return "", fmt.Errorf("base_dir is required")
	}

	// depth <= 0 means unlimited
	maxDepth := req.Depth

	// Collect all file paths from the walker
	fileListQueue := util.StartFileWalker(req.BaseDir, true)

	// Build a set of relative file paths
	filePaths := make(map[string]struct{})
	for f := range fileListQueue {
		rel, err := filepath.Rel(req.BaseDir, f.Location)
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

// GrepSearchTool allows fast pattern search
type GrepSearchTool struct{}

func NewGrepSearchTool() Tool {
	return &GrepSearchTool{}
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
			"base_dir": map[string]any{
				"type":        "string",
				"description": "The codebase root directory",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern to search for",
			},
		},
		"required": []string{"base_dir", "pattern"},
	}
}

func (t *GrepSearchTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		BaseDir string `json:"base_dir"`
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	if req.BaseDir == "" {
		return "", fmt.Errorf("base_dir is required")
	}

	// Try ripgrep first, then grep
	cmd := exec.CommandContext(ctx, "rg", "-n", "--max-count", "100", req.Pattern, req.BaseDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to standard grep
		cmd = exec.CommandContext(ctx, "grep", "-rn", "--max-count=100", req.Pattern, req.BaseDir)
		output, err = cmd.CombinedOutput()
	}

	// Clean up paths in output to be relative
	result := string(output)
	result = strings.ReplaceAll(result, req.BaseDir, ".")

	if result == "" && err != nil {
		return "No matches found.", nil
	}

	return result, nil
}
