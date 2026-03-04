package libsql

import (
	"fmt"
	"os"
	"testing"
)

func TestLibsql(t *testing.T) {
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	db, err := OpenLibsqlDb(dir + "/test.db")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	for i := 0; i < 10; i++ {
		_, err = db.Exec(fmt.Sprintf("INSERT INTO test (id, name) VALUES (%d, 'test-%d')", i, i))
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	rows, err := db.Query("SELECT * FROM test")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			t.Fatalf("scan: %v", err)
		}
		if id != i {
			t.Fatalf("expected id %d, got %d", i, id)
		}
		if name != fmt.Sprintf("test-%d", i) {
			t.Fatalf("expected name %s, got %s", fmt.Sprintf("test-%d", i), name)
		}
		i++
	}

	if rows.Err() != nil {
		t.Fatalf("rows error: %v", rows.Err())
	}

	if i != 10 {
		t.Fatalf("expected 10 rows, got %d", i)
	}

	t.Log("go-libsql basic test passed!")
}
