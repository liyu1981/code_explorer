package query

type SearchOptions struct {
	RepoIDs       []uint32
	Branches      []string
	MaxMatchCount int
	MaxSearchTime int
	ShardRankMax  int
}

func (o *SearchOptions) SetDefaults() {
	if o.MaxMatchCount == 0 {
		o.MaxMatchCount = 500
	}
}

type FileMatch struct {
	FileName    string      `json:"fileName"`
	Repository  string      `json:"repository"`
	Branch      string      `json:"branch"`
	Content     string      `json:"content"`
	LineMatches []LineMatch `json:"lineMatches"`
	Score       float64     `json:"score"`
}

type LineMatch struct {
	Line          string `json:"line"`
	LineNumber    int    `json:"lineNumber"`
	LineStart     int    `json:"lineStart"`
	LineEnd       int    `json:"lineEnd"`
	ContentBefore string `json:"contentBefore"`
	ContentAfter  string `json:"contentAfter"`
}

type SearchResult struct {
	Files []FileMatch `json:"files"`
	Stats SearchStats `json:"stats"`
}

type SearchStats struct {
	Duration      float64 `json:"duration"`
	FilesExamined int     `json:"filesExamined"`
	FilesMatched  int     `json:"filesMatched"`
	Shards        int     `json:"shards"`
}
