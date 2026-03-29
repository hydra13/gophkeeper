package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hydra13/gophkeeper/internal/repositories"
)

// LocalBlob хранит бинарные данные в локальной файловой системе.
type LocalBlob struct {
	baseDir string
}

// NewLocalBlob создаёт LocalBlob с базовой директорией baseDir.
func NewLocalBlob(baseDir string) (*LocalBlob, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("base directory is required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	return &LocalBlob{baseDir: baseDir}, nil
}

// Verify interface compliance.
var _ repositories.BlobStorage = (*LocalBlob)(nil)

func (l *LocalBlob) Save(path string, data []byte) error {
	fullPath := filepath.Join(l.baseDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0o644)
}

func (l *LocalBlob) Read(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(l.baseDir, path))
}

func (l *LocalBlob) Delete(path string) error {
	err := os.Remove(filepath.Join(l.baseDir, path))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *LocalBlob) Exists(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(l.baseDir, path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
