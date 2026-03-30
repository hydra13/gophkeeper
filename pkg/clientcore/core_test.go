package clientcore

import (
	"context"
	"fmt"
	"testing"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/apiclient"
	"github.com/hydra13/gophkeeper/pkg/cache"
	"github.com/hydra13/gophkeeper/pkg/clientcore/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCore(t *testing.T) (*ClientCore, *mocks.MockTransport, *cache.FileStore) {
	t.Helper()
	dir := t.TempDir()
	store, err := cache.NewFileStore(dir)
	require.NoError(t, err)

	transport := &mocks.MockTransport{}
	core := New(transport, store, Config{DeviceID: "test-device"})

	return core, transport, store
}

func loginHelper(t *testing.T, core *ClientCore) {
	t.Helper()
	err := core.Login(context.Background(), "test@example.com", "password123")
	require.NoError(t, err)
}

// --- Auth ---

func TestClientCore_Login_Success(t *testing.T) {
	core, _, _ := newTestCore(t)

	err := core.Login(context.Background(), "test@example.com", "password123")
	require.NoError(t, err)
	assert.True(t, core.IsAuthenticated())
}

func TestClientCore_Register_Success(t *testing.T) {
	core, _, _ := newTestCore(t)

	err := core.Register(context.Background(), "new@example.com", "password123")
	require.NoError(t, err)
}

func TestClientCore_Login_Failure(t *testing.T) {
	core, transport, _ := newTestCore(t)
	transport.LoginFunc = func(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
		return "", "", fmt.Errorf("invalid credentials")
	}

	err := core.Login(context.Background(), "test@example.com", "wrong")
	assert.Error(t, err)
	assert.False(t, core.IsAuthenticated())
}

func TestClientCore_Logout(t *testing.T) {
	core, _, _ := newTestCore(t)
	loginHelper(t, core)

	err := core.Logout(context.Background())
	require.NoError(t, err)
	assert.False(t, core.IsAuthenticated())
}

func TestClientCore_RestoreAuth(t *testing.T) {
	core, _, store := newTestCore(t)

	// Без авторизации
	assert.False(t, core.RestoreAuth())

	// С авторизацией
	store.Auth().Set(cache.AuthData{
		AccessToken:  "cached-token",
		RefreshToken: "cached-refresh",
		DeviceID:     "test-device",
	})
	assert.True(t, core.RestoreAuth())
	assert.True(t, core.IsAuthenticated())
}

// --- Records online ---

func TestClientCore_SaveRecord_Create_Online(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	created := false
	transport.CreateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		created = true
		result := *record
		result.ID = 42
		result.Revision = 1
		return &result, nil
	}

	rec := &models.Record{
		Type:    models.RecordTypeLogin,
		Name:    "my-login",
		Payload: models.LoginPayload{Login: "user", Password: "pass"},
	}

	result, err := core.SaveRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, int64(42), result.ID)
	assert.Equal(t, int64(1), result.Revision)
}

func TestClientCore_SaveRecord_Update_Online(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	updated := false
	transport.UpdateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		updated = true
		result := *record
		result.Revision = 2
		return &result, nil
	}

	rec := &models.Record{
		ID:       1,
		Type:     models.RecordTypeLogin,
		Name:     "updated-login",
		Payload:  models.LoginPayload{Login: "new-user", Password: "new-pass"},
		Revision: 1,
	}

	result, err := core.SaveRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, int64(2), result.Revision)
}

func TestClientCore_DeleteRecord_Online(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	// Добавим запись в кеш
	store.Records().Put(&models.Record{ID: 1, Name: "to-delete", Revision: 1})

	deleted := false
	transport.DeleteRecordFunc = func(ctx context.Context, id int64) error {
		deleted = true
		assert.Equal(t, int64(1), id)
		return nil
	}

	err := core.DeleteRecord(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, deleted)

	_, ok := store.Records().Get(1)
	assert.False(t, ok)
}

// --- Offline mode ---

func TestClientCore_SaveRecord_Offline(t *testing.T) {
	core, _, store := newTestCore(t)
	// Не логинимся — офлайн

	rec := &models.Record{
		Type:    models.RecordTypeText,
		Name:    "offline-note",
		Payload: models.TextPayload{Content: "hello offline"},
	}

	result, err := core.SaveRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.ID) // сервер не дал ID

	// Запись в кеше
	got, ok := store.Records().Get(0)
	assert.True(t, ok)
	assert.Equal(t, "offline-note", got.Name)

	// Pending операция добавлена
	assert.Equal(t, 1, store.Pending().Len())
	ops, err := store.Pending().Peek()
	require.NoError(t, err)
	assert.Equal(t, cache.OperationCreate, ops[0].Operation)
}

func TestClientCore_DeleteRecord_Offline(t *testing.T) {
	core, _, store := newTestCore(t)

	// Добавим запись в кеш напрямую
	store.Records().Put(&models.Record{
		ID:       5,
		Name:     "will-delete-offline",
		Type:     models.RecordTypeText,
		Payload:  models.TextPayload{Content: "data"},
		Revision: 3,
	})

	err := core.DeleteRecord(context.Background(), 5)
	require.NoError(t, err)

	// Запись помечена как удалённая
	got, ok := store.Records().Get(5)
	assert.True(t, ok)
	assert.True(t, got.IsDeleted())

	// Pending операция
	assert.Equal(t, 1, store.Pending().Len())
	ops, err := store.Pending().Peek()
	require.NoError(t, err)
	assert.Equal(t, cache.OperationDelete, ops[0].Operation)
}

func TestClientCore_GetRecord_FromCache(t *testing.T) {
	core, _, store := newTestCore(t)

	store.Records().Put(&models.Record{
		ID:      10,
		Name:    "cached",
		Type:    models.RecordTypeLogin,
		Payload: models.LoginPayload{Login: "u", Password: "p"},
	})

	rec, err := core.GetRecord(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, "cached", rec.Name)
}

func TestClientCore_ListRecords_FilterByType(t *testing.T) {
	core, _, store := newTestCore(t)

	store.Records().Put(&models.Record{ID: 1, Name: "login1", Type: models.RecordTypeLogin, Payload: models.LoginPayload{}})
	store.Records().Put(&models.Record{ID: 2, Name: "text1", Type: models.RecordTypeText, Payload: models.TextPayload{}})
	store.Records().Put(&models.Record{ID: 3, Name: "login2", Type: models.RecordTypeLogin, Payload: models.LoginPayload{}})

	records, err := core.ListRecords(context.Background(), models.RecordTypeLogin)
	require.NoError(t, err)
	assert.Len(t, records, 2)

	records, err = core.ListRecords(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

// --- Sync ---

func TestClientCore_SyncNow(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	// Добавим pending операцию
	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  1,
		Operation: cache.OperationCreate,
		Record: &models.Record{
			ID:      1,
			Name:    "sync-me",
			Type:    models.RecordTypeText,
			Payload: models.TextPayload{Content: "data"},
		},
	})

	created := false
	transport.CreateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		created = true
		result := *record
		result.Revision = 1
		return &result, nil
	}

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		return &apiclient.PullResult{NextRevision: 1}, nil
	}

	err := core.SyncNow(context.Background())
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, 0, store.Pending().Len())
	assert.Equal(t, int64(1), store.Sync().Get().LastRevision)
}

func TestClientCore_SyncNow_Offline(t *testing.T) {
	core, _, _ := newTestCore(t)
	// Не логинимся

	err := core.SyncNow(context.Background())
	assert.Error(t, err)
}

func TestClientCore_FlushPending_RetryOnError(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  1,
		Operation: cache.OperationCreate,
		Record: &models.Record{
			ID:      1,
			Name:    "will-fail",
			Type:    models.RecordTypeText,
			Payload: models.TextPayload{Content: "x"},
		},
	})

	transport.CreateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		return nil, fmt.Errorf("server error")
	}

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		return &apiclient.PullResult{}, nil
	}

	err := core.SyncNow(context.Background())
	assert.Error(t, err)

	// Pending операция должна вернуться в очередь
	assert.Equal(t, 1, store.Pending().Len())
}

// --- Upload/Download ---

func TestClientCore_UploadBinary(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	sessionCreated := false
	transport.CreateUploadSessionFunc = func(ctx context.Context, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
		sessionCreated = true
		assert.Equal(t, int64(10), recordID)
		assert.Equal(t, int64(3), totalChunks)
		return 100, nil
	}

	chunksUploaded := 0
	transport.UploadChunkFunc = func(ctx context.Context, uploadID, chunkIndex int64, data []byte) error {
		chunksUploaded++
		assert.Equal(t, int64(100), uploadID)
		return nil
	}

	data := make([]byte, 300) // 3 чанка по 100
	for i := range data {
		data[i] = byte(i)
	}

	err := core.UploadBinary(context.Background(), 10, data, 100)
	require.NoError(t, err)
	assert.True(t, sessionCreated)
	assert.Equal(t, 3, chunksUploaded)
}

func TestClientCore_DownloadBinary(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	sessionCreated := false
	transport.CreateDownloadSessionFunc = func(ctx context.Context, recordID int64) (int64, int64, error) {
		sessionCreated = true
		assert.Equal(t, int64(10), recordID)
		return 200, 2, nil // 2 чанка
	}

	transport.DownloadChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
		return []byte(fmt.Sprintf("chunk-%d", chunkIndex)), nil
	}

	transport.ConfirmChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) error {
		return nil
	}

	data, err := core.DownloadBinary(context.Background(), 10, 1024)
	require.NoError(t, err)
	assert.True(t, sessionCreated)
	assert.Equal(t, "chunk-0chunk-1", string(data))
}

func TestClientCore_UploadBinary_Offline(t *testing.T) {
	core, _, _ := newTestCore(t)
	// Не логинимся

	err := core.UploadBinary(context.Background(), 10, []byte("data"), 100)
	assert.Error(t, err)
}

func TestClientCore_DownloadBinary_Offline(t *testing.T) {
	core, _, _ := newTestCore(t)
	// Не логинимся

	_, err := core.DownloadBinary(context.Background(), 10, 1024)
	assert.Error(t, err)
}

// --- Upload resume ---

func TestClientCore_UploadBinary_Resume(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	// Имитируем прерванный upload: чанк 0 загружен, 1 и 2 — нет
	store.Transfers().Save(cache.Transfer{
		ID:           50,
		Type:         cache.TransferUpload,
		RecordID:     10,
		SessionID:    50,
		TotalChunks:  3,
		CompletedIdx: 0,
		Status:       cache.TransferStatusActive,
		ChunkSize:    100,
		TotalSize:    300,
	})

	transport.GetUploadStatusFunc = func(ctx context.Context, uploadID int64) (*apiclient.UploadStatus, error) {
		return &apiclient.UploadStatus{
			UploadID:       50,
			ReceivedChunks: 1,
			TotalChunks:    3,
			MissingChunks:  []int64{1, 2},
		}, nil
	}

	chunksUploaded := []int64{}
	transport.UploadChunkFunc = func(ctx context.Context, uploadID, chunkIndex int64, data []byte) error {
		chunksUploaded = append(chunksUploaded, chunkIndex)
		return nil
	}

	data := make([]byte, 300)
	err := core.UploadBinary(context.Background(), 10, data, 100)
	require.NoError(t, err)

	// Должны быть загружены только чанки 1 и 2
	assert.Equal(t, []int64{1, 2}, chunksUploaded)
}

// --- Upload resume from paused ---

func TestClientCore_UploadBinary_ResumeFromPaused(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	// Имитируем прерванный upload, сохранённый как paused
	store.Transfers().Save(cache.Transfer{
		ID:           50,
		Type:         cache.TransferUpload,
		RecordID:     10,
		SessionID:    50,
		TotalChunks:  3,
		CompletedIdx: 0,
		Status:       cache.TransferStatusPaused,
		ChunkSize:    100,
		TotalSize:    300,
	})

	transport.GetUploadStatusFunc = func(ctx context.Context, uploadID int64) (*apiclient.UploadStatus, error) {
		return &apiclient.UploadStatus{
			UploadID:       50,
			ReceivedChunks: 1,
			TotalChunks:    3,
			MissingChunks:  []int64{1, 2},
		}, nil
	}

	chunksUploaded := []int64{}
	transport.UploadChunkFunc = func(ctx context.Context, uploadID, chunkIndex int64, data []byte) error {
		chunksUploaded = append(chunksUploaded, chunkIndex)
		return nil
	}

	data := make([]byte, 300)
	err := core.UploadBinary(context.Background(), 10, data, 100)
	require.NoError(t, err)

	// Должны быть загружены только чанки 1 и 2 (чанк 0 уже был загружен)
	assert.Equal(t, []int64{1, 2}, chunksUploaded)
}

// --- Download resume ---

func TestClientCore_DownloadBinary_Resume(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	// Имитируем прерванный download: чанк 0 скачан, 1 — нет
	store.Transfers().Save(cache.Transfer{
		ID:           60,
		Type:         cache.TransferDownload,
		RecordID:     10,
		SessionID:    60,
		TotalChunks:  2,
		CompletedIdx: 0,
		Status:       cache.TransferStatusActive,
		ChunkSize:    1024,
	})

	chunksDownloaded := []int64{}
	transport.DownloadChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
		chunksDownloaded = append(chunksDownloaded, chunkIndex)
		return []byte("data"), nil
	}

	transport.ConfirmChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) error {
		return nil
	}

	data, err := core.DownloadBinary(context.Background(), 10, 1024)
	require.NoError(t, err)
	assert.Equal(t, []int64{1}, chunksDownloaded) // только чанк 1
	assert.Equal(t, "data", string(data))
}

// --- Download resume from paused ---

func TestClientCore_DownloadBinary_ResumeFromPaused(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	// Имитируем прерванный download, сохранённый как paused
	store.Transfers().Save(cache.Transfer{
		ID:           60,
		Type:         cache.TransferDownload,
		RecordID:     10,
		SessionID:    60,
		TotalChunks:  3,
		CompletedIdx: 0,
		Status:       cache.TransferStatusPaused,
		ChunkSize:    1024,
	})

	chunksDownloaded := []int64{}
	transport.DownloadChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
		chunksDownloaded = append(chunksDownloaded, chunkIndex)
		return []byte("data"), nil
	}

	transport.ConfirmChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) error {
		return nil
	}

	data, err := core.DownloadBinary(context.Background(), 10, 1024)
	require.NoError(t, err)
	// Чанк 0 уже загружен (CompletedIdx=0), продолжаем с чанка 1
	assert.Equal(t, []int64{1, 2}, chunksDownloaded)
	assert.Equal(t, "datadata", string(data))
}

// --- GetRecord additional tests ---

func TestClientCore_GetRecord_CacheMiss_Online(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		assert.Equal(t, int64(42), id)
		return &models.Record{
			ID:       42,
			UserID:   1,
			Type:     models.RecordTypeLogin,
			Name:     "from-server",
			Payload:  models.LoginPayload{Login: "u", Password: "p"},
			Revision: 1,
			DeviceID: "test-device",
		}, nil
	}

	rec, err := core.GetRecord(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, "from-server", rec.Name)
	assert.Equal(t, int64(42), rec.ID)
}

func TestClientCore_GetRecord_TransportError(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return nil, fmt.Errorf("connection refused")
	}

	_, err := core.GetRecord(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get record")
}

func TestClientCore_GetRecord_Offline_NotFound(t *testing.T) {
	core, _, _ := newTestCore(t)
	// Не логинимся — офлайн, и в кеше пусто

	_, err := core.GetRecord(context.Background(), 999)
	assert.ErrorIs(t, err, models.ErrRecordNotFound)
}

// --- Register additional tests ---

func TestClientCore_Register_TransportError(t *testing.T) {
	core, transport, _ := newTestCore(t)

	transport.RegisterFunc = func(ctx context.Context, email, password string) (int64, error) {
		return 0, fmt.Errorf("email already exists")
	}

	err := core.Register(context.Background(), "taken@example.com", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "register")
}

func TestClientCore_Register_SetsAuthAndFlushes(t *testing.T) {
	core, transport, store := newTestCore(t)

	transport.RegisterFunc = func(ctx context.Context, email, password string) (int64, error) {
		return 42, nil
	}

	err := core.Register(context.Background(), "new@example.com", "pass123")
	require.NoError(t, err)

	// Check that auth data was set with returned userID
	authData, ok := store.Auth().Get()
	require.True(t, ok)
	assert.Equal(t, int64(42), authData.UserID)
	assert.Equal(t, "new@example.com", authData.Email)
	assert.Equal(t, "test-device", authData.DeviceID)
}

// --- UpdateRecord via SaveRecord (existing record, offline) ---

func TestClientCore_SaveRecord_Update_Offline(t *testing.T) {
	core, _, store := newTestCore(t)
	// Не логинимся — офлайн

	rec := &models.Record{
		ID:       10,
		Type:     models.RecordTypeText,
		Name:     "updated-offline",
		Payload:  models.TextPayload{Content: "new data"},
		Revision: 1,
	}

	result, err := core.SaveRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(10), result.ID)
	assert.Equal(t, "test-device", result.DeviceID)

	// Pending операция — OperationUpdate (ID > 0)
	assert.Equal(t, 1, store.Pending().Len())
	ops, err := store.Pending().Peek()
	require.NoError(t, err)
	assert.Equal(t, cache.OperationUpdate, ops[0].Operation)
}

// --- flushPending direct tests ---

func TestClientCore_FlushPending_EmptyQueue(t *testing.T) {
	core, _, _ := newTestCore(t)
	loginHelper(t, core)

	// No pending ops — should succeed immediately
	err := core.flushPending(context.Background())
	require.NoError(t, err)
}

func TestClientCore_FlushPending_UpdateOperation(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  5,
		Operation: cache.OperationUpdate,
		Record: &models.Record{
			ID:      5,
			Name:    "update-me",
			Type:    models.RecordTypeText,
			Payload: models.TextPayload{Content: "updated"},
			Revision: 1,
		},
	})

	updated := false
	transport.UpdateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		updated = true
		result := *record
		result.Revision = 2
		return &result, nil
	}

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		return &apiclient.PullResult{}, nil
	}

	err := core.SyncNow(context.Background())
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, 0, store.Pending().Len())

	// Record in cache should have revision bumped
	rec, ok := store.Records().Get(5)
	require.True(t, ok)
	assert.Equal(t, int64(2), rec.Revision)
}

func TestClientCore_FlushPending_DeleteOperation(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	store.Records().Put(&models.Record{ID: 7, Name: "to-delete", Revision: 1})

	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  7,
		Operation: cache.OperationDelete,
		Record: &models.Record{
			ID:       7,
			Name:     "to-delete",
			Revision: 1,
		},
	})

	deleted := false
	transport.DeleteRecordFunc = func(ctx context.Context, id int64) error {
		deleted = true
		assert.Equal(t, int64(7), id)
		return nil
	}

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		return &apiclient.PullResult{}, nil
	}

	err := core.SyncNow(context.Background())
	require.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, 0, store.Pending().Len())

	// Record removed from cache
	_, ok := store.Records().Get(7)
	assert.False(t, ok)
}

func TestClientCore_FlushPending_UpdateError_Requeues(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  8,
		Operation: cache.OperationUpdate,
		Record: &models.Record{
			ID:      8,
			Name:    "will-fail-update",
			Type:    models.RecordTypeText,
			Payload: models.TextPayload{Content: "x"},
		},
	})

	transport.UpdateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		return nil, fmt.Errorf("conflict")
	}

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		return &apiclient.PullResult{}, nil
	}

	err := core.SyncNow(context.Background())
	assert.Error(t, err)

	// Pending op should be back in queue
	assert.Equal(t, 1, store.Pending().Len())
}

func TestClientCore_FlushPending_DeleteError_Requeues(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  9,
		Operation: cache.OperationDelete,
		Record: &models.Record{
			ID: 9, Name: "will-fail-delete",
		},
	})

	transport.DeleteRecordFunc = func(ctx context.Context, id int64) error {
		return fmt.Errorf("forbidden")
	}

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		return &apiclient.PullResult{}, nil
	}

	err := core.SyncNow(context.Background())
	assert.Error(t, err)

	assert.Equal(t, 1, store.Pending().Len())
}

// --- syncFromServer tests ---

func TestClientCore_SyncFromServer_PullsRecords(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		assert.Equal(t, int64(0), sinceRevision)
		assert.Equal(t, "test-device", deviceID)
		return &apiclient.PullResult{
			Records: []models.Record{
				{ID: 100, Name: "pulled-1", Type: models.RecordTypeText, Payload: models.TextPayload{Content: "a"}},
				{ID: 101, Name: "pulled-2", Type: models.RecordTypeLogin, Payload: models.LoginPayload{Login: "u", Password: "p"}},
			},
			NextRevision: 5,
		}, nil
	}

	err := core.SyncNow(context.Background())
	require.NoError(t, err)

	rec, ok := store.Records().Get(100)
	require.True(t, ok)
	assert.Equal(t, "pulled-1", rec.Name)

	rec, ok = store.Records().Get(101)
	require.True(t, ok)
	assert.Equal(t, "pulled-2", rec.Name)

	assert.Equal(t, int64(5), store.Sync().Get().LastRevision)
}

func TestClientCore_SyncNow_FlushError_StopsBeforePull(t *testing.T) {
	core, transport, store := newTestCore(t)
	loginHelper(t, core)

	store.Pending().Enqueue(cache.PendingOp{
		RecordID:  1,
		Operation: cache.OperationCreate,
		Record: &models.Record{
			ID: 1, Name: "fail-first", Type: models.RecordTypeText,
			Payload: models.TextPayload{Content: "x"},
		},
	})

	pullCalled := false
	transport.CreateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		return nil, fmt.Errorf("server down")
	}
	transport.PullFunc = func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
		pullCalled = true
		return &apiclient.PullResult{}, nil
	}

	err := core.SyncNow(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flush pending")
	assert.False(t, pullCalled, "Pull should not be called when flush fails")
}

// --- SaveRecord online error ---

func TestClientCore_SaveRecord_Create_Online_TransportError(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	transport.CreateRecordFunc = func(ctx context.Context, record *models.Record) (*models.Record, error) {
		return nil, fmt.Errorf("internal server error")
	}

	rec := &models.Record{
		Type:    models.RecordTypeLogin,
		Name:    "fail-create",
		Payload: models.LoginPayload{Login: "u", Password: "p"},
	}

	_, err := core.SaveRecord(context.Background(), rec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save record")
}

// --- DeleteRecord offline for non-existent record ---

func TestClientCore_DeleteRecord_Offline_NotInCache(t *testing.T) {
	core, _, store := newTestCore(t)
	// Не логинимся, записи с ID=999 нет в кеше

	err := core.DeleteRecord(context.Background(), 999)
	require.NoError(t, err)

	// Pending не добавлена, т.к. записи нет в кеше
	assert.Equal(t, 0, store.Pending().Len())
}

// --- Logout error from transport ---

func TestClientCore_Logout_TransportError(t *testing.T) {
	core, transport, _ := newTestCore(t)
	loginHelper(t, core)

	transport.LogoutFunc = func(ctx context.Context) error {
		return fmt.Errorf("session expired")
	}

	err := core.Logout(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logout")
}

// --- Login sets access token on transport ---

func TestClientCore_Login_SetsAccessToken(t *testing.T) {
	core, transport, _ := newTestCore(t)

	var capturedToken string
	transport.SetAccessTokenFunc = func(token string) {
		capturedToken = token
	}

	err := core.Login(context.Background(), "test@example.com", "password123")
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", capturedToken)
	assert.True(t, core.IsAuthenticated())
}
