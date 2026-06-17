package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/user/tt/internal/db"
)

// TestUpsertSession_StableKey: same (process_pid, process_start), different conversation_id
// → second call updates conversation_id, does NOT create new session.
func TestUpsertSession_StableKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dir, "test.db"))

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	base := db.Session{
		ID:           "conv-a",
		ProcessPID:   12345,
		ProcessStart: 1700000000,
		ConversationID: "conv-a",
		Project:      "/proj",
		Tool:         "claude-code",
		StartedAt:    time.Now().UTC(),
	}

	if err := db.UpsertSession(conn, base); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second call: same process key, new conversation UUID
	base.ConversationID = "conv-b"
	base.ID = "conv-b"
	if err := db.UpsertSession(conn, base); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE process_pid=12345 AND process_start=1700000000").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 session, got %d", count)
	}

	var convID string
	conn.QueryRow("SELECT conversation_id FROM sessions WHERE process_pid=12345 AND process_start=1700000000").Scan(&convID)
	if convID != "conv-b" {
		t.Errorf("conversation_id not updated: got %q, want %q", convID, "conv-b")
	}
}

// TestUpsertSession_FallbackToID: ProcessPID=0 → original id-based upsert behaviour.
func TestUpsertSession_FallbackToID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dir, "test.db"))

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	s := db.Session{
		ID:        "fallback-sess",
		Project:   "/proj",
		Tool:      "claude-code",
		StartedAt: time.Now().UTC(),
		// ProcessPID = 0 (zero value) → fallback
	}

	if err := db.UpsertSession(conn, s); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := db.UpsertSession(conn, s); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE id='fallback-sess'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 session, got %d", count)
	}
}

func TestUpsertSessionPreservesStartedAt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TT_DB_PATH", filepath.Join(dir, "test.db"))

	conn, err := db.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	original := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := db.Session{
		ID:        "sess-abc",
		Project:   "/home/user/proj",
		Tool:      "claude-code",
		StartedAt: original,
	}

	if err := db.UpsertSession(conn, s); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert with later time — started_at must stay original
	s.StartedAt = original.Add(time.Hour)
	if err := db.UpsertSession(conn, s); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var got string
	conn.QueryRow("SELECT started_at FROM sessions WHERE id=?", s.ID).Scan(&got)
	if got != original.UTC().Format(time.RFC3339) {
		t.Errorf("started_at changed: got %q want %q", got, original.UTC().Format(time.RFC3339))
	}
}
