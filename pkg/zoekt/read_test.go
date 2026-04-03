package zoekt

import (
	"bytes"
	"reflect"
	"testing"
)

func TestReadWrite(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}
	opts := Options{
		RepositoryDescription: Repository{
			ID:   "1",
			Name: "test-repo",
		},
		IndexFS:     mockFS,
		Parallelism: 1,
		ShardMax:    100 << 20,
	}

	b, err := NewBuilder(opts)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	if err := b.AddFile("filename", []byte("abcde")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	var data []byte
	for _, v := range mockFS.files {
		data = v
		break
	}

	f := NewIndexFile(data, "test.zoekt")
	rd := &reader{r: f}

	toc, err := rd.readTOC()
	if err != nil {
		t.Errorf("got read error %v", err)
	}
	if toc.fileContents.data.sz != 5 {
		t.Errorf("got contents size %d, want 5", toc.fileContents.data.sz)
	}

	id, err := rd.readIndexData(toc)
	if err != nil {
		t.Fatalf("readIndexData: %v", err)
	}
	defer id.Close()

	if got := id.fileName(0); string(got) != "filename" {
		t.Errorf("got filename %q, want %q", string(got), "filename")
	}

	if sec := id.contentNgrams.Get(stringToNGram("abc")); sec.sz == 0 {
		t.Errorf("did not find ngram abc")
	}

	if sec := id.contentNgrams.Get(stringToNGram("bcq")); sec.sz > 0 {
		t.Errorf("found ngram bcq")
	}
}

func TestReadWriteNames(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}
	opts := Options{
		RepositoryDescription: Repository{
			ID:   "2",
			Name: "test-repo-names",
		},
		IndexFS:     mockFS,
		Parallelism: 1,
		ShardMax:    100 << 20,
	}

	b, err := NewBuilder(opts)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	if err := b.AddFile("abCd", []byte("")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	var data []byte
	for _, v := range mockFS.files {
		data = v
		break
	}

	f := NewIndexFile(data, "test.zoekt")
	rd := &reader{r: f}

	toc, err := rd.readTOC()
	if err != nil {
		t.Errorf("got read error %v", err)
	}
	if toc.fileNames.data.sz != 4 {
		t.Errorf("got contents size %d, want 4", toc.fileNames.data.sz)
	}

	id, err := rd.readIndexData(toc)
	if err != nil {
		t.Fatalf("readIndexData: %v", err)
	}
	defer id.Close()

	if !reflect.DeepEqual([]uint32{0, 4}, id.fileNameIndex) {
		t.Errorf("got index %v, want {0,4}", id.fileNameIndex)
	}

	gotSec := id.nameNgrams.Get(stringToNGram("bCd"))
	if gotSec.sz == 0 {
		t.Fatalf("nameNgrams.Get failed for bCd")
	}

	postingData, err := f.Read(gotSec.off, gotSec.sz)
	if err != nil {
		t.Fatalf("Read postingData: %v", err)
	}

	// Posting for "bCd" (doc 0) should decode to [0] or [some_rune_offset]
	// Since filename is indexed by rune offsets as well.
	postings := decodePostingList(postingData)
	if len(postings) == 0 {
		t.Errorf("got no postings for bCd")
	}
}

func TestLoadIndexData(t *testing.T) {
	mockFS := &mockIndexFS{files: make(map[string][]byte)}
	opts := Options{
		RepositoryDescription: Repository{
			ID:   "3",
			Name: "test-repo-load",
		},
		IndexFS:     mockFS,
		Parallelism: 1,
		ShardMax:    100 << 20,
	}

	b, err := NewBuilder(opts)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	if err := b.AddFile("main.go", []byte("package main\nfunc main() {}\n")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	if err := b.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	var data []byte
	for _, v := range mockFS.files {
		data = v
		break
	}

	f := NewIndexFile(data, "test.zoekt")
	id, err := loadIndexData(f)
	if err != nil {
		t.Fatalf("loadIndexData: %v", err)
	}
	defer id.Close()

	if id.numDocs() != 1 {
		t.Errorf("got %d docs, want 1", id.numDocs())
	}

	if !bytes.Equal(id.fileName(0), []byte("main.go")) {
		t.Errorf("got filename %q, want main.go", string(id.fileName(0)))
	}
}
