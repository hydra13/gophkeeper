package main

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/app"
	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/repositories"
	"github.com/hydra13/gophkeeper/internal/storage"
)

func TestOpenDB_BadDriver(t *testing.T) {
	t.Parallel()

	_, err := sql.Open("nonexistent_driver", "whatever")
	require.Error(t, err)
}

type stubBlobStorage struct{}

func (stubBlobStorage) Save(string, []byte) error   { return nil }
func (stubBlobStorage) Read(string) ([]byte, error) { return nil, nil }
func (stubBlobStorage) Delete(string) error         { return nil }
func (stubBlobStorage) Exists(string) (bool, error) { return false, nil }

func TestWireDeps_Failures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		cfg             *config.Config
		useDefaultWire  bool
		setup           func(*config.Config) (wireDepsFactories, func(*testing.T))
		factories       wireDepsFactories
		wantErrContains []string
		wantCleanupNil  bool
		assert          func(*testing.T)
	}{
		{
			name:           "empty dsn",
			cfg:            &config.Config{Database: config.DatabaseConfig{DSN: ""}},
			useDefaultWire: true,
			wantErrContains: []string{
				"database DSN is required",
			},
			wantCleanupNil: true,
		},
		{
			name: "forwards blob provider config",
			cfg: &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
				Blob: config.BlobStorageConfig{
					Provider: "s3",
					Path:     "/tmp/blob",
					Endpoint: "http://localhost:9000",
					Bucket:   "gophkeeper-dev",
					Region:   "us-east-1",
				},
				Auth: config.AuthConfig{
					TokenDuration: time.Hour,
				},
			},
			setup: func(cfg *config.Config) (wireDepsFactories, func(*testing.T)) {
				var gotBlobCfg config.BlobStorageConfig
				return wireDepsFactories{
						applyMigrations: func(_ *sql.DB) error { return nil },
						openDB:          openTestDBWithoutPing,
						newBlobStorage: func(cfg config.BlobStorageConfig) (repositories.BlobStorage, error) {
							gotBlobCfg = cfg
							return stubBlobStorage{}, nil
						},
					}, func(t *testing.T) {
						t.Helper()
						require.Equal(t, cfg.Blob, gotBlobCfg)
					}
			},
			wantErrContains: []string{
				"jwt secret is required",
			},
		},
		{
			name: "invalid dsn",
			cfg: &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://invalid:invalid@localhost:9999/nonexistent?sslmode=disable"},
			},
			useDefaultWire: true,
			wantErrContains: []string{
				"failed to connect to database",
			},
		},
		{
			name: "migration failure",
			cfg: &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
			},
			factories: wireDepsFactories{
				applyMigrations: func(_ *sql.DB) error { return errors.New("migration boom") },
				openDB:          openTestDBWithoutPing,
				newBlobStorage:  storage.NewBlobStorage,
			},
			wantErrContains: []string{
				"failed to apply database migrations",
			},
		},
		{
			name: "blob storage failure",
			cfg: &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
				Blob:     config.BlobStorageConfig{Provider: "s3"},
			},
			factories: wireDepsFactories{
				applyMigrations: func(_ *sql.DB) error { return nil },
				openDB:          openTestDBWithoutPing,
				newBlobStorage: func(config.BlobStorageConfig) (repositories.BlobStorage, error) {
					return nil, errors.New("blob boom")
				},
			},
			wantErrContains: []string{
				"failed to initialize blob storage",
				"blob boom",
			},
		},
		{
			name: "open db failure",
			cfg: &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
			},
			factories: wireDepsFactories{
				openDB: func(string) (*sql.DB, error) {
					return nil, errors.New("db boom")
				},
				applyMigrations: func(_ *sql.DB) error { return nil },
				newBlobStorage: func(config.BlobStorageConfig) (repositories.BlobStorage, error) {
					return stubBlobStorage{}, nil
				},
			},
			wantErrContains: []string{
				"failed to connect to database",
			},
			wantCleanupNil: true,
		},
		{
			name: "key manager failure",
			cfg: &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
				Blob:     config.BlobStorageConfig{Provider: "local", Path: "/tmp/blob"},
				Auth: config.AuthConfig{
					JWTSecret:     "secret",
					TokenDuration: time.Hour,
				},
				Crypto: config.CryptoConfig{
					MasterKey: "bad-key",
				},
			},
			factories: wireDepsFactories{
				applyMigrations: func(_ *sql.DB) error { return nil },
				openDB:          openTestDBWithoutPing,
				newBlobStorage: func(config.BlobStorageConfig) (repositories.BlobStorage, error) {
					return stubBlobStorage{}, nil
				},
			},
			wantErrContains: []string{
				"master key",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				deps    app.AppDeps
				cleanup func()
				err     error
				assert  = tt.assert
			)

			if tt.useDefaultWire {
				deps, cleanup, err = wireDeps(tt.cfg, zerolog.Nop())
			} else {
				factories := tt.factories
				if tt.setup != nil {
					factories, assert = tt.setup(tt.cfg)
				}
				deps, cleanup, err = wireDepsWithFactories(tt.cfg, zerolog.Nop(), factories)
			}

			t.Cleanup(func() {
				if cleanup != nil {
					cleanup()
				}
			})

			require.Error(t, err)
			for _, want := range tt.wantErrContains {
				require.Contains(t, err.Error(), want)
			}
			if tt.wantCleanupNil {
				require.Nil(t, cleanup)
			}
			require.Equal(t, app.AppDeps{}, deps)
			if assert != nil {
				assert(t)
			}
		})
	}
}

func openTestDBWithoutPing(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
