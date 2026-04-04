// Package recordsbyidv1put реализует HTTP-ручку обновления записи.
//
// PUT /api/v1/records/{id}
//
//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsbyidv1put

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// UpdateRecordRequest — DTO для обновления записи.
type UpdateRecordRequest struct {
	Name           string         `json:"name"`
	Metadata       string         `json:"metadata,omitempty"`
	Revision       int64          `json:"revision"`
	DeviceID       string         `json:"device_id"`
	KeyVersion     int64          `json:"key_version"`
	PayloadVersion int64          `json:"payload_version,omitempty"`
	Login          *LoginPayload  `json:"login,omitempty"`
	Text           *TextPayload   `json:"text,omitempty"`
	Binary         *BinaryPayload `json:"binary,omitempty"`
	Card           *CardPayload   `json:"card,omitempty"`
}

// LoginPayload — DTO для логин/пароль.
type LoginPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextPayload — DTO для текстовых данных.
type TextPayload struct {
	Content string `json:"content"`
}

// BinaryPayload — DTO для бинарных данных (metadata-only).
// Содержимое управляется через uploads-слой (task_13).
type BinaryPayload struct{}

// CardPayload — DTO для данных банковской карты.
type CardPayload struct {
	Number     string `json:"number"`
	HolderName string `json:"holder_name"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

// UpdateRecordResponse — DTO ответа при обновлении записи.
type UpdateRecordResponse struct {
	Record recordscommon.RecordDTO `json:"record"`
}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
	UpdateRecord(record *models.Record) error
}

// Handler — HTTP-обработчик для PUT /api/v1/records/{id}.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle обновляет запись по идентификатору с проверкой ревизии.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		recordscommon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		recordscommon.WriteError(w, http.StatusBadRequest, "invalid record id")
		return
	}

	var req UpdateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		recordscommon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.service.GetRecord(id)
	if err != nil {
		if recordscommon.MapRecordError(w, err) {
			return
		}
		recordscommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if existing.UserID != userID {
		recordscommon.WriteError(w, http.StatusForbidden, "access denied")
		return
	}

	if existing.IsDeleted() {
		recordscommon.WriteError(w, http.StatusBadRequest, "record is deleted")
		return
	}

	currentRevision := existing.Revision
	if req.Revision <= currentRevision {
		recordscommon.WriteConflict(w, "revision conflict", &req.Revision, &currentRevision)
		return
	}

	existing.Name = req.Name
	existing.Metadata = req.Metadata
	existing.DeviceID = req.DeviceID
	existing.KeyVersion = req.KeyVersion
	existing.PayloadVersion = req.PayloadVersion

	payload, err := buildPayload(existing.Type, &req)
	if err != nil {
		recordscommon.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.Payload = payload

	if err := existing.BumpRevision(req.Revision, req.DeviceID); err != nil {
		recordscommon.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.UpdateRecord(existing); err != nil {
		if recordscommon.MapRecordError(w, err) {
			return
		}
		recordscommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := UpdateRecordResponse{
		Record: recordscommon.RecordToDTO(*existing),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// buildPayload создаёт payload на основе типа записи и DTO.
func buildPayload(rt models.RecordType, req *UpdateRecordRequest) (models.RecordPayload, error) {
	switch rt {
	case models.RecordTypeLogin:
		if req.Login == nil {
			return nil, errors.New("login payload is required")
		}
		return models.LoginPayload{Login: req.Login.Login, Password: req.Login.Password}, nil
	case models.RecordTypeText:
		if req.Text == nil {
			return nil, errors.New("text payload is required")
		}
		return models.TextPayload{Content: req.Text.Content}, nil
	case models.RecordTypeBinary:
		// Binary payload content управляется через uploads-слой (task_13).
		// CRUD оперирует только metadata и payload_version как ссылкой на вложение.
		return models.BinaryPayload{}, nil
	case models.RecordTypeCard:
		if req.Card == nil {
			return nil, errors.New("card payload is required")
		}
		card := models.CardPayload{
			Number: req.Card.Number, HolderName: req.Card.HolderName,
			ExpiryDate: req.Card.ExpiryDate, CVV: req.Card.CVV,
		}
		if err := card.Validate(); err != nil {
			return nil, err
		}
		return card, nil
	default:
		return nil, errors.New("invalid record type")
	}
}
