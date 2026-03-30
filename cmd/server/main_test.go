package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/app"
	"github.com/hydra13/gophkeeper/internal/config"
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
	t.Cleanup(func() {
		applyMigrations = origApply
		openDB = origOpenDB
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
