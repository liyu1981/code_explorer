package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type KnowledgePage struct {
	ID         string    `json:"id"`
	CodebaseID string    `json:"codebase_id"`
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Store) CreateKnowledgePage(ctx context.Context, page *KnowledgePage) error {
	if page.ID == "" {
		id, err := gonanoid.New()
		if err != nil {
			return fmt.Errorf("failed to generate nanoid: %w", err)
		}
		page.ID = id
	}

	now := time.Now()
	_, err := s.ExecWrite(ctx, `
		INSERT INTO knowledge_pages (id, codebase_id, slug, title, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, page.ID, page.CodebaseID, page.Slug, page.Title, page.Content, now, now)

	if err != nil {
		return fmt.Errorf("failed to create knowledge page: %w", err)
	}

	page.CreatedAt = now
	page.UpdatedAt = now
	return nil
}

func (s *Store) GetKnowledgePageBySlug(ctx context.Context, codebaseID, slug string) (*KnowledgePage, error) {
	var p KnowledgePage
	err := s.db.QueryRowContext(ctx, `
		SELECT id, codebase_id, slug, title, content, created_at, updated_at
		FROM knowledge_pages
		WHERE codebase_id = ? AND slug = ?
	`, codebaseID, slug).Scan(&p.ID, &p.CodebaseID, &p.Slug, &p.Title, &p.Content, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge page: %w", err)
	}

	return &p, nil
}

func (s *Store) ListKnowledgePages(ctx context.Context, codebaseID string) ([]KnowledgePage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, codebase_id, slug, title, content, created_at, updated_at
		FROM knowledge_pages
		WHERE codebase_id = ?
		ORDER BY created_at DESC
	`, codebaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge pages: %w", err)
	}
	defer rows.Close()

	var pages []KnowledgePage
	for rows.Next() {
		var p KnowledgePage
		if err := rows.Scan(&p.ID, &p.CodebaseID, &p.Slug, &p.Title, &p.Content, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan knowledge page: %w", err)
		}
		pages = append(pages, p)
	}

	return pages, nil
}

func (s *Store) UpdateKnowledgePage(ctx context.Context, page *KnowledgePage) error {
	now := time.Now()
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE knowledge_pages
			SET title = ?, content = ?, slug = ?, updated_at = ?
			WHERE id = ?
		`, page.Title, page.Content, page.Slug, now, page.ID)

		if err != nil {
			return err
		}

		affected, _ := res.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("knowledge page not found")
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update knowledge page: %w", err)
	}

	page.UpdatedAt = now
	return nil
}

func (s *Store) DeleteKnowledgePage(ctx context.Context, id string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM knowledge_pages WHERE id = ?", id)
	return err
}
