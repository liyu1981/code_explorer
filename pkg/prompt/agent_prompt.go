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

//go:embed agent_prompts/*.md
var embeddedAgentPrompts embed.FS

const MaxAgentPromptsBackups = 3

func GetBuiltinPromptNames() (map[string]bool, error) {
	entries, err := fs.ReadDir(embeddedAgentPrompts, "agent_prompts")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded prompts: %w", err)
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

// SyncBuiltinPrompts seeds the database with embedded prompt templates
// Prompt md file format:
//
//	tags=researcher
//	tools=codemogger_list_files codemogger_search
//	%%%%
//	You are a general researcher.
//	%%%%
//	Analyse the following code: {code_snippet}
//
// tags and tools are parsed from lines before the first %%%%, everything after '=' is the value
// the system prompt is between the first and second %%%%
// the user prompt template is after the second %%%% (can be empty)
func SyncBuiltinPrompts(ctx context.Context, store *db.Store) error {
	entries, err := fs.ReadDir(embeddedAgentPrompts, "agent_prompts")
	if err != nil {
		return fmt.Errorf("failed to read embedded prompts: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")

		existing, err := store.GetPromptByName(ctx, name)
		if err != nil {
			log.Info().Str("name", name).Err(err).Msg("Warning: failed to check existing prompt")
			continue
		}

		content, err := fs.ReadFile(embeddedAgentPrompts, filepath.Join("agent_prompts", entry.Name()))
		if err != nil {
			log.Info().Str("file", entry.Name()).Err(err).Msg("Warning: failed to read embedded prompt file")
			continue
		}

		fullContent := string(content)
		systemPrompt := ""
		userPromptTpl := ""
		tags := ""
		tools := ""

		if parts := strings.Split(fullContent, "%%%%"); len(parts) >= 3 {
			header := parts[0]
			systemPrompt = strings.TrimSpace(parts[1])
			userPromptTpl = strings.TrimSpace(parts[2])

			for _, line := range strings.Split(header, "\n") {
				if strings.HasPrefix(line, "tags=") {
					tags = strings.TrimSpace(strings.TrimPrefix(line, "tags="))
				} else if strings.HasPrefix(line, "tools=") {
					tools = strings.TrimSpace(strings.TrimPrefix(line, "tools="))
				}
			}
		}

		if existing != nil {
			if existing.SystemPrompt == systemPrompt && existing.UserPromptTpl == userPromptTpl && existing.Tags == tags && existing.Tools == tools {
				continue
			}

			backups, err := store.ListPromptsByNamePrefix(ctx, name)
			if err != nil {
				log.Info().Err(err).Str("name", name).Msg("Warning: failed to list backups")
			} else if len(backups) >= MaxAgentPromptsBackups {
				toDelete := len(backups) - MaxAgentPromptsBackups + 1
				for i := 0; i < toDelete; i++ {
					if err := store.DeletePrompt(ctx, backups[i].ID); err != nil {
						log.Info().Err(err).Str("backupId", backups[i].ID).Msg("Warning: failed to delete old backup")
					} else {
						log.Info().Str("backupName", backups[i].Name).Msg("Deleted old backup")
					}
				}
			}

			backupName := fmt.Sprintf("%s_%d", name, time.Now().Unix())
			if err := store.CreatePrompt(ctx, &db.Prompt{
				Name:          backupName,
				SystemPrompt:  existing.SystemPrompt,
				UserPromptTpl: existing.UserPromptTpl,
				Tags:          existing.Tags,
				Tools:         existing.Tools,
			}); err != nil {
				log.Info().Err(err).Str("name", name).Str("backupName", backupName).Msg("Error: failed to backup prompt")
			} else {
				log.Info().Str("name", name).Str("backupName", backupName).Msg("Backed up prompt")
			}

			existing.SystemPrompt = systemPrompt
			existing.UserPromptTpl = userPromptTpl
			existing.Tags = tags
			existing.Tools = tools
			if err := store.UpdatePrompt(ctx, existing); err != nil {
				log.Info().Err(err).Str("name", name).Msg("Error: failed to update prompt")
				continue
			}
			log.Info().Str("name", name).Msg("Updated prompt from embedded")
			continue
		}

		if err := store.CreatePrompt(ctx, &db.Prompt{
			Name:          name,
			SystemPrompt:  systemPrompt,
			UserPromptTpl: userPromptTpl,
			Tags:          tags,
			Tools:         tools,
		}); err != nil {
			log.Fatal().Err(err).Str("name", name).Msg("Error: failed to seed prompt")
		}
		log.Info().Str("name", name).Msg("Seeded built-in prompt")
	}

	return nil
}
