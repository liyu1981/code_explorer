package zoekt

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
)

func TestQueryString(t *testing.T) {
	tests := []struct {
		q    Query
		want string
	}{
		{
			q:    &Substring{Pattern: "hello"},
			want: "hello",
		},
		{
			q:    &Substring{Pattern: "main", FileName: true},
			want: "file:main",
		},
		{
			q:    &Substring{Pattern: "main", CaseSensitive: true},
			want: "case:main",
		},
		{
			q:    &Substring{Pattern: "main", FileName: true, CaseSensitive: true},
			want: "case:file:main",
		},
		{
			q:    &And{Children: []Query{&Substring{Pattern: "a"}, &Substring{Pattern: "b"}}},
			want: "and(a, b)",
		},
		{
			q:    &Or{Children: []Query{&Substring{Pattern: "a"}, &Substring{Pattern: "b"}}},
			want: "or(a, b)",
		},
		{
			q:    &Not{Child: &Substring{Pattern: "a"}},
			want: "not(a)",
		},
		{
			q:    &Branch{Pattern: "main"},
			want: "branch:main",
		},
		{
			q:    &Repo{Pattern: "repo"},
			want: "repo:repo",
		},
		{
			q:    &Language{Pattern: "Go"},
			want: "lang:Go",
		},
	}

	for _, tt := range tests {
		if got := tt.q.String(); got != tt.want {
			t.Errorf("%T.String() = %q, want %q", tt.q, got, tt.want)
		}
	}
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "<nil>"},
		{"  ", "<nil>"},
		{"hello", "hello"},
		{"  world  ", "world"},
	}

	for _, tt := range tests {
		q, err := ParseQuery(tt.input)
		if err != nil {
			t.Errorf("ParseQuery(%q) error: %v", tt.input, err)
			continue
		}
		got := "<nil>"
		if q != nil {
			got = q.String()
		}
		if got != tt.want {
			t.Errorf("ParseQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSearchOptionsSetDefaults(t *testing.T) {
	opts := SearchOptions{}
	opts.SetDefaults()
	if opts.MaxMatchCount != 500 {
		t.Errorf("MaxMatchCount = %d, want 500", opts.MaxMatchCount)
	}

	opts = SearchOptions{MaxMatchCount: 100}
	opts.SetDefaults()
	if opts.MaxMatchCount != 100 {
		t.Errorf("MaxMatchCount = %d, want 100", opts.MaxMatchCount)
	}
}

func TestQueryWithSQLiteFS(t *testing.T) {
	store, closeStore := db.SetupTestDB(t)
	defer closeStore()

	fs := sqlitefs.OpenFS(store)

	opts := Options{
		RepositoryDescription: Repository{
			ID:   "123",
			Name: "test-query-repo",
			Branches: []RepositoryBranch{
				{Name: "main"},
			},
		},
		IndexFS:     fs,
		Parallelism: 1,
		ShardMax:    100 << 20,
	}

	b, err := NewBuilder(opts)
	if err != nil {
		t.Fatalf("NewBuilder failed: %v", err)
	}

	docs := []Document{
		{Name: "main.go", Content: []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}"), Branches: []string{"main"}},
		{Name: "helper.go", Content: []byte("package main\n\nfunc help() {\n\tfmt.Println(\"help me\")\n}"), Branches: []string{"main"}},
	}

	for _, doc := range docs {
		if err := b.Add(doc); err != nil {
			t.Fatalf("Add failed for %s: %v", doc.Name, err)
		}
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	shardPath := "/repo_123/123_v16.00000.zoekt"
	exists, err := fs.Exists(shardPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatalf("Shard file %s does not exist", shardPath)
	}

	fileID, err := store.DB().QueryContext(context.Background(), "SELECT id, size FROM fs_nodes WHERE name = ?", "123_v16.00000.zoekt")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer fileID.Close()
	var id64 int64
	var size int64
	if fileID.Next() {
		if err := fileID.Scan(&id64, &size); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
	} else {
		t.Fatalf("Shard file %s not found in DB", shardPath)
	}

	data, err := fs.Read(shardPath, 0, int(size))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	indexFile := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(indexFile)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}
	defer id.Close()

	// Test Substring query
	q := &Substring{Pattern: "fmt.Println"}
	res, err := id.Search(q, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(res.Files) != 2 {
		t.Errorf("Search found %d files, want 2", len(res.Files))
	}

	// Test Filename query
	q = &Substring{Pattern: "main.go", FileName: true}
	res, err = id.Search(q, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(res.Files) != 1 {
		t.Errorf("Filename search found %d files, want 1", len(res.Files))
	}
	if len(res.Files) > 0 && res.Files[0].FileName != "main.go" {
		t.Errorf("Filename search found %q, want \"main.go\"", res.Files[0].FileName)
	}
}
