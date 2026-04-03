package zoekt

import (
	"regexp"
	"strings"
)

type Query interface {
	String() string
}

type Substring struct {
	Pattern       string
	FileName      bool
	Content       bool
	CaseSensitive bool
}

func (s *Substring) String() string {
	pref := ""
	if s.FileName {
		pref = "file:"
	}
	if s.CaseSensitive {
		pref = "case:" + pref
	}
	return pref + s.Pattern
}

type Regexp struct {
	Regexp        *regexp.Regexp
	FileName      bool
	Content       bool
	CaseSensitive bool
}

func (r *Regexp) String() string {
	pref := ""
	if r.FileName {
		pref = "file:"
	}
	if r.CaseSensitive {
		pref = "case:" + pref
	}
	return pref + "regex:" + r.Regexp.String()
}

type And struct {
	Children []Query
}

func (a *And) String() string {
	parts := make([]string, len(a.Children))
	for i, c := range a.Children {
		parts[i] = c.String()
	}
	return "and(" + strings.Join(parts, ", ") + ")"
}

type Or struct {
	Children []Query
}

func (o *Or) String() string {
	parts := make([]string, len(o.Children))
	for i, c := range o.Children {
		parts[i] = c.String()
	}
	return "or(" + strings.Join(parts, ", ") + ")"
}

type Not struct {
	Child Query
}

func (n *Not) String() string {
	return "not(" + n.Child.String() + ")"
}

type Branch struct {
	Pattern string
}

func (b *Branch) String() string {
	return "branch:" + b.Pattern
}

type Repo struct {
	Pattern string
}

func (r *Repo) String() string {
	return "repo:" + r.Pattern
}

type Language struct {
	Pattern string
}

func (l *Language) String() string {
	return "lang:" + l.Pattern
}

func ParseQuery(s string) (Query, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	return &Substring{Pattern: s}, nil
}

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
	FileName    string
	Repository  string
	Branch      string
	Content     string
	LineMatches []LineMatch
	Score       float64
}

type LineMatch struct {
	Line          string
	LineNumber    int
	LineStart     int
	LineEnd       int
	ContentBefore string
	ContentAfter  string
}

type SearchResult struct {
	Files []FileMatch
	Stats SearchStats
}

type SearchStats struct {
	Duration      float64
	FilesExamined int
	FilesMatched  int
	Shards        int
}

type Searcher interface {
	Search(query Query, opts *SearchOptions) (*SearchResult, error)
	Close() error
}
