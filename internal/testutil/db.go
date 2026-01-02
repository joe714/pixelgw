package testutil

import (
	"context"
	"database/sql"

	"github.com/canonical/sqlair"
	"github.com/joe714/pixelgw/internal/durable"
	_ "github.com/mattn/go-sqlite3"
)

// TestDB holds the database connection for testing
type TestDB struct {
	DB    *sqlair.DB
	sqlDB *sql.DB
}

// NewTestDB creates a new in-memory SQLite database for testing.
// Returns the database and a cleanup function that should be called when done.
func NewTestDB() (*TestDB, func(), error) {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, nil, err
	}

	db := sqlair.NewDB(sqldb)
	testDB := &TestDB{DB: db, sqlDB: sqldb}

	// Initialize schema
	if err := testDB.initSchema(); err != nil {
		sqldb.Close()
		return nil, nil, err
	}

	cleanup := func() {
		sqldb.Close()
	}

	return testDB, cleanup, nil
}

// initSchema creates the database schema for testing
func (t *TestDB) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version(version integer PRIMARY KEY)`,
		`CREATE TABLE channels (
			uuid TEXT PRIMARY KEY COLLATE NOCASE,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			comment TEXT
		)`,
		`CREATE TABLE channel_applets (
			uuid TEXT PRIMARY KEY COLLATE NOCASE,
			channel_uuid TEXT NOT NULL COLLATE NOCASE,
			idx INTEGER NOT NULL,
			app_id TEXT NOT NULL,
			config TEXT,
			UNIQUE (channel_uuid, idx)
		)`,
		`CREATE TABLE devices (
			uuid TEXT PRIMARY KEY COLLATE NOCASE,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			channel_uuid TEXT NOT NULL COLLATE NOCASE,
			last_ip TEXT,
			last_connect_time TEXT,
			last_disconnect_time TEXT,
			device_info TEXT,
			device_info_updated TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_channel_devices ON devices (channel_uuid)`,
		`INSERT INTO schema_version VALUES(3)`,
	}

	for _, stmt := range stmts {
		prepared := sqlair.MustPrepare(stmt)
		if err := t.DB.Query(context.Background(), prepared).Run(); err != nil {
			return err
		}
	}

	return nil
}

// Exec executes a raw SQL statement for test setup
func (t *TestDB) Exec(stmt string) error {
	prepared := sqlair.MustPrepare(stmt)
	return t.DB.Query(context.Background(), prepared).Run()
}

// Store returns a durable.Store using this test database
func (t *TestDB) Store() *durable.Store {
	return durable.NewStoreWithDB(t.DB)
}

// NewTestStore creates an in-memory durable.Store for testing.
// Returns the store and a cleanup function.
func NewTestStore() (*durable.Store, func(), error) {
	testDB, cleanup, err := NewTestDB()
	if err != nil {
		return nil, nil, err
	}
	return testDB.Store(), cleanup, nil
}
