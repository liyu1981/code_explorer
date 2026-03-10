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

func (s *Store) CreateTask(ctx context.Context, id, name string, payload any, maxRetries int) error {
	if err := s.reconnect(); err != nil {
		return err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, name, payload, status, max_retries)
		VALUES (?, ?, ?, ?, ?)
	`, id, name, string(payloadJSON), TaskStatusPending, maxRetries)
	return err
}

func (s *Store) ClaimNextTask(ctx context.Context) (*Task, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var t Task
	err = tx.QueryRowContext(ctx, `
		SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at
		FROM tasks
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks SET status = 'running', updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, t.ID)
	if err != nil {
		return nil, err
	}

	t.Status = TaskStatusRunning
	return &t, tx.Commit()
}

func (s *Store) UpdateTaskProgress(ctx context.Context, id string, progress int, message string) error {
	if err := s.reconnect(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET progress = ?, message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, progress, message, id)
	return err
}

func (s *Store) MarkTaskCompleted(ctx context.Context, id string) error {
	if err := s.reconnect(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, progress = 100, updated_at = CURRENT_TIMESTAMP, completed_at = CURRENT_TIMESTAMP WHERE id = ?
	`, TaskStatusCompleted, id)
	return err
}

func (s *Store) MarkTaskFailed(ctx context.Context, id string, errStr string, retry bool) error {
	if err := s.reconnect(); err != nil {
		return err
	}
	if retry {
		_, err := s.db.ExecContext(ctx, `
			UPDATE tasks SET status = ?, retries = retries + 1, error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, TaskStatusPending, errStr, id)
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, error = ?, updated_at = CURRENT_TIMESTAMP, completed_at = CURRENT_TIMESTAMP WHERE id = ?
	`, TaskStatusFailed, errStr, id)
	return err
}

func (s *Store) RecoverStuckTasks(ctx context.Context) error {
	if err := s.reconnect(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'pending' WHERE status = 'running'
	`)
	return err
}

func (s *Store) CleanupTasks(ctx context.Context, days int) (int64, error) {
	if err := s.reconnect(); err != nil {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM tasks WHERE created_at < datetime('now', '-' || ? || ' days')
	`, days)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) GetTasks(ctx context.Context, limit, offset int) ([]Task, int, error) {
	if err := s.reconnect(); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, payload, status, progress, message, retries, max_retries, error, created_at, updated_at, completed_at
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
		if err := rows.Scan(&t.ID, &t.Name, &t.Payload, &t.Status, &t.Progress, &t.Message, &t.Retries, &t.MaxRetries, &t.Error, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}

	var total int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks").Scan(&total)
	return tasks, total, err
}
