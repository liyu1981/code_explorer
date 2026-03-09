package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) CodemoggerGetOrCreateCodebase(rootPath string, name string) (string, error) {
	if err := s.reconnect(); err != nil {
		return "", err
	}

	var existing Codebase
	err := s.db.QueryRow(
		"SELECT id, root_path, name, indexed_at FROM codemogger_codebases WHERE root_path = ?",
		rootPath,
	).Scan(&existing.ID, &existing.RootPath, &existing.Name, &existing.IndexedAt)

	if err == nil {
		return existing.ID, nil
	}
	if err != sql.ErrNoRows {
		return "", err
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

	newID, _ := gonanoid.New()
	_, err = s.db.Exec(
		"INSERT INTO codemogger_codebases (id, root_path, name, indexed_at) VALUES (?, ?, ?, strftime('%s', 'now'))",
		newID, rootPath, name,
	)
	if err != nil {
		return "", err
	}

	return newID, nil
}

func (s *Store) CodemoggerListCodebases() ([]CodebaseInfo, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT 
			c.id,
			c.root_path,
			c.name,
			c.indexed_at,
			COUNT(DISTINCT f.file_path) as file_count,
			COALESCE(SUM(f.chunk_count), 0) as chunk_count
		FROM codemogger_codebases c
		LEFT JOIN codemogger_indexed_files f ON f.codebase_id = c.id
		GROUP BY c.id
		ORDER BY c.root_path
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CodebaseInfo
	for rows.Next() {
		var r CodebaseInfo
		if err := rows.Scan(&r.ID, &r.RootPath, &r.Name, &r.IndexedAt, &r.FileCount, &r.ChunkCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) CodemoggerTouchCodebase(codebaseID string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("UPDATE codemogger_codebases SET indexed_at = strftime('%s', 'now') WHERE id = ?", codebaseID)
	return err
}

func (s *Store) CodemoggerGetFileHash(codebaseID string, filePath string) (string, error) {
	if err := s.reconnect(); err != nil {
		return "", err
	}

	var fileHash string
	err := s.db.QueryRow(
		"SELECT file_hash FROM codemogger_indexed_files WHERE codebase_id = ? AND file_path = ?",
		codebaseID, filePath,
	).Scan(&fileHash)

	if err == sql.ErrNoRows {
		return "", nil
	}
	return fileHash, err
}

func (s *Store) CodemoggerBatchUpsertAllFileChunks(codebaseID string, fileChunks []struct {
	FilePath string
	FileHash string
	Chunks   []CodeChunk
}) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, fc := range fileChunks {
		_, err = tx.Exec(
			"DELETE FROM codemogger_chunks WHERE codebase_id = ? AND file_path = ?",
			codebaseID, fc.FilePath,
		)
		if err != nil {
			return err
		}

		for _, chunk := range fc.Chunks {
			newChunkID, _ := gonanoid.New()
			_, err = tx.Exec(`
				INSERT INTO codemogger_chunks 
				(id, codebase_id, file_path, chunk_key, language, kind, name, signature, snippet, start_line, end_line, file_hash, indexed_at, embedding, embedding_model)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now'), NULL, '')
			`,
				newChunkID, codebaseID, chunk.FilePath, chunk.ChunkKey, chunk.Language, chunk.Kind,
				chunk.Name, chunk.Signature, chunk.Snippet, chunk.StartLine, chunk.EndLine, chunk.FileHash,
			)
			if err != nil {
				return err
			}
		}

		newFileID, _ := gonanoid.New()
		_, err = tx.Exec(`
			INSERT INTO codemogger_indexed_files (id, codebase_id, file_path, file_hash, chunk_count, indexed_at)
			VALUES (?, ?, ?, ?, ?, strftime('%s', 'now'))
			ON CONFLICT(codebase_id, file_path) DO UPDATE SET
				file_hash = excluded.file_hash,
				chunk_count = excluded.chunk_count,
				indexed_at = excluded.indexed_at
		`,
			newFileID, codebaseID, fc.FilePath, fc.FileHash, len(fc.Chunks),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) CodemoggerRemoveStaleFiles(codebaseID string, activeFiles []string) (int, error) {
	if err := s.reconnect(); err != nil {
		return 0, err
	}

	rows, err := s.db.Query(
		"SELECT file_path FROM codemogger_indexed_files WHERE codebase_id = ?",
		codebaseID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var allFiles []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return 0, err
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
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, filePath := range staleFiles {
		_, err = tx.Exec(
			"DELETE FROM codemogger_chunks WHERE codebase_id = ? AND file_path = ?",
			codebaseID, filePath,
		)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(
			"DELETE FROM codemogger_indexed_files WHERE codebase_id = ? AND file_path = ?",
			codebaseID, filePath,
		)
		if err != nil {
			return 0, err
		}
	}

	err = tx.Commit()
	return len(staleFiles), err
}

func (s *Store) CodemoggerBatchUpsertEmbeddings(items []struct {
	ChunkKey  string
	Embedding []float32
	ModelName string
}) error {
	if len(items) == 0 {
		return nil
	}
	if err := s.reconnect(); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		blob, err := json.Marshal(item.Embedding)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			"UPDATE codemogger_chunks SET embedding = vector32(?), embedding_model = ? WHERE chunk_key = ?",
			string(blob), item.ModelName, item.ChunkKey,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) CodemoggerGetStaleEmbeddings(codebaseID string, modelName string, limit int) ([]struct {
	ChunkKey  string
	Name      string
	Signature string
	FilePath  string
	Kind      string
	Snippet   string
}, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT chunk_key, name, signature, file_path, kind, snippet
		FROM codemogger_chunks
		WHERE codebase_id = ? AND (embedding IS NULL OR embedding_model != ?)
		LIMIT ?
	`

	rows, err := s.db.Query(query, codebaseID, modelName, limit)
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

func (s *Store) CodemoggerRebuildFTSTable(codebaseID string) error {
	return nil
}

func (s *Store) CodemoggerVectorSearch(queryEmbedding []float32, limit int, includeSnippet bool) ([]SearchResult, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("empty query embedding")
	}
	if err := s.reconnect(); err != nil {
		return nil, err
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

	rows, err := s.db.Query(sqlQuery, queryVec, limit)
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

func (s *Store) CodemoggerFTSSearch(query string, limit int, includeSnippet bool) ([]SearchResult, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT chunk_key, file_path, codemogger_chunks.name, kind, codemogger_chunks.signature, codemogger_chunks.snippet, start_line, end_line
		FROM codemogger_chunks
		JOIN codemogger_chunks_fts ON codemogger_chunks_fts.rowid = codemogger_chunks.id
		WHERE codemogger_chunks_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := s.db.Query(sqlQuery, query, limit)
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

func (s *Store) CodemoggerListFiles(codebaseID string) ([]FileInfo, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	var rows *sql.Rows
	var err error

	if codebaseID != "" {
		rows, err = s.db.Query(
			"SELECT file_path, file_hash, chunk_count, indexed_at FROM codemogger_indexed_files WHERE codebase_id = ? ORDER BY file_path",
			codebaseID,
		)
	} else {
		rows, err = s.db.Query(
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
