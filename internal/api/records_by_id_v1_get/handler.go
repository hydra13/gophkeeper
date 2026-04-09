//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package records_by_id_v1_get

import (
	"net/http"
	"strconv"

	recordsCommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// GetRecordResponse описывает ответ с записью.
type GetRecordResponse struct {
	Record recordsCommon.RecordDTO `json:"record"`
}

// RecordService описывает получение записи.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
}

// Handler обрабатывает получение записи.
type Handler struct {
	service RecordService
}

// NewHandler создаёт обработчик получения записи.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle возвращает запись по идентификатору.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		responses.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		responses.Error(w, http.StatusBadRequest, "invalid record id")
		return
	}

	record, err := h.service.GetRecord(id)
	if err != nil {
		if recordsCommon.MapRecordError(w, err) {
			return
		}
		responses.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	if record.UserID != userID {
		responses.Error(w, http.StatusForbidden, "access denied")
		return
	}

	resp := recordToResponse(record)

	responses.JSON(w, http.StatusOK, resp)
}

func recordToResponse(r *models.Record) GetRecordResponse {
	return GetRecordResponse{Record: recordsCommon.RecordToDTO(*r)}
}
