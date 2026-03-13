package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) CodemoggerEnsureMetadata(ctx context.Context, codebaseID string) (string, error) {
	var metadataID string
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"SELECT id FROM codemogger_codebases WHERE codebase_id = ?",
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
			"INSERT INTO codemogger_codebases (id, codebase_id, indexed_at) VALUES (?, ?, unixepoch())",
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

func (s *Store) CodemoggerListCodebases(ctx context.Context) ([]CodebaseInfo, error) {
	query := `
		SELECT 
			c.id,
			c.root_path,
			c.name,
			c.type,
			c.version,
			COALESCE(mc.indexed_at, 0) as indexed_at,
			COUNT(DISTINCT f.file_path) as file_count,
			COALESCE(SUM(f.chunk_count), 0) as chunk_count
		FROM codebases c
		LEFT JOIN codemogger_codebases mc ON mc.codebase_id = c.id
		LEFT JOIN codemogger_indexed_files f ON f.codebase_id = mc.id
		GROUP BY c.id
		ORDER BY c.root_path
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CodebaseInfo
	for rows.Next() {
		var r CodebaseInfo
		if err := rows.Scan(&r.ID, &r.RootPath, &r.Name, &r.Type, &r.Version, &r.IndexedAt, &r.FileCount, &r.ChunkCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) CodemoggerTouchCodebase(ctx context.Context, metadataID string) error {
	_, err := s.ExecWrite(ctx, "UPDATE codemogger_codebases SET indexed_at = unixepoch() WHERE id = ?", metadataID)
	return err
}

func (s *Store) CodemoggerGetFileHash(ctx context.Context, metadataID string, filePath string) (string, error) {
	var fileHash string
	err := s.db.QueryRowContext(ctx,
		"SELECT file_hash FROM codemogger_indexed_files WHERE codebase_id = ? AND file_path = ?",
		metadataID, filePath,
	).Scan(&fileHash)

	if err == sql.ErrNoRows {
		return "", nil
	}
	return fileHash, err
}

func (s *Store) CodemoggerBatchUpsertAllFileChunks(ctx context.Context, metadataID string, fileChunks []struct {
	FilePath string
	FileHash string
	Chunks   []CodeChunk
}) error {
	return s.Transaction(ctx, func(tx *sql.Tx) error {
		for _, fc := range fileChunks {
			_, err := tx.ExecContext(ctx,
				"DELETE FROM codemogger_chunks WHERE codebase_id = ? AND file_path = ?",
				metadataID, fc.FilePath,
			)
			if err != nil {
				return err
			}

			for _, chunk := range fc.Chunks {
				newChunkID, _ := gonanoid.New()
				_, err = tx.ExecContext(ctx, `
					INSERT INTO codemogger_chunks 
					(id, codebase_id, file_path, chunk_key, language, kind, name, signature, snippet, start_line, end_line, file_hash, indexed_at, embedding, embedding_model)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), NULL, '')
				`,
					newChunkID, metadataID, chunk.FilePath, chunk.ChunkKey, chunk.Language, chunk.Kind,
					chunk.Name, chunk.Signature, chunk.Snippet, chunk.StartLine, chunk.EndLine, chunk.FileHash,
				)
				if err != nil {
					return err
				}
			}

			newFileID, _ := gonanoid.New()
			_, err = tx.ExecContext(ctx, `
				INSERT INTO codemogger_indexed_files (id, codebase_id, file_path, file_hash, chunk_count, indexed_at)
				VALUES (?, ?, ?, ?, ?, unixepoch())
				ON CONFLICT(codebase_id, file_path) DO UPDATE SET
					file_hash = excluded.file_hash,
					chunk_count = excluded.chunk_count,
					indexed_at = excluded.indexed_at
			`,
				newFileID, metadataID, fc.FilePath, fc.FileHash, len(fc.Chunks),
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) CodemoggerRemoveStaleFiles(ctx context.Context, metadataID string, activeFiles []string) (int, error) {
	var removedCount int
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			"SELECT file_path FROM codemogger_indexed_files WHERE codebase_id = ?",
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
				"DELETE FROM codemogger_chunks WHERE codebase_id = ? AND file_path = ?",
				metadataID, filePath,
			)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx,
				"DELETE FROM codemogger_indexed_files WHERE codebase_id = ? AND file_path = ?",
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

func (s *Store) CodemoggerBatchUpsertEmbeddings(ctx context.Context, items []struct {
	ChunkKey  string
	Embedding []float32
	ModelName string
}) error {
	if len(items) == 0 {
		return nil
	}

	return s.Transaction(ctx, func(tx *sql.Tx) error {
		for _, item := range items {
			blob, err := json.Marshal(item.Embedding)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx,
				"UPDATE codemogger_chunks SET embedding = vector32(?), embedding_model = ? WHERE chunk_key = ?",
				string(blob), item.ModelName, item.ChunkKey,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) CodemoggerGetStaleEmbeddings(ctx context.Context, metadataID string, modelName string, limit int) ([]struct {
	ChunkKey  string
	Name      string
	Signature string
	FilePath  string
	Kind      string
	Snippet   string
}, error) {
	query := `
		SELECT chunk_key, name, signature, file_path, kind, snippet
		FROM codemogger_chunks
		WHERE codebase_id = ? AND (embedding IS NULL OR embedding_model != ?)
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, metadataID, modelName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		ChunkKey  string
		Name      string
		Signature string
		FilePath  string
		Kind      string
		Snippet   string
	}

	for rows.Next() {
		var r struct {
			ChunkKey  string
			Name      string
			Signature string
			FilePath  string
			Kind      string
			Snippet   string
		}
		if err := rows.Scan(&r.ChunkKey, &r.Name, &r.Signature, &r.FilePath, &r.Kind, &r.Snippet); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) CodemoggerRebuildFTSTable(ctx context.Context, metadataID string) error {
	return nil
}

func (s *Store) CodemoggerVectorSearch(ctx context.Context, queryEmbedding []float32, limit int, includeSnippet bool) ([]SearchResult, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("empty query embedding")
	}

	queryVec := fmt.Sprintf("[%v]", strings.Join(func() []string {
		var s []string
		for _, v := range queryEmbedding {
			s = append(s, fmt.Sprintf("%v", v))
		}
		return s
	}(), ","))

	sqlQuery := `
		SELECT chunk_key, file_path, name, kind, signature, snippet, start_line, end_line,
			   vector_distance_cos(embedding, vector32(?)) as distance
		FROM codemogger_chunks
		WHERE embedding IS NOT NULL
		ORDER BY distance ASC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, queryVec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var snippetRaw sql.NullString
		err := rows.Scan(&r.ChunkKey, &r.FilePath, &r.Name, &r.Kind, &r.Signature,
			&snippetRaw, &r.StartLine, &r.EndLine, &r.Score)
		if err != nil {
			return nil, err
		}
		if includeSnippet && snippetRaw.Valid {
			r.Snippet = snippetRaw.String
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) CodemoggerFTSSearch(ctx context.Context, query string, limit int, includeSnippet bool) ([]SearchResult, error) {
	sqlQuery := `
		SELECT chunk_key, file_path, codemogger_chunks.name, kind, codemogger_chunks.signature, codemogger_chunks.snippet, start_line, end_line
		FROM codemogger_chunks
		JOIN codemogger_chunks_fts ON codemogger_chunks_fts.rowid = codemogger_chunks.rowid
		WHERE codemogger_chunks_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var snippetRaw sql.NullString
		err := rows.Scan(&r.ChunkKey, &r.FilePath, &r.Name, &r.Kind, &r.Signature,
			&snippetRaw, &r.StartLine, &r.EndLine)
		if err != nil {
			return nil, err
		}
		r.Score = 1.0
		if includeSnippet && snippetRaw.Valid {
			r.Snippet = snippetRaw.String
		} else if !includeSnippet {
			r.Snippet = ""
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) CodemoggerListFiles(ctx context.Context, metadataID string) ([]FileInfo, error) {
	var rows *sql.Rows
	var err error

	if metadataID != "" {
		rows, err = s.db.QueryContext(ctx,
			"SELECT file_path, file_hash, chunk_count, indexed_at FROM codemogger_indexed_files WHERE codebase_id = ? ORDER BY file_path",
			metadataID,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			"SELECT file_path, file_hash, chunk_count, indexed_at FROM codemogger_indexed_files ORDER BY file_path",
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileInfo
	for rows.Next() {
		var f FileInfo
		if err := rows.Scan(&f.FilePath, &f.FileHash, &f.ChunkCount, &f.IndexedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

func (s *Store) CodemoggerGetMetadataByCodebase(ctx context.Context, codebaseID string) (*CodemoggerMetadata, error) {
	var m CodemoggerMetadata
	err := s.db.QueryRowContext(ctx,
		"SELECT id, codebase_id, indexed_at FROM codemogger_codebases WHERE codebase_id = ?",
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
