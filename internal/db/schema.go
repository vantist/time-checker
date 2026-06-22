package db

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/user/tt/internal/pricing"
	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	path := os.Getenv("TT_DB_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".tt", "data.db")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id          TEXT PRIMARY KEY,
			project     TEXT,
			tool        TEXT,
			model       TEXT,
			branch      TEXT,
			work_item   TEXT,
			started_at  DATETIME NOT NULL,
			ended_at    DATETIME
		);

		CREATE TABLE IF NOT EXISTS turns (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id          TEXT NOT NULL REFERENCES sessions(id),
			prompt_at           DATETIME NOT NULL,
			response_at         DATETIME,
			input_tokens        INTEGER,
			output_tokens       INTEGER,
			cache_read_tokens   INTEGER,
			cache_creation_tokens INTEGER,
			estimated_cost_usd  REAL
		);
	`); err != nil {
		return err
	}

	if err := addSessionColumns(db); err != nil {
		return err
	}
	if err := addTurnColumns(db); err != nil {
		return err
	}
	if err := setupTurnModelUsages(db); err != nil {
		return err
	}
	return recalculateCosts(db)
}

func setupTurnModelUsages(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS turn_model_usages (
			id                          INTEGER PRIMARY KEY AUTOINCREMENT,
			turn_id                     INTEGER NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
			model                       TEXT NOT NULL,
			is_subagent                 BOOLEAN NOT NULL DEFAULT 0,
			input_tokens                INTEGER NOT NULL DEFAULT 0,
			output_tokens               INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens           INTEGER NOT NULL DEFAULT 0,
			cache_creation_tokens       INTEGER NOT NULL DEFAULT 0,
			cache_creation_5m_tokens    INTEGER NOT NULL DEFAULT 0,
			cache_creation_1h_tokens    INTEGER NOT NULL DEFAULT 0,
			estimated_cost_usd          REAL NOT NULL DEFAULT 0.0,
			UNIQUE(turn_id, model, is_subagent)
		);
		CREATE INDEX IF NOT EXISTS idx_turn_model_usages_turn_id ON turn_model_usages(turn_id);
	`); err != nil {
		return err
	}

	// Backfill existing turns
	_, err := db.Exec(`
		INSERT INTO turn_model_usages (
			turn_id,
			model,
			is_subagent,
			input_tokens,
			output_tokens,
			cache_read_tokens,
			cache_creation_tokens,
			cache_creation_5m_tokens,
			cache_creation_1h_tokens,
			estimated_cost_usd
		)
		SELECT 
			t.id,
			COALESCE(NULLIF(t.model, ''), NULLIF(s.model, ''), 'unknown'),
			0,
			COALESCE(t.input_tokens, 0),
			COALESCE(t.output_tokens, 0),
			COALESCE(t.cache_read_tokens, 0),
			COALESCE(t.cache_creation_tokens, 0),
			COALESCE(t.cache_creation_5m_tokens, 0),
			COALESCE(t.cache_creation_1h_tokens, 0),
			COALESCE(t.estimated_cost_usd, 0.0)
		FROM turns t
		LEFT JOIN sessions s ON t.session_id = s.id
		WHERE NOT EXISTS (
			SELECT 1 FROM turn_model_usages u WHERE u.turn_id = t.id
		);
	`)
	return err
}

// addTurnColumns adds transcript_path and prompt_line_offset to turns
// if they don't already exist (SQLite does not support ADD COLUMN IF NOT EXISTS).
func addTurnColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(turns)")
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		existing[name] = true
	}
	rows.Close()

	alters := []struct {
		col string
		def string
	}{
		{"transcript_path", "TEXT"},
		{"prompt_line_offset", "INTEGER"},
		{"model", "TEXT"},
		{"cache_creation_5m_tokens", "INTEGER"},
		{"cache_creation_1h_tokens", "INTEGER"},
		{"subagent_tokens_settled", "BOOLEAN DEFAULT 0"},
	}
	for _, a := range alters {
		if existing[a.col] {
			continue
		}
		if _, err := db.Exec("ALTER TABLE turns ADD COLUMN " + a.col + " " + a.def); err != nil {
			return err
		}
	}
	return nil
}

// addSessionColumns adds process_pid, process_start, conversation_id to sessions
// if they don't already exist (SQLite does not support ADD COLUMN IF NOT EXISTS).
func addSessionColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(sessions)")
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		existing[name] = true
	}
	rows.Close()

	alters := []struct {
		col string
		def string
	}{
		{"process_pid", "INTEGER"},
		{"process_start", "INTEGER"},
		{"conversation_id", "TEXT"},
	}
	for _, a := range alters {
		if existing[a.col] {
			continue
		}
		if _, err := db.Exec("ALTER TABLE sessions ADD COLUMN " + a.col + " " + a.def); err != nil {
			return err
		}
	}
	return nil
}

func recalculateCosts(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, turn_id, model, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens, estimated_cost_usd
		FROM turn_model_usages
		WHERE (input_tokens > 0 OR output_tokens > 0)
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type usageUpdate struct {
		id      int64
		turnID  int64
		costVal float64
	}
	var updates []usageUpdate

	for rows.Next() {
		var id, turnID int64
		var model string
		var input, output, cacheRead, cacheCreate, cache5m, cache1h int
		var currentCost float64
		if err := rows.Scan(&id, &turnID, &model, &input, &output, &cacheRead, &cacheCreate, &cache5m, &cache1h, &currentCost); err != nil {
			continue
		}

		costPtr := pricing.Calculate(model, input, output, cacheRead, cacheCreate, cache5m, cache1h)
		if costPtr != nil {
			newCost := *costPtr
			if newCost > currentCost+1e-9 || newCost < currentCost-1e-9 {
				updates = append(updates, usageUpdate{id: id, turnID: turnID, costVal: newCost})
			}
		}
	}
	rows.Close()

	if len(updates) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, u := range updates {
		_, err := tx.Exec("UPDATE turn_model_usages SET estimated_cost_usd = ? WHERE id = ?", u.costVal, u.id)
		if err != nil {
			return err
		}
	}

	turnIDs := map[int64]bool{}
	for _, u := range updates {
		turnIDs[u.turnID] = true
	}

	for tid := range turnIDs {
		var sum sql.NullFloat64
		err := tx.QueryRow("SELECT SUM(estimated_cost_usd) FROM turn_model_usages WHERE turn_id = ?", tid).Scan(&sum)
		if err != nil {
			return err
		}
		var val interface{}
		if sum.Valid {
			val = sum.Float64
		} else {
			val = nil
		}
		_, err = tx.Exec("UPDATE turns SET estimated_cost_usd = ? WHERE id = ?", val, tid)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
