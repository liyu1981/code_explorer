package db

import (
	"testing"
)

func TestCodemoggerStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Need a codebase
	cb, _ := store.GetOrCreateCodebase("/test", "test", "local")

	// Test Ensure Metadata
	metadataID, err := store.CodemoggerEnsureMetadata(cb.ID)
	if err != nil {
		t.Fatalf("ensure metadata: %v", err)
	}
	if metadataID == "" {
		t.Fatal("expected metadata id to be set")
	}

	// Test List Codebases
	list, err := store.CodemoggerListCodebases()
	if err != nil {
		t.Fatalf("list codebases: %v", err)
	}
	if len(list) != 1 || list[0].ID != cb.ID {
		t.Errorf("expected 1 codebase with id %s, got %v", cb.ID, list)
	}

	// Test Batch Upsert
	fileChunks := []struct {
		FilePath string
		FileHash string
		Chunks   []CodeChunk
	}{
		{
			FilePath: "main.go",
			FileHash: "hash1",
			Chunks: []CodeChunk{
				{
					FilePath: "main.go",
					ChunkKey: "main.go:1",
					Language: "go",
					Kind:     "function",
					Name:     "main",
					Snippet:  "func main() {}",
				},
			},
		},
	}
	if err := store.CodemoggerBatchUpsertAllFileChunks(metadataID, fileChunks); err != nil {
		t.Fatalf("batch upsert: %v", err)
	}

	// Test Get File Hash
	hash, err := store.CodemoggerGetFileHash(metadataID, "main.go")
	if err != nil {
		t.Fatalf("get file hash: %v", err)
	}
	if hash != "hash1" {
		t.Errorf("expected hash1, got %s", hash)
	}

	// Test List Files
	files, err := store.CodemoggerListFiles(metadataID)
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files) != 1 || files[0].FilePath != "main.go" {
		t.Errorf("expected main.go, got %v", files)
	}

	// Test Stale Files
	removed, err := store.CodemoggerRemoveStaleFiles(metadataID, []string{"main.go"})
	if err != nil {
		t.Fatalf("remove stale files (active): %v", err)
	}
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}

	removed, err = store.CodemoggerRemoveStaleFiles(metadataID, []string{})
	if err != nil {
		t.Fatalf("remove stale files (none): %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
}
