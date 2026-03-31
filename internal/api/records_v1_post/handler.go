// Package recordsv1post реализует HTTP-ручку создания записи.
//
// POST /api/v1/records
//
//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsv1post

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

// CreateRecordRequest — DTO для создания записи.
type CreateRecordRequest struct {
	// Type — тип секрета: "login", "text", "binary", "card".
	Type string `json:"type"`
	// Name — пользовательское название записи.
	Name string `json:"name"`
	// Metadata — произвольная текстовая метаинформация.
	Metadata string `json:"metadata,omitempty"`
	// DeviceID — идентификатор устройства.
	DeviceID string `json:"device_id"`
	// KeyVersion — версия серверного ключа шифрования.
	KeyVersion int64 `json:"key_version"`
	// PayloadVersion — версия payload (для binary записей).
	PayloadVersion int64 `json:"payload_version,omitempty"`
	// Login — данные для типа "login".
	Login *LoginPayload `json:"login,omitempty"`
	// Text — данные для типа "text".
	Text *TextPayload `json:"text,omitempty"`
	// Binary — данные для типа "binary".
	Binary *BinaryPayload `json:"binary,omitempty"`
	// Card — данные для типа "card".
	Card *CardPayload `json:"card,omitempty"`
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

// CreateRecordResponse — DTO ответа при создании записи.
type CreateRecordResponse struct {
	Record recordscommon.RecordDTO `json:"record"`
}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	CreateRecord(record *models.Record) error
}

// Handler — HTTP-обработчик для POST /api/v1/records.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

// Handle создаёт новую запись текущего пользователя.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		recordscommon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	record, err := requestToRecord(&req)
	if err != nil {
		recordscommon.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// UserID извлекается из контекста middleware аутентификации.
	userID, ok := middlewares.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		recordscommon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	record.UserID = userID

	if err := h.service.CreateRecord(record); err != nil {
		if recordscommon.MapRecordError(w, err) {
			return
		}
		recordscommon.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := CreateRecordResponse{
		Record: recordscommon.RecordToDTO(*record),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// requestToRecord преобразует DTO в доменную модель.
func requestToRecord(req *CreateRecordRequest) (*models.Record, error) {
	rt := models.RecordType(req.Type)
	if !models.ValidRecordTypes[rt] {
		return nil, errors.New("invalid record type")
	}

	if req.Name == "" {
		return nil, models.ErrEmptyRecordName
	}
	if req.DeviceID == "" {
		return nil, models.ErrEmptyDeviceID
	}

	var payload models.RecordPayload
	switch rt {
	case models.RecordTypeLogin:
		if req.Login == nil {
			return nil, errors.New("login payload is required")
		}
		payload = models.LoginPayload{Login: req.Login.Login, Password: req.Login.Password}
	case models.RecordTypeText:
		if req.Text == nil {
			return nil, errors.New("text payload is required")
		}
		payload = models.TextPayload{Content: req.Text.Content}
	case models.RecordTypeBinary:
		if req.PayloadVersion <= 0 {
			return nil, models.ErrInvalidPayloadVersion
		}
		// Binary payload content управляется через uploads-слой (task_13).
		// CRUD оперирует только metadata и payload_version как ссылкой на вложение.
		payload = models.BinaryPayload{}
	case models.RecordTypeCard:
		if req.Card == nil {
			return nil, errors.New("card payload is required")
		}
		payload = models.CardPayload{
			Number:     req.Card.Number,
			HolderName: req.Card.HolderName,
			ExpiryDate: req.Card.ExpiryDate,
			CVV:        req.Card.CVV,
		}
	}

	return &models.Record{
		Type:           rt,
		Name:           req.Name,
		Metadata:       req.Metadata,
		DeviceID:       req.DeviceID,
		KeyVersion:     0,
		PayloadVersion: req.PayloadVersion,
		Payload:        payload,
	}, nil
}
