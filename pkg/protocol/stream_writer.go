package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StreamWriter handles writing the research stream to an io.Writer.
type StreamWriter struct {
	w io.Writer
}

// NewStreamWriter creates a new StreamWriter.
func NewStreamWriter(w io.Writer) *StreamWriter {
	return &StreamWriter{w: w}
}

// WriteOpenAIChunk writes a standard OpenAI-style data chunk.
func (s *StreamWriter) WriteOpenAIChunk(id, model, content string, finishReason *string) error {
	chunk := OpenAIStreamChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []struct {
			Index int `json:"index"`
			Delta struct {
				Content string `json:"content,omitempty"`
				Role    string `json:"role,omitempty"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Delta: struct {
					Content string `json:"content,omitempty"`
					Role    string `json:"role,omitempty"`
				}{
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
	}

	return s.writePrefix("data: ", chunk)
}

// WriteCEEvent writes a custom Code Explorer event.
func (s *StreamWriter) WriteCEEvent(event CEEvent) error {
	err := s.writePrefix("ce: ", event)
	if err != nil {
		return err
	}
	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

// WriteDone writes the completion signal.
func (s *StreamWriter) WriteDone() error {
	_, err := fmt.Fprintf(s.w, "data: [DONE]\n\n")
	if err != nil {
		return err
	}
	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func (s *StreamWriter) writePrefix(prefix string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.w, "%s%s\n\n", prefix, data)
	if err != nil {
		return err
	}

	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

// Helper methods for common CE events

func (s *StreamWriter) SendReasoning(content string) error {
	return s.WriteCEEvent(CEEvent{
		Object:  "research.reasoning.delta",
		Content: content,
	})
}

func (s *StreamWriter) SendStepUpdate(id string, label string, status StepStatus) error {
	return s.WriteCEEvent(CEEvent{
		Object: "research.step.update",
		ID:     id,
		Label:  label,
		Status: status,
	})
}

func (s *StreamWriter) SendSourceAdded(source SourceMaterial) error {
	return s.WriteCEEvent(CEEvent{
		Object: "research.source.added",
		Source: &source,
	})
}

func (s *StreamWriter) SendResourceMaterial(resource SourceMaterial) error {
	return s.WriteCEEvent(CEEvent{
		Object:   "resource.material",
		Resource: &resource,
	})
}

func (s *StreamWriter) SendToolCall(tool string, params any) error {
	return s.WriteCEEvent(CEEvent{
		Object: "tool.call.request",
		Tool:   tool,
		Params: params,
	})
}

func (s *StreamWriter) SendToolResponse(tool string, response any) error {
	return s.WriteCEEvent(CEEvent{
		Object:   "tool.call.response",
		Tool:     tool,
		Response: response,
	})
}
