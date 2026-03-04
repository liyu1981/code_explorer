package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Embedder interface {
	Embed(texts []string) ([][]float32, error)
	Model() string
	Dimension() int
}

type OpenAIEmbedder struct {
	model   string
	apiBase string
	apiKey  string
	dim     int
	client  *http.Client
}

func NewOpenAIEmbedder(apiBase, model, apiKey string, dim int) *OpenAIEmbedder {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	return &OpenAIEmbedder{
		model:   model,
		apiBase: apiBase,
		apiKey:  apiKey,
		dim:     dim,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (e *OpenAIEmbedder) Embed(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// OpenAI supports up to 2048 inputs in a single request, 
	// but local models like Ollama might have different limits.
	// We'll process in batches of 100 for safety.
	batchSize := 100
	results := make([][]float32, len(texts))

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		reqBody := openAIEmbeddingRequest{
			Model: e.model,
			Input: batch,
		}
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", e.apiBase+"/embeddings", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if e.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+e.apiKey)
		}

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errResp openAIEmbeddingResponse
			json.NewDecoder(resp.Body).Decode(&errResp)
			msg := fmt.Sprintf("API returned status %d", resp.StatusCode)
			if errResp.Error != nil {
				msg += ": " + errResp.Error.Message
			}
			return nil, fmt.Errorf("%s", msg)
		}

		var res openAIEmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(res.Data) != len(batch) {
			return nil, fmt.Errorf("expected %d embeddings, got %d", len(batch), len(res.Data))
		}

		for _, d := range res.Data {
			if d.Index >= 0 && i+d.Index < len(results) {
				results[i+d.Index] = d.Embedding
			}
		}
	}

	return results, nil
}

func (e *OpenAIEmbedder) Model() string {
	return e.model
}

func (e *OpenAIEmbedder) Dimension() int {
	return e.dim
}
