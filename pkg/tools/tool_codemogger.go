package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/constant"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	list_files_tool_name = "codemogger_list_files"
	list_files_tool_desc = "Lists all indexed files in the codebase."
)

type CodeMoggerListFilesTool struct {
	index *codemogger.CodeIndex
}

func NewCodeMoggerListFilesTool(index *codemogger.CodeIndex) Tool {
	return &CodeMoggerListFilesTool{index: index}
}

func (t *CodeMoggerListFilesTool) Name() string {
	return list_files_tool_name
}

func (t *CodeMoggerListFilesTool) Description() string {
	return list_files_tool_desc
}

func (t *CodeMoggerListFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"codebaseID": map[string]any{
				"type":        "string",
				"description": "The codebase ID to list files from",
			},
		},
		"required": []string{"codebaseID"},
	}
}

func (t *CodeMoggerListFilesTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	if t.index == nil {
		return "", fmt.Errorf("index is nil")
	}

	var req struct {
		CodebaseID string `json:"codebaseID"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if req.CodebaseID == "" {
		return "", fmt.Errorf("codebaseID is required")
	}

	files, err := t.index.ListFiles(ctx, req.CodebaseID)
	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	data, err := json.Marshal(files)
	if err != nil {
		return "", fmt.Errorf("failed to marshal files: %w", err)
	}

	return string(data), nil
}

type CodeMoggerSearchTool struct {
	index *codemogger.CodeIndex
}

func NewCodeMoggerSearchTool(index *codemogger.CodeIndex) Tool {
	return &CodeMoggerSearchTool{index: index}
}

func (t *CodeMoggerSearchTool) Name() string {
	return "codemogger_search"
}

func (t *CodeMoggerSearchTool) Description() string {
	return "Search the codebase using natural language (semantic) or keyword search."
}

func (t *CodeMoggerSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"codebaseID": map[string]any{
				"type":        "string",
				"description": "The codebase ID to search within",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "The search query (e.g., 'how is authentication implemented?')",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Maximum number of results to return (default %d)", constant.DefaultSearchLimit),
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "Search mode: 'hybrid', 'semantic', or 'keyword' (default 'hybrid')",
				"enum":        []string{"hybrid", "semantic", "keyword"},
			},
		},
		"required": []string{"codebaseID", "query"},
	}
}

func (t *CodeMoggerSearchTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	if t.index == nil {
		return "", fmt.Errorf("index is nil")
	}

	var req struct {
		CodebaseID string                `json:"codebaseID"`
		Query      string                `json:"query"`
		Limit      int                   `json:"limit"`
		Mode       codemogger.SearchMode `json:"mode"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal input: %w", err)
	}

	if req.CodebaseID == "" {
		return "", fmt.Errorf("codebaseID is required")
	}

	if req.Limit <= 0 {
		req.Limit = constant.DefaultSearchLimit
	}
	if req.Mode == "" {
		req.Mode = codemogger.SearchModeHybrid
	}

	opts := &codemogger.SearchOptions{
		Limit:          req.Limit,
		Mode:           req.Mode,
		IncludeSnippet: true,
	}

	results, err := t.index.Search(ctx, req.CodebaseID, req.Query, opts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	var markdown strings.Builder
	for i, res := range results {
		if stream != nil {
			stream.SendReasoning(fmt.Sprintf("Found relevant snippet in %s:%d\n", res.FilePath, res.StartLine))

			id, _ := gonanoid.New()
			stream.SendResourceMaterial(protocol.SourceMaterial{
				ID:        fmt.Sprintf("search-%d-%s", i, id),
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
