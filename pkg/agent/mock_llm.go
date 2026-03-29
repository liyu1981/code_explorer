package agent

import (
	"context"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type MockLLM struct {
	model     string
	responses []string
	toolCalls []ToolCall
	callCount int
}

func NewMockLLM(model string, responses []string, toolCalls []ToolCall) *MockLLM {
	return &MockLLM{
		model:     model,
		responses: responses,
		toolCalls: toolCalls,
	}
}

func (l *MockLLM) Name() string {
	return l.model
}

func (l *MockLLM) SetNoThink(noThink bool) {}

func (l *MockLLM) Generate(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat) (string, []ToolCall, error) {
	if l.callCount < len(l.responses) {
		response := l.responses[l.callCount]
		l.callCount++
		return response, l.toolCalls, nil
	}
	if len(l.responses) > 0 {
		return l.responses[0], l.toolCalls, nil
	}
	return "", nil, nil
}

func (l *MockLLM) GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat, streamWriter protocol.IStreamWriter) (string, []ToolCall, error) {
	response := ""
	if l.callCount < len(l.responses) {
		response = l.responses[l.callCount]
		l.callCount++
	} else if len(l.responses) > 0 {
		response = l.responses[0]
	}

	for _, ch := range response {
		streamWriter.WriteOpenAIChunk("mock-id", l.model, string(ch), nil)
	}
	streamWriter.WriteDone()

	return response, l.toolCalls, nil
}
