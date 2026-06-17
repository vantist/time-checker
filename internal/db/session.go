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

// UpsertSession inserts or updates a session and returns the stable session ID.
// When ProcessPID and ProcessStart are both non-zero, (process_pid, process_start)
// is the stable key: the session is created once and conversation_id is updated
// on subsequent calls. The returned ID is the sessions.id of the stable row.
// Otherwise the original id-based INSERT OR IGNORE is used and s.ID is returned.
func UpsertSession(db *sql.DB, s Session) (string, error) {
	if s.ProcessPID != 0 && s.ProcessStart != 0 {
		return upsertByProcessKey(db, s)
	}
	_, err := db.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, tool, model, branch, work_item, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Project, s.Tool, s.Model, s.Branch, s.WorkItem,
		s.StartedAt.UTC().Format(time.RFC3339),
	)
	return s.ID, err
}

func upsertByProcessKey(db *sql.DB, s Session) (string, error) {
	var existingID string
	err := db.QueryRow(
		"SELECT id FROM sessions WHERE process_pid = ? AND process_start = ?",
		s.ProcessPID, s.ProcessStart,
	).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	if errors.Is(err, sql.ErrNoRows) {
		_, err = db.Exec(`
			INSERT INTO sessions
				(id, project, tool, model, branch, work_item, started_at, process_pid, process_start, conversation_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.Project, s.Tool, s.Model, s.Branch, s.WorkItem,
			s.StartedAt.UTC().Format(time.RFC3339),
			s.ProcessPID, s.ProcessStart, s.ConversationID,
		)
		return s.ID, err
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
	return existingID, err
}
