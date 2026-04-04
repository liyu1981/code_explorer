package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type mockTool struct {
	name string
}

func (m *mockTool) Name() string               { return m.name }
func (m *mockTool) Description() string        { return "mock description" }
func (m *mockTool) Parameters() map[string]any { return nil }
func (m *mockTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	return "result", nil
}

func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()
	tool := &mockTool{name: "test-tool"}
	reg.RegisterTool(tool)

	t.Run("Get", func(t *testing.T) {
		got, ok := reg.Get("test-tool")
		if !ok || got != tool {
			t.Errorf("expected to get test-tool")
		}

		_, ok = reg.Get("non-existent")
		if ok {
			t.Errorf("expected not to get non-existent tool")
		}
	})

	t.Run("List", func(t *testing.T) {
		list := reg.List()
		if len(list) != 1 || list[0].Name() != "test-tool" {
			t.Errorf("unexpected list: %+v", list)
		}
	})

	t.Run("MarshalToolsForLLM", func(t *testing.T) {
		marshaled := reg.MarshalToolsForLLM()
		if len(marshaled) != 1 {
			t.Errorf("expected 1 tool, got %d", len(marshaled))
		}
		fn := marshaled[0]["function"].(map[string]any)
		if fn["name"] != "test-tool" {
			t.Errorf("expected test-tool, got %v", fn["name"])
		}
	})
}
