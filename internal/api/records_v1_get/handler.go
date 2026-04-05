//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package records_v1_get

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	recordsCommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// ListRecordsResponse описывает ответ со списком записей.
type ListRecordsResponse struct {
	Records []recordsCommon.RecordDTO `json:"records"`
}

// RecordService описывает получение списка записей.
type RecordService interface {
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
}

// Handler обрабатывает список записей.
type Handler struct {
	service RecordService
}

// NewHandler создаёт обработчик списка записей.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle возвращает список записей.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		recordsCommon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	recordType := models.RecordType(r.URL.Query().Get("type"))
	if recordType != "" && !models.ValidRecordTypes[recordType] {
		recordsCommon.WriteError(w, http.StatusBadRequest, "invalid record type filter")
		return
	}

	includeDeleted, _ := strconv.ParseBool(r.URL.Query().Get("include_deleted"))

	records, err := h.service.ListRecords(userID, recordType, includeDeleted)
	if err != nil {
		if recordsCommon.MapRecordError(w, err) {
			return
		}
		recordsCommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := ListRecordsResponse{
		Records: make([]recordsCommon.RecordDTO, 0, len(records)),
	}
	for _, rec := range records {
		resp.Records = append(resp.Records, recordsCommon.RecordToDTO(rec))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("list records response encode failed: %v", err)
	}
}
