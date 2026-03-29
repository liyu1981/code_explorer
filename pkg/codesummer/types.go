package codesummer

type SummaryRequest struct {
	CodebaseID string
}

type NodeInfo struct {
	Path        string
	Type        string
	Language    string
	Content     string
	Hash        string
	Children    []string
	Definitions []Definition
}

type Definition struct {
	Kind      string
	Name      string
	Signature string
}

type NodeSummary struct {
	NodeInfo
	Summary         string
	Dependencies    []string
	DataManipulated []string
	DataFlow        DataFlowInfo
}

type DataFlowInfo struct {
	Inputs  []string
	Outputs []string
}

type FileSummaryResponse struct {
	Summary         string   `json:"summary"`
	Dependencies    []string `json:"dependencies"`
	DataManipulated []string `json:"data_manipulated"`
	DataFlow        struct {
		Inputs  []string `json:"inputs"`
		Outputs []string `json:"outputs"`
	} `json:"data_flow"`
}
