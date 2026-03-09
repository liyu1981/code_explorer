package db

import (
	"database/sql"
	"time"
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

func (s *Store) GetResearchSessionByCodebase(codebaseID int64) (*ResearchSession, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, codebase_id, title, state, created_at, archived_at
		FROM research_sessions
		WHERE codebase_id = ? AND archived_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`
	var sess ResearchSession
	err := s.db.QueryRow(query, codebaseID).Scan(
		&sess.ID, &sess.CodebaseID, &sess.Title,
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

func (s *Store) ListResearchSessions(includeArchived bool) ([]ResearchSession, error) {
	if err := s.reconnect(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, codebase_id, title, state, created_at, archived_at
		FROM research_sessions
	`
	if !includeArchived {
		query += " WHERE archived_at IS NULL"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ResearchSession
	for rows.Next() {
		var sess ResearchSession
		if err := rows.Scan(
			&sess.ID, &sess.CodebaseID, &sess.Title,
			&sess.State, &sess.CreatedAt, &sess.ArchivedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, sess)
	}
	return results, rows.Err()
}

func (s *Store) DeleteResearchSession(id string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	_, err := s.db.Exec("DELETE FROM research_sessions WHERE id = ?", id)
	return err
}

func (s *Store) SaveResearchReportChunk(sessionID, turnID, chunk string) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	query := `
		INSERT INTO research_reports (session_id, turn_id, stream_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(turn_id) DO UPDATE SET
			stream_data = stream_data || excluded.stream_data,
			updated_at = excluded.updated_at
	`
	_, err := s.db.Exec(query, sessionID, turnID, chunk, now, now)
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

	var results []ResearchReport
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

func (s *Store) PruneArchivedSessions(maxArchived int) error {
	if err := s.reconnect(); err != nil {
		return err
	}

	// Find sessions to delete
	query := `
		SELECT id FROM research_sessions
		WHERE archived_at IS NOT NULL
		ORDER BY archived_at ASC
		LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM research_sessions WHERE archived_at IS NOT NULL)
	`
	rows, err := s.db.Query(query, maxArchived)
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
