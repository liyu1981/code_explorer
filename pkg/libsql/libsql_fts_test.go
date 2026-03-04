package libsql

import (
	"os"
	"testing"
)

func TestFTS5(t *testing.T) {
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

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE articles USING fts5(
			title,
			content,
			tokenize='porter'
		)
	`)
	if err != nil {
		t.Fatalf("create FTS5 table: %v", err)
	}

	articles := []struct {
		title   string
		content string
	}{
		{"Go Programming", "Go is an open source programming language supported by Google."},
		{"Python Programming", "Python is a high-level programming language known for its simplicity."},
		{"Rust Programming", "Rust is a systems programming language focused on safety and performance."},
		{"JavaScript Web", "JavaScript is a programming language for web development."},
		{"Go HTTP Servers", "Go has excellent support for building HTTP servers and web applications."},
	}

	for _, a := range articles {
		_, err = db.Exec(
			"INSERT INTO articles (title, content) VALUES (?, ?)",
			a.title, a.content,
		)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	rows, err := db.Query(`
		SELECT title, highlight(articles, 1, '<b>', '</b>') as content
		FROM articles
		WHERE articles MATCH 'programming'
		ORDER BY rank
	`)
	if err != nil {
		t.Fatalf("FTS5 query: %v", err)
	}
	defer rows.Close()

	var results []struct {
		title   string
		content string
	}
	for rows.Next() {
		var r struct {
			title   string
			content string
		}
		if err := rows.Scan(&r.title, &r.content); err != nil {
			t.Fatalf("scan: %v", err)
		}
		results = append(results, r)
	}

	if rows.Err() != nil {
		t.Fatalf("rows error: %v", rows.Err())
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	rows, err = db.Query(`
		SELECT title FROM articles
		WHERE articles MATCH 'prog*'
		ORDER BY rank
	`)
	if err != nil {
		t.Fatalf("FTS5 prefix query: %v", err)
	}

	var prefixResults []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			t.Fatalf("scan: %v", err)
		}
		prefixResults = append(prefixResults, title)
	}
	rows.Close()

	if len(prefixResults) != 4 {
		t.Fatalf("expected 4 prefix results, got %d", len(prefixResults))
	}

	rows, err = db.Query(`
		SELECT title FROM articles
		WHERE articles MATCH 'go AND http'
		ORDER BY rank
	`)
	if err != nil {
		t.Fatalf("FTS5 boolean query: %v", err)
	}

	var booleanResults []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			t.Fatalf("scan: %v", err)
		}
		booleanResults = append(booleanResults, title)
	}
	rows.Close()

	if len(booleanResults) != 1 || booleanResults[0] != "Go HTTP Servers" {
		t.Fatalf("expected [Go HTTP Servers], got %v", booleanResults)
	}

	t.Logf("FTS5 basic results: %v", results)
	t.Logf("FTS5 prefix results: %v", prefixResults)
	t.Logf("FTS5 boolean results: %v", booleanResults)
	t.Log("FTS5 test passed!")
}
