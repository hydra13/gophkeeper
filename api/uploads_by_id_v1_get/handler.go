// Package uploads_by_id_v1_get implements the HTTP endpoint for getting upload/download session status
// and initiating chunk download with resume support.
//
// GET /api/v1/uploads/{id}
//
// Returns upload session status including missing chunks for resume.
// When used for download, returns download session with remaining chunks.
package uploads_by_id_v1_get

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

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

// DownloadResponse — DTO ответа для скачивания бинарных данных.
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

// UploadStatusGetter — интерфейс сервиса получения статуса upload/download.
type UploadStatusGetter interface {
	// GetUploadStatus возвращает статус upload-сессии по ID.
	GetUploadStatus(uploadID int64) (*UploadStatusResponse, error)
}

// Handler обрабатывает GET /api/v1/uploads/{id}.
type Handler struct {
	service UploadStatusGetter
}

// NewHandler создаёт новый обработчик получения статуса загрузки.
func NewHandler(service UploadStatusGetter) *Handler {
	return &Handler{service: service}
}

// ServeHTTP обрабатывает HTTP-запрос статуса загрузки/скачивания.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID, err := extractUploadID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid upload_id", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetUploadStatus(uploadID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// extractUploadID извлекает upload_id из URL path /api/v1/uploads/{id}.
func extractUploadID(path string) (int64, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "uploads" && i+1 < len(parts) {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return 0, strconv.ErrRange
}
