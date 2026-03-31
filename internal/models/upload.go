package models

import (
	"errors"
	"fmt"
)

// UploadStatus определяет состояние upload-сессии.
type UploadStatus string

const (
	// UploadStatusPending — загрузка начата, но не завершена.
	UploadStatusPending UploadStatus = "pending"
	// UploadStatusCompleted — загрузка успешно завершена.
	UploadStatusCompleted UploadStatus = "completed"
	// UploadStatusAborted — загрузка прервана/отменена.
	UploadStatusAborted UploadStatus = "aborted"
)

// UploadSession — сессия загрузки бинарных данных с поддержкой chunk upload/download и resume.
type UploadSession struct {
	// ID — уникальный идентификатор upload-сессии (upload_id).
	ID int64
	// RecordID — ссылка на запись типа binary, к которой относится загрузка.
	RecordID int64
	// UserID — владелец загрузки.
	UserID int64
	// Status — текущее состояние загрузки.
	Status UploadStatus
	// TotalChunks — общее количество чанков.
	TotalChunks int64
	// ReceivedChunks — количество полученных чанков.
	ReceivedChunks int64
	// ChunkSize — размер одного чанка в байтах.
	ChunkSize int64
	// TotalSize — общий размер загружаемого файла в байтах.
	TotalSize int64
	// KeyVersion — версия ключа шифрования для загружаемых данных.
	KeyVersion int64
	// ReceivedChunkSet — множество индексов уже принятых чанков для защиты от дублей.
	ReceivedChunkSet map[int64]bool
}

// IsCompleted проверяет, завершена ли загрузка.
func (u *UploadSession) IsCompleted() bool {
	return u.Status == UploadStatusCompleted
}

// IsAborted проверяет, прервана ли загрузка.
func (u *UploadSession) IsAborted() bool {
	return u.Status == UploadStatusAborted
}

// IsResumable проверяет, можно ли возобновить загрузку.
func (u *UploadSession) IsResumable() bool {
	return u.Status == UploadStatusPending && u.ReceivedChunks < u.TotalChunks
}

// CompleteChunk регистрирует получение очередного чанка.
// Чанки должны поступать строго по порядку: следующий ожидаемый индекс равен ReceivedChunks.
// Возвращает ErrChunkOutOfOrder если индекс не совпадает с ожидаемым.
// Возвращает ErrDuplicateChunk если чанк с таким индексом уже был принят.
// Возвращает ErrChunkOutOfRange если индекс вне диапазона.
// Возвращает ErrUploadNotPending или ErrUploadAborted если статус не позволяет принять чанк.
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

// Abort прерывает загрузку. Допускается только в статусе pending.
func (u *UploadSession) Abort() error {
	if u.Status != UploadStatusPending {
		return errors.New("only pending upload can be aborted")
	}
	u.Status = UploadStatusAborted
	return nil
}

// MissingChunks возвращает слайс индексов ещё не принятых чанков.
// Позволяет клиенту узнать какие чанки нужно отправить для resume.
func (u *UploadSession) MissingChunks() []int64 {
	var missing []int64
	for i := int64(0); i < u.TotalChunks; i++ {
		if u.ReceivedChunkSet == nil || !u.ReceivedChunkSet[i] {
			missing = append(missing, i)
		}
	}
	return missing
}

// Chunk — отдельный чанк бинарных данных.
type Chunk struct {
	// UploadID — ссылка на upload-сессию.
	UploadID int64
	// ChunkIndex — порядковый номер чанка (начиная с 0).
	ChunkIndex int64
	// Data — бинарное содержимое чанка.
	Data []byte
}

// DownloadStatus определяет состояние download-сессии.
type DownloadStatus string

const (
	// DownloadStatusActive — скачивание активно, чанки отдаются клиенту.
	DownloadStatusActive DownloadStatus = "active"
	// DownloadStatusCompleted — все чанки подтверждены клиентом, скачивание завершено.
	DownloadStatusCompleted DownloadStatus = "completed"
	// DownloadStatusAborted — скачивание прервано.
	DownloadStatusAborted DownloadStatus = "aborted"
)

// DownloadSession — сессия скачивания бинарных данных с поддержкой resume.
// Клиент подтверждает каждый полученный чанк, что позволяет отслеживать прогресс
// и возобновлять скачивание после разрыва соединения.
type DownloadSession struct {
	// ID — уникальный идентификатор download-сессии.
	ID int64
	// RecordID — ссылка на запись типа binary.
	RecordID int64
	// UserID — владелец скачивания.
	UserID int64
	// Status — текущее состояние скачивания.
	Status DownloadStatus
	// TotalChunks — общее количество чанков для скачивания.
	TotalChunks int64
	// ConfirmedChunks — количество чанков, подтверждённых клиентом.
	ConfirmedChunks int64
	// ConfirmedChunkSet — множество индексов чанков, подтверждённых клиентом.
	ConfirmedChunkSet map[int64]bool
}

// IsCompleted проверяет, завершено ли скачивание (все чанки подтверждены).
func (d *DownloadSession) IsCompleted() bool {
	return d.Status == DownloadStatusCompleted
}

// IsAborted проверяет, прервано ли скачивание.
func (d *DownloadSession) IsAborted() bool {
	return d.Status == DownloadStatusAborted
}

// IsResumable проверяет, можно ли возобновить скачивание.
func (d *DownloadSession) IsResumable() bool {
	return d.Status == DownloadStatusActive && d.ConfirmedChunks < d.TotalChunks
}

// ConfirmChunk подтверждает получение чанка клиентом.
// Чанки подтверждаются строго по порядку: следующий ожидаемый индекс равен ConfirmedChunks.
// Возвращает ErrChunkOutOfOrder если индекс не совпадает с ожидаемым.
// Возвращает ErrDownloadCompleted если сессия уже завершена.
// Возвращает ErrDownloadAborted если сессия прервана.
// Возвращает ErrDownloadNotActive если сессия не в статусе active.
// Возвращает ErrChunkOutOfRange если индекс вне диапазона.
// Возвращает ErrChunkAlreadyConfirmed если чанк уже был подтверждён.
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

// Abort прерывает скачивание. Допускается только в статусе active.
func (d *DownloadSession) Abort() error {
	if d.Status != DownloadStatusActive {
		return ErrDownloadNotActive
	}
	d.Status = DownloadStatusAborted
	return nil
}

// RemainingChunks возвращает слайс индексов чанков, ещё не подтверждённых клиентом.
// Позволяет серверу узнать какие чанки нужно повторно отправить для resume.
func (d *DownloadSession) RemainingChunks() []int64 {
	var remaining []int64
	for i := int64(0); i < d.TotalChunks; i++ {
		if d.ConfirmedChunkSet == nil || !d.ConfirmedChunkSet[i] {
			remaining = append(remaining, i)
		}
	}
	return remaining
}

// UploadStatusResponse — DTO ответа статуса upload-сессии.
type UploadStatusResponse struct {
	// UploadID — идентификатор upload-сессии.
	UploadID int64 `json:"upload_id"`
	// RecordID — ссылка на запись.
	RecordID int64 `json:"record_id"`
	// Status — текущее состояние загрузки (pending, completed, aborted).
	Status string `json:"status"`
	// TotalChunks — общее количество чанков.
	TotalChunks int64 `json:"total_chunks"`
	// ReceivedChunks — количество принятых чанков.
	ReceivedChunks int64 `json:"received_chunks"`
	// MissingChunks — индексы ещё не принятых чанков (для resume загрузки).
	MissingChunks []int64 `json:"missing_chunks,omitempty"`
}

// DownloadResponse — DTO ответа для статуса download-сессии.
type DownloadResponse struct {
	// DownloadID — идентификатор download-сессии.
	DownloadID int64 `json:"download_id"`
	// RecordID — ссылка на запись.
	RecordID int64 `json:"record_id"`
	// Status — текущее состояние скачивания (active, completed, aborted).
	Status string `json:"status"`
	// TotalChunks — общее количество чанков.
	TotalChunks int64 `json:"total_chunks"`
	// ConfirmedChunks — количество подтверждённых клиентом чанков.
	ConfirmedChunks int64 `json:"confirmed_chunks"`
	// RemainingChunks — индексы чанков, ещё не подтверждённых клиентом (для resume скачивания).
	RemainingChunks []int64 `json:"remaining_chunks,omitempty"`
}
