package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Prompt struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	SystemPrompt  string    `json:"system_prompt"`
	UserPromptTpl string    `json:"user_prompt_tpl"`
	Tags          string    `json:"tags"`
	Tools         string    `json:"tools"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsBuiltin     bool      `json:"is_builtin"`
}

func (s *Store) CreatePrompt(ctx context.Context, prompt *Prompt) error {
	if prompt.ID == "" {
		id, err := gonanoid.New()
		if err != nil {
			return fmt.Errorf("failed to generate nanoid: %w", err)
		}
		prompt.ID = id
	}

	now := time.Now()
	_, err := s.ExecWrite(ctx, `
		INSERT INTO agent_prompts (id, name, system_prompt, user_prompt_tpl, tags, tools, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, prompt.ID, prompt.Name, prompt.SystemPrompt, prompt.UserPromptTpl, prompt.Tags, prompt.Tools, now, now)

	if err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	prompt.CreatedAt = now
	prompt.UpdatedAt = now
	return nil
}

func (s *Store) GetPromptByName(ctx context.Context, name string) (*Prompt, error) {
	var sk Prompt
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, system_prompt, user_prompt_tpl, tags, tools, created_at, updated_at
		FROM agent_prompts
		WHERE name = ?
	`, name).Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.UserPromptTpl, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	return &sk, nil
}

func (s *Store) GetPromptByID(ctx context.Context, id string) (*Prompt, error) {
	var sk Prompt
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, system_prompt, user_prompt_tpl, tags, tools, created_at, updated_at
		FROM agent_prompts
		WHERE id = ?
	`, id).Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.UserPromptTpl, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	return &sk, nil
}

func (s *Store) ListAgentPrompts(ctx context.Context) ([]Prompt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, system_prompt, user_prompt_tpl, tags, tools, created_at, updated_at
		FROM agent_prompts
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var sk Prompt
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.UserPromptTpl, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan prompt: %w", err)
		}
		prompts = append(prompts, sk)
	}

	return prompts, nil
}

func (s *Store) ListPromptsByNamePrefix(ctx context.Context, prefix string) ([]Prompt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, system_prompt, user_prompt_tpl, tags, tools, created_at, updated_at
		FROM agent_prompts
		WHERE name LIKE ?
		ORDER BY created_at ASC
	`, prefix+"_%")
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts by prefix: %w", err)
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		var sk Prompt
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.UserPromptTpl, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan prompt: %w", err)
		}
		prompts = append(prompts, sk)
	}

	return prompts, nil
}

func (s *Store) UpdatePrompt(ctx context.Context, prompt *Prompt) error {
	now := time.Now()
	_, err := s.ExecWrite(ctx, `
		UPDATE agent_prompts
		SET system_prompt = ?, user_prompt_tpl = ?, tags = ?, tools = ?, updated_at = ?
		WHERE id = ?
	`, prompt.SystemPrompt, prompt.UserPromptTpl, prompt.Tags, prompt.Tools, now, prompt.ID)

	if err != nil {
		return fmt.Errorf("failed to update prompt: %w", err)
	}

	prompt.UpdatedAt = now
	return nil
}

func (s *Store) DeletePrompt(ctx context.Context, id string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM agent_prompts WHERE id = ?", id)
	return err
}
