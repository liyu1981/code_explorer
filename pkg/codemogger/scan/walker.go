package scan

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boyter/gocodewalker"
	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
	"github.com/rs/zerolog/log"
)

type ScannedFile struct {
	AbsPath string
	Content string
	Hash    string
}

func ScanDirectory(rootDir string, languages []string) ([]ScannedFile, []string) {
	log.Debug().Str("rootDir", rootDir).Interface("languages", languages).Msg("ScanDirectory started")
	var files []ScannedFile
	var errors []string

	supportedExts := chunk.SupportedExtensions()
	// ... (logic for languages filter)
	if len(languages) > 0 {
		var langExts []string
		for _, lang := range languages {
			if cfg, ok := chunk.Languages[lang]; ok {
				langExts = append(langExts, cfg.Extensions...)
			} else {
				if !strings.HasPrefix(lang, ".") {
					lang = "." + lang
				}
				if cfg := chunk.DetectLanguage("test" + lang); cfg != nil {
					langExts = append(langExts, cfg.Extensions...)
				}
			}
		}
		if len(langExts) > 0 {
			supportedExts = langExts
		}
	}

	log.Debug().Interface("supportedExts", supportedExts).Msg("Extensions filtered")

	var allowExts []string
	for _, ext := range supportedExts {
		allowExts = append(allowExts, strings.TrimPrefix(ext, "."))
	}

	fileListQueue := make(chan *gocodewalker.File, 1000)
	fileWalker := gocodewalker.NewFileWalker(rootDir, fileListQueue)
	fileWalker.AllowListExtensions = allowExts
	fileWalker.IgnoreBinaryFiles = false
	fileWalker.SetErrorHandler(func(err error) bool {
		log.Error().Err(err).Msg("FileWalker error")
		errors = append(errors, err.Error())
		return true
	})

	log.Debug().Msg("Starting gocodewalker...")
	go fileWalker.Start()

	for f := range fileListQueue {
		// Use absolute path
		absPath, err := filepath.Abs(f.Location)
		if err != nil {
			absPath = f.Location
		}

		log.Trace().Str("file", absPath).Msg("Processing file")
		content, err := os.ReadFile(f.Location)
		if err != nil {
			log.Warn().Str("file", f.Location).Err(err).Msg("Failed to read file")
			errors = append(errors, f.Location+": "+err.Error())
			continue
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(content))

		files = append(files, ScannedFile{
			AbsPath: absPath,
			Content: string(content),
			Hash:    hash,
		})
	}

	log.Debug().Int("scannedFiles", len(files)).Msg("ScanDirectory finished")
	return files, errors
}
