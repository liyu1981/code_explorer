package llm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/liyu1981/code_explorer/pkg/tools"
)

type TestUser struct {
	Name  string `json:"name" jsonschema:"The user's full name"`
	Age   int    `json:"age" jsonschema:"Age in years"`
	Email string `json:"email,omitempty" jsonschema:"Contact email address"`
}

func TestResponseFormatFromStruct(t *testing.T) {
	rf, err := ResponseFormatFromStruct[TestUser]("user")
	if err != nil {
		t.Fatalf("ResponseFormatFromStruct failed: %v", err)
	}

	if rf.Type != "json_schema" {
		t.Errorf("expected type json_schema, got %q", rf.Type)
	}

	if rf.JSONSchema == nil {
		t.Fatal("expected JSONSchema to be populated")
	}

	if rf.JSONSchema.Name != "user" {
		t.Errorf("expected name user, got %q", rf.JSONSchema.Name)
	}

	schema := rf.JSONSchema.Schema
	if schema["type"] != "object" {
		t.Errorf("expected schema type object, got %v", schema["type"])
	}

	properties := schema["properties"].(map[string]any)
	if _, ok := properties["name"]; !ok {
		t.Error("expected property 'name' to exist")
	}
	if _, ok := properties["age"]; !ok {
		t.Error("expected property 'age' to exist")
	}
	if _, ok := properties["email"]; !ok {
		t.Error("expected property 'email' to exist")
	}

	required := schema["required"].([]any)
	hasName := false
	hasAge := false
	for _, r := range required {
		if r == "name" {
			hasName = true
		}
		if r == "age" {
			hasAge = true
		}
	}

	if !hasName || !hasAge {
		t.Errorf("expected required fields 'name' and 'age', got %v", required)
	}
}

func TestResponseFormatFromSchema(t *testing.T) {
	manualSchema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"city": {Type: "string"},
		},
		Required: []string{"city"},
	}

	rf, err := ResponseFormatFromSchema("city_info", manualSchema)
	if err != nil {
		t.Fatalf("ResponseFormatFromSchema failed: %v", err)
	}

	if rf.JSONSchema.Name != "city_info" {
		t.Errorf("expected name city_info, got %q", rf.JSONSchema.Name)
	}

	schema := rf.JSONSchema.Schema
	properties := schema["properties"].(map[string]any)
	if _, ok := properties["city"]; !ok {
		t.Error("expected property 'city' to exist")
	}

	// Verify it can be marshaled to the expected OpenAI format
	b, err := json.Marshal(rf)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var target map[string]any
	if err := json.Unmarshal(b, &target); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestAgentContextLimit(t *testing.T) {
	mockLLM := NewMockLLM("gpt-4o", []string{"Hello"}, nil)

	tools.InitGlobalToolRegistry()

	limit := 10
	registry := tools.NewToolRegistry()
	ag := newAgent(mockLLM, "", "", registry, WithContextLength(limit))

	_, err := ag.Run(context.Background(), "This input is definitely longer than 10 characters", nil, nil)
	if err == nil {
		t.Fatal("expected error due to context limit, but got nil")
	}

	if !strings.Contains(err.Error(), "context length exceeded") {
		t.Errorf("expected context length exceeded error, got: %v", err)
	}
}

func TestValidateLLMResponse(t *testing.T) {
	agent := &Agent{}

	tests := []struct {
		name             string
		response         string
		toolCalls        []ToolCall
		expectedType     int
		expectedHasError bool
	}{
		{
			name:             "valid - response with no tool calls",
			response:         "Hello, world!",
			toolCalls:        []ToolCall{},
			expectedType:     InvalidTypeNone,
			expectedHasError: false,
		},
		{
			name:             "valid - empty response with tool calls",
			response:         "",
			toolCalls:        []ToolCall{{Name: "test_tool", Input: json.RawMessage("{}")}},
			expectedType:     InvalidTypeNone,
			expectedHasError: false,
		},
		{
			name:             "valid - response with multiple tool calls",
			response:         "",
			toolCalls:        []ToolCall{{Name: "tool1", Input: json.RawMessage("{}")}, {Name: "tool2", Input: json.RawMessage("{}")}},
			expectedType:     InvalidTypeNone,
			expectedHasError: false,
		},
		{
			name:             "invalid - both empty",
			response:         "",
			toolCalls:        []ToolCall{},
			expectedType:     InvalidTypeAllEmpty,
			expectedHasError: true,
		},
		{
			name:             "invalid - response with tool calls",
			response:         "some response",
			toolCalls:        []ToolCall{{Name: "test_tool", Input: json.RawMessage("{}")}},
			expectedType:     InvalidTypeResponseWithToolCall,
			expectedHasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalidType, err := agent.validateLLMResponse(tt.response, tt.toolCalls)
			if tt.expectedHasError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectedHasError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if invalidType != tt.expectedType {
				t.Errorf("expected type %d but got %d", tt.expectedType, invalidType)
			}
		})
	}
}

func TestTryEnforceLLMResponse(t *testing.T) {
	tests := []struct {
		name             string
		invalidType      int
		expectedMessages int
		expectedContent  string
	}{
		{
			name:             "enforce all empty",
			invalidType:      InvalidTypeAllEmpty,
			expectedMessages: 1,
			expectedContent:  "You must respond with either a non-empty string, or a empty response with at least one tool call.",
		},
		{
			name:             "enforce response with tool call",
			invalidType:      InvalidTypeResponseWithToolCall,
			expectedMessages: 1,
			expectedContent:  "You must respond with either a non-empty string, or a empty response with at least one tool call.",
		},
		{
			name:             "unknown invalid type - no message",
			invalidType:      999,
			expectedMessages: 0,
			expectedContent:  "",
		},
		{
			name:             "valid type - no message",
			invalidType:      InvalidTypeNone,
			expectedMessages: 0,
			expectedContent:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{}
			agent.tryEnforceLLMResponse(tt.invalidType)

			if len(agent.messages) != tt.expectedMessages {
				t.Errorf("expected %d messages but got %d", tt.expectedMessages, len(agent.messages))
			}

			if tt.expectedMessages > 0 {
				lastMsg := agent.messages[len(agent.messages)-1]
				if lastMsg.Role != "system" {
					t.Errorf("expected role 'system' but got %q", lastMsg.Role)
				}
				if lastMsg.Content != tt.expectedContent {
					t.Errorf("expected content %q but got %q", tt.expectedContent, lastMsg.Content)
				}
			}
		})
	}
}
