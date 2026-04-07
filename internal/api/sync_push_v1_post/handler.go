//go:generate minimock -i .SyncPusher -o mocks -s _mock.go -g
package sync_push_v1_post

import (
	"encoding/json"
	"net/http"

	recordsCommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// PendingChange описывает изменение для отправки.
type PendingChange struct {
	Record       recordsCommon.RecordDTO `json:"record"`
	Deleted      bool                    `json:"deleted"`
	BaseRevision int64                   `json:"base_revision"`
}

// Request описывает запрос на отправку изменений.
type Request struct {
	UserID   int64           `json:"user_id"`
	DeviceID string          `json:"device_id"`
	Changes  []PendingChange `json:"changes"`
}

// RecordRevisionDTO описывает ревизию записи в ответе.
type RecordRevisionDTO struct {
	ID       int64  `json:"id"`
	RecordID int64  `json:"record_id"`
	UserID   int64  `json:"user_id"`
	Revision int64  `json:"revision"`
	DeviceID string `json:"device_id"`
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

// Response описывает ответ на отправку изменений.
type Response struct {
	Accepted  []RecordRevisionDTO `json:"accepted"`
	Conflicts []SyncConflictDTO   `json:"conflicts,omitempty"`
}

// SyncPusher описывает отправку изменений синхронизации.
type SyncPusher interface {
	Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
}

// Handler обрабатывает отправку изменений синхронизации.
type Handler struct {
	service SyncPusher
}

// NewHandler создаёт обработчик отправки изменений.
func NewHandler(service SyncPusher) *Handler {
	return &Handler{service: service}
}

// ServeHTTP принимает изменения синхронизации.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responses.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID <= 0 {
		responses.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	if req.DeviceID == "" {
		responses.Error(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if len(req.Changes) == 0 {
		responses.Error(w, http.StatusBadRequest, "changes are required")
		return
	}

	accepted, conflicts, err := h.service.Push(req.UserID, req.DeviceID, toDomainChanges(req.Changes))
	if err != nil {
		responses.Error(w, http.StatusInternalServerError, "internal server error")
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
			localDTO := recordsCommon.RecordToDTO(*conflict.LocalRecord)
			dto.LocalRecord = &localDTO
		}
		if conflict.ServerRecord != nil {
			serverDTO := recordsCommon.RecordToDTO(*conflict.ServerRecord)
			dto.ServerRecord = &serverDTO
		}
		resp.Conflicts = append(resp.Conflicts, dto)
	}

	responses.JSON(w, http.StatusOK, resp)
}

func toDomainChanges(changes []PendingChange) []models.PendingChange {
	result := make([]models.PendingChange, 0, len(changes))
	for _, c := range changes {
		result = append(result, models.PendingChange{
			Record:       dtoToDomainRecord(c.Record),
			Deleted:      c.Deleted,
			BaseRevision: c.BaseRevision,
		})
	}
	return result
}

func dtoToDomainRecord(dto recordsCommon.RecordDTO) *models.Record {
	r := &models.Record{
		ID:             dto.ID,
		UserID:         dto.UserID,
		Type:           models.RecordType(dto.Type),
		Name:           dto.Name,
		Metadata:       dto.Metadata,
		DeviceID:       dto.DeviceID,
		KeyVersion:     dto.KeyVersion,
		PayloadVersion: dto.PayloadVersion,
		Payload:        dtoPayloadToDomain(dto),
	}
	return r
}

func dtoPayloadToDomain(dto recordsCommon.RecordDTO) models.RecordPayload {
	switch models.RecordType(dto.Type) {
	case models.RecordTypeLogin:
		if p, ok := dto.Payload.(map[string]interface{}); ok {
			return models.LoginPayload{
				Login:    strVal(p["login"]),
				Password: strVal(p["password"]),
			}
		}
		return models.LoginPayload{}
	case models.RecordTypeText:
		if p, ok := dto.Payload.(map[string]interface{}); ok {
			return models.TextPayload{Content: strVal(p["content"])}
		}
		return models.TextPayload{}
	case models.RecordTypeBinary:
		return models.BinaryPayload{}
	case models.RecordTypeCard:
		if p, ok := dto.Payload.(map[string]interface{}); ok {
			return models.CardPayload{
				Number:     strVal(p["number"]),
				HolderName: strVal(p["holder_name"]),
				ExpiryDate: strVal(p["expiry_date"]),
				CVV:        strVal(p["cvv"]),
			}
		}
		return models.CardPayload{}
	default:
		return nil
	}
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
