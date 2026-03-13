package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type HTTPClientLLM struct {
	model      string
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewHTTPClientLLM(model, baseURL, apiKey string) *HTTPClientLLM {
	return &HTTPClientLLM{
		model:      model,
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (l *HTTPClientLLM) Name() string {
	return l.model
}

func (l *HTTPClientLLM) Generate(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat) (string, []ToolCall, error) {
	payload := map[string]any{
		"model":    l.model,
		"messages": messages,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
	}
	if responseFormat != nil {
		payload["response_format"] = responseFormat
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	fullURL := strings.TrimSuffix(l.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(body))
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
						tType, _ := tcMap["type"].(string)
						funcMap, _ := tcMap["function"].(map[string]any)
						name, _ := funcMap["name"].(string)
						args, _ := funcMap["arguments"].(string)
						toolCalls = append(toolCalls, ToolCall{
							ID:   id,
							Type: tType,
							Function: ToolCallFunction{
								Name:      name,
								Arguments: args,
							},
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

func (l *HTTPClientLLM) GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat, streamWriter protocol.IStreamWriter) (string, []ToolCall, error) {
	payload := map[string]any{
		"model":    l.model,
		"messages": messages,
		"stream":   true,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
	}
	if responseFormat != nil {
		payload["response_format"] = responseFormat
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	fullURL := strings.TrimSuffix(l.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(body))
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
						Type     string `json:"type"`
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
						Type: tc.Type,
					}
				}
				if tc.Function.Name != "" {
					toolCallsMap[tc.Index].Function.Name = tc.Function.Name
					toolCallsMap[tc.Index].Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					toolCallsMap[tc.Index].Function.Arguments += tc.Function.Arguments
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

func (l *MockLLM) Generate(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat) (string, []ToolCall, error) {
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

func (l *MockLLM) GenerateStream(ctx context.Context, messages []Message, tools []map[string]any, responseFormat *ResponseFormat, stream protocol.IStreamWriter) (string, []ToolCall, error) {
	content, toolCalls, err := l.Generate(ctx, messages, tools, responseFormat)
	if err != nil {
		return "", nil, err
	}

	if stream != nil && content != "" {
		stream.WriteOpenAIChunk("mock-id", l.model, content, nil)
	}

	return content, toolCalls, nil
}

type EnvConfig struct {
	apiKey  string
	baseURL string
	model   string
}

func LoadEnvConfig() EnvConfig {
	return EnvConfig{
		apiKey:  os.Getenv("LLM_API_KEY"),
		baseURL: os.Getenv("LLM_BASE_URL"),
		model:   os.Getenv("LLM_MODEL"),
	}
}
