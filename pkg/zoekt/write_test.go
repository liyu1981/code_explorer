package zoekt

import (
	"bytes"
	"testing"
)

type mockIndexFS struct {
	files map[string][]byte
}

func (m *mockIndexFS) Create(path string, data []byte) error {
	m.files[path] = data
	return nil
}

func TestWriteShardToIndexFS(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}

	opts := Options{
		RepositoryDescription: Repository{
			ID:   123,
			Name: "test-repo",
			Branches: []RepositoryBranch{
				{Name: "main"},
			},
		},
		IndexFS:     mockFS,
		Parallelism: 1,
		ShardMax:    100 << 20,
	}

	b, err := NewBuilder(opts)
	if err != nil {
		t.Fatalf("NewBuilder failed: %v", err)
	}

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main\n\nfunc main() {}"),
		Branches: []string{"main"},
	}
	if err := b.Add(doc); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	if len(mockFS.files) == 0 {
		t.Error("Expected index file to be written to IndexFS")
	}

	var found bool
	var data []byte
	for path, d := range mockFS.files {
		if len(d) > 0 {
			t.Logf("Wrote index file: %s, size: %d", path, len(d))
			found = true
			data = d
			break
		}
	}

	if !found {
		t.Fatal("No index data written")
	}

	if len(data) == 0 {
		t.Error("Index data should not be empty")
	}

	if !bytes.HasPrefix(data, []byte{0, 0, 0, 0}) {
		t.Logf("Index data starts with: %v", data[:10])
	} else {
		t.Logf("Index data starts with TOC header")
	}
}

func TestShardFileName(t *testing.T) {
	tests := []struct {
		repoID   uint32
		shardNum int
		version  int
		expected string
	}{
		{123, 0, 16, "repo_00000123_v16.00000.zoekt"},
		{456, 1, 16, "repo_00000456_v16.00001.zoekt"},
		{0, 5, 17, "repo_00000000_v17.00005.zoekt"},
	}

	for _, tc := range tests {
		result := ShardFileName(tc.repoID, tc.shardNum, tc.version)
		if result != tc.expected {
			t.Errorf("ShardFileName(%d, %d, %d) = %q, want %q",
				tc.repoID, tc.shardNum, tc.version, result, tc.expected)
		}
	}
}

func TestShardPrefix(t *testing.T) {
	tests := []struct {
		repoID   uint32
		expected string
	}{
		{0, "repo_00000000"},
		{1, "repo_00000001"},
		{123, "repo_00000123"},
		{4294967295, "repo_4294967295"},
	}

	for _, tc := range tests {
		result := ShardPrefix(tc.repoID)
		if result != tc.expected {
			t.Errorf("ShardPrefix(%d) = %q, want %q", tc.repoID, result, tc.expected)
		}
	}
}
