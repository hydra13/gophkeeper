//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsv1post

import (
	"encoding/json"
	"errors"
	"net/http"

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

// BinaryPayload — DTO для бинарных данных.
type BinaryPayload struct {
	Data []byte `json:"data"`
}

// CardPayload — DTO для данных банковской карты.
type CardPayload struct {
	Number     string `json:"number"`
	HolderName string `json:"holder_name"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

// CreateRecordResponse — DTO ответа при создании записи.
type CreateRecordResponse struct {
	ID        int64  `json:"id"`
	Revision  int64  `json:"revision"`
	CreatedAt string `json:"created_at"`
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

// Handle обрабатывает запрос на создание записи.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	record, err := requestToRecord(&req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// UserID извлекается из контекста middleware аутентификации.
	userID, ok := r.Context().Value(userIDKey{}).(int64)
	if !ok || userID <= 0 {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	record.UserID = userID

	if err := record.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.CreateRecord(record); err != nil {
		if errors.Is(err, models.ErrRevisionConflict) {
			writeJSONError(w, http.StatusConflict, "revision conflict")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := CreateRecordResponse{
		ID:        record.ID,
		Revision:  record.Revision,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z"),
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
	if req.KeyVersion <= 0 {
		return nil, models.ErrInvalidKeyVersion
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
		if req.Binary == nil {
			return nil, errors.New("binary payload is required")
		}
		payload = models.BinaryPayload{Data: req.Binary.Data}
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
		KeyVersion:     req.KeyVersion,
		PayloadVersion: req.PayloadVersion,
		Payload:        payload,
	}, nil
}

type userIDKey struct{}

// writeJSONError записывает JSON-ошибку в ответ.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
