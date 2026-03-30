package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post"
	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/config"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc"
)


// ---------------------------------------------------------------------------
// validate
// ---------------------------------------------------------------------------

func TestValidate_AllDepsPresent(t *testing.T) {
	deps := AppDeps{
		AuthService:   &stubAuthService{},
		RecordService: &stubRecordService{},
		SyncService:   &stubSyncService{},
		UploadService: &stubUploadsService{},
	}
	require.NoError(t, deps.validate())
}

func TestValidate_MissingAuth(t *testing.T) {
	deps := AppDeps{
		RecordService: &stubRecordService{},
		SyncService:   &stubSyncService{},
		UploadService: &stubUploadsService{},
	}
	err := deps.validate()
	require.Error(t, err)
	require.Equal(t, "auth service dependency is required", err.Error())
}

func TestValidate_MissingRecord(t *testing.T) {
	deps := AppDeps{
		AuthService:   &stubAuthService{},
		SyncService:   &stubSyncService{},
		UploadService: &stubUploadsService{},
	}
	err := deps.validate()
	require.Error(t, err)
	require.Equal(t, "record service dependency is required", err.Error())
}

func TestValidate_MissingSync(t *testing.T) {
	deps := AppDeps{
		AuthService:   &stubAuthService{},
		RecordService: &stubRecordService{},
		UploadService: &stubUploadsService{},
	}
	err := deps.validate()
	require.Error(t, err)
	require.Equal(t, "sync service dependency is required", err.Error())
}

func TestValidate_MissingUpload(t *testing.T) {
	deps := AppDeps{
		AuthService:   &stubAuthService{},
		RecordService: &stubRecordService{},
		SyncService:   &stubSyncService{},
	}
	err := deps.validate()
	require.Error(t, err)
	require.Equal(t, "upload service dependency is required", err.Error())
}

func TestValidate_AllMissing(t *testing.T) {
	deps := AppDeps{}
	err := deps.validate()
	require.Error(t, err)
	// AuthService is checked first
	require.Equal(t, "auth service dependency is required", err.Error())
}

// ---------------------------------------------------------------------------
// NewStubDeps
// ---------------------------------------------------------------------------

func TestNewStubDeps_ReturnsAllDeps(t *testing.T) {
	deps := NewStubDeps()
	require.NotNil(t, deps.AuthService)
	require.NotNil(t, deps.RecordService)
	require.NotNil(t, deps.SyncService)
	require.NotNil(t, deps.UploadService)
	require.NoError(t, deps.validate())
}

// ---------------------------------------------------------------------------
// stubAuthService
// ---------------------------------------------------------------------------

func TestStubAuthService_ValidateToken_EmptyToken(t *testing.T) {
	svc := &stubAuthService{}
	id, err := svc.ValidateToken("")
	require.Error(t, err)
	require.Equal(t, int64(0), id)
}

func TestStubAuthService_ValidateToken_ValidToken(t *testing.T) {
	svc := &stubAuthService{}
	id, err := svc.ValidateToken("some-token")
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
}

func TestStubAuthService_ValidateSession_EmptyToken(t *testing.T) {
	svc := &stubAuthService{}
	id, err := svc.ValidateSession("")
	require.Error(t, err)
	require.Equal(t, int64(0), id)
}

func TestStubAuthService_ValidateSession_ValidToken(t *testing.T) {
	svc := &stubAuthService{}
	id, err := svc.ValidateSession("some-token")
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
}

func TestStubAuthService_Register(t *testing.T) {
	svc := &stubAuthService{}
	_, err := svc.Register(context.Background(), "user@test.com", "pass")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubAuthService_Login(t *testing.T) {
	svc := &stubAuthService{}
	_, _, err := svc.Login(context.Background(), "user@test.com", "pass", "dev1", "phone", "mobile")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubAuthService_Refresh(t *testing.T) {
	svc := &stubAuthService{}
	_, _, err := svc.Refresh(context.Background(), "refresh-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubAuthService_Logout(t *testing.T) {
	svc := &stubAuthService{}
	err := svc.Logout(context.Background(), "access-token")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

// ---------------------------------------------------------------------------
// stubRecordService
// ---------------------------------------------------------------------------

func TestStubRecordService_CreateRecord(t *testing.T) {
	svc := &stubRecordService{}
	err := svc.CreateRecord(&models.Record{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubRecordService_ListRecords(t *testing.T) {
	svc := &stubRecordService{}
	_, err := svc.ListRecords(1, models.RecordTypeLogin, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubRecordService_GetRecord(t *testing.T) {
	svc := &stubRecordService{}
	_, err := svc.GetRecord(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubRecordService_UpdateRecord(t *testing.T) {
	svc := &stubRecordService{}
	err := svc.UpdateRecord(&models.Record{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubRecordService_DeleteRecord(t *testing.T) {
	svc := &stubRecordService{}
	err := svc.DeleteRecord(1, "dev1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

// ---------------------------------------------------------------------------
// stubSyncService
// ---------------------------------------------------------------------------

func TestStubSyncService_Push(t *testing.T) {
	svc := &stubSyncService{}
	_, _, err := svc.Push(1, "dev1", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubSyncService_Pull(t *testing.T) {
	svc := &stubSyncService{}
	_, _, _, err := svc.Pull(1, "dev1", 0, 50)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubSyncService_GetConflicts(t *testing.T) {
	svc := &stubSyncService{}
	_, err := svc.GetConflicts(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubSyncService_ResolveConflict(t *testing.T) {
	svc := &stubSyncService{}
	_, err := svc.ResolveConflict(1, 1, "local")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

// ---------------------------------------------------------------------------
// stubUploadsService
// ---------------------------------------------------------------------------

func TestStubUploadsService_CreateSession(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.CreateSession(1, 1, 10, 1024, 10240, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubUploadsService_GetUploadStatus(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.GetUploadStatus(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubUploadsService_UploadChunk(t *testing.T) {
	svc := &stubUploadsService{}
	received, total, completed, missing, err := svc.UploadChunk(1, 0, []byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
	require.Equal(t, int64(0), received)
	require.Equal(t, int64(0), total)
	require.False(t, completed)
	require.Nil(t, missing)
}

func TestStubUploadsService_DownloadChunk(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.DownloadChunk(1, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubUploadsService_GetUploadSessionByID(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.GetUploadSessionByID(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubUploadsService_CreateDownloadSession(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.CreateDownloadSession(1, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubUploadsService_DownloadChunkByID(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.DownloadChunkByID(1, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

func TestStubUploadsService_ConfirmChunk(t *testing.T) {
	svc := &stubUploadsService{}
	confirmed, total, status, err := svc.ConfirmChunk(1, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
	require.Equal(t, int64(0), confirmed)
	require.Equal(t, int64(0), total)
	require.Equal(t, models.DownloadStatus(""), status)
}

func TestStubUploadsService_GetDownloadStatus(t *testing.T) {
	svc := &stubUploadsService{}
	_, err := svc.GetDownloadStatus(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not implemented")
}

// ---------------------------------------------------------------------------
// healthChecker
// ---------------------------------------------------------------------------

func TestHealthChecker_Health(t *testing.T) {
	hc := &healthChecker{}
	require.NoError(t, hc.Health())
}

// ---------------------------------------------------------------------------
// isPublicPath
// ---------------------------------------------------------------------------

func TestIsPublicPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "health endpoint", path: "/api/v1/health", want: true},
		{name: "auth register", path: "/api/v1/auth/register", want: true},
		{name: "auth login", path: "/api/v1/auth/login", want: true},
		{name: "auth refresh", path: "/api/v1/auth/refresh", want: true},
		{name: "auth logout", path: "/api/v1/auth/logout", want: true},
		{name: "auth prefix with trailing slash", path: "/api/v1/auth/", want: true},
		{name: "records endpoint", path: "/api/v1/records", want: false},
		{name: "sync push", path: "/api/v1/sync/push", want: false},
		{name: "uploads", path: "/api/v1/uploads", want: false},
		{name: "root", path: "/", want: false},
		{name: "unknown path", path: "/api/v1/unknown", want: false},
		{name: "empty path", path: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isPublicPath(tt.path))
		})
	}
}

// ---------------------------------------------------------------------------
// recordToDTO
// ---------------------------------------------------------------------------

func TestRecordToDTO_NilRecord(t *testing.T) {
	dto := recordToDTO(nil)
	require.Equal(t, recordscommon.RecordDTO{}, dto)
}

func TestRecordToDTO_LoginPayload(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	record := &models.Record{
		ID:             1,
		UserID:         10,
		Type:           models.RecordTypeLogin,
		Name:           "My Login",
		Metadata:       "meta",
		Payload:        models.LoginPayload{Login: "user@example.com", Password: "secret"},
		Revision:       5,
		DeviceID:       "dev-1",
		KeyVersion:     2,
		PayloadVersion: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	dto := recordToDTO(record)

	require.Equal(t, int64(1), dto.ID)
	require.Equal(t, int64(10), dto.UserID)
	require.Equal(t, "login", dto.Type)
	require.Equal(t, "My Login", dto.Name)
	require.Equal(t, "meta", dto.Metadata)
	require.Equal(t, int64(5), dto.Revision)
	require.Equal(t, "dev-1", dto.DeviceID)
	require.Equal(t, int64(2), dto.KeyVersion)
	require.Equal(t, "2025-06-15T12:30:00Z", dto.CreatedAt)
	require.Equal(t, "2025-06-15T12:30:00Z", dto.UpdatedAt)
	require.Nil(t, dto.DeletedAt)

	loginPayload, ok := dto.Payload.(recordscommon.LoginPayloadDTO)
	require.True(t, ok)
	require.Equal(t, "user@example.com", loginPayload.Login)
	require.Equal(t, "secret", loginPayload.Password)
}

func TestRecordToDTO_TextPayload(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	record := &models.Record{
		ID:        2,
		UserID:    20,
		Type:      models.RecordTypeText,
		Name:      "Note",
		Payload:   models.TextPayload{Content: "hello world"},
		DeviceID:  "dev-2",
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := recordToDTO(record)

	require.Equal(t, "text", dto.Type)
	textPayload, ok := dto.Payload.(recordscommon.TextPayloadDTO)
	require.True(t, ok)
	require.Equal(t, "hello world", textPayload.Content)
}

func TestRecordToDTO_BinaryPayload(t *testing.T) {
	now := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	record := &models.Record{
		ID:             3,
		UserID:         30,
		Type:           models.RecordTypeBinary,
		Name:           "File",
		Payload:        models.BinaryPayload{Data: []byte("bin")},
		PayloadVersion: 1,
		DeviceID:       "dev-3",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	dto := recordToDTO(record)

	require.Equal(t, "binary", dto.Type)
	require.Equal(t, int64(1), dto.PayloadVersion)
	_, ok := dto.Payload.(recordscommon.BinaryPayloadDTO)
	require.True(t, ok)
}

func TestRecordToDTO_CardPayload(t *testing.T) {
	now := time.Date(2025, 7, 20, 15, 45, 0, 0, time.UTC)
	record := &models.Record{
		ID:        4,
		UserID:    40,
		Type:      models.RecordTypeCard,
		Name:      "Visa",
		Payload:   models.CardPayload{Number: "4111111111111111", HolderName: "John Doe", ExpiryDate: "12/28", CVV: "123"},
		DeviceID:  "dev-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := recordToDTO(record)

	require.Equal(t, "card", dto.Type)
	cardPayload, ok := dto.Payload.(recordscommon.CardPayloadDTO)
	require.True(t, ok)
	require.Equal(t, "4111111111111111", cardPayload.Number)
	require.Equal(t, "John Doe", cardPayload.HolderName)
	require.Equal(t, "12/28", cardPayload.ExpiryDate)
	require.Equal(t, "123", cardPayload.CVV)
}

func TestRecordToDTO_WithDeletedAt(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	deletedAt := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	record := &models.Record{
		ID:        5,
		UserID:    50,
		Type:      models.RecordTypeText,
		Name:      "Deleted",
		Payload:   models.TextPayload{Content: "gone"},
		DeletedAt: &deletedAt,
		DeviceID:  "dev-5",
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := recordToDTO(record)

	require.NotNil(t, dto.DeletedAt)
	require.Equal(t, "2025-07-01T00:00:00Z", *dto.DeletedAt)
}

func TestRecordToDTO_NoDeletedAt(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	record := &models.Record{
		ID:        6,
		UserID:    60,
		Type:      models.RecordTypeText,
		Name:      "Active",
		Payload:   models.TextPayload{Content: "active"},
		DeviceID:  "dev-6",
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := recordToDTO(record)

	require.Nil(t, dto.DeletedAt)
}

func TestRecordToDTO_NilPayload(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	record := &models.Record{
		ID:        7,
		UserID:    70,
		Type:      models.RecordTypeLogin,
		Name:      "NoPayload",
		DeviceID:  "dev-7",
		CreatedAt: now,
		UpdatedAt: now,
		// Payload is nil
	}

	dto := recordToDTO(record)

	require.Nil(t, dto.Payload)
}

// ---------------------------------------------------------------------------
// chain
// ---------------------------------------------------------------------------

func TestChain_AppliesMiddlewaresInOrder(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	handler := chain(inner, mw1, mw2)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}, order)
}

func TestChain_NoMiddlewares(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := chain(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, called)
}

// ---------------------------------------------------------------------------
// methodHandler
// ---------------------------------------------------------------------------

func TestMethodHandler_CorrectMethod(t *testing.T) {
	called := false
	handler := methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestMethodHandler_WrongMethod(t *testing.T) {
	called := false
	handler := methodHandler(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.False(t, called)
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	require.Contains(t, rec.Body.String(), "method not allowed")
}

// ---------------------------------------------------------------------------
// methodRouter
// ---------------------------------------------------------------------------

func TestMethodRouter_MatchingMethod(t *testing.T) {
	getCalled := false
	postCalled := false

	handler := methodRouter(map[string]func(http.ResponseWriter, *http.Request){
		http.MethodGet: func(w http.ResponseWriter, r *http.Request) {
			getCalled = true
		},
		http.MethodPost: func(w http.ResponseWriter, r *http.Request) {
			postCalled = true
		},
	})

	// Test GET
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, getCalled)
	require.False(t, postCalled)

	// Test POST
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, postCalled)
}

func TestMethodRouter_NoMatchingMethod(t *testing.T) {
	handler := methodRouter(map[string]func(http.ResponseWriter, *http.Request){
		http.MethodGet: func(w http.ResponseWriter, r *http.Request) {},
	})

	req := httptest.NewRequest(http.MethodDelete, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	require.Contains(t, rec.Body.String(), "method not allowed")
}

func TestMethodRouter_EmptyMap(t *testing.T) {
	handler := methodRouter(map[string]func(http.ResponseWriter, *http.Request){})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// ---------------------------------------------------------------------------
// syncUseCaseAdapter
// ---------------------------------------------------------------------------

type mockSyncService struct {
	pushedChanges []sync_push_v1_post.PendingChange
	pushErr       error
	pullErr       error
	conflictsErr  error
	resolveErr    error
}

func (m *mockSyncService) Push(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
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

func TestSyncUseCaseAdapter_Push_ConvertsChanges(t *testing.T) {
	mock := &mockSyncService{}
	adapter := &syncUseCaseAdapter{svc: mock}

	record := &models.Record{
		ID:        1,
		UserID:    10,
		Type:      models.RecordTypeLogin,
		Name:      "Test",
		Payload:   models.LoginPayload{Login: "user", Password: "pass"},
		DeviceID:  "dev-1",
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	changes := []rpc.PendingChange{
		{
			Record:       record,
			Deleted:      false,
			BaseRevision: 3,
		},
	}

	_, _, err := adapter.Push(10, "dev-1", changes)
	require.NoError(t, err)

	require.Len(t, mock.pushedChanges, 1)
	require.Equal(t, int64(1), mock.pushedChanges[0].Record.ID)
	require.Equal(t, "login", mock.pushedChanges[0].Record.Type)
	require.Equal(t, "Test", mock.pushedChanges[0].Record.Name)
	require.False(t, mock.pushedChanges[0].Deleted)
	require.Equal(t, int64(3), mock.pushedChanges[0].BaseRevision)
}

func TestSyncUseCaseAdapter_Push_EmptyChanges(t *testing.T) {
	mock := &mockSyncService{}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, _, err := adapter.Push(1, "dev-1", nil)
	require.NoError(t, err)
	require.Empty(t, mock.pushedChanges)
}

func TestSyncUseCaseAdapter_Push_Error(t *testing.T) {
	mock := &mockSyncService{pushErr: context.DeadlineExceeded}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, _, err := adapter.Push(1, "dev-1", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSyncUseCaseAdapter_Pull(t *testing.T) {
	mock := &mockSyncService{}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, _, _, err := adapter.Pull(1, "dev-1", 0, 50)
	require.NoError(t, err)
}

func TestSyncUseCaseAdapter_Pull_Error(t *testing.T) {
	mock := &mockSyncService{pullErr: context.DeadlineExceeded}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, _, _, err := adapter.Pull(1, "dev-1", 0, 50)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSyncUseCaseAdapter_GetConflicts(t *testing.T) {
	mock := &mockSyncService{}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, err := adapter.GetConflicts(1)
	require.NoError(t, err)
}

func TestSyncUseCaseAdapter_GetConflicts_Error(t *testing.T) {
	mock := &mockSyncService{conflictsErr: context.DeadlineExceeded}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, err := adapter.GetConflicts(1)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSyncUseCaseAdapter_ResolveConflict(t *testing.T) {
	mock := &mockSyncService{}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, err := adapter.ResolveConflict(1, 42, "local")
	require.NoError(t, err)
}

func TestSyncUseCaseAdapter_ResolveConflict_Error(t *testing.T) {
	mock := &mockSyncService{resolveErr: context.DeadlineExceeded}
	adapter := &syncUseCaseAdapter{svc: mock}

	_, err := adapter.ResolveConflict(1, 42, "local")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSyncUseCaseAdapter_Push_RecordToDTO_NilRecord(t *testing.T) {
	mock := &mockSyncService{}
	adapter := &syncUseCaseAdapter{svc: mock}

	changes := []rpc.PendingChange{
		{Record: nil, Deleted: true, BaseRevision: 1},
	}

	_, _, err := adapter.Push(1, "dev-1", changes)
	require.NoError(t, err)
	require.Len(t, mock.pushedChanges, 1)
	// recordToDTO(nil) returns empty RecordDTO
	require.Equal(t, recordscommon.RecordDTO{}, mock.pushedChanges[0].Record)
	require.True(t, mock.pushedChanges[0].Deleted)
}

// ---------------------------------------------------------------------------
// buildHTTPServer (integration-level: verify server builds without error)
// ---------------------------------------------------------------------------

func TestBuildHTTPServer_Success(t *testing.T) {
	deps := NewStubDeps()
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

// Note: buildGRPCServer requires valid TLS certs so it's not unit-tested here.
// The Run function also requires TLS certs and background jobs, making it
// better suited for integration tests.
