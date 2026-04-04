package zoekt

import (
	"database/sql"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/libsql"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
)

func TestNewBuilder(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
	}

	b, err := NewBuilder(opts)
	if err != nil {
		t.Fatalf("NewBuilder failed: %v", err)
	}
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}
}

func TestNewBuilderNoName(t *testing.T) {
	opts := Options{}

	_, err := NewBuilder(opts)
	if err == nil {
		t.Error("NewBuilder should fail without Name")
	}
}

func TestBuilderAdd(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
	}
	b, _ := NewBuilder(opts)

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main"),
		Branches: []string{"main"},
	}

	err := b.Add(doc)
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if len(b.todo) != 1 {
		t.Errorf("todo len = %d, want 1", len(b.todo))
	}
}

func TestBuilderAddFile(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
	}
	b, _ := NewBuilder(opts)

	err := b.AddFile("test.go", []byte("package main"))
	if err != nil {
		t.Errorf("AddFile failed: %v", err)
	}

	if len(b.todo) != 1 {
		t.Errorf("todo len = %d, want 1", len(b.todo))
	}
}

func TestBuilderAddAfterFinish(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
	}
	b, _ := NewBuilder(opts)

	b.Finish()

	doc := Document{Name: "test.go", Content: []byte("test")}
	err := b.Add(doc)
	if err != nil {
		t.Errorf("Add after Finish failed: %v", err)
	}
}

func TestBuilderFinishMultiple(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
	}
	b, _ := NewBuilder(opts)

	b.AddFile("test.go", []byte("package main"))

	err1 := b.Finish()
	err2 := b.Finish()

	if err1 != nil {
		t.Errorf("First Finish failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Finish failed: %v", err2)
	}
}

func TestBuilderSizeTracking(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
		ShardMax: 100,
	}
	b, _ := NewBuilder(opts)

	initialSize := b.size

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main func main() {}"),
		Branches: []string{"main"},
	}
	b.Add(doc)

	if b.size == initialSize {
		t.Error("size should increase after Add")
	}
}

func TestBuilderFlush(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name:     "test-repo",
			Branches: []RepositoryBranch{{Name: "main"}},
		},
		ShardMax:    10,
		Parallelism: 1,
	}
	b, _ := NewBuilder(opts)

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main func main() {}"),
		Branches: []string{"main"},
	}

	err := b.Add(doc)
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}
}

func TestBuilderSkipReason(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
		SizeMax:     5,
		Parallelism: 1,
	}
	b, _ := NewBuilder(opts)

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main"),
		Branches: []string{"main"},
	}

	err := b.Add(doc)
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if len(b.todo) != 1 {
		t.Errorf("todo len = %d, want 1", len(b.todo))
	}
	if b.todo[0].SkipReason != SkipReasonTooLarge {
		t.Errorf("SkipReason = %v, want SkipReasonTooLarge", b.todo[0].SkipReason)
	}
}

func TestBuilderIgnoreSizeMax(t *testing.T) {
	opts := Options{
		RepositoryDescription: Repository{
			Name: "test-repo",
		},
		SizeMax:     5,
		LargeFiles:  []string{"big.go"},
		Parallelism: 1,
	}
	b, _ := NewBuilder(opts)

	doc := Document{
		Name:     "big.go",
		Content:  []byte("package main func main() {}"),
		Branches: []string{"main"},
	}

	err := b.Add(doc)
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if b.todo[0].SkipReason != SkipReasonNone {
		t.Errorf("SkipReason = %v, want SkipReasonNone", b.todo[0].SkipReason)
	}
}

func TestSortDocuments(t *testing.T) {
	docs := []*Document{
		{Name: "a.go"},
		{Name: "z.go", SkipReason: SkipReasonBinary},
		{Name: "m.go"},
	}

	sortDocuments(docs)

	if docs[len(docs)-1].SkipReason != SkipReasonBinary {
		t.Error("Skipped doc should be last")
	}
}

func TestRank(t *testing.T) {
	doc1 := &Document{Name: "a.go", Content: []byte("short")}
	doc2 := &Document{Name: "b.go", Content: []byte("much longer content")}

	r1 := rank(doc1, 0)
	r2 := rank(doc2, 1)

	if len(r1) != len(r2) {
		t.Errorf("rank slices have different lengths: %d vs %d", len(r1), len(r2))
	}
}

func TestSquashRange(t *testing.T) {
	if squashRange(0) != 0 {
		t.Error("squashRange(0) should be 0")
	}

	result := squashRange(100)
	if result <= 0 || result >= 1 {
		t.Errorf("squashRange(100) = %v, want between 0 and 1", result)
	}

	if squashRange(10) >= squashRange(100) {
		t.Error("squashRange should be monotonic")
	}
}

func TestCheckIsNegatePattern(t *testing.T) {
	negated, validated := checkIsNegatePattern("!test.go")
	if !negated || validated != "test.go" {
		t.Errorf("checkIsNegatePattern(!test.go) = (%v, %q), want (true, test.go)", negated, validated)
	}

	negated, validated = checkIsNegatePattern("test.go")
	if negated || validated != "test.go" {
		t.Errorf("checkIsNegatePattern(test.go) = (%v, %q), want (false, test.go)", negated, validated)
	}
}

func TestBuilderWithSQLiteFS(t *testing.T) {
	var store *db.Store
	var closeStore func()
	store, closeStore = db.SetupTestDB(t)
	defer closeStore()

	fs := sqlitefs.OpenFS(store)

	opts := Options{
		RepositoryDescription: Repository{
			ID:   "456",
			Name: "test-repo-sqlite",
			Branches: []RepositoryBranch{
				{Name: "main"},
				{Name: "develop"},
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
		{Name: "main.go", Content: []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"), Branches: []string{"main", "develop"}},
		{Name: "utils.go", Content: []byte("package utils\n\nfunc Add(a, b int) int {\n\treturn a + b\n}"), Branches: []string{"main"}},
		{Name: "README.md", Content: []byte("# Test Project\n\nThis is a test."), Branches: []string{"main"}},
	}

	for _, doc := range docs {
		if err := b.Add(doc); err != nil {
			t.Fatalf("Add failed for %s: %v", doc.Name, err)
		}
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	shardPath := "repo_456/456_v16.00000.zoekt"
	exists, err := fs.Exists("/" + shardPath)
	if err != nil {
		t.Fatalf("Exists check failed: %v", err)
	}
	if !exists {
		t.Errorf("Shard file %s should exist in sqlitefs", shardPath)
	}

	data, err := fs.Read("/"+shardPath, 0, 100)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Shard data should not be empty")
	}
	t.Logf("Successfully wrote shard to sqlitefs: %s, size: %d bytes", shardPath, len(data))
}

func OpenDB(dbPath string) (*sql.DB, error) {
	dbConn, err := libsql.OpenLibsqlDb(dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(dbPath); err != nil {
		_ = dbConn.Close()
		return nil, err
	}
	return dbConn, nil
}
