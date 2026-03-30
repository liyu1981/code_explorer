package workflow

import (
	"context"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

type SimpleWorkflowRunner struct {
	llm llm.LLM
}

func NewSimpleWorkflowRunner(ai llm.LLM) *SimpleWorkflowRunner {
	return &SimpleWorkflowRunner{
		llm: ai,
	}
}

func (s *SimpleWorkflowRunner) Run(ctx context.Context, goal string) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: goal},
	}

	response, _, err := s.llm.Generate(ctx, messages, nil, nil)
	if err != nil {
		return "", fmt.Errorf("direct llm: %w", err)
	}
	return response, nil
}
