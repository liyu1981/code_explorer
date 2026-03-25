package agent

import (
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type mockStreamWriter struct {
	builder *strings.Builder
}

func (w *mockStreamWriter) WriteOpenAIChunk(id, model, content string, finishReason *string) error {
	w.builder.WriteString(content)
	return nil
}

func (w *mockStreamWriter) WriteCEEvent(event protocol.CEEvent) error { return nil }
func (w *mockStreamWriter) WriteDone() error                          { return nil }
func (w *mockStreamWriter) SendReasoning(content string) error        { return nil }
func (w *mockStreamWriter) SendTurnStarted(id string, query string, timestamp int64) error {
	return nil
}
func (w *mockStreamWriter) SendStepUpdate(id, message string, status protocol.StepStatus) error {
	return nil
}
func (w *mockStreamWriter) SendSourceAdded(source protocol.SourceMaterial) error        { return nil }
func (w *mockStreamWriter) SendResourceMaterial(resource protocol.SourceMaterial) error { return nil }
func (w *mockStreamWriter) SendToolCall(tool string, params any) error                  { return nil }
func (w *mockStreamWriter) SendToolResponse(tool string, response any) error            { return nil }
func (w *mockStreamWriter) SendTryRunStart(turnID string, try int64) error              { return nil }
func (w *mockStreamWriter) SendTryRunEnd(turnID string, try int64) error                { return nil }
func (w *mockStreamWriter) SendTryRunFailed(turnID string, try int64) error             { return nil }
