// Package sync_push_v1_post реализует HTTP-ручку отправки локальных изменений на сервер.
//
// POST /api/v1/sync/push
//
//go:generate minimock -i .SyncPusher -o mocks -s _mock.go -g
package sync_push_v1_post

import (
	"encoding/json"
	"net/http"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/models"
)

// PendingChange — DTO одной локальной операции, отправляемой на сервер.
type PendingChange struct {
	// Record — полные данные записи (создание или обновление).
	Record recordscommon.RecordDTO `json:"record"`
	// Deleted — признак удаления записи.
	Deleted bool `json:"deleted"`
	// BaseRevision — ревизия, на основе которой было сделано изменение.
	BaseRevision int64 `json:"base_revision"`
}

// Request — DTO для запроса push-синхронизации.
type Request struct {
	// UserID — идентификатор пользователя (из JWT-токена).
	UserID int64 `json:"user_id"`
	// DeviceID — устройство, выполняющее push.
	DeviceID string `json:"device_id"`
	// Changes — список локальных операций для отправки на сервер.
	Changes []PendingChange `json:"changes"`
}

// RecordRevisionDTO — DTO изменённой записи (ревизии).
type RecordRevisionDTO struct {
	ID       int64  `json:"id"`
	RecordID int64  `json:"record_id"`
	UserID   int64  `json:"user_id"`
	Revision int64  `json:"revision"`
	DeviceID string `json:"device_id"`
}

// SyncConflictDTO — DTO конфликта синхронизации.
type SyncConflictDTO struct {
	ID             int64                    `json:"id"`
	UserID         int64                    `json:"user_id"`
	RecordID       int64                    `json:"record_id"`
	LocalRevision  int64                    `json:"local_revision"`
	ServerRevision int64                    `json:"server_revision"`
	Resolved       bool                     `json:"resolved"`
	Resolution     string                   `json:"resolution"`
	LocalRecord    *recordscommon.RecordDTO `json:"local_record,omitempty"`
	ServerRecord   *recordscommon.RecordDTO `json:"server_record,omitempty"`
}

// Response — DTO для ответа push-синхронизации.
type Response struct {
	// Accepted — список принятых изменений с обновлёнными ревизиями.
	Accepted []RecordRevisionDTO `json:"accepted"`
	// Conflicts — список конфликтов (если есть).
	Conflicts []SyncConflictDTO `json:"conflicts,omitempty"`
}

// SyncPusher — интерфейс сервиса push-синхронизации.
type SyncPusher interface {
	// Push отправляет локальные изменения на сервер.
	// Возвращает принятые ревизии и список конфликтов.
	Push(userID int64, deviceID string, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
}

// Handler обрабатывает POST /api/v1/sync/push.
type Handler struct {
	service SyncPusher
}

// NewHandler создаёт новый обработчик push-синхронизации.
func NewHandler(service SyncPusher) *Handler {
	return &Handler{service: service}
}

// ServeHTTP принимает локальные изменения и возвращает принятые ревизии и конфликты.
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
	if len(req.Changes) == 0 {
		http.Error(w, "changes are required", http.StatusBadRequest)
		return
	}

	accepted, conflicts, err := h.service.Push(req.UserID, req.DeviceID, req.Changes)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := Response{}
	for _, rev := range accepted {
		resp.Accepted = append(resp.Accepted, RecordRevisionDTO{
			ID:       rev.ID,
			RecordID: rev.RecordID,
			UserID:   rev.UserID,
			Revision: rev.Revision,
			DeviceID: rev.DeviceID,
		})
	}
	for _, conflict := range conflicts {
		dto := SyncConflictDTO{
			ID:             conflict.ID,
			UserID:         conflict.UserID,
			RecordID:       conflict.RecordID,
			LocalRevision:  conflict.LocalRevision,
			ServerRevision: conflict.ServerRevision,
			Resolved:       conflict.Resolved,
			Resolution:     conflict.Resolution,
		}
		if conflict.LocalRecord != nil {
			localDTO := recordscommon.RecordToDTO(*conflict.LocalRecord)
			dto.LocalRecord = &localDTO
		}
		if conflict.ServerRecord != nil {
			serverDTO := recordscommon.RecordToDTO(*conflict.ServerRecord)
			dto.ServerRecord = &serverDTO
		}
		resp.Conflicts = append(resp.Conflicts, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
