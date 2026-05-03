package database

import (
	"database/sql"
	"testing"
)

func setupTestDB(t *testing.T) *testDB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return &testDB{db}
}

type testDB struct {
	*sql.DB
}

func (tdb *testDB) close() {
	tdb.DB.Close()
}

func TestOpenAndMigrate(t *testing.T) {
	db := setupTestDB(t)
	defer db.close()

	// Verify tables exist by querying them
	_, err := db.Exec(`SELECT count(*) FROM commands`)
	if err != nil {
		t.Fatalf("commands table should exist: %v", err)
	}

	_, err = db.Exec(`SELECT count(*) FROM command_aliases`)
	if err != nil {
		t.Fatalf("command_aliases table should exist: %v", err)
	}

	_, err = db.Exec(`SELECT count(*) FROM quotes`)
	if err != nil {
		t.Fatalf("quotes table should exist: %v", err)
	}
}
