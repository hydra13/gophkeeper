//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsbyidv1delete

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

type DeleteRecordRequest struct {
	DeviceID string `json:"device_id"`
}

type DeleteRecordResponse struct{}

type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
	DeleteRecord(id int64, deviceID string) error
}

type Handler struct {
	service RecordService
}

func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(DeleteRecordResponse{}); err != nil {
				log.Printf("delete record response encode failed: %v", err)
			}
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
		if err := json.NewEncoder(w).Encode(DeleteRecordResponse{}); err != nil {
			log.Printf("delete already deleted response encode failed: %v", err)
		}
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("delete record final response encode failed: %v", err)
	}
}
