//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsv1get

import (
	"encoding/json"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordItem — DTO записи в списке.
type RecordItem struct {
	ID             int64  `json:"id"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	Metadata       string `json:"metadata,omitempty"`
	Revision       int64  `json:"revision"`
	Deleted        bool   `json:"deleted"`
	DeviceID       string `json:"device_id"`
	KeyVersion     int64  `json:"key_version"`
	PayloadVersion int64  `json:"payload_version,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ListRecordsResponse — DTO ответа списка записей.
type ListRecordsResponse struct {
	Records []RecordItem `json:"records"`
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

type userIDKey struct{}

// Handle обрабатывает запрос на получение списка записей.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey{}).(int64)
	if !ok || userID <= 0 {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	records, err := h.service.ListRecords(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := ListRecordsResponse{
		Records: make([]RecordItem, 0, len(records)),
	}
	for _, rec := range records {
		resp.Records = append(resp.Records, recordToItem(rec))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// recordToItem преобразует доменную модель в DTO.
func recordToItem(r models.Record) RecordItem {
	item := RecordItem{
		ID:             r.ID,
		Type:           string(r.Type),
		Name:           r.Name,
		Metadata:       r.Metadata,
		Revision:       r.Revision,
		Deleted:        r.IsDeleted(),
		DeviceID:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
		CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      r.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	return item
}

// writeJSONError записывает JSON-ошибку в ответ.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
