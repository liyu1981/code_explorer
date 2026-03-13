package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

// ListFilesTool exposes codemogger's ListFiles functionality to the agent
type ListFilesTool struct {
	index *codemogger.CodeIndex
}

func NewListFilesTool(index *codemogger.CodeIndex) *ListFilesTool {
	return &ListFilesTool{index: index}
}

func (t *ListFilesTool) Name() string {
	return "codemogger_list_files"
}

func (t *ListFilesTool) Description() string {
	return "Lists all indexed files in the codebase."
}

func (t *ListFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	files, err := t.index.ListFiles(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	data, err := json.Marshal(files)
	if err != nil {
		return "", fmt.Errorf("failed to marshal files: %w", err)
	}

	return string(data), nil
}

// SearchTool exposes codemogger's Search functionality to the agent
type SearchTool struct {
	index *codemogger.CodeIndex
}

func NewSearchTool(index *codemogger.CodeIndex) *SearchTool {
	return &SearchTool{index: index}
}

func (t *SearchTool) Name() string {
	return "codemogger_search"
}

func (t *SearchTool) Description() string {
	return "Search the codebase using natural language (semantic) or keyword search."
}

func (t *SearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query (e.g., 'how is authentication implemented?')",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default 5)",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "Search mode: 'hybrid', 'semantic', or 'keyword' (default 'hybrid')",
				"enum":        []string{"hybrid", "semantic", "keyword"},
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Query string                `json:"query"`
		Limit int                   `json:"limit"`
		Mode  codemogger.SearchMode `json:"mode"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Mode == "" {
		req.Mode = codemogger.SearchModeHybrid
	}

	opts := &codemogger.SearchOptions{
		Limit:          req.Limit,
		Mode:           req.Mode,
		IncludeSnippet: true,
	}

	results, err := t.index.Search(ctx, req.Query, opts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	var markdown strings.Builder
	for i, res := range results {
		if stream != nil {
			// Emit a reasoning update so the user sees progress for each file
			stream.SendReasoning(fmt.Sprintf("Found relevant snippet in %s:%d\n", res.FilePath, res.StartLine))

			stream.SendResourceMaterial(protocol.SourceMaterial{
				ID:        fmt.Sprintf("search-%d", i),
				Path:      res.FilePath,
				Snippet:   res.Snippet,
				StartLine: res.StartLine,
				EndLine:   res.EndLine,
			})
		}

		markdown.WriteString(fmt.Sprintf("### %s:%d-%d\n", res.FilePath, res.StartLine, res.EndLine))
		markdown.WriteString("```\n")
		markdown.WriteString(res.Snippet)
		if !strings.HasSuffix(res.Snippet, "\n") {
			markdown.WriteString("\n")
		}
		markdown.WriteString("```\n\n")
	}

	return markdown.String(), nil
}
