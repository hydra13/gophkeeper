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
)

func TestOpenDB_BadDriver(t *testing.T) {
	_, err := sql.Open("nonexistent_driver", "whatever")
	require.Error(t, err)
}

func TestWireDeps_EmptyDSN_Fails(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{DSN: ""},
	}
	log := zerolog.Nop()

	deps, cleanup, err := wireDeps(cfg, log)
	require.Error(t, err)
	require.Contains(t, err.Error(), "database DSN is required")
	require.Nil(t, cleanup)
	require.Equal(t, app.AppDeps{}, deps)
}

type stubBlobStorage struct{}

func (stubBlobStorage) Save(string, []byte) error   { return nil }
func (stubBlobStorage) Read(string) ([]byte, error) { return nil, nil }
func (stubBlobStorage) Delete(string) error         { return nil }
func (stubBlobStorage) Exists(string) (bool, error) { return false, nil }

func TestWireDeps_ForwardsBlobProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "local", provider: "local"},
		{name: "s3", provider: "s3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origApply := applyMigrations
			origOpenDB := openDB
			origBlob := newBlobStorage
			t.Cleanup(func() {
				applyMigrations = origApply
				openDB = origOpenDB
				newBlobStorage = origBlob
			})

			applyMigrations = func(_ *sql.DB) error { return nil }
			openDB = func(dsn string) (*sql.DB, error) {
				db, err := sql.Open("pgx", dsn)
				if err != nil {
					return nil, err
				}
				return db, nil
			}

			var gotBlobCfg config.BlobStorageConfig
			newBlobStorage = func(cfg config.BlobStorageConfig) (repositories.BlobStorage, error) {
				gotBlobCfg = cfg
				return stubBlobStorage{}, nil
			}

			cfg := &config.Config{
				Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
				Blob: config.BlobStorageConfig{
					Provider: tt.provider,
					Path:     "/tmp/blob",
					Endpoint: "http://localhost:9000",
					Bucket:   "gophkeeper-dev",
					Region:   "us-east-1",
				},
				Auth: config.AuthConfig{
					JWTSecret:     "",
					TokenDuration: time.Hour,
				},
			}
			log := zerolog.Nop()

			deps, cleanup, err := wireDeps(cfg, log)
			require.Error(t, err)
			require.Contains(t, err.Error(), "jwt secret is required")
			require.Equal(t, cfg.Blob, gotBlobCfg)
			if cleanup != nil {
				cleanup()
			}
			require.Equal(t, app.AppDeps{}, deps)
		})
	}
}

func TestWireDeps_InvalidDSN_Fails(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{DSN: "postgres://invalid:invalid@localhost:9999/nonexistent?sslmode=disable"},
	}
	log := zerolog.Nop()

	deps, cleanup, err := wireDeps(cfg, log)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to connect to database")
	if cleanup != nil {
		cleanup()
	}
	require.Equal(t, app.AppDeps{}, deps)
}

func TestWireDeps_MigrationFailure_Fails(t *testing.T) {
	origApply := applyMigrations
	origOpenDB := openDB
	origBlob := newBlobStorage
	t.Cleanup(func() {
		applyMigrations = origApply
		openDB = origOpenDB
		newBlobStorage = origBlob
	})

	applyMigrations = func(_ *sql.DB) error {
		return errors.New("migration boom")
	}
	openDB = func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		// Skip Ping — we want openDB to succeed so we reach applyMigrations.
		return db, nil
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
	}
	log := zerolog.Nop()

	deps, cleanup, err := wireDeps(cfg, log)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply database migrations")
	if cleanup != nil {
		cleanup()
	}
	require.Equal(t, app.AppDeps{}, deps)
}

func TestWireDeps_BlobStorageFailure_Fails(t *testing.T) {
	origApply := applyMigrations
	origOpenDB := openDB
	origBlob := newBlobStorage
	t.Cleanup(func() {
		applyMigrations = origApply
		openDB = origOpenDB
		newBlobStorage = origBlob
	})

	applyMigrations = func(_ *sql.DB) error { return nil }
	openDB = func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		return db, nil
	}
	newBlobStorage = func(config.BlobStorageConfig) (repositories.BlobStorage, error) {
		return nil, errors.New("blob boom")
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{DSN: "postgres://test@localhost/test"},
		Blob:     config.BlobStorageConfig{Provider: "s3"},
	}
	log := zerolog.Nop()

	deps, cleanup, err := wireDeps(cfg, log)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to initialize blob storage")
	require.Contains(t, err.Error(), "blob boom")
	if cleanup != nil {
		cleanup()
	}
	require.Equal(t, app.AppDeps{}, deps)
}
