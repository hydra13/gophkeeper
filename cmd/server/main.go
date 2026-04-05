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
	"github.com/hydra13/gophkeeper/internal/repositories"
	dbrepo "github.com/hydra13/gophkeeper/internal/repositories/database"
	authsvc "github.com/hydra13/gophkeeper/internal/services/auth"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	keysvc "github.com/hydra13/gophkeeper/internal/services/keys"
	recordsvc "github.com/hydra13/gophkeeper/internal/services/records"
	syncsvc "github.com/hydra13/gophkeeper/internal/services/sync"
	uploadsvc "github.com/hydra13/gophkeeper/internal/services/uploads"
	"github.com/hydra13/gophkeeper/internal/storage"
)

type wireDepsFactories struct {
	applyMigrations func(*sql.DB) error
	openDB          func(string) (*sql.DB, error)
	newBlobStorage  func(config.BlobStorageConfig) (repositories.BlobStorage, error)
}

// Фабрики вынесены на уровень пакета, чтобы тесты могли подменять отдельные
// шаги сборки зависимостей без вмешательства в main.
var defaultWireDepsFactories = wireDepsFactories{
	applyMigrations: migrations.Apply,
	openDB: func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, err
		}
		return db, nil
	},
	newBlobStorage: storage.NewBlobStorage,
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
	return wireDepsWithFactories(cfg, log, defaultWireDepsFactories)
}

func wireDepsWithFactories(cfg *config.Config, log zerolog.Logger, factories wireDepsFactories) (app.AppDeps, func(), error) {
	var cleanup func()

	if cfg.Database.DSN == "" {
		return app.AppDeps{}, cleanup, errors.New("database DSN is required: set GK_DATABASE_DSN or configure database.dsn")
	}

	db, err := factories.openDB(cfg.Database.DSN)
	if err != nil {
		return app.AppDeps{}, cleanup, fmt.Errorf("failed to connect to database: %w", err)
	}
	cleanup = func() {
		if err := db.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close database connection")
		}
	}

	if err := factories.applyMigrations(db); err != nil {
		return app.AppDeps{}, cleanup, fmt.Errorf("failed to apply database migrations: %w", err)
	}

	blob, err := factories.newBlobStorage(cfg.Blob)
	if err != nil {
		return app.AppDeps{}, cleanup, fmt.Errorf("failed to initialize blob storage: %w", err)
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
