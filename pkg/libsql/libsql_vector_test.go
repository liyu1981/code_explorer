package libsql

import (
	"os"
	"testing"
)

func TestVectorSearch(t *testing.T) {
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
		CREATE TABLE movies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			year INT,
			embedding F32_BLOB(4)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	embeddings := []struct {
		title string
		year  int
		emb   string
	}{
		{"Napoleon", 2023, "[0.800, 0.579, 0.481, 0.229]"},
		{"Black Hawk Down", 2001, "[0.406, 0.027, 0.378, 0.056]"},
		{"Gladiator", 2000, "[0.698, 0.140, 0.073, 0.125]"},
		{"Blade Runner", 1982, "[0.379, 0.637, 0.011, 0.647]"},
	}

	for _, m := range embeddings {
		_, err = db.Exec(
			"INSERT INTO movies (title, year, embedding) VALUES (?, ?, vector32(?))",
			m.title, m.year, m.emb,
		)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	queryVec := "[0.064, 0.777, 0.661, 0.687]"
	rows, err := db.Query(`
		SELECT title, year, vector_distance_cos(embedding, vector32(?)) AS distance
		FROM movies
		ORDER BY distance ASC
	`, queryVec)
	if err != nil {
		t.Fatalf("vector query: %v", err)
	}
	defer rows.Close()

	var results []struct {
		title    string
		year     int
		distance float64
	}
	for rows.Next() {
		var r struct {
			title    string
			year     int
			distance float64
		}
		if err := rows.Scan(&r.title, &r.year, &r.distance); err != nil {
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

	for i := 1; i < len(results); i++ {
		if results[i].distance < results[i-1].distance {
			t.Fatalf("results not sorted by distance")
		}
	}

	for _, r := range results {
		if r.distance < 0 || r.distance > 2 {
			t.Fatalf("distance out of range: %v", r.distance)
		}
	}

	t.Logf("Vector search results: %+v", results)
	t.Log("vector search test passed!")
}
