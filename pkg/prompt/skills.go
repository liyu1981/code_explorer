package prompt

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"
)

//go:embed skills/*.md
var embeddedSkills embed.FS

var buildinSkillTags = map[string]string{
	"concise-topic-summarizer": "summarizer",
	"general-researcher":       "researcher",
	"knowledge-base-builder":   "knowledge-builder",
	"knowledge-base-planner":   "knowledge-builder",
}

var buildinSkillTools = map[string]string{
	"concise-topic-summarizer": "",
	"general-researcher":       "codemogger_list_files codemogger_search",
	"knowledge-base-builder":   "codemogger_list_files codemogger_search",
	"knowledge-base-planner":   "read_file get_tree grep",
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
		}

		lines := strings.Split(systemPrompt, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Tags: ") {
				tags = strings.TrimSpace(strings.TrimPrefix(line, "Tags: "))
			}
		}

		// Override tags from map if exists
		if t, ok := buildinSkillTags[name]; ok {
			tags = t
		}

		tools := ""
		if t, ok := buildinSkillTools[name]; ok {
			tools = t
		}

		if existing != nil {
			// Compare existing skill with embedded version
			if existing.SystemPrompt == systemPrompt && existing.Tags == tags && existing.Tools == tools {
				// Skills are equal, skip seeding
				continue
			}

			// Skills are different, create backup by renaming the existing skill
			backupName := fmt.Sprintf("%s_%d", name, time.Now().Unix())
			existing.Name = backupName
			if err := store.UpdateSkill(ctx, existing); err != nil {
				log.Printf("Warning: failed to backup skill %s: %v", name, err)
				continue
			}
			log.Printf("Backed up skill %s as %s", name, backupName)
		}

		// Create or update skill with embedded version
		skill := &db.Skill{
			Name:         name,
			SystemPrompt: systemPrompt,
			Tags:         tags,
			Tools:        tools,
		}

		if existing == nil {
			// Skill doesn't exist, create new
			if err := store.CreateSkill(ctx, skill); err != nil {
				log.Printf("Warning: failed to seed skill %s: %v", name, err)
			} else {
				log.Printf("Seeded built-in skill: %s", name)
			}
		} else {
			// Skill exists but was different, update it
			if err := store.UpdateSkill(ctx, skill); err != nil {
				log.Printf("Warning: failed to update skill %s: %v", name, err)
			} else {
				log.Printf("Updated built-in skill: %s", name)
			}
		}
	}

	return nil
}
