package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func formatFloat(f float32) string {
	return strconv.FormatFloat(float64(f), 'f', -1, 32)
}

func (s *Store) CodesummerGetOrCreateCodebase(ctx context.Context, codebaseID string) (string, error) {
	var metadataID string
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"SELECT id FROM codesummer_codebases WHERE codebase_id = ?",
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
			"INSERT INTO codesummer_codebases (id, codebase_id, indexed_at) VALUES (?, ?, unixepoch())",
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

func (s *Store) CodesummerUpsertSummary(ctx context.Context, summary CodesummerSummary) error {
	summaryID := summary.ID
	if summaryID == "" {
		summaryID, _ = gonanoid.New()
	}

	embeddingJSON, err := json.Marshal(summary.Embedding)
	if err != nil {
		return err
	}

	_, err = s.ExecWrite(ctx, `
		INSERT INTO codesummer_summaries 
		(id, codesummer_id, node_path, node_type, language, summary, definitions, dependencies, data_manipulated, data_flow, embedding, embedding_model, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())
		ON CONFLICT(codesummer_id, node_path) DO UPDATE SET
			summary = excluded.summary,
			definitions = excluded.definitions,
			dependencies = excluded.dependencies,
			data_manipulated = excluded.data_manipulated,
			data_flow = excluded.data_flow,
			embedding = excluded.embedding,
			embedding_model = excluded.embedding_model,
			indexed_at = excluded.indexed_at
	`,
		summaryID, summary.CodesummerID, summary.NodePath, summary.NodeType, summary.Language,
		summary.Summary, summary.Definitions, summary.Dependencies, summary.DataManipulated,
		summary.DataFlow, string(embeddingJSON), summary.EmbeddingModel,
	)
	return err
}

func (s *Store) CodesummerUpsertBatchSummaries(ctx context.Context, summaries []CodesummerSummary) error {
	return s.Transaction(ctx, func(tx *sql.Tx) error {
		for _, summary := range summaries {
			summaryID := summary.ID
			if summaryID == "" {
				summaryID, _ = gonanoid.New()
			}

			embeddingJSON, err := json.Marshal(summary.Embedding)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO codesummer_summaries 
				(id, codesummer_id, node_path, node_type, language, summary, definitions, dependencies, data_manipulated, data_flow, embedding, embedding_model, indexed_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())
				ON CONFLICT(codesummer_id, node_path) DO UPDATE SET
					summary = excluded.summary,
					definitions = excluded.definitions,
					dependencies = excluded.dependencies,
					data_manipulated = excluded.data_manipulated,
					data_flow = excluded.data_flow,
					embedding = excluded.embedding,
					embedding_model = excluded.embedding_model,
					indexed_at = excluded.indexed_at
			`,
				summaryID, summary.CodesummerID, summary.NodePath, summary.NodeType, summary.Language,
				summary.Summary, summary.Definitions, summary.Dependencies, summary.DataManipulated,
				summary.DataFlow, string(embeddingJSON), summary.EmbeddingModel,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) CodesummerUpsertEmbeddings(ctx context.Context, items []struct {
	CodesummerID string
	NodePath     string
	Embedding    []float32
	ModelName    string
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
				"UPDATE codesummer_summaries SET embedding = vector32(?), embedding_model = ? WHERE codesummer_id = ? AND node_path = ?",
				string(blob), item.ModelName, item.CodesummerID, item.NodePath,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) CodesummerGetSummary(ctx context.Context, codesummerID string, nodePath string) (*CodesummerSummary, error) {
	var summary CodesummerSummary
	var embeddingJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, codesummer_id, node_path, node_type, language, summary, definitions, 
		       dependencies, data_manipulated, data_flow, embedding, embedding_model, indexed_at
		FROM codesummer_summaries 
		WHERE codesummer_id = ? AND node_path = ?
	`, codesummerID, nodePath).Scan(
		&summary.ID, &summary.CodesummerID, &summary.NodePath, &summary.NodeType, &summary.Language,
		&summary.Summary, &summary.Definitions, &summary.Dependencies, &summary.DataManipulated,
		&summary.DataFlow, &embeddingJSON, &summary.EmbeddingModel, &summary.IndexedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if embeddingJSON != "" {
		json.Unmarshal([]byte(embeddingJSON), &summary.Embedding)
	}

	return &summary, nil
}

func (s *Store) CodesummerListSummaries(ctx context.Context, codesummerID string) ([]CodesummerSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, codesummer_id, node_path, node_type, language, summary, definitions, 
		       dependencies, data_manipulated, data_flow, embedding, embedding_model, indexed_at
		FROM codesummer_summaries 
		WHERE codesummer_id = ?
		ORDER BY node_path
	`, codesummerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CodesummerSummary
	for rows.Next() {
		var summary CodesummerSummary
		var embeddingJSON string
		err := rows.Scan(
			&summary.ID, &summary.CodesummerID, &summary.NodePath, &summary.NodeType, &summary.Language,
			&summary.Summary, &summary.Definitions, &summary.Dependencies, &summary.DataManipulated,
			&summary.DataFlow, &embeddingJSON, &summary.EmbeddingModel, &summary.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		if embeddingJSON != "" {
			json.Unmarshal([]byte(embeddingJSON), &summary.Embedding)
		}
		results = append(results, summary)
	}

	return results, rows.Err()
}

func (s *Store) CodesummerVectorSearch(ctx context.Context, codesummerID string, queryEmbedding []float32, limit int) ([]CodesummerSummary, error) {
	if len(queryEmbedding) == 0 {
		return nil, nil
	}

	queryVec := "["
	for i, v := range queryEmbedding {
		if i > 0 {
			queryVec += ","
		}
		queryVec += formatFloat(v)
	}
	queryVec += "]"

	sqlQuery := `
		SELECT id, codesummer_id, node_path, node_type, language, summary, definitions, 
		       dependencies, data_manipulated, data_flow, embedding, embedding_model, indexed_at
		FROM codesummer_summaries
		WHERE codesummer_id = ? AND embedding IS NOT NULL
		ORDER BY vector_distance_cos(embedding, vector32(?)) ASC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, codesummerID, queryVec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CodesummerSummary
	for rows.Next() {
		var summary CodesummerSummary
		var embeddingJSON string
		err := rows.Scan(
			&summary.ID, &summary.CodesummerID, &summary.NodePath, &summary.NodeType, &summary.Language,
			&summary.Summary, &summary.Definitions, &summary.Dependencies, &summary.DataManipulated,
			&summary.DataFlow, &embeddingJSON, &summary.EmbeddingModel, &summary.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		if embeddingJSON != "" {
			json.Unmarshal([]byte(embeddingJSON), &summary.Embedding)
		}
		results = append(results, summary)
	}

	return results, rows.Err()
}

func (s *Store) CodesummerUpsertIndexedPath(ctx context.Context, path IndexedPath) error {
	pathID := path.ID
	if pathID == "" {
		pathID, _ = gonanoid.New()
	}

	_, err := s.ExecWrite(ctx, `
		INSERT INTO codesummer_indexed_paths 
		(id, codesummer_id, node_path, node_type, file_hash, indexed_at)
		VALUES (?, ?, ?, ?, ?, unixepoch())
		ON CONFLICT(codesummer_id, node_path) DO UPDATE SET
			node_type = excluded.node_type,
			file_hash = excluded.file_hash,
			indexed_at = excluded.indexed_at
	`,
		pathID, path.CodesummerID, path.NodePath, path.NodeType, path.FileHash,
	)
	return err
}

func (s *Store) CodesummerGetIndexedPaths(ctx context.Context, codesummerID string) ([]IndexedPath, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, codesummer_id, node_path, node_type, file_hash, indexed_at
		FROM codesummer_indexed_paths 
		WHERE codesummer_id = ?
		ORDER BY node_path
	`, codesummerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []IndexedPath
	for rows.Next() {
		var path IndexedPath
		if err := rows.Scan(&path.ID, &path.CodesummerID, &path.NodePath, &path.NodeType, &path.FileHash, &path.IndexedAt); err != nil {
			return nil, err
		}
		results = append(results, path)
	}

	return results, rows.Err()
}

func (s *Store) CodesummerRemoveStalePaths(ctx context.Context, codesummerID string, activePaths []string) (int, error) {
	var removedCount int
	err := s.Transaction(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			"SELECT node_path FROM codesummer_indexed_paths WHERE codesummer_id = ?",
			codesummerID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		var allPaths []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err != nil {
				return err
			}
			allPaths = append(allPaths, p)
		}

		activeSet := make(map[string]bool)
		for _, p := range activePaths {
			activeSet[p] = true
		}

		var stalePaths []string
		for _, p := range allPaths {
			if !activeSet[p] {
				stalePaths = append(stalePaths, p)
			}
		}

		if len(stalePaths) == 0 {
			removedCount = 0
			return nil
		}

		for _, nodePath := range stalePaths {
			_, err = tx.ExecContext(ctx,
				"DELETE FROM codesummer_summaries WHERE codesummer_id = ? AND node_path = ?",
				codesummerID, nodePath,
			)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx,
				"DELETE FROM codesummer_indexed_paths WHERE codesummer_id = ? AND node_path = ?",
				codesummerID, nodePath,
			)
			if err != nil {
				return err
			}
		}

		removedCount = len(stalePaths)
		return nil
	})

	return removedCount, err
}

func (s *Store) CodesummerGetMetadataByCodebase(ctx context.Context, codebaseID string) (*CodesummerCodebase, error) {
	var m CodesummerCodebase
	err := s.db.QueryRowContext(ctx,
		"SELECT id, codebase_id, indexed_at FROM codesummer_codebases WHERE codebase_id = ?",
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
