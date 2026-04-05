//go:generate minimock -i .ChunkUploader -o mocks -s _mock.go -g
package uploads_by_id_chunks_v1_post

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// ChunkRequest описывает запрос на загрузку чанка.
type ChunkRequest struct {
	UploadID   int64  `json:"upload_id"`
	ChunkIndex int64  `json:"chunk_index"`
	Data       []byte `json:"data"`
}

// ChunkResponse описывает ответ на загрузку чанка.
type ChunkResponse struct {
	UploadID       int64   `json:"upload_id"`
	ReceivedChunks int64   `json:"received_chunks"`
	TotalChunks    int64   `json:"total_chunks"`
	Completed      bool    `json:"completed"`
	MissingChunks  []int64 `json:"missing_chunks,omitempty"`
}

// ChunkUploader описывает загрузку чанков.
type ChunkUploader interface {
	UploadChunk(uploadID, chunkIndex int64, data []byte) (receivedChunks, totalChunks int64, completed bool, missingChunks []int64, err error)
}

// Handler обрабатывает загрузку чанка.
type Handler struct {
	service ChunkUploader
}

// NewHandler создаёт обработчик загрузки чанка.
func NewHandler(service ChunkUploader) *Handler {
	return &Handler{service: service}
}

// ServeHTTP принимает чанк загрузки.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID, err := extractUploadID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid upload_id", http.StatusBadRequest)
		return
	}

	var req ChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.UploadID = uploadID

	if req.ChunkIndex < 0 {
		http.Error(w, "chunk_index must be non-negative", http.StatusBadRequest)
		return
	}
	if len(req.Data) == 0 {
		http.Error(w, "data is required", http.StatusBadRequest)
		return
	}

	received, total, completed, missing, err := h.service.UploadChunk(req.UploadID, req.ChunkIndex, req.Data)
	if err != nil {
		mapChunkError(w, err)
		return
	}

	resp := ChunkResponse{
		UploadID:       uploadID,
		ReceivedChunks: received,
		TotalChunks:    total,
		Completed:      completed,
	}
	if !completed && len(missing) > 0 {
		resp.MissingChunks = missing
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("upload chunk response encode failed: %v", err)
	}
}

func extractUploadID(path string) (int64, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "uploads" && i+1 < len(parts) {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return 0, strconv.ErrRange
}

func mapChunkError(w http.ResponseWriter, err error) {
	switch {
	case isErr(err, "upload session not found"):
		http.Error(w, err.Error(), http.StatusNotFound)
	case isErr(err, "upload session already completed"):
		http.Error(w, err.Error(), http.StatusConflict)
	case isErr(err, "upload session is aborted"):
		http.Error(w, err.Error(), http.StatusGone)
	case isErr(err, "upload session is not pending"):
		http.Error(w, err.Error(), http.StatusConflict)
	case isErr(err, "chunk index out of range"):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case isErr(err, "chunk already received"):
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
