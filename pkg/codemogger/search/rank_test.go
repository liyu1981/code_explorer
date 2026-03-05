package search

import (
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func TestRRFMerge(t *testing.T) {
	ftsResults := []db.SearchResult{
		{ChunkKey: "A", Name: "Alpha"},
		{ChunkKey: "B", Name: "Beta"},
	}
	vecResults := []db.SearchResult{
		{ChunkKey: "B", Name: "Beta"},
		{ChunkKey: "C", Name: "Gamma"},
	}

	merged := RRFMerge(ftsResults, vecResults, 10, 60, 1.0, 1.0)

	if len(merged) != 3 {
		t.Errorf("Expected 3 results, got %d", len(merged))
	}

	// B is in both, so it should be rank 1
	if merged[0].ChunkKey != "B" {
		t.Errorf("Expected rank 1 to be B, got %s", merged[0].ChunkKey)
	}

	// Verify scores are set
	for _, r := range merged {
		if r.Score == 0 {
			t.Errorf("Score not set for %s", r.ChunkKey)
		}
	}
}

func TestPreprocessQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"The quick brown fox", "quick brown fox"},
		{"How to do it in Go", "how it go"},
		{"A and B or C", "b c"}, // "a", "and", "or" are stopwords
		{"   Multiple   Spaces   ", "multiple spaces"},
	}

	for _, tt := range tests {
		got := PreprocessQuery(tt.query)
		if got != tt.expected {
			t.Errorf("PreprocessQuery(%q) = %q, want %q", tt.query, got, tt.expected)
		}
	}
}

func TestPreprocessQueryAllStopwords(t *testing.T) {
	query := "The a an and or"
	got := PreprocessQuery(query)
	if got != "" {
		t.Errorf("Expected empty string for all stopwords, got %q", got)
	}
}
