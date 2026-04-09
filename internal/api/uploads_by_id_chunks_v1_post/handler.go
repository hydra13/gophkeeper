//go:generate minimock -i .ChunkUploader -o mocks -s _mock.go -g
package uploads_by_id_chunks_v1_post

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/hydra13/gophkeeper/internal/api/responses"
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
		responses.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	uploadID, err := extractUploadID(r.URL.Path)
	if err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid upload_id")
		return
	}

	var req ChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.UploadID = uploadID

	if req.ChunkIndex < 0 {
		responses.Error(w, http.StatusBadRequest, "chunk_index must be non-negative")
		return
	}
	if len(req.Data) == 0 {
		responses.Error(w, http.StatusBadRequest, "data is required")
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

	responses.JSON(w, http.StatusOK, resp)
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
		responses.Error(w, http.StatusNotFound, err.Error())
	case isErr(err, "upload session already completed"):
		responses.Error(w, http.StatusConflict, err.Error())
	case isErr(err, "upload session is aborted"):
		responses.Error(w, http.StatusGone, err.Error())
	case isErr(err, "upload session is not pending"):
		responses.Error(w, http.StatusConflict, err.Error())
	case isErr(err, "chunk index out of range"):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case isErr(err, "chunk already received"):
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
