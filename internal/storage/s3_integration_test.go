package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/config"
)

func TestS3Blob_MinIOIntegration(t *testing.T) {
	endpoint := os.Getenv("GK_TEST_MINIO_ENDPOINT")
	if endpoint == "" {
		t.Skip("GK_TEST_MINIO_ENDPOINT is not set")
	}

	cfg := config.BlobStorageConfig{
		Provider:  "s3",
		Endpoint:  endpoint,
		Bucket:    envOrDefault("GK_TEST_MINIO_BUCKET", "gophkeeper-dev"),
		AccessKey: envOrDefault("GK_TEST_MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: envOrDefault("GK_TEST_MINIO_SECRET_KEY", "minioadmin"),
		Region:    envOrDefault("GK_TEST_MINIO_REGION", "us-east-1"),
	}

	blob, err := NewS3Blob(cfg)
	require.NoError(t, err)

	key := fmt.Sprintf("integration/%d/payload.bin", time.Now().UnixNano())
	payload := []byte("minio-roundtrip")

	t.Cleanup(func() {
		require.NoError(t, blob.Delete(key))
	})

	require.NoError(t, blob.Save(key, payload))

	exists, err := blob.Exists(key)
	require.NoError(t, err)
	require.True(t, exists)

	got, err := blob.Read(key)
	require.NoError(t, err)
	require.Equal(t, payload, got)

	require.NoError(t, blob.Delete(key))

	exists, err = blob.Exists(key)
	require.NoError(t, err)
	require.False(t, exists)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
