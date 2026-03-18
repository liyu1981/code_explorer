package codesummer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/codemogger/embed"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/util"
	"github.com/rs/zerolog/log"
)

type Codesummer struct {
	db         *db.Store
	summarizer *Summarizer
	embedder   *CodesummerEmbedder
	classifier *Classifier
	extractor  *Extractor
}

func NewCodesummer(
	dbStore *db.Store,
	summarizer *Summarizer,
	embedder *CodesummerEmbedder,
) *Codesummer {
	return &Codesummer{
		db:         dbStore,
		summarizer: summarizer,
		embedder:   embedder,
		classifier: NewClassifier(),
		extractor:  NewExtractor(),
	}
}

func Summary(ctx context.Context, dbStore *db.Store, codebaseID string) error {
	codebase, err := dbStore.GetCodebaseByID(ctx, codebaseID)
	if err != nil {
		return err
	}
	if codebase == nil {
		return nil
	}

	llm := agent.NewHTTPClientLLM(
		os.Getenv("LLM_MODEL"),
		os.Getenv("LLM_BASE_URL"),
		os.Getenv("LLM_API_KEY"),
	)

	promptBuilder, err := NewPromptBuilder(ctx, dbStore)
	if err != nil {
		return err
	}

	emb := embed.NewOpenAIEmbedder(
		os.Getenv("LLM_BASE_URL"),
		os.Getenv("EMBED_MODEL"),
		os.Getenv("LLM_API_KEY"),
		1536,
	)

	summer, err := NewSummarizer(llm, promptBuilder)
	if err != nil {
		return err
	}

	codesummerEmbedder := NewCodesummerEmbedder(emb)

	cs := NewCodesummer(dbStore, summer, codesummerEmbedder)
	return cs.processCodebase(ctx, codebase.RootPath, codebase.ID)
}

func (c *Codesummer) processCodebase(ctx context.Context, rootPath string, codebaseID string) error {
	codebase, err := c.db.GetCodebaseByID(ctx, codebaseID)
	if err != nil {
		return err
	}
	if codebase == nil {
		log.Error().Str("codebaseID", codebaseID).Msg("codebase not found")
		return nil
	}

	codesummerID, err := c.db.CodesummerGetOrCreateCodebase(ctx, codebase.ID)
	if err != nil {
		return err
	}

	files := make(map[string]*NodeInfo)
	dirs := make(map[string]*NodeInfo)

	walker := util.StartFileWalker(codebase.RootPath, false)
	for f := range walker {
		nodeType, language, err := c.classifier.Classify(f.Location)
		if err != nil {
			log.Error().Err(err).Str("path", f.Location).Msg("failed to classify")
			continue
		}

		relPath, err := filepath.Rel(codebase.RootPath, f.Location)
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
				relChild, err := filepath.Rel(codebase.RootPath, child)
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

	summaries := make(map[string]*NodeSummary)

	for path, file := range files {
		summary, err := c.summarizer.SummarizeFile(ctx, file.Language, file.Content, file.Definitions)
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
	}

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

		summary, err := c.summarizer.SummarizeDirectoryBatch(ctx, dirPath, childrenSummaries)
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
	}

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

	if len(embeddingsToGenerate) > 0 {
		texts := make([]string, len(embeddingsToGenerate))
		for i, e := range embeddingsToGenerate {
			texts[i] = e.Text
		}

		embeddings, err := c.embedder.EmbedText(texts)
		if err != nil {
			log.Error().Err(err).Msg("failed to generate embeddings")
		} else {
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
			}
		}
	}

	log.Info().Int("files", len(files)).Int("dirs", len(dirs)).Msg("codesummer summary completed")
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
