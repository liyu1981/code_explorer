package db

import (
	"context"
	"testing"
)

func TestKnowledgeStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Need a codebase first
	cb, err := store.GetOrCreateCodebase("/test", "test", "local")
	if err != nil {
		t.Fatalf("create codebase: %v", err)
	}

	// Test Create
	page := &KnowledgePage{
		CodebaseID: cb.ID,
		Slug:       "test-slug",
		Title:      "Test Title",
		Content:    "Test Content",
	}
	if err := store.CreateKnowledgePage(ctx, page); err != nil {
		t.Fatalf("create page: %v", err)
	}
	if page.ID == "" {
		t.Error("expected ID to be set")
	}

	// Test Get by Slug
	got, err := store.GetKnowledgePageBySlug(ctx, cb.ID, "test-slug")
	if err != nil {
		t.Fatalf("get page: %v", err)
	}
	if got == nil || got.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %v", got)
	}

	// Test Update
	page.Title = "Updated Title"
	if err := store.UpdateKnowledgePage(ctx, page); err != nil {
		t.Fatalf("update page: %v", err)
	}

	got, _ = store.GetKnowledgePageBySlug(ctx, cb.ID, "test-slug")
	if got.Title != "Updated Title" {
		t.Errorf("expected updated title, got %s", got.Title)
	}

	// Test List
	pages, err := store.ListKnowledgePages(ctx, cb.ID)
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	// Test Delete
	if err := store.DeleteKnowledgePage(ctx, page.ID); err != nil {
		t.Fatalf("delete page: %v", err)
	}

	got, _ = store.GetKnowledgePageBySlug(ctx, cb.ID, "test-slug")
	if got != nil {
		t.Error("expected page to be deleted")
	}
}
