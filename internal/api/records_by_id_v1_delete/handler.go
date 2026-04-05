//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package records_by_id_v1_delete

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	recordsCommon "github.com/hydra13/gophkeeper/internal/api/records_common"
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
		recordsCommon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		recordsCommon.WriteError(w, http.StatusBadRequest, "invalid record id")
		return
	}

	var req DeleteRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		recordsCommon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DeviceID == "" {
		recordsCommon.WriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	record, err := h.service.GetRecord(id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(DeleteRecordResponse{}); err != nil {
				log.Printf("delete record response encode failed: %v", err)
			}
			return
		}
		if recordsCommon.MapRecordError(w, err) {
			return
		}
		recordsCommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if record.UserID != userID {
		recordsCommon.WriteError(w, http.StatusForbidden, "access denied")
		return
	}

	if record.IsDeleted() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(DeleteRecordResponse{}); err != nil {
			log.Printf("delete already deleted response encode failed: %v", err)
		}
		return
	}

	if err := h.service.DeleteRecord(id, req.DeviceID); err != nil {
		if recordsCommon.MapRecordError(w, err) {
			return
		}
		recordsCommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := DeleteRecordResponse{}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("delete record final response encode failed: %v", err)
	}
}
