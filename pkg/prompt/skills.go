package prompt

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"

	"github.com/rs/zerolog/log"
)

//go:embed skills/*.md
var embeddedSkills embed.FS

const MaxSkillBackups = 3

func GetBuiltinSkillNames() (map[string]bool, error) {
	entries, err := fs.ReadDir(embeddedSkills, "skills")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded skills: %w", err)
	}

	builtinNames := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		builtinNames[name] = true
	}
	return builtinNames, nil
}

// SyncBuiltinSkills seeds the database with embedded skill prompts
// Skill md file format:
//
//	tags=researcher
//	tools=codemogger_list_files codemogger_search
//	%%%%
//	You are a general researcher.
//
// tags and tools are parsed from lines before %%%%, everything after '=' is the value
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
			log.Info().Str("name", name).Err(err).Msg("Warning: failed to check existing skill")
			continue
		}

		content, err := fs.ReadFile(embeddedSkills, filepath.Join("skills", entry.Name()))
		if err != nil {
			log.Info().Str("file", entry.Name()).Err(err).Msg("Warning: failed to read embedded skill file")
			continue
		}

		fullContent := string(content)
		systemPrompt := fullContent
		tags := ""
		tools := ""

		// Parse format: tags=xxx\ntools=xxx\n%%%%\nprompt
		if parts := strings.Split(fullContent, "%%%%"); len(parts) > 1 {
			header := parts[0]
			systemPrompt = strings.TrimSpace(strings.Join(parts[1:], "%%%%"))

			for _, line := range strings.Split(header, "\n") {
				if strings.HasPrefix(line, "tags=") {
					tags = strings.TrimSpace(strings.TrimPrefix(line, "tags="))
				} else if strings.HasPrefix(line, "tools=") {
					tools = strings.TrimSpace(strings.TrimPrefix(line, "tools="))
				}
			}
		}

		if existing != nil {
			// Check if existing skill is different from embedded version
			if existing.SystemPrompt == systemPrompt && existing.Tags == tags && existing.Tools == tools {
				// Skills are equal, no action needed
				continue
			}

			// Skill exists in DB but different - backup existing value first, then update with embedded

			// Clean up old backups if we have more than MaxSkillBackups
			backups, err := store.ListSkillsByNamePrefix(ctx, name)
			if err != nil {
				log.Info().Err(err).Str("name", name).Msg("Warning: failed to list backups")
			} else if len(backups) >= MaxSkillBackups {
				// Delete oldest backups (keep only MaxSkillBackups)
				toDelete := len(backups) - MaxSkillBackups + 1
				for i := 0; i < toDelete; i++ {
					if err := store.DeleteSkill(ctx, backups[i].ID); err != nil {
						log.Info().Err(err).Str("backupId", backups[i].ID).Msg("Warning: failed to delete old backup")
					} else {
						log.Info().Str("backupName", backups[i].Name).Msg("Deleted old backup")
					}
				}
			}

			// Create backup of existing skill
			backupName := fmt.Sprintf("%s_%d", name, time.Now().Unix())
			if err := store.CreateSkill(ctx, &db.Skill{
				Name:         backupName,
				SystemPrompt: existing.SystemPrompt,
				Tags:         existing.Tags,
				Tools:        existing.Tools,
			}); err != nil {
				log.Info().Err(err).Str("name", name).Str("backupName", backupName).Msg("Error: failed to backup skill")
			} else {
				log.Info().Str("name", name).Str("backupName", backupName).Msg("Backed up skill")
			}

			// Update the skill with embedded content
			existing.SystemPrompt = systemPrompt
			existing.Tags = tags
			existing.Tools = tools
			if err := store.UpdateSkill(ctx, existing); err != nil {
				log.Info().Err(err).Str("name", name).Msg("Error: failed to update skill")
				continue
			}
			log.Info().Str("name", name).Msg("Updated skill from embedded")
			continue
		}

		// Create skill with embedded version
		if err := store.CreateSkill(ctx, &db.Skill{
			Name:         name,
			SystemPrompt: systemPrompt,
			Tags:         tags,
			Tools:        tools,
		}); err != nil {
			log.Info().Err(err).Str("name", name).Msg("Error: failed to seed skill")
			fmt.Errorf("Error: failed to seed skill")
		}
		log.Info().Str("name", name).Msg("Seeded built-in skill")
	}

	return nil
}
