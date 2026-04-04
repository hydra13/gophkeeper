package models

import (
	"errors"
	"fmt"
)

// UploadStatus описывает состояние upload-сессии.
type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusAborted   UploadStatus = "aborted"
)

// UploadSession хранит состояние загрузки бинарного payload.
type UploadSession struct {
	ID               int64
	RecordID         int64
	UserID           int64
	Status           UploadStatus
	TotalChunks      int64
	ReceivedChunks   int64
	ChunkSize        int64
	TotalSize        int64
	KeyVersion       int64
	ReceivedChunkSet map[int64]bool
}

// IsCompleted сообщает, что загрузка завершена.
func (u *UploadSession) IsCompleted() bool {
	return u.Status == UploadStatusCompleted
}

// IsAborted сообщает, что загрузка прервана.
func (u *UploadSession) IsAborted() bool {
	return u.Status == UploadStatusAborted
}

// IsResumable сообщает, что загрузку можно продолжить.
func (u *UploadSession) IsResumable() bool {
	return u.Status == UploadStatusPending && u.ReceivedChunks < u.TotalChunks
}

// CompleteChunk отмечает очередной принятый чанк.
func (u *UploadSession) CompleteChunk(chunkIndex int64) error {
	if u.Status == UploadStatusCompleted {
		return ErrUploadCompleted
	}
	if u.Status == UploadStatusAborted {
		return ErrUploadAborted
	}
	if u.Status != UploadStatusPending {
		return fmt.Errorf("upload session is in unexpected state: %s", u.Status)
	}
	if chunkIndex < 0 || chunkIndex >= u.TotalChunks {
		return ErrChunkOutOfRange
	}
	if chunkIndex != u.ReceivedChunks {
		return ErrChunkOutOfOrder
	}
	if u.ReceivedChunkSet != nil && u.ReceivedChunkSet[chunkIndex] {
		return ErrDuplicateChunk
	}
	if u.ReceivedChunkSet == nil {
		u.ReceivedChunkSet = make(map[int64]bool)
	}
	u.ReceivedChunkSet[chunkIndex] = true
	u.ReceivedChunks++
	if u.ReceivedChunks >= u.TotalChunks {
		u.Status = UploadStatusCompleted
	}
	return nil
}

// Abort переводит upload-сессию в состояние aborted.
func (u *UploadSession) Abort() error {
	if u.Status != UploadStatusPending {
		return errors.New("only pending upload can be aborted")
	}
	u.Status = UploadStatusAborted
	return nil
}

// MissingChunks возвращает индексы ещё не полученных чанков.
func (u *UploadSession) MissingChunks() []int64 {
	var missing []int64
	for i := int64(0); i < u.TotalChunks; i++ {
		if u.ReceivedChunkSet == nil || !u.ReceivedChunkSet[i] {
			missing = append(missing, i)
		}
	}
	return missing
}

// Chunk описывает отдельный фрагмент бинарных данных.
type Chunk struct {
	UploadID   int64
	ChunkIndex int64
	Data       []byte
}

// DownloadStatus описывает состояние download-сессии.
type DownloadStatus string

const (
	DownloadStatusActive    DownloadStatus = "active"
	DownloadStatusCompleted DownloadStatus = "completed"
	DownloadStatusAborted   DownloadStatus = "aborted"
)

// DownloadSession хранит состояние скачивания бинарного payload.
type DownloadSession struct {
	ID                int64
	RecordID          int64
	UserID            int64
	Status            DownloadStatus
	TotalChunks       int64
	ConfirmedChunks   int64
	ConfirmedChunkSet map[int64]bool
}

// IsCompleted сообщает, что скачивание завершено.
func (d *DownloadSession) IsCompleted() bool {
	return d.Status == DownloadStatusCompleted
}

// IsAborted сообщает, что скачивание прервано.
func (d *DownloadSession) IsAborted() bool {
	return d.Status == DownloadStatusAborted
}

// IsResumable сообщает, что скачивание можно продолжить.
func (d *DownloadSession) IsResumable() bool {
	return d.Status == DownloadStatusActive && d.ConfirmedChunks < d.TotalChunks
}

// ConfirmChunk подтверждает получение чанка клиентом.
func (d *DownloadSession) ConfirmChunk(chunkIndex int64) error {
	if d.Status == DownloadStatusCompleted {
		return ErrDownloadCompleted
	}
	if d.Status == DownloadStatusAborted {
		return ErrDownloadAborted
	}
	if d.Status != DownloadStatusActive {
		return ErrDownloadNotActive
	}
	if chunkIndex < 0 || chunkIndex >= d.TotalChunks {
		return ErrChunkOutOfRange
	}
	if chunkIndex != d.ConfirmedChunks {
		return ErrChunkOutOfOrder
	}
	if d.ConfirmedChunkSet != nil && d.ConfirmedChunkSet[chunkIndex] {
		return ErrChunkAlreadyConfirmed
	}
	if d.ConfirmedChunkSet == nil {
		d.ConfirmedChunkSet = make(map[int64]bool)
	}
	d.ConfirmedChunkSet[chunkIndex] = true
	d.ConfirmedChunks++
	if d.ConfirmedChunks >= d.TotalChunks {
		d.Status = DownloadStatusCompleted
	}
	return nil
}

// Abort переводит download-сессию в состояние aborted.
func (d *DownloadSession) Abort() error {
	if d.Status != DownloadStatusActive {
		return ErrDownloadNotActive
	}
	d.Status = DownloadStatusAborted
	return nil
}

// RemainingChunks возвращает индексы ещё не подтверждённых чанков.
func (d *DownloadSession) RemainingChunks() []int64 {
	var remaining []int64
	for i := int64(0); i < d.TotalChunks; i++ {
		if d.ConfirmedChunkSet == nil || !d.ConfirmedChunkSet[i] {
			remaining = append(remaining, i)
		}
	}
	return remaining
}

// UploadStatusResponse описывает ответ со статусом загрузки.
type UploadStatusResponse struct {
	UploadID       int64   `json:"upload_id"`
	RecordID       int64   `json:"record_id"`
	Status         string  `json:"status"`
	TotalChunks    int64   `json:"total_chunks"`
	ReceivedChunks int64   `json:"received_chunks"`
	MissingChunks  []int64 `json:"missing_chunks,omitempty"`
}

// DownloadResponse описывает ответ со статусом скачивания.
type DownloadResponse struct {
	DownloadID      int64   `json:"download_id"`
	RecordID        int64   `json:"record_id"`
	Status          string  `json:"status"`
	TotalChunks     int64   `json:"total_chunks"`
	ConfirmedChunks int64   `json:"confirmed_chunks"`
	RemainingChunks []int64 `json:"remaining_chunks,omitempty"`
}
