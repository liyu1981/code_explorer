package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDev(t *testing.T) {
	// Save current env
	orig := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", orig)

	os.Setenv("APP_ENV", "development")
	if !IsDev() {
		t.Errorf("IsDev() expected true, got false")
	}

	os.Setenv("APP_ENV", "production")
	if IsDev() {
		t.Errorf("IsDev() expected false, got true")
	}

	os.Unsetenv("APP_ENV")
	if IsDev() {
		t.Errorf("IsDev() expected false, got true")
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("skipping TestExpandPath: could not determine user home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"/abs/path", "/abs/path"},
		{"rel/path", "rel/path"},
		{"~", home},
		{"~/test", filepath.Join(home, "test")},
		{"~/a/b/c", filepath.Join(home, "a/b/c")},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.expected {
			t.Errorf("ExpandPath(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
