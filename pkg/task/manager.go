package task

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type TaskHandler func(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error

type Manager struct {
	store      *db.Store
	handlers   map[string]TaskHandler
	publishFn  func(topic string, payload any)
	numWorkers int
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func NewManager(store *db.Store, numWorkers int, publishFn func(topic string, payload any)) *Manager {
	return &Manager{
		store:      store,
		handlers:   make(map[string]TaskHandler),
		publishFn:  publishFn,
		numWorkers: numWorkers,
		stopChan:   make(chan struct{}),
	}
}

func (m *Manager) RegisterHandler(name string, handler TaskHandler) {
	m.handlers[name] = handler
}

func (m *Manager) Submit(ctx context.Context, name string, payload any, maxRetries int) (string, error) {
	id, _ := gonanoid.New()
	err := m.store.CreateTask(ctx, id, name, payload, maxRetries)
	if err != nil {
		return "", err
	}

	m.notifyTaskUpdate(id, name, db.TaskStatusPending, 0, "Task submitted", time.Now())
	return id, nil
}

func (m *Manager) Start(ctx context.Context) error {
	if err := m.store.RecoverStuckTasks(ctx); err != nil {
		return fmt.Errorf("failed to recover stuck tasks: %w", err)
	}

	for i := 0; i < m.numWorkers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}

	m.wg.Add(1)
	go m.cleanupLoop()

	return nil
}

func (m *Manager) cleanupLoop() {
	defer m.wg.Done()
	log.Printf("Queue cleanup worker started")

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run cleanup once at start
	m.runCleanup()

	for {
		select {
		case <-m.stopChan:
			log.Printf("Queue cleanup worker stopping")
			return
		case <-ticker.C:
			m.runCleanup()
		}
	}
}

func (m *Manager) runCleanup() {
	days := config.Get().System.MaxTaskRetentionDays
	if days <= 0 {
		return
	}
	log.Printf("Running task cleanup (older than %d days)", days)
	affected, err := m.store.CleanupTasks(context.Background(), days)
	if err != nil {
		log.Printf("Task cleanup failed: %v", err)
	} else if affected > 0 {
		log.Printf("Task cleanup completed: removed %d tasks", affected)
	}
}

func (m *Manager) StartWorkers(ctx context.Context, isDev bool) error {
	numWorkers := m.numWorkers
	if isDev {
		numWorkers = 1
	} else if numWorkers < 1 {
		numWorkers = 1
	}

	m.numWorkers = numWorkers
	return m.Start(ctx)
}

func (m *Manager) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

func (m *Manager) worker(id int) {
	defer m.wg.Done()
	log.Printf("Queue worker %d started", id)

	for {
		select {
		case <-m.stopChan:
			log.Printf("Queue worker %d stopping", id)
			return
		default:
			task, err := m.store.ClaimNextTask(context.Background())
			if err != nil {
				log.Printf("Worker %d: failed to claim task: %v", id, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if task == nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			m.runTask(task)
		}
	}
}

func (m *Manager) runTask(task *db.Task) {
	handler, ok := m.handlers[task.Name]
	if !ok {
		errStr := fmt.Sprintf("no handler registered for task: %s", task.Name)
		log.Println(errStr)
		m.store.MarkTaskFailed(context.Background(), task.ID, errStr, false)
		m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusFailed, task.Progress, errStr, time.Now())
		return
	}

	m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusRunning, task.Progress, "Task started", time.Now())

	lastNotifyTime := time.Now()
	lastProgress := task.Progress

	updateProgress := func(progress int, message string) {
		now := time.Now()
		// Throttling: only update DB and Notify if progress changed or enough time passed
		if progress != lastProgress || now.Sub(lastNotifyTime) > 500*time.Millisecond {
			m.store.UpdateTaskProgress(context.Background(), task.ID, progress, message)
			m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusRunning, progress, message, now)
			lastNotifyTime = now
			lastProgress = progress
		}
	}

	err := handler(context.Background(), task, updateProgress)
	if err != nil {
		log.Printf("Task %s (%s) failed: %v", task.Name, task.ID, err)
		retry := task.Retries < task.MaxRetries
		m.store.MarkTaskFailed(context.Background(), task.ID, err.Error(), retry)

		status := db.TaskStatusFailed
		if retry {
			status = db.TaskStatusPending
		}
		m.notifyTaskUpdate(task.ID, task.Name, status, task.Progress, err.Error(), time.Now())
		return
	}

	log.Printf("Task %s (%s) completed", task.Name, task.ID)
	m.store.MarkTaskCompleted(context.Background(), task.ID)
	m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusCompleted, 100, "Task completed", time.Now())
}

func (m *Manager) notifyTaskUpdate(id, name string, status db.TaskStatus, progress int, message string, timestamp time.Time) {
	if m.publishFn == nil {
		return
	}

	payload := map[string]any{
		"taskId":    id,
		"name":      name,
		"status":    status,
		"progress":  progress,
		"message":   message,
		"timestamp": timestamp.UnixMilli(),
	}
	m.publishFn("tasks", payload)
}
