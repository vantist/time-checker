package db

import (
	"database/sql"
	"errors"
	"time"
)

type Session struct {
	ID             string
	Project        string
	Tool           string
	Model          string
	Branch         string
	WorkItem       string
	StartedAt      time.Time
	EndedAt        *time.Time
	ProcessPID     int64
	ProcessStart   int64
	ConversationID string
}

// UpsertSession inserts or updates a session.
// When ProcessPID and ProcessStart are both non-zero, (process_pid, process_start)
// is used as the stable key: the session is created once and conversation_id is
// updated on subsequent calls. Otherwise the original id-based INSERT OR IGNORE
// behaviour is preserved.
func UpsertSession(db *sql.DB, s Session) error {
	if s.ProcessPID != 0 && s.ProcessStart != 0 {
		return upsertByProcessKey(db, s)
	}
	_, err := db.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, tool, model, branch, work_item, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Project, s.Tool, s.Model, s.Branch, s.WorkItem,
		s.StartedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func upsertByProcessKey(db *sql.DB, s Session) error {
	// Check if a session with this process key already exists.
	var existingID string
	err := db.QueryRow(
		"SELECT id FROM sessions WHERE process_pid = ? AND process_start = ?",
		s.ProcessPID, s.ProcessStart,
	).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		// First time: insert new session.
		_, err = db.Exec(`
			INSERT INTO sessions
				(id, project, tool, model, branch, work_item, started_at, process_pid, process_start, conversation_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.Project, s.Tool, s.Model, s.Branch, s.WorkItem,
			s.StartedAt.UTC().Format(time.RFC3339),
			s.ProcessPID, s.ProcessStart, s.ConversationID,
		)
		return err
	}

	// Existing session: update conversation_id (and ended_at if set).
	var endedAt interface{}
	if s.EndedAt != nil {
		endedAt = s.EndedAt.UTC().Format(time.RFC3339)
	}
	_, err = db.Exec(`
		UPDATE sessions SET conversation_id = ?, ended_at = ?
		WHERE process_pid = ? AND process_start = ?`,
		s.ConversationID, endedAt, s.ProcessPID, s.ProcessStart,
	)
	return err
}
