package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func TestQueueTaskTool(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	stream := &mockStreamWriter{}

	queueTool := NewQueueTaskTool()
	state := map[string]any{"store": store}
	if err := queueTool.Bind(context.Background(), &state); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	taskID := "task-1"
	taskName := "test-task"
	taskPayload := map[string]string{"foo": "bar"}

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
}
