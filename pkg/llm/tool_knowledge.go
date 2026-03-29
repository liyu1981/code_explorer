package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/util"
)

// SaveKnowledgeTool allows an agent to save a knowledge page
type SaveKnowledgeTool struct {
	store *db.Store
}

func NewSaveKnowledgeTool() Tool {
	return &SaveKnowledgeTool{}
}

func (t *SaveKnowledgeTool) Name() string {
	return "save_knowledge"
}

func (t *SaveKnowledgeTool) Description() string {
	return "Saves or updates a knowledge page for a codebase."
}

func (t *SaveKnowledgeTool) Clone() Tool {
	return &SaveKnowledgeTool{store: t.store}
}

func (t *SaveKnowledgeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"codebase_id": map[string]any{
				"type":        "string",
				"description": "ID of the codebase",
			},
			"slug": map[string]any{
				"type":        "string",
				"description": "Unique slug for the page (e.g. 'architecture', 'api-reference')",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Title of the page",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Markdown content of the page",
			},
		},
		"required": []string{"codebase_id", "slug", "title", "content"},
	}
}

func (t *SaveKnowledgeTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	if t.store == nil {
		return "", fmt.Errorf("store is nil")
	}

	var req struct {
		CodebaseID string `json:"codebase_id"`
		Slug       string `json:"slug"`
		Title      string `json:"title"`
		Content    string `json:"content"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	// Check if already exists
	existing, err := t.store.GetKnowledgePageBySlug(ctx, req.CodebaseID, req.Slug)
	if err != nil {
		return "", err
	}

	if existing != nil {
		existing.Title = req.Title
		existing.Content = req.Content
		err = t.store.UpdateKnowledgePage(ctx, existing)
	} else {
		page := &db.KnowledgePage{
			CodebaseID: req.CodebaseID,
			Slug:       req.Slug,
			Title:      req.Title,
			Content:    req.Content,
		}
		err = t.store.CreateKnowledgePage(ctx, page)
	}

	if err != nil {
		return "", fmt.Errorf("failed to save knowledge: %w", err)
	}

	return "Knowledge page saved successfully", nil
}

func (t *SaveKnowledgeTool) Bind(ctx context.Context, state *map[string]any) error {
	store, err := util.SafeExtract[*db.Store](state, "store")
	if err != nil {
		return fmt.Errorf("bind failed: %v", err)
	}
	if store != nil {
		t.store = store
		return nil
	} else {
		return fmt.Errorf("bind failed: store is nil")
	}
}
