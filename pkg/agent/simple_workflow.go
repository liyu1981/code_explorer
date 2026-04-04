package agent

import (
	"context"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type SimpleWorkflowRunner struct {
	llm llm.LLM
}

func NewSimpleWorkflowRunner(ai llm.LLM) *SimpleWorkflowRunner {
	return &SimpleWorkflowRunner{
		llm: ai,
	}
}

func (s *SimpleWorkflowRunner) Run(ctx context.Context, goal string, stream protocol.IStreamWriter) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: goal},
	}

	var response string
	var err error

	if stream != nil {
		response, _, err = s.llm.GenerateStream(ctx, messages, nil, nil, stream)
	} else {
		response, _, err = s.llm.Generate(ctx, messages, nil, nil)
	}

	if err != nil {
		return "", fmt.Errorf("direct llm: %w", err)
	}
	return response, nil
}
