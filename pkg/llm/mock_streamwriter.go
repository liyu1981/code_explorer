package llm

import (
	"strings"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type MockStreamWriter struct {
	builder *strings.Builder
}

func (w *MockStreamWriter) WriteOpenAIChunk(id, model, content string, finishReason *string) error {
	w.builder.WriteString(content)
	return nil
}

func (w *MockStreamWriter) WriteCEEvent(event protocol.CEEvent) error { return nil }
func (w *MockStreamWriter) WriteDone() error                          { return nil }
func (w *MockStreamWriter) SendReasoning(content string) error        { return nil }
func (w *MockStreamWriter) SendTurnStarted(id string, query string, timestamp int64) error {
	return nil
}
func (w *MockStreamWriter) SendStepUpdate(id, message string, status protocol.StepStatus) error {
	return nil
}
func (w *MockStreamWriter) SendSourceAdded(source protocol.SourceMaterial) error        { return nil }
func (w *MockStreamWriter) SendResourceMaterial(resource protocol.SourceMaterial) error { return nil }
func (w *MockStreamWriter) SendToolCall(tool string, params any) error                  { return nil }
func (w *MockStreamWriter) SendToolResponse(tool string, response any) error            { return nil }
func (w *MockStreamWriter) SendTryRunStart(turnID string, try int64) error              { return nil }
func (w *MockStreamWriter) SendTryRunEnd(turnID string, try int64) error                { return nil }
func (w *MockStreamWriter) SendTryRunFailed(turnID string, try int64) error             { return nil }
