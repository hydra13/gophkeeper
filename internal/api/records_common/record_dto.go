package recordscommon

import (
	"github.com/hydra13/gophkeeper/internal/models"
)

const timeLayout = "2006-01-02T15:04:05Z"

type LoginPayloadDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type TextPayloadDTO struct {
	Content string `json:"content"`
}

type BinaryPayloadDTO struct{}

type CardPayloadDTO struct {
	Number     string `json:"number"`
	HolderName string `json:"holder_name"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

type RecordDTO struct {
	ID             int64       `json:"id"`
	UserID         int64       `json:"user_id"`
	Type           string      `json:"type"`
	Name           string      `json:"name"`
	Metadata       string      `json:"metadata,omitempty"`
	Revision       int64       `json:"revision"`
	DeletedAt      *string     `json:"deleted_at,omitempty"`
	DeviceID       string      `json:"device_id"`
	KeyVersion     int64       `json:"key_version"`
	PayloadVersion int64       `json:"payload_version,omitempty"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
	Payload        interface{} `json:"payload"`
}

func RecordToDTO(r models.Record) RecordDTO {
	dto := RecordDTO{
		ID:             r.ID,
		UserID:         r.UserID,
		Type:           string(r.Type),
		Name:           r.Name,
		Metadata:       r.Metadata,
		Revision:       r.Revision,
		DeviceID:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
		CreatedAt:      r.CreatedAt.Format(timeLayout),
		UpdatedAt:      r.UpdatedAt.Format(timeLayout),
	}
	if r.DeletedAt != nil {
		deletedAt := r.DeletedAt.Format(timeLayout)
		dto.DeletedAt = &deletedAt
	}

	switch p := r.Payload.(type) {
	case models.LoginPayload:
		dto.Payload = LoginPayloadDTO{Login: p.Login, Password: p.Password}
	case models.TextPayload:
		dto.Payload = TextPayloadDTO{Content: p.Content}
	case models.BinaryPayload:
		dto.Payload = BinaryPayloadDTO{}
	case models.CardPayload:
		dto.Payload = CardPayloadDTO{
			Number: p.Number, HolderName: p.HolderName,
			ExpiryDate: p.ExpiryDate, CVV: p.CVV,
		}
	}

	return dto
}
