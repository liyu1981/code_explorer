//go:build integration

package agent

import (
	"context"
	"strings"
	"testing"
)

func TestLLMIntegration(t *testing.T) {
	baseURL, model, apiKey := GetIntegrationTestParams()

	llm := newHTTPClientLLM(model, baseURL, apiKey)

	t.Run("Simple Text Generation", func(t *testing.T) {
		messages := []Message{
			{Role: "user", Content: "Say 'hello world' and nothing else."},
		}

		content, toolCalls, err := llm.Generate(context.Background(), messages, nil, nil)
		if err != nil {
			t.Fatalf("LLM Generate failed: %v", err)
		}

		if content == "" {
			t.Fatal("Expected non-empty content")
		}

		t.Logf("Content: %s", content)
		t.Logf("Tool calls: %d", len(toolCalls))
	})

	t.Run("Text Generation With Tools", func(t *testing.T) {
		messages := []Message{
			{Role: "user", Content: "What is 2 + 2? Use the calculate tool."},
		}

		tools := []map[string]any{
			{
				"type": "function",
				"function": map[string]any{
					"name":        "calculate",
					"description": "Performs basic arithmetic",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"operation": map[string]any{
								"type":        "string",
								"description": "Operation to perform: add, sub, mul",
								"enum":        []any{"add", "sub", "mul"},
							},
							"a": map[string]any{"type": "integer", "description": "First number"},
							"b": map[string]any{"type": "integer", "description": "Second number"},
						},
						"required": []string{"operation", "a", "b"},
					},
				},
			},
		}

		content, toolCalls, err := llm.Generate(context.Background(), messages, tools, nil)
		if err != nil {
			t.Fatalf("LLM Generate failed: %v", err)
		}

		t.Logf("Content: %s", content)
		t.Logf("Tool calls: %d", len(toolCalls))

		if len(toolCalls) == 0 {
			t.Log("Note: Model did not call any tools (expected for some models)")
		} else {
			for _, tc := range toolCalls {
				t.Logf("Tool called: %s with args: %s", tc.Name, string(tc.Input))
				if tc.Name != "calculate" {
					t.Errorf("Expected tool 'calculate', got '%s'", tc.Name)
				}
			}
		}
	})

	t.Run("Streaming Text Generation", func(t *testing.T) {
		messages := []Message{
			{Role: "user", Content: "Count from 1 to 3."},
		}

		var sb strings.Builder
		mockWriter := &mockStreamWriter{builder: &sb}

		content, toolCalls, err := llm.GenerateStream(context.Background(), messages, nil, nil, mockWriter)
		if err != nil {
			t.Fatalf("LLM GenerateStream failed: %v", err)
		}

		if content == "" {
			t.Fatal("Expected non-empty content from streaming")
		}

		streamed := sb.String()
		if streamed == "" {
			t.Log("Note: No streaming output captured")
		} else {
			t.Logf("Streamed content: %s", streamed)
		}

		t.Logf("Final content: %s", content)
		t.Logf("Tool calls: %d", len(toolCalls))
	})
}

func TestLLMNoThinkIntegration(t *testing.T) {
	baseURL, model, apiKey := GetIntegrationTestParams()

	llmNoThink := newHTTPClientLLM(model, baseURL, apiKey)
	llmNoThink.SetNoThink(true)

	messages := []Message{
		{Role: "user", Content: "What is 1+1?"},
	}

	content, toolCalls, err := llmNoThink.Generate(context.Background(), messages, nil, nil)
	if err != nil {
		t.Fatalf("LLM Generate with noThink failed: %v", err)
	}

	if content == "" {
		t.Fatal("Expected non-empty content")
	}

	t.Logf("Content (noThink): %s", content)
	t.Logf("Tool calls: %d", len(toolCalls))
}
