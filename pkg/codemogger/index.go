package codemogger

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
	"github.com/liyu1981/code_explorer/pkg/codemogger/scan"
	"github.com/liyu1981/code_explorer/pkg/codemogger/search"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/rs/zerolog/log"
)

type CodeIndex struct {
	store          *db.Store
	embedder       embed.Embedder
	embeddingModel string
}

func NewCodeIndex(cfg *config.Config, store *db.Store) (*CodeIndex, error) {
	embCfg := cfg.CodeMogger.Embedder

	if cfg.CodeMogger.InheritSystemLLM && cfg.System.LLM != nil {
		if t, ok := cfg.System.LLM["type"].(string); ok && t != "" {
			embCfg.Type = t
		}
		if ep, ok := cfg.System.LLM["base_url"].(string); ok && ep != "" {
			embCfg.OpenAI.APIBase = ep
		}
		if key, ok := cfg.System.LLM["api_key"].(string); ok && key != "" {
			embCfg.OpenAI.APIKey = key
		}
		if model, ok := cfg.System.LLM["model"].(string); ok && model != "" {
			embCfg.OpenAI.Model = model
		}
		if embeddingDim, ok := cfg.System.LLM["embedding_dim"].(int); ok && embeddingDim != 0 {
			embCfg.OpenAI.EmbeddingDim = embeddingDim
		}
	}

	emb := embed.NewEmbedderFromConfig(embCfg)

	return &CodeIndex{
		store:          store,
		embedder:       emb,
		embeddingModel: cfg.CodeMogger.Embedder.Model,
	}, nil
}

func (c *CodeIndex) Index(ctx context.Context, dir string, opts *IndexOptions) (*IndexResult, error) {
	log.Info().Str("dir", dir).Msg("Starting indexing")
	start := time.Now()
	rootDir, _ := filepath.Abs(dir)

	// 1. Get/Create system codebase
	cb, err := c.store.GetOrCreateCodebase(ctx, rootDir, "", "local")
	if err != nil {
		return nil, fmt.Errorf("failed to get/create system codebase: %w", err)
	}
	log.Debug().Str("codebaseID", cb.ID).Msg("System codebase entry identified")

	// 2. Ensure codemogger metadata exists
	metadataID, err := c.store.CodemoggerEnsureMetadata(ctx, cb.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure codemogger metadata: %w", err)
	}
	log.Debug().Str("metadataID", metadataID).Msg("Codemogger metadata entry identified")

	// 3. Detect version (git commit)
	version := detectVersion(rootDir)
	if version != "" {
		_ = c.store.UpdateCodebaseVersion(ctx, cb.ID, version)
		log.Debug().Str("version", version).Msg("Detected codebase version")
	}

	log.Info().Msg("Scanning directory...")
	files, scanErrors := scan.ScanDirectory(rootDir, opts.Languages)
	log.Info().Int("filesFound", len(files)).Int("scanErrors", len(scanErrors)).Msg("Scan completed")
	if opts.Progress != nil {
		opts.Progress(len(files), len(files), "scan")
	}

	filesProcessed := 0
	chunksCreated := 0
	skipped := 0

	var filesToProcess []scan.ScannedFile
	activeFiles := make(map[string]bool)

	log.Info().Msg("Checking file hashes...")
	for i, file := range files {
		activeFiles[file.RelPath] = true
		storedHash, err := c.store.CodemoggerGetFileHash(ctx, metadataID, file.RelPath)
		if err != nil {
			log.Warn().Str("file", file.RelPath).Err(err).Msg("Failed to get file hash")
			continue
		}
		if storedHash == file.Hash {
			skipped++
		} else {
			filesToProcess = append(filesToProcess, file)
		}
		if opts.Progress != nil && i%100 == 0 {
			opts.Progress(i+1, len(files), "check")
		}
	}
	log.Info().Int("toProcess", len(filesToProcess)).Int("skipped", skipped).Msg("Hash check completed")

	var fileChunks []struct {
		FilePath string
		FileHash string
		Chunks   []db.CodeChunk
	}

	log.Info().Msg("Chunking files...")
	for i, file := range filesToProcess {
		langConfig := chunk.DetectLanguage(file.RelPath)
		if langConfig == nil {
			continue
		}

		chunks := chunk.ChunkFile(file.RelPath, file.Content, file.Hash, langConfig)
		if len(chunks) > 0 {
			dbChunks := make([]db.CodeChunk, len(chunks))
			for j, chk := range chunks {
				dbChunks[j] = db.CodeChunk{
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
				FilePath: file.RelPath,
				FileHash: file.Hash,
				Chunks:   dbChunks,
			})
			filesProcessed++
			chunksCreated += len(chunks)
		}
		if opts.Progress != nil && i%10 == 0 {
			opts.Progress(i+1, len(filesToProcess), "chunk")
		}
	}
	log.Info().Int("filesProcessed", filesProcessed).Int("chunksCreated", chunksCreated).Msg("Chunking completed")

	if len(fileChunks) > 0 {
		log.Info().Msg("Saving chunks to database...")
		if err := c.store.CodemoggerBatchUpsertAllFileChunks(ctx, metadataID, fileChunks); err != nil {
			return nil, fmt.Errorf("failed to upsert chunks: %w", err)
		}
		log.Info().Msg("Chunks saved successfully")
	}

	embedded := 0
	log.Info().Str("model", c.embeddingModel).Msg("Starting embedding...")
	for {
		stale, err := c.store.CodemoggerGetStaleEmbeddings(ctx, metadataID, c.embeddingModel, 1000)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get stale embeddings")
			break
		}
		if len(stale) == 0 {
			break
		}

		log.Debug().Int("batchSize", len(stale)).Msg("Embedding batch")
		if opts.Progress != nil {
			opts.Progress(embedded, embedded+len(stale), "embed")
		}

		texts := make([]string, len(stale))
		for i, s := range stale {
			texts[i] = buildEmbedText(s.FilePath, s.Kind, s.Name, s.Signature, s.Snippet)
		}

		vectors, err := c.embedder.Embed(texts)
		if err != nil {
			log.Error().Err(err).Msg("Embedding failed")
			break
		}

		items := make([]struct {
			ChunkKey  string
			Embedding []float32
			ModelName string
		}, len(stale))
		for i, s := range stale {
			items[i].ChunkKey = s.ChunkKey
			items[i].Embedding = vectors[i]
			items[i].ModelName = c.embeddingModel
		}

		if err := c.store.CodemoggerBatchUpsertEmbeddings(ctx, items); err != nil {
			log.Error().Err(err).Msg("Failed to save embeddings")
			break
		}
		embedded += len(vectors)
		log.Debug().Int("totalEmbedded", embedded).Msg("Embedding progress")

		if len(stale) < 1000 {
			break
		}
	}
	log.Info().Int("embeddedCount", embedded).Msg("Embedding completed")

	activeFilesList := make([]string, 0, len(activeFiles))
	for k := range activeFiles {
		activeFilesList = append(activeFilesList, k)
	}
	removed, _ := c.store.CodemoggerRemoveStaleFiles(ctx, metadataID, activeFilesList)

	_ = c.store.CodemoggerRebuildFTSTable(ctx, metadataID)
	_ = c.store.CodemoggerTouchCodebase(ctx, metadataID)

	duration := int(time.Since(start).Milliseconds())

	var errors []string
	errors = append(errors, scanErrors...)

	res := &IndexResult{
		Files:    filesProcessed,
		Chunks:   chunksCreated,
		Embedded: embedded,
		Skipped:  skipped,
		Removed:  removed,
		Errors:   errors,
		Duration: duration,
	}
	log.Info().
		Int("files", res.Files).
		Int("chunks", res.Chunks).
		Int("embedded", res.Embedded).
		Int("skipped", res.Skipped).
		Int("duration_ms", res.Duration).
		Msg("Indexing completed")

	return res, nil
}

func detectVersion(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

func (c *CodeIndex) Search(ctx context.Context, codebaseID, query string, opts *SearchOptions) ([]SearchResult, error) {
	log.Info().Str("codebaseID", codebaseID).Str("query", query).Interface("opts", opts).Msg("Searching index")
	if query == "" {
		return []SearchResult{}, nil
	}
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
		if len(vectors) == 0 || vectors[0] == nil {
			return []SearchResult{}, nil
		}
		results, err := c.store.CodemoggerVectorSearch(ctx, codebaseID, vectors[0], limit, includeSnippet)
		if err != nil {
			return nil, err
		}
		return convertResults(results), nil

	case SearchModeKeyword:
		processed := search.PreprocessQuery(query)
		if processed == "" {
			return []SearchResult{}, nil
		}
		results, err := c.store.CodemoggerFTSSearch(ctx, codebaseID, processed, limit, includeSnippet)
		if err != nil {
			return nil, err
		}
		return convertResults(results), nil

	case SearchModeHybrid:
		processed := search.PreprocessQuery(query)
		ftsResults, _ := c.store.CodemoggerFTSSearch(ctx, codebaseID, processed, limit, includeSnippet)

		vectors, _ := c.embedder.Embed([]string{query})
		var vecResults []db.SearchResult
		if len(vectors) > 0 && vectors[0] != nil {
			vecResults, _ = c.store.CodemoggerVectorSearch(ctx, codebaseID, vectors[0], limit, includeSnippet)
		}

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

func (c *CodeIndex) ListFiles(ctx context.Context, codebaseID string) ([]IndexedFile, error) {
	files, err := c.store.CodemoggerListFiles(ctx, codebaseID)
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

func (c *CodeIndex) ListCodebases(ctx context.Context) ([]Codebase, error) {
	codebases, err := c.store.CodemoggerListCodebases(ctx)
	if err != nil {
		return nil, err
	}
	converted := make([]Codebase, len(codebases))
	for i, cb := range codebases {
		converted[i] = Codebase{
			ID:         cb.ID,
			RootPath:   cb.RootPath,
			Name:       cb.Name,
			Type:       cb.Type,
			Version:    cb.Version,
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

func (c *CodeIndex) SetEmbedder(emb embed.Embedder) {
	c.embedder = emb
}

func (c *CodeIndex) ReloadConfig() error {
	cfg := config.Get()
	var emb embed.Embedder
	embCfg := cfg.CodeMogger.Embedder

	if cfg.CodeMogger.InheritSystemLLM && cfg.System.LLM != nil {
		if t, ok := cfg.System.LLM["type"].(string); ok && t != "" {
			embCfg.Type = t
		}
		if ep, ok := cfg.System.LLM["base_url"].(string); ok && ep != "" {
			embCfg.OpenAI.APIBase = ep
		}
		if key, ok := cfg.System.LLM["api_key"].(string); ok && key != "" {
			embCfg.OpenAI.APIKey = key
		}
		if model, ok := cfg.System.LLM["model"].(string); ok && model != "" {
			embCfg.OpenAI.Model = model
		}
	}

	switch embCfg.Type {
	case "openai":
		model := embCfg.OpenAI.Model
		if model == "" {
			model = embCfg.Model
		}
		if model == "" {
			model = "text-embedding-3-small"
		}
		emb = embed.NewOpenAIEmbedder(
			embCfg.OpenAI.APIBase,
			model,
			embCfg.OpenAI.APIKey,
			1536,
		)
	default:
		apiBase := embCfg.OpenAI.APIBase
		if apiBase == "" {
			apiBase = "http://localhost:11434/v1"
		}
		model := embCfg.Model
		if model == "" {
			model = "all-minilm:l6-v2"
		}
		emb = embed.NewOpenAIEmbedder(
			apiBase,
			model,
			embCfg.OpenAI.APIKey,
			384,
		)
	}

	c.embedder = emb
	c.embeddingModel = cfg.CodeMogger.Embedder.Model
	return nil
}

func (c *CodeIndex) GetStore() *db.Store {
	return c.store
}
