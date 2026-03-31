// Package uploads_v1_post implements the HTTP endpoint for creating an upload session.
//
// POST /api/v1/uploads
//
// Client initiates a new upload session by specifying record metadata and chunk parameters.
// Server creates an upload session in "pending" status and returns the session ID.
package uploads_v1_post

import (
	"encoding/json"
	"net/http"
)

// Request — DTO для создания upload-сессии.
type Request struct {
	// UserID — идентификатор пользователя (из JWT-токена).
	UserID int64 `json:"user_id"`
	// RecordID — ссылка на запись типа binary, к которой относится загрузка.
	RecordID int64 `json:"record_id"`
	// TotalChunks — общее количество чанков.
	TotalChunks int64 `json:"total_chunks"`
	// ChunkSize — размер одного чанка в байтах.
	ChunkSize int64 `json:"chunk_size"`
	// TotalSize — общий размер загружаемого файла в байтах.
	TotalSize int64 `json:"total_size"`
	// KeyVersion — версия ключа шифрования.
	KeyVersion int64 `json:"key_version"`
}

// Response — DTO ответа при создании upload-сессии.
type Response struct {
	// UploadID — идентификатор созданной upload-сессии.
	UploadID int64 `json:"upload_id"`
	// Status — текущий статус сессии ("pending").
	Status string `json:"status"`
}

// UploadCreator — интерфейс сервиса создания upload-сессии.
type UploadCreator interface {
	// CreateSession создаёт новую upload-сессию.
	CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
}

// Handler обрабатывает POST /api/v1/uploads.
type Handler struct {
	service UploadCreator
}

// NewHandler создаёт новый обработчик создания upload-сессии.
func NewHandler(service UploadCreator) *Handler {
	return &Handler{service: service}
}

// ServeHTTP обрабатывает HTTP-запрос создания upload-сессии.
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
	json.NewEncoder(w).Encode(Response{
		UploadID: uploadID,
		Status:   "pending",
	})
}
