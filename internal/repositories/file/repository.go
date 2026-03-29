package file

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Repository реализует файловое blob-хранилище.
type Repository struct {
	basePath string
}

// New создаёт файловое хранилище по указанному пути.
func New(basePath string) (*Repository, error) {
	if basePath == "" {
		return nil, errors.New("blob base path is required")
	}
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, err
	}
	return &Repository{basePath: basePath}, nil
}

// Save сохраняет бинарные данные по относительному пути.
func (r *Repository) Save(path string, data []byte) error {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return err
	}
	tmpPath := fullPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, fullPath)
}

// Read возвращает бинарные данные по относительному пути.
func (r *Repository) Read(path string) ([]byte, error) {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

// Delete удаляет бинарные данные по относительному пути.
func (r *Repository) Delete(path string) error {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Exists проверяет наличие данных по относительному пути.
func (r *Repository) Exists(path string) (bool, error) {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (r *Repository) resolvePath(path string) (string, error) {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New("invalid blob path")
	}
	return filepath.Join(r.basePath, clean), nil
}
