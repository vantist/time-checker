package main

import (
	"bytes"
	"database/sql"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "tt-test-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	binPath = filepath.Join(tmpDir, "tt")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to compile tt binary: %v\nOutput: %s", err, string(output))
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestIntegration_BinaryExists(t *testing.T) {
	if binPath == "" {
		t.Fatal("binPath is not set")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("compiled binary does not exist at %s: %v", binPath, err)
	}
}

func runTT(t *testing.T, home, dbPath, stdin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+home, "TT_DB_PATH="+dbPath)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestIntegration_RunTTHelper(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	stdout, stderr, err := runTT(t, home, dbPath, "", "version")
	if err != nil {
		t.Fatalf("runTT failed: %v, stderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "dev") {
		t.Errorf("expected version output 'dev', got: %s", stdout)
	}
}

type dbSession struct {
	ID        string
	Project   string
	Tool      string
	Model     string
	Branch    *string
	WorkItem  *string
	StartedAt string
	EndedAt   *string
}

type dbTurn struct {
	ID                  int64
	SessionID           string
	PromptAt            string
	ResponseAt          *string
	InputTokens         *int64
	OutputTokens        *int64
	CacheReadTokens     *int64
	CacheCreationTokens *int64
	EstimatedCostUSD    *float64
}

type dbTurnModelUsage struct {
	ID                  int64
	TurnID              int64
	Model               string
	IsSubagent          bool
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	EstimatedCostUSD    float64
}

func getSession(t *testing.T, dbPath, sessionID string) (*dbSession, error) {
	t.Helper()
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	var s dbSession
	err = dbConn.QueryRow("SELECT id, project, tool, model, branch, work_item, started_at, ended_at FROM sessions WHERE id = ?", sessionID).
		Scan(&s.ID, &s.Project, &s.Tool, &s.Model, &s.Branch, &s.WorkItem, &s.StartedAt, &s.EndedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func getTurns(t *testing.T, dbPath, sessionID string) ([]dbTurn, error) {
	t.Helper()
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	rows, err := dbConn.Query("SELECT id, session_id, prompt_at, response_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd FROM turns WHERE session_id = ? ORDER BY id ASC", sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turns []dbTurn
	for rows.Next() {
		var r dbTurn
		err := rows.Scan(&r.ID, &r.SessionID, &r.PromptAt, &r.ResponseAt, &r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheCreationTokens, &r.EstimatedCostUSD)
		if err != nil {
			return nil, err
		}
		turns = append(turns, r)
	}
	return turns, nil
}

func getTurnModelUsages(t *testing.T, dbPath string, turnID int64) ([]dbTurnModelUsage, error) {
	t.Helper()
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	rows, err := dbConn.Query("SELECT id, turn_id, model, is_subagent, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd FROM turn_model_usages WHERE turn_id = ? ORDER BY id ASC", turnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []dbTurnModelUsage
	for rows.Next() {
		var u dbTurnModelUsage
		err := rows.Scan(&u.ID, &u.TurnID, &u.Model, &u.IsSubagent, &u.InputTokens, &u.OutputTokens, &u.CacheReadTokens, &u.CacheCreationTokens, &u.EstimatedCostUSD)
		if err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, nil
}

func TestIntegration_DBAssertHelpers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_assert.db")
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			project TEXT,
			tool TEXT,
			model TEXT,
			branch TEXT,
			work_item TEXT,
			started_at DATETIME,
			ended_at DATETIME
		);
		CREATE TABLE turns (
			id INTEGER PRIMARY KEY,
			session_id TEXT,
			prompt_at DATETIME,
			response_at DATETIME,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cache_read_tokens INTEGER,
			cache_creation_tokens INTEGER,
			estimated_cost_usd REAL
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	_, err = dbConn.Exec(`
		INSERT INTO sessions (id, project, tool, model, branch, work_item, started_at)
		VALUES ('sess-1', '/proj', 'claude-code', 'claude-3-5', 'main', 'wi-1', '2026-06-22T00:00:00Z');
		INSERT INTO turns (id, session_id, prompt_at, response_at, input_tokens, output_tokens)
		VALUES (1, 'sess-1', '2026-06-22T00:00:05Z', '2026-06-22T00:00:10Z', 10, 20);
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	sess, err := getSession(t, dbPath, "sess-1")
	if err != nil {
		t.Fatalf("getSession failed: %v", err)
	}
	if sess.ID != "sess-1" {
		t.Errorf("expected session ID 'sess-1', got %q", sess.ID)
	}

	turns, err := getTurns(t, dbPath, "sess-1")
	if err != nil {
		t.Fatalf("getTurns failed: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
}
