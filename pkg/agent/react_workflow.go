package agent

import (
	"context"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/rs/zerolog/log"
)

const (
	ReactDefaultMaxIterations = 10
)

type ReactWorkflowRunner struct {
	generator    *llm.Generator
	toolRegistry *llm.ToolRegistry
	systemPrompt string
	messages     []llm.Message
}

type ReactWorkflowRunnerOption func(*ReactWorkflowRunner)

func ReactWithMaxIterations(n int) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.generator.Options(llm.WithGeneratorMaxIterations(n))
	}
}

func ReactWithMaxRetry(n int) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.generator.Options(llm.WithGeneratorMaxRetry(n))
	}
}

func ReactWithSystemPrompt(prompt string) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.systemPrompt = prompt
	}
}

func ReactWithResponseFormat(rf *llm.ResponseFormat) ReactWorkflowRunnerOption {
	return func(r *ReactWorkflowRunner) {
		r.generator.Options(llm.WithGeneratorResponseFormat(rf))
	}
}

func NewReactWorkflowRunner(ai llm.LLM, toolRegistry *llm.ToolRegistry, opts ...ReactWorkflowRunnerOption) *ReactWorkflowRunner {
	r := &ReactWorkflowRunner{
		generator:    llm.NewGenerator(ai, llm.WithGeneratorToolRegistry(toolRegistry)),
		toolRegistry: toolRegistry,
		systemPrompt: DefaultReactSystemPrompt,
		messages:     make([]llm.Message, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

const DefaultReactSystemPrompt = `You are a helpful AI assistant that can use tools to accomplish tasks.

When you need to use a tool, make a tool call. If no more tool calls are needed, provide your final answer.`

func (r *ReactWorkflowRunner) Run(ctx context.Context, goal string, stream protocol.IStreamWriter) (string, error) {
	messages := []llm.Message{
		{Role: "system", Content: r.systemPrompt},
		{Role: "user", Content: goal},
	}

	tools := r.toolRegistry.MarshalToolsForLLM()
	r.messages = messages

	if stream != nil {
		stream.SendTurnStarted("", goal, 0)
	}

	var response string
	var toolCalls []llm.ToolCall
	var err error

	if stream != nil {
		response, toolCalls, err = r.generator.GenerateStream(ctx, messages, tools, nil, stream)
	} else {
		response, toolCalls, err = r.generator.Generate(ctx, messages, tools, nil)
	}

	if err != nil {
		log.Error().Err(err).Msg("ReactWorkflowRunner generation failed")
		return "", err
	}

	r.messages = r.generator.Messages()

	if stream != nil && len(toolCalls) == 0 {
		stream.WriteDone()
	}

	return response, nil
}

func (r *ReactWorkflowRunner) Messages() []llm.Message {
	return r.messages
}
