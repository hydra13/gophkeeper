package backend

import (
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/clientui"
)

func toListItem(rec models.Record) RecordListItem {
	return RecordListItem{
		ID:              rec.ID,
		Type:            string(rec.Type),
		Name:            rec.Name,
		Metadata:        rec.Metadata,
		MetadataPreview: clientui.MetadataPreview(rec.Metadata, 80),
		Revision:        rec.Revision,
		Deleted:         rec.IsDeleted(),
		PayloadVersion:  rec.PayloadVersion,
		Payload:         toPayloadDTO(rec.Payload),
	}
}

func toRecordDetails(rec *models.Record) *RecordDetails {
	if rec == nil {
		return nil
	}

	return &RecordDetails{
		ID:             rec.ID,
		Type:           string(rec.Type),
		Name:           rec.Name,
		Metadata:       rec.Metadata,
		Revision:       rec.Revision,
		Deleted:        rec.IsDeleted(),
		DeviceID:       rec.DeviceID,
		KeyVersion:     rec.KeyVersion,
		PayloadVersion: rec.PayloadVersion,
		CreatedAt:      formatTime(rec.CreatedAt),
		UpdatedAt:      formatTime(rec.UpdatedAt),
		Payload:        toPayloadDTO(rec.Payload),
	}
}

func toPayloadDTO(payload models.RecordPayload) RecordPayloadDTO {
	switch typed := payload.(type) {
	case models.LoginPayload:
		return RecordPayloadDTO{
			Login:    typed.Login,
			Password: typed.Password,
		}
	case models.TextPayload:
		return RecordPayloadDTO{
			Content: typed.Content,
		}
	case models.CardPayload:
		return RecordPayloadDTO{
			Number: typed.Number,
			Holder: typed.HolderName,
			Expiry: typed.ExpiryDate,
			CVV:    typed.CVV,
		}
	case models.BinaryPayload:
		return RecordPayloadDTO{
			BinarySize: len(typed.Data),
		}
	default:
		return RecordPayloadDTO{}
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
