//go:build libsql
// +build libsql

package task

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func setupTestStore(t *testing.T) (*db.Store, func()) {
	// For testing, unique filenames in a temp dir are more reliable for migrations than :memory:
	// because some migration logic might expect a physical file or multiple connections to same memory DB is tricky.
	dir, err := os.MkdirTemp("", "queue-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "fresh-test.db")
	sqlDb, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("open db: %v", err)
	}

	// Enable WAL mode for better concurrency in tests
	_, _ = sqlDb.Exec("PRAGMA journal_mode=WAL")

	store := db.NewStore(sqlDb, dbPath)
	return store, func() {
		sqlDb.Close()
		os.RemoveAll(dir)
	}
}

func TestManager_SubmitAndRun(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	var mu sync.Mutex
	var events []map[string]any
	publishFn := func(topic string, payload any) {
		if topic == "tasks" {
			mu.Lock()
			events = append(events, payload.(map[string]any))
			mu.Unlock()
		}
	}

	manager := NewManager(store, 1, publishFn)

	taskCompleted := make(chan string, 1)
	manager.RegisterHandler("test-task", func(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
		updateProgress(50, "halfway there")
		time.Sleep(50 * time.Millisecond)
		updateProgress(100, "done")
		taskCompleted <- task.ID
		return nil
	})

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("failed to start manager: %v", err)
	}
	defer manager.Stop()

	payload := map[string]string{"foo": "bar"}
	id, err := manager.Submit(context.Background(), "test-task", payload, 3)
	if err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	if id == "" {
		t.Fatal("expected non-empty task id")
	}

	// Wait for completion
	select {
	case completedID := <-taskCompleted:
		if completedID != id {
			t.Errorf("expected completed task id %s, got %s", id, completedID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for task completion")
	}

	// Give a bit of time for the completed event to be published
	time.Sleep(200 * time.Millisecond)

	// Verify events
	mu.Lock()
	defer mu.Unlock()

	hasPending := false
	hasRunning := false
	hasCompleted := false
	for _, ev := range events {
		if ev["taskId"] == id {
			if ev["status"] == db.TaskStatusPending {
				hasPending = true
			}
			if ev["status"] == db.TaskStatusRunning {
				hasRunning = true
			}
			if ev["status"] == db.TaskStatusCompleted {
				hasCompleted = true
			}
		}
	}

	if !hasPending {
		t.Error("missing pending event")
	}
	if !hasRunning {
		t.Error("missing running event")
	}
	if !hasCompleted {
		t.Error("missing completed event")
	}
}

func TestManager_TaskFailureAndRetry(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	manager := NewManager(store, 1, nil)

	var mu sync.Mutex
	attempts := 0
	done := make(chan struct{})

	manager.RegisterHandler("fail-task", func(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
		mu.Lock()
		attempts++
		currentAttempts := attempts
		mu.Unlock()

		if currentAttempts <= 2 {
			return context.DeadlineExceeded // Simulate failure
		}
		close(done)
		return nil
	})

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("failed to start manager: %v", err)
	}
	defer manager.Stop()

	_, err := manager.Submit(context.Background(), "fail-task", nil, 3)
	if err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	// Wait for completion after retries
	select {
	case <-done:
		mu.Lock()
		finalAttempts := attempts
		mu.Unlock()
		if finalAttempts != 3 {
			t.Errorf("expected 3 attempts, got %d", finalAttempts)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for task completion after retries")
	}
}
