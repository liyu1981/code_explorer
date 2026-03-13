package agent

import (
	"context"
	"testing"
)

func TestMockLLM(t *testing.T) {
	responses := []string{"Response 1", "Response 2"}
	toolCalls := [][]ToolCall{
		{{ID: "call-1", Name: "tool-1"}},
		{},
	}
	llm := NewMockLLM("mock-model", responses, toolCalls)

	ctx := context.Background()

	// Call 1
	resp, tcs, err := llm.Generate(ctx, nil, nil, nil)
	if err != nil {
		t.Fatalf("Generate 1 failed: %v", err)
	}
	if resp != "Response 1" {
		t.Errorf("expected Response 1, got %q", resp)
	}
	if len(tcs) != 1 || tcs[0].Name != "tool-1" {
		t.Errorf("unexpected tool calls: %+v", tcs)
	}

	// Call 2
	resp, tcs, err = llm.Generate(ctx, nil, nil, nil)
	if err != nil {
		t.Fatalf("Generate 2 failed: %v", err)
	}
	if resp != "Response 2" {
		t.Errorf("expected Response 2, got %q", resp)
	}
	if len(tcs) != 0 {
		t.Errorf("expected no tool calls, got %d", len(tcs))
	}

	// Call 3 (exhausted)
	resp, tcs, err = llm.Generate(ctx, nil, nil, nil)
	if err != nil {
		t.Fatalf("Generate 3 failed: %v", err)
	}
	if resp != "" {
		t.Errorf("expected empty response, got %q", resp)
	}
}
