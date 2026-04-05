package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/repositories"
)

func TestNewBlobStorage_Local(t *testing.T) {
	origLocal := newLocalBlob
	origS3 := newS3Blob
	t.Cleanup(func() {
		newLocalBlob = origLocal
		newS3Blob = origS3
	})

	var gotPath string
	newLocalBlob = func(path string) (repositories.BlobStorage, error) {
		gotPath = path
		return &LocalBlob{}, nil
	}
	newS3Blob = func(config.BlobStorageConfig) (repositories.BlobStorage, error) {
		return nil, errors.New("unexpected s3 constructor call")
	}

	blob, err := NewBlobStorage(config.BlobStorageConfig{
		Provider: "local",
		Path:     "blob-dir",
	})
	require.NoError(t, err)
	require.IsType(t, &LocalBlob{}, blob)
	require.Equal(t, "blob-dir", gotPath)
}

func TestNewBlobStorage_S3(t *testing.T) {
	origLocal := newLocalBlob
	origS3 := newS3Blob
	t.Cleanup(func() {
		newLocalBlob = origLocal
		newS3Blob = origS3
	})

	wantCfg := config.BlobStorageConfig{
		Provider:  "s3",
		Endpoint:  "http://localhost:9000",
		Bucket:    "gophkeeper-dev",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Region:    "us-east-1",
	}
	var gotCfg config.BlobStorageConfig
	newLocalBlob = func(string) (repositories.BlobStorage, error) {
		return nil, errors.New("unexpected local constructor call")
	}
	newS3Blob = func(cfg config.BlobStorageConfig) (repositories.BlobStorage, error) {
		gotCfg = cfg
		return &S3Blob{}, nil
	}

	blob, err := NewBlobStorage(wantCfg)
	require.NoError(t, err)
	require.IsType(t, &S3Blob{}, blob)
	require.Equal(t, wantCfg, gotCfg)
}

func TestNewBlobStorage_UnsupportedProvider(t *testing.T) {
	blob, err := NewBlobStorage(config.BlobStorageConfig{
		Provider: "gcs",
	})
	require.Error(t, err)
	require.Nil(t, blob)
	require.Contains(t, err.Error(), "unsupported blob provider")
}
