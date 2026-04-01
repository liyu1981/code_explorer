package llm

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

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/liyu1981/code_explorer/pkg/protocol"
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

type HTTPClientLLM struct {
	model      string
	baseURL    string
	apiKey     string
	noThink    bool
	httpClient *http.Client
}

func newHTTPClientLLM(model, baseURL, apiKey string) *HTTPClientLLM {
	return &HTTPClientLLM{
		model:      model,
		baseURL:    baseURL,
		apiKey:     apiKey,
		noThink:    false,
		httpClient: &http.Client{},
	}
}

func (l *HTTPClientLLM) SetNoThink(noThink bool) {
	l.noThink = noThink
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
	if l.noThink {
		// Note: only llamacpp server support this, ollama is not yet.
		payload["chat_template_kwargs"] = map[string]any{"enable_thinking": false}
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
	if l.noThink {
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

func BuildLLM(cfg map[string]any) (LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm config is required")
	}

	llmType, _ := cfg["type"].(string)
	var llm LLM
	switch llmType {
	case "openai":
		baseURL, _ := cfg["base_url"].(string)
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "qwen3.5:4b"
		}
		apiKey := os.Getenv("LLM_API_KEY")
		if ak, ok := cfg["api_key"].(string); ok {
			apiKey = ak
		}
		httpLLMClient := newHTTPClientLLM(model, baseURL, apiKey)
		if _, ok := cfg["no_think"]; ok {
			if cfg["no_think"].(bool) {
				httpLLMClient.SetNoThink(true)
			}
		}
		llm = httpLLMClient

	default:
		// Fallback for when type is not specified but it looks like an OpenAI-compatible config
		if model, ok := cfg["model"].(string); ok && model != "" {
			baseURL, _ := cfg["base_url"].(string)
			apiKey, _ := cfg["api_key"].(string)
			return newHTTPClientLLM(model, baseURL, apiKey), nil
		}
		return nil, fmt.Errorf("unknown llm type: %s", llmType)
	}

	return llm, nil
}
