package chunk

import (
	"fmt"
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

	if len(chunks) < 3 {
		t.Errorf("Expected at least 3 chunks, got %d", len(chunks))
	}

	foundHello := false
	foundUser := false
	foundGetName := false
	for _, c := range chunks {
		if c.Name == "Hello" && c.Kind == "function" {
			foundHello = true
		}
		if c.Name == "User" && c.Kind == "type" {
			foundUser = true
		}
		if c.Name == "User.GetName" && c.Kind == "method" {
			foundGetName = true
		}
	}
	if !foundHello {
		t.Error("Did not find Hello chunk")
	}
	if !foundUser {
		t.Error("Did not find User chunk")
	}
	if !foundGetName {
		t.Error("Did not find User.GetName chunk")
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

func TestChunkFileTypeScript(t *testing.T) {
	content := `export class MyClass {
  constructor(public x: number) {}
  getX(): number { return this.x; }
}

export function myHelper() {
  return 42;
}
`
	config := Languages["typescript"]
	chunks := ChunkFile("app.ts", content, "hash123", &config)

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	foundClass := false
	foundHelper := false
	for _, c := range chunks {
		if c.Name == "MyClass" && c.Kind == "class" {
			foundClass = true
		}
		if c.Name == "myHelper" && c.Kind == "function" {
			foundHelper = true
		}
	}
	if !foundClass {
		t.Error("Did not find MyClass chunk")
	}
	if !foundHelper {
		t.Error("Did not find myHelper chunk")
	}
}

func TestChunkFileSplitLarge(t *testing.T) {
	// Create a class with many lines to trigger splitting
	methods := ""
	for i := 0; i < 200; i++ {
		methods += fmt.Sprintf("  method%d() { return %d; }\n", i, i)
	}
	content := fmt.Sprintf("class LargeClass {\n%s}", methods)

	config := Languages["javascript"]
	chunks := ChunkFile("large.js", content, "hash123", &config)

	// Since MAX_CHUNK_LINES is 150, LargeClass itself should be split into its methods
	if len(chunks) < 200 {
		t.Errorf("Expected at least 200 chunks (methods), got %d", len(chunks))
	}

	foundMethod0 := false
	for _, c := range chunks {
		if c.Name == "method0" && c.Kind == "function" {
			foundMethod0 = true
			break
		}
	}
	if !foundMethod0 {
		t.Error("Did not find method0 chunk after splitting")
	}
}
