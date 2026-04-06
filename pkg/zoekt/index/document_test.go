package zoekt

import (
	"bytes"
	"testing"
)

func TestSkipReasonExplanation(t *testing.T) {
	tests := []struct {
		reason   SkipReason
		expected string
	}{
		{SkipReasonNone, ""},
		{SkipReasonTooLarge, "exceeds the maximum size limit"},
		{SkipReasonTooSmall, "contains too few trigrams"},
		{SkipReasonBinary, "contains binary content"},
		{SkipReasonTooManyTrigrams, "contains too many trigrams"},
		{SkipReasonMissing, "object missing from repository"},
		{SkipReason(99), "unknown skip reason"},
	}

	for _, tc := range tests {
		if got := tc.reason.explanation(); got != tc.expected {
			t.Errorf("SkipReason(%d).explanation() = %q, want %q", tc.reason, got, tc.expected)
		}
	}
}

func TestDocumentSection(t *testing.T) {
	sec := DocumentSection{Start: 10, End: 20}
	if sec.Start != 10 || sec.End != 20 {
		t.Errorf("DocumentSection = %v, want {Start:10, End:20}", sec)
	}
}

func TestDocumentFields(t *testing.T) {
	doc := Document{
		Name:              "test.go",
		Content:           []byte("package main"),
		Branches:          []string{"main", "develop"},
		SubRepositoryPath: "",
		Language:          "Go",
		SkipReason:        SkipReasonNone,
		Symbols:           []DocumentSection{{Start: 0, End: 6}},
		SymbolsMetaData:   []*Symbol{{Sym: "main", Kind: "func"}},
	}

	if doc.Name != "test.go" {
		t.Errorf("Name = %q, want %q", doc.Name, "test.go")
	}
	if !bytes.Equal(doc.Content, []byte("package main")) {
		t.Errorf("Content = %v, want %v", doc.Content, []byte("package main"))
	}
	if len(doc.Branches) != 2 {
		t.Errorf("Branches len = %d, want 2", len(doc.Branches))
	}
	if doc.Language != "Go" {
		t.Errorf("Language = %q, want %q", doc.Language, "Go")
	}
	if doc.SkipReason != SkipReasonNone {
		t.Errorf("SkipReason = %v, want SkipReasonNone", doc.SkipReason)
	}
	if len(doc.Symbols) != 1 {
		t.Errorf("Symbols len = %d, want 1", len(doc.Symbols))
	}
	if len(doc.SymbolsMetaData) != 1 {
		t.Errorf("SymbolsMetaData len = %d, want 1", len(doc.SymbolsMetaData))
	}
}
