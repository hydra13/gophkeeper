package recordscommon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
)

func baseRecord() models.Record {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	return models.Record{
		ID:             1,
		UserID:         42,
		Type:           models.RecordTypeLogin,
		Name:           "test record",
		Metadata:       "some metadata",
		Revision:       3,
		DeviceID:       "device-001",
		KeyVersion:     2,
		PayloadVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestRecordToDTO_LoginPayload(t *testing.T) {
	rec := baseRecord()
	rec.Type = models.RecordTypeLogin
	rec.Payload = models.LoginPayload{Login: "user@example.com", Password: "s3cret"}

	dto := RecordToDTO(rec)

	require.Equal(t, int64(1), dto.ID)
	require.Equal(t, int64(42), dto.UserID)
	require.Equal(t, "login", dto.Type)
	require.Equal(t, "test record", dto.Name)
	require.Equal(t, "some metadata", dto.Metadata)
	require.Equal(t, int64(3), dto.Revision)
	require.Equal(t, "device-001", dto.DeviceID)
	require.Equal(t, int64(2), dto.KeyVersion)
	require.Equal(t, int64(1), dto.PayloadVersion)
	require.Equal(t, "2025-06-15T12:00:00Z", dto.CreatedAt)
	require.Equal(t, "2025-06-15T12:00:00Z", dto.UpdatedAt)
	require.Nil(t, dto.DeletedAt)

	loginDTO, ok := dto.Payload.(LoginPayloadDTO)
	require.True(t, ok, "payload should be LoginPayloadDTO")
	require.Equal(t, "user@example.com", loginDTO.Login)
	require.Equal(t, "s3cret", loginDTO.Password)
}

func TestRecordToDTO_TextPayload(t *testing.T) {
	rec := baseRecord()
	rec.Type = models.RecordTypeText
	rec.Payload = models.TextPayload{Content: "my secret note"}

	dto := RecordToDTO(rec)

	require.Equal(t, "text", dto.Type)

	textDTO, ok := dto.Payload.(TextPayloadDTO)
	require.True(t, ok, "payload should be TextPayloadDTO")
	require.Equal(t, "my secret note", textDTO.Content)
}

func TestRecordToDTO_BinaryPayload(t *testing.T) {
	rec := baseRecord()
	rec.Type = models.RecordTypeBinary
	rec.Payload = models.BinaryPayload{}

	dto := RecordToDTO(rec)

	require.Equal(t, "binary", dto.Type)

	_, ok := dto.Payload.(BinaryPayloadDTO)
	require.True(t, ok, "payload should be BinaryPayloadDTO")
}

func TestRecordToDTO_CardPayload(t *testing.T) {
	rec := baseRecord()
	rec.Type = models.RecordTypeCard
	rec.Payload = models.CardPayload{
		Number:     "4111111111111111",
		HolderName: "Ivan Ivanov",
		ExpiryDate: "12/28",
		CVV:        "123",
	}

	dto := RecordToDTO(rec)

	require.Equal(t, "card", dto.Type)

	cardDTO, ok := dto.Payload.(CardPayloadDTO)
	require.True(t, ok, "payload should be CardPayloadDTO")
	require.Equal(t, "4111111111111111", cardDTO.Number)
	require.Equal(t, "Ivan Ivanov", cardDTO.HolderName)
	require.Equal(t, "12/28", cardDTO.ExpiryDate)
	require.Equal(t, "123", cardDTO.CVV)
}

func TestRecordToDTO_DeletedAtSet(t *testing.T) {
	rec := baseRecord()
	deletedAt := time.Date(2025, 7, 1, 10, 30, 0, 0, time.UTC)
	rec.DeletedAt = &deletedAt

	dto := RecordToDTO(rec)

	require.NotNil(t, dto.DeletedAt)
	require.Equal(t, "2025-07-01T10:30:00Z", *dto.DeletedAt)
}

func TestRecordToDTO_DeletedAtNil(t *testing.T) {
	rec := baseRecord()
	rec.DeletedAt = nil

	dto := RecordToDTO(rec)

	require.Nil(t, dto.DeletedAt)
}

func TestRecordToDTO_NilPayload(t *testing.T) {
	rec := baseRecord()
	rec.Payload = nil

	dto := RecordToDTO(rec)

	require.Nil(t, dto.Payload)
}
