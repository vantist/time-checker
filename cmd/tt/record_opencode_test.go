package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/recorder"
)

// Task 2.1: resolvePromptInput with --tool opencode must not require
// --transcript-path (no error) and must leave TranscriptPath empty so that
// RecordPrompt stores NULL prompt_line_offset for the turn.
func TestResolvePromptInput_OpenCode_NoTranscriptPath(t *testing.T) {
	t.Setenv("PROCESS_PID", "")
	t.Setenv("PROCESS_START", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("session", "", "")
	cmd.Flags().String("project", "", "")
	cmd.Flags().String("tool", "claude-code", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("transcript-path", "", "")

	cmd.Flags().Set("tool", "opencode")
	cmd.Flags().Set("session", "sess-oc-p1")
	cmd.Flags().Set("project", "/repo")

	// Empty stdin (no hook payload) — mimic opencode plugin calling via flag.
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close()

	input, err := resolvePromptInput(cmd)
	if err != nil {
		t.Fatalf("resolvePromptInput: %v", err)
	}
	if input.TranscriptPath != "" {
		t.Errorf("TranscriptPath = %q, want empty for opencode", input.TranscriptPath)
	}
	if input.Tool != "opencode" {
		t.Errorf("Tool = %q, want opencode", input.Tool)
	}
}

// Task 2.1: opencode prompt stores NULL prompt_line_offset in turns table.
func TestRecordPrompt_OpenCode_NullOffset(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("TT_DB_PATH", dbDir+string(filepath.Separator)+"test.db")
	conn, err := db.Open()
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer conn.Close()

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-oc-p2",
		Project:   "/repo",
		Tool:      "opencode",
		Model:     "",
	}); err != nil {
		t.Fatalf("RecordPrompt: %v", err)
	}

	var offset *int
	var transcriptPath *string
	err = conn.QueryRow(
		"SELECT prompt_line_offset, transcript_path FROM turns WHERE session_id=?",
		"sess-oc-p2",
	).Scan(&offset, &transcriptPath)
	if err != nil {
		t.Fatalf("query turn: %v", err)
	}
	if offset != nil {
		t.Errorf("prompt_line_offset = %d, want NULL", *offset)
	}
	if transcriptPath != nil {
		t.Errorf("transcript_path = %q, want NULL", *transcriptPath)
	}
}