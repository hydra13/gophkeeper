//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsbyidv1delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/hydra13/gophkeeper/internal/models"
)

// DeleteRecordResponse — DTO ответа при удалении записи.
type DeleteRecordResponse struct {
	ID        int64  `json:"id"`
	Deleted   bool   `json:"deleted"`
	UpdatedAt string `json:"updated_at"`
}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
	DeleteRecord(id int64) error
}

// Handler — HTTP-обработчик для DELETE /api/v1/records/{id}.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

type userIDKey struct{}

// Handle обрабатывает запрос на удаление записи (soft delete).
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey{}).(int64)
	if !ok || userID <= 0 {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid record id")
		return
	}

	record, err := h.service.GetRecord(id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			writeJSONError(w, http.StatusNotFound, "record not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if record.UserID != userID {
		writeJSONError(w, http.StatusForbidden, "access denied")
		return
	}

	if record.IsDeleted() {
		writeJSONError(w, http.StatusBadRequest, "record is already deleted")
		return
	}

	if err := h.service.DeleteRecord(id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := DeleteRecordResponse{
		ID:        id,
		Deleted:   true,
		UpdatedAt: record.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// writeJSONError записывает JSON-ошибку в ответ.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
