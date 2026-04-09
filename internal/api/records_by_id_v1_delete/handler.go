//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package records_by_id_v1_delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	recordsCommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// DeleteRecordRequest описывает запрос на удаление записи.
type DeleteRecordRequest struct {
	DeviceID string `json:"device_id"`
}

// DeleteRecordResponse описывает ответ на удаление записи.
type DeleteRecordResponse struct{}

// RecordService описывает удаление записи.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
	DeleteRecord(id int64, deviceID string) error
}

// Handler обрабатывает удаление записи.
type Handler struct {
	service RecordService
}

// NewHandler создаёт обработчик удаления записи.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle удаляет запись.
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

	var req DeleteRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DeviceID == "" {
		responses.Error(w, http.StatusBadRequest, "device_id is required")
		return
	}

	record, err := h.service.GetRecord(id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			responses.JSON(w, http.StatusOK, DeleteRecordResponse{})
			return
		}
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

	if record.IsDeleted() {
		responses.JSON(w, http.StatusOK, DeleteRecordResponse{})
		return
	}

	if err := h.service.DeleteRecord(id, req.DeviceID); err != nil {
		if recordsCommon.MapRecordError(w, err) {
			return
		}
		responses.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	responses.JSON(w, http.StatusOK, DeleteRecordResponse{})
}
