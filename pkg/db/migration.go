package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	
	"github.com/liyu1981/code_explorer/pkg/libsql"

	_ "github.com/tursodatabase/go-libsql"
)

//go:embed *.sql
var MigrationFiles embed.FS

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

func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return err
	}

	d, err := iofs.New(MigrationFiles, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func Migrate(dbPath string) error {
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	return runMigrations(db)
}

func Rollback(dbPath string) error {
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return err
	}

	d, err := iofs.New(MigrationFiles, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func Step(dbPath string, n int) error {
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return err
	}

	d, err := iofs.New(MigrationFiles, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	return m.Steps(n)
}

func Force(dbPath string, v int) error {
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return err
	}

	d, err := iofs.New(MigrationFiles, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	return m.Force(v)
}

func Drop(dbPath string) error {
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return err
	}

	d, err := iofs.New(MigrationFiles, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	return m.Drop()
}

var ErrNoChange = migrate.ErrNoChange
