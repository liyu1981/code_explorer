package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type PipelineStepFunc func(ctx context.Context, input string) (string, error)

func NewPipelineStepFromFunc(name string, fn PipelineStepFunc) PipelineStep {
	return &funcPipelineStep{name: name, fn: fn}
}

type funcPipelineStep struct {
	name string
	fn   PipelineStepFunc
}

func (s *funcPipelineStep) Name() string {
	return s.name
}

func (s *funcPipelineStep) Execute(ctx context.Context, input string) (string, error) {
	return s.fn(ctx, input)
}

type PipelineStep interface {
	Name() string
	Execute(ctx context.Context, input string) (string, error)
}

type PromptTemplateStep struct {
	template string
	vars     map[string]string
}

func NewPromptTemplateStep(template string, vars map[string]string) *PromptTemplateStep {
	return &PromptTemplateStep{template: template, vars: vars}
}

func (s *PromptTemplateStep) Name() string {
	return "prompt_template"
}

func (s *PromptTemplateStep) Execute(ctx context.Context, input string) (string, error) {
	result := s.template
	for k, v := range s.vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	result = strings.ReplaceAll(result, "{{input}}", input)
	return result, nil
}

type RouterStep struct {
	routes map[string]PipelineStep
}

func NewRouterStep(routes map[string]PipelineStep) *RouterStep {
	return &RouterStep{routes: routes}
}

func (s *RouterStep) Name() string {
	return "router"
}

func (s *RouterStep) Execute(ctx context.Context, input string) (string, error) {
	for route, step := range s.routes {
		if strings.Contains(strings.ToLower(input), strings.ToLower(route)) {
			return step.Execute(ctx, input)
		}
	}
	return input, nil
}

type HTTPClientLLM struct {
	model      string
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewHTTPClientLLM(model, endpoint, apiKey string) *HTTPClientLLM {
	return &HTTPClientLLM{
		model:      model,
		endpoint:   endpoint,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (l *HTTPClientLLM) Name() string {
	return l.model
}

func (l *HTTPClientLLM) Generate(ctx context.Context, messages []Message, tools []map[string]any) (string, []ToolCall, error) {
	payload := map[string]any{
		"model":    l.model,
		"messages": messages,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", nil, err
	}

	content := ""
	toolCalls := []ToolCall{}

	if choices, ok := result["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if msg, ok := choice["message"].(map[string]any); ok {
				if ccontent, ok := msg["content"].(string); ok {
					content = ccontent
				}
				if tcs, ok := msg["tool_calls"].([]any); ok {
					for _, tcItem := range tcs {
						tcMap, ok := tcItem.(map[string]any)
						if !ok {
							continue
						}
						id, _ := tcMap["id"].(string)
						funcName, _ := tcMap["function"].(map[string]any)
						name, _ := funcName["name"].(string)
						args, _ := funcName["arguments"].(string)
						toolCalls = append(toolCalls, ToolCall{
							ID:    id,
							Name:  name,
							Input: json.RawMessage(args),
						})
					}
				}
			}
		}
	}

	return content, toolCalls, nil
}

func (l *HTTPClientLLM) GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, streamWriter *protocol.StreamWriter) (string, []ToolCall, error) {
	payload := map[string]any{
		"model":    l.model,
		"messages": messages,
		"stream":   true,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	fullContent := ""
	toolCallsMap := make(map[int]*ToolCall)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			ID      string `json:"id"`
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			content := choice.Delta.Content
			if content != "" {
				fullContent += content
				streamWriter.WriteOpenAIChunk(chunk.ID, l.model, content, choice.FinishReason)
			}

			for _, tc := range choice.Delta.ToolCalls {
				if _, ok := toolCallsMap[tc.Index]; !ok {
					toolCallsMap[tc.Index] = &ToolCall{
						ID:   tc.ID,
						Name: tc.Function.Name,
					}
				}
				if tc.Function.Arguments != "" {
					toolCallsMap[tc.Index].Input = append(toolCallsMap[tc.Index].Input, []byte(tc.Function.Arguments)...)
				}
			}
		}
	}

	var toolCalls []ToolCall
	for i := 0; i < len(toolCallsMap); i++ {
		if tc, ok := toolCallsMap[i]; ok {
			toolCalls = append(toolCalls, *tc)
		}
	}

	return fullContent, toolCalls, nil
}

type MockLLM struct {
	model     string
	responses []string
	toolCalls [][]ToolCall
	callIndex int
}

func NewMockLLM(model string, responses []string, toolCalls [][]ToolCall) *MockLLM {
	return &MockLLM{
		model:     model,
		responses: responses,
		toolCalls: toolCalls,
	}
}

func (l *MockLLM) Name() string {
	return l.model
}

func (l *MockLLM) Generate(ctx context.Context, messages []Message, tools []map[string]any) (string, []ToolCall, error) {
	if l.callIndex >= len(l.responses) {
		return "", nil, nil
	}

	response := l.responses[l.callIndex]
	var tcs []ToolCall
	if l.callIndex < len(l.toolCalls) {
		tcs = l.toolCalls[l.callIndex]
	}
	l.callIndex++

	return response, tcs, nil
}

func (l *MockLLM) GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, stream *protocol.StreamWriter) (string, []ToolCall, error) {
	content, toolCalls, err := l.Generate(ctx, messages, tools)
	if err != nil {
		return "", nil, err
	}

	if stream != nil && content != "" {
		stream.WriteOpenAIChunk("mock-id", l.model, content, nil)
	}

	return content, toolCalls, nil
}

type BaseTool struct {
	name        string
	description string
	parameters  map[string]any
	executeFn   func(ctx context.Context, input json.RawMessage) (string, error)
}

func (t *BaseTool) Name() string        { return t.name }
func (t *BaseTool) Description() string { return t.description }
func (t *BaseTool) Parameters() map[string]any {
	if t.parameters == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}
	return t.parameters
}

func (t *BaseTool) Execute(ctx context.Context, input json.RawMessage, stream *protocol.StreamWriter) (string, error) {
	return t.executeFn(ctx, input)
}

func NewBaseTool(name, description string, fn func(ctx context.Context, input json.RawMessage) (string, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		executeFn:   fn,
	}
}

type EnvConfig struct {
	apiKey   string
	endpoint string
	model    string
}

func LoadEnvConfig() EnvConfig {
	return EnvConfig{
		apiKey:   os.Getenv("LLM_API_KEY"),
		endpoint: os.Getenv("LLM_ENDPOINT"),
		model:    os.Getenv("LLM_MODEL"),
	}
}
