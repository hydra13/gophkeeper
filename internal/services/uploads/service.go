//go:generate minimock -i .UploadRepo -o mocks -s _mock.go -g
package uploads

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hydra13/gophkeeper/internal/models"
)

type UploadRepo interface {
	CreateUploadSession(session *models.UploadSession) error
	GetUploadSession(id int64) (*models.UploadSession, error)
	GetCompletedUploadByRecordID(recordID int64) (*models.UploadSession, error)
	SaveChunk(chunk *models.Chunk) error
	GetChunks(uploadID int64) ([]models.Chunk, error)
}

// Service реализует бизнес-логику upload/download бинарных вложений.
type Service struct {
	repo UploadRepo
	mu   sync.RWMutex
	// downloadSessions хранит активные download-сессии in-memory.
	downloadSessions sync.Map
	downloadSeq      atomic.Int64
}

// NewService создаёт новый uploads service.
func NewService(repo UploadRepo) (*Service, error) {
	if repo == nil {
		return nil, errors.New("upload repository is required")
	}
	return &Service{repo: repo}, nil
}

// CreateSession создаёт новую upload-сессию.
func (s *Service) CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	if userID <= 0 {
		return 0, models.ErrInvalidUserID
	}
	if recordID <= 0 {
		return 0, errors.New("record_id is required")
	}
	if totalChunks <= 0 {
		return 0, errors.New("total_chunks must be positive")
	}
	if chunkSize <= 0 {
		return 0, errors.New("chunk_size must be positive")
	}
	if totalSize <= 0 {
		return 0, errors.New("total_size must be positive")
	}

	session := &models.UploadSession{
		UserID:           userID,
		RecordID:         recordID,
		Status:           models.UploadStatusPending,
		TotalChunks:      totalChunks,
		ChunkSize:        chunkSize,
		TotalSize:        totalSize,
		KeyVersion:       keyVersion,
		ReceivedChunkSet: make(map[int64]bool),
	}

	if err := s.repo.CreateUploadSession(session); err != nil {
		return 0, fmt.Errorf("create upload session: %w", err)
	}
	return session.ID, nil
}

// GetUploadStatus возвращает статус upload-сессии по ID.
func (s *Service) GetUploadStatus(uploadID int64) (*models.UploadStatusResponse, error) {
	session, err := s.repo.GetUploadSession(uploadID)
	if err != nil {
		return nil, fmt.Errorf("get upload session: %w", err)
	}
	return &models.UploadStatusResponse{
		UploadID:       session.ID,
		RecordID:       session.RecordID,
		Status:         string(session.Status),
		TotalChunks:    session.TotalChunks,
		ReceivedChunks: session.ReceivedChunks,
		MissingChunks:  session.MissingChunks(),
	}, nil
}

// UploadChunk загружает один чанк в указанную upload-сессию.
func (s *Service) UploadChunk(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error) {
	session, err := s.repo.GetUploadSession(uploadID)
	if err != nil {
		return 0, 0, false, nil, fmt.Errorf("get upload session: %w", err)
	}

	if err := session.CompleteChunk(chunkIndex); err != nil {
		return 0, 0, false, nil, err
	}

	chunk := &models.Chunk{
		UploadID:   uploadID,
		ChunkIndex: chunkIndex,
		Data:       data,
	}
	if err := s.repo.SaveChunk(chunk); err != nil {
		return 0, 0, false, nil, fmt.Errorf("save chunk: %w", err)
	}

	// Пере читаем сессию после сохранения чанка (репозиторий обновляет received_chunks и status)
	updated, err := s.repo.GetUploadSession(uploadID)
	if err != nil {
		return session.ReceivedChunks, session.TotalChunks, session.IsCompleted(), session.MissingChunks(), nil
	}

	return updated.ReceivedChunks, updated.TotalChunks, updated.IsCompleted(), updated.MissingChunks(), nil
}

// DownloadChunk возвращает данные чанка для скачивания по upload-сессии.
func (s *Service) DownloadChunk(uploadID, chunkIndex int64) (*models.ChunkDownloadResponse, error) {
	session, err := s.repo.GetUploadSession(uploadID)
	if err != nil {
		return nil, fmt.Errorf("get upload session: %w", err)
	}

	if session.Status != models.UploadStatusCompleted {
		return nil, models.ErrUploadNotPending
	}

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return nil, models.ErrChunkOutOfRange
	}

	chunks, err := s.repo.GetChunks(uploadID)
	if err != nil {
		return nil, fmt.Errorf("get chunks: %w", err)
	}

	var targetChunk *models.Chunk
	for i := range chunks {
		if chunks[i].ChunkIndex == chunkIndex {
			targetChunk = &chunks[i]
			break
		}
	}
	if targetChunk == nil {
		return nil, fmt.Errorf("chunk %d not found for upload %d", chunkIndex, uploadID)
	}

	return &models.ChunkDownloadResponse{
		UploadID:    uploadID,
		ChunkIndex:  chunkIndex,
		RecordID:    session.RecordID,
		Data:        targetChunk.Data,
		TotalChunks: session.TotalChunks,
	}, nil
}

// CreateDownloadSession создаёт сессию скачивания по recordID (для gRPC).
func (s *Service) CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error) {
	if recordID <= 0 {
		return nil, errors.New("record_id is required")
	}

	// Ищем завершённую upload-сессию для данной записи
	session, err := s.findCompletedUploadByRecordID(recordID)
	if err != nil {
		return nil, err
	}

	downloadID := s.downloadSeq.Add(1)
	download := &models.DownloadSession{
		ID:                downloadID,
		RecordID:          recordID,
		UserID:            userID,
		Status:            models.DownloadStatusActive,
		TotalChunks:       session.TotalChunks,
		ConfirmedChunks:   0,
		ConfirmedChunkSet: make(map[int64]bool),
	}

	s.downloadSessions.Store(downloadID, download)
	return download, nil
}

// DownloadChunkByID возвращает данные чанка по download-сессии (для gRPC).
func (s *Service) DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error) {
	download, err := s.getDownloadSession(downloadID)
	if err != nil {
		return nil, err
	}

	if download.Status != models.DownloadStatusActive {
		return nil, models.ErrDownloadNotActive
	}

	if chunkIndex < 0 || chunkIndex >= download.TotalChunks {
		return nil, models.ErrChunkOutOfRange
	}

	// Находим upload-сессию по recordID
	upload, err := s.findCompletedUploadByRecordID(download.RecordID)
	if err != nil {
		return nil, err
	}

	chunks, err := s.repo.GetChunks(upload.ID)
	if err != nil {
		return nil, fmt.Errorf("get chunks: %w", err)
	}

	for i := range chunks {
		if chunks[i].ChunkIndex == chunkIndex {
			return &chunks[i], nil
		}
	}
	return nil, fmt.Errorf("chunk %d not found", chunkIndex)
}

// ConfirmChunk подтверждает получение чанка клиентом в download-сессии.
func (s *Service) ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error) {
	download, err := s.getDownloadSession(downloadID)
	if err != nil {
		return 0, 0, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := download.ConfirmChunk(chunkIndex); err != nil {
		return 0, 0, "", err
	}

	return download.ConfirmedChunks, download.TotalChunks, download.Status, nil
}

// GetDownloadStatus возвращает статус download-сессии.
func (s *Service) GetDownloadStatus(downloadID int64) (*models.DownloadSession, error) {
	return s.getDownloadSession(downloadID)
}

// GetUploadSessionByID возвращает upload-сессию по ID (для gRPC).
func (s *Service) GetUploadSessionByID(uploadID int64) (*models.UploadSession, error) {
	session, err := s.repo.GetUploadSession(uploadID)
	if err != nil {
		return nil, fmt.Errorf("get upload session: %w", err)
	}
	return session, nil
}

func (s *Service) getDownloadSession(id int64) (*models.DownloadSession, error) {
	val, ok := s.downloadSessions.Load(id)
	if !ok {
		return nil, models.ErrDownloadNotFound
	}
	download, ok := val.(*models.DownloadSession)
	if !ok {
		return nil, models.ErrDownloadNotFound
	}
	return download, nil
}

// findCompletedUploadByRecordID ищет завершённую upload-сессию по recordID.
func (s *Service) findCompletedUploadByRecordID(recordID int64) (*models.UploadSession, error) {
	session, err := s.repo.GetCompletedUploadByRecordID(recordID)
	if err != nil {
		return nil, fmt.Errorf("find completed upload for record %d: %w", recordID, err)
	}
	return session, nil
}
