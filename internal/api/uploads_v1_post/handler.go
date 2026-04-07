//go:generate minimock -i .UploadCreator -o mocks -s _mock.go -g
package uploads_v1_post

import (
	"encoding/json"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/api/responses"
)

// Request описывает запрос на создание загрузки.
type Request struct {
	UserID      int64 `json:"user_id"`
	RecordID    int64 `json:"record_id"`
	TotalChunks int64 `json:"total_chunks"`
	ChunkSize   int64 `json:"chunk_size"`
	TotalSize   int64 `json:"total_size"`
	KeyVersion  int64 `json:"key_version"`
}

// Response описывает ответ на создание загрузки.
type Response struct {
	UploadID int64  `json:"upload_id"`
	Status   string `json:"status"`
}

// UploadCreator описывает создание сессии загрузки.
type UploadCreator interface {
	CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
}

// Handler обрабатывает создание загрузки.
type Handler struct {
	service UploadCreator
}

// NewHandler создаёт обработчик создания загрузки.
func NewHandler(service UploadCreator) *Handler {
	return &Handler{service: service}
}

// ServeHTTP создаёт сессию загрузки.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responses.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID <= 0 {
		responses.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	if req.RecordID <= 0 {
		responses.Error(w, http.StatusBadRequest, "invalid record_id")
		return
	}
	if req.TotalChunks <= 0 {
		responses.Error(w, http.StatusBadRequest, "total_chunks must be positive")
		return
	}
	if req.ChunkSize <= 0 {
		responses.Error(w, http.StatusBadRequest, "chunk_size must be positive")
		return
	}
	if req.TotalSize <= 0 {
		responses.Error(w, http.StatusBadRequest, "total_size must be positive")
		return
	}
	if req.KeyVersion <= 0 {
		responses.Error(w, http.StatusBadRequest, "key_version must be positive")
		return
	}

	uploadID, err := h.service.CreateSession(
		req.UserID,
		req.RecordID,
		req.TotalChunks,
		req.ChunkSize,
		req.TotalSize,
		req.KeyVersion,
	)
	if err != nil {
		responses.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.JSON(w, http.StatusCreated, Response{
		UploadID: uploadID,
		Status:   "pending",
	})
}
