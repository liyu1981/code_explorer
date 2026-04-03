package zoekt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func readFullFile(mockFS *mockIndexFS, t *testing.T) []byte {
	t.Logf("Files in mockFS: %d", len(mockFS.files))
	for path, d := range mockFS.files {
		t.Logf("  path=%s len=%d", path, len(d))
		if len(d) == 0 {
			continue
		}
		// Check for TOC at end: last 8 bytes = tocSection (off + sz)
		if len(d) >= 8 {
			last8 := d[len(d)-8:]
			t.Logf("  last 8 bytes (potential TOC): %v", last8)
			tocOff := uint32(last8[0])<<24 | uint32(last8[1])<<16 | uint32(last8[2])<<8 | uint32(last8[3])
			tocSz := uint32(last8[4])<<24 | uint32(last8[5])<<16 | uint32(last8[6])<<8 | uint32(last8[7])
			t.Logf("  TOC: off=%d sz=%d", tocOff, tocSz)
			if tocOff > 0 && tocOff < uint32(len(d)) {
				return d
			}
		}
		// Check for TOC at beginning (older format)
		if d[0] == 0 && d[1] == 0 && d[2] == 0 && d[3] == 0 {
			return d
		}
	}
	return nil
}

func TestIndexDataLoad(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}

	opts := Options{
		RepositoryDescription: Repository{
			ID:   789,
			Name: "test-repo-search",
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

	docs := []Document{
		{Name: "main.go", Content: []byte("package main\n\nfunc main() {\n\tprintln(\"hello world\")\n}"), Branches: []string{"main"}},
		{Name: "utils.go", Content: []byte("package utils\n\nfunc Add(a, b int) int {\n\treturn a + b\n}"), Branches: []string{"main"}},
		{Name: "test.go", Content: []byte("package main\n\nfunc TestAdd(t *testing.T) {\n\texpected := 3\n\tif Add(1, 2) != expected {\n\t\tt.Error(\"fail\")\n\t}\n}"), Branches: []string{"main"}},
	}

	for _, doc := range docs {
		if err := b.Add(doc); err != nil {
			t.Fatalf("Add failed for %s: %v", doc.Name, err)
		}
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	t.Logf("Files in mockFS: %d", len(mockFS.files))
	for path, d := range mockFS.files {
		t.Logf("  path=%s len=%d first4=%v", path, len(d), d[:min(4, len(d))])
		if len(d) > 0 {
			// Debug: show what's in the first few bytes
			t.Logf("  First 20 bytes: %v", d[:min(20, len(d))])
			t.Logf("  Last 20 bytes: %v", d[max(0, len(d)-20):])
		}
	}

	data := readFullFile(mockFS, t)
	if data == nil {
		t.Fatal("No valid shard data found")
	}
	t.Logf("Index data length: %d bytes", len(data))

	indexFile := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(indexFile)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}
	defer id.Close()

	if id.numDocs() != 3 {
		t.Errorf("numDocs = %d, want 3", id.numDocs())
	}

	fileName := id.fileName(0)
	if !bytes.Equal(fileName, []byte("main.go")) {
		t.Errorf("fileName(0) = %q, want \"main.go\"", string(fileName))
	}
}

func TestIndexDataSearchSubstring(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}

	opts := Options{
		RepositoryDescription: Repository{
			ID:   790,
			Name: "test-repo-search2",
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

	data := readFullFile(mockFS, t)
	if data == nil {
		t.Fatal("No valid shard data found")
	}

	indexFile := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(indexFile)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}
	defer id.Close()

	t.Logf("numDocs: %d", id.numDocs())
	t.Logf("fileNameIndex len: %d", len(id.fileNameIndex))
	t.Logf("boundaries len: %d", len(id.boundaries))

	if id.numDocs() == 0 {
		t.Fatal("numDocs is 0!")
	}

	// Test ngrams directly
	ngrams := id.ngrams(false)
	t.Logf("contentNgrams btree: %+v", ngrams)

	// Get a known ngram like "fmt"
	testNgram := stringToNGram("fmt")
	sec := ngrams.Get(testNgram)
	t.Logf("ngram %v (fmt): off=%d sz=%d", testNgram, sec.off, sec.sz)

	// Also get an ngram that's in the index (like ngram[0])
	bucketData, _ := id.file.Read(ngrams.ngramSec.off, ngrams.ngramSec.sz)
	if len(bucketData) >= 8 {
		ng0 := ngram(binary.BigEndian.Uint64(bucketData[:8]))
		sec0 := ngrams.Get(ng0)
		t.Logf("first ngram in index 0x%016x: off=%d sz=%d", ng0, sec0.off, sec0.sz)
	}

	// Also check name ngrams (for filename search)
	nameNgrams := id.ngrams(true)
	t.Logf("nameNgrams: %+v", nameNgrams)

	if sec.sz > 0 {
		postingData, err := id.file.Read(sec.off, sec.sz)
		if err != nil {
			t.Logf("error reading postings: %v", err)
		} else {
			postings := decodePostingList(postingData)
			t.Logf("postings for 0x666d74: %v", postings)
		}
	}

	result, err := id.Search(&Substring{Pattern: "fmt"}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Errorf("Search found %d files, want 2", len(result.Files))
	}
}

func TestIndexDataSearchBranch(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}

	opts := Options{
		RepositoryDescription: Repository{
			ID:   791,
			Name: "test-repo-search3",
			Branches: []RepositoryBranch{
				{Name: "main"},
				{Name: "feature"},
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

	docs := []Document{
		{Name: "main.go", Content: []byte("package main\n\nfunc main() {}"), Branches: []string{"main"}},
		{Name: "feature.go", Content: []byte("package main\n\nfunc feature() {}"), Branches: []string{"feature"}},
		{Name: "both.go", Content: []byte("package main\n\nfunc both() {}"), Branches: []string{"main", "feature"}},
	}

	for _, doc := range docs {
		if err := b.Add(doc); err != nil {
			t.Fatalf("Add failed for %s: %v", doc.Name, err)
		}
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	data := readFullFile(mockFS, t)
	if data == nil {
		t.Fatal("No valid shard data found")
	}

	indexFile := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(indexFile)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}
	defer id.Close()

	result, err := id.Search(&Branch{Pattern: "main"}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Errorf("Search found %d files, want 2 (main.go and both.go)", len(result.Files))
	}
}

func TestIndexDataSearchFileName(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}

	opts := Options{
		RepositoryDescription: Repository{
			ID:   792,
			Name: "test-repo-search4",
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

	docs := []Document{
		{Name: "main.go", Content: []byte("package main"), Branches: []string{"main"}},
		{Name: "utils.go", Content: []byte("package utils"), Branches: []string{"main"}},
		{Name: "main_test.go", Content: []byte("package main"), Branches: []string{"main"}},
	}

	for _, doc := range docs {
		if err := b.Add(doc); err != nil {
			t.Fatalf("Add failed for %s: %v", doc.Name, err)
		}
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	data := readFullFile(mockFS, t)
	if data == nil {
		t.Fatal("No valid shard data found")
	}

	indexFile := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(indexFile)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}
	defer id.Close()

	result, err := id.Search(&Substring{Pattern: "main", FileName: true}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Errorf("Search found %d files, want 2 (main.go and main_test.go)", len(result.Files))
	}
}

func TestIndexDataSearchEmptyResult(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}

	opts := Options{
		RepositoryDescription: Repository{
			ID:   793,
			Name: "test-repo-search5",
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

	docs := []Document{
		{Name: "main.go", Content: []byte("package main\n\nfunc main() {}"), Branches: []string{"main"}},
	}

	if err := b.Add(docs[0]); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish failed: %v", err)
	}

	data := readFullFile(mockFS, t)
	if data == nil {
		t.Fatal("No valid shard data found")
	}

	indexFile := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(indexFile)
	if err != nil {
		t.Fatalf("loadIndexData failed: %v", err)
	}
	defer id.Close()

	result, err := id.Search(&Substring{Pattern: "nonexistent"}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("Search found %d files, want 0", len(result.Files))
	}
}
