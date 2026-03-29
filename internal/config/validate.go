package config

import (
	"errors"
	"fmt"
	"os"
)

// Validate проверяет корректность загруженной конфигурации
func Validate(cfg *Config) error {
	var errs []error

	if cfg.Server.Address == "" {
		errs = append(errs, errors.New("server.address is required"))
	}
	if cfg.Server.GRPCAddress == "" {
		errs = append(errs, errors.New("server.grpc_address is required"))
	}
	if cfg.Server.TLSCertFile == "" {
		errs = append(errs, errors.New("server.tls_cert_file is required"))
	}
	if cfg.Server.TLSKeyFile == "" {
		errs = append(errs, errors.New("server.tls_key_file is required"))
	}
	if cfg.Server.TLSCertFile != "" {
		if _, err := os.Stat(cfg.Server.TLSCertFile); err != nil {
			errs = append(errs, fmt.Errorf("server.tls_cert_file not found: %w", err))
		}
	}
	if cfg.Server.TLSKeyFile != "" {
		if _, err := os.Stat(cfg.Server.TLSKeyFile); err != nil {
			errs = append(errs, fmt.Errorf("server.tls_key_file not found: %w", err))
		}
	}
	if cfg.Database.DSN == "" {
		errs = append(errs, errors.New("database.dsn is required"))
	}
	if cfg.Auth.JWTSecret == "" {
		errs = append(errs, errors.New("auth.jwt_secret is required"))
	}
	if cfg.Auth.TokenDuration <= 0 {
		errs = append(errs, errors.New("auth.token_duration must be positive"))
	}
	if cfg.Crypto.MasterKey == "" {
		errs = append(errs, errors.New("crypto.master_key is required"))
	}
	if cfg.Blob.Provider == "" {
		errs = append(errs, errors.New("blob.provider is required"))
	}
	if cfg.Blob.Provider == "local" && cfg.Blob.Path == "" {
		errs = append(errs, errors.New("blob.path is required for local provider"))
	}
	if cfg.Upload.MaxFileSize <= 0 {
		errs = append(errs, errors.New("upload.max_file_size must be positive"))
	}
	if cfg.Upload.MaxChunkSize <= 0 {
		errs = append(errs, errors.New("upload.max_chunk_size must be positive"))
	}
	if cfg.Upload.MaxTotalSize <= 0 {
		errs = append(errs, errors.New("upload.max_total_size must be positive"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %w", errors.Join(errs...))
	}
	return nil
}
