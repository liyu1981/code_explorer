package zoekt

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	zkindex "github.com/liyu1981/code_explorer/pkg/zoekt/index"
	zkq "github.com/liyu1981/code_explorer/pkg/zoekt/query"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	FileSearcherTypeCodemogger = "codemogger"
	FileSearcherTypeZoekt      = "zoekt"
)

func ParseQuery(s string) (zkq.Q, error) {
	return zkq.Parse(s)
}

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

type ZkSearcher struct {
	store *db.Store
	fs    *sqlitefs.SQLiteFS
}

func NewZkSearcher(store *db.Store, fs *sqlitefs.SQLiteFS) *ZkSearcher {
	return &ZkSearcher{
		store: store,
		fs:    fs,
	}
}

func (z *ZkSearcher) Search(ctx context.Context, codebaseID string, queryStr string, opts *zkq.SearchOptions) (*zkq.SearchResult, error) {
	metadata, err := z.store.ZoektGetMetadataByCodebase(ctx, codebaseID)
	log.Debug().Str("codebaseID", codebaseID).Interface("metadata", metadata).Msg("Fetched zoekt metadata for search")
	if err != nil {
		return nil, fmt.Errorf("failed to get zoekt metadata for codebase %v: %w", codebaseID, err)
	}
	if metadata == nil {
		return &zkq.SearchResult{}, nil
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
		return &zkq.SearchResult{}, nil
	}

	if opts == nil {
		opts = &zkq.SearchOptions{}
	}
	opts.SetDefaults()

	start := time.Now()

	shardDir := fmt.Sprintf("/%s", zkindex.ShardPrefix(repoID))
	entries, err := z.fs.List(shardDir)
	log.Debug().Interface("entries", entries).Msg("found entries for query")
	if err != nil {
		return &zkq.SearchResult{}, nil
	}

	var shardPaths []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name, ".zoekt") {
			shardPaths = append(shardPaths, fmt.Sprintf("%s/%s", shardDir, entry.Name))
		}
	}

	if len(shardPaths) == 0 {
		return &zkq.SearchResult{}, nil
	}

	type shardResult struct {
		path   string
		result *zkq.SearchResult
		err    error
	}

	resultsChan := make(chan shardResult, len(shardPaths))
	sem := make(chan struct{}, opts.MaxConcurrentShards)

	var wg errgroup.Group

	for _, shardPath := range shardPaths {
		shardPath := shardPath
		wg.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() { <-sem }()

			log.Debug().Str("shardPath", shardPath).Msg("Will load shard")
			file, err := z.fsOpen(shardPath)
			if err != nil {
				log.Warn().Str("path", shardPath).Err(err).Msg("Failed to open shard")
				resultsChan <- shardResult{path: shardPath, err: err}
				return nil
			}

			searcher, err := zkindex.OpenShard(file)
			if err != nil {
				log.Warn().Str("path", shardPath).Err(err).Msg("Failed to open shard searcher")
				file.Close()
				resultsChan <- shardResult{path: shardPath, err: err}
				return nil
			}

			result, err := searcher.Search(parsedQuery, opts)
			searcher.Close()
			if err != nil {
				log.Warn().Str("path", shardPath).Err(err).Msg("Failed to search shard")
				resultsChan <- shardResult{path: shardPath, err: err}
				return nil
			}

			resultsChan <- shardResult{path: shardPath, result: result}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		log.Warn().Err(err).Msg("Error during parallel search")
	}
	close(resultsChan)

	var allResults zkq.SearchResult
	mu := sync.Mutex{}
	totalShards := 0

	for result := range resultsChan {
		totalShards++
		if result.err != nil {
			continue
		}
		if result.result == nil {
			continue
		}

		mu.Lock()
		allResults.Files = append(allResults.Files, result.result.Files...)
		allResults.Stats.FilesExamined += result.result.Stats.FilesExamined
		allResults.Stats.FilesMatched += result.result.Stats.FilesMatched

		if opts.MaxMatchCount > 0 && len(allResults.Files) >= opts.MaxMatchCount {
			mu.Unlock()
			break
		}
		mu.Unlock()
	}

	allResults.Stats.Shards = totalShards
	allResults.Stats.Duration = time.Since(start).Seconds()

	if len(allResults.Files) > opts.MaxMatchCount {
		allResults.Files = allResults.Files[:opts.MaxMatchCount]
	}

	return &allResults, nil
}

func (z *ZkSearcher) fsOpen(path string) (zkindex.IndexFile, error) {
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
