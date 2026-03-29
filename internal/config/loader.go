package config

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

// Load загружает конфигурацию из JSON файла и применяет env overrides (GK_*)
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyEnv(&cfg)

	return &cfg, nil
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
