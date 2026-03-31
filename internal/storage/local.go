package storage

import (
	"fmt"
	"os"
	"path/filepath"
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

// Save сохраняет бинарные данные по относительному пути внутри baseDir.
func (l *LocalBlob) Save(path string, data []byte) error {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0o644)
}

// Read читает бинарные данные по относительному пути внутри baseDir.
func (l *LocalBlob) Read(path string) ([]byte, error) {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

// Delete удаляет объект по относительному пути внутри baseDir.
func (l *LocalBlob) Delete(path string) error {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return err
	}
	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Exists проверяет наличие объекта по относительному пути внутри baseDir.
func (l *LocalBlob) Exists(path string) (bool, error) {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *LocalBlob) resolvePath(path string) (string, error) {
	key, err := normalizeBlobPath(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(l.baseDir, key), nil
}
