package util

import (
	"os"
	"path/filepath"
	"strings"
)

func IsDev() bool {
	return os.Getenv("APP_ENV") == "development"
}

func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~/") && path != "~" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	return filepath.Join(home, path[2:])
}
