package prompt

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/db"
)

//go:embed skills/*.md
var embeddedSkills embed.FS

var buildinSkillTags = map[string]string{
	"concise_topic_summarizer": "summarizer",
	"general-researcher":       "researcher",
	"knowledge-base-builder":   "knowledge-builder",
	"knowledge-base-planner":   "knowledge-builder",
}

// SyncBuiltinSkills seeds the database with embedded skill prompts
func SyncBuiltinSkills(ctx context.Context, store *db.Store) error {
	entries, err := fs.ReadDir(embeddedSkills, "skills")
	if err != nil {
		return fmt.Errorf("failed to read embedded skills: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")

		// Check if skill already exists in DB
		existing, err := store.GetSkillByName(ctx, name)
		if err != nil {
			log.Printf("Warning: failed to check existing skill %s: %v", name, err)
			continue
		}

		if existing != nil {
			// Skip if already exists to preserve user modifications
			continue
		}

		content, err := fs.ReadFile(embeddedSkills, filepath.Join("skills", entry.Name()))
		if err != nil {
			log.Printf("Warning: failed to read embedded skill file %s: %v", entry.Name(), err)
			continue
		}

		fullContent := string(content)
		systemPrompt := fullContent
		tags := ""

		if parts := strings.Split(fullContent, "\n---\n"); len(parts) > 1 {
			systemPrompt = strings.TrimSpace(parts[0])
			// The part after --- was userPrompt, but we no longer need it.
		}

		description := "Built-in agent skill"
		lines := strings.Split(systemPrompt, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				description = strings.TrimPrefix(line, "# ")
			} else if strings.HasPrefix(line, "Tags: ") {
				tags = strings.TrimSpace(strings.TrimPrefix(line, "Tags: "))
			}
		}

		// Override tags from map if exists
		if t, ok := buildinSkillTags[name]; ok {
			tags = t
		}

		skill := &db.Skill{
			Name:         name,
			Description:  description,
			SystemPrompt: systemPrompt,
			Tags:         tags,
			IsBuiltin:    true,
		}

		if err := store.CreateSkill(ctx, skill); err != nil {
			log.Printf("Warning: failed to seed skill %s: %v", name, err)
		} else {
			log.Printf("Seeded built-in skill: %s", name)
		}
	}

	return nil
}

// ResetSkillToDefault reverts a builtin skill to its embedded prompt
func ResetSkillToDefault(ctx context.Context, name string, store *db.Store) error {
	content, err := fs.ReadFile(embeddedSkills, filepath.Join("skills", name+".md"))
	if err != nil {
		return fmt.Errorf("skill %s not found in built-ins: %w", name, err)
	}

	existing, err := store.GetSkillByName(ctx, name)
	if err != nil {
		return err
	}

	if existing == nil {
		return fmt.Errorf("skill %s not found in database", name)
	}

	fullContent := string(content)
	systemPrompt := fullContent
	tags := ""

	if parts := strings.Split(fullContent, "\n---\n"); len(parts) > 1 {
		systemPrompt = strings.TrimSpace(parts[0])
	}

	existing.SystemPrompt = systemPrompt

	description := "Built-in agent skill"
	lines := strings.Split(systemPrompt, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			description = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "Tags: ") {
			tags = strings.TrimSpace(strings.TrimPrefix(line, "Tags: "))
		}
	}
	existing.Description = description
	existing.Tags = tags

	// Override tags from map if exists
	if t, ok := buildinSkillTags[name]; ok {
		existing.Tags = t
	}

	return store.UpdateSkill(ctx, existing)
}
