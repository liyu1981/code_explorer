package scan

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boyter/gocodewalker"
	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
)

type ScannedFile struct {
	AbsPath string
	Content string
	Hash    string
}

func ScanDirectory(rootDir string, languages []string) ([]ScannedFile, []string) {
	var files []ScannedFile
	var errors []string

	supportedExts := chunk.SupportedExtensions()
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

	extSet := make(map[string]bool)
	for _, ext := range supportedExts {
		extSet[ext] = true
	}

	var allowExts []string
	for _, ext := range supportedExts {
		allowExts = append(allowExts, strings.TrimPrefix(ext, "."))
	}

	fileListQueue := make(chan *gocodewalker.File, 1000)
	fileWalker := gocodewalker.NewFileWalker(rootDir, fileListQueue)
	fileWalker.AllowListExtensions = allowExts
	// IgnoreBinaryFiles=true seems to misidentify small text files in tests.
	// We'll rely on AllowListExtensions and .gitignore instead.
	fileWalker.IgnoreBinaryFiles = false
	fileWalker.SetErrorHandler(func(err error) bool {
		errors = append(errors, err.Error())
		return true
	})

	go fileWalker.Start()

	for f := range fileListQueue {
		// Use absolute path
		absPath, err := filepath.Abs(f.Location)
		if err != nil {
			absPath = f.Location
		}

		content, err := os.ReadFile(f.Location)
		if err != nil {
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

	return files, errors
}
