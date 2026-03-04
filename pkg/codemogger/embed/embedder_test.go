package embed

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIEmbedderMock(t *testing.T) {
	// Mock OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected to request /embeddings, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"data": [{"embedding": [0.1, 0.2, 0.3], "index": 0}]}`)
	}))
	defer server.Close()

	emb := NewOpenAIEmbedder(server.URL, "test-model", "test-key", 3)
	texts := []string{"hello"}
	vectors, err := emb.Embed(texts)
	if err != nil {
		t.Fatalf("Failed to embed: %v", err)
	}

	if len(vectors) != 1 {
		t.Fatalf("Expected 1 vector, got %d", len(vectors))
	}
	if len(vectors[0]) != 3 {
		t.Fatalf("Expected dimension 3, got %d", len(vectors[0]))
	}
}

// TestOpenAILocalOllama is a placeholder for testing with a local Ollama instance.
// To run this, ensure Ollama is running and has the model pulled.
// Example: ollama pull all-minilm
func TestOpenAILocalOllama(t *testing.T) {
	t.Skip("Skipping local Ollama test. Remove this line to run if Ollama is available.")

	// Local Ollama configuration
	apiBase := "http://localhost:11434/v1"
	model := "all-minilm"
	
	emb := NewOpenAIEmbedder(apiBase, model, "", 384)
	texts := []string{"This is a test of local Ollama embedding."}
	
	vectors, err := emb.Embed(texts)
	if err != nil {
		t.Fatalf("Failed to embed with local Ollama: %v", err)
	}

	if len(vectors) != 1 {
		t.Fatalf("Expected 1 vector, got %d", len(vectors))
	}
	t.Logf("Vector length: %d", len(vectors[0]))
}
