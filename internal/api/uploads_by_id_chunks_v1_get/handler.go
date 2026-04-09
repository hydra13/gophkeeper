//go:generate minimock -i .ChunkDownloader -o mocks -s _mock.go -g
package uploads_by_id_chunks_v1_get

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// ChunkDownloader описывает скачивание чанков.
type ChunkDownloader interface {
	DownloadChunk(uploadID, chunkIndex int64) (*models.ChunkDownloadResponse, error)
}

// Handler обрабатывает скачивание чанка.
type Handler struct {
	service ChunkDownloader
}

// NewHandler создаёт обработчик скачивания чанка.
func NewHandler(service ChunkDownloader) *Handler {
	return &Handler{service: service}
}

// ServeHTTP возвращает чанк загрузки.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responses.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	uploadID, chunkIndex, err := extractUploadIDAndChunkIndex(r.URL.Path)
	if err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid upload_id or chunk_index")
		return
	}
	if chunkIndex < 0 {
		responses.Error(w, http.StatusBadRequest, "chunk_index must be non-negative")
		return
	}

	resp, err := h.service.DownloadChunk(uploadID, chunkIndex)
	if err != nil {
		mapDownloadError(w, err)
		return
	}

	responses.JSON(w, http.StatusOK, resp)
}

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

func mapDownloadError(w http.ResponseWriter, err error) {
	switch {
	case isErr(err, "upload session not found"):
		responses.Error(w, http.StatusNotFound, err.Error())
	case isErr(err, "chunk") && isErr(err, "not found"):
		responses.Error(w, http.StatusNotFound, err.Error())
	case isErr(err, "download session not found"):
		responses.Error(w, http.StatusNotFound, err.Error())
	case isErr(err, "download session already completed"):
		responses.Error(w, http.StatusConflict, err.Error())
	case isErr(err, "download session is aborted"):
		responses.Error(w, http.StatusGone, err.Error())
	case isErr(err, "download session is not active"):
		responses.Error(w, http.StatusConflict, err.Error())
	case isErr(err, "upload session is not pending"):
		responses.Error(w, http.StatusConflict, err.Error())
	case isErr(err, "chunk index out of range"):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case isErr(err, "chunk already confirmed"):
		responses.Error(w, http.StatusConflict, err.Error())
	case isErr(err, "chunk order violated"):
		responses.Error(w, http.StatusConflict, err.Error())
	default:
		responses.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func isErr(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
