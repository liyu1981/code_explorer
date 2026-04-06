package zoekt

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	"github.com/liyu1981/code_explorer/pkg/zoekt/query"
	"github.com/rs/zerolog/log"
)

type Q = query.Q
type Query = query.Q

type Substring = query.Substring
type Regexp = query.Regexp
type And = query.And
type Or = query.Or
type Not = query.Not
type Branch = query.Branch
type Repo = query.Repo
type Language = query.Language
type Const = query.Const

func ParseQuery(s string) (Q, error) {
	return query.Parse(s)
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
	Search(query Q, opts *SearchOptions) (*SearchResult, error)
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

func (z *FileSearcher) Search(ctx context.Context, codebaseID string, queryStr string, opts *SearchOptions) (*SearchResult, error) {
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

	parsedQuery, err := ParseQuery(queryStr)
	log.Debug().Str("query", queryStr).Interface("parsedQuery", parsedQuery).Msg("Parsed search query")
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
