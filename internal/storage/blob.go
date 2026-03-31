package storage

import (
	"fmt"

	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/repositories"
)

var (
	newLocalBlob = func(path string) (repositories.BlobStorage, error) {
		return NewLocalBlob(path)
	}
	newS3Blob = func(cfg config.BlobStorageConfig) (repositories.BlobStorage, error) {
		return NewS3Blob(cfg)
	}
)

// NewBlobStorage создаёт blob-хранилище по конфигурации провайдера.
func NewBlobStorage(cfg config.BlobStorageConfig) (repositories.BlobStorage, error) {
	switch cfg.Provider {
	case "local":
		return newLocalBlob(cfg.Path)
	case "s3":
		return newS3Blob(cfg)
	default:
		return nil, fmt.Errorf("unsupported blob provider: %s", cfg.Provider)
	}
}
