package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/libsql"
)

func TestMigrateFunction(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	var tables []string
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}

	expected := []string{"codemogger_codebases", "codemogger_indexed_files", "codemogger_chunks"}
	for _, e := range expected {
		found := false
		for _, t := range tables {
			if t == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q not found", e)
		}
	}
}

func TestMigrateNoChange(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("second migrate (no change): %v", err)
	}
}

func TestRollback(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := Rollback(dbPath); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	var tables []string
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}

	if len(tables) != 0 {
		t.Errorf("expected no tables after rollback, got %v", tables)
	}
}

func TestStep(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := Step(dbPath, 1); err != nil {
		t.Fatalf("step forward: %v", err)
	}

	if err := Step(dbPath, -1); err != nil {
		t.Fatalf("step backward: %v", err)
	}
}

func TestForce(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := Force(dbPath, 0); err != nil {
		t.Fatalf("force: %v", err)
	}

	if err := Force(dbPath, 1); err != nil {
		t.Fatalf("force: %v", err)
	}
}

func TestDrop(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	if err := Migrate(dbPath); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := Drop(dbPath); err != nil {
		t.Fatalf("drop: %v", err)
	}

	db, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	var tables []string
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}

	if len(tables) != 0 {
		t.Errorf("expected no tables after drop, got %v", tables)
	}
}

func TestOpenWithMigrations(t *testing.T) {
	dir, err := os.MkdirTemp("", "migration-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}

	expected := []string{"codemogger_codebases", "codemogger_indexed_files", "codemogger_chunks"}
	for _, e := range expected {
		found := false
		for _, t := range tables {
			if t == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q not found", e)
		}
	}
}
