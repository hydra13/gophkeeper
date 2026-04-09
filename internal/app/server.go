//go:generate minimock -i .AuthService,.RecordService,.SyncService,.UploadService -o mocks -s _mock.go -g
package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	authLoginV1Post "github.com/hydra13/gophkeeper/internal/api/auth_login_v1_post"
	authLogoutV1Post "github.com/hydra13/gophkeeper/internal/api/auth_logout_v1_post"
	authRefreshV1Post "github.com/hydra13/gophkeeper/internal/api/auth_refresh_v1_post"
	authRegisterV1Post "github.com/hydra13/gophkeeper/internal/api/auth_register_v1_post"
	healthV1Get "github.com/hydra13/gophkeeper/internal/api/health_v1_get"
	recordsByIdBinaryV1Get "github.com/hydra13/gophkeeper/internal/api/records_by_id_binary_v1_get"
	recordsByIdV1Delete "github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_delete"
	recordsByIdV1Get "github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_get"
	recordsByIdV1Put "github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_put"
	recordsV1Get "github.com/hydra13/gophkeeper/internal/api/records_v1_get"
	recordsV1Post "github.com/hydra13/gophkeeper/internal/api/records_v1_post"
	syncPullV1Post "github.com/hydra13/gophkeeper/internal/api/sync_pull_v1_post"
	syncPushV1Post "github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post"
	uploadsByIdChunksV1Get "github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_get"
	uploadsByIdChunksV1Post "github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_post"
	uploadsByIdV1Get "github.com/hydra13/gophkeeper/internal/api/uploads_by_id_v1_get"
	uploadsV1Post "github.com/hydra13/gophkeeper/internal/api/uploads_v1_post"
	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/jobs/reencrypt"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc"
)

const (
	defaultShutdownTimeout = 10 * time.Second
	defaultRateLimit       = 100
	defaultRateWindow      = time.Second
)

// AuthService описывает операции аутентификации.
type AuthService interface {
	Register(ctx context.Context, email, password string) (int64, error)
	Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error)
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, accessToken string) error
	ValidateToken(token string) (int64, error)
	ValidateSession(token string) (int64, error)
}

// RecordService описывает операции с записями.
type RecordService interface {
	CreateRecord(record *models.Record) error
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	GetRecord(id int64) (*models.Record, error)
	UpdateRecord(record *models.Record) error
	DeleteRecord(id int64, deviceID string) error
}

// SyncService описывает операции синхронизации.
type SyncService interface {
	Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
	Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
	GetConflicts(userID int64) ([]models.SyncConflict, error)
	ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error)
}

// UploadService описывает операции загрузки и скачивания файлов.
type UploadService interface {
	CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
	GetUploadStatus(uploadID int64) (*models.UploadStatusResponse, error)
	UploadChunk(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error)
	DownloadChunk(uploadID, chunkIndex int64) (*models.ChunkDownloadResponse, error)
	GetUploadSessionByID(uploadID int64) (*models.UploadSession, error)
	CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error)
	DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error)
	ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error)
	GetDownloadStatus(downloadID int64) (*models.DownloadSession, error)
}

// AppDeps хранит зависимости приложения.
type AppDeps struct {
	AuthService   AuthService
	RecordService RecordService
	SyncService   SyncService
	UploadService UploadService
}

func (d AppDeps) validate() error {
	switch {
	case d.AuthService == nil:
		return errors.New("auth service dependency is required")
	case d.RecordService == nil:
		return errors.New("record service dependency is required")
	case d.SyncService == nil:
		return errors.New("sync service dependency is required")
	case d.UploadService == nil:
		return errors.New("upload service dependency is required")
	default:
		return nil
	}
}

// Run запускает HTTP-, gRPC-серверы и фоновые задачи.
func Run(ctx context.Context, cfg *config.Config, log zerolog.Logger, deps AppDeps) error {
	if err := deps.validate(); err != nil {
		return err
	}

	limiter := middlewares.NewRateLimiter(defaultRateLimit, defaultRateWindow)

	httpServer, err := buildHTTPServer(cfg, log, limiter, deps)
	if err != nil {
		return err
	}

	grpcServer, grpcListener, err := buildGRPCServer(cfg, log, limiter, deps)
	if err != nil {
		return err
	}

	jobs := []backgroundJob{reencrypt.New()}
	for _, job := range jobs {
		if err := job.Start(ctx); err != nil {
			return err
		}
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Str("address", cfg.Server.Address).Msg("http server started")
		if err := httpServer.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("http server error")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Str("address", cfg.Server.GRPCAddress).Msg("grpc server started")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error().Err(err).Msg("grpc server error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	shutdownWg := sync.WaitGroup{}

	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
			log.Info().Msg("grpc server stopped gracefully")
		case <-shutdownCtx.Done():
			log.Warn().Msg("grpc graceful stop timed out, forcing stop")
			grpcServer.Stop()
		}
	}()

	shutdownWg.Add(1)
	go func() {
		defer shutdownWg.Done()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("http server shutdown error")
		}
	}()

	for _, job := range jobs {
		shutdownWg.Add(1)
		go func(j backgroundJob) {
			defer shutdownWg.Done()
			if err := j.Stop(shutdownCtx); err != nil {
				log.Error().Err(err).Msg("job stop error")
			}
		}(job)
	}

	shutdownWg.Wait()
	wg.Wait()
	log.Info().Msg("shutdown complete")
	return nil
}

type backgroundJob interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func buildHTTPServer(cfg *config.Config, log zerolog.Logger, limiter *middlewares.RateLimiter, deps AppDeps) (*http.Server, error) {
	authService := deps.AuthService
	recordService := deps.RecordService
	syncService := deps.SyncService
	uploadService := deps.UploadService

	healthHandler := healthV1Get.NewHandler(&healthChecker{})

	authRegisterHandler := authRegisterV1Post.NewHandler(authService, log)
	authLoginHandler := authLoginV1Post.NewHandler(authService, log)
	authRefreshHandler := authRefreshV1Post.NewHandler(authService, log)
	authLogoutHandler := authLogoutV1Post.NewHandler(authService, log)

	recordsPostHandler := recordsV1Post.NewHandler(recordService)
	recordsGetHandler := recordsV1Get.NewHandler(recordService)
	recordGetHandler := recordsByIdV1Get.NewHandler(recordService)
	recordPutHandler := recordsByIdV1Put.NewHandler(recordService)
	recordDeleteHandler := recordsByIdV1Delete.NewHandler(recordService)
	recordBinaryHandler := recordsByIdBinaryV1Get.NewHandler(recordService, uploadService)

	syncPushHandler := syncPushV1Post.NewHandler(syncService)
	syncPullHandler := syncPullV1Post.NewHandler(syncService)

	uploadsStartHandler := uploadsV1Post.NewHandler(uploadService)
	uploadStatusHandler := uploadsByIdV1Get.NewHandler(uploadService)
	uploadChunkHandler := uploadsByIdChunksV1Post.NewHandler(uploadService)
	downloadChunkHandler := uploadsByIdChunksV1Get.NewHandler(uploadService)

	r := chi.NewRouter()

	r.Use(middlewares.TLS())
	r.Use(middlewares.RateLimit(limiter))
	r.Use(middlewares.Logger(log))
	r.Use(middlewares.Compression())

	r.Group(func(r chi.Router) {
		r.Get("/api/v1/health", healthHandler.ServeHTTP)
		r.Post("/api/v1/auth/register", authRegisterHandler.Handle)
		r.Post("/api/v1/auth/login", authLoginHandler.Handle)
		r.Post("/api/v1/auth/refresh", authRefreshHandler.Handle)
		r.Post("/api/v1/auth/logout", authLogoutHandler.Handle)
	})

	r.Group(func(r chi.Router) {
		r.Use(middlewares.Auth(authService, log))
		r.Post("/api/v1/records", recordsPostHandler.Handle)
		r.Get("/api/v1/records", recordsGetHandler.Handle)
		r.Route("/api/v1/records/{id}", func(r chi.Router) {
			r.Get("/", recordGetHandler.Handle)
			r.Put("/", recordPutHandler.Handle)
			r.Delete("/", recordDeleteHandler.Handle)
			r.Get("/binary", recordBinaryHandler.Handle)
		})
		r.Post("/api/v1/sync/push", syncPushHandler.ServeHTTP)
		r.Post("/api/v1/sync/pull", syncPullHandler.ServeHTTP)
		r.Post("/api/v1/uploads", uploadsStartHandler.ServeHTTP)
		r.Get("/api/v1/uploads/{id}", uploadStatusHandler.ServeHTTP)
		r.Post("/api/v1/uploads/{id}/chunks", uploadChunkHandler.ServeHTTP)
		r.Get("/api/v1/uploads/{id}/chunks/{index}", downloadChunkHandler.ServeHTTP)
	})

	return &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, nil
}

func buildGRPCServer(cfg *config.Config, log zerolog.Logger, limiter *middlewares.RateLimiter, deps AppDeps) (*grpc.Server, net.Listener, error) {
	creds, err := credentials.NewServerTLSFromFile(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
	if err != nil {
		return nil, nil, err
	}

	allowMethods := map[string]struct{}{
		"/gophkeeper.v1.AuthService/Register":      {},
		"/gophkeeper.v1.AuthService/Login":         {},
		"/gophkeeper.v1.AuthService/Refresh":       {},
		"/gophkeeper.v1.HealthService/HealthCheck": {},
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			middlewares.RequireTLS(),
			middlewares.UnaryRateLimit(limiter),
			middlewares.UnaryAuth(deps.AuthService, allowMethods),
			middlewares.UnaryLogger(log),
		),
		grpc.ChainStreamInterceptor(
			middlewares.RequireTLSStream(),
			middlewares.StreamRateLimit(limiter),
			middlewares.StreamAuth(deps.AuthService, allowMethods),
			middlewares.StreamLogger(log),
		),
	)

	authRPCService := rpc.NewAuthService(deps.AuthService, log)
	dataService := rpc.NewDataService(deps.RecordService, log)
	syncService := rpc.NewSyncService(deps.SyncService, log)
	uploadsService := rpc.NewUploadsService(deps.UploadService, log)
	healthService := rpc.NewHealthService()

	rpc.NewServer(authRPCService, dataService, syncService, uploadsService, healthService).Register(grpcServer)

	listener, err := net.Listen("tcp", cfg.Server.GRPCAddress)
	if err != nil {
		return nil, nil, err
	}

	return grpcServer, listener, nil
}

type healthChecker struct{}

func (h *healthChecker) Health() error {
	return nil
}
