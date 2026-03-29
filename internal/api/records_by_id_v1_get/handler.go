//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsbyidv1get

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/hydra13/gophkeeper/internal/models"
)

// LoginPayloadDTO — DTO для логин/пароль.
type LoginPayloadDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextPayloadDTO — DTO для текстовых данных.
type TextPayloadDTO struct {
	Content string `json:"content"`
}

// BinaryPayloadDTO — DTO для бинарных данных.
type BinaryPayloadDTO struct {
	Data []byte `json:"data"`
}

// CardPayloadDTO — DTO для данных банковской карты.
type CardPayloadDTO struct {
	Number     string `json:"number"`
	HolderName string `json:"holder_name"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

// GetRecordResponse — DTO ответа при получении записи.
type GetRecordResponse struct {
	ID             int64       `json:"id"`
	UserID         int64       `json:"user_id"`
	Type           string      `json:"type"`
	Name           string      `json:"name"`
	Metadata       string      `json:"metadata,omitempty"`
	Revision       int64       `json:"revision"`
	Deleted        bool        `json:"deleted"`
	DeviceID       string      `json:"device_id"`
	KeyVersion     int64       `json:"key_version"`
	PayloadVersion int64       `json:"payload_version,omitempty"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
	Payload        interface{} `json:"payload"`
}

// RecordService — интерфейс бизнес-логики для работы с записями.
type RecordService interface {
	GetRecord(id int64) (*models.Record, error)
}

// Handler — HTTP-обработчик для GET /api/v1/records/{id}.
type Handler struct {
	service RecordService
}

// NewHandler создаёт новый обработчик.
func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

type userIDKey struct{}

// Handle обрабатывает запрос на получение записи по ID.
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

	resp := recordToResponse(record)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// recordToResponse преобразует доменную модель в DTO ответа.
func recordToResponse(r *models.Record) GetRecordResponse {
	resp := GetRecordResponse{
		ID:             r.ID,
		UserID:         r.UserID,
		Type:           string(r.Type),
		Name:           r.Name,
		Metadata:       r.Metadata,
		Revision:       r.Revision,
		Deleted:        r.IsDeleted(),
		DeviceID:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
		CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      r.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	switch p := r.Payload.(type) {
	case models.LoginPayload:
		resp.Payload = LoginPayloadDTO{Login: p.Login, Password: p.Password}
	case models.TextPayload:
		resp.Payload = TextPayloadDTO{Content: p.Content}
	case models.BinaryPayload:
		resp.Payload = BinaryPayloadDTO{Data: p.Data}
	case models.CardPayload:
		resp.Payload = CardPayloadDTO{
			Number: p.Number, HolderName: p.HolderName,
			ExpiryDate: p.ExpiryDate, CVV: p.CVV,
		}
	}

	return resp
}

// writeJSONError записывает JSON-ошибку в ответ.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
