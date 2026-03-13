package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/util"
	"github.com/rs/zerolog/log"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
	Name     string           `json:"-"` // For internal use
	Input    json.RawMessage  `json:"-"` // For internal use
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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

func (a *Agent) SetSystemPrompt(prompt string) {
	// If the first message is a system prompt, update it
	if len(a.messages) > 0 && a.messages[0].Role == "system" {
		a.messages[0].Content = prompt
	} else {
		// Otherwise, prepend it
		newMsgs := append([]Message{{Role: "system", Content: prompt}}, a.messages...)
		a.messages = newMsgs
	}
}

type AgentPipelineStep struct {
	SystemPrompt string
	UserInput    string
}

func (a *Agent) RunPipeline(ctx context.Context, steps []AgentPipelineStep, turnID string, stream protocol.IStreamWriter) (string, error) {
	var lastOutput string
	var err error

	for _, step := range steps {
		a.SetSystemPrompt(step.SystemPrompt)
		lastOutput, err = a.run(ctx, step.UserInput, turnID, stream, 1)
		if err != nil {
			return "", err
		}
	}
	return lastOutput, nil
}

func (a *Agent) RunLoop(ctx context.Context, input string, turnID string, stream protocol.IStreamWriter, maxIterations ...int) (string, error) {
	iterations := a.maxIterations
	if len(maxIterations) > 0 {
		iterations = maxIterations[0]
	}
	return a.run(ctx, input, turnID, stream, iterations)
}

func (a *Agent) run(ctx context.Context, input string, turnID string, stream protocol.IStreamWriter, maxIterations int) (string, error) {
	log.Info().Str("input", input).Str("turn", turnID).Int("max_iterations", maxIterations).Msg("Agent starting run")
	a.messages = append(a.messages, Message{Role: "user", Content: input})

	if turnID != "" {
		ctx = util.WithInitiatorID(ctx, turnID)
	}

	tools := a.tools.MarshalToolsForLLM()

	for i := 0; i < maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("Agent iteration start")
		var response string
		var toolCalls []ToolCall
		var err error

		// Use stable step ID for thinking in each turn
		thinkingStepID := fmt.Sprintf("turn-%s-thinking", turnID)
		if stream != nil {
			stream.SendStepUpdate(thinkingStepID, "Thinking and reasoning", protocol.StepActive)
		}

		if stream != nil {
			response, toolCalls, err = a.llm.GenerateStream(ctx, a.messages, tools, stream)
		} else {
			response, toolCalls, err = a.llm.Generate(ctx, a.messages, tools)
		}

		if stream != nil {
			stream.SendStepUpdate(thinkingStepID, "Thinking and reasoning", protocol.StepCompleted)
		}

		if err != nil {
			log.Error().Err(err).Int("iteration", i).Msg("LLM generation failed")
			return "", fmt.Errorf("llm generation failed: %w", err)
		}

		log.Debug().Int("tool_calls", len(toolCalls)).Msg("LLM response received")
		a.messages = append(a.messages, Message{
			Role:      "assistant",
			Content:   response,
			ToolCalls: toolCalls,
		})

		if len(toolCalls) == 0 {
			log.Info().Msg("Agent finished without tool calls")
			return response, nil
		}

		for _, tc := range toolCalls {
			log.Info().Str("tool", tc.Name).RawJSON("input", tc.Input).Msg("Executing tool")
			// Use tool name as part of the ID to keep tool execution steps somewhat stable,
			// but still allow multiple tools to be shown.
			toolStepID := fmt.Sprintf("turn-%s-tool-%s", turnID, tc.Name)
			if stream != nil {
				stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepActive)
				stream.SendToolCall(tc.Name, tc.Input)
			}

			tool, ok := a.tools.Get(tc.Name)
			if !ok {
				msg := fmt.Sprintf("Error: tool %s not found", tc.Name)
				a.messages = append(a.messages, Message{
					Role:       "tool",
					Content:    msg,
					ToolCallID: tc.ID,
				})
				if stream != nil {
					stream.SendToolResponse(tc.Name, msg)
					stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
				}
				continue
			}

			if len(tc.Input) == 0 {
				msg := "Error: tool was called without any arguments."
				a.messages = append(a.messages, Message{
					Role:       "tool",
					Content:    msg,
					ToolCallID: tc.ID,
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
					Role:       "tool",
					Content:    err.Error(),
					ToolCallID: tc.ID,
				})
				if stream != nil {
					stream.SendToolResponse(tc.Name, err.Error())
				}
			} else {
				log.Debug().Str("tool", tc.Name).Str("output", output).Msg("Tool execution successful")
				a.messages = append(a.messages, Message{
					Role:       "tool",
					Content:    output,
					ToolCallID: tc.ID,
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

	return "", fmt.Errorf("max iterations (%d) reached", maxIterations)
}

func (a *Agent) Messages() []Message {
	return a.messages
}
