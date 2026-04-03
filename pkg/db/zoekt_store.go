package db

import (
	"context"
	"database/sql"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ZoektMetadata struct {
	ID         string `json:"id"`
	CodebaseID string `json:"codebaseId"`
	IndexedAt  int64  `json:"indexedAt"`
}

func (s *Store) ZoektEnsureMetadata(ctx context.Context, codebaseID string) (string, error) {
	var metadataID string
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"SELECT id FROM zoekt_codebases WHERE codebase_id = ?",
			codebaseID,
		).Scan(&metadataID)

		if err == nil {
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}

		newID, _ := gonanoid.New()
		_, err = tx.ExecContext(ctx,
			"INSERT INTO zoekt_codebases (id, codebase_id, indexed_at) VALUES (?, ?, unixepoch())",
			newID, codebaseID,
		)
		if err != nil {
			return err
		}
		metadataID = newID
		return nil
	})

	if err != nil {
		return "", err
	}

	return metadataID, nil
}

func (s *Store) ZoektGetFileHash(ctx context.Context, metadataID string, filePath string) (string, error) {
	var fileHash string
	err := s.db.QueryRowContext(ctx,
		"SELECT file_hash FROM zoekt_indexed_files WHERE codebase_id = ? AND file_path = ?",
		metadataID, filePath,
	).Scan(&fileHash)

	if err == sql.ErrNoRows {
		return "", nil
	}
	return fileHash, err
}

func (s *Store) ZoektUpsertFileHash(ctx context.Context, metadataID string, filePath string, fileHash string) error {
	newID, _ := gonanoid.New()
	_, err := s.ExecWrite(ctx, `
		INSERT INTO zoekt_indexed_files (id, codebase_id, file_path, file_hash, indexed_at)
		VALUES (?, ?, ?, ?, unixepoch())
		ON CONFLICT(codebase_id, file_path) DO UPDATE SET
			file_hash = excluded.file_hash,
			indexed_at = excluded.indexed_at
	`, newID, metadataID, filePath, fileHash)
	return err
}

func (s *Store) ZoektRemoveStaleFiles(ctx context.Context, metadataID string, activeFiles []string) (int, error) {
	var removedCount int
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			"SELECT file_path FROM zoekt_indexed_files WHERE codebase_id = ?",
			metadataID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		var allFiles []string
		for rows.Next() {
			var f string
			if err := rows.Scan(&f); err != nil {
				return err
			}
			allFiles = append(allFiles, f)
		}

		activeSet := make(map[string]bool)
		for _, f := range activeFiles {
			activeSet[f] = true
		}

		var staleFiles []string
		for _, f := range allFiles {
			if !activeSet[f] {
				staleFiles = append(staleFiles, f)
			}
		}

		if len(staleFiles) == 0 {
			removedCount = 0
			return nil
		}

		for _, filePath := range staleFiles {
			_, err = tx.ExecContext(ctx,
				"DELETE FROM zoekt_indexed_files WHERE codebase_id = ? AND file_path = ?",
				metadataID, filePath,
			)
			if err != nil {
				return err
			}
		}

		removedCount = len(staleFiles)
		return nil
	})

	return removedCount, err
}

func (s *Store) ZoektTouchCodebase(ctx context.Context, metadataID string) error {
	_, err := s.ExecWrite(ctx, "UPDATE zoekt_codebases SET indexed_at = unixepoch() WHERE id = ?", metadataID)
	return err
}

func (s *Store) ZoektGetMetadataByCodebase(ctx context.Context, codebaseID string) (*ZoektMetadata, error) {
	var m ZoektMetadata
	err := s.db.QueryRowContext(ctx,
		"SELECT id, codebase_id, indexed_at FROM zoekt_codebases WHERE codebase_id = ?",
		codebaseID,
	).Scan(&m.ID, &m.CodebaseID, &m.IndexedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) ZoektListFiles(ctx context.Context, metadataID string) ([]FileInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT file_path, file_hash, indexed_at FROM zoekt_indexed_files WHERE codebase_id = ? ORDER BY file_path",
		metadataID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileInfo
	for rows.Next() {
		var f FileInfo
		if err := rows.Scan(&f.FilePath, &f.FileHash, &f.IndexedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, rows.Err()
}
