package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/tools"
	"github.com/rs/zerolog/log"
)

type contextKey string

const (
	DefaultGeneratorMaxIterations = 10
	DefaultGeneratorMaxRetry      = 3
	DefaultGeneratorContextLength = 262144 / 2

	CodemoggerIndexKey contextKey = "codemogger_index"
)

type GeneratorOption func(*Generator)

func WithGeneratorMaxIterations(n int) GeneratorOption {
	return func(g *Generator) {
		g.maxIterations = n
	}
}

func WithGeneratorMaxRetry(n int) GeneratorOption {
	return func(g *Generator) {
		g.maxRetry = n
	}
}

func WithGeneratorContextLength(n int) GeneratorOption {
	return func(g *Generator) {
		g.contextLength = n
	}
}

func WithGeneratorToolRegistry(registry *tools.ToolRegistry) GeneratorOption {
	return func(g *Generator) {
		g.toolRegistry = registry
	}
}

func WithGeneratorResponseFormat(rf *ResponseFormat) GeneratorOption {
	return func(g *Generator) {
		g.responseFormat = rf
	}
}

type Generator struct {
	llm             LLM
	toolRegistry    *tools.ToolRegistry
	messages        []Message
	maxIterations   int
	maxRetry        int
	contextLength   int
	responseFormat  *ResponseFormat
	codemoggerIndex any
}

func WithGeneratorCodemoggerIndex(idx any) GeneratorOption {
	return func(g *Generator) {
		g.codemoggerIndex = idx
	}
}

func NewGenerator(llm LLM, opts ...GeneratorOption) *Generator {
	g := &Generator{
		llm:           llm,
		messages:      make([]Message, 0),
		maxIterations: DefaultGeneratorMaxIterations,
		maxRetry:      DefaultGeneratorMaxRetry,
		contextLength: DefaultGeneratorContextLength,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func (g *Generator) Options(opts ...GeneratorOption) {
	for _, opt := range opts {
		opt(g)
	}
}

func (g *Generator) Reset() {
	g.messages = make([]Message, 0)
}

func (g *Generator) MeasureContextLength(tools []map[string]any) int {
	total := 0
	for _, m := range g.messages {
		total += len(m.Role)
		total += len(m.Content)
		for _, tc := range m.ToolCalls {
			total += len(tc.ID)
			total += len(tc.Function.Name)
			total += len(tc.Function.Arguments)
		}
		total += len(m.ToolCallID)
	}
	for _, t := range tools {
		if b, err := json.Marshal(t); err == nil {
			total += len(b)
		}
	}
	return total
}

func (g *Generator) Generate(
	ctx context.Context,
	messages []Message,
	tools []map[string]any,
	responseFormat *ResponseFormat,
) (string, []ToolCall, error) {
	g.messages = make([]Message, len(messages))
	copy(g.messages, messages)

	log.Debug().Interface("responseFormat", responseFormat).Msg("will use reponseFormat")

	for i := 0; i < g.maxIterations; i++ {
		g.messages = append(g.messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("You can call tools max %d times. This is attempt #%d.", g.maxIterations, i+1),
		})

		currentLength := g.MeasureContextLength(tools)
		if g.contextLength > 0 && currentLength > g.contextLength {
			return "", nil, fmt.Errorf("context length exceeded: current %d, limit %d", currentLength, g.contextLength)
		}

		// disable tools in the last iteration to force a final answer
		availableTools := tools
		if i+1 >= g.maxIterations {
			availableTools = nil
			g.messages = append(g.messages, Message{
				Role:    "user",
				Content: "This is the final attempt, you must provide final answer according to response format without calling any tools.",
			})
		}

		response, toolCalls, err := g.generate(ctx, availableTools, responseFormat, nil)
		if err != nil {
			return "", nil, err
		}

		if len(toolCalls) == 0 {
			return response, nil, nil
		}

		for _, tc := range toolCalls {
			g.executeTool(ctx, tc, nil)
		}
	}

	return "", nil, fmt.Errorf("max iterations (%d) reached", g.maxIterations)
}

func (g *Generator) GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat, stream protocol.IStreamWriter) (string, []ToolCall, error) {
	g.messages = make([]Message, len(messages))
	copy(g.messages, messages)

	for i := 0; i < g.maxIterations; i++ {
		g.messages = append(g.messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("You can call tools max %d times. This is attempt #%d.", g.maxIterations, i+1),
		})

		currentLength := g.MeasureContextLength(tools)
		if g.contextLength > 0 && currentLength > g.contextLength {
			stream.WriteCEEvent(protocol.CEEvent{
				Object:  "error",
				Content: fmt.Sprintf("context length exceeded: current %d, limit %d", currentLength, g.contextLength),
			})
			return "", nil, fmt.Errorf("context length exceeded: current %d, limit %d", currentLength, g.contextLength)
		}

		stream.SendTryRunStart("", int64(i))
		stream.SendStepUpdate(fmt.Sprintf("gen-thinking-%d", i), "Thinking", protocol.StepActive)

		// disable tools in the last iteration to force a final answer
		availableTools := tools
		if i+1 >= g.maxIterations {
			availableTools = nil
			g.messages = append(g.messages, Message{
				Role:    "user",
				Content: "This is the final attempt, you must provide final answer according to response format without calling any tools.",
			})
		}

		response, toolCalls, err := g.generate(ctx, availableTools, responseFormat, stream)
		if err != nil {
			stream.SendStepUpdate(fmt.Sprintf("gen-thinking-%d", i), "Thinking", protocol.StepFailed)
			stream.SendTryRunFailed("", int64(i))
			return "", nil, err
		}

		stream.SendStepUpdate(fmt.Sprintf("gen-thinking-%d", i), "Thinking", protocol.StepCompleted)

		if len(toolCalls) == 0 {
			stream.SendTryRunEnd("", int64(i))
			stream.WriteDone()
			return response, nil, nil
		}

		for _, tc := range toolCalls {
			g.executeTool(ctx, tc, stream)
		}

		stream.SendTryRunEnd("", int64(i))
	}

	stream.WriteDone()
	return "", nil, fmt.Errorf("max iterations (%d) reached", g.maxIterations)
}

func (g *Generator) generate(
	ctx context.Context,
	tools []map[string]any,
	responseFormat *ResponseFormat,
	stream protocol.IStreamWriter,
) (string, []ToolCall, error) {
	for i := 0; i < g.maxRetry; i++ {
		var response string
		var toolCalls []ToolCall
		var err error

		if stream != nil {
			response, toolCalls, err = g.llm.GenerateStream(ctx, g.messages, tools, responseFormat, stream)
		} else {
			response, toolCalls, err = g.llm.Generate(ctx, g.messages, tools, responseFormat)
		}

		if err != nil {
			log.Error().Err(err).Int("retry", i).Msg("LLM generation failed")
			continue
		}

		if invalidType, err := g.validateResponse(response, toolCalls); err != nil {
			log.Warn().Err(err).Int("retry", i).Msg("Invalid response, retrying")
			g.enforceResponse(invalidType)
			continue
		}

		if len(toolCalls) > 0 {
			g.messages = append(g.messages, Message{
				Role:      "assistant",
				Content:   response,
				ToolCalls: toolCalls,
			})
		}

		return response, toolCalls, nil
	}

	return "", nil, fmt.Errorf("llm generation failed after %d retries", g.maxRetry)
}

func (g *Generator) validateResponse(response string, toolCalls []ToolCall) (int, error) {
	if len(response) == 0 && len(toolCalls) == 0 {
		return 1, fmt.Errorf("both response and toolCalls are empty")
	}
	if len(response) != 0 && len(toolCalls) > 0 {
		return 2, fmt.Errorf("response and toolCalls both present")
	}
	return 0, nil
}

func (g *Generator) enforceResponse(invalidType int) {
	hint := "You must respond with EITHER a text answer OR tool calls, not both."
	found := false
	for i := len(g.messages) - 1; i >= 0; i-- {
		if g.messages[i].Role == "user" {
			if !strings.Contains(g.messages[i].Content, hint) {
				g.messages[i].Content += "\n" + hint
			}
			found = true
			break
		}
	}
	if !found {
		g.messages = append(g.messages, Message{
			Role:    "user",
			Content: hint,
		})
	}
}

func (g *Generator) executeTool(ctx context.Context, tc ToolCall, stream protocol.IStreamWriter) {
	log.Info().Str("tool", tc.Name).Msg("Generator executing tool")

	if stream != nil {
		toolStepID := fmt.Sprintf("gen-tool-%s", tc.Name)
		stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing %s", tc.Name), protocol.StepActive)
		stream.SendToolCall(tc.Name, tc.Input)
	}

	if g.toolRegistry == nil {
		msg := "Error: no tool registry configured"
		g.messages = append(g.messages, Message{
			Role:       "tool",
			Content:    msg,
			ToolCallID: tc.ID,
		})
		if stream != nil {
			stream.SendToolResponse(tc.Name, msg)
			stream.SendStepUpdate(fmt.Sprintf("gen-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepFailed)
		}
		return
	}

	tool, ok := g.toolRegistry.Get(tc.Name)
	if !ok {
		msg := fmt.Sprintf("Error: tool '%s' not found", tc.Name)
		g.messages = append(g.messages, Message{
			Role:       "tool",
			Content:    msg,
			ToolCallID: tc.ID,
		})
		if stream != nil {
			stream.SendToolResponse(tc.Name, msg)
			stream.SendStepUpdate(fmt.Sprintf("gen-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepFailed)
		}
		return
	}

	execCtx := ctx
	if g.codemoggerIndex != nil {
		execCtx = context.WithValue(ctx, CodemoggerIndexKey, g.codemoggerIndex)
	}

	output, err := tool.Execute(execCtx, tc.Input, stream)
	if err != nil {
		log.Error().Err(err).Str("tool", tc.Name).Msg("Generator tool execution failed")
		g.messages = append(g.messages, Message{
			Role:       "tool",
			Content:    err.Error(),
			ToolCallID: tc.ID,
		})
		if stream != nil {
			stream.SendToolResponse(tc.Name, err.Error())
			stream.SendStepUpdate(fmt.Sprintf("gen-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepFailed)
		}
		return
	}

	log.Debug().Str("tool", tc.Name).Str("output", output).Msg("Generator tool executed successfully")
	g.messages = append(g.messages, Message{
		Role:       "tool",
		Content:    output,
		ToolCallID: tc.ID,
	})

	if stream != nil {
		var structured any
		if json.Unmarshal([]byte(output), &structured) == nil {
			stream.SendToolResponse(tc.Name, structured)
		} else {
			stream.SendToolResponse(tc.Name, output)
		}
		stream.SendStepUpdate(fmt.Sprintf("gen-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepCompleted)
	}
}

func (g *Generator) Messages() []Message {
	return g.messages
}

func (g *Generator) LLM() LLM {
	return g.llm
}
