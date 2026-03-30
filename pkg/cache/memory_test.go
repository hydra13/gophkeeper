package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	require.NoError(t, err)
	return store
}

func TestFileStore_Records_CRUD(t *testing.T) {
	s := newTestStore(t)
	rc := s.Records()

	rec := &models.Record{
		ID:       1,
		UserID:   1,
		Type:     models.RecordTypeLogin,
		Name:     "test-record",
		Payload:  models.LoginPayload{Login: "user", Password: "pass"},
		Revision: 1,
		DeviceID: "dev1",
	}

	// Put + Get
	rc.Put(rec)
	got, ok := rc.Get(1)
	assert.True(t, ok)
	assert.Equal(t, "test-record", got.Name)
	assert.Equal(t, models.RecordTypeLogin, got.Type)

	// GetAll
	all := rc.GetAll()
	assert.Len(t, all, 1)

	// Delete
	rc.Delete(1)
	_, ok = rc.Get(1)
	assert.False(t, ok)

	// Clear
	rc.Put(rec)
	rc.Clear()
	assert.Len(t, rc.GetAll(), 0)
}

func TestFileStore_Records_PutAll(t *testing.T) {
	s := newTestStore(t)
	rc := s.Records()

	records := []models.Record{
		{ID: 1, Name: "rec1", Type: models.RecordTypeText, Payload: models.TextPayload{Content: "a"}},
		{ID: 2, Name: "rec2", Type: models.RecordTypeCard, Payload: models.CardPayload{Number: "1234"}},
	}

	rc.PutAll(records)
	all := rc.GetAll()
	assert.Len(t, all, 2)
}

func TestFileStore_PendingQueue(t *testing.T) {
	s := newTestStore(t)
	pq := s.Pending()

	assert.Equal(t, 0, pq.Len())

	// Enqueue
	err := pq.Enqueue(PendingOp{
		RecordID:  1,
		Operation: OperationCreate,
		Record: &models.Record{ID: 1, Name: "test"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pq.Len())

	err = pq.Enqueue(PendingOp{
		RecordID:  2,
		Operation: OperationUpdate,
		Record:    &models.Record{ID: 2, Name: "test2"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, pq.Len())

	// Peek
	ops, err := pq.Peek()
	require.NoError(t, err)
	assert.Len(t, ops, 2)
	assert.Equal(t, 2, pq.Len()) // не удалено

	// DequeueAll
	ops, err = pq.DequeueAll()
	require.NoError(t, err)
	assert.Len(t, ops, 2)
	assert.Equal(t, 0, pq.Len())

	// Clear
	pq.Enqueue(PendingOp{RecordID: 3, Operation: OperationDelete})
	pq.Clear()
	assert.Equal(t, 0, pq.Len())
}

func TestFileStore_TransferState(t *testing.T) {
	s := newTestStore(t)
	ts := s.Transfers()

	tr := Transfer{
		ID:           1,
		Type:         TransferUpload,
		RecordID:     10,
		SessionID:    100,
		TotalChunks:  5,
		CompletedIdx: 2,
		Status:       TransferStatusActive,
	}

	// Save + Get
	err := ts.Save(tr)
	require.NoError(t, err)
	got, ok := ts.Get(1)
	assert.True(t, ok)
	assert.Equal(t, TransferUpload, got.Type)
	assert.Equal(t, int64(10), got.RecordID)

	// GetByRecord
	got, ok = ts.GetByRecord(10)
	assert.True(t, ok)
	assert.Equal(t, int64(1), got.ID)

	// ListActive
	active := ts.ListActive()
	assert.Len(t, active, 1)

	// Save completed
	tr.Status = TransferStatusCompleted
	ts.Save(tr)
	active = ts.ListActive()
	assert.Len(t, active, 0)
	assert.Len(t, ts.ListPending(), 0)

	// Save paused
	tr.Status = TransferStatusPaused
	ts.Save(tr)
	assert.Len(t, ts.ListPending(), 1)

	// Delete
	ts.Delete(1)
	_, ok = ts.Get(1)
	assert.False(t, ok)

	// Clear
	ts.Save(Transfer{ID: 2, Status: TransferStatusActive})
	ts.Save(Transfer{ID: 3, Status: TransferStatusActive})
	ts.Clear()
	active = ts.ListActive()
	assert.Len(t, active, 0)
}

func TestFileStore_AuthStore(t *testing.T) {
	s := newTestStore(t)
	auth := s.Auth()

	// Empty
	_, ok := auth.Get()
	assert.False(t, ok)

	// Set
	err := auth.Set(AuthData{
		AccessToken:  "token-123",
		RefreshToken: "refresh-456",
		UserID:       1,
		Email:        "test@example.com",
		DeviceID:     "dev1",
	})
	require.NoError(t, err)

	// Get
	data, ok := auth.Get()
	assert.True(t, ok)
	assert.Equal(t, "token-123", data.AccessToken)
	assert.Equal(t, "test@example.com", data.Email)

	// Clear
	auth.Clear()
	_, ok = auth.Get()
	assert.False(t, ok)
}

func TestFileStore_SyncState(t *testing.T) {
	s := newTestStore(t)
	sync := s.Sync()

	assert.Equal(t, int64(0), sync.Get().LastRevision)

	err := sync.SetLastRevision(42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), sync.Get().LastRevision)
}

func TestFileStore_FlushAndReload(t *testing.T) {
	dir := t.TempDir()

	// Создаём store и заполняем данными
	store1, err := NewFileStore(dir)
	require.NoError(t, err)

	store1.Records().Put(&models.Record{
		ID:      1,
		Name:    "persistent-record",
		Type:    models.RecordTypeText,
		Payload: models.TextPayload{Content: "hello"},
	})
	store1.Pending().Enqueue(PendingOp{
		RecordID:  1,
		Operation: OperationCreate,
	})
	store1.Auth().Set(AuthData{
		AccessToken: "saved-token",
		UserID:      1,
	})
	store1.Sync().SetLastRevision(10)

	require.NoError(t, store1.Flush())

	// Проверяем, что файл создан
	_, err = os.Stat(filepath.Join(dir, "cache.json"))
	require.NoError(t, err)

	// Загружаем в новый store
	store2, err := NewFileStore(dir)
	require.NoError(t, err)

	// Записи
	rec, ok := store2.Records().Get(1)
	require.True(t, ok)
	assert.Equal(t, "persistent-record", rec.Name)

	// Pending
	assert.Equal(t, 1, store2.Pending().Len())

	// Auth
	auth, ok := store2.Auth().Get()
	require.True(t, ok)
	assert.Equal(t, "saved-token", auth.AccessToken)

	// Sync
	assert.Equal(t, int64(10), store2.Sync().Get().LastRevision)
}
