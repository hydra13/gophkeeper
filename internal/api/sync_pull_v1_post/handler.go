//go:generate minimock -i .SyncPuller -o mocks -s _mock.go -g
package sync_pull_v1_post

import (
	"encoding/json"
	"log"
	"net/http"

	recordsCommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/models"
)

// Request описывает запрос на получение изменений.
type Request struct {
	UserID        int64  `json:"user_id"`
	DeviceID      string `json:"device_id"`
	SinceRevision int64  `json:"since_revision"`
	Limit         int64  `json:"limit"`
}

// RecordRevisionDTO описывает ревизию записи в ответе синхронизации.
type RecordRevisionDTO struct {
	ID       int64  `json:"id"`
	RecordID int64  `json:"record_id"`
	UserID   int64  `json:"user_id"`
	Revision int64  `json:"revision"`
	DeviceID string `json:"device_id"`
	Deleted  bool   `json:"deleted"`
}

// SyncConflictDTO описывает конфликт синхронизации.
type SyncConflictDTO struct {
	ID             int64                    `json:"id"`
	UserID         int64                    `json:"user_id"`
	RecordID       int64                    `json:"record_id"`
	LocalRevision  int64                    `json:"local_revision"`
	ServerRevision int64                    `json:"server_revision"`
	Resolved       bool                     `json:"resolved"`
	Resolution     string                   `json:"resolution"`
	LocalRecord    *recordsCommon.RecordDTO `json:"local_record,omitempty"`
	ServerRecord   *recordsCommon.RecordDTO `json:"server_record,omitempty"`
}

// Response описывает ответ на получение изменений.
type Response struct {
	Changes      []RecordRevisionDTO       `json:"changes"`
	Records      []recordsCommon.RecordDTO `json:"records"`
	Conflicts    []SyncConflictDTO         `json:"conflicts,omitempty"`
	NextRevision int64                     `json:"next_revision"`
	HasMore      bool                      `json:"has_more"`
}

// SyncPuller описывает получение изменений синхронизации.
type SyncPuller interface {
	Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
}

// Handler обрабатывает получение изменений синхронизации.
type Handler struct {
	service SyncPuller
}

// NewHandler создаёт обработчик получения изменений.
func NewHandler(service SyncPuller) *Handler {
	return &Handler{service: service}
}

// ServeHTTP возвращает изменения синхронизации.
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
	recordsDTO := make([]recordsCommon.RecordDTO, 0, len(records))
	for _, rec := range records {
		recordByID[rec.ID] = rec
		recordsDTO = append(recordsDTO, recordsCommon.RecordToDTO(rec))
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
			localDTO := recordsCommon.RecordToDTO(*conflict.LocalRecord)
			dto.LocalRecord = &localDTO
		}
		if conflict.ServerRecord != nil {
			serverDTO := recordsCommon.RecordToDTO(*conflict.ServerRecord)
			dto.ServerRecord = &serverDTO
		}
		resp.Conflicts = append(resp.Conflicts, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("sync pull response encode failed: %v", err)
	}
}
