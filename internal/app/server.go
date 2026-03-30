package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/hydra13/gophkeeper/internal/api/auth_login_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/auth_logout_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/auth_refresh_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/auth_register_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/health_v1_get"
	"github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_delete"
	"github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_get"
	"github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_put"
	"github.com/hydra13/gophkeeper/internal/api/records_v1_get"
	"github.com/hydra13/gophkeeper/internal/api/records_v1_post"
	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/api/sync_pull_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_get"
	"github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_post"
	"github.com/hydra13/gophkeeper/internal/api/uploads_by_id_v1_get"
	"github.com/hydra13/gophkeeper/internal/api/uploads_v1_post"
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

// AppDeps описывает зависимости приложения.
type AppDeps struct {
	// AuthService реализует все auth-операции: register, login, refresh, logout, validate token.
	AuthService interface {
		Register(ctx context.Context, email, password string) (int64, error)
		Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error)
		Refresh(ctx context.Context, refreshToken string) (string, string, error)
		Logout(ctx context.Context, accessToken string) error
		ValidateToken(token string) (int64, error)
		ValidateSession(token string) (int64, error)
	}
	RecordService interface {
		CreateRecord(record *models.Record) error
		ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
		GetRecord(id int64) (*models.Record, error)
		UpdateRecord(record *models.Record) error
		DeleteRecord(id int64, deviceID string) error
	}
	SyncService interface {
		Push(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
		Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
		GetConflicts(userID int64) ([]models.SyncConflict, error)
		ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error)
	}
	UploadService interface {
		CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
		GetUploadStatus(uploadID int64) (*uploads_by_id_v1_get.UploadStatusResponse, error)
		UploadChunk(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error)
		DownloadChunk(uploadID, chunkIndex int64) (*uploads_by_id_chunks_v1_get.ChunkDownloadResponse, error)
		GetUploadSessionByID(uploadID int64) (*models.UploadSession, error)
		CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error)
		DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error)
		ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error)
		GetDownloadStatus(downloadID int64) (*models.DownloadSession, error)
	}
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

// NewStubDeps возвращает набор заглушек для запуска без бизнес-реализаций.
func NewStubDeps() AppDeps {
	return AppDeps{
		AuthService:   &stubAuthService{},
		RecordService: &stubRecordService{},
		SyncService:   &stubSyncService{},
		UploadService: &stubUploadsService{},
	}
}

// Run поднимает HTTP и gRPC серверы и обеспечивает graceful shutdown.
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

	healthHandler := health_v1_get.NewHandler(&healthChecker{})

	authRegisterHandler := auth_register_v1_post.NewHandler(authService, log)
	authLoginHandler := auth_login_v1_post.NewHandler(authService, log)
	authRefreshHandler := auth_refresh_v1_post.NewHandler(authService, log)
	authLogoutHandler := auth_logout_v1_post.NewHandler(authService, log)

	recordsPostHandler := recordsv1post.NewHandler(recordService)
	recordsGetHandler := recordsv1get.NewHandler(recordService)
	recordGetHandler := recordsbyidv1get.NewHandler(recordService)
	recordPutHandler := recordsbyidv1put.NewHandler(recordService)
	recordDeleteHandler := recordsbyidv1delete.NewHandler(recordService)

	syncPushHandler := sync_push_v1_post.NewHandler(syncService)
	syncPullHandler := sync_pull_v1_post.NewHandler(syncService)

	uploadsStartHandler := uploads_v1_post.NewHandler(uploadService)
	uploadStatusHandler := uploads_by_id_v1_get.NewHandler(uploadService)
	uploadChunkHandler := uploads_by_id_chunks_v1_post.NewHandler(uploadService)
	downloadChunkHandler := uploads_by_id_chunks_v1_get.NewHandler(uploadService)

	publicMux := http.NewServeMux()
	publicMux.Handle("/api/v1/health", healthHandler)
	publicMux.Handle("/api/v1/auth/register", methodHandler(http.MethodPost, authRegisterHandler.Handle))
	publicMux.Handle("/api/v1/auth/login", methodHandler(http.MethodPost, authLoginHandler.Handle))
	publicMux.Handle("/api/v1/auth/refresh", methodHandler(http.MethodPost, authRefreshHandler.Handle))
	publicMux.Handle("/api/v1/auth/logout", methodHandler(http.MethodPost, authLogoutHandler.Handle))

	protectedMux := http.NewServeMux()
	protectedMux.Handle("/api/v1/records", methodRouter(map[string]func(http.ResponseWriter, *http.Request){
		http.MethodPost: recordsPostHandler.Handle,
		http.MethodGet:  recordsGetHandler.Handle,
	}))
	protectedMux.Handle("/api/v1/records/{id}", methodRouter(map[string]func(http.ResponseWriter, *http.Request){
		http.MethodGet:    recordGetHandler.Handle,
		http.MethodPut:    recordPutHandler.Handle,
		http.MethodDelete: recordDeleteHandler.Handle,
	}))

	protectedMux.Handle("/api/v1/sync/push", syncPushHandler)
	protectedMux.Handle("/api/v1/sync/pull", syncPullHandler)

	protectedMux.Handle("/api/v1/uploads", uploadsStartHandler)
	protectedMux.Handle("/api/v1/uploads/{id}", uploadStatusHandler)
	protectedMux.Handle("/api/v1/uploads/{id}/chunks", uploadChunkHandler)
	protectedMux.Handle("/api/v1/uploads/{id}/chunks/{index}", downloadChunkHandler)

	protectedHandler := middlewares.Auth(authService, log)(protectedMux)
	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			publicMux.ServeHTTP(w, r)
			return
		}
		protectedHandler.ServeHTTP(w, r)
	})

	handler := chain(
		root,
		middlewares.TLS(),
		middlewares.RateLimit(limiter),
		middlewares.Logger(log),
		middlewares.Compression(),
	)

	return &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      handler,
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
	syncService := rpc.NewSyncService(&syncUseCaseAdapter{svc: deps.SyncService}, log)
	uploadsService := rpc.NewUploadsService(deps.UploadService, log)
	healthService := rpc.NewHealthService()

	rpc.NewServer(authRPCService, dataService, syncService, uploadsService, healthService).Register(grpcServer)

	listener, err := net.Listen("tcp", cfg.Server.GRPCAddress)
	if err != nil {
		return nil, nil, err
	}

	return grpcServer, listener, nil
}

func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	h := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func methodHandler(method string, handler func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	})
}

func methodRouter(handlers map[string]func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := handlers[r.Method]; ok {
			handler(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func isPublicPath(path string) bool {
	if path == "/api/v1/health" {
		return true
	}
	return strings.HasPrefix(path, "/api/v1/auth/")
}

// healthChecker always reports healthy. This is safe because the fail-fast
// bootstrap in cmd/server guarantees that all persistence dependencies
// (database connection, migrations, services) are fully initialized before
// the HTTP server starts accepting requests. If any critical dependency
// fails, the process exits with a diagnostic message and never reaches Run.
type healthChecker struct{}

func (h *healthChecker) Health() error {
	return nil
}

type stubAuthService struct{}

func (s *stubAuthService) Register(ctx context.Context, email, password string) (int64, error) {
	return 0, errors.New("auth service not implemented")
}

func (s *stubAuthService) Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
	return "", "", errors.New("auth service not implemented")
}

func (s *stubAuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	return "", "", errors.New("auth service not implemented")
}

func (s *stubAuthService) Logout(ctx context.Context, accessToken string) error {
	return errors.New("auth service not implemented")
}

func (s *stubAuthService) ValidateToken(token string) (int64, error) {
	if token == "" {
		return 0, errors.New("empty token")
	}
	return 1, nil
}

func (s *stubAuthService) ValidateSession(token string) (int64, error) {
	if token == "" {
		return 0, errors.New("empty token")
	}
	return 1, nil
}

type stubRecordService struct{}

func (s *stubRecordService) CreateRecord(record *models.Record) error {
	return errors.New("record service not implemented")
}

func (s *stubRecordService) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return nil, errors.New("record service not implemented")
}

func (s *stubRecordService) GetRecord(id int64) (*models.Record, error) {
	return nil, errors.New("record service not implemented")
}

func (s *stubRecordService) UpdateRecord(record *models.Record) error {
	return errors.New("record service not implemented")
}

func (s *stubRecordService) DeleteRecord(id int64, deviceID string) error {
	return errors.New("record service not implemented")
}

type stubSyncService struct{}

func (s *stubSyncService) Push(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	return nil, nil, errors.New("sync service not implemented")
}

func (s *stubSyncService) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	return nil, nil, nil, errors.New("sync service not implemented")
}

func (s *stubSyncService) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return nil, errors.New("sync service not implemented")
}

func (s *stubSyncService) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
	return nil, errors.New("sync service not implemented")
}

type stubUploadsService struct{}

func (s *stubUploadsService) CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	return 0, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) GetUploadStatus(uploadID int64) (*uploads_by_id_v1_get.UploadStatusResponse, error) {
	return nil, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) UploadChunk(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error) {
	return 0, 0, false, nil, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) DownloadChunk(uploadID, chunkIndex int64) (*uploads_by_id_chunks_v1_get.ChunkDownloadResponse, error) {
	return nil, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) GetUploadSessionByID(uploadID int64) (*models.UploadSession, error) {
	return nil, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error) {
	return nil, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error) {
	return nil, errors.New("uploads service not implemented")
}

func (s *stubUploadsService) ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error) {
	return 0, 0, "", errors.New("uploads service not implemented")
}

func (s *stubUploadsService) GetDownloadStatus(downloadID int64) (*models.DownloadSession, error) {
	return nil, errors.New("uploads service not implemented")
}

// syncUseCaseAdapter адаптирует SyncService из AppDeps к интерфейсу rpc.SyncUseCase.
type syncUseCaseAdapter struct {
	svc interface {
		Push(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
		Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
		GetConflicts(userID int64) ([]models.SyncConflict, error)
		ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error)
	}
}

func (a *syncUseCaseAdapter) Push(userID int64, deviceID string, changes []rpc.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	dtoChanges := make([]sync_push_v1_post.PendingChange, 0, len(changes))
	for _, c := range changes {
		dtoChanges = append(dtoChanges, sync_push_v1_post.PendingChange{
			Record:       recordToDTO(c.Record),
			Deleted:      c.Deleted,
			BaseRevision: c.BaseRevision,
		})
	}
	return a.svc.Push(userID, deviceID, dtoChanges)
}

func (a *syncUseCaseAdapter) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	return a.svc.Pull(userID, deviceID, sinceRevision, limit)
}

func (a *syncUseCaseAdapter) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return a.svc.GetConflicts(userID)
}

func (a *syncUseCaseAdapter) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
	return a.svc.ResolveConflict(userID, conflictID, resolution)
}

func recordToDTO(r *models.Record) recordscommon.RecordDTO {
	if r == nil {
		return recordscommon.RecordDTO{}
	}
	dto := recordscommon.RecordDTO{
		ID:             r.ID,
		UserID:         r.UserID,
		Type:           string(r.Type),
		Name:           r.Name,
		Metadata:       r.Metadata,
		Revision:       r.Revision,
		DeviceID:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
		CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      r.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if r.DeletedAt != nil {
		deletedAt := r.DeletedAt.Format("2006-01-02T15:04:05Z")
		dto.DeletedAt = &deletedAt
	}

	switch p := r.Payload.(type) {
	case models.LoginPayload:
		dto.Payload = recordscommon.LoginPayloadDTO{Login: p.Login, Password: p.Password}
	case models.TextPayload:
		dto.Payload = recordscommon.TextPayloadDTO{Content: p.Content}
	case models.BinaryPayload:
		dto.Payload = recordscommon.BinaryPayloadDTO{}
	case models.CardPayload:
		dto.Payload = recordscommon.CardPayloadDTO{
			Number: p.Number, HolderName: p.HolderName,
			ExpiryDate: p.ExpiryDate, CVV: p.CVV,
		}
	}

	return dto
}
