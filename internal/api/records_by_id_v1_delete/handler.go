// Package recordsbyidv1delete реализует HTTP-ручку мягкого удаления записи.
//
// DELETE /api/v1/records/{id}
//
//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsbyidv1delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// DeleteRecordRequest — DTO запроса на удаление записи.
type DeleteRecordRequest struct {
	// DeviceID — идентификатор устройства, выполняющего удаление.
	DeviceID string `json:"device_id"`
}

// DeleteRecordResponse — DTO ответа при удалении записи.
type DeleteRecordResponse struct{}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
	DeleteRecord(id int64, deviceID string) error
}

// Handler — HTTP-обработчик для DELETE /api/v1/records/{id}.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle помечает запись как удалённую.
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

	var req DeleteRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		recordscommon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DeviceID == "" {
		recordscommon.WriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	record, err := h.service.GetRecord(id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			// Delete is idempotent: record doesn't exist or already soft-deleted.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(DeleteRecordResponse{})
			return
		}
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

	if record.IsDeleted() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeleteRecordResponse{})
		return
	}

	if err := h.service.DeleteRecord(id, req.DeviceID); err != nil {
		if recordscommon.MapRecordError(w, err) {
			return
		}
		recordscommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := DeleteRecordResponse{}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
