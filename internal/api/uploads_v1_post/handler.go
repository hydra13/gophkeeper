//go:generate minimock -i .UploadCreator -o mocks -s _mock.go -g
package uploads_v1_post

import (
	"encoding/json"
	"log"
	"net/http"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	if req.RecordID <= 0 {
		http.Error(w, "invalid record_id", http.StatusBadRequest)
		return
	}
	if req.TotalChunks <= 0 {
		http.Error(w, "total_chunks must be positive", http.StatusBadRequest)
		return
	}
	if req.ChunkSize <= 0 {
		http.Error(w, "chunk_size must be positive", http.StatusBadRequest)
		return
	}
	if req.TotalSize <= 0 {
		http.Error(w, "total_size must be positive", http.StatusBadRequest)
		return
	}
	if req.KeyVersion <= 0 {
		http.Error(w, "key_version must be positive", http.StatusBadRequest)
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(Response{
		UploadID: uploadID,
		Status:   "pending",
	}); err != nil {
		log.Printf("create upload response encode failed: %v", err)
	}
}
