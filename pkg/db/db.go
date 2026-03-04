package db

import (
	"database/sql"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func Open(dbPath string) (*sql.DB, error) {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}
