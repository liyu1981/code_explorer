package db

import (
	"context"
	"testing"
)

func TestSkillStore(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test Create
	skill := &Skill{
		Name:         "go-expert",
		Description:  "Expert in Go",
		SystemPrompt: "You are an expert in Go.",
		IsBuiltin:    true,
	}
	if err := store.CreateSkill(ctx, skill); err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if skill.ID == "" {
		t.Error("expected ID to be set")
	}

	// Test Get by Name
	got, err := store.GetSkillByName(ctx, "go-expert")
	if err != nil {
		t.Fatalf("get skill: %v", err)
	}
	if got == nil || got.Description != "Expert in Go" {
		t.Errorf("expected description 'Expert in Go', got %v", got)
	}

	// Test Update
	skill.Description = "Deep expert in Go"
	if err := store.UpdateSkill(ctx, skill); err != nil {
		t.Fatalf("update skill: %v", err)
	}

	got, _ = store.GetSkillByName(ctx, "go-expert")
	if got.Description != "Deep expert in Go" {
		t.Errorf("expected updated description, got %s", got.Description)
	}

	// Test List
	skills, err := store.ListSkills(ctx)
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}

	// Test Delete
	if err := store.DeleteSkill(ctx, skill.ID); err != nil {
		t.Fatalf("delete skill: %v", err)
	}

	got, _ = store.GetSkillByName(ctx, "go-expert")
	if got != nil {
		t.Error("expected skill to be deleted")
	}
}
