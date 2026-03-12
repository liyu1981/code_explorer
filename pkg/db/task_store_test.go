package db

import (
	"context"
	"testing"
)

func TestTaskStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test Create
	taskID := "task-1"
	payload := map[string]string{"foo": "bar"}
	if err := store.CreateTask(ctx, taskID, "test-task", payload, 3); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Test GetTasks
	tasks, total, err := store.GetTasks(ctx, 10, 0)
	if err != nil {
		t.Fatalf("get tasks: %v", err)
	}
	if total != 1 || len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d (total %d)", len(tasks), total)
	}
	if tasks[0].ID != taskID {
		t.Errorf("expected task id %s, got %s", taskID, tasks[0].ID)
	}

	// Test Claim
	claimed, err := store.ClaimNextTask(ctx)
	if err != nil {
		t.Fatalf("claim task: %v", err)
	}
	if claimed == nil || claimed.ID != taskID {
		t.Errorf("expected to claim task %s, got %v", taskID, claimed)
	}
	if claimed.Status != TaskStatusRunning {
		t.Errorf("expected status running, got %s", claimed.Status)
	}

	// Test Update Progress
	if err := store.UpdateTaskProgress(ctx, taskID, 50, "halfway done"); err != nil {
		t.Fatalf("update progress: %v", err)
	}
	tasks, _, _ = store.GetTasks(ctx, 1, 0)
	if tasks[0].Progress != 50 || tasks[0].Message.String != "halfway done" {
		t.Errorf("expected progress 50 and message, got %d, %s", tasks[0].Progress, tasks[0].Message.String)
	}

	// Test Mark Completed
	if err := store.MarkTaskCompleted(ctx, taskID, "Done"); err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	tasks, _, _ = store.GetTasks(ctx, 1, 0)
	if tasks[0].Status != TaskStatusCompleted || tasks[0].Progress != 100 {
		t.Errorf("expected status completed and progress 100, got %s, %d", tasks[0].Status, tasks[0].Progress)
	}

	// Test Mark Failed with retry
	taskID2 := "task-2"
	store.CreateTask(ctx, taskID2, "retry-task", payload, 3)
	store.ClaimNextTask(ctx)
	if err := store.MarkTaskFailed(ctx, taskID2, "error", true); err != nil {
		t.Fatalf("mark failed retry: %v", err)
	}
	tasks, _, _ = store.GetTasks(ctx, 10, 0)
	var t2 *Task
	for i := range tasks {
		if tasks[i].ID == taskID2 {
			t2 = &tasks[i]
		}
	}
	if t2.Status != TaskStatusPending || t2.Retries != 1 {
		t.Errorf("expected status pending and retries 1, got %s, %d", t2.Status, t2.Retries)
	}

	// Test Mark Failed final
	if err := store.MarkTaskFailed(ctx, taskID2, "final error", false); err != nil {
		t.Fatalf("mark failed final: %v", err)
	}
	tasks, _, _ = store.GetTasks(ctx, 10, 0)
	for i := range tasks {
		if tasks[i].ID == taskID2 {
			t2 = &tasks[i]
		}
	}
	if t2.Status != TaskStatusFailed {
		t.Errorf("expected status failed, got %s", t2.Status)
	}
}
