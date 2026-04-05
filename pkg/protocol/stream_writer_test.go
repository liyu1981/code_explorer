package protocol

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestStreamWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	turnID := "test-turn"
	sw := NewStreamWriter(turnID, buf)

	t.Run("WriteOpenAIChunk", func(t *testing.T) {
		buf.Reset()
		err := sw.WriteOpenAIChunk("test-id", "test-model", "hello", nil)
		if err != nil {
			t.Fatalf("WriteOpenAIChunk failed: %v", err)
		}
		output := buf.String()
		if !strings.HasPrefix(output, "data: ") {
			t.Errorf("expected data prefix, got %q", output)
		}
		var chunk OpenAIStreamChunk
		if err := json.Unmarshal([]byte(strings.TrimPrefix(output, "data: ")), &chunk); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if chunk.Choices[0].Delta.Content != "hello" {
			t.Errorf("expected content 'hello', got %q", chunk.Choices[0].Delta.Content)
		}
	})

	t.Run("WriteCEEvent", func(t *testing.T) {
		buf.Reset()
		err := sw.SendReasoning("thinking...")
		if err != nil {
			t.Fatalf("SendReasoning failed: %v", err)
		}
		output := buf.String()
		if !strings.HasPrefix(output, "ce: ") {
			t.Errorf("expected ce prefix, got %q", output)
		}
		var event CEEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(output, "ce: ")), &event); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if event.Object != "research.reasoning.delta" || event.Content != "thinking..." {
			t.Errorf("unexpected event: %+v", event)
		}
	})

	t.Run("WriteDone", func(t *testing.T) {
		buf.Reset()
		err := sw.WriteDone()
		if err != nil {
			t.Fatalf("WriteDone failed: %v", err)
		}
		if buf.String() != "data: [DONE]\n\n" {
			t.Errorf("expected [DONE] signal, got %q", buf.String())
		}
	})

	t.Run("SendStepUpdate", func(t *testing.T) {
		buf.Reset()
		err := sw.SendStepUpdate("step-1", "Step Label", StepActive)
		if err != nil {
			t.Fatalf("SendStepUpdate failed: %v", err)
		}
		var event CEEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(buf.String(), "ce: ")), &event); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if event.Status != StepActive || event.Label != "Step Label" {
			t.Errorf("unexpected event: %+v", event)
		}
	})
}
