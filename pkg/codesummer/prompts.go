package codesummer

import (
	"context"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/db"
)

const (
	SkillNameSystem    = "codesummer-system"
	SkillNameFile      = "codesummer-file-summarizer"
	SkillNameDirectory = "codesummer-directory-summarizer"
)

type PromptLoader struct {
	store *db.Store
}

func NewPromptLoader(store *db.Store) *PromptLoader {
	return &PromptLoader{store: store}
}

func (p *PromptLoader) LoadSystemPrompt(ctx context.Context) (string, error) {
	skill, err := p.store.GetPromptByName(ctx, SkillNameSystem)
	if err != nil {
		return "", err
	}
	if skill != nil {
		return skill.SystemPrompt, nil
	}
	return "", nil
}

func (p *PromptLoader) LoadFilePrompt(ctx context.Context) (string, error) {
	skill, err := p.store.GetPromptByName(ctx, SkillNameFile)
	if err != nil {
		return "", err
	}
	if skill != nil {
		return skill.SystemPrompt, nil
	}
	return "", nil
}

func (p *PromptLoader) LoadDirectoryPrompt(ctx context.Context) (string, error) {
	skill, err := p.store.GetPromptByName(ctx, SkillNameDirectory)
	if err != nil {
		return "", err
	}
	if skill != nil {
		return skill.SystemPrompt, nil
	}
	return "", nil
}

type PromptBuilder struct {
	SystemPrompt    string
	FilePrompt      string
	DirectoryPrompt string
}

func NewPromptBuilder(ctx context.Context, store *db.Store) (*PromptBuilder, error) {
	loader := NewPromptLoader(store)

	systemPrompt, err := loader.LoadSystemPrompt(ctx)
	if err != nil {
		return nil, err
	}

	filePrompt, err := loader.LoadFilePrompt(ctx)
	if err != nil {
		return nil, err
	}

	dirPrompt, err := loader.LoadDirectoryPrompt(ctx)
	if err != nil {
		return nil, err
	}

	return &PromptBuilder{
		SystemPrompt:    systemPrompt,
		FilePrompt:      filePrompt,
		DirectoryPrompt: dirPrompt,
	}, nil
}

func (b *PromptBuilder) BuildFilePrompt(language string, content string, definitions []Definition) string {
	var defStr string
	if len(definitions) > 0 {
		defStr = "Extracted definitions:\n"
		for _, def := range definitions {
			defStr += def.Kind + ": " + def.Name
			if def.Signature != "" {
				defStr += " - " + def.Signature
			}
			defStr += "\n"
		}
	}

	prompt := b.FilePrompt
	prompt = strings.ReplaceAll(prompt, "{language}", language)
	prompt = strings.ReplaceAll(prompt, "{content}", content)
	prompt = strings.ReplaceAll(prompt, "{definitions}", defStr)

	return prompt
}

func (b *PromptBuilder) BuildDirectoryPrompt(dirPath string, childrenSummaries []NodeSummary) string {
	childrenStr := "Children:\n"
	for _, child := range childrenSummaries {
		childrenStr += child.Path + " (" + child.Type + "): " + child.Summary + "\n"
	}

	prompt := b.DirectoryPrompt
	prompt = strings.ReplaceAll(prompt, "{dir_path}", dirPath)
	prompt = strings.ReplaceAll(prompt, "{children_summaries}", childrenStr)

	return prompt
}
