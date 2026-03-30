package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validConfig(t *testing.T) *Config {
	t.Helper()

	certFile := filepath.Join(t.TempDir(), "cert.pem")
	keyFile := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0o644))
	require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0o644))

	return &Config{
		Server: ServerConfig{
			Address:     ":8080",
			GRPCAddress: ":9090",
			TLSCertFile: certFile,
			TLSKeyFile:  keyFile,
		},
		Database: DatabaseConfig{
			DSN: "postgres://user:pass@localhost:5432/db",
		},
		Auth: AuthConfig{
			JWTSecret:     "secret",
			TokenDuration: time.Hour,
		},
		Crypto: CryptoConfig{
			MasterKey: "0123456789abcdef0123456789abcdef",
		},
		Blob: BlobStorageConfig{
			Provider: "local",
			Path:     "/tmp/blobs",
		},
		Upload: UploadLimitsConfig{
			MaxFileSize:  1048576,
			MaxChunkSize: 65536,
			MaxTotalSize: 10485760,
		},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig(t)
	require.NoError(t, Validate(cfg))
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr string
	}{
		{
			name:   "missing server.address",
			mutate: func(cfg *Config) { cfg.Server.Address = "" },
			wantErr: "server.address is required",
		},
		{
			name:   "missing server.grpc_address",
			mutate: func(cfg *Config) { cfg.Server.GRPCAddress = "" },
			wantErr: "server.grpc_address is required",
		},
		{
			name:   "missing server.tls_cert_file",
			mutate: func(cfg *Config) { cfg.Server.TLSCertFile = "" },
			wantErr: "server.tls_cert_file is required",
		},
		{
			name:   "missing server.tls_key_file",
			mutate: func(cfg *Config) { cfg.Server.TLSKeyFile = "" },
			wantErr: "server.tls_key_file is required",
		},
		{
			name:   "missing database.dsn",
			mutate: func(cfg *Config) { cfg.Database.DSN = "" },
			wantErr: "database.dsn is required",
		},
		{
			name:   "missing auth.jwt_secret",
			mutate: func(cfg *Config) { cfg.Auth.JWTSecret = "" },
			wantErr: "auth.jwt_secret is required",
		},
		{
			name:   "zero auth.token_duration",
			mutate: func(cfg *Config) { cfg.Auth.TokenDuration = 0 },
			wantErr: "auth.token_duration must be positive",
		},
		{
			name:   "negative auth.token_duration",
			mutate: func(cfg *Config) { cfg.Auth.TokenDuration = -1 },
			wantErr: "auth.token_duration must be positive",
		},
		{
			name:   "missing crypto.master_key",
			mutate: func(cfg *Config) { cfg.Crypto.MasterKey = "" },
			wantErr: "crypto.master_key is required",
		},
		{
			name:   "missing blob.provider",
			mutate: func(cfg *Config) { cfg.Blob.Provider = "" },
			wantErr: "blob.provider is required",
		},
		{
			name:   "missing blob.path for local provider",
			mutate: func(cfg *Config) { cfg.Blob.Path = "" },
			wantErr: "blob.path is required for local provider",
		},
		{
			name:   "zero upload.max_file_size",
			mutate: func(cfg *Config) { cfg.Upload.MaxFileSize = 0 },
			wantErr: "upload.max_file_size must be positive",
		},
		{
			name:   "negative upload.max_file_size",
			mutate: func(cfg *Config) { cfg.Upload.MaxFileSize = -1 },
			wantErr: "upload.max_file_size must be positive",
		},
		{
			name:   "zero upload.max_chunk_size",
			mutate: func(cfg *Config) { cfg.Upload.MaxChunkSize = 0 },
			wantErr: "upload.max_chunk_size must be positive",
		},
		{
			name:   "negative upload.max_chunk_size",
			mutate: func(cfg *Config) { cfg.Upload.MaxChunkSize = -1 },
			wantErr: "upload.max_chunk_size must be positive",
		},
		{
			name:   "zero upload.max_total_size",
			mutate: func(cfg *Config) { cfg.Upload.MaxTotalSize = 0 },
			wantErr: "upload.max_total_size must be positive",
		},
		{
			name:   "negative upload.max_total_size",
			mutate: func(cfg *Config) { cfg.Upload.MaxTotalSize = -1 },
			wantErr: "upload.max_total_size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(t)
			tt.mutate(cfg)
			err := Validate(cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidate_TLSCertFileNotFound(t *testing.T) {
	cfg := validConfig(t)
	cfg.Server.TLSCertFile = "/nonexistent/path/cert.pem"
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server.tls_cert_file not found")
}

func TestValidate_TLSKeyFileNotFound(t *testing.T) {
	cfg := validConfig(t)
	cfg.Server.TLSKeyFile = "/nonexistent/path/key.pem"
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server.tls_key_file not found")
}

func TestValidate_BlobPathNotRequiredForNonLocalProvider(t *testing.T) {
	cfg := validConfig(t)
	cfg.Blob.Provider = "s3"
	cfg.Blob.Path = ""
	require.NoError(t, Validate(cfg))
}

func TestValidate_MultipleErrorsReported(t *testing.T) {
	cfg := &Config{}
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server.address is required")
	require.Contains(t, err.Error(), "server.grpc_address is required")
	require.Contains(t, err.Error(), "database.dsn is required")
}
