package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Skill struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SystemPrompt string    `json:"system_prompt"`
	Tags         string    `json:"tags"`
	Tools        string    `json:"tools"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	IsBuiltin    bool      `json:"is_builtin"`
}

func (s *Store) CreateSkill(ctx context.Context, skill *Skill) error {
	if skill.ID == "" {
		id, err := gonanoid.New()
		if err != nil {
			return fmt.Errorf("failed to generate nanoid: %w", err)
		}
		skill.ID = id
	}

	now := time.Now()
	_, err := s.ExecWrite(ctx, `
		INSERT INTO agent_skills (id, name, system_prompt, tags, tools, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, skill.ID, skill.Name, skill.SystemPrompt, skill.Tags, skill.Tools, now, now)

	if err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	skill.CreatedAt = now
	skill.UpdatedAt = now
	return nil
}

func (s *Store) GetSkillByName(ctx context.Context, name string) (*Skill, error) {
	var sk Skill
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, system_prompt, tags, tools, created_at, updated_at
		FROM agent_skills
		WHERE name = ?
	`, name).Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	return &sk, nil
}

func (s *Store) GetSkillByID(ctx context.Context, id string) (*Skill, error) {
	var sk Skill
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, system_prompt, tags, tools, created_at, updated_at
		FROM agent_skills
		WHERE id = ?
	`, id).Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	return &sk, nil
}

func (s *Store) ListAgentSkills(ctx context.Context) ([]Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, system_prompt, tags, tools, created_at, updated_at
		FROM agent_skills
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, sk)
	}

	return skills, nil
}

func (s *Store) ListSkillsByNamePrefix(ctx context.Context, prefix string) ([]Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, system_prompt, tags, tools, created_at, updated_at
		FROM agent_skills
		WHERE name LIKE ?
		ORDER BY created_at ASC
	`, prefix+"_%")
	if err != nil {
		return nil, fmt.Errorf("failed to list skills by prefix: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.SystemPrompt, &sk.Tags, &sk.Tools, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, sk)
	}

	return skills, nil
}

func (s *Store) UpdateSkill(ctx context.Context, skill *Skill) error {
	now := time.Now()
	_, err := s.ExecWrite(ctx, `
		UPDATE agent_skills
		SET system_prompt = ?, tags = ?, tools = ?, updated_at = ?
		WHERE id = ?
	`, skill.SystemPrompt, skill.Tags, skill.Tools, now, skill.ID)

	if err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	skill.UpdatedAt = now
	return nil
}

func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM agent_skills WHERE id = ?", id)
	return err
}
