package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- loadFromFile ---

func TestLoadFromFile_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	raw := map[string]interface{}{
		"server": map[string]interface{}{
			"address":       ":8080",
			"grpc_address":  ":9090",
			"tls_cert_file": "/cert.pem",
			"tls_key_file":  "/key.pem",
		},
		"database": map[string]interface{}{
			"dsn": "postgres://localhost/db",
		},
		"auth": map[string]interface{}{
			"jwt_secret":     "secret",
			"token_duration": int64(time.Hour),
		},
		"crypto": map[string]interface{}{
			"master_key": "key",
		},
		"blob": map[string]interface{}{
			"provider": "local",
			"path":     filepath.Join(dir, "blob"),
		},
		"upload": map[string]interface{}{
			"max_file_size":  1024,
			"max_chunk_size": 512,
			"max_total_size": 2048,
		},
	}

	data, err := json.Marshal(raw)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgPath, data, 0o644))

	var cfg Config
	require.NoError(t, loadFromFile(&cfg, cfgPath))

	require.Equal(t, ":8080", cfg.Server.Address)
	require.Equal(t, ":9090", cfg.Server.GRPCAddress)
	require.Equal(t, "postgres://localhost/db", cfg.Database.DSN)
	require.Equal(t, time.Hour, cfg.Auth.TokenDuration)
	require.Equal(t, int64(1024), cfg.Upload.MaxFileSize)
}

func TestLoadFromFile_NonExistentFile(t *testing.T) {
	var cfg Config
	err := loadFromFile(&cfg, "/nonexistent/config.json")
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(cfgPath, []byte("{invalid json}"), 0o644))

	var cfg Config
	err := loadFromFile(&cfg, cfgPath)
	require.Error(t, err)
}

// --- envStr ---

func TestEnvStr(t *testing.T) {
	t.Run("sets value when env is present", func(t *testing.T) {
		t.Setenv("TEST_ENV_STR", "hello")
		var s string
		envStr(&s, "TEST_ENV_STR")
		require.Equal(t, "hello", s)
	})

	t.Run("does not overwrite when env is empty", func(t *testing.T) {
		t.Setenv("TEST_ENV_STR_EMPTY", "")
		s := "original"
		envStr(&s, "TEST_ENV_STR_EMPTY")
		require.Equal(t, "original", s)
	})

	t.Run("does not overwrite when env is unset", func(t *testing.T) {
		s := "original"
		envStr(&s, "TEST_ENV_STR_UNSET")
		require.Equal(t, "original", s)
	})
}

// --- envDuration ---

func TestEnvDuration(t *testing.T) {
	t.Run("parses valid duration", func(t *testing.T) {
		t.Setenv("TEST_ENV_DUR", "30m")
		var d time.Duration
		envDuration(&d, "TEST_ENV_DUR")
		require.Equal(t, 30*time.Minute, d)
	})

	t.Run("ignores invalid duration", func(t *testing.T) {
		t.Setenv("TEST_ENV_DUR_BAD", "not-a-duration")
		d := time.Hour
		envDuration(&d, "TEST_ENV_DUR_BAD")
		require.Equal(t, time.Hour, d) // unchanged
	})

	t.Run("ignores empty env", func(t *testing.T) {
		t.Setenv("TEST_ENV_DUR_EMPTY", "")
		d := time.Hour
		envDuration(&d, "TEST_ENV_DUR_EMPTY")
		require.Equal(t, time.Hour, d)
	})
}

// --- envBool ---

func TestEnvBool(t *testing.T) {
	t.Run("parses true", func(t *testing.T) {
		t.Setenv("TEST_ENV_BOOL", "true")
		var b bool
		envBool(&b, "TEST_ENV_BOOL")
		require.True(t, b)
	})

	t.Run("parses false", func(t *testing.T) {
		t.Setenv("TEST_ENV_BOOL_F", "false")
		b := true
		envBool(&b, "TEST_ENV_BOOL_F")
		require.False(t, b)
	})

	t.Run("ignores invalid value", func(t *testing.T) {
		t.Setenv("TEST_ENV_BOOL_BAD", "notbool")
		b := true
		envBool(&b, "TEST_ENV_BOOL_BAD")
		require.True(t, b) // unchanged
	})
}

// --- envInt64 ---

func TestEnvInt64(t *testing.T) {
	t.Run("parses valid integer", func(t *testing.T) {
		t.Setenv("TEST_ENV_INT", "42")
		var n int64
		envInt64(&n, "TEST_ENV_INT")
		require.Equal(t, int64(42), n)
	})

	t.Run("ignores invalid integer", func(t *testing.T) {
		t.Setenv("TEST_ENV_INT_BAD", "not-a-number")
		var n int64 = 99
		envInt64(&n, "TEST_ENV_INT_BAD")
		require.Equal(t, int64(99), n) // unchanged
	})

	t.Run("ignores empty env", func(t *testing.T) {
		t.Setenv("TEST_ENV_INT_EMPTY", "")
		var n int64 = 99
		envInt64(&n, "TEST_ENV_INT_EMPTY")
		require.Equal(t, int64(99), n)
	})
}

// --- applyEnv ---

func TestApplyEnv(t *testing.T) {
	t.Setenv("GK_SERVER_ADDRESS", ":9999")
	t.Setenv("GK_DATABASE_DSN", "postgres://env:pass@localhost/db")
	t.Setenv("GK_TOKEN_DURATION", "2h")
	t.Setenv("GK_UPLOAD_MAX_FILE_SIZE", "2048")

	cfg := &Config{}
	applyEnv(cfg)

	require.Equal(t, ":9999", cfg.Server.Address)
	require.Equal(t, "postgres://env:pass@localhost/db", cfg.Database.DSN)
	require.Equal(t, 2*time.Hour, cfg.Auth.TokenDuration)
	require.Equal(t, int64(2048), cfg.Upload.MaxFileSize)
}

// --- resolveConfigPath ---

func TestResolveConfigPath_FromEnv(t *testing.T) {
	t.Setenv("GK_CONFIG", "/custom/path.json")

	// Clear os.Args so argValue does not pick up stray flags.
	origArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = origArgs }()

	path := resolveConfigPath()
	require.Equal(t, "/custom/path.json", path)
}

func TestResolveConfigPath_EmptyWhenNothingSet(t *testing.T) {
	t.Setenv("GK_CONFIG", "")

	origArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = origArgs }()

	// Change working dir to temp so defaultConfigPath does not accidentally exist.
	dir := t.TempDir()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origWd)

	path := resolveConfigPath()
	require.Equal(t, "", path)
}

// --- argValue ---

func TestArgValue(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	t.Run("finds flag value", func(t *testing.T) {
		os.Args = []string{"prog", "-config", "/path/to/config.json"}
		require.Equal(t, "/path/to/config.json", argValue("-config"))
	})

	t.Run("returns empty when flag not present", func(t *testing.T) {
		os.Args = []string{"prog", "-other", "value"}
		require.Equal(t, "", argValue("-config"))
	})

	t.Run("returns empty when flag is last arg without value", func(t *testing.T) {
		os.Args = []string{"prog", "-config"}
		require.Equal(t, "", argValue("-config"))
	})
}

// --- Load (integration) ---

// writeValidConfig writes a minimal valid config JSON that will pass Validate.
// It creates real TLS cert/key files so Validate's os.Stat checks pass.
func writeValidConfig(t *testing.T, dir string) string {
	t.Helper()

	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0o644))
	require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0o644))

	raw := map[string]interface{}{
		"server": map[string]interface{}{
			"address":       ":8080",
			"grpc_address":  ":9090",
			"tls_cert_file": certFile,
			"tls_key_file":  keyFile,
		},
		"database": map[string]interface{}{
			"dsn": "postgres://localhost/db",
		},
		"auth": map[string]interface{}{
			"jwt_secret":     "secret",
			"token_duration": int64(time.Hour),
		},
		"crypto": map[string]interface{}{
			"master_key": "key",
		},
		"blob": map[string]interface{}{
			"provider": "local",
			"path":     filepath.Join(dir, "blob"),
		},
		"upload": map[string]interface{}{
			"max_file_size":  1024,
			"max_chunk_size": 512,
			"max_total_size": 2048,
		},
	}

	data, err := json.Marshal(raw)
	require.NoError(t, err)

	cfgPath := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(cfgPath, data, 0o644))

	return cfgPath
}

func TestLoad_ValidConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeValidConfig(t, dir)

	origArgs := os.Args
	os.Args = []string{"test", "-config", cfgPath}
	defer func() { os.Args = origArgs }()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, ":8080", cfg.Server.Address)
	require.Equal(t, ":9090", cfg.Server.GRPCAddress)
	require.Equal(t, "postgres://localhost/db", cfg.Database.DSN)
	require.Equal(t, "secret", cfg.Auth.JWTSecret)
	require.Equal(t, time.Hour, cfg.Auth.TokenDuration)
	require.Equal(t, "key", cfg.Crypto.MasterKey)
	require.Equal(t, "local", cfg.Blob.Provider)
	require.Equal(t, int64(1024), cfg.Upload.MaxFileSize)
	require.Equal(t, int64(512), cfg.Upload.MaxChunkSize)
	require.Equal(t, int64(2048), cfg.Upload.MaxTotalSize)
}

func TestLoad_ValidConfigShorthandFlag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeValidConfig(t, dir)

	origArgs := os.Args
	os.Args = []string{"test", "-c", cfgPath}
	defer func() { os.Args = origArgs }()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, ":8080", cfg.Server.Address)
}

func TestLoad_ConfigFromEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeValidConfig(t, dir)

	t.Setenv("GK_CONFIG", cfgPath)

	origArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = origArgs }()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, ":8080", cfg.Server.Address)
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeValidConfig(t, dir)

	t.Setenv("GK_CONFIG", cfgPath)
	t.Setenv("GK_SERVER_ADDRESS", ":9999")
	t.Setenv("GK_DATABASE_DSN", "postgres://overridden/db")

	origArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = origArgs }()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Env overrides file values.
	require.Equal(t, ":9999", cfg.Server.Address)
	require.Equal(t, "postgres://overridden/db", cfg.Database.DSN)

	// Non-overridden values stay from file.
	require.Equal(t, ":9090", cfg.Server.GRPCAddress)
}

func TestLoad_FlagOverridesEnvAndFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeValidConfig(t, dir)

	t.Setenv("GK_CONFIG", cfgPath)
	// Env sets address to :7777
	t.Setenv("GK_SERVER_ADDRESS", ":7777")

	origArgs := os.Args
	os.Args = []string{"test", "-http-address", ":5555"}
	defer func() { os.Args = origArgs }()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Flag overrides env.
	require.Equal(t, ":5555", cfg.Server.Address)
}

func TestLoad_ValidationFailsOnEmptyConfig(t *testing.T) {
	// No config file, no env, no flags -- empty config should fail validation.
	t.Setenv("GK_CONFIG", "")

	origArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = origArgs }()

	// Change to temp dir so defaultConfigPath is not found.
	dir := t.TempDir()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(origWd)

	_, err = Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "config validation failed")
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(cfgPath, []byte("{bad}"), 0o644))

	origArgs := os.Args
	os.Args = []string{"test", "-config", cfgPath}
	defer func() { os.Args = origArgs }()

	_, err := Load()
	require.Error(t, err)
}

func TestLoad_NonExistentConfigFile(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"test", "-config", "/nonexistent/config.json"}
	defer func() { os.Args = origArgs }()

	_, err := Load()
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLoad_TLSFilesNotFound(t *testing.T) {
	dir := t.TempDir()

	raw := map[string]interface{}{
		"server": map[string]interface{}{
			"address":       ":8080",
			"grpc_address":  ":9090",
			"tls_cert_file": "/nonexistent/cert.pem",
			"tls_key_file":  "/nonexistent/key.pem",
		},
		"database": map[string]interface{}{
			"dsn": "postgres://localhost/db",
		},
		"auth": map[string]interface{}{
			"jwt_secret":     "secret",
			"token_duration": int64(time.Hour),
		},
		"crypto": map[string]interface{}{
			"master_key": "key",
		},
		"blob": map[string]interface{}{
			"provider": "local",
			"path":     filepath.Join(dir, "blob"),
		},
		"upload": map[string]interface{}{
			"max_file_size":  1024,
			"max_chunk_size": 512,
			"max_total_size": 2048,
		},
	}

	data, err := json.Marshal(raw)
	require.NoError(t, err)
	cfgPath := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(cfgPath, data, 0o644))

	origArgs := os.Args
	os.Args = []string{"test", "-config", cfgPath}
	defer func() { os.Args = origArgs }()

	_, err = Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "tls_cert_file not found")
}

// --- applyFlags ---

func TestApplyFlags_AllFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{
		"test",
		"-http-address", ":1111",
		"-grpc-address", ":2222",
		"-tls-cert", "/cert.pem",
		"-tls-key", "/key.pem",
		"-db-dsn", "postgres://flag/db",
		"-jwt-secret", "flag-secret",
		"-token-duration", "45m",
		"-master-key", "flag-key",
		"-blob-provider", "local",
		"-blob-path", "/data/blob",
		"-blob-endpoint", "http://minio:9000",
		"-blob-bucket", "mybucket",
		"-blob-access-key", "ak",
		"-blob-secret-key", "sk",
		"-blob-region", "us-east-1",
		"-upload-max-file-size", "999",
		"-upload-max-chunk-size", "888",
		"-upload-max-total-size", "777",
	}

	cfg := &Config{}
	applyFlags(cfg)

	assert.Equal(t, ":1111", cfg.Server.Address)
	assert.Equal(t, ":2222", cfg.Server.GRPCAddress)
	assert.Equal(t, "/cert.pem", cfg.Server.TLSCertFile)
	assert.Equal(t, "/key.pem", cfg.Server.TLSKeyFile)
	assert.Equal(t, "postgres://flag/db", cfg.Database.DSN)
	assert.Equal(t, "flag-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, 45*time.Minute, cfg.Auth.TokenDuration)
	assert.Equal(t, "flag-key", cfg.Crypto.MasterKey)
	assert.Equal(t, "local", cfg.Blob.Provider)
	assert.Equal(t, "/data/blob", cfg.Blob.Path)
	assert.Equal(t, "http://minio:9000", cfg.Blob.Endpoint)
	assert.Equal(t, "mybucket", cfg.Blob.Bucket)
	assert.Equal(t, "ak", cfg.Blob.AccessKey)
	assert.Equal(t, "sk", cfg.Blob.SecretKey)
	assert.Equal(t, "us-east-1", cfg.Blob.Region)
	assert.Equal(t, int64(999), cfg.Upload.MaxFileSize)
	assert.Equal(t, int64(888), cfg.Upload.MaxChunkSize)
	assert.Equal(t, int64(777), cfg.Upload.MaxTotalSize)
}

func TestApplyFlags_OverridesExistingValues(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{
		"test",
		"-http-address", ":overridden",
	}

	cfg := &Config{
		Server: ServerConfig{
			Address:     ":original",
			GRPCAddress: ":9090",
		},
	}
	applyFlags(cfg)

	assert.Equal(t, ":overridden", cfg.Server.Address)
	// Non-flagged field stays unchanged.
	assert.Equal(t, ":9090", cfg.Server.GRPCAddress)
}

func TestApplyFlags_EmptyArgs(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"test"}

	cfg := &Config{
		Server: ServerConfig{Address: ":8080"},
	}
	applyFlags(cfg)

	// No flags means nothing changes.
	assert.Equal(t, ":8080", cfg.Server.Address)
}

func TestApplyFlags_IgnoresUnknownFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{
		"test",
		"-http-address", ":6060",
		"-unknown-flag", "value",
	}

	cfg := &Config{}
	applyFlags(cfg)

	// Known flags before the unknown one are parsed; unknown ones are silently
	// ignored because ContinueOnError is set.  Note: flags after an unknown
	// flag may not be parsed because flag.Parse stops at the first error.
	assert.Equal(t, ":6060", cfg.Server.Address)
}
