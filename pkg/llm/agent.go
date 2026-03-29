package llm

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

const (
	DEFAULT_MAX_ITERATIONS = 5
	DEFAULT_MAX_RETRY      = 3
	DEFAULT_CONTEXT_LENGTH = 262144 / 2
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
	maxRetry       int
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

func WithMaxRetry(n int) AgentOption {
	return func(a *Agent) {
		a.maxRetry = n
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
		maxIterations: DEFAULT_MAX_ITERATIONS,
		maxRetry:      DEFAULT_MAX_RETRY,
		contextLength: DEFAULT_CONTEXT_LENGTH,
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

func (a *Agent) ensureSystemPrompt(prompt string) {
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
		Int("max_retry", a.maxRetry).
		Bool("stream", stream != nil).
		Msg("Agent starting run")

	a.ensureSystemPrompt(strings.TrimSpace(a.SystemPrompt))

	a.messages = append(a.messages, Message{Role: "user", Content: strings.TrimSpace(input)})

	if turnID != "" {
		ctx = util.WithInitiatorID(ctx, turnID)
	}

	tools := a.tools.MarshalToolsForLLM()
	log.Debug().Interface("tools", tools).Msg("Agent Found Tools")

	// 	if len(tools) > 0 {
	// 		a.messages = ensureEnforceMessage(
	// 			EMDSystem,
	// 			a.messages,
	// 			fmt.Sprintf(`You may call tools at most %v times.
	// If you already have enough information, DO NOT call tools again. Return the final answer instead.
	// If a tool does not return useful data, do not retry more than once.`, a.maxIterations-1),
	// 		)
	// 	}

	for i := 0; i < maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("Agent iteration start")

		// if i+1 == maxIterations {
		// 	// this is the last iteration, enforce no tools called
		// 	a.messages = ensureEnforceMessage(
		// 		EMDUser,
		// 		a.messages,
		// 		"You have reached the maximum number of tool calls. Provide the best possible final answer now.",
		// 	)
		// }

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

		result, stop, err := a.tryRun(ctx, turnID, tools, stream)
		if err != nil {
			log.Error().Err(err).Msg("Agent run failed")
			return "", err
		}

		if stop {
			log.Debug().Str("result", result).Msg("Agent iteration stop")
			return result, nil
		}
	}

	log.Error().Int("maxIterations", maxIterations).Msg("Max iterations reached")
	return "", fmt.Errorf("max iterations (%d) reached", maxIterations)
}

func (a *Agent) tryRun(
	ctx context.Context,
	turnID string,
	tools []map[string]any,
	stream *StreamUpdate,
) (string, bool, error) {

	for i := 0; i < a.maxRetry; i++ {
		var response string
		var toolCalls []ToolCall
		var err error

		if stream != nil {
			stream.Stream.SendTryRunStart(turnID, int64(i))
		}

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
			log.Error().Err(err).Int("i", i).Msg("try LLM generation failed")
			return "", true, fmt.Errorf("llm generation failed: %w", err)
		}

		var invalidType int
		if invalidType, err = a.validateLLMResponse(response, toolCalls); err != nil {
			log.Error().Int("try", i).Int("invalid_type", invalidType).Err(err).Msg("Invalid LLM response, will retry")
			a.tryEnforceLLMResponse(invalidType)
			if stream != nil {
				stream.Stream.SendTryRunFailed(turnID, int64(i))
			}
			continue
		}

		log.Debug().Int("tool_calls", len(toolCalls)).Msg("LLM response received")
		a.messages = append(a.messages, Message{
			Role:      "assistant",
			Content:   response,
			ToolCalls: toolCalls,
		})

		if len(toolCalls) == 0 {
			log.Info().Msg("Agent finished without tool calls")
			if stream != nil {
				stream.Stream.SendTryRunEnd(turnID, int64(i))
			}
			return response, true, nil
		}

		for _, tc := range toolCalls {
			a.executeTool(ctx, tc, turnID, stream)
		}
		if stream != nil {
			stream.Stream.SendTryRunEnd(turnID, int64(i))
		}
		return response, false, nil
	}

	log.Error().Int("maxRetry", a.maxRetry).Msg("LLM generation failed with max retry")
	if stream != nil {
		stream.Stream.SendTryRunFailed(turnID, int64(a.maxRetry))
	}
	return "", true, fmt.Errorf("llm generation failed after max retry")
}

func (a *Agent) executeTool(ctx context.Context, tc ToolCall, turnID string, stream *StreamUpdate) {
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
		return
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
		return
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

const (
	InvalidTypeNone                 = 0
	InvalidTypeAllEmpty             = 1
	InvalidTypeResponseWithToolCall = 2
)

func (a *Agent) validateLLMResponse(response string, toolCalls []ToolCall) (int, error) {
	if len(response) == 0 && len(toolCalls) == 0 {
		return InvalidTypeAllEmpty, fmt.Errorf("both response and toolCalls are empty")
	}

	if len(response) != 0 && len(toolCalls) > 0 {
		return InvalidTypeResponseWithToolCall, fmt.Errorf("response is not empty with toolCalls")
	}

	// TODO: more cases?

	return InvalidTypeNone, nil
}

func (a *Agent) tryEnforceLLMResponse(invalidType int) {
	if invalidType == InvalidTypeAllEmpty {
		a.messages = ensureEnforceMessage(
			EMDUser,
			a.messages,
			"You must respond with either a non-empty string, or a empty response with at least one tool call.",
		)
		return
	}

	if invalidType == InvalidTypeResponseWithToolCall {
		a.messages = ensureEnforceMessage(
			EMDUser,
			a.messages,
			"You must respond with either a non-empty string, or a empty response with at least one tool call.",
		)
		return
	}
}

func (a *Agent) Messages() []Message {
	return a.messages
}

type EnforceMesssageDest int

const (
	EMDUser EnforceMesssageDest = iota
	EMDSystem
)

func ensureEnforceMessage(dest EnforceMesssageDest, messages []Message, enforcer string) []Message {
	if !((dest == EMDSystem) || (dest == EMDUser)) {
		log.Warn().Int("dest", int(dest)).Msg("Unknown EnforceMesssageDest, skip")
		return messages
	}

	destRole := "user"
	if dest == EMDSystem {
		destRole = "system"
	}

	// Search for the last Role=destRole message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == destRole {
			// Check if enforcer is already in the message
			if !strings.Contains(messages[i].Content, enforcer) {
				messages[i].Content += "\n" + enforcer
			}
			return messages
		}
	}

	// No destRole message found, append a new one
	messages = append(messages, Message{
		Role:    destRole,
		Content: enforcer,
	})
	return messages
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
	MaxRetry        int            `json:"max_retry"`
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

	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = DEFAULT_MAX_ITERATIONS
	}

	maxRetry := cfg.MaxRetry
	if maxRetry <= 0 {
		maxRetry = DEFAULT_MAX_RETRY
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
		WithMaxIterations(maxIterations),
		WithMaxRetry(maxRetry),
		WithContextLength(contextLength),
	)
	return agent, nil
}
