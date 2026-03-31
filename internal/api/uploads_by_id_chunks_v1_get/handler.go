// Package uploads_by_id_chunks_v1_get реализует HTTP-ручку скачивания чанка.
//
// GET /api/v1/uploads/{id}/chunks/{index}
//
//go:generate minimock -i .ChunkDownloader -o mocks -s _mock.go -g
package uploads_by_id_chunks_v1_get

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/hydra13/gophkeeper/internal/models"
)

// ChunkDownloader — интерфейс сервиса скачивания чанка.
type ChunkDownloader interface {
	// DownloadChunk возвращает данные чанка и статус download-сессии.
	DownloadChunk(uploadID, chunkIndex int64) (*models.ChunkDownloadResponse, error)
}

// Handler обрабатывает GET /api/v1/uploads/{id}/chunks/{index}.
type Handler struct {
	service ChunkDownloader
}

// NewHandler создаёт новый обработчик скачивания чанка.
func NewHandler(service ChunkDownloader) *Handler {
	return &Handler{service: service}
}

// ServeHTTP возвращает один чанк и текущее состояние download-сессии.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID, chunkIndex, err := extractUploadIDAndChunkIndex(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid upload_id or chunk_index", http.StatusBadRequest)
		return
	}
	if chunkIndex < 0 {
		http.Error(w, "chunk_index must be non-negative", http.StatusBadRequest)
		return
	}

	resp, err := h.service.DownloadChunk(uploadID, chunkIndex)
	if err != nil {
		mapDownloadError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// extractUploadIDAndChunkIndex извлекает upload_id и chunk_index из URL path /api/v1/uploads/{id}/chunks/{index}.
func extractUploadIDAndChunkIndex(path string) (int64, int64, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	var uploadID, chunkIndex int64
	var uploadErr, chunkErr error
	var hasUpload, hasChunk bool
	for i, p := range parts {
		switch p {
		case "uploads":
			if i+1 < len(parts) {
				uploadID, uploadErr = strconv.ParseInt(parts[i+1], 10, 64)
				hasUpload = true
			}
		case "chunks":
			if i+1 < len(parts) {
				chunkIndex, chunkErr = strconv.ParseInt(parts[i+1], 10, 64)
				hasChunk = true
			}
		}
	}
	if uploadErr != nil || chunkErr != nil || !hasUpload || !hasChunk {
		if uploadErr != nil {
			return 0, 0, uploadErr
		}
		if chunkErr != nil {
			return 0, 0, chunkErr
		}
		return 0, 0, strconv.ErrRange
	}
	return uploadID, chunkIndex, nil
}

// mapDownloadError мапит ошибки домена на HTTP-статусы.
func mapDownloadError(w http.ResponseWriter, err error) {
	switch {
	case isErr(err, "upload session not found"):
		http.Error(w, err.Error(), http.StatusNotFound)
	case isErr(err, "chunk") && isErr(err, "not found"):
		http.Error(w, err.Error(), http.StatusNotFound)
	case isErr(err, "download session not found"):
		http.Error(w, err.Error(), http.StatusNotFound)
	case isErr(err, "download session already completed"):
		http.Error(w, err.Error(), http.StatusConflict)
	case isErr(err, "download session is aborted"):
		http.Error(w, err.Error(), http.StatusGone)
	case isErr(err, "download session is not active"):
		http.Error(w, err.Error(), http.StatusConflict)
	case isErr(err, "upload session is not pending"):
		http.Error(w, err.Error(), http.StatusConflict)
	case isErr(err, "chunk index out of range"):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case isErr(err, "chunk already confirmed"):
		http.Error(w, err.Error(), http.StatusConflict)
	case isErr(err, "chunk order violated"):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func isErr(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
