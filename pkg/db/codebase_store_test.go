package db

import (
	"testing"
)

func TestCodebaseStore(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Test GetOrCreate
	cb, err := store.GetOrCreateCodebase("/test", "test", "local")
	if err != nil {
		t.Fatalf("get or create: %v", err)
	}
	if cb.Name != "test" {
		t.Errorf("expected name 'test', got %s", cb.Name)
	}

	// Test GetOrCreate existing
	cb2, err := store.GetOrCreateCodebase("/test", "other", "remote")
	if err != nil {
		t.Fatalf("get or create existing: %v", err)
	}
	if cb2.ID != cb.ID || cb2.Name != "test" {
		t.Errorf("expected same codebase, got %v", cb2)
	}

	// Test List
	list, err := store.ListCodebases()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 codebase, got %d", len(list))
	}

	// Test GetByID
	got, err := store.GetCodebaseByID(cb.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.ID != cb.ID {
		t.Errorf("expected id %s, got %s", cb.ID, got.ID)
	}

	// Test Update Version
	if err := store.UpdateCodebaseVersion(cb.ID, "v1.0.0"); err != nil {
		t.Fatalf("update version: %v", err)
	}
	got, _ = store.GetCodebaseByID(cb.ID)
	if got.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", got.Version)
	}

	// Test Delete
	if err := store.DeleteCodebase(cb.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ = store.GetCodebaseByID(cb.ID)
	if got != nil {
		t.Error("expected codebase to be deleted")
	}
}
