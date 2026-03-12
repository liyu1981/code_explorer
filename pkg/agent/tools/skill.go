package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

// ListSkillsTool allows an agent to list available skills
type ListSkillsTool struct {
	store *db.Store
}

func NewListSkillsTool(store *db.Store) *ListSkillsTool {
	return &ListSkillsTool{store: store}
}

func (t *ListSkillsTool) Name() string {
	return "list_skills"
}

func (t *ListSkillsTool) Description() string {
	return "Lists all available agent skills and their descriptions, including tags."
}

func (t *ListSkillsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

func (t *ListSkillsTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	skills, err := t.store.ListSkills(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list skills: %w", err)
	}

	type SkillInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Tags        string `json:"tags"`
	}

	info := make([]SkillInfo, len(skills))
	for i, s := range skills {
		info[i] = SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			Tags:        s.Tags,
		}
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal skills: %w", err)
	}

	return string(data), nil
}
