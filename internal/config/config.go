// Package config описывает конфигурацию сервиса и её загрузку.
package config

import "time"

// Config описывает полную конфигурацию сервиса.
type Config struct {
	Server   ServerConfig       `json:"server"`
	Database DatabaseConfig     `json:"database"`
	Auth     AuthConfig         `json:"auth"`
	Crypto   CryptoConfig       `json:"crypto"`
	Blob     BlobStorageConfig  `json:"blob"`
	Upload   UploadLimitsConfig `json:"upload"`
}

// ServerConfig содержит параметры HTTP и gRPC серверов.
type ServerConfig struct {
	Address     string `json:"address"`
	GRPCAddress string `json:"grpc_address"`
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`
}

// DatabaseConfig содержит параметры подключения к PostgreSQL.
type DatabaseConfig struct {
	DSN string `json:"dsn"`
}

// AuthConfig содержит параметры JWT-аутентификации.
type AuthConfig struct {
	JWTSecret     string        `json:"jwt_secret"`
	TokenDuration time.Duration `json:"token_duration"`
}

// CryptoConfig содержит параметры криптографии и мастер-ключа.
type CryptoConfig struct {
	MasterKey string `json:"master_key"`
}

// BlobStorageConfig содержит параметры blob-хранилища бинарных данных.
type BlobStorageConfig struct {
	Provider  string `json:"provider"`
	Path      string `json:"path"`
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

// UploadLimitsConfig содержит ограничения на загрузку бинарных файлов.
type UploadLimitsConfig struct {
	MaxFileSize  int64 `json:"max_file_size"`
	MaxChunkSize int64 `json:"max_chunk_size"`
	MaxTotalSize int64 `json:"max_total_size"`
}
