//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsv1get

import (
	"encoding/json"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// ListRecordsResponse — DTO ответа списка записей.
type ListRecordsResponse struct {
	Records []recordscommon.RecordDTO `json:"records"`
}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	ListRecords(userID int64) ([]models.Record, error)
}

// Handler — HTTP-обработчик для GET /api/v1/records.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle обрабатывает запрос на получение списка записей.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		recordscommon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	records, err := h.service.ListRecords(userID)
	if err != nil {
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
	json.NewEncoder(w).Encode(resp)
}
