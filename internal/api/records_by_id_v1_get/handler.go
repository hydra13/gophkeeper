//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsbyidv1get

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// GetRecordResponse — DTO ответа при получении записи.
type GetRecordResponse struct {
	Record recordscommon.RecordDTO `json:"record"`
}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
}

// Handler — HTTP-обработчик для GET /api/v1/records/{id}.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle обрабатывает запрос на получение записи по ID.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		recordscommon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		recordscommon.WriteError(w, http.StatusBadRequest, "invalid record id")
		return
	}

	record, err := h.service.GetRecord(id)
	if err != nil {
		if recordscommon.MapRecordError(w, err) {
			return
		}
		recordscommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if record.UserID != userID {
		recordscommon.WriteError(w, http.StatusForbidden, "access denied")
		return
	}

	resp := recordToResponse(record)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// recordToResponse преобразует доменную модель в DTO ответа.
func recordToResponse(r *models.Record) GetRecordResponse {
	return GetRecordResponse{Record: recordscommon.RecordToDTO(*r)}
}
