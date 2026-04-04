package codesummer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/llm"
)

const MaxContextLength = 100000

type Summarizer struct {
	store           *db.Store
	responseFormat  *llm.ResponseFormat
	promptTemplates map[string]string
	summarizerLLM   *SummarizerLLM
}

func NewSummarizer(agentPromptName string, store *db.Store, ai llm.LLM) (*Summarizer, error) {
	rf, err := llm.ResponseFormatFromStruct[FileSummaryResponse]("file_summary")
	if err != nil {
		return nil, err
	}

	promptTemplates, err := loadPromptTemplates(agentPromptName)
	if err != nil {
		return nil, err
	}

	return &Summarizer{
		store:           store,
		responseFormat:  rf,
		promptTemplates: promptTemplates,
		summarizerLLM:   NewSummarizerLLM(ai),
	}, nil
}

func loadPromptTemplates(name string) (map[string]string, error) {
	prompts := make(map[string]string)

	filePromptFile := fmt.Sprintf("prompts/%s_file.md", name)
	if data, err := os.ReadFile(filePromptFile); err == nil {
		prompts["file"] = string(data)
	} else {
		prompts["file"] = defaultFilePrompt
	}

	dirPromptFile := fmt.Sprintf("prompts/%s_directory.md", name)
	if data, err := os.ReadFile(dirPromptFile); err == nil {
		prompts["directory"] = string(data)
	} else {
		prompts["directory"] = defaultDirectoryPrompt
	}

	return prompts, nil
}

var defaultFilePrompt = `You are a code summerizer. Summarize the following {{.Language}} file.

File content:
{{.Content}}

Definitions:
{{range .Definitions}}
- {{.Name}} ({{.Type}}): {{.Description}}
{{end}}

Provide a JSON response with:
{
  "summary": "Brief summary of what this file does",
  "dependencies": ["list of dependencies"],
  "data_manipulated": ["data structures or variables manipulated"],
  "data_flow": {
    "inputs": ["inputs to this code"],
    "outputs": ["outputs from this code"]
  }
}`

var defaultDirectoryPrompt = `You are a code summerizer. Summarize the following directory.

Directory: {{.DirPath}}

Children summaries:
{{range .ChildrenSummaries}}
- {{.Path}}: {{.Summary}}
{{end}}

Provide a JSON response with:
{
  "summary": "Brief summary of what this directory contains",
  "dependencies": ["list of dependencies"],
  "data_manipulated": ["data structures or variables manipulated"],
  "data_flow": {
    "inputs": ["inputs to this directory"],
    "outputs": ["outputs from this directory"]
  }
}`

type SummarizerLLM struct {
	llm llm.LLM
}

func NewSummarizerLLM(ai llm.LLM) *SummarizerLLM {
	return &SummarizerLLM{llm: ai}
}

func (s *SummarizerLLM) Summarize(ctx context.Context, prompt string, rf *llm.ResponseFormat) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	response, _, err := s.llm.Generate(ctx, messages, nil, rf)
	if err != nil {
		return "", err
	}

	return response, nil
}

func (s *Summarizer) SummarizeFile(
	ctx context.Context,
	language string,
	content string,
	definitions []Definition,
) (*NodeSummary, error) {

	tpl, err := template.New("file").Parse(s.promptTemplates["file"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse file prompt template: %w", err)
	}

	var promptBuilder strings.Builder
	err = tpl.Execute(&promptBuilder, map[string]any{
		"Language":    language,
		"Content":     content,
		"Definitions": definitions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute file prompt template: %w", err)
	}

	response, err := s.summarizerLLM.Summarize(ctx, promptBuilder.String(), s.responseFormat)
	if err != nil {
		return nil, err
	}

	var fileSummary FileSummaryResponse
	if err := json.Unmarshal([]byte(response), &fileSummary); err != nil {
		return nil, err
	}

	return &NodeSummary{
		Summary:         fileSummary.Summary,
		Dependencies:    fileSummary.Dependencies,
		DataManipulated: fileSummary.DataManipulated,
		DataFlow: DataFlowInfo{
			Inputs:  fileSummary.DataFlow.Inputs,
			Outputs: fileSummary.DataFlow.Outputs,
		},
	}, nil
}

func (s *Summarizer) SummarizeDirectory(
	ctx context.Context,
	dirPath string,
	childrenSummaries []NodeSummary,
) (*NodeSummary, error) {

	tpl, err := template.New("directory").Parse(s.promptTemplates["directory"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory prompt template: %w", err)
	}

	var promptBuilder strings.Builder
	err = tpl.Execute(&promptBuilder, map[string]any{
		"DirPath":           dirPath,
		"ChildrenSummaries": childrenSummaries,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute directory prompt template: %w", err)
	}

	response, err := s.summarizerLLM.Summarize(ctx, promptBuilder.String(), s.responseFormat)
	if err != nil {
		return nil, err
	}

	var fileSummary FileSummaryResponse
	if err := json.Unmarshal([]byte(response), &fileSummary); err != nil {
		return nil, err
	}

	return &NodeSummary{
		Summary:         fileSummary.Summary,
		Dependencies:    fileSummary.Dependencies,
		DataManipulated: fileSummary.DataManipulated,
		DataFlow: DataFlowInfo{
			Inputs:  fileSummary.DataFlow.Inputs,
			Outputs: fileSummary.DataFlow.Outputs,
		},
	}, nil
}

func (s *Summarizer) SummarizeDirectoryBatch(
	ctx context.Context,
	dir string,
	childrenSummaries []NodeSummary,
) (NodeSummary, error) {
	if totalLength(childrenSummaries) < MaxContextLength {
		summary, err := s.summarizeDirectorySingle(ctx, dir, childrenSummaries)
		if err != nil {
			return NodeSummary{}, err
		}
		return *summary, nil
	}

	batches := partitionByLength(childrenSummaries, MaxContextLength)
	var intermediate []NodeSummary
	for _, batch := range batches {
		summary, err := s.summarizeDirectorySingle(ctx, dir, batch)
		if err != nil {
			return NodeSummary{}, err
		}
		intermediate = append(intermediate, *summary)
	}

	if len(intermediate) > 1 {
		return s.SummarizeDirectoryBatch(ctx, dir+"_merged", intermediate)
	}
	return intermediate[0], nil
}

func (s *Summarizer) summarizeDirectorySingle(
	ctx context.Context,
	dir string,
	childrenSummaries []NodeSummary,
) (*NodeSummary, error) {
	return s.SummarizeDirectory(ctx, dir, childrenSummaries)
}

func totalLength(summaries []NodeSummary) int {
	total := 0
	for _, s := range summaries {
		total += len(s.Path) + len(s.Summary)
	}
	return total
}

func partitionByLength(summaries []NodeSummary, maxLength int) [][]NodeSummary {
	var batches [][]NodeSummary
	var currentBatch []NodeSummary
	currentLength := 0

	for _, s := range summaries {
		itemLength := len(s.Path) + len(s.Summary)
		if currentLength+itemLength > maxLength && len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentLength = 0
		}
		currentBatch = append(currentBatch, s)
		currentLength += itemLength
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}
