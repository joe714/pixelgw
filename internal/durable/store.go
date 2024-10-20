package durable

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/canonical/sqlair"

	_ "github.com/mattn/go-sqlite3"
)

type Count struct {
	Count int `db:"count"`
}

type SchemaVersion struct {
	Version int `db:"version"`
}

type Store struct {
	DB *sqlair.DB
}

type TX struct {
	Context context.Context
	tx      *sqlair.TX
}

func (tx *TX) Query(s *sqlair.Statement, inputArgs ...any) *sqlair.Query {
	return tx.tx.Query(tx.Context, s, inputArgs...)
}

func NewStore() (*Store, error) {
	sqldb, err := sql.Open("sqlite3", "./etc/cfg.db")
	if err != nil {
		return nil, err
	}

	db := sqlair.NewDB(sqldb)
	store := Store{DB: db}

	stmt := sqlair.MustPrepare("CREATE TABLE IF NOT EXISTS schema_version(version integer PRIMARY KEY);")
	err = db.Query(context.Background(), stmt).Run()
	if err != nil {
		return nil, err
	}

	stmt = sqlair.MustPrepare(
		"SELECT &SchemaVersion.version FROM schema_version ORDER BY schema_version.version DESC LIMIT 1",
		SchemaVersion{})
	var v SchemaVersion
	err = db.Query(context.Background(), stmt).Get(&v)
	if err != nil {
		if errors.Is(err, sqlair.ErrNoRows) {
			v, err = store.initSchema()
		} else {
			log.Printf("Error validating schema version: %v\n", err)
			return nil, err
		}
	}

	log.Printf("Current database schema: %v\n", v.Version)

	return &store, nil
}

func (store *Store) Transaction(ctx context.Context, opts *sqlair.TXOptions, fn func(*TX) error) error {
	tx, err := store.DB.Begin(ctx, opts)
	if err != nil {
		return err
	}

	stx := TX{Context: ctx, tx: tx}
	err = fn(&stx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (store *Store) Update(ctx context.Context, fn func(*TX) error) error {
	opts := sqlair.TXOptions{ReadOnly: false}
	return store.Transaction(ctx, &opts, fn)
}

func (store *Store) View(ctx context.Context, fn func(*TX) error) error {
	opts := sqlair.TXOptions{ReadOnly: true}
	return store.Transaction(ctx, &opts, fn)
}

func (store *Store) initSchema() (SchemaVersion, error) {
	stmts := []string{
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
			last_time TEXT
			)`,
		`CREATE INDEX idx_channel_devices ON devices (channel_uuid, uuid)`,
		`INSERT INTO channels VALUES ('76ffcb18-d3c7-40d5-abea-3fe86d02a4ba', 'default', 'The default channel')`,
		`INSERT INTO channel_applets VALUES ('efe35cfa-4076-4e84-9c9c-961e821769bd',
                                             '76ffcb18-d3c7-40d5-abea-3fe86d02a4ba',
                                             0,
                                             'clock-by-henry',
                                             '{"blink_time": "true", "use_12h": "true"}')`,
		`INSERT INTO channel_applets VALUES ('e7a8d2d4-f8a7-44a7-8158-1525582f88e0',
                                             '76ffcb18-d3c7-40d5-abea-3fe86d02a4ba',
                                             1,
                                             'dvd-logo',
                                             NULL)`,
		`INSERT INTO schema_version VALUES(1);`,
	}
	log.Println("Perform initial database setup")

	err := store.Update(context.Background(), func(tx *TX) error {
		for _, s := range stmts {
			log.Println(s)
			stmt := sqlair.MustPrepare(s)
			err := tx.Query(stmt).Run()
			if err != nil {
				log.Printf("Error: %v", err)
				return err
			}
		}

		return nil
	})

	return SchemaVersion{Version: 1}, err
}
