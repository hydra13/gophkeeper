// Package sync_pull_v1_post реализует HTTP-ручку получения изменений синхронизации.
//
// POST /api/v1/sync/pull
package sync_pull_v1_post

import (
	"encoding/json"
	"net/http"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/models"
)

// Request — DTO для запроса pull-синхронизации.
type Request struct {
	// UserID — идентификатор пользователя (из JWT-токена, проставляется middleware).
	UserID int64 `json:"user_id"`
	// DeviceID — устройство, запрашивающее изменения.
	DeviceID string `json:"device_id"`
	// SinceRevision — ревизия, начиная с которой клиент хочет получить изменения (0 = с начала).
	SinceRevision int64 `json:"since_revision"`
	// Limit — максимальное количество изменений в ответе.
	Limit int64 `json:"limit"`
}

// RecordRevisionDTO — DTO одной изменённой записи в ответе pull.
type RecordRevisionDTO struct {
	// ID — идентификатор ревизии.
	ID int64 `json:"id"`
	// RecordID — идентификатор записи.
	RecordID int64 `json:"record_id"`
	// UserID — владелец записи.
	UserID int64 `json:"user_id"`
	// Revision — ревизия записи.
	Revision int64 `json:"revision"`
	// DeviceID — устройство, инициировавшее изменение.
	DeviceID string `json:"device_id"`
	// Deleted — признак soft delete.
	Deleted bool `json:"deleted"`
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

// Response — DTO для ответа pull-синхронизации.
type Response struct {
	// Changes — список изменений после курсора.
	Changes []RecordRevisionDTO `json:"changes"`
	// Records — полные данные изменённых записей.
	Records []recordscommon.RecordDTO `json:"records"`
	// Conflicts — список конфликтов.
	Conflicts []SyncConflictDTO `json:"conflicts,omitempty"`
	// NextRevision — новый курсор для следующего запроса.
	NextRevision int64 `json:"next_revision"`
	// HasMore — признак наличия ещё изменений.
	HasMore bool `json:"has_more"`
}

// SyncPuller — интерфейс сервиса pull-синхронизации.
type SyncPuller interface {
	// Pull возвращает изменения для пользователя начиная с указанного курсора.
	Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
}

// Handler обрабатывает POST /api/v1/sync/pull.
type Handler struct {
	service SyncPuller
}

// NewHandler создаёт новый обработчик pull-синхронизации.
func NewHandler(service SyncPuller) *Handler {
	return &Handler{service: service}
}

// ServeHTTP возвращает изменения, записи и конфликты после указанной ревизии.
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

	revs, records, conflicts, err := h.service.Pull(req.UserID, req.DeviceID, req.SinceRevision, req.Limit)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	recordByID := make(map[int64]models.Record, len(records))
	recordsDTO := make([]recordscommon.RecordDTO, 0, len(records))
	for _, rec := range records {
		recordByID[rec.ID] = rec
		recordsDTO = append(recordsDTO, recordscommon.RecordToDTO(rec))
	}

	changes := make([]RecordRevisionDTO, 0, len(revs))
	var nextRevision int64
	for _, rev := range revs {
		deleted := false
		if rec, ok := recordByID[rev.RecordID]; ok && rec.DeletedAt != nil {
			deleted = true
		}
		changes = append(changes, RecordRevisionDTO{
			ID:       rev.ID,
			RecordID: rev.RecordID,
			UserID:   rev.UserID,
			Revision: rev.Revision,
			DeviceID: rev.DeviceID,
			Deleted:  deleted,
		})
		nextRevision = rev.Revision
	}

	hasMore := len(revs) == int(req.Limit)

	resp := Response{
		Changes:      changes,
		Records:      recordsDTO,
		NextRevision: nextRevision,
		HasMore:      hasMore,
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
