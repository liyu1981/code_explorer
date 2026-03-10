package db

import (
	"database/sql"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) GetOrCreateCodebase(rootPath string, name string, codebaseType string) (*Codebase, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	var cb Codebase
	err := s.db.QueryRow(
		"SELECT id, name, root_path, type, version, created_at FROM codebases WHERE root_path = ?",
		rootPath,
	).Scan(&cb.ID, &cb.Name, &cb.RootPath, &cb.Type, &cb.Version, &cb.CreatedAt)

	if err == nil {
		return &cb, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
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

	newID, _ := gonanoid.New()
	_, err = s.db.Exec(
		"INSERT INTO codebases (id, name, root_path, type, version, created_at) VALUES (?, ?, ?, ?, ?, unixepoch())",
		newID, name, rootPath, codebaseType, "",
	)
	if err != nil {
		return nil, err
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

func (s *Store) ListCodebases() ([]Codebase, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	rows, err := s.db.Query("SELECT id, name, root_path, type, version, created_at FROM codebases ORDER BY root_path")
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

func (s *Store) GetCodebaseByID(id string) (*Codebase, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	var cb Codebase
	err := s.db.QueryRow(
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

func (s *Store) UpdateCodebaseVersion(id string, version string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("UPDATE codebases SET version = ? WHERE id = ?", version, id)
	return err
}

func (s *Store) DeleteCodebase(id string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("DELETE FROM codebases WHERE id = ?", id)
	return err
}
