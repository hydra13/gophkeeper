package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/hydra13/gophkeeper/internal/config"
)

type s3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// S3Blob хранит бинарные данные в S3-compatible object storage.
type S3Blob struct {
	client s3Client
	bucket string
}

func NewS3Blob(cfg config.BlobStorageConfig) (*S3Blob, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("s3 endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("s3 bucket is required")
	}
	if cfg.AccessKey == "" {
		return nil, errors.New("s3 access key is required")
	}
	if cfg.SecretKey == "" {
		return nil, errors.New("s3 secret key is required")
	}
	if cfg.Region == "" {
		return nil, errors.New("s3 region is required")
	}

	client := newS3Client(cfg)
	blob := &S3Blob{
		client: client,
		bucket: cfg.Bucket,
	}
	if err := blob.verifyBucket(context.Background()); err != nil {
		return nil, err
	}
	return blob, nil
}

func newS3Client(cfg config.BlobStorageConfig) s3Client {
	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: aws.NewCredentialsCache(staticCredentialsProvider{
			accessKey: cfg.AccessKey,
			secretKey: cfg.SecretKey,
		}),
		HTTPClient: http.DefaultClient,
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})
}

type staticCredentialsProvider struct {
	accessKey string
	secretKey string
}

func (p staticCredentialsProvider) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     p.accessKey,
		SecretAccessKey: p.secretKey,
		Source:          "gophkeeper-static",
	}, nil
}

func (s *S3Blob) Save(path string, data []byte) error {
	key, err := normalizeBlobPath(path)
	if err != nil {
		return err
	}

	_, err = s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (s *S3Blob) Read(path string) ([]byte, error) {
	key, err := normalizeBlobPath(path)
	if err != nil {
		return nil, err
	}

	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, fmt.Errorf("get object %q: %w", key, os.ErrNotExist)
		}
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read object %q: %w", key, err)
	}
	return data, nil
}

func (s *S3Blob) Delete(path string) error {
	key, err := normalizeBlobPath(path)
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil && !isS3NotFound(err) {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

func (s *S3Blob) Exists(path string) (bool, error) {
	key, err := normalizeBlobPath(path)
	if err != nil {
		return false, err
	}

	_, err = s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("head object %q: %w", key, err)
	}

	return true, nil
}

func (s *S3Blob) verifyBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("head bucket %q: %w", s.bucket, err)
	}
	return nil
}

func isS3NotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey", "NoSuchUpload":
			return true
		}
	}
	return false
}
