package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/liyu1981/code_explorer/pkg/agent"
)

type echoTool struct{}

func (t *echoTool) Name() string        { return "echo" }
func (t *echoTool) Description() string { return "Echoes the input back" }
func (t *echoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]string{"type": "string", "description": "Message to echo back"},
		},
		"required": []string{"message"},
	}
}
func (t *echoTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return fmt.Sprintf("echo: %s", string(input)), nil
}

type calculateTool struct{}

func (t *calculateTool) Name() string        { return "calculate" }
func (t *calculateTool) Description() string { return "Performs basic arithmetic" }
func (t *calculateTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "Operation to perform: add, sub, mul",
				"enum":        []interface{}{"add", "sub", "mul"},
			},
			"a": map[string]interface{}{"type": "integer", "description": "First number"},
			"b": map[string]interface{}{"type": "integer", "description": "Second number"},
		},
		"required": []string{"operation", "a", "b"},
	}
}
func (t *calculateTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Operation string `json:"operation"`
		A         int    `json:"a"`
		B         int    `json:"b"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	switch req.Operation {
	case "add":
		return fmt.Sprintf("%d", req.A+req.B), nil
	case "sub":
		return fmt.Sprintf("%d", req.A-req.B), nil
	case "mul":
		return fmt.Sprintf("%d", req.A*req.B), nil
	default:
		return "", fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func main() {
	model := "gpt-4"
	if len(os.Args) > 1 {
		model = os.Args[1]
	}

	llmURL := "http://localhost:11434/v1/chat/completions"
	if len(os.Args) > 2 {
		llmURL = os.Args[2]
	}

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		apiKey = "test"
	}

	prompt := "Hello, how are you?"
	if len(os.Args) > 3 {
		prompt = os.Args[3]
	}

	registry := agent.NewToolRegistry()
	registry.Register(&echoTool{})
	registry.Register(&calculateTool{})

	llm := agent.NewHTTPClientLLM(model, llmURL, apiKey)

	agentInstance := agent.NewAgent(llm, registry,
		agent.WithMaxIterations(5),
	)

	ctx := context.Background()
	result, err := agentInstance.Run(ctx, prompt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Result:", result)
	fmt.Println("Messages:")
	for _, msg := range agentInstance.Messages() {
		fmt.Printf("  [%s] %s\n", msg.Role, msg.Content)
	}
}
