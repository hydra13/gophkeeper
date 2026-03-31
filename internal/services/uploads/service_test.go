package uploads

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
)

// ---------------------------------------------------------------------------
// In-memory UploadRepository mock
// ---------------------------------------------------------------------------

type memUploadRepo struct {
	sessions map[int64]*models.UploadSession
	chunks   map[int64][]models.Chunk // key = uploadID
	nextID   int64
	// optional overrides for error injection
	createSessionErr          error
	getSessionErr             error
	getCompletedByRecordIDErr error
	updateSessionErr          error
	saveChunkErr              error
	getChunksErr              error
}

func newMemUploadRepo() *memUploadRepo {
	return &memUploadRepo{
		sessions: make(map[int64]*models.UploadSession),
		chunks:   make(map[int64][]models.Chunk),
		nextID:   1,
	}
}

func (m *memUploadRepo) CreateUploadSession(session *models.UploadSession) error {
	if m.createSessionErr != nil {
		return m.createSessionErr
	}
	id := m.nextID
	m.nextID++
	session.ID = id
	clone := *session
	clone.ReceivedChunkSet = make(map[int64]bool)
	for k, v := range session.ReceivedChunkSet {
		clone.ReceivedChunkSet[k] = v
	}
	m.sessions[id] = &clone
	return nil
}

func (m *memUploadRepo) GetUploadSession(id int64) (*models.UploadSession, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	s, ok := m.sessions[id]
	if !ok {
		return nil, models.ErrUploadNotFound
	}
	clone := *s
	clone.ReceivedChunkSet = make(map[int64]bool)
	for k, v := range s.ReceivedChunkSet {
		clone.ReceivedChunkSet[k] = v
	}
	return &clone, nil
}

func (m *memUploadRepo) GetCompletedUploadByRecordID(recordID int64) (*models.UploadSession, error) {
	if m.getCompletedByRecordIDErr != nil {
		return nil, m.getCompletedByRecordIDErr
	}
	for _, s := range m.sessions {
		if s.RecordID == recordID && s.Status == models.UploadStatusCompleted {
			clone := *s
			clone.ReceivedChunkSet = make(map[int64]bool)
			for k, v := range s.ReceivedChunkSet {
				clone.ReceivedChunkSet[k] = v
			}
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("no completed upload for record %d: %w", recordID, models.ErrUploadNotFound)
}

func (m *memUploadRepo) UpdateUploadSession(session *models.UploadSession) error {
	if m.updateSessionErr != nil {
		return m.updateSessionErr
	}
	clone := *session
	clone.ReceivedChunkSet = make(map[int64]bool)
	for k, v := range session.ReceivedChunkSet {
		clone.ReceivedChunkSet[k] = v
	}
	m.sessions[session.ID] = &clone
	return nil
}

func (m *memUploadRepo) SaveChunk(chunk *models.Chunk) error {
	if m.saveChunkErr != nil {
		return m.saveChunkErr
	}
	m.chunks[chunk.UploadID] = append(m.chunks[chunk.UploadID], *chunk)
	// simulate repo updating session: increment received chunks, update status
	s, ok := m.sessions[chunk.UploadID]
	if ok {
		if s.ReceivedChunkSet == nil {
			s.ReceivedChunkSet = make(map[int64]bool)
		}
		s.ReceivedChunkSet[chunk.ChunkIndex] = true
		s.ReceivedChunks++
		if s.ReceivedChunks >= s.TotalChunks {
			s.Status = models.UploadStatusCompleted
		}
	}
	return nil
}

func (m *memUploadRepo) GetChunks(uploadID int64) ([]models.Chunk, error) {
	if m.getChunksErr != nil {
		return nil, m.getChunksErr
	}
	return m.chunks[uploadID], nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(t *testing.T) (*Service, *memUploadRepo) {
	t.Helper()
	repo := newMemUploadRepo()
	svc, err := NewService(repo)
	require.NoError(t, err)
	return svc, repo
}

// pendingSession возвращает готовую pending-сессию с заданными параметрами.
func pendingSession(userID, recordID, totalChunks int64) *models.UploadSession {
	return &models.UploadSession{
		UserID:           userID,
		RecordID:         recordID,
		Status:           models.UploadStatusPending,
		TotalChunks:      totalChunks,
		ReceivedChunks:   0,
		ChunkSize:        1024,
		TotalSize:        totalChunks * 1024,
		KeyVersion:       1,
		ReceivedChunkSet: make(map[int64]bool),
	}
}

// completedSession создаёт completed upload-сессию с указанным числом чанков.
func completedSession(userID, recordID, totalChunks int64) *models.UploadSession {
	s := pendingSession(userID, recordID, totalChunks)
	s.ReceivedChunks = totalChunks
	s.Status = models.UploadStatusCompleted
	for i := int64(0); i < totalChunks; i++ {
		s.ReceivedChunkSet[i] = true
	}
	return s
}

// seedSession добавляет сессию в repo и возвращает её ID.
func seedSession(t *testing.T, repo *memUploadRepo, session *models.UploadSession) int64 {
	t.Helper()
	err := repo.CreateUploadSession(session)
	require.NoError(t, err)
	return session.ID
}

// seedChunks добавляет чанки в repo для заданного uploadID.
func seedChunks(t *testing.T, repo *memUploadRepo, uploadID int64, indices []int64, data []byte) {
	t.Helper()
	for _, idx := range indices {
		chunk := &models.Chunk{
			UploadID:   uploadID,
			ChunkIndex: idx,
			Data:       data,
		}
		err := repo.SaveChunk(chunk)
		require.NoError(t, err)
	}
}

// createDownloadSession helper — создаёт download-сессию через service.
func createDownloadSession(t *testing.T, svc *Service, repo *memUploadRepo, userID, recordID, totalChunks int64) int64 {
	t.Helper()
	// Seed completed upload session
	sess := completedSession(userID, recordID, totalChunks)
	seedSession(t, repo, sess)
	// Seed chunks
	indices := make([]int64, totalChunks)
	for i := int64(0); i < totalChunks; i++ {
		indices[i] = i
	}
	seedChunks(t, repo, sess.ID, indices, []byte("data"))

	download, err := svc.CreateDownloadSession(userID, recordID)
	require.NoError(t, err)
	return download.ID
}

// containsChunk проверяет наличие чанка с нужным индексом в слайсе.
func containsChunk(chunks []models.Chunk, chunkIndex int64) bool {
	for _, c := range chunks {
		if c.ChunkIndex == chunkIndex {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// NewService
// ---------------------------------------------------------------------------

func TestNewService_NilRepo(t *testing.T) {
	svc, err := NewService(nil)
	require.Error(t, err)
	require.Nil(t, svc)
	require.Contains(t, err.Error(), "upload repository is required")
}

func TestNewService_Success(t *testing.T) {
	repo := newMemUploadRepo()
	svc, err := NewService(repo)
	require.NoError(t, err)
	require.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// CreateSession
// ---------------------------------------------------------------------------

func TestCreateSession_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)

	uploadID, err := svc.CreateSession(1, 10, 4, 1024, 4096, 1)
	require.NoError(t, err)
	require.Greater(t, uploadID, int64(0))

	session, err := repo.GetUploadSession(uploadID)
	require.NoError(t, err)
	require.Equal(t, int64(1), session.UserID)
	require.Equal(t, int64(10), session.RecordID)
	require.Equal(t, models.UploadStatusPending, session.Status)
	require.Equal(t, int64(4), session.TotalChunks)
	require.Equal(t, int64(1024), session.ChunkSize)
	require.Equal(t, int64(4096), session.TotalSize)
	require.Equal(t, int64(1), session.KeyVersion)
}

func TestCreateSession_НеверныеПараметры(t *testing.T) {
	svc, _ := newTestService(t)

	tests := []struct {
		name        string
		userID      int64
		recordID    int64
		totalChunks int64
		chunkSize   int64
		totalSize   int64
		keyVersion  int64
		wantErr     string
	}{
		{"userID <= 0", 0, 10, 4, 1024, 4096, 1, "invalid user id"},
		{"recordID <= 0", 1, 0, 4, 1024, 4096, 1, "record_id is required"},
		{"totalChunks <= 0", 1, 10, 0, 1024, 4096, 1, "total_chunks must be positive"},
		{"chunkSize <= 0", 1, 10, 4, 0, 4096, 1, "chunk_size must be positive"},
		{"totalSize <= 0", 1, 10, 4, 1024, 0, 1, "total_size must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateSession(tt.userID, tt.recordID, tt.totalChunks, tt.chunkSize, tt.totalSize, tt.keyVersion)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCreateSession_ОшибкаРепозитория(t *testing.T) {
	repo := newMemUploadRepo()
	repo.createSessionErr = errors.New("db error")
	svc, err := NewService(repo)
	require.NoError(t, err)

	_, err = svc.CreateSession(1, 10, 4, 1024, 4096, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "create upload session")
	require.Contains(t, err.Error(), "db error")
}

// ---------------------------------------------------------------------------
// GetUploadStatus
// ---------------------------------------------------------------------------

func TestGetUploadStatus_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 4)
	id := seedSession(t, repo, session)

	resp, err := svc.GetUploadStatus(id)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, id, resp.UploadID)
	require.Equal(t, int64(10), resp.RecordID)
	require.Equal(t, "pending", resp.Status)
	require.Equal(t, int64(4), resp.TotalChunks)
	require.Equal(t, int64(0), resp.ReceivedChunks)
	require.Equal(t, []int64{0, 1, 2, 3}, resp.MissingChunks)
}

func TestGetUploadStatus_ОшибкаРепозитория(t *testing.T) {
	svc, repo := newTestService(t)
	repo.getSessionErr = errors.New("connection lost")

	_, err := svc.GetUploadStatus(999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get upload session")
	require.Contains(t, err.Error(), "connection lost")
}

// ---------------------------------------------------------------------------
// UploadChunk
// ---------------------------------------------------------------------------

func TestUploadChunk_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 3)
	id := seedSession(t, repo, session)

	received, total, completed, missing, err := svc.UploadChunk(id, 0, []byte("chunk-data"))
	require.NoError(t, err)
	require.Equal(t, int64(1), received)
	require.Equal(t, int64(3), total)
	require.False(t, completed)
	require.Equal(t, []int64{1, 2}, missing)
}

func TestUploadChunk_ПоследнийЧанкЗавершаетСессию(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 2)
	id := seedSession(t, repo, session)

	// Первый чанк
	_, _, _, _, err := svc.UploadChunk(id, 0, []byte("a"))
	require.NoError(t, err)

	// Второй (последний) чанк
	received, total, completed, missing, err := svc.UploadChunk(id, 1, []byte("b"))
	require.NoError(t, err)
	require.Equal(t, int64(2), received)
	require.Equal(t, int64(2), total)
	require.True(t, completed)
	require.Empty(t, missing)
}

func TestUploadChunk_СессияНеНайдена(t *testing.T) {
	svc, _ := newTestService(t)

	_, _, _, _, err := svc.UploadChunk(999, 0, []byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "get upload session")
}

func TestUploadChunk_СессияУжеЗавершена(t *testing.T) {
	svc, repo := newTestService(t)

	session := completedSession(1, 10, 2)
	id := seedSession(t, repo, session)

	_, _, _, _, err := svc.UploadChunk(id, 0, []byte("data"))
	require.ErrorIs(t, err, models.ErrUploadCompleted)
}

func TestUploadChunk_СессияПрервана(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 2)
	session.Status = models.UploadStatusAborted
	id := seedSession(t, repo, session)

	_, _, _, _, err := svc.UploadChunk(id, 0, []byte("data"))
	require.ErrorIs(t, err, models.ErrUploadAborted)
}

func TestUploadChunk_ИндексВнеДиапазона(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 3)
	id := seedSession(t, repo, session)

	_, _, _, _, err := svc.UploadChunk(id, 5, []byte("data"))
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestUploadChunk_ДубльЧанка(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 3)
	id := seedSession(t, repo, session)

	_, _, _, _, err := svc.UploadChunk(id, 0, []byte("data"))
	require.NoError(t, err)

	// Повторная загрузка того же чанка — теперь ошибка порядка, т.к. ожидается индекс 1
	_, _, _, _, err = svc.UploadChunk(id, 0, []byte("data-again"))
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

func TestUploadChunk_ОшибкаСохраненияЧанка(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 3)
	id := seedSession(t, repo, session)

	repo.saveChunkErr = errors.New("disk full")

	_, _, _, _, err := svc.UploadChunk(id, 0, []byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "save chunk")
	require.Contains(t, err.Error(), "disk full")
}

// ---------------------------------------------------------------------------
// DownloadChunk
// ---------------------------------------------------------------------------

func TestDownloadChunk_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)

	session := completedSession(1, 10, 3)
	id := seedSession(t, repo, session)
	seedChunks(t, repo, id, []int64{0, 1, 2}, []byte("hello"))

	resp, err := svc.DownloadChunk(id, 0)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, id, resp.UploadID)
	require.Equal(t, int64(0), resp.ChunkIndex)
	require.Equal(t, int64(10), resp.RecordID)
	require.Equal(t, []byte("hello"), resp.Data)
	require.Equal(t, int64(3), resp.TotalChunks)
}

func TestDownloadChunk_ИндексВнеДиапазона(t *testing.T) {
	svc, repo := newTestService(t)

	session := completedSession(1, 10, 3)
	id := seedSession(t, repo, session)

	_, err := svc.DownloadChunk(id, 5)
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestDownloadChunk_ОтрицательныйИндекс(t *testing.T) {
	svc, repo := newTestService(t)

	session := completedSession(1, 10, 3)
	id := seedSession(t, repo, session)

	_, err := svc.DownloadChunk(id, -1)
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestDownloadChunk_СессияНеНайдена(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.DownloadChunk(999, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get upload session")
}

func TestDownloadChunk_ЧанкНеНайден(t *testing.T) {
	svc, repo := newTestService(t)

	session := completedSession(1, 10, 3)
	id := seedSession(t, repo, session)
	// Не добавляем чанк с индексом 1

	_, err := svc.DownloadChunk(id, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chunk 1 not found")
}

func TestDownloadChunk_ОшибкаПолученияЧанков(t *testing.T) {
	svc, repo := newTestService(t)

	session := completedSession(1, 10, 3)
	id := seedSession(t, repo, session)
	repo.getChunksErr = errors.New("read error")

	_, err := svc.DownloadChunk(id, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get chunks")
}

func TestDownloadChunk_НезавершённаяUploadСессия(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 3)
	id := seedSession(t, repo, session)

	_, err := svc.DownloadChunk(id, 0)
	require.ErrorIs(t, err, models.ErrUploadNotPending)
}

// ---------------------------------------------------------------------------
// CreateDownloadSession
// ---------------------------------------------------------------------------

func TestCreateDownloadSession_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)

	// Создаём завершённую upload-сессию
	session := completedSession(1, 10, 4)
	seedSession(t, repo, session)
	seedChunks(t, repo, session.ID, []int64{0, 1, 2, 3}, []byte("x"))

	download, err := svc.CreateDownloadSession(1, 10)
	require.NoError(t, err)
	require.NotNil(t, download)
	require.Greater(t, download.ID, int64(0))
	require.Equal(t, int64(10), download.RecordID)
	require.Equal(t, int64(1), download.UserID)
	require.Equal(t, models.DownloadStatusActive, download.Status)
	require.Equal(t, int64(4), download.TotalChunks)
	require.Equal(t, int64(0), download.ConfirmedChunks)
}

func TestCreateDownloadSession_НеверныйRecordID(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.CreateDownloadSession(1, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "record_id is required")
}

func TestCreateDownloadSession_НетЗавершённойUploadСессии(t *testing.T) {
	svc, repo := newTestService(t)

	// Создаём pending сессию (не completed)
	session := pendingSession(1, 10, 4)
	seedSession(t, repo, session)

	_, err := svc.CreateDownloadSession(1, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "find completed upload")
}

func TestCreateDownloadSession_ОшибкаРепозитория(t *testing.T) {
	svc, repo := newTestService(t)
	repo.getCompletedByRecordIDErr = errors.New("db down")

	_, err := svc.CreateDownloadSession(1, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "find completed upload")
}

// ---------------------------------------------------------------------------
// DownloadChunkByID
// ---------------------------------------------------------------------------

func TestDownloadChunkByID_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	chunk, err := svc.DownloadChunkByID(downloadID, 0)
	require.NoError(t, err)
	require.NotNil(t, chunk)
	require.Equal(t, int64(0), chunk.ChunkIndex)
	require.Equal(t, []byte("data"), chunk.Data)
}

func TestDownloadChunkByID_СессияНеНайдена(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.DownloadChunkByID(999, 0)
	require.ErrorIs(t, err, models.ErrDownloadNotFound)
}

func TestDownloadChunkByID_СессияНеАктивна(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	// Подтверждаем все чанки, чтобы завершить download
	for i := int64(0); i < 3; i++ {
		_, _, _, err := svc.ConfirmChunk(downloadID, i)
		require.NoError(t, err)
	}

	// Теперь сессия completed
	_, err := svc.DownloadChunkByID(downloadID, 0)
	require.ErrorIs(t, err, models.ErrDownloadNotActive)
}

func TestDownloadChunkByID_ИндексВнеДиапазона(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	_, err := svc.DownloadChunkByID(downloadID, 5)
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestDownloadChunkByID_ОтрицательныйИндекс(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	_, err := svc.DownloadChunkByID(downloadID, -1)
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestDownloadChunkByID_ЧанкНеНайден(t *testing.T) {
	svc, repo := newTestService(t)

	// Создаём completed upload без чанка с индексом 2
	session := completedSession(1, 10, 3)
	seedSession(t, repo, session)
	seedChunks(t, repo, session.ID, []int64{0, 1}, []byte("x")) // нет чанка 2

	download, err := svc.CreateDownloadSession(1, 10)
	require.NoError(t, err)

	_, err = svc.DownloadChunkByID(download.ID, 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "chunk 2 not found")
}

// ---------------------------------------------------------------------------
// ConfirmChunk
// ---------------------------------------------------------------------------

func TestConfirmChunk_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	confirmed, total, status, err := svc.ConfirmChunk(downloadID, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), confirmed)
	require.Equal(t, int64(3), total)
	require.Equal(t, models.DownloadStatusActive, status)
}

func TestConfirmChunk_ПоследнийЧанкЗавершаетСессию(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 2)

	confirmed, total, status, err := svc.ConfirmChunk(downloadID, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), confirmed)
	require.Equal(t, models.DownloadStatusActive, status)

	confirmed, total, status, err = svc.ConfirmChunk(downloadID, 1)
	require.NoError(t, err)
	require.Equal(t, int64(2), confirmed)
	require.Equal(t, int64(2), total)
	require.Equal(t, models.DownloadStatusCompleted, status)
}

func TestConfirmChunk_СессияНеНайдена(t *testing.T) {
	svc, _ := newTestService(t)

	_, _, _, err := svc.ConfirmChunk(999, 0)
	require.ErrorIs(t, err, models.ErrDownloadNotFound)
}

func TestConfirmChunk_СессияУжеЗавершена(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 1)

	// Подтверждаем единственный чанк — сессия completed
	_, _, _, err := svc.ConfirmChunk(downloadID, 0)
	require.NoError(t, err)

	// Повторное подтверждение
	_, _, _, err = svc.ConfirmChunk(downloadID, 0)
	require.ErrorIs(t, err, models.ErrDownloadCompleted)
}

func TestConfirmChunk_ИндексВнеДиапазона(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	_, _, _, err := svc.ConfirmChunk(downloadID, 10)
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestConfirmChunk_ЧанкУжеПодтверждён(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	_, _, _, err := svc.ConfirmChunk(downloadID, 0)
	require.NoError(t, err)

	// Повторно подтверждаем чанк 0 — уже подтверждён, но т.к. ConfirmedChunks=1, ожидается 1, а не 0 — ErrChunkOutOfOrder
	_, _, _, err = svc.ConfirmChunk(downloadID, 0)
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

func TestConfirmChunk_НарушениеПорядка(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	// Пробуем подтвердить чанк 2, пропустив 0 и 1
	_, _, _, err := svc.ConfirmChunk(downloadID, 2)
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

// ---------------------------------------------------------------------------
// GetDownloadStatus
// ---------------------------------------------------------------------------

func TestGetDownloadStatus_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)
	downloadID := createDownloadSession(t, svc, repo, 1, 10, 3)

	download, err := svc.GetDownloadStatus(downloadID)
	require.NoError(t, err)
	require.NotNil(t, download)
	require.Equal(t, downloadID, download.ID)
	require.Equal(t, int64(10), download.RecordID)
	require.Equal(t, models.DownloadStatusActive, download.Status)
	require.Equal(t, int64(3), download.TotalChunks)
}

func TestGetDownloadStatus_НеНайдена(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetDownloadStatus(999)
	require.ErrorIs(t, err, models.ErrDownloadNotFound)
}

// ---------------------------------------------------------------------------
// GetUploadSessionByID
// ---------------------------------------------------------------------------

func TestGetUploadSessionByID_HappyPath(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 4)
	id := seedSession(t, repo, session)

	got, err := svc.GetUploadSessionByID(id)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, id, got.ID)
	require.Equal(t, int64(10), got.RecordID)
}

func TestGetUploadSessionByID_НеНайдена(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetUploadSessionByID(999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get upload session")
}

// ---------------------------------------------------------------------------
// Edge cases / integration-style scenarios
// ---------------------------------------------------------------------------

func TestПолныйЦиклUploadИDownload(t *testing.T) {
	svc, _ := newTestService(t)

	// 1. Создаём upload-сессию
	uploadID, err := svc.CreateSession(1, 42, 3, 256, 768, 1)
	require.NoError(t, err)

	// 2. Загружаем все чанки
	for i := int64(0); i < 3; i++ {
		data := []byte(fmt.Sprintf("chunk-%d", i))
		received, total, completed, missing, uploadErr := svc.UploadChunk(uploadID, i, data)
		require.NoError(t, uploadErr)
		require.Equal(t, i+1, received)
		require.Equal(t, int64(3), total)
		if i < 2 {
			require.False(t, completed)
			require.Len(t, missing, int(2-i))
		}
	}
	// Проверяем, что upload завершён
	status, err := svc.GetUploadStatus(uploadID)
	require.NoError(t, err)
	require.Equal(t, "completed", status.Status)

	// 3. Скачиваем чанк по upload ID
	downloadResp, err := svc.DownloadChunk(uploadID, 1)
	require.NoError(t, err)
	require.Equal(t, []byte("chunk-1"), downloadResp.Data)

	// 4. Создаём download-сессию
	download, err := svc.CreateDownloadSession(1, 42)
	require.NoError(t, err)
	require.Equal(t, models.DownloadStatusActive, download.Status)

	// 5. Скачиваем чанк по download ID
	chunk, err := svc.DownloadChunkByID(download.ID, 2)
	require.NoError(t, err)
	require.Equal(t, []byte("chunk-2"), chunk.Data)

	// 6. Подтверждаем все чанки
	for i := int64(0); i < 3; i++ {
		confirmed, total, _, confirmErr := svc.ConfirmChunk(download.ID, i)
		require.NoError(t, confirmErr)
		require.Equal(t, i+1, confirmed)
		require.Equal(t, int64(3), total)
	}

	// 7. Проверяем, что download завершён
	finalStatus, err := svc.GetDownloadStatus(download.ID)
	require.NoError(t, err)
	require.Equal(t, models.DownloadStatusCompleted, finalStatus.Status)
}

func TestUploadChunk_НарушениеПорядка(t *testing.T) {
	svc, repo := newTestService(t)

	session := pendingSession(1, 10, 3)
	id := seedSession(t, repo, session)

	// Пробуем загрузить чанк 2, пропустив 0 и 1
	_, _, _, _, err := svc.UploadChunk(id, 2, []byte("data"))
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

func TestResumeUploadПослеОбрыва(t *testing.T) {
	svc, _ := newTestService(t)

	uploadID, err := svc.CreateSession(1, 5, 4, 512, 2048, 1)
	require.NoError(t, err)

	// Загружаем чанки 0 и 1 по порядку
	_, _, _, _, err = svc.UploadChunk(uploadID, 0, []byte("a"))
	require.NoError(t, err)
	_, _, _, _, err = svc.UploadChunk(uploadID, 1, []byte("b"))
	require.NoError(t, err)

	// Проверяем статус — missing: 2, 3
	status, err := svc.GetUploadStatus(uploadID)
	require.NoError(t, err)
	require.Equal(t, "pending", status.Status)
	require.Equal(t, int64(2), status.ReceivedChunks)
	require.Equal(t, []int64{2, 3}, status.MissingChunks)

	// Догружаем оставшиеся чанки по порядку
	_, _, _, _, err = svc.UploadChunk(uploadID, 2, []byte("c"))
	require.NoError(t, err)

	_, _, completed, missing, err := svc.UploadChunk(uploadID, 3, []byte("d"))
	require.NoError(t, err)
	require.True(t, completed)
	require.Empty(t, missing)
}

// ---------------------------------------------------------------------------
// Unused import prevention (responses used in return types)
// ---------------------------------------------------------------------------

var (
	_ *models.UploadStatusResponse  = nil
	_ *models.ChunkDownloadResponse = nil
)
