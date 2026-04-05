package config

import "time"

// Config объединяет конфигурацию приложения.
type Config struct {
	Server   ServerConfig       `json:"server"`
	Database DatabaseConfig     `json:"database"`
	Auth     AuthConfig         `json:"auth"`
	Crypto   CryptoConfig       `json:"crypto"`
	Blob     BlobStorageConfig  `json:"blob"`
	Upload   UploadLimitsConfig `json:"upload"`
}

// ServerConfig описывает настройки HTTP- и gRPC-серверов.
type ServerConfig struct {
	Address     string `json:"address"`
	GRPCAddress string `json:"grpc_address"`
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`
}

// DatabaseConfig описывает подключение к БД.
type DatabaseConfig struct {
	DSN string `json:"dsn"`
}

// AuthConfig описывает параметры аутентификации.
type AuthConfig struct {
	JWTSecret     string        `json:"jwt_secret"`
	TokenDuration time.Duration `json:"token_duration"`
}

// CryptoConfig описывает параметры шифрования.
type CryptoConfig struct {
	MasterKey string `json:"master_key"`
}

// BlobStorageConfig описывает настройки blob-хранилища.
type BlobStorageConfig struct {
	Provider  string `json:"provider"`
	Path      string `json:"path"`
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

// UploadLimitsConfig задаёт ограничения на бинарные payload.
type UploadLimitsConfig struct {
	MaxFileSize  int64 `json:"max_file_size"`
	MaxChunkSize int64 `json:"max_chunk_size"`
	MaxTotalSize int64 `json:"max_total_size"`
}
