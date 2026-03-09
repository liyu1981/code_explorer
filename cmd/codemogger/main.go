package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/codemogger/format"
	"github.com/liyu1981/code_explorer/pkg/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	logger.Init()
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "index":
		handleIndex()
	case "search":
		handleSearch()
	case "list-files":
		handleListFiles()
	case "list-codebases":
		handleListCodebases()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: codemogger <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  index <dir>         Index a directory")
	fmt.Println("  search <query>      Search indexed codebases")
	fmt.Println("  list-files          List indexed files")
	fmt.Println("  list-codebases      List indexed codebases")
}

func getIndex(dbPath string) (*codemogger.CodeIndex, error) {
	cfg := codemogger.DefaultConfig()

	// Try to load config from file
	configPath := os.Getenv("CODE_EXPLORER_CONFIG")
	if configPath == "" {
		if _, err := os.Stat(".config.json"); err == nil {
			configPath = ".config.json"
		} else {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, ".code_explorer", "config.json")
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			var fileCfg codemogger.Config
			if err := json.Unmarshal(data, &fileCfg); err == nil {
				// Merge configs
				if fileCfg.System.DBPath != "" {
					cfg.System.DBPath = fileCfg.System.DBPath
				}
				if fileCfg.System.LLM != nil {
					cfg.System.LLM = fileCfg.System.LLM
				}
				if fileCfg.Research.MaxReportsPerCodebase > 0 {
					cfg.Research.MaxReportsPerCodebase = fileCfg.Research.MaxReportsPerCodebase
				}
				if fileCfg.CodeMogger.Embedder.Type != "" {
					cfg.CodeMogger.Embedder.Type = fileCfg.CodeMogger.Embedder.Type
				}
				if fileCfg.CodeMogger.Embedder.Model != "" {
					cfg.CodeMogger.Embedder.Model = fileCfg.CodeMogger.Embedder.Model
				}
				if fileCfg.CodeMogger.Embedder.OpenAI.APIBase != "" {
					cfg.CodeMogger.Embedder.OpenAI.APIBase = fileCfg.CodeMogger.Embedder.OpenAI.APIBase
				}
				if fileCfg.CodeMogger.Embedder.OpenAI.APIKey != "" {
					cfg.CodeMogger.Embedder.OpenAI.APIKey = fileCfg.CodeMogger.Embedder.OpenAI.APIKey
				}
				if fileCfg.CodeMogger.Embedder.OpenAI.Model != "" {
					cfg.CodeMogger.Embedder.OpenAI.Model = fileCfg.CodeMogger.Embedder.OpenAI.Model
				}
				cfg.CodeMogger.InheritSystemLLM = fileCfg.CodeMogger.InheritSystemLLM
				if fileCfg.CodeMogger.ChunkLines > 0 {
					cfg.CodeMogger.ChunkLines = fileCfg.CodeMogger.ChunkLines
				}
				if len(fileCfg.CodeMogger.Languages) > 0 {
					cfg.CodeMogger.Languages = fileCfg.CodeMogger.Languages
				}
			}
		}
	}

	if dbPath == "" {
		if cfg.System.DBPath != "" {
			dbPath = cfg.System.DBPath
		} else {
			dbPath = codemogger.ProjectDbPath(".")
		}
	}

	// Create db directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	return codemogger.NewCodeIndex(dbPath, cfg, configPath)
}

func handleIndex() {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	dbPath := indexCmd.String("db", "", "Path to database")
	langs := indexCmd.String("langs", "", "Comma-separated list of languages to index")

	indexCmd.Parse(os.Args[2:])

	if indexCmd.NArg() < 1 {
		fmt.Println("Usage: codemogger index <dir> [-db <path>] [-langs <go,py>]")
		os.Exit(1)
	}

	dir := indexCmd.Arg(0)
	idx, err := getIndex(*dbPath)
	if err != nil {
		log.Fatal().Msgf("Failed to open index: %v", err)
	}
	defer idx.Close()

	opts := &codemogger.IndexOptions{}
	if *langs != "" {
		opts.Languages = strings.Split(*langs, ",")
	}

	opts.Progress = func(current, total int, stage string) {
		fmt.Printf("\r[%s] %d/%d...", stage, current, total)
		if current == total {
			fmt.Println()
		}
	}

	fmt.Printf("Indexing %s...\n", dir)
	res, err := idx.Index(dir, opts)
	if err != nil {
		log.Fatal().Msgf("\nIndexing failed: %v", err)
	}

	fmt.Printf("Processed %d files, created %d chunks, embedded %d chunks, skipped %d, removed %d stale files\n",
		res.Files, res.Chunks, res.Embedded, res.Skipped, res.Removed)
	fmt.Printf("Duration: %d ms\n", res.Duration)
	if len(res.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range res.Errors {
			fmt.Printf(" - %s\n", e)
		}
	}
}

func handleSearch() {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	dbPath := searchCmd.String("db", "", "Path to database")
	limit := searchCmd.Int("limit", 5, "Number of results to return")
	mode := searchCmd.String("mode", "hybrid", "Search mode (semantic, keyword, hybrid)")
	output := searchCmd.String("output", "text", "Output format (text, json)")

	searchCmd.Parse(os.Args[2:])

	if searchCmd.NArg() < 1 {
		fmt.Println("Usage: codemogger search <query> [-db <path>] [-limit <n>] [-mode <hybrid>]")
		os.Exit(1)
	}

	query := searchCmd.Arg(0)
	idx, err := getIndex(*dbPath)
	if err != nil {
		log.Fatal().Msgf("Failed to open index: %v", err)
	}
	defer idx.Close()

	opts := &codemogger.SearchOptions{
		Limit:          *limit,
		Mode:           codemogger.SearchMode(*mode),
		IncludeSnippet: true,
	}

	results, err := idx.Search(query, opts)
	if err != nil {
		log.Fatal().Msgf("Search failed: %v", err)
	}

	if *output == "json" {
		data, _ := format.JSON(results)
		fmt.Println(string(data))
	} else {
		fmt.Print(format.Text(results))
	}
}

func handleListFiles() {
	listCmd := flag.NewFlagSet("list-files", flag.ExitOnError)
	dbPath := listCmd.String("db", "", "Path to database")
	listCmd.Parse(os.Args[2:])

	idx, err := getIndex(*dbPath)
	if err != nil {
		log.Fatal().Msgf("Failed to open index: %v", err)
	}
	defer idx.Close()

	files, err := idx.ListFiles()
	if err != nil {
		log.Fatal().Msgf("Failed to list files: %v", err)
	}

	fmt.Printf("%-50s %-20s %-10s\n", "Path", "Indexed At", "Chunks")
	fmt.Println(strings.Repeat("-", 85))
	for _, f := range files {
		indexedAt := time.Unix(f.IndexedAt, 0).Format("2006-01-02 15:04:05")
		fmt.Printf("%-50s %-20s %-10d\n", f.FilePath, indexedAt, f.ChunkCount)
	}
}

func handleListCodebases() {
	listCmd := flag.NewFlagSet("list-codebases", flag.ExitOnError)
	dbPath := listCmd.String("db", "", "Path to database")
	listCmd.Parse(os.Args[2:])

	idx, err := getIndex(*dbPath)
	if err != nil {
		log.Fatal().Msgf("Failed to open index: %v", err)
	}
	defer idx.Close()

	codebases, err := idx.ListCodebases()
	if err != nil {
		log.Fatal().Msgf("Failed to list codebases: %v", err)
	}

	fmt.Printf("%-5s %-40s %-20s %-10s %-10s\n", "ID", "Path", "Indexed At", "Files", "Chunks")
	fmt.Println(strings.Repeat("-", 95))
	for _, c := range codebases {
		indexedAt := time.Unix(c.IndexedAt, 0).Format("2006-01-02 15:04:05")
		fmt.Printf("%-5d %-40s %-20s %-10d %-10d\n", c.ID, c.RootPath, indexedAt, c.FileCount, c.ChunkCount)
	}
}
