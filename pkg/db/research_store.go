package db

import (
	"context"
	"database/sql"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) SaveResearchSession(ctx context.Context, session *ResearchSession) error {
	query := `
		INSERT INTO research_sessions (id, codebase_id, title, state, created_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			state = excluded.state,
			archived_at = excluded.archived_at
	`
	_, err := s.ExecWrite(ctx, query,
		session.ID, session.CodebaseID, session.Title,
		session.State, session.CreatedAt, session.ArchivedAt,
	)
	return err
}

func (s *Store) GetResearchSession(ctx context.Context, id string) (*ResearchSession, error) {
	query := `
		SELECT rs.id, rs.codebase_id, c.root_path, c.name, c.version, rs.title, rs.state, rs.created_at, rs.archived_at
		FROM research_sessions rs
		JOIN codebases c ON rs.codebase_id = c.id
		WHERE rs.id = ?
	`
	var sess ResearchSession
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sess.ID, &sess.CodebaseID, &sess.CodebasePath, &sess.CodebaseName, &sess.CodebaseVersion, &sess.Title,
		&sess.State, &sess.CreatedAt, &sess.ArchivedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) GetResearchSessionsByCodebase(ctx context.Context, codebaseID string, includeArchived bool) ([]ResearchSession, error) {
	query := `
		SELECT rs.id, rs.codebase_id, c.root_path, c.name, c.version, rs.title, rs.state, rs.created_at, rs.archived_at
		FROM research_sessions rs
		JOIN codebases c ON rs.codebase_id = c.id
		WHERE rs.codebase_id = ?
	`
	if !includeArchived {
		query += " AND rs.archived_at IS NULL"
	}
	// Order by active first (archived_at IS NULL is 1 in boolean expression, so DESC puts NULLs first), then by creation date DESC
	query += " ORDER BY rs.archived_at IS NULL DESC, rs.created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, codebaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results = []ResearchSession{}
	for rows.Next() {
		var sess ResearchSession
		if err := rows.Scan(
			&sess.ID, &sess.CodebaseID, &sess.CodebasePath, &sess.CodebaseName, &sess.CodebaseVersion, &sess.Title,
			&sess.State, &sess.CreatedAt, &sess.ArchivedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, sess)
	}
	return results, rows.Err()
}

func (s *Store) ListResearchSessions(ctx context.Context, includeArchived bool) ([]ResearchSession, error) {
	query := `
		SELECT rs.id, rs.codebase_id, c.root_path, c.name, c.version, rs.title, rs.state, rs.created_at, rs.archived_at
		FROM research_sessions rs
		JOIN codebases c ON rs.codebase_id = c.id
	`
	if !includeArchived {
		query += " WHERE rs.archived_at IS NULL"
	}
	query += " ORDER BY rs.archived_at IS NULL DESC, rs.created_at DESC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results = []ResearchSession{}
	for rows.Next() {
		var sess ResearchSession
		if err := rows.Scan(
			&sess.ID, &sess.CodebaseID, &sess.CodebasePath, &sess.CodebaseName, &sess.CodebaseVersion, &sess.Title,
			&sess.State, &sess.CreatedAt, &sess.ArchivedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, sess)
	}
	return results, rows.Err()
}

// GetResearchSessionsPaginated returns a paginated list of sessions for management
func (s *Store) GetResearchSessionsPaginated(ctx context.Context, codebaseID string, page, pageSize int) ([]ResearchSession, int, error) {
	offset := (page - 1) * pageSize
	whereClause := ""
	var args []any
	if codebaseID != "" {
		whereClause = "WHERE rs.codebase_id = ?"
		args = append(args, codebaseID)
	}

	countQuery := "SELECT COUNT(*) FROM research_sessions rs " + whereClause
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT rs.id, rs.codebase_id, c.root_path, c.name, c.version, rs.title, rs.state, rs.created_at, rs.archived_at
		FROM research_sessions rs
		JOIN codebases c ON rs.codebase_id = c.id
		` + whereClause + `
		ORDER BY rs.created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, pageSize, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results = []ResearchSession{}
	for rows.Next() {
		var sess ResearchSession
		if err := rows.Scan(
			&sess.ID, &sess.CodebaseID, &sess.CodebasePath, &sess.CodebaseName, &sess.CodebaseVersion, &sess.Title,
			&sess.State, &sess.CreatedAt, &sess.ArchivedAt,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, sess)
	}
	return results, total, rows.Err()
}

func (s *Store) DeleteResearchSession(ctx context.Context, id string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM research_sessions WHERE id = ?", id)
	return err
}

func (s *Store) SaveResearchReportChunk(ctx context.Context, sessionID, turnID, chunk string) error {
	now := time.Now().UnixMilli()
	query := `
		INSERT INTO research_reports (id, session_id, turn_id, stream_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(turn_id) DO UPDATE SET
			stream_data = stream_data || excluded.stream_data,
			updated_at = excluded.updated_at
	`
	newID, _ := gonanoid.New()
	_, err := s.ExecWrite(ctx, query, newID, sessionID, turnID, chunk, now, now)
	return err
}

func (s *Store) GetResearchReportsBySession(ctx context.Context, sessionID string) ([]ResearchReport, error) {
	query := `
		SELECT id, session_id, turn_id, stream_data, created_at, updated_at
		FROM research_reports
		WHERE session_id = ?
		ORDER BY id ASC
	`
	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results = []ResearchReport{}
	for rows.Next() {
		var r ResearchReport
		if err := rows.Scan(&r.ID, &r.SessionID, &r.TurnID, &r.StreamData, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) DeleteReportsBySession(ctx context.Context, sessionID string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM research_reports WHERE session_id = ?", sessionID)
	return err
}

func (s *Store) DeleteResearchReport(ctx context.Context, turnID string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM research_reports WHERE turn_id = ?", turnID)
	return err
}

func (s *Store) SaveSavedReport(ctx context.Context, report *SavedReport) error {
	if report.ID == "" {
		id, _ := gonanoid.New()
		report.ID = id
	}
	if report.CreatedAt == 0 {
		report.CreatedAt = time.Now().UnixMilli()
	}

	query := `
		INSERT INTO saved_reports (id, session_id, codebase_id, title, query, stream_data, codebase_name, codebase_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			query = excluded.query,
			stream_data = excluded.stream_data
	`
	_, err := s.ExecWrite(ctx, query,
		report.ID, report.SessionID, report.CodebaseID, report.Title,
		report.Query, report.StreamData, report.CodebaseName, report.CodebasePath, report.CreatedAt,
	)
	return err
}

func (s *Store) GetSavedReport(ctx context.Context, id string) (*SavedReport, error) {
	query := `
		SELECT id, session_id, codebase_id, title, query, stream_data, codebase_name, codebase_path, created_at
		FROM saved_reports
		WHERE id = ?
	`
	var r SavedReport
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.SessionID, &r.CodebaseID, &r.Title, &r.Query, &r.StreamData, &r.CodebaseName, &r.CodebasePath, &r.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) ListSavedReports(ctx context.Context, page, pageSize int, searchText string) ([]SavedReport, int, error) {
	offset := (page - 1) * pageSize
	var total int
	var rows *sql.Rows
	var err error

	if searchText != "" {
		countQuery := "SELECT COUNT(*) FROM saved_reports_fts WHERE saved_reports_fts MATCH ?"
		err = s.db.QueryRowContext(ctx, countQuery, searchText).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		query := `
			SELECT r.id, r.session_id, r.codebase_id, r.title, r.query, r.stream_data, r.codebase_name, r.codebase_path, r.created_at
			FROM saved_reports r
			JOIN saved_reports_fts f ON r.id = f.id
			WHERE f.saved_reports_fts MATCH ?
			ORDER BY r.created_at DESC
			LIMIT ? OFFSET ?
		`
		rows, err = s.db.QueryContext(ctx, query, searchText, pageSize, offset)
	} else {
		countQuery := "SELECT COUNT(*) FROM saved_reports"
		err = s.db.QueryRowContext(ctx, countQuery).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		query := `
			SELECT id, session_id, codebase_id, title, query, stream_data, codebase_name, codebase_path, created_at
			FROM saved_reports
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		rows, err = s.db.QueryContext(ctx, query, pageSize, offset)
	}

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results = []SavedReport{}
	for rows.Next() {
		var r SavedReport
		if err := rows.Scan(
			&r.ID, &r.SessionID, &r.CodebaseID, &r.Title, &r.Query, &r.StreamData, &r.CodebaseName, &r.CodebasePath, &r.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, r)
	}
	return results, total, rows.Err()
}

func (s *Store) DeleteSavedReport(ctx context.Context, id string) error {
	_, err := s.ExecWrite(ctx, "DELETE FROM saved_reports WHERE id = ?", id)
	return err
}

func (s *Store) PruneReportsBySession(ctx context.Context, sessionID string, maxReports int) error {
	query := `
		SELECT turn_id FROM research_reports
		WHERE session_id = ?
		ORDER BY created_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_reports WHERE session_id = ?)
	`
	rows, err := s.db.QueryContext(ctx, query, sessionID, maxReports, sessionID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var turnIDs []string
	for rows.Next() {
		var turnID string
		if err := rows.Scan(&turnID); err != nil {
			return err
		}
		turnIDs = append(turnIDs, turnID)
	}

	if len(turnIDs) == 0 {
		return nil
	}

	for _, turnID := range turnIDs {
		if err := s.DeleteResearchReport(ctx, turnID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) PruneSessionsByCodebase(ctx context.Context, codebaseID string, maxTotal int) error {
	// Find sessions to delete
	// First prioritize archived (archived_at IS NOT NULL is 1, so DESC puts them first)
	// Then within that group, oldest first (created_at ASC)
	query := `
		SELECT id FROM research_sessions
		WHERE codebase_id = ?
		ORDER BY archived_at IS NOT NULL DESC, created_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_sessions WHERE codebase_id = ?)
	`
	rows, err := s.db.QueryContext(ctx, query, codebaseID, maxTotal, codebaseID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		if err := s.DeleteResearchSession(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) PruneArchivedSessions(ctx context.Context, maxTotal int) error {
	query := `
		SELECT id FROM research_sessions
		ORDER BY archived_at IS NOT NULL DESC, created_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_sessions)
	`
	rows, err := s.db.QueryContext(ctx, query, maxTotal)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		if err := s.DeleteResearchSession(ctx, id); err != nil {
			return err
		}
	}

	return nil
}
