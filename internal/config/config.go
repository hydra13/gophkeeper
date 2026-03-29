package config

import "time"

type Config struct {
	Server   ServerConfig       `json:"server"`
	Database DatabaseConfig     `json:"database"`
	Auth     AuthConfig         `json:"auth"`
	Crypto   CryptoConfig       `json:"crypto"`
	Blob     BlobStorageConfig  `json:"blob"`
	Upload   UploadLimitsConfig `json:"upload"`
}

type ServerConfig struct {
	Address     string `json:"address"`
	GRPCAddress string `json:"grpc_address"`
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`
}

type DatabaseConfig struct {
	DSN string `json:"dsn"`
}

type AuthConfig struct {
	JWTSecret     string        `json:"jwt_secret"`
	TokenDuration time.Duration `json:"token_duration"`
}

type CryptoConfig struct {
	MasterKey string `json:"master_key"`
}

type BlobStorageConfig struct {
	Provider  string `json:"provider"`
	Path      string `json:"path"`
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

type UploadLimitsConfig struct {
	MaxFileSize  int64 `json:"max_file_size"`
	MaxChunkSize int64 `json:"max_chunk_size"`
	MaxTotalSize int64 `json:"max_total_size"`
}
