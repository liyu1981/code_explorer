package db

import (
	"context"
	"database/sql"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) GetOrCreateCodebase(ctx context.Context, rootPath string, name string, codebaseType string) (*Codebase, error) {
	var cb Codebase
	var exists bool
	var newID string

	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"SELECT id, name, root_path, type, version, created_at FROM codebases WHERE root_path = ?",
			rootPath,
		).Scan(&cb.ID, &cb.Name, &cb.RootPath, &cb.Type, &cb.Version, &cb.CreatedAt)

		if err == nil {
			exists = true
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}

		if name == "" {
			name = rootPath
			if idx := len(name) - 1; idx >= 0 && name[idx] == '/' {
				for i := len(name) - 2; i >= 0; i-- {
					if name[i] == '/' {
						name = name[i+1:]
						break
					}
				}
			}
		}

		if codebaseType == "" {
			codebaseType = "local"
		}

		newID, _ = gonanoid.New()
		_, err = tx.ExecContext(ctx,
			"INSERT INTO codebases (id, name, root_path, type, version, created_at) VALUES (?, ?, ?, ?, ?, unixepoch())",
			newID, name, rootPath, codebaseType, "",
		)
		return err
	})

	if err != nil {
		return nil, err
	}

	if exists {
		return &cb, nil
	}

	return &Codebase{
		ID:        newID,
		Name:      name,
		RootPath:  rootPath,
		Type:      codebaseType,
		Version:   "",
		CreatedAt: 0, // Will be set by DB
	}, nil
}

func (s *Store) ListCodebases(ctx context.Context) ([]Codebase, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, root_path, type, version, created_at FROM codebases ORDER BY root_path")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Codebase
	for rows.Next() {
		var cb Codebase
		if err := rows.Scan(&cb.ID, &cb.Name, &cb.RootPath, &cb.Type, &cb.Version, &cb.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, cb)
	}
	return result, nil
}

func (s *Store) GetCodebaseByID(ctx context.Context, id string) (*Codebase, error) {
	var cb Codebase
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, root_path, type, version, created_at FROM codebases WHERE id = ?",
		id,
	).Scan(&cb.ID, &cb.Name, &cb.RootPath, &cb.Type, &cb.Version, &cb.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cb, nil
}

func (s *Store) UpdateCodebaseVersion(ctx context.Context, id string, version string) error {
	_, err := s.ExecWrite(ctx, "UPDATE codebases SET version = ? WHERE id = ?", version, id)
	return err
}

func (s *Store) DeleteCodebase(ctx context.Context, id string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM codebases WHERE id = ?", id)
	return err
}
