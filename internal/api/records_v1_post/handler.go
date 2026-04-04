//go:generate minimock -i .RecordService -o mocks -s _mock.go -g
package recordsv1post

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

type CreateRecordRequest struct {
	Type           string         `json:"type"`
	Name           string         `json:"name"`
	Metadata       string         `json:"metadata,omitempty"`
	DeviceID       string         `json:"device_id"`
	KeyVersion     int64          `json:"key_version"`
	PayloadVersion int64          `json:"payload_version,omitempty"`
	Login          *LoginPayload  `json:"login,omitempty"`
	Text           *TextPayload   `json:"text,omitempty"`
	Binary         *BinaryPayload `json:"binary,omitempty"`
	Card           *CardPayload   `json:"card,omitempty"`
}

type LoginPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type TextPayload struct {
	Content string `json:"content"`
}

type BinaryPayload struct{}

type CardPayload struct {
	Number     string `json:"number"`
	HolderName string `json:"holder_name"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

type CreateRecordResponse struct {
	Record recordscommon.RecordDTO `json:"record"`
}

type RecordService interface {
	CreateRecord(record *models.Record) error
}

type Handler struct {
	service RecordService
}

func NewHandler(service RecordService) *Handler {
	return &Handler{service: service}
}

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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("create record response encode failed: %v", err)
	}
}

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
		payload = models.BinaryPayload{}
	case models.RecordTypeCard:
		if req.Card == nil {
			return nil, errors.New("card payload is required")
		}
		card := models.CardPayload{
			Number:     req.Card.Number,
			HolderName: req.Card.HolderName,
			ExpiryDate: req.Card.ExpiryDate,
			CVV:        req.Card.CVV,
		}
		if err := card.Validate(); err != nil {
			return nil, err
		}
		payload = card
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
