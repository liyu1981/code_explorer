package prompt

import (
	"embed"
	"fmt"
	"io"
)

// FS embeds all markdown files in the current directory.
//
//go:embed *.md
var FS embed.FS

// GetPrompt retrieves the content of an embedded markdown file by its name.
// The name should include the extension, e.g., "research_instruction.md".
func GetPrompt(name string) (string, error) {
	file, err := FS.Open(name)
	if err != nil {
		return "", fmt.Errorf("failed to open prompt %q: %w", name, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt %q: %w", name, err)
	}

	return string(content), nil
}

// AllPrompts returns a list of all embedded prompt filenames.
func AllPrompts() ([]string, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}
