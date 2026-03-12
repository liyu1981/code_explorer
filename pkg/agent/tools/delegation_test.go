package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func TestDelegationTools(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	stream := &mockStreamWriter{}

	queueTool := NewQueueTaskTool(store)
	pollTool := NewPollTasksTool(store)
	readTool := NewReadTaskOutputTool(store)

	taskID := "task-1"
	taskName := "test-task"
	taskPayload := map[string]string{"foo": "bar"}

	t.Run("QueueTaskTool", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{
			"id":      taskID,
			"name":    taskName,
			"payload": taskPayload,
		})
		res, err := queueTool.Execute(ctx, input, stream)
		if err != nil {
			t.Fatalf("QueueTaskTool failed: %v", err)
		}
		if res != taskID {
			t.Errorf("Expected task ID %s, got %s", taskID, res)
		}

		// Verify task was created
		task, err := store.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}
		if task == nil {
			t.Fatal("Task not found in DB")
		}
		if task.Name != taskName {
			t.Errorf("Expected name %s, got %s", taskName, task.Name)
		}
	})

	t.Run("PollTasksTool", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{
			"ids": []string{taskID, "non-existent"},
		})
		res, err := pollTool.Execute(ctx, input, stream)
		if err != nil {
			t.Fatalf("PollTasksTool failed: %v", err)
		}

		var results map[string]string
		if err := json.Unmarshal([]byte(res), &results); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if results[taskID] != "pending" {
			t.Errorf("Expected status pending for %s, got %s", taskID, results[taskID])
		}
		if results["non-existent"] != "not_found" {
			t.Errorf("Expected status not_found for non-existent, got %s", results["non-existent"])
		}
	})

	t.Run("ReadTaskOutputTool", func(t *testing.T) {
		// Mock task update
		err := store.MarkTaskCompleted(ctx, taskID, "Task finished successfully")
		if err != nil {
			t.Fatalf("MarkTaskCompleted failed: %v", err)
		}

		input, _ := json.Marshal(map[string]any{"id": taskID})
		res, err := readTool.Execute(ctx, input, stream)
		if err != nil {
			t.Fatalf("ReadTaskOutputTool failed: %v", err)
		}

		var out map[string]string
		if err := json.Unmarshal([]byte(res), &out); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if out["status"] != string(db.TaskStatusCompleted) {
			t.Errorf("Expected status completed, got %s", out["status"])
		}
		if out["message"] != "Task finished successfully" {
			t.Errorf("Expected message, got %s", out["message"])
		}
	})

	t.Run("ReadTaskOutputTool Not Found", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{"id": "not-existent"})
		res, err := readTool.Execute(ctx, input, stream)
		if err != nil {
			t.Fatalf("ReadTaskOutputTool failed: %v", err)
		}
		if !strings.Contains(res, "Task not found") {
			t.Errorf("Expected 'Task not found', got %s", res)
		}
	})
}
