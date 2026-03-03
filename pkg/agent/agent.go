package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolCall struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	Output   string          `json:"output,omitempty"`
	Error    string          `json:"error,omitempty"`
	Finished bool            `json:"finished"`
}

type ToolResult struct {
	ToolCallID string
	Output     string
	Error      error
}

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *ToolRegistry) MarshalToolsForLLM() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		params := t.Parameters()
		if params == nil {
			params = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			}
		}
		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  params,
			},
		})
	}
	return result
}

type LLM interface {
	Generate(ctx context.Context, messages []Message, tools []map[string]interface{}) (string, []ToolCall, error)
	Name() string
}

type Agent struct {
	llm           LLM
	tools         *ToolRegistry
	messages      []Message
	maxIterations int
}

type AgentOption func(*Agent)

func WithMaxIterations(n int) AgentOption {
	return func(a *Agent) {
		a.maxIterations = n
	}
}

func WithMessages(msgs []Message) AgentOption {
	return func(a *Agent) {
		a.messages = msgs
	}
}

func NewAgent(llm LLM, tools *ToolRegistry, opts ...AgentOption) *Agent {
	a := &Agent{
		llm:           llm,
		tools:         tools,
		messages:      make([]Message, 0),
		maxIterations: 10,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	a.messages = append(a.messages, Message{Role: "user", Content: input})

	tools := a.tools.MarshalToolsForLLM()

	for i := 0; i < a.maxIterations; i++ {
		response, toolCalls, err := a.llm.Generate(ctx, a.messages, tools)
		if err != nil {
			return "", fmt.Errorf("llm generation failed: %w", err)
		}

		a.messages = append(a.messages, Message{Role: "assistant", Content: response})

		if len(toolCalls) == 0 {
			return response, nil
		}

		for _, tc := range toolCalls {
			tool, ok := a.tools.Get(tc.Name)
			if !ok {
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: tool %s not found", tc.Name),
				})
				continue
			}

			if len(tc.Input) == 0 {
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: tool %s was called without any arguments. Please provide the required parameters in JSON format.", tc.Name),
				})
				continue
			}

			var args map[string]interface{}
			if err := json.Unmarshal(tc.Input, &args); err != nil {
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: tool %s received invalid JSON arguments: %s. Please provide valid JSON.", tc.Name, string(tc.Input)),
				})
				continue
			}

			output, err := tool.Execute(ctx, tc.Input)
			if err != nil {
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: err.Error(),
				})
			} else {
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: output,
				})
			}
		}
	}

	return "", fmt.Errorf("max iterations (%d) reached", a.maxIterations)
}

func (a *Agent) Messages() []Message {
	return a.messages
}

type Config struct {
	LLM           map[string]interface{} `json:"llm"`
	Tools         []string               `json:"tools"`
	MaxIterations int                    `json:"max_iterations"`
}
