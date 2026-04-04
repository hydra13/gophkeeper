package app

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/app/mocks"
	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestValidate(t *testing.T) {
	authMock := mocks.NewAuthServiceMock(t)
	recordMock := mocks.NewRecordServiceMock(t)
	syncMock := mocks.NewSyncServiceMock(t)
	uploadMock := mocks.NewUploadServiceMock(t)

	tests := []struct {
		name    string
		deps    AppDeps
		wantErr string
	}{
		{
			name:    "all deps present",
			deps:    AppDeps{AuthService: authMock, RecordService: recordMock, SyncService: syncMock, UploadService: uploadMock},
			wantErr: "",
		},
		{
			name:    "missing auth",
			deps:    AppDeps{RecordService: recordMock, SyncService: syncMock, UploadService: uploadMock},
			wantErr: "auth service dependency is required",
		},
		{
			name:    "missing record",
			deps:    AppDeps{AuthService: authMock, SyncService: syncMock, UploadService: uploadMock},
			wantErr: "record service dependency is required",
		},
		{
			name:    "missing sync",
			deps:    AppDeps{AuthService: authMock, RecordService: recordMock, UploadService: uploadMock},
			wantErr: "sync service dependency is required",
		},
		{
			name:    "missing upload",
			deps:    AppDeps{AuthService: authMock, RecordService: recordMock, SyncService: syncMock},
			wantErr: "upload service dependency is required",
		},
		{
			name:    "all missing",
			deps:    AppDeps{},
			wantErr: "auth service dependency is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.deps.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestHealthChecker_Health(t *testing.T) {
	hc := &healthChecker{}
	require.NoError(t, hc.Health())
}

func TestBuildHTTPServer_Success(t *testing.T) {
	authMock := mocks.NewAuthServiceMock(t)
	recordMock := mocks.NewRecordServiceMock(t)
	syncMock := mocks.NewSyncServiceMock(t)
	uploadMock := mocks.NewUploadServiceMock(t)

	deps := AppDeps{
		AuthService:   authMock,
		RecordService: recordMock,
		SyncService:   syncMock,
		UploadService: uploadMock,
	}

	log := zerolog.Nop()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Address: ":0",
		},
	}

	limiter := middlewares.NewRateLimiter(100, time.Second)

	srv, err := buildHTTPServer(cfg, log, limiter, deps)
	require.NoError(t, err)
	require.NotNil(t, srv)
	require.Equal(t, ":0", srv.Addr)
	require.NotNil(t, srv.Handler)
}

// mockSyncService implements SyncService for testing.
type mockSyncService struct {
	pushedChanges []models.PendingChange
	pushErr       error
	pullErr       error
	conflictsErr  error
	resolveErr    error
}

func (m *mockSyncService) Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	m.pushedChanges = changes
	return nil, nil, m.pushErr
}

func (m *mockSyncService) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	return nil, nil, nil, m.pullErr
}

func (m *mockSyncService) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return nil, m.conflictsErr
}

func (m *mockSyncService) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
	return nil, m.resolveErr
}
