package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

// QueueTaskTool allows an agent to queue a new background task
type QueueTaskTool struct {
	store *db.Store
}

func NewQueueTaskTool(store *db.Store) *QueueTaskTool {
	return &QueueTaskTool{store: store}
}

func (t *QueueTaskTool) Name() string {
	return "queue_task"
}

func (t *QueueTaskTool) Description() string {
	return "Queues a new background task. Returns the task ID."
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

	err := t.store.CreateTask(ctx, req.ID, req.Name, req.Payload, req.MaxRetries)
	if err != nil {
		return "", fmt.Errorf("failed to queue task: %w", err)
	}

	return req.ID, nil
}

// PollTasksTool allows an agent to check status of tasks
type PollTasksTool struct {
	store *db.Store
}

func NewPollTasksTool(store *db.Store) *PollTasksTool {
	return &PollTasksTool{store: store}
}

func (t *PollTasksTool) Name() string {
	return "poll_tasks"
}

func (t *PollTasksTool) Description() string {
	return "Checks the status of multiple tasks by their IDs."
}

func (t *PollTasksTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ids": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
		"required": []string{"ids"},
	}
}

func (t *PollTasksTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	results := make(map[string]string)
	for _, id := range req.IDs {
		task, err := t.store.GetTask(ctx, id)
		if err != nil {
			return "", err
		}

		if task != nil {
			results[id] = string(task.Status)
		} else {
			results[id] = "not_found"
		}
	}

	data, _ := json.Marshal(results)
	return string(data), nil
}

// ReadTaskOutputTool allows reading the final message/error of a task
type ReadTaskOutputTool struct {
	store *db.Store
}

func NewReadTaskOutputTool(store *db.Store) *ReadTaskOutputTool {
	return &ReadTaskOutputTool{store: store}
}

func (t *ReadTaskOutputTool) Name() string {
	return "read_task_output"
}

func (t *ReadTaskOutputTool) Description() string {
	return "Reads the final message or error of a completed/failed task."
}

func (t *ReadTaskOutputTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "string"},
		},
		"required": []string{"id"},
	}
}

func (t *ReadTaskOutputTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	task, err := t.store.GetTask(ctx, req.ID)
	if err != nil {
		return "", err
	}

	if task != nil {
		out := map[string]string{
			"status":  string(task.Status),
			"message": task.Message.String,
			"error":   task.Error.String,
		}
		data, _ := json.Marshal(out)
		return string(data), nil
	}

	return "Task not found", nil
}
