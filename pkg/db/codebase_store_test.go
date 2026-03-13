package db

import (
	"context"
	"testing"
)

func TestCodebaseStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test GetOrCreate
	cb, err := store.GetOrCreateCodebase(ctx, "/test/path", "test-repo", "git")
	if err != nil {
		t.Fatalf("GetOrCreateCodebase: %v", err)
	}
	if cb.Name != "test-repo" || cb.RootPath != "/test/path" {
		t.Errorf("unexpected codebase: %+v", cb)
	}

	// Test duplicate GetOrCreate
	cb2, err := store.GetOrCreateCodebase(ctx, "/test/path", "other-name", "git")
	if err != nil {
		t.Fatalf("GetOrCreateCodebase duplicate: %v", err)
	}
	if cb2.ID != cb.ID {
		t.Errorf("expected same ID, got %s != %s", cb2.ID, cb.ID)
	}

	// Test List
	codebases, err := store.ListCodebases(ctx)
	if err != nil {
		t.Fatalf("ListCodebases: %v", err)
	}
	if len(codebases) != 1 {
		t.Errorf("expected 1 codebase, got %d", len(codebases))
	}

	// Test GetByID
	got, err := store.GetCodebaseByID(ctx, cb.ID)
	if err != nil {
		t.Fatalf("GetCodebaseByID: %v", err)
	}
	if got.ID != cb.ID {
		t.Errorf("expected ID %s, got %s", cb.ID, got.ID)
	}

	// Test UpdateVersion
	if err := store.UpdateCodebaseVersion(ctx, cb.ID, "v1.0.0"); err != nil {
		t.Fatalf("UpdateCodebaseVersion: %v", err)
	}
	got, _ = store.GetCodebaseByID(ctx, cb.ID)
	if got.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", got.Version)
	}

	// Test Delete
	if err := store.DeleteCodebase(ctx, cb.ID); err != nil {
		t.Fatalf("DeleteCodebase: %v", err)
	}
	got, _ = store.GetCodebaseByID(ctx, cb.ID)
	if got != nil {
		t.Error("expected codebase to be deleted")
	}
}
