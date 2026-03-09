package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/rs/zerolog/log"
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
	Parameters() map[string]any
	Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error)
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

func (r *ToolRegistry) MarshalToolsForLLM() []map[string]any {
	result := make([]map[string]any, 0, len(r.tools))
	for _, t := range r.tools {
		params := t.Parameters()
		if params == nil {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			}
		}
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  params,
			},
		})
	}
	return result
}

type LLM interface {
	Generate(ctx context.Context, messages []Message, tools []map[string]any) (string, []ToolCall, error)
	GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, stream protocol.IStreamWriter) (string, []ToolCall, error)
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

func (a *Agent) Run(ctx context.Context, input string, stream protocol.IStreamWriter) (string, error) {
	log.Info().Str("input", input).Msg("Agent starting run")
	a.messages = append(a.messages, Message{Role: "user", Content: input})

	tools := a.tools.MarshalToolsForLLM()

	for i := 0; i < a.maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("Agent iteration start")
		var response string
		var toolCalls []ToolCall
		var err error

		stepID := fmt.Sprintf("iteration-%d", i)
		if stream != nil {
			stream.SendStepUpdate(stepID, "Thinking and reasoning", protocol.StepActive)
		}

		if stream != nil {
			response, toolCalls, err = a.llm.GenerateStream(ctx, a.messages, tools, stream)
		} else {
			response, toolCalls, err = a.llm.Generate(ctx, a.messages, tools)
		}

		if stream != nil {
			stream.SendStepUpdate(stepID, "Thinking and reasoning", protocol.StepCompleted)
		}

		if err != nil {
			log.Error().Err(err).Int("iteration", i).Msg("LLM generation failed")
			return "", fmt.Errorf("llm generation failed: %w", err)
		}

		log.Debug().Int("tool_calls", len(toolCalls)).Msg("LLM response received")
		a.messages = append(a.messages, Message{Role: "assistant", Content: response})

		if len(toolCalls) == 0 {
			log.Info().Msg("Agent finished without tool calls")
			return response, nil
		}

		for _, tc := range toolCalls {
			log.Info().Str("tool", tc.Name).RawJSON("input", tc.Input).Msg("Executing tool")
			toolStepID := fmt.Sprintf("tool-%s-%d-%s", tc.Name, i, tc.ID)
			if stream != nil {
				stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepActive)
				stream.SendToolCall(tc.Name, tc.Input)
			}

			tool, ok := a.tools.Get(tc.Name)
			if !ok {
				msg := fmt.Sprintf("Error: tool %s not found", tc.Name)
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: msg,
				})
				if stream != nil {
					stream.SendToolResponse(tc.Name, msg)
					stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
				}
				continue
			}

			if len(tc.Input) == 0 {
				msg := fmt.Sprintf("Error: tool %s was called without any arguments. Please provide the required parameters in JSON format.", tc.Name)
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: msg,
				})
				if stream != nil {
					stream.SendToolResponse(tc.Name, msg)
					stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
				}
				continue
			}

			output, err := tool.Execute(ctx, tc.Input, stream)
			if err != nil {
				log.Error().Err(err).Str("tool", tc.Name).Msg("Tool execution failed")
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: err.Error(),
				})
				if stream != nil {
					stream.SendToolResponse(tc.Name, err.Error())
				}
			} else {
				log.Debug().Str("tool", tc.Name).Str("output", output).Msg("Tool execution successful")
				a.messages = append(a.messages, Message{
					Role:    "tool",
					Content: output,
				})
				if stream != nil {
					// Try to parse output as JSON to send structured response
					var structured any
					if json.Unmarshal([]byte(output), &structured) == nil {
						stream.SendToolResponse(tc.Name, structured)
					} else {
						stream.SendToolResponse(tc.Name, output)
					}
				}
			}

			if stream != nil {
				stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
			}
		}
	}

	return "", fmt.Errorf("max iterations (%d) reached", a.maxIterations)
}

func (a *Agent) Messages() []Message {
	return a.messages
}

type Config struct {
	LLM           map[string]any `json:"llm"`
	Tools         []string       `json:"tools"`
	MaxIterations int            `json:"max_iterations"`
}
