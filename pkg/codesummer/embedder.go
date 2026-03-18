package codesummer

import (
	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
)

type CodesummerEmbedder struct {
	embedder embed.Embedder
}

func NewCodesummerEmbedder(embedder embed.Embedder) *CodesummerEmbedder {
	return &CodesummerEmbedder{
		embedder: embedder,
	}
}

func (e *CodesummerEmbedder) EmbedText(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	return e.embedder.Embed(texts)
}

func (e *CodesummerEmbedder) Model() string {
	return e.embedder.Model()
}

func (e *CodesummerEmbedder) Dimension() int {
	return e.embedder.Dimension()
}
