package scan

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "walker-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Define files to create
	filesToCreate := map[string]string{
		"main.go":           "package main\n\nfunc main() {}\n",
		"utils.py":          "def add(a, b):\n    return a + b\n",
		"README.md":         "# Test Project\n", // Should be ignored if not in languages
		"ignored.txt":       "This file should be ignored by .gitignore\n",
		"sub/helper.go":     "package sub\n",
		".gitignore":        "*.txt\nREADME.md\n",
		"node_modules/a.js": "console.log('ignored');\n", // gocodewalker usually ignores node_modules
	}

	// Create files and directories
	for path, content := range filesToCreate {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Run ScanDirectory
	scannedFiles, errors := ScanDirectory(tempDir, []string{"go", "python"})

	// Check for errors
	if len(errors) > 0 {
		t.Errorf("ScanDirectory returned errors: %v", errors)
	}

	// Define expected files (absolute paths)
	expectedFiles := map[string]string{
		filepath.Join(tempDir, "main.go"):       filesToCreate["main.go"],
		filepath.Join(tempDir, "utils.py"):      filesToCreate["utils.py"],
		filepath.Join(tempDir, "sub/helper.go"): filesToCreate["sub/helper.go"],
	}

	// Verify scanned files
	if len(scannedFiles) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(scannedFiles))
	}

	foundMap := make(map[string]bool)
	for _, f := range scannedFiles {
		content, ok := expectedFiles[f.AbsPath]
		if !ok {
			t.Errorf("Unexpected file found: %s", f.AbsPath)
			continue
		}
		foundMap[f.AbsPath] = true

		// Check content
		if f.Content != content {
			t.Errorf("Content mismatch for %s", f.AbsPath)
		}

		// Check hash
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
		if f.Hash != expectedHash {
			t.Errorf("Hash mismatch for %s: expected %s, got %s", f.AbsPath, expectedHash, f.Hash)
		}
	}

	for path := range expectedFiles {
		if !foundMap[path] {
			t.Errorf("Expected file not found: %s", path)
		}
	}
}

func TestScanDirectoryLanguageFilter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "walker-lang-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tempDir, "main.py"), []byte("print('hello')"), 0644)

	// Filter only for Go
	scannedFiles, _ := ScanDirectory(tempDir, []string{"go"})
	if len(scannedFiles) != 1 {
		t.Errorf("Expected 1 file (Go), got %d", len(scannedFiles))
	} else if filepath.Base(scannedFiles[0].AbsPath) != "main.go" {
		t.Errorf("Expected main.go, got %s", filepath.Base(scannedFiles[0].AbsPath))
	}
}
