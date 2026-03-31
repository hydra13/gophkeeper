package config

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"strconv"
	"time"
)

const defaultConfigPath = "configs/config.dev.json"

// Load загружает конфигурацию из файла, переменных окружения и флагов командной строки.
func Load() (*Config, error) {
	cfg := &Config{}

	configPath := resolveConfigPath()
	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			return nil, err
		}
	}

	applyEnv(cfg)
	applyFlags(cfg)

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

func resolveConfigPath() string {
	if arg := argValue("-config"); arg != "" {
		return arg
	}
	if arg := argValue("-c"); arg != "" {
		return arg
	}
	if v := os.Getenv("GK_CONFIG"); v != "" {
		return v
	}
	if _, err := os.Stat(defaultConfigPath); err == nil {
		return defaultConfigPath
	}
	return ""
}

func argValue(flagName string) string {
	for i, arg := range os.Args {
		if arg == flagName && len(os.Args) > i+1 {
			return os.Args[i+1]
		}
	}
	return ""
}

func applyEnv(cfg *Config) {
	envStr(&cfg.Server.Address, "GK_SERVER_ADDRESS")
	envStr(&cfg.Server.GRPCAddress, "GK_GRPC_ADDRESS")
	envStr(&cfg.Server.TLSCertFile, "GK_TLS_CERT_FILE")
	envStr(&cfg.Server.TLSKeyFile, "GK_TLS_KEY_FILE")
	envStr(&cfg.Database.DSN, "GK_DATABASE_DSN")
	envStr(&cfg.Auth.JWTSecret, "GK_JWT_SECRET")
	envDuration(&cfg.Auth.TokenDuration, "GK_TOKEN_DURATION")
	envStr(&cfg.Crypto.MasterKey, "GK_MASTER_KEY")
	envStr(&cfg.Blob.Provider, "GK_BLOB_PROVIDER")
	envStr(&cfg.Blob.Path, "GK_BLOB_PATH")
	envStr(&cfg.Blob.Endpoint, "GK_BLOB_ENDPOINT")
	envStr(&cfg.Blob.Bucket, "GK_BLOB_BUCKET")
	envStr(&cfg.Blob.AccessKey, "GK_BLOB_ACCESS_KEY")
	envStr(&cfg.Blob.SecretKey, "GK_BLOB_SECRET_KEY")
	envStr(&cfg.Blob.Region, "GK_BLOB_REGION")
	envInt64(&cfg.Upload.MaxFileSize, "GK_UPLOAD_MAX_FILE_SIZE")
	envInt64(&cfg.Upload.MaxChunkSize, "GK_UPLOAD_MAX_CHUNK_SIZE")
	envInt64(&cfg.Upload.MaxTotalSize, "GK_UPLOAD_MAX_TOTAL_SIZE")
}

func envStr(target *string, key string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}

func envDuration(target *time.Duration, key string) {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*target = d
		}
	}
}

func envBool(target *bool, key string) {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			*target = b
		}
	}
}

func envInt64(target *int64, key string) {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			*target = parsed
		}
	}
}

func applyFlags(cfg *Config) {
	fs := flag.NewFlagSet("gophkeeper-server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var configPath string
	fs.StringVar(&configPath, "config", "", "path to config file")
	fs.StringVar(&configPath, "c", "", "path to config file (shorthand)")

	fs.StringVar(&cfg.Server.Address, "http-address", cfg.Server.Address, "HTTP server address")
	fs.StringVar(&cfg.Server.GRPCAddress, "grpc-address", cfg.Server.GRPCAddress, "gRPC server address")
	fs.StringVar(&cfg.Server.TLSCertFile, "tls-cert", cfg.Server.TLSCertFile, "TLS certificate path")
	fs.StringVar(&cfg.Server.TLSKeyFile, "tls-key", cfg.Server.TLSKeyFile, "TLS key path")

	fs.StringVar(&cfg.Database.DSN, "db-dsn", cfg.Database.DSN, "PostgreSQL DSN")

	fs.StringVar(&cfg.Auth.JWTSecret, "jwt-secret", cfg.Auth.JWTSecret, "JWT secret")
	fs.DurationVar(&cfg.Auth.TokenDuration, "token-duration", cfg.Auth.TokenDuration, "JWT token duration")

	fs.StringVar(&cfg.Crypto.MasterKey, "master-key", cfg.Crypto.MasterKey, "master key")

	fs.StringVar(&cfg.Blob.Provider, "blob-provider", cfg.Blob.Provider, "blob storage provider")
	fs.StringVar(&cfg.Blob.Path, "blob-path", cfg.Blob.Path, "blob storage base path")
	fs.StringVar(&cfg.Blob.Endpoint, "blob-endpoint", cfg.Blob.Endpoint, "blob storage endpoint")
	fs.StringVar(&cfg.Blob.Bucket, "blob-bucket", cfg.Blob.Bucket, "blob storage bucket")
	fs.StringVar(&cfg.Blob.AccessKey, "blob-access-key", cfg.Blob.AccessKey, "blob storage access key")
	fs.StringVar(&cfg.Blob.SecretKey, "blob-secret-key", cfg.Blob.SecretKey, "blob storage secret key")
	fs.StringVar(&cfg.Blob.Region, "blob-region", cfg.Blob.Region, "blob storage region")

	fs.Int64Var(&cfg.Upload.MaxFileSize, "upload-max-file-size", cfg.Upload.MaxFileSize, "max upload file size in bytes")
	fs.Int64Var(&cfg.Upload.MaxChunkSize, "upload-max-chunk-size", cfg.Upload.MaxChunkSize, "max upload chunk size in bytes")
	fs.Int64Var(&cfg.Upload.MaxTotalSize, "upload-max-total-size", cfg.Upload.MaxTotalSize, "max total upload size in bytes")

	_ = fs.Parse(os.Args[1:])
}
