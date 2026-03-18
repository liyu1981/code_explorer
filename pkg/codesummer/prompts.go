package codesummer

import (
	"strings"
)

const (
	AgentNameFile      = "codesummer-file-summarizer"
	AgentNameDirectory = "codesummer-directory-summarizer"
)

func BuildFileSummerizerPrompt(promptTpl string, language string, content string, definitions []Definition) string {
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

	prompt := promptTpl
	prompt = strings.ReplaceAll(prompt, "{language}", language)
	prompt = strings.ReplaceAll(prompt, "{content}", content)
	prompt = strings.ReplaceAll(prompt, "{definitions}", defStr)

	return prompt
}

func BuildDirectorySummerizerPrompt(promptTpl string, dirPath string, childrenSummaries []NodeSummary) string {
	childrenStr := "Children:\n"
	for _, child := range childrenSummaries {
		childrenStr += child.Path + " (" + child.Type + "): " + child.Summary + "\n"
	}

	prompt := promptTpl
	prompt = strings.ReplaceAll(prompt, "{dir_path}", dirPath)
	prompt = strings.ReplaceAll(prompt, "{children_summaries}", childrenStr)

	return prompt
}
