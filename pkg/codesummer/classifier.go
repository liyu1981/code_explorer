package codesummer

import (
	"os"
	"path/filepath"

	"github.com/liyu1981/code_explorer/pkg/codemogger/chunk"
)

const (
	NodeTypeSourceFile = "source_file"
	NodeTypeNormalFile = "normal_file"
	NodeTypeDirectory  = "directory"
)

type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

func (c *Classifier) Classify(path string) (string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}

	if info.IsDir() {
		return NodeTypeDirectory, "", nil
	}

	lang := chunk.DetectLanguage(path)
	if lang != nil {
		return NodeTypeSourceFile, lang.Name, nil
	}

	return NodeTypeNormalFile, "", nil
}

func (c *Classifier) GetChildren(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var children []string
	for _, entry := range entries {
		childPath := filepath.Join(dirPath, entry.Name())
		children = append(children, childPath)
	}

	return children, nil
}
