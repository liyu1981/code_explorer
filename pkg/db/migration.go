package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/libsql"
	_ "github.com/tursodatabase/go-libsql"
)

//go:embed *.sql
var MigrationFiles embed.FS

var ErrNoChange = errors.New("no change")

type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

type Migrator struct {
	db *sql.DB
	fs embed.FS
}

func NewMigrator(db *sql.DB, fsys embed.FS) *Migrator {
	return &Migrator{db: db, fs: fsys}
}

func (m *Migrator) ensureMigrationTable() error {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL)`)
	return err
}

func (m *Migrator) getVersion() (int, bool, error) {
	var version int
	var dirty bool
	err := m.db.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return version, dirty, err
}

func (m *Migrator) setVersion(version int, dirty bool) error {
	_, err := m.db.Exec(`DELETE FROM schema_migrations`)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)`, version, dirty)
	return err
}

func (m *Migrator) getMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(m.fs, ".")
	if err != nil {
		return nil, err
	}

	migrationsMap := make(map[int]*Migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		nameParts := strings.Split(parts[1], ".")
		if len(nameParts) < 3 {
			continue
		}
		name := nameParts[0]
		direction := nameParts[1]

		mig, ok := migrationsMap[version]
		if !ok {
			mig = &Migration{Version: version, Name: name}
			migrationsMap[version] = mig
		}

		content, err := fs.ReadFile(m.fs, filename)
		if err != nil {
			return nil, err
		}

		if direction == "up" {
			mig.Up = string(content)
		} else if direction == "down" {
			mig.Down = string(content)
		}
	}

	var migrations []Migration
	for _, mig := range migrationsMap {
		migrations = append(migrations, *mig)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *Migrator) Up() error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	version, dirty, err := m.getVersion()
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("database is dirty, version %d", version)
	}

	migrations, err := m.getMigrations()
	if err != nil {
		return err
	}

	applied := false
	for _, mig := range migrations {
		if mig.Version > version {
			if err := m.applyMigration(mig, true); err != nil {
				return err
			}
			applied = true
		}
	}

	if !applied {
		return ErrNoChange
	}
	return nil
}

func (m *Migrator) Down() error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	version, dirty, err := m.getVersion()
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("database is dirty, version %d", version)
	}

	migrations, err := m.getMigrations()
	if err != nil {
		return err
	}

	applied := false
	for i := len(migrations) - 1; i >= 0; i-- {
		mig := migrations[i]
		if mig.Version <= version {
			if err := m.applyMigration(mig, false); err != nil {
				return err
			}
			applied = true
		}
	}

	if !applied {
		return ErrNoChange
	}
	return nil
}

func (m *Migrator) Step(n int) error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	version, dirty, err := m.getVersion()
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("database is dirty, version %d", version)
	}

	migrations, err := m.getMigrations()
	if err != nil {
		return err
	}

	if n > 0 {
		count := 0
		for _, mig := range migrations {
			if mig.Version > version {
				if err := m.applyMigration(mig, true); err != nil {
					return err
				}
				count++
				if count == n {
					break
				}
			}
		}
	} else if n < 0 {
		count := 0
		for i := len(migrations) - 1; i >= 0; i-- {
			mig := migrations[i]
			if mig.Version <= version {
				if err := m.applyMigration(mig, false); err != nil {
					return err
				}
				count++
				if count == -n {
					break
				}
			}
		}
	}

	return nil
}

func (m *Migrator) Force(v int) error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}
	return m.setVersion(v, false)
}

func (m *Migrator) Drop() error {
	// Revert all migrations
	if err := m.Down(); err != nil && err != ErrNoChange {
		return err
	}

	// Drop the migration table itself
	_, err := m.db.Exec(`DROP TABLE IF EXISTS schema_migrations`)
	return err
}

func (m *Migrator) Status() (version int, dirty bool, all []Migration, err error) {
	if err := m.ensureMigrationTable(); err != nil {
		return 0, false, nil, err
	}

	version, dirty, err = m.getVersion()
	if err != nil {
		return 0, false, nil, err
	}

	all, err = m.getMigrations()
	return version, dirty, all, err
}

func (m *Migrator) applyMigration(mig Migration, up bool) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Set dirty state
	if err := m.setVersion(mig.Version, true); err != nil {
		return err
	}

	query := mig.Up
	newVersion := mig.Version
	if !up {
		query = mig.Down
		// Find previous version
		migrations, err := m.getMigrations()
		if err != nil {
			return err
		}
		newVersion = 0
		for i := len(migrations) - 1; i >= 0; i-- {
			if migrations[i].Version < mig.Version {
				newVersion = migrations[i].Version
				break
			}
		}
	}

	// Split by semicolon and execute one by one
	statements := strings.Split(query, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w\nStatement: %s", mig.Version, mig.Name, err, stmt)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Set clean state with new version
	return m.setVersion(newVersion, false)
}

func runMigrations(db *sql.DB) error {
	m := NewMigrator(db, MigrationFiles)
	err := m.Up()
	if err == ErrNoChange {
		return nil
	}
	return err
}

func Migrate(dbPath string) error {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	return runMigrations(db)
}

func Rollback(dbPath string) error {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	m := NewMigrator(db, MigrationFiles)
	err = m.Down()
	if err == ErrNoChange {
		return nil
	}
	return err
}

func Step(dbPath string, n int) error {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	m := NewMigrator(db, MigrationFiles)
	return m.Step(n)
}

func Force(dbPath string, v int) error {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	m := NewMigrator(db, MigrationFiles)
	return m.Force(v)
}

func Drop(dbPath string) error {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	m := NewMigrator(db, MigrationFiles)
	return m.Drop()
}

func GetStatus(dbPath string) (int, bool, []Migration, error) {
	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return 0, false, nil, err
	}
	defer db.Close()

	m := NewMigrator(db, MigrationFiles)
	return m.Status()
}
