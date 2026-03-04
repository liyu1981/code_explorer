package codemogger

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
	"github.com/liyu1981/code_explorer/pkg/codemogger/scan"
	"github.com/liyu1981/code_explorer/pkg/codemogger/search"
	"github.com/liyu1981/code_explorer/pkg/db"
)

type CodeIndex struct {
	store          *db.Store
	dbPath         string
	embedder       embed.Embedder
	embeddingModel string
}

func NewCodeIndex(dbPath string, cfg *Config) (*CodeIndex, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var emb embed.Embedder
	switch cfg.Embedder.Type {
	case "openai":
		emb = embed.NewOpenAIEmbedder(
			cfg.Embedder.OpenAI.APIBase,
			cfg.Embedder.OpenAI.Model,
			cfg.Embedder.OpenAI.APIKey,
		)
	default:
		emb = embed.NewLocalEmbedder()
	}

	dbConn, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := db.NewStore(dbConn, dbPath)

	return &CodeIndex{
		store:          store,
		dbPath:         dbPath,
		embedder:       emb,
		embeddingModel: cfg.Embedder.Model,
	}, nil
}

func ProjectDbPath(dir string) string {
	absDir, _ := filepath.Abs(dir)
	dbDir := filepath.Join(absDir, ".codemogger")
	return filepath.Join(dbDir, "index.db")
}

func (c *CodeIndex) Index(dir string, opts *IndexOptions) (*IndexResult, error) {
	start := time.Now()
	rootDir, _ := filepath.Abs(dir)

	codebaseID, err := c.store.CodemoggerGetOrCreateCodebase(rootDir, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get/create codebase: %w", err)
	}

	files, scanErrors := scan.ScanDirectory(rootDir, opts.Languages)

	filesProcessed := 0
	chunksCreated := 0
	skipped := 0

	var filesToProcess []scan.ScannedFile
	activeFiles := make(map[string]bool)

	for _, file := range files {
		activeFiles[file.AbsPath] = true
		storedHash, err := c.store.CodemoggerGetFileHash(codebaseID, file.AbsPath)
		if err != nil {
			continue
		}
		if storedHash == file.Hash {
			skipped++
		} else {
			filesToProcess = append(filesToProcess, file)
		}
	}

	var fileChunks []struct {
		FilePath string
		FileHash string
		Chunks   []db.CodeChunk
	}

	for _, file := range filesToProcess {
		langConfig := chunk.DetectLanguage(file.AbsPath)
		if langConfig == nil {
			continue
		}

		chunks := chunk.ChunkFile(file.AbsPath, file.Content, file.Hash, langConfig)
		if len(chunks) > 0 {
			dbChunks := make([]db.CodeChunk, len(chunks))
			for i, chk := range chunks {
				dbChunks[i] = db.CodeChunk{
					ChunkKey:  chk.ChunkKey,
					FilePath:  chk.FilePath,
					Language:  chk.Language,
					Kind:      chk.Kind,
					Name:      chk.Name,
					Signature: chk.Signature,
					Snippet:   chk.Snippet,
					StartLine: chk.StartLine,
					EndLine:   chk.EndLine,
					FileHash:  chk.FileHash,
				}
			}
			fileChunks = append(fileChunks, struct {
				FilePath string
				FileHash string
				Chunks   []db.CodeChunk
			}{
				FilePath: file.AbsPath,
				FileHash: file.Hash,
				Chunks:   dbChunks,
			})
			filesProcessed++
			chunksCreated += len(chunks)
		}
	}

	if len(fileChunks) > 0 {
		if err := c.store.CodemoggerBatchUpsertAllFileChunks(codebaseID, fileChunks); err != nil {
			return nil, fmt.Errorf("failed to upsert chunks: %w", err)
		}
	}

	embedded := 0
	for {
		stale, err := c.store.CodemoggerGetStaleEmbeddings(codebaseID, c.embeddingModel, 1000)
		if err != nil || len(stale) == 0 {
			break
		}

		texts := make([]string, len(stale))
		for i, s := range stale {
			texts[i] = buildEmbedText(s.FilePath, s.Kind, s.Name, s.Signature, s.Snippet)
		}

		vectors, err := c.embedder.Embed(texts)
		if err != nil {
			break
		}

		items := make([]struct {
			ChunkKey  string
			Embedding []float32
			ModelName string
		}, len(stale))
		for i, s := range stale {
			items[i] = struct {
				ChunkKey  string
				Embedding []float32
				ModelName string
			}{
				ChunkKey:  s.ChunkKey,
				Embedding: vectors[i],
				ModelName: c.embeddingModel,
			}
		}

		if err := c.store.CodemoggerBatchUpsertEmbeddings(items); err != nil {
			break
		}
		embedded += len(vectors)

		if len(stale) < 1000 {
			break
		}
	}

	activeFilesList := make([]string, 0, len(activeFiles))
	for k := range activeFiles {
		activeFilesList = append(activeFilesList, k)
	}
	removed, _ := c.store.CodemoggerRemoveStaleFiles(codebaseID, activeFilesList)

	_ = c.store.CodemoggerRebuildFTSTable(codebaseID)
	_ = c.store.CodemoggerTouchCodebase(codebaseID)

	duration := int(time.Since(start).Milliseconds())

	var errors []string
	errors = append(errors, scanErrors...)

	return &IndexResult{
		Files:    filesProcessed,
		Chunks:   chunksCreated,
		Embedded: embedded,
		Skipped:  skipped,
		Removed:  removed,
		Errors:   errors,
		Duration: duration,
	}, nil
}

func buildEmbedText(filePath, kind, name, signature, snippet string) string {
	text := filePath
	if kind != "" && name != "" {
		text += ": " + kind + " " + name
	} else if name != "" {
		text += ": " + name
	}
	if signature != "" {
		text += "\n" + signature
	}
	if len(snippet) > 500 {
		snippet = snippet[:500]
	}
	if snippet != "" {
		text += "\n" + snippet
	}
	return text
}

func (c *CodeIndex) Search(query string, opts *SearchOptions) ([]SearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}
	if opts.Limit == 0 {
		opts.Limit = 5
	}
	if opts.Mode == "" {
		opts.Mode = SearchModeSemantic
	}

	limit := opts.Limit
	includeSnippet := opts.IncludeSnippet

	switch opts.Mode {
	case SearchModeSemantic:
		vectors, err := c.embedder.Embed([]string{query})
		if err != nil {
			return nil, err
		}
		results, err := c.store.CodemoggerVectorSearch(vectors[0], limit, includeSnippet)
		if err != nil {
			return nil, err
		}
		return convertResults(results), nil

	case SearchModeKeyword:
		processed := search.PreprocessQuery(query)
		if processed == "" {
			return []SearchResult{}, nil
		}
		results, err := c.store.CodemoggerFTSSearch(processed, limit, includeSnippet)
		if err != nil {
			return nil, err
		}
		return convertResults(results), nil

	case SearchModeHybrid:
		processed := search.PreprocessQuery(query)
		ftsResults, _ := c.store.CodemoggerFTSSearch(processed, limit, includeSnippet)

		vectors, _ := c.embedder.Embed([]string{query})
		vecResults, _ := c.store.CodemoggerVectorSearch(vectors[0], limit, includeSnippet)

		merged := search.RRFMerge(ftsResults, vecResults, limit, 60, 0.4, 0.6)
		return convertResults(merged), nil
	}

	return []SearchResult{}, nil
}

func convertResults(results []db.SearchResult) []SearchResult {
	converted := make([]SearchResult, len(results))
	for i, r := range results {
		converted[i] = SearchResult{
			ChunkKey:  r.ChunkKey,
			FilePath:  r.FilePath,
			Name:      r.Name,
			Kind:      r.Kind,
			Signature: r.Signature,
			Snippet:   r.Snippet,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
			Score:     r.Score,
		}
	}
	return converted
}

func (c *CodeIndex) ListFiles() ([]IndexedFile, error) {
	files, err := c.store.CodemoggerListFiles(0)
	if err != nil {
		return nil, err
	}
	converted := make([]IndexedFile, len(files))
	for i, f := range files {
		converted[i] = IndexedFile{
			FilePath:   f.FilePath,
			FileHash:   f.FileHash,
			ChunkCount: f.ChunkCount,
			IndexedAt:  f.IndexedAt,
		}
	}
	return converted, nil
}

func (c *CodeIndex) ListCodebases() ([]Codebase, error) {
	codebases, err := c.store.CodemoggerListCodebases()
	if err != nil {
		return nil, err
	}
	converted := make([]Codebase, len(codebases))
	for i, cb := range codebases {
		converted[i] = Codebase{
			ID:         int(cb.ID),
			RootPath:   cb.RootPath,
			Name:       cb.Name,
			IndexedAt:  cb.IndexedAt,
			FileCount:  cb.FileCount,
			ChunkCount: cb.ChunkCount,
		}
	}
	return converted, nil
}

func (c *CodeIndex) Close() error {
	return c.store.Close()
}
