package models

// ChunkDownloadResponse — DTO ответа при скачивании чанка.
type ChunkDownloadResponse struct {
	// UploadID — идентификатор upload-сессии.
	UploadID int64 `json:"upload_id"`
	// DownloadID — идентификатор download-сессии.
	DownloadID int64 `json:"download_id"`
	// RecordID — ссылка на запись.
	RecordID int64 `json:"record_id"`
	// ChunkIndex — порядковый номер чанка.
	ChunkIndex int64 `json:"chunk_index"`
	// Data — бинарное содержимое чанка (base64 в JSON).
	Data []byte `json:"data"`
	// TotalChunks — общее количество чанков.
	TotalChunks int64 `json:"total_chunks"`
	// ConfirmedChunks — количество подтверждённых клиентом чанков.
	ConfirmedChunks int64 `json:"confirmed_chunks"`
	// RemainingChunks — индексы чанков, ещё не подтверждённых клиентом (для resume).
	RemainingChunks []int64 `json:"remaining_chunks,omitempty"`
	// Completed — признак завершения скачивания.
	Completed bool `json:"completed"`
}
