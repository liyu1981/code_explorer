package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Payload     string         `json:"payload"`
	InitiatorID sql.NullString `json:"initiator_id"`
	Status      TaskStatus     `json:"status"`
	Progress    int            `json:"progress"`
	Message     sql.NullString `json:"message"`
	Retries     int            `json:"retries"`
	MaxRetries  int            `json:"max_retries"`
	Error       sql.NullString `json:"error"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt sql.NullTime   `json:"completed_at"`
}

func (s *Store) CreateTask(ctx context.Context, id, name string, payload any, maxRetries int, initiatorID string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var initiator sql.NullString
	if initiatorID != "" {
		initiator = sql.NullString{String: initiatorID, Valid: true}
	}

	_, err = s.ExecWrite(ctx, `
		INSERT INTO tasks (id, name, payload, status, max_retries, initiator_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, name, string(payloadJSON), TaskStatusPending, maxRetries, initiator)
	return err
}

func (s *Store) GetTasks(ctx context.Context, limit, offset int) ([]Task, int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at, initiator_id
		FROM tasks
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt, &t.InitiatorID); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}

	var total int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks").Scan(&total)
	return tasks, total, err
}

func (s *Store) GetTask(ctx context.Context, id string) (*Task, error) {
	var t Task
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at, initiator_id
		FROM tasks
		WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt, &t.InitiatorID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// GetTaskTree returns a task and all its descendants (recursive).
func (s *Store) GetTaskTree(ctx context.Context, rootID string) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE task_tree AS (
			-- Anchor: start with the root task
			SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at, initiator_id
			FROM tasks
			WHERE id = ?
			
			UNION ALL
			
			-- Recursive member: find tasks initiated by tasks already in task_tree
			SELECT t.id, t.name, t.payload, t.status, t.progress, t.message, t.retries, t.max_retries, t.error, t.created_at, t.updated_at, t.completed_at, t.initiator_id
			FROM tasks t
			INNER JOIN task_tree tt ON t.initiator_id = tt.id
		)
		SELECT * FROM task_tree
		ORDER BY created_at ASC
	`, rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task tree: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt, &t.InitiatorID); err != nil {
			return nil, fmt.Errorf("failed to scan task tree row: %w", err)
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}

// Internal CRUD methods for task manager

func (s *Store) UpdateTaskStatusTx(ctx context.Context, tx *sql.Tx, id string, status TaskStatus) error {
	_, err := tx.ExecContext(ctx, "UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, id)
	return err
}

func (s *Store) UpdateTaskStatus(ctx context.Context, id string, status TaskStatus) error {
	_, err := s.ExecWrite(ctx, "UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, id)
	return err
}

func (s *Store) UpdateTaskProgress(ctx context.Context, id string, progress int, message string) error {
	_, err := s.ExecWrite(ctx, `
		UPDATE tasks SET progress = ?, message = COALESCE(message, '') || ? || char(10), updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, progress, message, id)
	return err
}

func (s *Store) MarkTaskCompleted(ctx context.Context, id string, message string) error {
	_, err := s.ExecWrite(ctx, `
		UPDATE tasks SET status = ?, progress = 100, message = COALESCE(message, '') || ? || char(10), updated_at = CURRENT_TIMESTAMP, completed_at = CURRENT_TIMESTAMP WHERE id = ?
	`, TaskStatusCompleted, message, id)
	return err
}

func (s *Store) MarkTaskFailed(ctx context.Context, id string, errStr string, retry bool) error {
	if retry {
		_, err := s.ExecWrite(ctx, `
			UPDATE tasks SET status = ?, retries = retries + 1, error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, TaskStatusPending, errStr, id)
		return err
	}

	_, err := s.ExecWrite(ctx, `
		UPDATE tasks SET status = ?, error = ?, updated_at = CURRENT_TIMESTAMP, completed_at = CURRENT_TIMESTAMP WHERE id = ?
	`, TaskStatusFailed, errStr, id)
	return err
}

func (s *Store) RecoverStuckTasks(ctx context.Context) error {
	_, err := s.ExecWrite(ctx, `
		UPDATE tasks SET status = 'pending' WHERE status = 'running'
	`)
	return err
}

func (s *Store) CleanupTasks(ctx context.Context, days int) (int64, error) {
	res, err := s.ExecWrite(ctx, `
		DELETE FROM tasks WHERE created_at < datetime('now', '-' || ? || ' days')
	`, days)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) GetNextPendingTaskTx(ctx context.Context, tx *sql.Tx) (*Task, error) {
	var t Task
	err := tx.QueryRowContext(ctx, `
		SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at, initiator_id
		FROM tasks
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt, &t.InitiatorID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (s *Store) GetNextPendingTask(ctx context.Context) (*Task, error) {
	var t Task
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at, initiator_id
		FROM tasks
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt, &t.InitiatorID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}
