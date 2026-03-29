// Package sync_pull_v1_post implements the HTTP endpoint for pulling sync changes.
//
// POST /api/v1/sync/pull
//
// Client sends its cursor (last known revision) and receives pending changes
// that occurred after that cursor, along with the updated cursor value.
package sync_pull_v1_post

import (
	"encoding/json"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/models"
)

// Request — DTO для запроса pull-синхронизации.
type Request struct {
	// UserID — идентификатор пользователя (из JWT-токена, проставляется middleware).
	UserID int64 `json:"user_id"`
	// DeviceID — устройство, запрашивающее изменения.
	DeviceID string `json:"device_id"`
	// Cursor — ревизия, начиная с которой клиент хочет получить изменения (0 = с начала).
	Cursor int64 `json:"cursor"`
	// Limit — максимальное количество изменений в ответе.
	Limit int64 `json:"limit"`
}

// ChangedRecord — DTO одной изменённой записи в ответе pull.
type ChangedRecord struct {
	// RecordID — идентификатор записи.
	RecordID int64 `json:"record_id"`
	// Revision — ревизия записи.
	Revision int64 `json:"revision"`
	// Deleted — признак soft delete.
	Deleted bool `json:"deleted"`
}

// Response — DTO для ответа pull-синхронизации.
type Response struct {
	// Changes — список изменений после курсора.
	Changes []ChangedRecord `json:"changes"`
	// NextCursor — новый курсор для следующего запроса.
	NextCursor int64 `json:"next_cursor"`
	// HasMore — признак наличия ещё изменений.
	HasMore bool `json:"has_more"`
}

// SyncPuller — интерфейс сервиса pull-синхронизации.
type SyncPuller interface {
	// Pull возвращает изменения для пользователя начиная с указанного курсора.
	Pull(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error)
}

// Handler обрабатывает POST /api/v1/sync/pull.
type Handler struct {
	service SyncPuller
}

// NewHandler создаёт новый обработчик pull-синхронизации.
func NewHandler(service SyncPuller) *Handler {
	return &Handler{service: service}
}

// ServeHTTP обрабатывает HTTP-запрос pull-синхронизации.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}

	revs, err := h.service.Pull(req.UserID, req.DeviceID, req.Cursor, req.Limit)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	changes := make([]ChangedRecord, 0, len(revs))
	var nextCursor int64
	for _, rev := range revs {
		changes = append(changes, ChangedRecord{
			RecordID: rev.RecordID,
			Revision: rev.Revision,
		})
		nextCursor = rev.Revision
	}

	hasMore := len(revs) == int(req.Limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Changes:    changes,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
