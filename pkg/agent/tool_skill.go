package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type ListAgentSkillsTool struct {
	store *db.Store
}

func NewListAgentSkillsTool() Tool {
	return &ListAgentSkillsTool{}
}

func (t *ListAgentSkillsTool) Name() string {
	return "list_agent_skills"
}

func (t *ListAgentSkillsTool) Description() string {
	return "Lists all available agent skills and their descriptions, including tags."
}

func (t *ListAgentSkillsTool) Clone() Tool {
	return &ListAgentSkillsTool{store: t.store}
}

func (t *ListAgentSkillsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

func (t *ListAgentSkillsTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	if t.store == nil {
		return "", fmt.Errorf("store is nil")
	}

	skills, err := t.store.ListAgentSkills(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list agent skills: %w", err)
	}

	type SkillInfo struct {
		Name  string `json:"name"`
		Tags  string `json:"tags"`
		Tools string `json:"tools"`
	}

	info := make([]SkillInfo, len(skills))
	for i, s := range skills {
		info[i] = SkillInfo{
			Name:  s.Name,
			Tags:  s.Tags,
			Tools: s.Tools,
		}
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal agent skills: %w", err)
	}

	return string(data), nil
}

func (t *ListAgentSkillsTool) Bind(ctx context.Context, state *map[string]any) error {
	store := (*state)["store"].(*db.Store)
	if store != nil {
		t.store = store
		return nil
	} else {
		return fmt.Errorf("bind failed: store is nil")
	}
}
