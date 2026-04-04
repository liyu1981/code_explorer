package zoekt

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	"github.com/rs/zerolog/log"
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

type Searcher interface {
	Search(query Query, opts *SearchOptions) (*SearchResult, error)
	Close() error
}

const (
	FileSearcherTypeCodemogger = "codemogger"
	FileSearcherTypeZoekt      = "zoekt"
)

type sqlitefsIndexFile struct {
	path string
	data []byte
}

func (f *sqlitefsIndexFile) Read(off uint32, sz uint32) ([]byte, error) {
	if off+sz > uint32(len(f.data)) {
		sz = uint32(len(f.data)) - off
	}
	return f.data[off : off+sz], nil
}

func (f *sqlitefsIndexFile) Size() (uint32, error) {
	return uint32(len(f.data)), nil
}

func (f *sqlitefsIndexFile) Close() error {
	return nil
}

func (f *sqlitefsIndexFile) Name() string {
	return f.path
}

// FileSearcher provides file-level search operations over a codebase's
// Zoekt shards stored in SQLiteFS.
type FileSearcher struct {
	store *db.Store
	fs    *sqlitefs.SQLiteFS
}

func NewFileSearcher(store *db.Store, fs *sqlitefs.SQLiteFS) *FileSearcher {
	return &FileSearcher{
		store: store,
		fs:    fs,
	}
}

// Search executes a query across all Zoekt shard files for the given codebase.
func (z *FileSearcher) Search(ctx context.Context, codebaseID string, query string, opts *SearchOptions) (*SearchResult, error) {
	metadata, err := z.store.ZoektGetMetadataByCodebase(ctx, codebaseID)
	log.Debug().Str("codebaseID", codebaseID).Interface("metadata", metadata).Msg("Fetched zoekt metadata for search")
	if err != nil {
		return nil, fmt.Errorf("failed to get zoekt metadata for codebase %v: %w", codebaseID, err)
	}
	if metadata == nil {
		return &SearchResult{}, nil
	}

	cb, err := z.store.GetCodebaseByID(ctx, codebaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get codebase: %w", err)
	}

	repoID := cb.ID

	parsedQuery, err := ParseQuery(query)
	log.Debug().Str("query", query).Interface("parsedQuery", parsedQuery).Msg("Parsed search query")
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}
	if parsedQuery == nil {
		return &SearchResult{}, nil
	}

	if opts == nil {
		opts = &SearchOptions{}
	}
	opts.SetDefaults()

	start := time.Now()

	shardDir := fmt.Sprintf("/%s", ShardPrefix(repoID))
	entries, err := z.fs.List(shardDir)
	log.Debug().Interface("entries", entries).Msg("found entries for query")
	if err != nil {
		return &SearchResult{}, nil
	}

	var allResults SearchResult
	totalShards := 0

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name, ".zoekt") {
			continue
		}

		shardPath := fmt.Sprintf("%s/%s", shardDir, entry.Name)
		log.Debug().Str("shardPath", shardPath).Msg("Will load shard")
		file, err := z.fsOpen(shardPath)
		if err != nil {
			log.Warn().Str("path", shardPath).Err(err).Msg("Failed to open shard")
			continue
		}

		searcher, err := OpenShard(file)
		if err != nil {
			log.Warn().Str("path", shardPath).Err(err).Msg("Failed to open shard searcher")
			file.Close()
			continue
		}

		totalShards++
		result, err := searcher.Search(parsedQuery, opts)
		searcher.Close()
		if err != nil {
			log.Warn().Str("path", shardPath).Err(err).Msg("Failed to search shard")
			continue
		}

		allResults.Files = append(allResults.Files, result.Files...)
		allResults.Stats.FilesExamined += result.Stats.FilesExamined
		allResults.Stats.FilesMatched += result.Stats.FilesMatched

		if opts.MaxMatchCount > 0 && len(allResults.Files) >= opts.MaxMatchCount {
			break
		}
	}

	allResults.Stats.Shards = totalShards
	allResults.Stats.Duration = time.Since(start).Seconds()

	if len(allResults.Files) > opts.MaxMatchCount {
		allResults.Files = allResults.Files[:opts.MaxMatchCount]
	}

	return &allResults, nil
}

func (z *FileSearcher) fsOpen(path string) (IndexFile, error) {
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	entries, err := z.fs.List(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list dir for size: %w", err)
	}
	var fileSize int64
	for _, e := range entries {
		if e.Name == name {
			fileSize = e.Size
			break
		}
	}
	if fileSize == 0 {
		return nil, fmt.Errorf("shard file not found or empty: %s", path)
	}
	fullData, err := z.fs.Read(path, 0, int(fileSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read shard: %w", err)
	}
	return &sqlitefsIndexFile{data: fullData, path: path}, nil
}
