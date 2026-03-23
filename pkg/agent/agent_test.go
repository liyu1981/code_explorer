package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
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

	InitGlobalToolRegistry()

	limit := 10
	registry := NewToolRegistry()
	ag := newAgent(mockLLM, "", "", registry, WithContextLength(limit))

	_, err := ag.Run(context.Background(), "This input is definitely longer than 10 characters", nil, nil)
	if err == nil {
		t.Fatal("expected error due to context limit, but got nil")
	}

	if !strings.Contains(err.Error(), "context length exceeded") {
		t.Errorf("expected context length exceeded error, got: %v", err)
	}
}
