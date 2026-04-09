//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
//go:generate minimock -i .UploadService -o mocks -s _mock.go -g
package records_by_id_binary_v1_get

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordService описывает получение записи.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
}

// UploadService описывает скачивание бинарной записи.
type UploadService interface {
	CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error)
	DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error)
	ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error)
}

// Handler обрабатывает скачивание бинарной записи.
type Handler struct {
	records RecordService
	uploads UploadService
}

// NewHandler создаёт обработчик скачивания бинарной записи.
func NewHandler(records RecordService, uploads UploadService) *Handler {
	return &Handler{
		records: records,
		uploads: uploads,
	}
}

// Handle скачивает бинарную запись по частям.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		responses.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		responses.Error(w, http.StatusBadRequest, "invalid record id")
		return
	}

	record, err := h.records.GetRecord(id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			responses.Error(w, http.StatusNotFound, "record not found")
		default:
			responses.Error(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	if record.UserID != userID {
		responses.Error(w, http.StatusForbidden, "access denied")
		return
	}
	if record.Type != models.RecordTypeBinary {
		responses.Error(w, http.StatusBadRequest, "record is not binary")
		return
	}

	download, err := h.uploads.CreateDownloadSession(userID, record.ID)
	if err != nil {
		responses.Error(w, http.StatusInternalServerError, "create download session: "+err.Error())
		return
	}

	var payload bytes.Buffer
	for chunkIndex := int64(0); chunkIndex < download.TotalChunks; chunkIndex++ {
		chunk, err := h.uploads.DownloadChunkByID(download.ID, chunkIndex)
		if err != nil {
			responses.Error(w, http.StatusInternalServerError, fmt.Sprintf("download chunk %d: %v", chunkIndex, err))
			return
		}
		payload.Write(chunk.Data)

		if _, _, _, err := h.uploads.ConfirmChunk(download.ID, chunkIndex); err != nil {
			responses.Error(w, http.StatusInternalServerError, fmt.Sprintf("confirm chunk %d: %v", chunkIndex, err))
			return
		}
	}

	filename := record.Name
	if filename == "" {
		filename = fmt.Sprintf("record-%d.bin", record.ID)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload.Bytes())
}
