package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/util"
)

// QueueTaskTool allows an agent to queue a new background task
type QueueTaskTool struct {
	store *db.Store
}

func NewQueueTaskTool() Tool {
	return &QueueTaskTool{}
}

func (t *QueueTaskTool) Name() string {
	return "queue_task"
}

func (t *QueueTaskTool) Description() string {
	return "Queues a new background task. Returns the task ID."
}

func (t *QueueTaskTool) Clone() Tool {
	return &QueueTaskTool{store: t.store}
}

func (t *QueueTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Unique ID for the task (recommend nanoid)",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Name of the task type (e.g., 'wiki-analyze')",
			},
			"payload": map[string]any{
				"type":        "object",
				"description": "JSON payload for the task",
			},
			"max_retries": map[string]any{
				"type":        "integer",
				"description": "Max retries (default 3)",
			},
		},
		"required": []string{"id", "name", "payload"},
	}
}

func (t *QueueTaskTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	if t.store == nil {
		return "", fmt.Errorf("store is nil")
	}

	var req struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Payload    any    `json:"payload"`
		MaxRetries int    `json:"max_retries"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	if req.MaxRetries <= 0 {
		req.MaxRetries = 3
	}

	initiatorID := util.GetInitiatorID(ctx)

	err := t.store.CreateTask(ctx, req.ID, req.Name, req.Payload, req.MaxRetries, initiatorID)
	if err != nil {
		return "", fmt.Errorf("failed to queue task: %w", err)
	}

	return req.ID, nil
}

func (t *QueueTaskTool) Bind(ctx context.Context, state *map[string]any) error {
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
