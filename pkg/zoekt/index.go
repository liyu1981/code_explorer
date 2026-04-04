package zoekt

import (
	"context"
	"fmt"
	"path/filepath"
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
	store    *db.Store
	fs       *sqlitefs.SQLiteFS
	searcher *FileSearcher
}

func NewZoektIndex(store *db.Store, fs *sqlitefs.SQLiteFS) *ZoektIndex {
	return &ZoektIndex{
		store:    store,
		fs:       fs,
		searcher: NewFileSearcher(store, fs),
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
	builderOpts := Options{
		RepositoryDescription: Repository{
			ID:   cb.ID,
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
	return z.searcher.Search(ctx, codebaseID, query, opts)
}
