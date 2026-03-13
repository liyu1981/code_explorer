package db

import (
	"context"
	"testing"
)

func TestCodemoggerStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	cb, _ := store.GetOrCreateCodebase(ctx, "/test", "test", "local")
	metadataID, err := store.CodemoggerEnsureMetadata(ctx, cb.ID)
	if err != nil {
		t.Fatalf("EnsureMetadata: %v", err)
	}

	// Test Touch
	if err := store.CodemoggerTouchCodebase(ctx, metadataID); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	// Test File Hash
	hash, err := store.CodemoggerGetFileHash(ctx, metadataID, "main.go")
	if err != nil {
		t.Fatalf("GetFileHash: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash, got %s", hash)
	}

	// Test Upsert Chunks
	chunks := []struct {
		FilePath string
		FileHash string
		Chunks   []CodeChunk
	}{
		{
			FilePath: "main.go",
			FileHash: "abc",
			Chunks: []CodeChunk{
				{ChunkKey: "main.go:1:5", FilePath: "main.go", Name: "main", Kind: "function"},
			},
		},
	}
	if err := store.CodemoggerBatchUpsertAllFileChunks(ctx, metadataID, chunks); err != nil {
		t.Fatalf("BatchUpsert: %v", err)
	}

	hash, _ = store.CodemoggerGetFileHash(ctx, metadataID, "main.go")
	if hash != "abc" {
		t.Errorf("expected hash abc, got %s", hash)
	}

	// Test List Files
	files, err := store.CodemoggerListFiles(ctx, metadataID)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 || files[0].FilePath != "main.go" {
		t.Errorf("expected main.go, got %+v", files)
	}

	// Test Stale Files
	removed, err := store.CodemoggerRemoveStaleFiles(ctx, metadataID, []string{"main.go"})
	if err != nil {
		t.Fatalf("RemoveStaleFiles: %v", err)
	}
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}

	removed, _ = store.CodemoggerRemoveStaleFiles(ctx, metadataID, []string{})
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
}
