package prompt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func setupTestDB(t *testing.T) (*db.Store, func()) {
	dir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("open db: %v", err)
	}

	store := db.NewStore(database, dbPath)
	return store, func() {
		database.Close()
		os.RemoveAll(dir)
	}
}

func TestSyncBuiltinSkills(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initial sync
	if err := SyncBuiltinSkills(ctx, store); err != nil {
		t.Fatalf("sync skills: %v", err)
	}

	skills, err := store.ListSkills(ctx)
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}

	if len(skills) == 0 {
		t.Error("expected skills to be seeded")
	}

	// Modify one skill
	originalSkill := skills[0]
	originalPrompt := originalSkill.SystemPrompt
	originalSkill.SystemPrompt = "Revised prompt"
	if err := store.UpdateSkill(ctx, &originalSkill); err != nil {
		t.Fatalf("update skill: %v", err)
	}

	// Sync again - should skip modified skill
	if err := SyncBuiltinSkills(ctx, store); err != nil {
		t.Fatalf("sync skills 2: %v", err)
	}

	got, _ := store.GetSkillByName(ctx, originalSkill.Name)
	if got.SystemPrompt != "Revised prompt" {
		t.Errorf("expected revised prompt to be preserved, got %s", got.SystemPrompt)
	}

	_ = originalPrompt
}
