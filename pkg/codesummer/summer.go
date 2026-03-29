package codesummer

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/util"
	"github.com/rs/zerolog/log"
)

type Codesummer struct {
	db             *db.Store
	fileSummarizer *Summarizer
	dirSummarizer  *Summarizer
	embedder       embed.Embedder
	classifier     *Classifier
	extractor      *Extractor
}

func NewCodesummer(
	dbStore *db.Store,
) (*Codesummer, error) {
	fileSummerizer, err := NewSummarizer("codesummer-file-summarizer", dbStore)
	if err != nil {
		return nil, err
	}
	dirSummerizer, err := NewSummarizer("codesummer-directory-summarizer", dbStore)
	if err != nil {
		return nil, err
	}
	emb := embed.NewEmbedderFromConfig(config.Get().CodeMogger.Embedder)
	return &Codesummer{
		db:             dbStore,
		fileSummarizer: fileSummerizer,
		dirSummarizer:  dirSummerizer,
		embedder:       emb,
		classifier:     NewClassifier(),
		extractor:      NewExtractor(),
	}, nil
}

func Summary(ctx context.Context, dbStore *db.Store, codebaseID string) error {
	codebase, err := dbStore.GetCodebaseByID(ctx, codebaseID)
	if err != nil {
		return err
	}
	if codebase == nil {
		return nil
	}

	cs, err := NewCodesummer(dbStore)
	if err != nil {
		return err
	}
	return cs.processCodebase(ctx, codebase)
}

func (c *Codesummer) processCodebase(ctx context.Context, codebase *db.Codebase) error {
	codesummerID, err := c.db.CodesummerGetOrCreateCodebase(ctx, codebase.ID)
	if err != nil {
		return err
	}

	files, dirs, err := c.indexCodebase(ctx, codesummerID, codebase.RootPath)
	if err != nil {
		return err
	}

	allPaths := make([]string, 0, len(files)+len(dirs))
	for path := range files {
		allPaths = append(allPaths, path)
	}
	for path := range dirs {
		allPaths = append(allPaths, path)
	}

	_, err = c.db.CodesummerRemoveStalePaths(ctx, codesummerID, allPaths)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove stale paths")
	}

	log.Debug().Int("files", len(files)).Int("dirs", len(dirs)).Msg("codesummer scanning completed")

	summaries, err := c.summarizeFiles(ctx, codesummerID, files)
	if err != nil {
		return err
	}

	if err := c.summarizeDirectories(ctx, codesummerID, dirs, summaries); err != nil {
		return err
	}

	if err := c.generateEmbeddings(ctx, codesummerID, summaries); err != nil {
		return err
	}

	log.Info().Int("files", len(files)).Int("dirs", len(dirs)).Msg("codesummer summary completed")
	return nil
}

func (c *Codesummer) indexCodebase(
	ctx context.Context,
	codesummerID string,
	rootPath string,
) (map[string]*NodeInfo, map[string]*NodeInfo, error) {
	files := make(map[string]*NodeInfo)
	dirs := make(map[string]*NodeInfo)

	walker := util.StartFileWalker(rootPath, false)
	for f := range walker {
		nodeType, language, err := c.classifier.Classify(f.Location)
		if err != nil {
			log.Error().Err(err).Str("path", f.Location).Msg("failed to classify")
			continue
		}

		relPath, err := filepath.Rel(rootPath, f.Location)
		if err != nil {
			continue
		}

		nodeInfo := &NodeInfo{
			Path:     relPath,
			Type:     nodeType,
			Language: language,
		}

		if nodeType == NodeTypeDirectory {
			children, err := c.classifier.GetChildren(f.Location)
			if err != nil {
				log.Error().Err(err).Str("path", f.Location).Msg("failed to get children")
				continue
			}
			var relChildren []string
			for _, child := range children {
				relChild, err := filepath.Rel(rootPath, child)
				if err != nil {
					continue
				}
				relChildren = append(relChildren, relChild)
			}
			nodeInfo.Children = relChildren
			dirs[relPath] = nodeInfo
		} else {
			content, err := c.extractor.ReadFile(f.Location)
			if err != nil {
				log.Error().Err(err).Str("path", f.Location).Msg("failed to read file")
				continue
			}
			nodeInfo.Content = content

			hash, err := c.extractor.ComputeFileHash(f.Location)
			if err != nil {
				log.Error().Err(err).Str("path", f.Location).Msg("failed to compute hash")
				continue
			}
			nodeInfo.Hash = hash

			if language != "" {
				definitions, err := c.extractor.ExtractDefinitions(f.Location, content, language)
				if err != nil {
					log.Error().Err(err).Str("path", f.Location).Msg("failed to extract definitions")
				}
				nodeInfo.Definitions = definitions
			}

			files[relPath] = nodeInfo
		}

		err = c.db.CodesummerUpsertIndexedPath(ctx, db.IndexedPath{
			CodesummerID: codesummerID,
			NodePath:     relPath,
			NodeType:     nodeType,
			FileHash:     nodeInfo.Hash,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to upsert indexed path")
		}
	}

	return files, dirs, nil
}

func (c *Codesummer) summarizeFiles(
	ctx context.Context,
	codesummerID string,
	files map[string]*NodeInfo,
) (map[string]*NodeSummary, error) {
	summaries := make(map[string]*NodeSummary)

	for path, file := range files {
		log.Debug().Interface("file", file).Msg("started summarizing file")
		summary, err := c.fileSummarizer.SummarizeFile(ctx, file.Language, file.Content, file.Definitions)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("failed to summarize file")
			continue
		}
		summary.NodeInfo = *file
		summaries[path] = summary

		definitionsJSON, _ := json.Marshal(summary.Definitions)
		dependenciesJSON, _ := json.Marshal(summary.Dependencies)
		dataManipulatedJSON, _ := json.Marshal(summary.DataManipulated)
		dataFlowJSON, _ := json.Marshal(summary.DataFlow)

		dbSummary := db.CodesummerSummary{
			CodesummerID:    codesummerID,
			NodePath:        path,
			NodeType:        file.Type,
			Language:        file.Language,
			Summary:         summary.Summary,
			Definitions:     string(definitionsJSON),
			Dependencies:    string(dependenciesJSON),
			DataManipulated: string(dataManipulatedJSON),
			DataFlow:        string(dataFlowJSON),
		}

		err = c.db.CodesummerUpsertSummary(ctx, dbSummary)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("failed to upsert summary")
		}
		log.Debug().Str("path", file.Path).Interface("summary", summary).Msg("finished summarizing file")
	}

	return summaries, nil
}

func (c *Codesummer) summarizeDirectories(
	ctx context.Context,
	codesummerID string,
	dirs map[string]*NodeInfo,
	summaries map[string]*NodeSummary,
) error {
	dirPaths := make([]string, 0, len(dirs))
	for path := range dirs {
		dirPaths = append(dirPaths, path)
	}
	sortPathsByDepth(dirPaths)

	for _, dirPath := range dirPaths {
		dir := dirs[dirPath]
		var childrenSummaries []NodeSummary
		for _, childPath := range dir.Children {
			if childSummary, ok := summaries[childPath]; ok {
				childrenSummaries = append(childrenSummaries, *childSummary)
			}
		}

		log.Debug().Interface("dir", dir).Msg("started summarizing directory")

		summary, err := c.dirSummarizer.SummarizeDirectoryBatch(ctx, dirPath, childrenSummaries)
		if err != nil {
			log.Error().Err(err).Str("path", dirPath).Msg("failed to summarize directory")
			continue
		}
		summary.NodeInfo = *dir
		summaries[dirPath] = &summary

		dependenciesJSON, _ := json.Marshal(summary.Dependencies)
		dataManipulatedJSON, _ := json.Marshal(summary.DataManipulated)
		dataFlowJSON, _ := json.Marshal(summary.DataFlow)

		dbSummary := db.CodesummerSummary{
			CodesummerID:    codesummerID,
			NodePath:        dirPath,
			NodeType:        dir.Type,
			Language:        "",
			Summary:         summary.Summary,
			Definitions:     "[]",
			Dependencies:    string(dependenciesJSON),
			DataManipulated: string(dataManipulatedJSON),
			DataFlow:        string(dataFlowJSON),
		}

		err = c.db.CodesummerUpsertSummary(ctx, dbSummary)
		if err != nil {
			log.Error().Err(err).Str("path", dirPath).Msg("failed to upsert summary")
		}

		log.Debug().Str("dir_path", dir.Path).Interface("summary", summary).Msg("finished summarizing directory")
	}

	return nil
}

func (c *Codesummer) generateEmbeddings(
	ctx context.Context,
	codesummerID string,
	summaries map[string]*NodeSummary,
) error {
	log.Debug().Int("summaries_count", len(summaries)).Msg("started generating embeddings")
	var embeddingsToGenerate []struct {
		NodePath string
		Text     string
	}
	for path, summary := range summaries {
		textToEmbed := summary.Summary
		if len(summary.Dependencies) > 0 {
			textToEmbed += "\n\nDependencies: " + strings.Join(summary.Dependencies, ", ")
		}
		embeddingsToGenerate = append(embeddingsToGenerate, struct {
			NodePath string
			Text     string
		}{path, textToEmbed})
	}

	if len(embeddingsToGenerate) == 0 {
		return nil
	}

	texts := make([]string, len(embeddingsToGenerate))
	for i, e := range embeddingsToGenerate {
		texts[i] = e.Text
	}

	embeddings, err := c.embedder.Embed(texts)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate embeddings")
		return err
	}

	var embeddingItems []struct {
		CodesummerID string
		NodePath     string
		Embedding    []float32
		ModelName    string
	}
	for i, e := range embeddingsToGenerate {
		embeddingItems = append(embeddingItems, struct {
			CodesummerID string
			NodePath     string
			Embedding    []float32
			ModelName    string
		}{codesummerID, e.NodePath, embeddings[i], c.embedder.Model()})
	}
	err = c.db.CodesummerUpsertEmbeddings(ctx, embeddingItems)
	if err != nil {
		log.Error().Err(err).Msg("failed to upsert embeddings")
		return err
	}

	log.Debug().Msg("finished generating embeddings")

	return nil
}

func sortPathsByDepth(paths []string) {
	for i := 0; i < len(paths)-1; i++ {
		for j := i + 1; j < len(paths); j++ {
			if strings.Count(paths[i], "/") < strings.Count(paths[j], "/") {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
}
