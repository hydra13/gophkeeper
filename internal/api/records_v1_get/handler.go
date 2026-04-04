//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsv1get

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

type ListRecordsResponse struct {
	Records []recordscommon.RecordDTO `json:"records"`
}

type RecordService interface {
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
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

	recordType := models.RecordType(r.URL.Query().Get("type"))
	if recordType != "" && !models.ValidRecordTypes[recordType] {
		recordscommon.WriteError(w, http.StatusBadRequest, "invalid record type filter")
		return
	}

	includeDeleted, _ := strconv.ParseBool(r.URL.Query().Get("include_deleted"))

	records, err := h.service.ListRecords(userID, recordType, includeDeleted)
	if err != nil {
		if recordscommon.MapRecordError(w, err) {
			return
		}
		recordscommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := ListRecordsResponse{
		Records: make([]recordscommon.RecordDTO, 0, len(records)),
	}
	for _, rec := range records {
		resp.Records = append(resp.Records, recordscommon.RecordToDTO(rec))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("list records response encode failed: %v", err)
	}
}
