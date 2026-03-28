package scan

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
	"github.com/liyu1981/code_explorer/pkg/util"
	"github.com/rs/zerolog/log"
)

type ScannedFile struct {
	AbsPath string
	RelPath string
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

	fileListQueue := util.StartFileWalker(rootDir, false)

	for f := range fileListQueue {
		// Filter by extension if allowExts is not empty
		if len(allowExts) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(f.Location), ".")
			found := false
			for _, allowed := range allowExts {
				if ext == allowed {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Get both absolute and relative paths
		absPath, err := filepath.Abs(f.Location)
		if err != nil {
			absPath = f.Location
		}
		relPath, err := filepath.Rel(rootDir, f.Location)
		if err != nil {
			log.Warn().Str("file", f.Location).Err(err).Msg("Failed to get relative path")
			errors = append(errors, f.Location+": "+err.Error())
			continue
		}

		log.Trace().Str("file", relPath).Msg("Processing file")
		content, err := os.ReadFile(f.Location)
		if err != nil {
			log.Warn().Str("file", f.Location).Err(err).Msg("Failed to read file")
			errors = append(errors, f.Location+": "+err.Error())
			continue
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(content))

		files = append(files, ScannedFile{
			AbsPath: absPath,
			RelPath: relPath,
			Content: string(content),
			Hash:    hash,
		})
	}

	log.Debug().Int("scannedFiles", len(files)).Msg("ScanDirectory finished")
	return files, errors
}
