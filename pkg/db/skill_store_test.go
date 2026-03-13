package db

import (
	"context"
	"testing"
)

func TestSkillStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	skill := &Skill{
		Name:         "test-skill",
		Description:  "test description",
		SystemPrompt: "test prompt",
		Tags:         "test tags",
		IsBuiltin:    false,
	}

	// Test Create
	if err := store.CreateSkill(ctx, skill); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	// Test Get
	got, err := store.GetSkillByName(ctx, "test-skill")
	if err != nil {
		t.Fatalf("get skill: %v", err)
	}
	if got.Name != "test-skill" {
		t.Errorf("expected test-skill, got %s", got.Name)
	}

	// Test Update
	skill.Tags = "go devops"
	if err := store.UpdateSkill(ctx, skill); err != nil {
		t.Fatalf("update skill: %v", err)
	}
	got, _ = store.GetSkillByName(ctx, "test-skill")
	if got.Tags != "go devops" {
		t.Errorf("expected updated tags, got %s", got.Tags)
	}

	// Test List
	skills, err := store.ListAgentSkills(ctx)
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
	got, _ = store.GetSkillByName(ctx, "test-skill")
	if got != nil {
		t.Error("expected skill to be deleted")
	}
}
