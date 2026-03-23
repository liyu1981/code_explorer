package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/liyu1981/code_explorer/pkg/db"
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

type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

type JSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}

// ResponseFormatFromStruct creates a ResponseFormat from a Go struct type
func ResponseFormatFromStruct[T any](name string) (*ResponseFormat, error) {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}
	return ResponseFormatFromSchema(name, schema)
}

// ResponseFormatFromSchema creates a ResponseFormat from a jsonschema.Schema
func ResponseFormatFromSchema(name string, schema *jsonschema.Schema) (*ResponseFormat, error) {
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(b, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema to map: %w", err)
	}

	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: &JSONSchema{
			Name:   name,
			Schema: schemaMap,
		},
	}, nil
}

type LLM interface {
	Generate(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat) (string, []ToolCall, error)
	GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat, stream protocol.IStreamWriter) (string, []ToolCall, error)
	Name() string
}

type Agent struct {
	llm            LLM
	tools          *ToolRegistry
	SystemPrompt   string
	UserPromptTpl  string
	messages       []Message
	maxIterations  int
	contextLength  int
	responseFormat *ResponseFormat
	nothink        bool
}

type AgentOption func(*Agent)

func WithMaxIterations(n int) AgentOption {
	return func(a *Agent) {
		a.maxIterations = n
	}
}

func WithContextLength(n int) AgentOption {
	return func(a *Agent) {
		a.contextLength = n
	}
}

func WithMessages(msgs []Message) AgentOption {
	return func(a *Agent) {
		a.messages = msgs
	}
}

func WithResponseFormat(rf *ResponseFormat) AgentOption {
	return func(a *Agent) {
		a.responseFormat = rf
	}
}

func WithNoThink(noThink bool) AgentOption {
	return func(a *Agent) {
		a.nothink = noThink
	}
}

func newAgent(llm LLM, systemPrompt string, userPromptTpl string, tools *ToolRegistry, opts ...AgentOption) *Agent {
	a := &Agent{
		llm:           llm,
		tools:         tools,
		SystemPrompt:  systemPrompt,
		UserPromptTpl: userPromptTpl,
		messages:      make([]Message, 0),
		maxIterations: 10,
		contextLength: 262144, // Default to 256k
	}
	for _, opt := range opts {
		opt(a)
	}
	log.Debug().Interface("agent", a).Msg("new agent")
	return a
}

// MeasureContextLength approximates the total character count of the current context.
func (a *Agent) MeasureContextLength() int {
	total := 0

	// Count messages
	for _, m := range a.messages {
		total += len(m.Role)
		total += len(m.Content)
		for _, tc := range m.ToolCalls {
			total += len(tc.ID)
			total += len(tc.Function.Name)
			total += len(tc.Function.Arguments)
		}
		total += len(m.ToolCallID)
	}

	// Count tools
	tools := a.tools.MarshalToolsForLLM()
	for _, t := range tools {
		if b, err := json.Marshal(t); err == nil {
			total += len(b)
		}
	}

	return total
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

type StreamUpdate struct {
	TurnID string
	Stream protocol.IStreamWriter
}

func (a *Agent) Run(
	ctx context.Context,
	userInput string,
	responseFormat *ResponseFormat,
	streamUpdate *StreamUpdate,
	maxIterations ...int,
) (string, error) {
	a.responseFormat = responseFormat
	iterations := a.maxIterations
	if len(maxIterations) > 0 {
		iterations = maxIterations[0]
	}
	return a.run(ctx, userInput, streamUpdate, iterations)
}

func (a *Agent) run(
	ctx context.Context,
	input string,
	stream *StreamUpdate,
	maxIterations int,
) (string, error) {
	var turnID string
	if stream != nil {
		turnID = stream.TurnID
	}

	log.Info().
		Str("input", input).
		Str("turn", turnID).
		Int("max_iterations", maxIterations).
		Bool("stream", stream != nil).
		Msg("Agent starting run")

	a.SetSystemPrompt(a.SystemPrompt)

	a.messages = append(a.messages, Message{Role: "user", Content: input})

	if turnID != "" {
		ctx = util.WithInitiatorID(ctx, turnID)
	}

	tools := a.tools.MarshalToolsForLLM()
	log.Debug().Interface("tools", tools).Msg("Agent Found Tools")

	for i := 0; i < maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("Agent iteration start")

		// Check context length
		currentLength := a.MeasureContextLength()
		if a.contextLength > 0 && currentLength > a.contextLength {
			log.Error().
				Int("iteration", i).
				Int("current_length", currentLength).
				Int("context_length", a.contextLength).
				Msg("Context length exceeded")
			return "", fmt.Errorf("context length exceeded: current %d, limit %d", currentLength, a.contextLength)
		}

		var response string
		var toolCalls []ToolCall
		var err error

		// Use stable step ID for thinking in each turn
		thinkingStepID := fmt.Sprintf("turn-%s-thinking", turnID)
		if stream != nil {
			stream.Stream.SendStepUpdate(thinkingStepID, "Thinking and reasoning", protocol.StepActive)
		}

		if stream != nil {
			response, toolCalls, err = a.llm.GenerateStream(ctx, a.messages, tools, a.responseFormat, stream.Stream)
		} else {
			response, toolCalls, err = a.llm.Generate(ctx, a.messages, tools, a.responseFormat)
		}

		if stream != nil {
			stream.Stream.SendStepUpdate(thinkingStepID, "Thinking and reasoning", protocol.StepCompleted)
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
				stream.Stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepActive)
				stream.Stream.SendToolCall(tc.Name, tc.Input)
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
					stream.Stream.SendToolResponse(tc.Name, msg)
					stream.Stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
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
					stream.Stream.SendToolResponse(tc.Name, msg)
					stream.Stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
				}
				continue
			}

			log.Debug().Interface("tool", tool).Interface("tc.Input", tc.Input).Msg("try exec tool")
			var tcStream protocol.IStreamWriter
			if stream != nil {
				tcStream = stream.Stream
			}
			output, err := tool.Execute(ctx, tc.Input, tcStream)
			if err != nil {
				log.Error().Err(err).Str("tool", tc.Name).Msg("Tool execution failed")
				a.messages = append(a.messages, Message{
					Role:       "tool",
					Content:    err.Error(),
					ToolCallID: tc.ID,
				})
				if stream != nil {
					stream.Stream.SendToolResponse(tc.Name, err.Error())
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
						stream.Stream.SendToolResponse(tc.Name, structured)
					} else {
						stream.Stream.SendToolResponse(tc.Name, output)
					}
				}
			}

			if stream != nil {
				stream.Stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing tool: %s", tc.Name), protocol.StepCompleted)
			}
		}
	}

	log.Error().Int("maxIterations", maxIterations).Msg("Max iterations reached")
	return "", fmt.Errorf("max iterations (%d) reached", maxIterations)
}

func (a *Agent) Messages() []Message {
	return a.messages
}

type AgentBindDataProvider func(m *map[string]any)

func WithBindData(key string, value any) AgentBindDataProvider {
	return func(m *map[string]any) {
		(*m)[key] = value
	}
}

type AgentConfig struct {
	LLM             map[string]any `json:"llm"`
	Tools           []string       `json:"tools"`
	MaxIterations   int            `json:"max_iterations"`
	ContextLength   int            `json:"context_length"`
	AgentPromptName string         `json:"agent_prompt_name"`
	NoThink         bool           `json:"no_think"`
}

func GetAgentPromptSystemPrompt(ctx context.Context, name string) (string, error) {
	store := db.GetStore()
	if store == nil {
		return "", fmt.Errorf("store not initialized before calling GetAgentPromptSystemPrompt")
	}

	p, err := store.GetPromptByName(ctx, name)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", fmt.Errorf("agent prompt %s not found", name)
	}

	return p.SystemPrompt, nil
}

func GetAgentUserPromptTpl(ctx context.Context, name string) (string, error) {
	store := db.GetStore()
	if store == nil {
		return "", fmt.Errorf("store not initialized before calling GetAgentUserPromptTpl")
	}

	p, err := store.GetPromptByName(ctx, name)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", fmt.Errorf("agent prompt %s not found", name)
	}

	return p.UserPromptTpl, nil
}

func GetAgentPromptTools(ctx context.Context, name string) ([]string, error) {
	store := db.GetStore()
	if store == nil {
		return nil, fmt.Errorf("store not initialized before calling GetAgentPromptTools")
	}

	p, err := store.GetPromptByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("agent prompt %s not found", name)
	}
	if p.Tools == "" {
		return nil, nil
	}

	return strings.Fields(p.Tools), nil
}

func NewAgentFromConfig(
	ctx context.Context,
	cfg *AgentConfig,
	bindDataProviders ...AgentBindDataProvider,
) (*Agent, error) {
	llmCfg := cfg.LLM
	if llmCfg == nil {
		log.Error().Msg("LLM config is nil")
		return nil, fmt.Errorf("llm config is nil")
	}

	if cfg.NoThink {
		llmCfg["no_think"] = true
	}

	llm, err := BuildLLM(llmCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build LLM: %w", err)
	}

	contextLength := cfg.ContextLength
	if contextLength <= 0 {
		if cl, ok := llmCfg["context_length"].(int); ok {
			contextLength = cl
		} else if cl, ok := llmCfg["context_length"].(float64); ok {
			contextLength = int(cl)
		} else {
			contextLength = 262144
		}
	}

	bindData := &map[string]any{}
	for _, bindDataProvider := range bindDataProviders {
		bindDataProvider(bindData)
	}
	log.Debug().Interface("bindData", bindData).Msg("bind data prepared")

	systemPrompt, err := GetAgentPromptSystemPrompt(ctx, cfg.AgentPromptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent prompt system prompt: %w", err)
	}

	userPromptTpl, err := GetAgentUserPromptTpl(ctx, cfg.AgentPromptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent prompt user prompt template: %w", err)
	}

	toolRegistry := NewToolRegistry()
	globalRegistry := GetGlobalToolRegistry()

	if cfg.AgentPromptName != "" {
		promptTools, err := GetAgentPromptTools(ctx, cfg.AgentPromptName)
		if err != nil {
			return nil, fmt.Errorf("failed to get skill tools: %w", err)
		}
		if len(promptTools) > 0 {
			for _, toolName := range promptTools {
				tool, ok := globalRegistry.Get(toolName)
				if !ok {
					return nil, fmt.Errorf("tool %s not found in registry", toolName)
				}
				boundTool := tool.Clone()
				if err := boundTool.Bind(ctx, bindData); err != nil {
					return nil, fmt.Errorf("failed to bind tool %s: %w", toolName, err)
				}
				toolRegistry.Register(boundTool)
			}
		}
	}

	agent := newAgent(
		llm,
		systemPrompt,
		userPromptTpl,
		toolRegistry,
		WithMaxIterations(cfg.MaxIterations),
		WithContextLength(contextLength),
	)
	return agent, nil
}
