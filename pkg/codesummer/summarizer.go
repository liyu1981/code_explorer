package codesummer

import (
	"context"
	"encoding/json"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/db"
)

const MaxContextLength = 100000

type Summarizer struct {
	store           *db.Store
	agentPromptName string
	responseFormat  *agent.ResponseFormat
}

func NewSummarizer(agentPromptName string, store *db.Store) (*Summarizer, error) {
	rf, err := agent.ResponseFormatFromStruct[FileSummaryResponse]("file_summary")
	if err != nil {
		return nil, err
	}
	return &Summarizer{
		store:           store,
		agentPromptName: agentPromptName,
		responseFormat:  rf,
	}, nil
}

func (s *Summarizer) SummarizeFile(
	ctx context.Context,
	language string,
	content string,
	definitions []Definition,
) (*NodeSummary, error) {
	af := agent.GetAgentFactory()

	a, err := af.BuildFromConfig(ctx, &agent.Config{
		AgentPromptName: s.agentPromptName,
	})
	if err != nil {
		return nil, err
	}

	prompt := BuildFileSummerizerPrompt(a.UserPromptTpl, language, content, definitions)

	response, err := a.RunOnce(ctx, prompt, s.responseFormat, nil)
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
	af := agent.GetAgentFactory()
	a, err := af.BuildFromConfig(ctx, &agent.Config{
		AgentPromptName: s.agentPromptName,
	})
	if err != nil {
		return nil, err
	}

	prompt := BuildDirectorySummerizerPrompt(a.UserPromptTpl, dirPath, childrenSummaries)

	response, err := a.RunOnce(ctx, prompt, s.responseFormat, nil)
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
