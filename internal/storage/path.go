package storage

import (
	"errors"
	"path/filepath"
	"strings"
)

func normalizeBlobPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("blob path is required")
	}

	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == "" {
		return "", errors.New("blob path is required")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", errors.New("invalid blob path")
	}

	return clean, nil
}
