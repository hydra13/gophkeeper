// Package sync_push_v1_post implements the HTTP endpoint for pushing local changes to the server.
//
// POST /api/v1/sync/push
//
// Client sends its local changes (pending operations). Server validates them
// against current server state and either accepts or returns conflicts.
package sync_push_v1_post

import (
	"encoding/json"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/models"
)

// PendingChange — DTO одной локальной операции, отправляемой на сервер.
type PendingChange struct {
	// RecordID — идентификатор записи (0 для новой записи).
	RecordID int64 `json:"record_id"`
	// Revision — ревизия, на которой основано изменение (для conflict detection).
	Revision int64 `json:"revision"`
	// Operation — тип операции: "create", "update", "delete".
	Operation string `json:"operation"`
	// DeviceID — устройство, инициировавшее изменение.
	DeviceID string `json:"device_id"`
}

// Request — DTO для запроса push-синхронизации.
type Request struct {
	// UserID — идентификатор пользователя (из JWT-токена).
	UserID int64 `json:"user_id"`
	// Changes — список локальных операций для отправки на сервер.
	Changes []PendingChange `json:"changes"`
}

// ConflictInfo — DTO информации о конфликте для одной записи.
type ConflictInfo struct {
	// RecordID — запись с конфликтом.
	RecordID int64 `json:"record_id"`
	// LocalRevision — ревизия клиента.
	LocalRevision int64 `json:"local_revision"`
	// ServerRevision — текущая ревизия на сервере.
	ServerRevision int64 `json:"server_revision"`
}

// Response — DTO для ответа push-синхронизации.
type Response struct {
	// Accepted — количество принятых изменений.
	Accepted int `json:"accepted"`
	// Conflicts — список конфликтов (если есть).
	Conflicts []ConflictInfo `json:"conflicts,omitempty"`
}

// SyncPusher — интерфейс сервиса push-синхронизации.
type SyncPusher interface {
	// Push отправляет локальные изменения на сервер.
	// Возвращает принятые ревизии и список конфликтов.
	Push(userID int64, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
}

// Handler обрабатывает POST /api/v1/sync/push.
type Handler struct {
	service SyncPusher
}

// NewHandler создаёт новый обработчик push-синхронизации.
func NewHandler(service SyncPusher) *Handler {
	return &Handler{service: service}
}

// ServeHTTP обрабатывает HTTP-запрос push-синхронизации.
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
	if len(req.Changes) == 0 {
		http.Error(w, "changes are required", http.StatusBadRequest)
		return
	}

	_, conflicts, err := h.service.Push(req.UserID, req.Changes)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Accepted: len(req.Changes) - len(conflicts),
	}
	for _, c := range conflicts {
		resp.Conflicts = append(resp.Conflicts, ConflictInfo{
			RecordID:       c.RecordID,
			LocalRevision:  c.LocalRevision,
			ServerRevision: c.ServerRevision,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
