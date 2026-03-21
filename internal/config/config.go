package config

import "time"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Crypto   CryptoConfig
}

type ServerConfig struct {
	Address     string
	GRPCAddress string
	EnableHTTPS bool
	TLSCertFile string
	TLSKeyFile  string
}

type DatabaseConfig struct {
	DSN string
}

type AuthConfig struct {
	JWTSecret     string
	TokenDuration time.Duration
}

type CryptoConfig struct {
	MasterKey string
}
