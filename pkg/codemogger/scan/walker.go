package scan

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
			if cfg := chunk.DetectLanguage("test." + lang); cfg != nil {
				langExts = append(langExts, cfg.Extensions...)
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

	ignorePatterns := []string{
		".git", "node_modules", "vendor", ".venv", "dist", "build",
		".next", ".nuxt", "target", "__pycache__", ".pytest_cache",
		".idea", ".vscode", "*.pb.go", "*.pb.gw.go",
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}

		if info.IsDir() {
			for _, pattern := range ignorePatterns {
				if strings.HasPrefix(relPath, pattern) || relPath == pattern {
					return filepath.SkipDir
				}
			}
			return nil
		}

		ext := filepath.Ext(path)
		if !extSet[ext] {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			errors = append(errors, path+": "+err.Error())
			return nil
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(content))

		files = append(files, ScannedFile{
			AbsPath: path,
			Content: string(content),
			Hash:    hash,
		})

		return nil
	})

	if err != nil {
		errors = append(errors, err.Error())
	}

	return files, errors
}
