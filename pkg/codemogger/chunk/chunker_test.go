package chunk

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"lib.rs", "rust"},
		{"script.py", "python"},
		{"app.ts", "typescript"},
		{"index.js", "javascript"},
		{"component.tsx", "tsx"},
		{"App.java", "java"},
		{"main.c", "c"},
		{"header.h", "c"},
		{"main.cpp", "cpp"},
		{"unknown.txt", ""},
	}

	for _, tt := range tests {
		lang := DetectLanguage(tt.path)
		if tt.expected == "" {
			if lang != nil {
				t.Errorf("DetectLanguage(%s) = %s, want nil", tt.path, lang.Name)
			}
		} else {
			if lang == nil || lang.Name != tt.expected {
				got := "nil"
				if lang != nil {
					got = lang.Name
				}
				t.Errorf("DetectLanguage(%s) = %s, want %s", tt.path, got, tt.expected)
			}
		}
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := SupportedExtensions()
	if len(exts) == 0 {
		t.Error("SupportedExtensions() returned empty slice")
	}

	foundGo := false
	for _, ext := range exts {
		if ext == ".go" {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Error("SupportedExtensions() did not include .go")
	}
}

func TestChunkFileGo(t *testing.T) {
	content := `package main

import "fmt"

// Hello function
func Hello() {
	fmt.Println("Hello")
}

type User struct {
	Name string
}

func (u *User) GetName() string {
	return u.Name
}
`
	config := Languages["go"]
	chunks := ChunkFile("main.go", content, "hash123", &config)

	// Expected chunks: Hello, User, GetName
	// Note: The current simple chunker might treat "type User struct" as one and "func (u *User) GetName()" as another.
	
	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	foundHello := false
	for _, c := range chunks {
		if c.Name == "Hello" && c.Kind == "function" {
			foundHello = true
			if c.StartLine != 6 {
				t.Errorf("Hello chunk StartLine = %d, want 6", c.StartLine)
			}
		}
	}
	if !foundHello {
		t.Error("Did not find Hello chunk")
	}
}

func TestChunkFilePython(t *testing.T) {
	content := `def add(a, b):
    return a + b

class Calc:
    def sub(self, a, b):
        return a - b
`
	config := Languages["python"]
	chunks := ChunkFile("main.py", content, "hash123", &config)

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	foundAdd := false
	foundCalc := false
	for _, c := range chunks {
		if c.Name == "add" && c.Kind == "function" {
			foundAdd = true
		}
		if c.Name == "Calc" && c.Kind == "class" {
			foundCalc = true
		}
	}
	if !foundAdd {
		t.Error("Did not find add chunk")
	}
	if !foundCalc {
		t.Error("Did not find Calc chunk")
	}
}
