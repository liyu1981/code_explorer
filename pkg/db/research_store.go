package db

import (
	"database/sql"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) SaveResearchSession(session *ResearchSession) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	query := `
		INSERT INTO research_sessions (id, codebase_id, title, state, created_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			state = excluded.state,
			archived_at = excluded.archived_at
	`
	_, err := s.db.Exec(query,
		session.ID, session.CodebaseID, session.Title,
		session.State, session.CreatedAt, session.ArchivedAt,
	)
	return err
}

func (s *Store) GetResearchSession(id string) (*ResearchSession, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT rs.id, rs.codebase_id, c.root_path, c.name, c.version, rs.title, rs.state, rs.created_at, rs.archived_at
		FROM research_sessions rs
		JOIN codebases c ON rs.codebase_id = c.id
		WHERE rs.id = ?
	`
	var sess ResearchSession
	err := s.db.QueryRow(query, id).Scan(
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

func (s *Store) GetResearchSessionsByCodebase(codebaseID string, includeArchived bool) ([]ResearchSession, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

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

	rows, err := s.db.Query(query, codebaseID)
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

func (s *Store) ListResearchSessions(includeArchived bool) ([]ResearchSession, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT rs.id, rs.codebase_id, c.root_path, c.name, c.version, rs.title, rs.state, rs.created_at, rs.archived_at
		FROM research_sessions rs
		JOIN codebases c ON rs.codebase_id = c.id
	`
	if !includeArchived {
		query += " WHERE rs.archived_at IS NULL"
	}
	query += " ORDER BY rs.archived_at IS NULL DESC, rs.created_at DESC"

	rows, err := s.db.Query(query)
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
func (s *Store) GetResearchSessionsPaginated(codebaseID string, page, pageSize int) ([]ResearchSession, int, error) {
	if err := s.reconnect(); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	whereClause := ""
	var args []any
	if codebaseID != "" {
		whereClause = "WHERE rs.codebase_id = ?"
		args = append(args, codebaseID)
	}

	countQuery := "SELECT COUNT(*) FROM research_sessions rs " + whereClause
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
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
	rows, err := s.db.Query(query, args...)
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

func (s *Store) DeleteResearchSession(id string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	// Because of ON DELETE CASCADE, reports will be deleted automatically
	_, err := s.db.Exec("DELETE FROM research_sessions WHERE id = ?", id)
	return err
}

func (s *Store) SaveResearchReportChunk(sessionID, turnID, chunk string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	query := `
		INSERT INTO research_reports (id, session_id, turn_id, stream_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(turn_id) DO UPDATE SET
			stream_data = stream_data || excluded.stream_data,
			updated_at = excluded.updated_at
	`
	newID, _ := gonanoid.New()
	_, err := s.db.Exec(query, newID, sessionID, turnID, chunk, now, now)
	return err
}

func (s *Store) GetResearchReportsBySession(sessionID string) ([]ResearchReport, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, session_id, turn_id, stream_data, created_at, updated_at
		FROM research_reports
		WHERE session_id = ?
		ORDER BY id ASC
	`
	rows, err := s.db.Query(query, sessionID)
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

func (s *Store) DeleteReportsBySession(sessionID string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("DELETE FROM research_reports WHERE session_id = ?", sessionID)
	return err
}

func (s *Store) DeleteResearchReport(turnID string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("DELETE FROM research_reports WHERE turn_id = ?", turnID)
	return err
}

func (s *Store) SaveSavedReport(report *SavedReport) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	if report.ID == "" {
		id, _ := gonanoid.New()
		report.ID = id
	}
	if report.CreatedAt == 0 {
		report.CreatedAt = time.Now().UnixMilli()
	}

	query := `
		INSERT INTO saved_reports (id, session_id, codebase_id, title, query, content, codebase_name, codebase_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			query = excluded.query,
			content = excluded.content
	`
	_, err := s.db.Exec(query,
		report.ID, report.SessionID, report.CodebaseID, report.Title,
		report.Query, report.Content, report.CodebaseName, report.CodebasePath, report.CreatedAt,
	)
	return err
}

func (s *Store) GetSavedReport(id string) (*SavedReport, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, session_id, codebase_id, title, query, content, codebase_name, codebase_path, created_at
		FROM saved_reports
		WHERE id = ?
	`
	var r SavedReport
	err := s.db.QueryRow(query, id).Scan(
		&r.ID, &r.SessionID, &r.CodebaseID, &r.Title, &r.Query, &r.Content, &r.CodebaseName, &r.CodebasePath, &r.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) ListSavedReports(page, pageSize int, searchText string) ([]SavedReport, int, error) {
	if err := s.reconnect(); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var total int
	var rows *sql.Rows
	var err error

	if searchText != "" {
		countQuery := "SELECT COUNT(*) FROM saved_reports_fts WHERE saved_reports_fts MATCH ?"
		err = s.db.QueryRow(countQuery, searchText).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		query := `
			SELECT r.id, r.session_id, r.codebase_id, r.title, r.query, r.content, r.codebase_name, r.codebase_path, r.created_at
			FROM saved_reports r
			JOIN saved_reports_fts f ON r.id = f.id
			WHERE f.saved_reports_fts MATCH ?
			ORDER BY r.created_at DESC
			LIMIT ? OFFSET ?
		`
		rows, err = s.db.Query(query, searchText, pageSize, offset)
	} else {
		countQuery := "SELECT COUNT(*) FROM saved_reports"
		err = s.db.QueryRow(countQuery).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		query := `
			SELECT id, session_id, codebase_id, title, query, content, codebase_name, codebase_path, created_at
			FROM saved_reports
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		rows, err = s.db.Query(query, pageSize, offset)
	}

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results = []SavedReport{}
	for rows.Next() {
		var r SavedReport
		if err := rows.Scan(
			&r.ID, &r.SessionID, &r.CodebaseID, &r.Title, &r.Query, &r.Content, &r.CodebaseName, &r.CodebasePath, &r.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, r)
	}
	return results, total, rows.Err()
}

func (s *Store) DeleteSavedReport(id string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("DELETE FROM saved_reports WHERE id = ?", id)
	return err
}

func (s *Store) PruneReportsBySession(sessionID string, maxReports int) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	query := `
		SELECT turn_id FROM research_reports
		WHERE session_id = ?
		ORDER BY created_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_reports WHERE session_id = ?)
	`
	rows, err := s.db.Query(query, sessionID, maxReports, sessionID)
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
		if err := s.DeleteResearchReport(turnID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) PruneSessionsByCodebase(codebaseID string, maxTotal int) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	// Find sessions to delete
	// First prioritize archived (archived_at IS NOT NULL is 1, so DESC puts them first)
	// Then within that group, oldest first (created_at ASC)
	query := `
		SELECT id FROM research_sessions
		WHERE codebase_id = ?
		ORDER BY archived_at IS NOT NULL DESC, created_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_sessions WHERE codebase_id = ?)
	`
	rows, err := s.db.Query(query, codebaseID, maxTotal, codebaseID)
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
		if err := s.DeleteResearchSession(id); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) PruneArchivedSessions(maxTotal int) error {
	// Reusing PruneSessionsByCodebase logic but globally if needed,
	// however, the requirement is "per room" (codebase).
	// Let's keep this as a global safety prune if requested,
	// but the primary one is PruneSessionsByCodebase.
	if err := s.reconnect(); err != nil {
		return err
	}

	query := `
		SELECT id FROM research_sessions
		ORDER BY archived_at IS NOT NULL DESC, created_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_sessions)
	`
	rows, err := s.db.Query(query, maxTotal)
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
		if err := s.DeleteResearchSession(id); err != nil {
			return err
		}
	}

	return nil
}
