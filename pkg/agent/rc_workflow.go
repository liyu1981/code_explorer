package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/tools"
	"github.com/rs/zerolog/log"
)

const (
	RCDefaultMaxReflections = 3
	RCDefaultMaxIterations  = 10
	RCDefaultMaxRetry       = 3
)

type CritiqueResult struct {
	HasIssues  bool     `json:"has_issues"`
	Issues     []string `json:"issues"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type CritiqueResponse struct {
	HasIssues  bool     `json:"has_issues"`
	Issues     []string `json:"issues"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type RCStep struct {
	Draft     string
	ToolCalls []llm.ToolCall
	Critique  *CritiqueResult
}

type RCWorkflowRunner struct {
	generator      *llm.Generator
	toolRegistry   *tools.ToolRegistry
	systemPrompt   string
	maxReflections int
	maxIterations  int
	maxRetry       int
	responseFormat *llm.ResponseFormat
	critiqueFormat *llm.ResponseFormat
	messages       []llm.Message
	history        []RCStep
}

type RCWorkflowRunnerOption func(*RCWorkflowRunner)

func RCWithMaxReflections(n int) RCWorkflowRunnerOption {
	return func(r *RCWorkflowRunner) {
		r.maxReflections = n
	}
}

func RCWithMaxIterations(n int) RCWorkflowRunnerOption {
	return func(r *RCWorkflowRunner) {
		r.generator.Options(llm.WithGeneratorMaxIterations(n))
	}
}

func RCWithMaxRetry(n int) RCWorkflowRunnerOption {
	return func(r *RCWorkflowRunner) {
		r.generator.Options(llm.WithGeneratorMaxRetry(n))
	}
}

func RCWithSystemPrompt(prompt string) RCWorkflowRunnerOption {
	return func(r *RCWorkflowRunner) {
		r.systemPrompt = prompt
	}
}

func RCWithResponseFormat(rf *llm.ResponseFormat) RCWorkflowRunnerOption {
	return func(r *RCWorkflowRunner) {
		r.responseFormat = rf
	}
}

func NewRCWorkflowRunner(ai llm.LLM, toolRegistry *tools.ToolRegistry, opts ...RCWorkflowRunnerOption) *RCWorkflowRunner {
	r := &RCWorkflowRunner{
		generator:      llm.NewGenerator(ai, llm.WithGeneratorToolRegistry(toolRegistry)),
		toolRegistry:   toolRegistry,
		maxReflections: RCDefaultMaxReflections,
		maxIterations:  RCDefaultMaxIterations,
		maxRetry:       RCDefaultMaxRetry,
		systemPrompt:   DefaultRCSystemPrompt,
		messages:       make([]llm.Message, 0),
		history:        make([]RCStep, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func NewRCWorkflowRunnerWithJSONFormat(ai llm.LLM, toolRegistry *tools.ToolRegistry, opts ...RCWorkflowRunnerOption) (*RCWorkflowRunner, error) {
	critiqueFormat, err := llm.ResponseFormatFromStruct[CritiqueResponse]("critique_result")
	if err != nil {
		return nil, fmt.Errorf("failed to create critique response format: %w", err)
	}

	r := &RCWorkflowRunner{
		generator:      llm.NewGenerator(ai, llm.WithGeneratorToolRegistry(toolRegistry)),
		toolRegistry:   toolRegistry,
		maxReflections: RCDefaultMaxReflections,
		maxIterations:  RCDefaultMaxIterations,
		maxRetry:       RCDefaultMaxRetry,
		systemPrompt:   DefaultRCSystemPrompt,
		critiqueFormat: critiqueFormat,
		messages:       make([]llm.Message, 0),
		history:        make([]RCStep, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

const DefaultRCSystemPrompt = `You are a precise AI assistant that uses tools to accomplish tasks.

For each task:
1. Make a draft response (tool calls if needed)
2. After executing tools, critique your work
3. Revise if needed
4. When satisfied, provide your final answer`

func (r *RCWorkflowRunner) Run(ctx context.Context, goal string, stream protocol.IStreamWriter) (string, error) {
	r.messages = []llm.Message{
		{Role: "system", Content: r.systemPrompt},
		{Role: "user", Content: goal},
	}
	r.history = make([]RCStep, 0)

	if stream != nil {
		stream.SendTurnStarted("", goal, 0)
	}

	for i := 0; i < r.maxIterations; i++ {
		log.Debug().Int("iteration", i).Msg("RC runner iteration start")

		if stream != nil {
			stream.SendTryRunStart("", int64(i))
			stream.SendStepUpdate("", fmt.Sprintf("rc-draft-%d", i), "Drafting response", protocol.StepActive)
		}

		draft, toolCalls, err := r.generateDraft(ctx, stream)
		if err != nil {
			if stream != nil {
				stream.SendTryRunFailed("", int64(i))
			}
			return "", fmt.Errorf("generate draft: %w", err)
		}

		if stream != nil {
			stream.SendStepUpdate("", fmt.Sprintf("rc-draft-%d", i), "Drafting response", protocol.StepCompleted)
		}

		step := RCStep{
			Draft:     draft,
			ToolCalls: toolCalls,
		}

		if len(toolCalls) > 0 {
			for _, tc := range toolCalls {
				r.executeTool(ctx, tc, stream)
			}
		}

		if len(toolCalls) > 0 {
			if stream != nil {
				stream.SendStepUpdate("", fmt.Sprintf("rc-critique-%d", i), "Self-critique", protocol.StepActive)
			}

			critique, err := r.critique(ctx)
			if stream != nil {
				stream.SendStepUpdate("", fmt.Sprintf("rc-critique-%d", i), "Self-critique", protocol.StepCompleted)
			}

			if err != nil {
				log.Warn().Err(err).Msg("critique failed, continuing")
			} else {
				step.Critique = critique
				r.history = append(r.history, step)

				if critique.HasIssues {
					log.Info().Strs("issues", critique.Issues).Msg("Critique found issues, revising")

					if err := r.revise(critique); err != nil {
						log.Warn().Err(err).Msg("revise failed, continuing")
					}

					if i < r.maxReflections {
						if stream != nil {
							stream.SendTryRunEnd("", int64(i))
						}
						continue
					}
					log.Info().Int("limit", r.maxReflections).Msg("Max reflections reached")
				}
			}
		}

		if stream != nil {
			stream.SendTryRunEnd("", int64(i))
		}

		if len(toolCalls) == 0 {
			if stream != nil {
				stream.WriteOpenAIChunk("", "", draft, nil)
				stream.WriteDone()
			}
			return draft, nil
		}

		if step.Critique == nil || !step.Critique.HasIssues {
			if stream != nil {
				stream.WriteOpenAIChunk("", "", draft, nil)
				stream.WriteDone()
			}
			return draft, nil
		}
	}

	if stream != nil {
		stream.WriteDone()
	}
	return "", fmt.Errorf("max iterations (%d) reached", r.maxIterations)
}

func (r *RCWorkflowRunner) generateDraft(ctx context.Context, stream protocol.IStreamWriter) (string, []llm.ToolCall, error) {
	var tools []map[string]any = nil
	if r.toolRegistry != nil {
		tools = r.toolRegistry.MarshalToolsForLLM()
	}

	for i := 0; i < r.maxRetry; i++ {
		var response string
		var toolCalls []llm.ToolCall
		var err error

		if stream != nil {
			response, toolCalls, err = r.generator.GenerateStream(ctx, r.messages, tools, r.responseFormat, stream)
		} else {
			response, toolCalls, err = r.generator.Generate(ctx, r.messages, tools, r.responseFormat)
		}

		if err != nil {
			log.Error().Err(err).Int("retry", i).Msg("LLM generation failed in generateDraft")
			continue
		}

		if len(response) == 0 && len(toolCalls) == 0 {
			log.Warn().Int("retry", i).Msg("Empty response in generateDraft")
			r.addEnforcer("You must provide either a text response or tool calls.")
			continue
		}

		if len(toolCalls) > 0 {
			r.messages = r.generator.Messages()
			return response, toolCalls, nil
		}

		return response, nil, nil
	}

	return "", nil, fmt.Errorf("generate draft failed after %d retries", r.maxRetry)
}

func (r *RCWorkflowRunner) critique(ctx context.Context) (*CritiqueResult, error) {
	critiquePrompt := r.buildCritiquePrompt()

	r.messages = append(r.messages, llm.Message{
		Role:    "user",
		Content: critiquePrompt,
	})

	response, _, err := r.generator.LLM().Generate(ctx, r.messages, nil, r.critiqueFormat)
	if err != nil {
		return nil, fmt.Errorf("critique llm: %w", err)
	}

	r.messages = append(r.messages, llm.Message{
		Role:    "assistant",
		Content: response,
	})

	var parsed CritiqueResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		log.Warn().Err(err).Str("response", response).Msg("failed to parse critique JSON, treating as no issues")
		return &CritiqueResult{HasIssues: false}, nil
	}

	return &CritiqueResult{
		HasIssues:  parsed.HasIssues,
		Issues:     parsed.Issues,
		Suggestion: parsed.Suggestion,
	}, nil
}

func (r *RCWorkflowRunner) buildCritiquePrompt() string {
	var sb strings.Builder
	sb.WriteString("Critique your previous tool call and results:\n\n")

	for i := len(r.messages) - 1; i >= 0; i-- {
		if r.messages[i].Role == "assistant" && len(r.messages[i].ToolCalls) > 0 {
			sb.WriteString("Tool calls made:\n")
			for _, tc := range r.messages[i].ToolCalls {
				sb.WriteString(fmt.Sprintf("- %s(%s)\n", tc.Name, string(tc.Input)))
			}
			break
		}
	}

	sb.WriteString("\nTool results:\n")
	for _, msg := range r.messages {
		if msg.Role == "tool" {
			sb.WriteString(fmt.Sprintf("- %s\n", msg.Content))
		}
	}

	sb.WriteString("\nAnalyze and respond with JSON:\n")
	sb.WriteString(`{"has_issues": true/false, "issues": ["list of issues"], "suggestion": "optional fix suggestion"}`)

	return sb.String()
}

func (r *RCWorkflowRunner) revise(critique *CritiqueResult) error {
	var sb strings.Builder
	sb.WriteString("Based on your critique, please revise:\n\n")

	if critique.HasIssues {
		sb.WriteString("Issues to address:\n")
		for _, issue := range critique.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}

	if critique.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\nSuggestion: %s\n", critique.Suggestion))
	}

	sb.WriteString("\nPlease make the necessary corrections and try again.")

	r.messages = append(r.messages, llm.Message{
		Role:    "user",
		Content: sb.String(),
	})

	return nil
}

func (r *RCWorkflowRunner) executeTool(ctx context.Context, tc llm.ToolCall, stream protocol.IStreamWriter) {
	log.Info().Str("tool", tc.Name).Msg("RC executing tool")

	if stream != nil {
		toolStepID := fmt.Sprintf("rc-tool-%s", tc.Name)
		stream.SendStepUpdate("", toolStepID, fmt.Sprintf("Executing %s", tc.Name), protocol.StepActive)
		stream.SendToolCall("", tc.Name, tc.Input)
	}

	tool, ok := r.toolRegistry.Get(tc.Name)
	if !ok {
		msg := fmt.Sprintf("Error: tool '%s' not found", tc.Name)
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    msg,
			ToolCallID: tc.ID,
		})
		if stream != nil {
			stream.SendToolResponse("", tc.Name, msg)
			stream.SendStepUpdate("", fmt.Sprintf("rc-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepFailed)
		}
		return
	}

	inputJSON, err := json.Marshal(tc.Input)
	if err != nil {
		err = fmt.Errorf("marshal input: %w", err)
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    err.Error(),
			ToolCallID: tc.ID,
		})
		if stream != nil {
			stream.SendToolResponse("", tc.Name, err.Error())
			stream.SendStepUpdate("", fmt.Sprintf("rc-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepFailed)
		}
		return
	}

	output, err := tool.Execute(ctx, inputJSON, stream)
	if err != nil {
		log.Error().Err(err).Str("tool", tc.Name).Msg("RC tool execution failed")
		r.messages = append(r.messages, llm.Message{
			Role:       "tool",
			Content:    err.Error(),
			ToolCallID: tc.ID,
		})
		if stream != nil {
			stream.SendToolResponse("", tc.Name, err.Error())
			stream.SendStepUpdate("", fmt.Sprintf("rc-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepFailed)
		}
		return
	}

	log.Debug().Str("tool", tc.Name).Str("output", output).Msg("RC tool executed successfully")
	r.messages = append(r.messages, llm.Message{
		Role:       "tool",
		Content:    output,
		ToolCallID: tc.ID,
	})

	if stream != nil {
		var structured any
		if json.Unmarshal([]byte(output), &structured) == nil {
			stream.SendToolResponse("", tc.Name, structured)
		} else {
			stream.SendToolResponse("", tc.Name, output)
		}
		stream.SendStepUpdate("", fmt.Sprintf("rc-tool-%s", tc.Name), fmt.Sprintf("Executing %s", tc.Name), protocol.StepCompleted)
	}
}

func (r *RCWorkflowRunner) addEnforcer(hint string) {
	r.messages = append(r.messages, llm.Message{
		Role:    "user",
		Content: hint,
	})
}

func (r *RCWorkflowRunner) Messages() []llm.Message {
	return r.messages
}

func (r *RCWorkflowRunner) History() []RCStep {
	return r.history
}
