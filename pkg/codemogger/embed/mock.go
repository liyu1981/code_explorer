package embed

type MockEmbedder struct {
	DimVal int
}

func (m *MockEmbedder) Embed(texts []string) ([][]float32, error) {
	dim := m.DimVal
	if dim == 0 {
		dim = 384
	}
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			vectors[i][j] = float32(j) / float32(dim)
		}
	}
	return vectors, nil
}

func (m *MockEmbedder) Model() string {
	return "mock-model"
}

func (m *MockEmbedder) Dimension() int {
	if m.DimVal == 0 {
		return 384
	}
	return m.DimVal
}
