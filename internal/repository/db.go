package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func NewSQLiteDB(path string) (*sql.DB, error) {
	// Pragmas must live in the DSN, not be applied via db.Exec: database/sql
	// pools connections, and a pragma set through Exec only takes effect on
	// whichever single connection happened to run it. Any later connection
	// the pool opens would silently fall back to SQLite's default rollback
	// journal with a zero busy_timeout, so a concurrent writer got an
	// immediate "database is locked" error instead of waiting.
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// SQLite allows only one writer at a time. Serializing all access
	// through a single connection avoids SQLITE_BUSY entirely instead of
	// relying on busy_timeout retries under contention.
	db.SetMaxOpenConns(1)

	return db, nil
}
