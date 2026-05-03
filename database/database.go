package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS commands (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL UNIQUE,
			response   TEXT    NOT NULL,
			trigger    TEXT    NOT NULL DEFAULT 'command',
			cooldown   INTEGER NOT NULL DEFAULT 0,
			enabled    INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS command_aliases (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			command_id INTEGER NOT NULL REFERENCES commands(id) ON DELETE CASCADE,
			alias      TEXT    NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS quotes (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			text       TEXT    NOT NULL,
			added_by   TEXT    NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// Migrate existing databases that predate the cooldown column.
	// Fails silently on fresh databases where the column already exists.
	db.Exec(`ALTER TABLE commands ADD COLUMN cooldown INTEGER NOT NULL DEFAULT 0`)

	return nil
}
