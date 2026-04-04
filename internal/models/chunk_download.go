package models

// ChunkDownloadResponse описывает результат чтения чанка для скачивания.
type ChunkDownloadResponse struct {
	UploadID        int64   `json:"upload_id"`
	DownloadID      int64   `json:"download_id"`
	RecordID        int64   `json:"record_id"`
	ChunkIndex      int64   `json:"chunk_index"`
	Data            []byte  `json:"data"`
	TotalChunks     int64   `json:"total_chunks"`
	ConfirmedChunks int64   `json:"confirmed_chunks"`
	RemainingChunks []int64 `json:"remaining_chunks,omitempty"`
	Completed       bool    `json:"completed"`
}
