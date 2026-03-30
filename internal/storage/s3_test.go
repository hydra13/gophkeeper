package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/config"
)

type mockS3Client struct {
	putObjectFunc    func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFunc    func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFunc func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	headObjectFunc   func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	headBucketFunc   func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.putObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.deleteObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return m.headObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return m.headBucketFunc(ctx, params, optFns...)
}

func TestS3Blob_SaveReadDeleteExists(t *testing.T) {
	t.Parallel()

	store := make(map[string][]byte)
	blob := &S3Blob{
		bucket: "test-bucket",
		client: &mockS3Client{
			putObjectFunc: func(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				data, err := io.ReadAll(params.Body)
				require.NoError(t, err)
				store[aws.ToString(params.Key)] = data
				require.Equal(t, "test-bucket", aws.ToString(params.Bucket))
				return &s3.PutObjectOutput{}, nil
			},
			getObjectFunc: func(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				data, ok := store[aws.ToString(params.Key)]
				if !ok {
					return nil, notFoundAPIError{code: "NoSuchKey"}
				}
				return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
			},
			deleteObjectFunc: func(_ context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				delete(store, aws.ToString(params.Key))
				return &s3.DeleteObjectOutput{}, nil
			},
			headObjectFunc: func(_ context.Context, params *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				if _, ok := store[aws.ToString(params.Key)]; !ok {
					return nil, notFoundAPIError{code: "NotFound"}
				}
				return &s3.HeadObjectOutput{}, nil
			},
			headBucketFunc: func(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	require.NoError(t, blob.Save("a/b.bin", []byte("payload")))

	exists, err := blob.Exists("a/b.bin")
	require.NoError(t, err)
	require.True(t, exists)

	got, err := blob.Read("a/b.bin")
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), got)

	require.NoError(t, blob.Delete("a/b.bin"))

	exists, err = blob.Exists("a/b.bin")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestS3Blob_Exists_NotFound(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "test-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, notFoundAPIError{code: "NotFound"}
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	exists, err := blob.Exists("missing.bin")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestS3Blob_Delete_NotFoundIgnored(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "test-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, notFoundAPIError{code: "NoSuchKey"}
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	require.NoError(t, blob.Delete("missing.bin"))
}

func TestS3Blob_Delete_MissingBucketReturnsError(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "missing-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, notFoundAPIError{code: "NoSuchBucket"}
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	err := blob.Delete("missing.bin")
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete object")
	require.Contains(t, err.Error(), "NoSuchBucket")
}

func TestS3Blob_Read_NotFoundMappedToErrNotExist(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "test-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, notFoundAPIError{code: "NoSuchKey"}
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	_, err := blob.Read("missing.bin")
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestS3Blob_Read_MissingBucketReturnsBackendError(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "missing-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, notFoundAPIError{code: "NoSuchBucket"}
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	_, err := blob.Read("missing.bin")
	require.Error(t, err)
	require.NotErrorIs(t, err, os.ErrNotExist)
	require.Contains(t, err.Error(), "get object")
	require.Contains(t, err.Error(), "NoSuchBucket")
}

func TestS3Blob_Exists_MissingBucketReturnsBackendError(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "missing-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, notFoundAPIError{code: "NoSuchBucket"}
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			},
		},
	}

	exists, err := blob.Exists("missing.bin")
	require.Error(t, err)
	require.False(t, exists)
	require.Contains(t, err.Error(), "head object")
	require.Contains(t, err.Error(), "NoSuchBucket")
}

func TestNewS3Blob_RequiresConfigFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.BlobStorageConfig
		wantErr string
	}{
		{
			name: "missing endpoint",
			cfg: config.BlobStorageConfig{
				Bucket:    "bucket",
				AccessKey: "ak",
				SecretKey: "sk",
				Region:    "us-east-1",
			},
			wantErr: "s3 endpoint is required",
		},
		{
			name: "missing bucket",
			cfg: config.BlobStorageConfig{
				Endpoint:  "http://localhost:9000",
				AccessKey: "ak",
				SecretKey: "sk",
				Region:    "us-east-1",
			},
			wantErr: "s3 bucket is required",
		},
		{
			name: "missing access key",
			cfg: config.BlobStorageConfig{
				Endpoint:  "http://localhost:9000",
				Bucket:    "bucket",
				SecretKey: "sk",
				Region:    "us-east-1",
			},
			wantErr: "s3 access key is required",
		},
		{
			name: "missing secret key",
			cfg: config.BlobStorageConfig{
				Endpoint:  "http://localhost:9000",
				Bucket:    "bucket",
				AccessKey: "ak",
				Region:    "us-east-1",
			},
			wantErr: "s3 secret key is required",
		},
		{
			name: "missing region",
			cfg: config.BlobStorageConfig{
				Endpoint:  "http://localhost:9000",
				Bucket:    "bucket",
				AccessKey: "ak",
				SecretKey: "sk",
			},
			wantErr: "s3 region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob, err := NewS3Blob(tt.cfg)
			require.Error(t, err)
			require.Nil(t, blob)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestS3Blob_VerifyBucket_Error(t *testing.T) {
	t.Parallel()

	blob := &S3Blob{
		bucket: "test-bucket",
		client: &mockS3Client{
			putObjectFunc: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			getObjectFunc: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			deleteObjectFunc: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headObjectFunc: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("unexpected call")
			},
			headBucketFunc: func(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return nil, errors.New("boom")
			},
		},
	}

	err := blob.verifyBucket(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "head bucket")
}

type notFoundAPIError struct {
	code string
}

func (e notFoundAPIError) Error() string {
	return e.code
}

func (e notFoundAPIError) ErrorCode() string {
	return e.code
}

func (e notFoundAPIError) ErrorMessage() string {
	return e.code
}

func (e notFoundAPIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultClient
}
