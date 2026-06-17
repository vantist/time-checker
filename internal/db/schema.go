package db

import (
	"database/sql"
	"os"
	"path/filepath"

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
	_, err := db.Exec(`
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
	`)
	return err
}
