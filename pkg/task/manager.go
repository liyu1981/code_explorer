package task

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/rs/zerolog/log"
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
	initiatorID := util.GetInitiatorID(ctx)
	err := m.store.CreateTask(ctx, id, name, payload, maxRetries, initiatorID)
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
	log.Info().Msg("Queue cleanup worker started")

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Initial delay to avoid startup contention
	select {
	case <-m.stopChan:
		return
	case <-time.After(1 * time.Second):
		m.runCleanup()
	}

	for {
		select {
		case <-m.stopChan:
			log.Info().Msg("Queue cleanup worker stopping")
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
	log.Info().Int("days", days).Msg("Running task cleanup")
	affected, err := m.store.CleanupTasks(context.Background(), days)
	if err != nil {
		log.Info().Err(err).Msg("Task cleanup failed")
	} else if affected > 0 {
		log.Info().Int64("affected", affected).Msg("Task cleanup completed")
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
	log.Info().Int("id", id).Msg("Queue worker started")

	for {
		select {
		case <-m.stopChan:
			log.Info().Int("id", id).Msg("Queue worker stopping")
			return
		default:
			var task *db.Task
			err := m.store.Transaction(context.Background(), func(tx *sql.Tx) error {
				t, err := m.store.GetNextPendingTaskTx(context.Background(), tx)
				if err != nil {
					return err
				}
				if t == nil {
					return nil
				}
				if err := m.store.UpdateTaskStatusTx(context.Background(), tx, t.ID, db.TaskStatusRunning); err != nil {
					return err
				}
				t.Status = db.TaskStatusRunning
				task = t
				return nil
			})

			if err != nil {
				log.Info().Int("id", id).Err(err).Msg("Worker failed to claim task")
				select {
				case <-m.stopChan:
					log.Info().Int("id", id).Msg("Queue worker stopping")
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}

			if task == nil {
				select {
				case <-m.stopChan:
					log.Info().Int("id", id).Msg("Queue worker stopping")
					return
				case <-time.After(500 * time.Millisecond):
					continue
				}
			}

			m.runTask(task)
		}
	}
}

func (m *Manager) runTask(task *db.Task) {
	handler, ok := m.handlers[task.Name]
	if !ok {
		errStr := fmt.Sprintf("no handler registered for task: %s", task.Name)
		log.Info().Str("taskName", task.Name).Msg(errStr)
		m.store.MarkTaskFailed(context.Background(), task.ID, errStr, false)
		m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusFailed, task.Progress, errStr, time.Now())
		return
	}

	m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusRunning, task.Progress, "Task started", time.Now())

	ctx := context.Background()
	ctx = util.WithInitiatorID(ctx, task.ID)

	lastNotifyTime := time.Now()
	lastProgress := task.Progress
	lastMessage := ""

	updateProgress := func(progress int, message string) {
		now := time.Now()
		lastMessage = message
		// Throttling: only update DB and Notify if progress changed or enough time passed
		if progress != lastProgress || now.Sub(lastNotifyTime) > 500*time.Millisecond {
			m.store.UpdateTaskProgress(context.Background(), task.ID, progress, message)
			m.notifyTaskUpdate(task.ID, task.Name, db.TaskStatusRunning, progress, message, now)
			lastNotifyTime = now
			lastProgress = progress
		}
	}

	err := handler(ctx, task, updateProgress)
	if err != nil {
		log.Info().Str("name", task.Name).Str("id", task.ID).Err(err).Msg("Task failed")
		retry := task.Retries < task.MaxRetries
		m.store.MarkTaskFailed(context.Background(), task.ID, err.Error(), retry)

		status := db.TaskStatusFailed
		if retry {
			status = db.TaskStatusPending
		}
		m.notifyTaskUpdate(task.ID, task.Name, status, task.Progress, err.Error(), time.Now())
		return
	}

	log.Info().Str("name", task.Name).Str("id", task.ID).Msg("Task completed")
	m.store.MarkTaskCompleted(context.Background(), task.ID, lastMessage)
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

func (m *Manager) GetTask(ctx context.Context, id string) (*db.Task, error) {
	return m.store.GetTask(ctx, id)
}
