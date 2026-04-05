package backend

import (
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/stretchr/testify/require"
)

func TestToPayloadDTO(t *testing.T) {
	t.Run("login payload", func(t *testing.T) {
		dto := toPayloadDTO(models.LoginPayload{
			Login:    "alice",
			Password: "secret",
		})

		require.Equal(t, "alice", dto.Login)
		require.Equal(t, "secret", dto.Password)
	})

	t.Run("binary payload", func(t *testing.T) {
		dto := toPayloadDTO(models.BinaryPayload{
			Data: []byte("hello"),
		})

		require.Equal(t, 5, dto.BinarySize)
	})
}

func TestToRecordDetails(t *testing.T) {
	now := time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC)
	rec := &models.Record{
		ID:             42,
		Type:           models.RecordTypeText,
		Name:           "note",
		Metadata:       "first line\nsecond line",
		Payload:        models.TextPayload{Content: "hello"},
		Revision:       7,
		DeviceID:       "desktop-host",
		KeyVersion:     3,
		PayloadVersion: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	dto := toRecordDetails(rec)
	require.Equal(t, int64(42), dto.ID)
	require.Equal(t, "text", dto.Type)
	require.Equal(t, "hello", dto.Payload.Content)
	require.Equal(t, now.Format(time.RFC3339), dto.CreatedAt)
}
