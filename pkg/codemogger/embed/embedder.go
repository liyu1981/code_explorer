package embed

type Embedder interface {
	Embed(texts []string) ([][]float32, error)
	Model() string
	Dimension() int
}

type LocalEmbedder struct {
	model string
	dim   int
}

func NewLocalEmbedder() *LocalEmbedder {
	return &LocalEmbedder{
		model: "all-MiniLM-L6-v2",
		dim:   384,
	}
}

func (e *LocalEmbedder) Embed(texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = make([]float32, e.dim)
		for j := 0; j < e.dim; j++ {
			vectors[i][j] = float32(j+1) / float32(e.dim)
		}
	}
	return vectors, nil
}

func (e *LocalEmbedder) Model() string {
	return e.model
}

func (e *LocalEmbedder) Dimension() int {
	return e.dim
}

type OpenAIEmbedder struct {
	model   string
	apiBase string
	apiKey  string
	dim     int
}

func NewOpenAIEmbedder(apiBase, model, apiKey string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		model:   model,
		apiBase: apiBase,
		apiKey:  apiKey,
		dim:     1536,
	}
}

func (e *OpenAIEmbedder) Embed(texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = make([]float32, e.dim)
	}
	return vectors, nil
}

func (e *OpenAIEmbedder) Model() string {
	return e.model
}

func (e *OpenAIEmbedder) Dimension() int {
	return e.dim
}
