package zoekt

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger/scan"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	"github.com/rs/zerolog/log"
)

type IndexOptions struct {
	Languages []string
	Progress  func(current, total int, phase string)
}

type IndexResult struct {
	Files    int
	Skipped  int
	Removed  int
	Errors   []string
	Duration int
}

type ZoektIndex struct {
	store *db.Store
	fs    *sqlitefs.SQLiteFS
}

func NewZoektIndex(store *db.Store, fs *sqlitefs.SQLiteFS) *ZoektIndex {
	return &ZoektIndex{
		store: store,
		fs:    fs,
	}
}

func (z *ZoektIndex) GetStore() *db.Store {
	return z.store
}

func (z *ZoektIndex) Index(ctx context.Context, dir string, opts *IndexOptions) (*IndexResult, error) {
	log.Info().Str("dir", dir).Msg("Starting Zoekt indexing")
	start := time.Now()
	rootDir, _ := filepath.Abs(dir)

	// 1. Get/Create system codebase
	cb, err := z.store.GetOrCreateCodebase(ctx, rootDir, "", "local")
	if err != nil {
		return nil, fmt.Errorf("failed to get/create system codebase: %w", err)
	}

	// 2. Ensure zoekt metadata exists
	metadataID, err := z.store.ZoektEnsureMetadata(ctx, cb.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure zoekt metadata: %w", err)
	}

	log.Info().Msg("Scanning directory...")
	files, scanErrors := scan.ScanDirectory(rootDir, opts.Languages)
	log.Info().Int("filesFound", len(files)).Int("scanErrors", len(scanErrors)).Msg("Scan completed")
	if opts.Progress != nil {
		opts.Progress(len(files), len(files), "scan")
	}

	// 3. Check if we need to rebuild
	needsRebuild := false
	activeFiles := make(map[string]bool)
	skipped := 0

	for i, file := range files {
		activeFiles[file.RelPath] = true
		storedHash, err := z.store.ZoektGetFileHash(ctx, metadataID, file.RelPath)
		if err != nil {
			log.Warn().Str("file", file.RelPath).Err(err).Msg("Failed to get file hash")
			continue
		}
		if storedHash == file.Hash {
			skipped++
		} else {
			needsRebuild = true
		}
		if opts.Progress != nil && i%100 == 0 {
			opts.Progress(i+1, len(files), "check")
		}
	}

	// Also check if any files were removed by checking the count
	if !needsRebuild {
		storedFiles, err := z.store.ZoektListFiles(ctx, metadataID)
		if err != nil {
			return nil, fmt.Errorf("failed to list indexed files: %w", err)
		}
		if len(storedFiles) != len(files) {
			needsRebuild = true
		}
	}

	if !needsRebuild && len(files) > 0 {
		log.Info().Msg("No changes detected, skipping Zoekt indexing")
		return &IndexResult{
			Files:    len(files),
			Skipped:  len(files),
			Duration: int(time.Since(start).Milliseconds()),
		}, nil
	}

	log.Info().Msg("Changes detected or initial index, building Zoekt index...")

	// 4. Initialize Builder
	// We use the ID from our codebase record
	// But Repository.ID is uint32 in Zoekt, and our cb.ID is string (nanoid)
	// We'll use a hash or just use a simple mapping if needed.
	// For now, let's use a simple CRC32 of cb.ID as uint32.
	repoID := uint32(0)
	for _, c := range cb.ID {
		repoID = repoID*31 + uint32(c)
	}

	builderOpts := Options{
		RepositoryDescription: Repository{
			ID:   repoID,
			Name: cb.Name,
		},
		IndexFS:     z.fs,
		Parallelism: 4,
		ShardMax:    100 << 20,
	}
	builderOpts.SetDefaults()

	builder, err := NewBuilder(builderOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	// 5. Add all files to builder
	for i, file := range files {
		if err := builder.AddFile(file.RelPath, []byte(file.Content)); err != nil {
			log.Warn().Str("file", file.RelPath).Err(err).Msg("Failed to add file to builder")
			scanErrors = append(scanErrors, file.RelPath+": "+err.Error())
			continue
		}
		if opts.Progress != nil && i%100 == 0 {
			opts.Progress(i+1, len(files), "index")
		}
	}

	// 6. Finish building
	if err := builder.Finish(); err != nil {
		return nil, fmt.Errorf("failed to finish indexing: %w", err)
	}

	// 7. Update database with new hashes
	log.Info().Msg("Updating indexed files metadata...")
	for _, file := range files {
		if err := z.store.ZoektUpsertFileHash(ctx, metadataID, file.RelPath, file.Hash); err != nil {
			log.Warn().Str("file", file.RelPath).Err(err).Msg("Failed to update file hash")
		}
	}

	// 8. Remove stale files from DB
	activeFilesList := make([]string, 0, len(activeFiles))
	for k := range activeFiles {
		activeFilesList = append(activeFilesList, k)
	}
	removed, _ := z.store.ZoektRemoveStaleFiles(ctx, metadataID, activeFilesList)

	z.store.ZoektTouchCodebase(ctx, metadataID)

	duration := int(time.Since(start).Milliseconds())

	res := &IndexResult{
		Files:    len(files),
		Skipped:  skipped,
		Removed:  removed,
		Errors:   scanErrors,
		Duration: duration,
	}

	log.Info().
		Int("files", res.Files).
		Int("skipped", res.Skipped).
		Int("removed", res.Removed).
		Int("duration_ms", res.Duration).
		Msg("Zoekt indexing completed")

	return res, nil
}

func (z *ZoektIndex) ListFiles(ctx context.Context, codebaseID string) ([]db.FileInfo, error) {
	metadata, err := z.store.ZoektGetMetadataByCodebase(ctx, codebaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get zoekt metadata for codebase %v: %w", codebaseID, err)
	}
	if metadata == nil {
		return []db.FileInfo{}, nil
	}

	return z.store.ZoektListFiles(ctx, metadata.ID)
}

func (z *ZoektIndex) Search(ctx context.Context, codebaseID string, query string, opts *SearchOptions) (*SearchResult, error) {
	metadata, err := z.store.ZoektGetMetadataByCodebase(ctx, codebaseID)
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

	repoID := uint32(0)
	for _, c := range cb.ID {
		repoID = repoID*31 + uint32(c)
	}

	parsedQuery, err := ParseQuery(query)
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

func (z *ZoektIndex) fsOpen(path string) (IndexFile, error) {
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
