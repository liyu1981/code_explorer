package index

import (
	"testing"
)

func TestOptionsSetDefaults(t *testing.T) {
	opts := Options{}
	opts.SetDefaults()

	if opts.Parallelism != 4 {
		t.Errorf("Parallelism = %d, want 4", opts.Parallelism)
	}
	if opts.SizeMax != 2<<20 {
		t.Errorf("SizeMax = %d, want %d", opts.SizeMax, 2<<20)
	}
	if opts.ShardMax != 100<<20 {
		t.Errorf("ShardMax = %d, want %d", opts.ShardMax, 100<<20)
	}
	if opts.TrigramMax != 20000 {
		t.Errorf("TrigramMax = %d, want 20000", opts.TrigramMax)
	}
}

func TestOptionsSetDefaultsAlreadySet(t *testing.T) {
	opts := Options{
		Parallelism: 8,
		SizeMax:     1024,
		ShardMax:    2048,
		TrigramMax:  5000,
	}
	opts.SetDefaults()

	if opts.Parallelism != 8 {
		t.Errorf("Parallelism = %d, want 8", opts.Parallelism)
	}
	if opts.SizeMax != 1024 {
		t.Errorf("SizeMax = %d, want 1024", opts.SizeMax)
	}
	if opts.ShardMax != 2048 {
		t.Errorf("ShardMax = %d, want 2048", opts.ShardMax)
	}
	if opts.TrigramMax != 5000 {
		t.Errorf("TrigramMax = %d, want 5000", opts.TrigramMax)
	}
}

func TestIgnoreSizeMax(t *testing.T) {
	tests := []struct {
		name       string
		largeFiles []string
		file       string
		expected   bool
	}{
		{
			name:       "no patterns",
			largeFiles: []string{},
			file:       "test.go",
			expected:   false,
		},
		{
			name:       "exact match",
			largeFiles: []string{"test.go"},
			file:       "test.go",
			expected:   true,
		},
		{
			name:       "no match",
			largeFiles: []string{"test.go"},
			file:       "other.go",
			expected:   false,
		},
		{
			name:       "negated pattern",
			largeFiles: []string{"!test.go"},
			file:       "test.go",
			expected:   false,
		},
		{
			name:       "negated then positive - matches first",
			largeFiles: []string{"!test.go", "other.go"},
			file:       "test.go",
			expected:   false,
		},
	}

	for _, tc := range tests {
		opts := Options{LargeFiles: tc.largeFiles}
		result := opts.IgnoreSizeMax(tc.file)
		if result != tc.expected {
			t.Errorf("IgnoreSizeMax(%q) with patterns %v = %v, want %v",
				tc.file, tc.largeFiles, result, tc.expected)
		}
	}
}

func TestIndexState(t *testing.T) {
	states := []IndexState{
		IndexStateMissing,
		IndexStateCorrupt,
		IndexStateVersion,
		IndexStateOption,
		IndexStateMeta,
		IndexStateContent,
		IndexStateEqual,
	}

	expectedStrings := []string{
		"missing",
		"corrupt",
		"version-mismatch",
		"option-mismatch",
		"meta-mismatch",
		"content-mismatch",
		"equal",
	}

	for i, state := range states {
		if string(state) != expectedStrings[i] {
			t.Errorf("IndexState(%d) = %q, want %q", i, state, expectedStrings[i])
		}
	}
}
