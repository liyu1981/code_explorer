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
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Tags         string    `json:"tags"`
	IsBuiltin    bool      `json:"is_builtin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Store) CreateSkill(ctx context.Context, skill *Skill) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	if skill.ID == "" {
		id, err := gonanoid.New()
		if err != nil {
			return fmt.Errorf("failed to generate nanoid: %w", err)
		}
		skill.ID = id
	}

	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_skills (id, name, description, system_prompt, tags, is_builtin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, skill.ID, skill.Name, skill.Description, skill.SystemPrompt, skill.Tags, skill.IsBuiltin, now, now)

	if err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	skill.CreatedAt = now
	skill.UpdatedAt = now
	return nil
}

func (s *Store) GetSkillByName(ctx context.Context, name string) (*Skill, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	var sk Skill
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, system_prompt, tags, is_builtin, created_at, updated_at
		FROM agent_skills
		WHERE name = ?
	`, name).Scan(&sk.ID, &sk.Name, &sk.Description, &sk.SystemPrompt, &sk.Tags, &sk.IsBuiltin, &sk.CreatedAt, &sk.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	return &sk, nil
}

func (s *Store) ListAgentSkills(ctx context.Context) ([]Skill, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, system_prompt, tags, is_builtin, created_at, updated_at
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
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.SystemPrompt, &sk.Tags, &sk.IsBuiltin, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, sk)
	}

	return skills, nil
}

func (s *Store) UpdateSkill(ctx context.Context, skill *Skill) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
		UPDATE agent_skills
		SET description = ?, system_prompt = ?, tags = ?, updated_at = ?
		WHERE id = ?
	`, skill.Description, skill.SystemPrompt, skill.Tags, now, skill.ID)

	if err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("skill not found")
	}

	skill.UpdatedAt = now
	return nil
}

func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, "DELETE FROM agent_skills WHERE id = ?", id)
	return err
}
