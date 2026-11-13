package database

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

func Connect(dsn string) (*sql.DB, error) {
	optimizedDSN := dsn + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=ON"

	db, err := sql.Open("sqlite", optimizedDSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(1 * time.Hour)
	return db, nil
}