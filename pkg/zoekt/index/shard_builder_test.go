package index

import (
	"bytes"
	"testing"
)

func TestPostingsBuilderBasic(t *testing.T) {
	pb := newPostingsBuilder(100 << 20)
	if pb == nil {
		t.Fatal("newPostingsBuilder returned nil")
	}
	if pb.postings == nil {
		t.Error("postings map not initialized")
	}
	if !pb.isPlainASCII {
		t.Error("isPlainASCII should be true initially")
	}
}

func TestPostingsBuilderReset(t *testing.T) {
	pb := newPostingsBuilder(100 << 20)

	// Add some data
	_, _, err := pb.newSearchableString([]byte("hello world"), nil)
	if err != nil {
		t.Fatalf("newSearchableString failed: %v", err)
	}

	// Reset
	pb.reset()

	if pb.runeCount != 0 {
		t.Errorf("runeCount after reset = %d, want 0", pb.runeCount)
	}
	if pb.endByte != 0 {
		t.Errorf("endByte after reset = %d, want 0", pb.endByte)
	}
}

func TestPostingsBuilderSearchableString(t *testing.T) {
	pb := newPostingsBuilder(100 << 20)

	// Test with simple ASCII
	ss, secs, err := pb.newSearchableString([]byte("hello"), nil)
	if err != nil {
		t.Fatalf("newSearchableString failed: %v", err)
	}
	if ss == nil {
		t.Fatal("searchableString is nil")
	}
	if !bytes.Equal(ss.data, []byte("hello")) {
		t.Errorf("searchableString.data = %v, want %v", ss.data, []byte("hello"))
	}
	// No sections provided, so secs should be empty
	if len(secs) != 0 {
		t.Errorf("sections = %v, want empty", secs)
	}
}

func TestPostingsBuilderSearchableStringWithSections(t *testing.T) {
	pb := newPostingsBuilder(100 << 20)

	content := []byte("func main() {}")
	sections := []DocumentSection{{Start: 5, End: 9}} // "main"

	_, secs, err := pb.newSearchableString(content, sections)
	if err != nil {
		t.Fatalf("newSearchableString failed: %v", err)
	}
	// Should have converted byte sections to rune sections
	if len(secs) != len(sections) {
		t.Errorf("secs len = %d, want %d", len(secs), len(sections))
	}
}

func TestPostingsBuilderNonASCII(t *testing.T) {
	pb := newPostingsBuilder(100 << 20)

	// Test with non-ASCII (Chinese characters)
	_, _, err := pb.newSearchableString([]byte("你好世界"), nil)
	if err != nil {
		t.Fatalf("newSearchableString failed: %v", err)
	}

	if pb.isPlainASCII {
		t.Error("isPlainASCII should be false after adding non-ASCII")
	}
}

func TestDocCheckerCheck(t *testing.T) {
	checker := DocChecker{}

	tests := []struct {
		content         []byte
		maxTrigramCount int
		expected        SkipReason
	}{
		{[]byte{}, 10000, SkipReasonNone},
		{[]byte("ab"), 10000, SkipReasonTooSmall},
		{[]byte("hello world"), 10000, SkipReasonNone},
		// Null byte at position 0 doesn't trigger binary check (only position > 0)
		{[]byte{0, 1, 2, 3}, 10000, SkipReasonNone},
		{[]byte("abc\x00def"), 10000, SkipReasonBinary}, // null at position 3
	}

	for _, tc := range tests {
		result := checker.Check(tc.content, tc.maxTrigramCount, false)
		if result != tc.expected {
			t.Errorf("Check(%v, %d) = %v, want %v",
				tc.content, tc.maxTrigramCount, result, tc.expected)
		}
	}
}

func TestDocCheckerCheckTooManyTrigrams(t *testing.T) {
	checker := DocChecker{}

	content := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 100)

	result := checker.Check(content, 10, false)
	if result != SkipReasonTooManyTrigrams {
		t.Errorf("Check with too many trigrams = %v, want SkipReasonTooManyTrigrams", result)
	}
}

func TestDocCheckerCheckAllowLargeFile(t *testing.T) {
	checker := DocChecker{}

	content := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 100)

	result := checker.Check(content, 10, true)
	if result != SkipReasonNone {
		t.Errorf("Check with allowLargeFile = %v, want SkipReasonNone", result)
	}
}

func TestDocCheckerClearTrigrams(t *testing.T) {
	checker := DocChecker{}

	// Use content that exceeds maxTrigramCount to ensure the trigrams map is populated
	// 2600 bytes -> 2598 trigrams upper bound, with maxTrigramCount=10000, it returns early
	// Use more content or smaller maxTrigramCount to force trigram counting
	content := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 1000)
	_ = checker.Check(content, 100, false) // Use smaller threshold to force counting

	if checker.trigrams == nil {
		t.Fatal("trigrams map not initialized after Check")
	}

	checker.clearTrigrams(100)

	if len(checker.trigrams) != 0 {
		t.Errorf("trigrams after clear = %d, want 0", len(checker.trigrams))
	}
}

func TestShardBuilderBasic(t *testing.T) {
	sb := newShardBuilder(100 << 20)
	if sb == nil {
		t.Fatal("newShardBuilder returned nil")
	}
	if sb.contentPostings == nil {
		t.Error("contentPostings not initialized")
	}
	if sb.namePostings == nil {
		t.Error("namePostings not initialized")
	}
	if sb.symIndex == nil {
		t.Error("symIndex not initialized")
	}
	if sb.languageMap == nil {
		t.Error("languageMap not initialized")
	}
}

func TestShardBuilderContentSize(t *testing.T) {
	sb := newShardBuilder(100 << 20)

	if sb.ContentSize() != 0 {
		t.Error("ContentSize should be 0 for empty builder")
	}

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main"),
		Branches: []string{"main"},
	}
	if err := sb.Add(doc); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if sb.ContentSize() == 0 {
		t.Error("ContentSize should not be 0 after adding document")
	}
}

func TestShardBuilderNumFiles(t *testing.T) {
	sb := newShardBuilder(100 << 20)

	if sb.NumFiles() != 0 {
		t.Error("NumFiles should be 0 for empty builder")
	}

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main"),
		Branches: []string{"main"},
	}
	sb.Add(doc)

	if sb.NumFiles() != 1 {
		t.Errorf("NumFiles = %d, want 1", sb.NumFiles())
	}
}

func TestShardBuilderAddWithSkipReason(t *testing.T) {
	sb := newShardBuilder(100 << 20)

	doc := Document{
		Name:       "test.bin",
		Content:    []byte{0, 1, 2, 0, 1, 2},
		Branches:   []string{"main"},
		SkipReason: SkipReasonBinary,
	}
	err := sb.Add(doc)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if sb.NumFiles() != 1 {
		t.Errorf("NumFiles = %d, want 1", sb.NumFiles())
	}
}

func TestShardBuilderAddWithBranches(t *testing.T) {
	sb := newShardBuilder(100 << 20)

	repo := Repository{
		Branches: []RepositoryBranch{
			{Name: "main"},
			{Name: "develop"},
		},
	}
	sb.setRepository(&repo)

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main"),
		Branches: []string{"main", "develop"},
	}
	err := sb.Add(doc)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if len(sb.branchMasks) != 1 {
		t.Errorf("branchMasks len = %d, want 1", len(sb.branchMasks))
	}
}

func TestShardBuilderAddInvalidBranch(t *testing.T) {
	sb := newShardBuilder(100 << 20)

	repo := Repository{
		Branches: []RepositoryBranch{
			{Name: "main"},
		},
	}
	sb.setRepository(&repo)

	doc := Document{
		Name:     "test.go",
		Content:  []byte("package main"),
		Branches: []string{"nonexistent"},
	}
	err := sb.Add(doc)
	if err == nil {
		t.Error("Add should fail with nonexistent branch")
	}
}

func TestSymbolSlice(t *testing.T) {
	syms := []DocumentSection{{Start: 20, End: 30}, {Start: 0, End: 10}, {Start: 10, End: 20}}
	metas := []*Symbol{{Sym: "b"}, {Sym: "a"}, {Sym: "c"}}

	slice := symbolSlice{symbols: syms, metaData: metas}

	if slice.Len() != 3 {
		t.Errorf("Len = %d, want 3", slice.Len())
	}

	slice.Swap(0, 2)
	if slice.symbols[0].Start != 10 || slice.symbols[2].Start != 20 {
		t.Error("Swap did not work correctly")
	}
}

func TestMkSubRepoIndices(t *testing.T) {
	repo := Repository{
		SubRepoMap: map[string]*Repository{
			"sub1": {},
			"sub2": {},
		},
	}

	indices := mkSubRepoIndices(repo)

	if len(indices) != 3 {
		t.Errorf("len = %d, want 3", len(indices))
	}

	if idx, ok := indices[""]; !ok || idx != 0 {
		t.Error("empty string should map to 0")
	}
}
