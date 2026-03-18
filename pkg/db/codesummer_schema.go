package db

type CodesummerCodebase struct {
	ID         string `json:"id"`
	CodebaseID string `json:"codebaseId"`
	IndexedAt  int64  `json:"indexedAt"`
}

type CodesummerSummary struct {
	ID              string `json:"id"`
	CodesummerID    string `json:"codesummerId"`
	NodePath        string `json:"nodePath"`
	NodeType        string `json:"nodeType"`
	Language        string `json:"language"`
	Summary         string `json:"summary"`
	Definitions     string `json:"definitions"`
	Dependencies    string `json:"dependencies"`
	DataManipulated string `json:"dataManipulated"`
	DataFlow        string `json:"dataFlow"`
	Embedding       []float32
	EmbeddingModel  string `json:"embeddingModel"`
	IndexedAt       int64  `json:"indexedAt"`
}

type IndexedPath struct {
	ID           string `json:"id"`
	CodesummerID string `json:"codesummerId"`
	NodePath     string `json:"nodePath"`
	NodeType     string `json:"nodeType"`
	FileHash     string `json:"fileHash"`
	IndexedAt    int64  `json:"indexedAt"`
}
