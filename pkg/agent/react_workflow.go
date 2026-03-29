package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/rs/zerolog/log"
)

const (
	ReactDefaultMaxIterations = 10
	ReactDefaultMaxRetry      = 3
)

type ReactWorkflowRunner struct {
	llm            llm.LLM
	toolRegistry   *llm.ToolRegistry
	systemPrompt   string
	maxIterations  int
	maxRetry       int
	responseFormat *llm.ResponseFormat
	messages       []llm.Message
}

type ReactWorkflowRunnerOption func(*ReactWorkflowRunner)

func ReactWithMaxIterations(n int) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.maxIterations = n
	}
}

func ReactWithMaxRetry(n int) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.maxRetry = n
	}
}

func ReactWithSystemPrompt(prompt string) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.systemPrompt = prompt
	}
}

func ReactWithResponseFormat(rf *llm.ResponseFormat) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.responseFormat = rf
	}
}

func NewReactWorkflowRunner(ai llm.LLM, toolRegistry *llm.ToolRegistry, opts ...ReactWorkflowRunnerOption) *ReactWorkflowRunner {
	r := &ReactWorkflowRunner{
		llm:           ai,
		toolRegistry:  toolRegistry,
		maxIterations: ReactDefaultMaxIterations,
		maxRetry:      ReactDefaultMaxRetry,
		systemPrompt:  DefaultReactSystemPrompt,
		messages:      make([]llm.Message, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

const DefaultReactSystemPrompt = `You are a helpful AI assistant that can use tools to accomplish tasks.

When you need to use a tool, make a tool call. If no more tool calls are needed, provide your final answer.`

func (r *ReactWorkflowRunner) Run(ctx context.Context, goal string) (string, error) {
	r.messages = []llm.Message{
		{Role: "system", Content: r.systemPrompt},
		{Role: "user", Content: goal},
	}

	for i := 0; i < r.maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("ReactWorkflowRunner iteration start")

		result, stop, err := r.tryRun(ctx, i)
		if err != nil {
			log.Error().Err(err).Int("iteration", i).Msg("ReactWorkflowRunner failed")
			return "", err
		}

		if stop {
			log.Debug().Str("result", result).Msg("ReactWorkflowRunner finished")
			return result, nil
		}
	}

	log.Error().Int("maxIterations", r.maxIterations).Msg("Max iterations reached")
	return "", fmt.Errorf("max iterations (%d) reached", r.maxIterations)
}

func (r *ReactWorkflowRunner) tryRun(ctx context.Context, iteration int) (string, bool, error) {
	tools := r.toolRegistry.MarshalToolsForLLM()

	for i := 0; i < r.maxRetry; i++ {
		response, toolCalls, err := r.llm.Generate(ctx, r.messages, tools, r.responseFormat)
		if err != nil {
			log.Error().Err(err).Int("retry", i).Msg("LLM generation failed")
			continue
		}

		if err := r.validateResponse(response, toolCalls); err != nil {
			log.Warn().Err(err).Int("retry", i).Msg("Invalid response, retrying")
			r.enforceResponseFix(response, toolCalls)
			continue
		}

		if len(toolCalls) == 0 {
			return response, true, nil
		}

		r.messages = append(r.messages, llm.Message{
			Role:      "assistant",
			Content:   response,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			r.executeTool(ctx, tc)
		}

		return response, false, nil
	}

	return "", true, fmt.Errorf("llm generation failed after max retry (%d)", r.maxRetry)
}

func (r *ReactWorkflowRunner) validateResponse(response string, toolCalls []llm.ToolCall) error {
	if len(response) == 0 && len(toolCalls) == 0 {
		return fmt.Errorf("both response and toolCalls are empty")
	}
	if len(response) != 0 && len(toolCalls) > 0 {
		return fmt.Errorf("response and toolCalls both present")
	}
	return nil
}

func (r *ReactWorkflowRunner) enforceResponseFix(response string, toolCalls []llm.ToolCall) {
	hint := "You must respond with EITHER a text answer OR tool calls, not both."

	found := false
	for i := len(r.messages) - 1; i >= 0; i-- {
		if r.messages[i].Role == "user" {
			if !strings.Contains(r.messages[i].Content, hint) {
				r.messages[i].Content += "\n" + hint
			}
			found = true
			break
		}
	}

	if !found {
		r.messages = append(r.messages, llm.Message{
			Role:    "user",
			Content: hint,
		})
	}
}

func (r *ReactWorkflowRunner) executeTool(ctx context.Context, tc llm.ToolCall) {
	log.Info().Str("tool", tc.Name).Msg("Executing tool")

	tool, ok := r.toolRegistry.Get(tc.Name)
	if !ok {
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    fmt.Sprintf("Error: tool '%s' not found", tc.Name),
			ToolCallID: tc.ID,
		})
		return
	}

	output, err := tool.Execute(ctx, tc.Input, nil)
	if err != nil {
		log.Error().Err(err).Str("tool", tc.Name).Msg("Tool execution failed")
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    fmt.Sprintf("Error: %v", err),
			ToolCallID: tc.ID,
		})
		return
	}

	log.Debug().Str("tool", tc.Name).Str("output", output).Msg("Tool executed successfully")
	r.messages = append(r.messages, llm.Message{
		Role:       "tool",
		Content:    output,
		ToolCallID: tc.ID,
	})
}

func (r *ReactWorkflowRunner) RunWithStream(ctx context.Context, goal string, stream protocol.IStreamWriter) (string, error) {
	r.messages = []llm.Message{
		{Role: "system", Content: r.systemPrompt},
		{Role: "user", Content: goal},
	}

	for i := 0; i < r.maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("ReactWorkflowRunner (streaming) iteration start")

		result, stop, err := r.tryRunWithStream(ctx, i, stream)
		if err != nil {
			log.Error().Err(err).Int("iteration", i).Msg("ReactWorkflowRunner (streaming) failed")
			return "", err
		}

		if stop {
			log.Debug().Str("result", result).Msg("ReactWorkflowRunner (streaming) finished")
			return result, nil
		}
	}

	log.Error().Int("maxIterations", r.maxIterations).Msg("Max iterations reached")
	return "", fmt.Errorf("max iterations (%d) reached", r.maxIterations)
}

func (r *ReactWorkflowRunner) tryRunWithStream(ctx context.Context, iteration int, stream protocol.IStreamWriter) (string, bool, error) {
	tools := r.toolRegistry.MarshalToolsForLLM()

	for i := 0; i < r.maxRetry; i++ {
		stream.SendTryRunStart("", int64(i))

		thinkingStepID := fmt.Sprintf("react-thinking-%d", iteration)
		stream.SendStepUpdate(thinkingStepID, "Thinking", protocol.StepActive)

		response, toolCalls, err := r.llm.GenerateStream(ctx, r.messages, tools, r.responseFormat, stream)

		stream.SendStepUpdate(thinkingStepID, "Thinking", protocol.StepCompleted)

		if err != nil {
			log.Error().Err(err).Int("retry", i).Msg("LLM streaming generation failed")
			stream.SendTryRunFailed("", int64(i))
			continue
		}

		if err := r.validateResponse(response, toolCalls); err != nil {
			log.Warn().Err(err).Int("retry", i).Msg("Invalid response in stream, retrying")
			r.enforceResponseFix(response, toolCalls)
			stream.SendTryRunFailed("", int64(i))
			continue
		}

		if len(toolCalls) == 0 {
			stream.SendTryRunEnd("", int64(i))
			return response, true, nil
		}

		r.messages = append(r.messages, llm.Message{
			Role:      "assistant",
			Content:   response,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			r.executeToolStream(ctx, tc, stream)
		}

		stream.SendTryRunEnd("", int64(i))
		return response, false, nil
	}

	stream.SendTryRunFailed("", int64(r.maxRetry))
	return "", true, fmt.Errorf("llm generation failed after max retry (%d)", r.maxRetry)
}

func (r *ReactWorkflowRunner) executeToolStream(ctx context.Context, tc llm.ToolCall, stream protocol.IStreamWriter) {
	log.Info().Str("tool", tc.Name).Msg("Executing tool (stream)")

	toolStepID := fmt.Sprintf("react-tool-%s", tc.Name)
	stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing %s", tc.Name), protocol.StepActive)
	stream.SendToolCall(tc.Name, tc.Input)

	tool, ok := r.toolRegistry.Get(tc.Name)
	if !ok {
		msg := fmt.Sprintf("Error: tool '%s' not found", tc.Name)
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    msg,
			ToolCallID: tc.ID,
		})
		stream.SendToolResponse(tc.Name, msg)
		stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing %s", tc.Name), protocol.StepCompleted)
		return
	}

	output, err := tool.Execute(ctx, tc.Input, stream)
	if err != nil {
		log.Error().Err(err).Str("tool", tc.Name).Msg("Tool execution failed (stream)")
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    err.Error(),
			ToolCallID: tc.ID,
		})
		stream.SendToolResponse(tc.Name, err.Error())
	} else {
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    output,
			ToolCallID: tc.ID,
		})

		var structured any
		if json.Unmarshal([]byte(output), &structured) == nil {
			stream.SendToolResponse(tc.Name, structured)
		} else {
			stream.SendToolResponse(tc.Name, output)
		}
	}

	stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing %s", tc.Name), protocol.StepCompleted)
}

func (r *ReactWorkflowRunner) Messages() []llm.Message {
	return r.messages
}
