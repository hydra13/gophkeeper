package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/app"
	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/migrations"
	dbrepo "github.com/hydra13/gophkeeper/internal/repositories/database"
	authsvc "github.com/hydra13/gophkeeper/internal/services/auth"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	keysvc "github.com/hydra13/gophkeeper/internal/services/keys"
	recordsvc "github.com/hydra13/gophkeeper/internal/services/records"
	syncsvc "github.com/hydra13/gophkeeper/internal/services/sync"
	uploadsvc "github.com/hydra13/gophkeeper/internal/services/uploads"
	"github.com/hydra13/gophkeeper/internal/storage"
)

// applyMigrations allows tests to override migrations.Apply.
var applyMigrations = migrations.Apply

// openDB allows tests to override sql.Open+Ping.
var openDB = func(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	deps, cleanup, err := wireDeps(cfg, log)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		log.Fatal().Err(err).Msg("failed to wire dependencies")
	}

	if err := app.Run(ctx, cfg, log, deps); err != nil {
		log.Fatal().Err(err).Msg("server stopped with error")
	}
}

func wireDeps(cfg *config.Config, log zerolog.Logger) (app.AppDeps, func(), error) {
	var cleanup func()

	if cfg.Database.DSN == "" {
		return app.AppDeps{}, cleanup, errors.New("database DSN is required: set GK_DATABASE_DSN or configure database.dsn")
	}

	db, err := openDB(cfg.Database.DSN)
	if err != nil {
		return app.AppDeps{}, cleanup, fmt.Errorf("failed to connect to database: %w", err)
	}
	cleanup = func() { db.Close() }

	if err := applyMigrations(db); err != nil {
		return app.AppDeps{}, cleanup, fmt.Errorf("failed to apply database migrations: %w", err)
	}

	blob, err := storage.NewLocalBlob(cfg.Blob.Path)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	repo, err := dbrepo.New(db, blob)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	jwtManager, err := authsvc.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.TokenDuration)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	authService, err := authsvc.NewService(
		repo,
		repo,
		jwtManager,
		0,
	)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	keyManager, err := keysvc.NewManager(repo, cfg.Crypto.MasterKey)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	cryptoService := cryptosvc.New(keyManager)
	repo.SetCrypto(cryptoService)

	recordService, err := recordsvc.NewService(repo, keyManager)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	uploadService, err := uploadsvc.NewService(repo)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	syncService, err := syncsvc.NewService(repo, repo)
	if err != nil {
		return app.AppDeps{}, cleanup, err
	}

	return app.AppDeps{
		AuthService:   authService,
		RecordService: recordService,
		SyncService:   syncService,
		UploadService: uploadService,
	}, cleanup, nil
}
